// Disposable model feasibility harness; never linked into the application.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/sys/windows/svc"
	"immichclone/internal/media"
	"os"
	"path/filepath"
	"time"
)

type service struct{ root, python string }

func (p service) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending, WaitHint: 120000}
	done := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	go func() {
		r, e := media.Run(ctx, p.python, []string{"-B", filepath.Join(p.root, "probe.py"), p.root}, nil)
		if e == nil {
			var data map[string]any
			e = json.Unmarshal(r.Output, &data)
			if e == nil && data["originals_unchanged"] != true {
				e = fmt.Errorf("invalid model result")
			}
			if e == nil {
				data["peak_job_committed_bytes"] = uint64(r.Peak)
				data["job_limit_bytes"] = media.MemoryLimit
				var b []byte
				b, e = json.Marshal(data)
				if e == nil {
					e = os.WriteFile(filepath.Join(p.root, "passed.json"), b, 0600)
				}
			}
		}
		done <- e
	}()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	checkpoint := uint32(1)
	for {
		select {
		case e := <-done:
			if e != nil {
				_ = os.WriteFile(filepath.Join(p.root, "failed.txt"), []byte(e.Error()), 0600)
				return false, 1
			}
			status <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
			for r := range requests {
				switch r.Cmd {
				case svc.Interrogate:
					status <- r.CurrentStatus
				case svc.Stop, svc.Shutdown:
					return false, 0
				}
			}
			return false, 0
		case <-ticker.C:
			checkpoint++
			status <- svc.Status{State: svc.StartPending, CheckPoint: checkpoint, WaitHint: 120000}
		}
	}
}
func main() {
	root := flag.String("root", "", "disposable inputs")
	python := flag.String("python", "", "cloud harness interpreter")
	flag.Parse()
	if !filepath.IsAbs(*root) || !filepath.IsAbs(*python) {
		os.Exit(2)
	}
	if svc.Run("ImmichCloneAIProbe", service{*root, *python}) != nil {
		os.Exit(1)
	}
}
