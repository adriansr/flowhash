package flowhash_test

import (
	"fmt"
	"net"

	"github.com/adriansr/flowhash"
)

func ExampleCommunityIDHash() {
	flow := flowhash.Flow{
		SourceIP:        net.ParseIP("10.1.2.3"),
		DestinationIP:   net.ParseIP("8.8.8.8"),
		SourcePort:      63521,
		DestinationPort: 53,
		Protocol:        17,
	}
	fmt.Println(flowhash.CommunityID.Hash(flow))
	// Output: 1:R7iR6vkxw+jaz3wjDfWMWooBdfc=
}
