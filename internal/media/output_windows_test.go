package media

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestAnalysisOutputHelper(t *testing.T) {
	for _, arg := range os.Args {
		if arg == "6a-child-exit" {
			os.Exit(17)
		}
		if arg == "6a-two-mib" {
			_, _ = os.Stdout.Write(make([]byte, 2<<20))
			os.Exit(0)
		}
		if arg == "6a-four-mib" {
			_, _ = os.Stdout.Write(make([]byte, 4<<20))
			os.Exit(0)
		}
	}
}
func TestAnalysisOutputAllowanceRemainsFinite(t *testing.T) {
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		arg   string
		limit int64
		ok    bool
	}{{"6a-two-mib", 3 << 20, true}, {"6a-four-mib", 3 << 20, false}, {"6a-two-mib", OutputLimit, false}} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		result, e := runLimited(ctx, exe, []string{"-test.run=^TestAnalysisOutputHelper$", "--", tc.arg}, nil, tc.limit)
		cancel()
		if tc.ok {
			if e != nil || len(result.Output) != 2<<20 {
				t.Fatal("analysis output positive control failed")
			}
		} else if !errors.Is(e, errOutputLimit) {
			t.Fatal("output excess not rejected", e)
		}
	}
}

func TestNativeChildExitCodeIsPreserved(t *testing.T) {
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, e = Run(ctx, exe, []string{"-test.run=^TestAnalysisOutputHelper$", "--", "6a-child-exit"}, nil)
	var exit *ChildExitError
	if !errors.As(e, &exit) || exit.NativeExitCode() != 17 {
		t.Fatal("native child exit code lost", e)
	}
}
