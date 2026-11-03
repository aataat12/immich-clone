// Disposable native AI execution proof. Never linked into the application.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"golang.org/x/sys/windows/svc"
	"immichclone/internal/media"
	"os"
	"path/filepath"
	"time"
)

type face struct {
	Box        []float64 `json:"box"`
	Confidence float64   `json:"confidence"`
	Embedding  []float64 `json:"embedding"`
}
type object struct {
	Class      int       `json:"class"`
	Confidence float64   `json:"confidence"`
	Box        []float64 `json:"box"`
}
type response struct {
	Protocol int      `json:"protocol"`
	Faces    []face   `json:"faces"`
	Objects  []object `json:"objects"`
}

func run(root string) (summary []byte, err error) {
	var manifest map[string]string
	b, e := os.ReadFile(filepath.Join(root, "input-hashes.json"))
	if e != nil {
		return nil, e
	}
	if e = json.Unmarshal(b, &manifest); e != nil {
		return nil, e
	}
	verify := func() error {
		for name, want := range manifest {
			if filepath.Base(name) != name {
				return errors.New("invalid manifest")
			}
			b, e := os.ReadFile(filepath.Join(root, name))
			if e != nil {
				return e
			}
			h := sha256.Sum256(b)
			if hex.EncodeToString(h[:]) != want {
				return errors.New("input changed")
			}
		}
		return nil
	}
	if e = verify(); e != nil {
		return nil, e
	}
	var gold map[string][][]float64
	b, e = os.ReadFile(filepath.Join(root, "golden.json"))
	if e != nil {
		return nil, e
	}
	if e = json.Unmarshal(b, &gold); e != nil {
		return nil, e
	}
	exe := filepath.Join(root, "ai-worker.exe")
	peak := uint64(0)
	securityCtx, securityCancel := context.WithTimeout(context.Background(), 20*time.Second)
	securityResult, securityErr := media.Run(securityCtx, filepath.Join(root, "native-security-check.exe"), nil, nil)
	securityCancel()
	if securityErr != nil {
		return nil, errors.New("native security regressions failed under LocalService")
	}
	var securityReport struct {
		Passed bool `json:"passed"`
	}
	if json.Unmarshal(securityResult.Output, &securityReport) != nil || !securityReport.Passed {
		return nil, errors.New("native security evidence invalid")
	}
	if uint64(securityResult.Peak) > peak {
		peak = uint64(securityResult.Peak)
	}
	call := func(task, name string) (response, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		r, e := media.Run(ctx, exe, []string{task, root, filepath.Join(root, name)}, nil)
		if uint64(r.Peak) > peak {
			peak = uint64(r.Peak)
		}
		var out response
		if e != nil {
			return out, e
		}
		e = json.Unmarshal(r.Output, &out)
		if e == nil && out.Protocol != 1 {
			e = errors.New("invalid protocol")
		}
		return out, e
	}
	minimum := 1.0
	vectors := map[string][]float64{}
	for _, name := range []string{"astronaut.png.bgr", "dimmed.bgr", "grace_hopper.jpg.bgr", "coffee.png.bgr", "chelsea.png.bgr"} {
		out, e := call("faces", name)
		if e != nil {
			return nil, fmt.Errorf("native faces: %w", e)
		}
		if len(out.Faces) != len(gold[name]) {
			return nil, errors.New("face count parity")
		}
		for i, f := range out.Faces {
			if len(f.Embedding) != 128 || len(f.Box) != 4 || f.Confidence < .9 {
				return nil, errors.New("face shape")
			}
			dot := 0.0
			for j, v := range f.Embedding {
				dot += v * gold[name][i][j]
			}
			if dot < .999 {
				return nil, errors.New("native embedding parity")
			}
			if dot < minimum {
				minimum = dot
			}
			vectors[name] = f.Embedding
		}
	}
	dot := func(a, b []float64) float64 {
		v := 0.0
		for i, x := range a {
			v += x * b[i]
		}
		return v
	}
	same := dot(vectors["astronaut.png.bgr"], vectors["dimmed.bgr"])
	different := dot(vectors["astronaut.png.bgr"], vectors["grace_hopper.jpg.bgr"])
	if same < .9 || different > .25 {
		return nil, errors.New("native similarity")
	}
	for name, expected := range map[string]int{"astronaut.png.bgr": 0, "chelsea.png.bgr": 15, "coffee.png.bgr": 41} {
		out, e := call("objects", name)
		if e != nil {
			return nil, fmt.Errorf("native objects: %w", e)
		}
		found := false
		for _, o := range out.Objects {
			if len(o.Box) != 4 || o.Box[0] < 0 || o.Box[1] < 0 || o.Box[2] <= 0 || o.Box[3] <= 0 || o.Confidence < .5 || o.Class < 0 || o.Class >= 80 {
				return nil, errors.New("object shape")
			}
			if o.Class == expected {
				found = true
			}
		}
		if !found {
			return nil, errors.New("object class")
		}
	}
	for _, name := range []string{"truncated.bgr", "oversized.bgr", "invalid.bgr", "trailing.bgr"} {
		if _, e := call("faces", name); e == nil {
			return nil, errors.New("malformed frame accepted")
		}
	}
	if e = os.Rename(filepath.Join(root, "sface.onnx"), filepath.Join(root, "sface.absent")); e != nil {
		return nil, e
	}
	_, missing := call("faces", "astronaut.png.bgr")
	if e = os.Rename(filepath.Join(root, "sface.absent"), filepath.Join(root, "sface.onnx")); e != nil {
		return nil, e
	}
	if missing == nil {
		return nil, errors.New("missing model accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, e = media.Run(ctx, exe, []string{"objects", root, filepath.Join(root, "coffee.png.bgr")}, func(uint32) { time.AfterFunc(20*time.Millisecond, cancel) })
	if !errors.Is(e, context.Canceled) {
		return nil, errors.New("native cancellation not confirmed")
	}
	if _, e = call("faces", "astronaut.png.bgr"); e != nil {
		return nil, errors.New("post-failure recovery")
	}
	if e = verify(); e != nil {
		return nil, e
	}
	return json.Marshal(map[string]any{"security_backport_regressions": true, "native_inference": true, "minimum_python_embedding_cosine": minimum, "same_photo_cosine": same, "different_people_cosine": different, "object_fixtures": 3, "frame_rejections": 4, "model_failure_recovery": true, "cancellation_recovery": true, "peak_job_committed_bytes": peak, "job_limit_bytes": media.MemoryLimit, "originals_unchanged": true})
}

type service struct{ root string }

func (p service) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending, WaitHint: 120000}
	done := make(chan error, 1)
	go func() {
		b, e := run(p.root)
		if e == nil {
			e = os.WriteFile(filepath.Join(p.root, "passed.json"), b, 0600)
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
	root := flag.String("root", "", "disposable public fixture directory")
	flag.Parse()
	if !filepath.IsAbs(*root) {
		os.Exit(2)
	}
	if svc.Run("ImmichCloneNativeAIProbe", service{*root}) != nil {
		os.Exit(1)
	}
}
