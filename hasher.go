package flowhash

// Hasher is an interface that provides a method to hash flows.
type Hasher interface {
	Hash(flow Flow) string
}
