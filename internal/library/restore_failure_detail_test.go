package library

import (
	"errors"
	"fmt"
	"syscall"
	"testing"
)

func TestRestoreFailureDetailIsBoundedAndKeepsCategory(t *testing.T) {
	e := restoreStageError("progress-read", fmt.Errorf("private path and content: %w", syscall.Errno(32)))
	if !errors.Is(e, ErrRestoreWorkerFailed) || RestoreFailureDetail(fmt.Errorf("outer private text: %w", e)) != "progress-read:32" {
		t.Fatal("detail or category lost")
	}
	if RestoreFailureDetail(errors.New("private path")) != "unavailable" || RestoreFailureDetail(restoreStageError("private path", nil)) != "unavailable" {
		t.Fatal("untrusted detail exposed")
	}
	if RestoreFailureDetail(restoreWorkerFailure(23)) != "worker-exit:23" {
		t.Fatal("exit code lost")
	}
}
