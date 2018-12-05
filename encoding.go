package flowhash

import (
	"encoding/base64"
	"encoding/hex"
)

// Encoding is used to encode the flow hash.
type Encoding interface {
	EncodeToString([]byte) string
}

var (
	// HexEncoding encodes the checksum in hexadecimal.
	HexEncoding = hexEncoding{}

	// Base64Encoding uses Base64 to encode the checksum, including
	// padding characters. This is the default for a Community ID.
	// This is an alias for the StdEncoding in the encoding/base64 package.
	Base64Encoding = base64.StdEncoding
)

type hexEncoding struct{}

func (hexEncoding) EncodeToString(data []byte) string {
	return hex.EncodeToString(data)
}
