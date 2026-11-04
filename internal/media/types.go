package media

import (
	"errors"
	"fmt"
)

const MemoryLimit = 512 << 20
const OutputLimit = 1 << 20
const ProviderID = "ffmpeg-8.1.3-7eb77d88e2e9"

var ErrUnavailable = errors.New("optional media provider unavailable; original remains available")

type Result struct {
	Output []byte
	Peak   uintptr
	PID    uint32
}

// ChildExitError exposes only a numeric process result, never stderr or paths.
type ChildExitError struct{ Code uint32 }

func (e *ChildExitError) Error() string          { return fmt.Sprintf("child exit code %d", e.Code) }
func (e *ChildExitError) NativeExitCode() uint32 { return e.Code }
