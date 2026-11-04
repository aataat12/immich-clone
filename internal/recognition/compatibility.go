package recognition

// CompatibleEmbeddings keeps binary provenance separate from embedding-space
// compatibility. These exact serviced providers retain the same model hashes and
// BGR960 preprocessing; actual reference parity is recorded in dependency-evidence
// and opencv-nr01. Unknown/future providers require independent evidence.
func CompatibleEmbeddings(a, b string) bool {
	if a == b {
		return true
	}
	known := func(id string) bool {
		switch id {
		case "opencv-4.12.0-a83dcef02ebe-bgr960-v1",
			"opencv-4.12.0-zlib-1.3.2-cb35f0ccb3e3-bgr960-v1",
			"opencv-4.12.0-nr01-zlib-1.3.2-15d1c2ca7ef3-bgr960-v1":
			return true
		}
		return false
	}
	return known(a) && known(b)
}
