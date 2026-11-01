// Disposable compatibility experiment, never packaged with the application.
package main

import (
	"flag"
	"golang.org/x/sys/windows/svc"
	"os"
	"path/filepath"
)

type probe struct{ root string }

func (p probe) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}
	if err := run(p.root); err != nil {
		_ = os.WriteFile(filepath.Join(p.root, "failed.txt"), []byte(err.Error()), 0600)
		return false, 1
	}
	if err := os.WriteFile(filepath.Join(p.root, "passed.txt"), []byte("actual decode, pixels, PNG encode, malformed rejection and original hashes passed"), 0600); err != nil {
		return false, 2
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
}
func main() {
	root := flag.String("root", "", "synthetic fixtures")
	flag.Parse()
	if *root == "" {
		os.Exit(2)
	}
	if svc.Run("ImmichCloneImageProbe", probe{*root}) != nil {
		os.Exit(1)
	}
}
