// Disposable synthetic authority probe; never linked into the application.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"immichclone/internal/media"
)

const secretValue = "synthetic-worker-secret"
const catalogValue = "synthetic-catalog-canary"
const writeValue = "synthetic-child-write"

type accessResult struct {
	LocalService    bool `json:"local_service"`
	SecretRead      bool `json:"secret_read"`
	CatalogRead     bool `json:"catalog_read"`
	OriginalWrite   bool `json:"original_write"`
	DeniedRead      bool `json:"denied_read"`
	DeniedWrite     bool `json:"denied_write"`
	LargeAllocation bool `json:"large_allocation"`
	ChildSpawn      bool `json:"child_spawn"`
	InputRead       bool `json:"input_read"`
	ScratchWrite    bool `json:"scratch_write"`
}

func localService() bool {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	return err == nil && user.User.Sid.String() == "S-1-5-19"
}
func readEquals(path, expected string) bool {
	data, err := os.ReadFile(path)
	return err == nil && string(data) == expected
}
func canWrite(path string) bool {
	return os.WriteFile(path, []byte(writeValue), 0600) == nil
}
func child(root string) accessResult {
	r := accessResult{LocalService: localService()}
	r.InputRead = readEquals(filepath.Join(root, "input", "required.txt"), "synthetic-required-input")
	r.ScratchWrite = os.WriteFile(filepath.Join(root, "scratch", "worker.tmp"), []byte("synthetic-scratch"), 0600) == nil
	r.SecretRead = readEquals(filepath.Join(root, "state", "secret.txt"), secretValue)
	r.CatalogRead = readEquals(filepath.Join(root, "state", "catalog-canary.bin"), catalogValue)
	r.OriginalWrite = canWrite(filepath.Join(root, "state", "original-canary.bin"))
	r.DeniedRead = readEquals(filepath.Join(root, "denied", "secret.txt"), secretValue)
	r.DeniedWrite = canWrite(filepath.Join(root, "denied", "forbidden.txt"))
	allocation, err := windows.VirtualAlloc(0, 600<<20, windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
	if err == nil {
		r.LargeAllocation = true
		_ = windows.VirtualFree(allocation, 0, windows.MEM_RELEASE)
	}
	exe, err := os.Executable()
	if err == nil {
		leaf := exec.Command(exe, "-mode", "leaf")
		if leaf.Start() == nil {
			r.ChildSpawn = true
			_ = leaf.Process.Kill()
			_ = leaf.Wait()
		}
	}
	return r
}
func restrictedProbe(ctx context.Context, root string) map[string]bool {
	result := map[string]bool{"service_identity_localservice": localService()}
	exe, err := os.Executable()
	if err != nil { result["probe_complete"] = false; return result }
	jobCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	bounded, err := media.RunRestricted(jobCtx, exe, []string{"-mode", "child", "-root", root}, nil)
	cancel()
	var childResult accessResult
	if err != nil { result["restricted_launcher_failed"] = true; result["probe_complete"] = false; return result }
	if json.Unmarshal(bounded.Output, &childResult) != nil { result["restricted_output_invalid"] = true; result["probe_complete"] = false; return result }
	result["restricted_identity_localservice"] = childResult.LocalService
	result["restricted_input_read_allowed"] = childResult.InputRead
	result["restricted_scratch_write_allowed"] = childResult.ScratchWrite
	result["restricted_state_secret_denied"] = !childResult.SecretRead
	result["restricted_catalog_denied"] = !childResult.CatalogRead
	result["restricted_original_write_denied"] = !childResult.OriginalWrite
	result["restricted_private_read_denied"] = !childResult.DeniedRead
	result["restricted_process_spawn_denied"] = !childResult.ChildSpawn
	result["job_peak_within_configured_bound"] = bounded.Peak > 0 && bounded.Peak <= media.MemoryLimit
	result["probe_complete"] = true
	return result
}
func probe(ctx context.Context, root string) map[string]bool {
	result := map[string]bool{"service_identity_localservice": localService()}
	exe, err := os.Executable()
	if err != nil {
		result["probe_complete"] = false
		return result
	}
	controlCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	command := exec.CommandContext(controlCtx, exe, "-mode", "child", "-root", root)
	plain, err := command.Output()
	cancel()
	var control accessResult
	if err != nil || json.Unmarshal(plain, &control) != nil {
		result["probe_complete"] = false
		return result
	}
	jobCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	bounded, err := media.Run(jobCtx, exe, []string{"-mode", "child", "-root", root}, nil)
	cancel()
	var job accessResult
	if err != nil || json.Unmarshal(bounded.Output, &job) != nil {
		result["probe_complete"] = false
		return result
	}
	result["unrestricted_identity_localservice"] = control.LocalService
	result["job_identity_localservice"] = job.LocalService
	result["unrestricted_secret_read_allowed"] = control.SecretRead
	result["job_secret_read_allowed"] = job.SecretRead
	result["unrestricted_catalog_read_allowed"] = control.CatalogRead
	result["job_catalog_read_allowed"] = job.CatalogRead
	result["unrestricted_original_write_allowed"] = control.OriginalWrite
	result["job_original_write_allowed"] = job.OriginalWrite
	result["unrestricted_private_read_denied"] = !control.DeniedRead
	result["job_private_read_denied"] = !job.DeniedRead
	result["unrestricted_private_write_denied"] = !control.DeniedWrite
	result["job_private_write_denied"] = !job.DeniedWrite
	result["unrestricted_large_allocation_allowed"] = control.LargeAllocation
	result["job_large_allocation_refused"] = !job.LargeAllocation
	result["unrestricted_spawn_allowed"] = control.ChildSpawn
	result["job_spawn_refused"] = !job.ChildSpawn
	result["job_peak_within_configured_bound"] = bounded.Peak > 0 && bounded.Peak <= media.MemoryLimit
	result["synthetic_original_write_verified"] = readEquals(filepath.Join(root, "state", "original-canary.bin"), writeValue)
	result["probe_complete"] = true
	return result
}

type serviceProbe struct{ root string; candidate bool }

func (p *serviceProbe) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	done := make(chan map[string]bool, 1)
	go func() { if p.candidate { done <- restrictedProbe(ctx, p.root) } else { done <- probe(ctx, p.root) } }()
	status <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for {
		select {
		case result := <-done:
			data, err := json.Marshal(result)
			part := filepath.Join(p.root, "control", "result.part")
			if err != nil || os.WriteFile(part, data, 0600) != nil || os.Rename(part, filepath.Join(p.root, "control", "result.json")) != nil {
				return false, 2
			}
			done = nil
		case request := <-requests:
			switch request.Cmd {
			case svc.Interrogate:
				status <- request.CurrentStatus
			case svc.Stop, svc.Shutdown:
				status <- svc.Status{State: svc.StopPending}
				cancel()
				return false, 0
			}
		}
	}
}
func main() {
	mode := flag.String("mode", "service", "fixed probe mode")
	root := flag.String("root", "", "disposable fixture directory")
	name := flag.String("service", "ImmichCloneWorkerCanary", "disposable service name")
	flag.Parse()
	if *mode == "leaf" {
		time.Sleep(5 * time.Second)
		return
	}
	if *root == "" || !filepath.IsAbs(*root) {
		os.Exit(2)
	}
	if *mode == "child" {
		if json.NewEncoder(os.Stdout).Encode(child(*root)) != nil {
			os.Exit(2)
		}
		return
	}
	if *mode == "candidate" {
		if json.NewEncoder(os.Stdout).Encode(restrictedProbe(context.Background(), *root)) != nil { os.Exit(2) }
		return
	}
	if *mode != "service" && *mode != "candidate-service" {
		os.Exit(2)
	}
	if svc.Run(*name, &serviceProbe{root: *root, candidate: *mode == "candidate-service"}) != nil {
		os.Exit(2)
	}
}

