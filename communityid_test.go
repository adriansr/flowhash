package flowhash

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

const (
	pcapDir   = "testdata/pcap"
	goldenDir = "testdata/golden"
)

var (
	update = flag.Bool("update", false, "updates the golden files")
)

func TestPCAPFiles(t *testing.T) {
	pcaps, err := filepath.Glob(filepath.Join(pcapDir, "*.pcap"))
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range pcaps {
		testName := strings.TrimSuffix(filepath.Base(file), ".pcap")

		t.Run(testName, func(t *testing.T) {
			goldenName := filepath.Join(goldenDir, testName+".pcap.log")
			result := getFlowsFromPCAP(t, testName, file)

			if *update {
				/*data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					t.Fatal(err)
				}

				if err = os.MkdirAll(goldenDir, 0755); err != nil {
					t.Fatal(err)
				}

				err = ioutil.WriteFile(goldenName, data, 0644)
				if err != nil {
					t.Fatal(err)
				}

				return*/
				// TODO
			}

			goldenData := readGoldenFile(t, goldenName)
			assert.Equal(t, goldenData, result)
		})
	}
}

func readGoldenFile(t testing.TB, name string) []string {
	file, err := os.Open(name)
	if err != nil {
		t.Fatal(err, name)
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	var flows []string
	for {
		flow, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err, name)
		}
		flows = append(flows, flow)
	}
	return flows
}

func getFlowsFromPCAP(t testing.TB, name, pcapFile string) []string {
	t.Helper()

	r, err := pcap.OpenOffline(pcapFile)
	if err != nil {
		t.Fatal(err, name)
	}
	defer r.Close()

	packetSource := gopacket.NewPacketSource(r, r.LinkType())
	var flows []string

	// Process packets in PCAP and get flow records.
	for packet := range packetSource.Packets() {
		var flow Flow
		var isIP bool
		if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
			if ipLayer, ok := ipLayer.(*layers.IPv4); ok {
				flow.SourceIP = ipLayer.SrcIP
				flow.DestinationIP = ipLayer.DstIP
				flow.Protocol = uint8(ipLayer.Protocol)
				isIP = true
			}
		}
		if ipLayer := packet.Layer(layers.LayerTypeIPv6); ipLayer != nil {
			if ipLayer, ok := ipLayer.(*layers.IPv6); ok {
				flow.SourceIP = ipLayer.SrcIP
				flow.DestinationIP = ipLayer.DstIP
				flow.Protocol = uint8(ipLayer.NextHeader)
				isIP = true
			}
		}

		flowID := "<not IP>"
		flowStr := ""
		if isIP {
			switch flow.Protocol {
			case kIPProtoTCP:
				if layer := packet.Layer(layers.LayerTypeTCP); layer != nil {
					if layer, ok := layer.(*layers.TCP); ok {
						flow.SourcePort = uint16(layer.SrcPort)
						flow.DestinationPort = uint16(layer.DstPort)
					}
				}
			case kIPProtoUDP:
				if layer := packet.Layer(layers.LayerTypeUDP); layer != nil {
					if layer, ok := layer.(*layers.UDP); ok {
						flow.SourcePort = uint16(layer.SrcPort)
						flow.DestinationPort = uint16(layer.DstPort)
					}
				}
			case kIPProtoSCTP:
				if layer := packet.Layer(layers.LayerTypeSCTP); layer != nil {
					if layer, ok := layer.(*layers.SCTP); ok {
						flow.SourcePort = uint16(layer.SrcPort)
						flow.DestinationPort = uint16(layer.DstPort)
					}
				}
			case kIPProtoICMPv4:
				if layer := packet.Layer(layers.LayerTypeICMPv4); layer != nil {
					if layer, ok := layer.(*layers.ICMPv4); ok {
						flow.ICMP.Type = layer.TypeCode.Type()
						flow.ICMP.Code = layer.TypeCode.Code()
					}
				}
			case kIPProtoICMPv6:
				if layer := packet.Layer(layers.LayerTypeICMPv6); layer != nil {
					if layer, ok := layer.(*layers.ICMPv6); ok {
						flow.ICMP.Type = layer.TypeCode.Type()
						flow.ICMP.Code = layer.TypeCode.Code()
					}
				}
			}
			flowID = CommunityID.Hash(flow)
			flowStr = flowToString(flow)
		}

		flows = append(flows, fmt.Sprintf("%d.%06d | %s | %v\n",
			packet.Metadata().Timestamp.Unix(),
			time.Duration(packet.Metadata().Timestamp.Nanosecond())/time.Microsecond,
			flowID,
			flowStr))
	}

	return flows
}

func flowToString(flow Flow) string {
	switch flow.Protocol {
	case kIPProtoICMPv4, kIPProtoICMPv6:
		return fmt.Sprintf("%s %s %d %d %d",
			ipToStr(flow.SourceIP),
			ipToStr(flow.DestinationIP),
			flow.Protocol,
			flow.ICMP.Type,
			flow.ICMP.Code,
		)
	case kIPProtoSCTP, kIPProtoTCP, kIPProtoUDP:
		return fmt.Sprintf("%s %s %d %d %d",
			ipToStr(flow.SourceIP),
			ipToStr(flow.DestinationIP),
			flow.Protocol,
			flow.SourcePort,
			flow.DestinationPort,
		)
	default:
		return fmt.Sprintf("%s %s %d",
			ipToStr(flow.SourceIP),
			ipToStr(flow.DestinationIP),
			flow.Protocol,
		)
	}
}

// This is needed because golden data from corelight/community-id
// has the IPv6 addresses compressed, but Golang doesn't compress them.
// Example: 1234:0:0:0:0:5678 => 1234::5678
func ipToStr(ip net.IP) string {
	if len(ip) != 16 {
		return ip.String()
	}
	curRun := 0
	bestPos := 0
	bestRun := 0
	for pos := 0; pos < 8; pos++ {
		isZero := ip[2*pos] == 0 && ip[1+2*pos] == 0
		if !isZero {
			if curRun > bestRun {
				bestRun = curRun
				bestPos = pos - curRun
			}
			curRun = 0
		} else {
			curRun++
		}
	}
	if curRun > bestRun {
		bestRun = curRun
		bestPos = 16 - curRun
	}
	if bestRun == 0 {
		return ip.String()
	}
	var s string
	for pos := 0; pos < bestPos; pos++ {
		if pos > 0 {
			s += ":"
		}
		val := binary.BigEndian.Uint16(ip[2*pos:])
		s += fmt.Sprintf("%x", val)
	}
	s += ":"
	for pos := bestPos + bestRun; pos < 8; pos++ {
		val := binary.BigEndian.Uint16(ip[2*pos:])
		s += fmt.Sprintf(":%x", val)
	}
	if len(s) == 1 {
		s += ":"
	}
	return s
}