package flowhash

type Hasher interface {
	Hash(flow Flow) string
}
