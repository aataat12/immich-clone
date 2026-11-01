package media

import "errors"

const MemoryLimit = 512 << 20
const OutputLimit = 1 << 20
const ProviderID = "ffmpeg-8.1.3-7eb77d88e2e9"

var ErrUnavailable = errors.New("optional media provider unavailable; original remains available")

type Result struct {
	Output []byte
	Peak   uintptr
	PID    uint32
}
