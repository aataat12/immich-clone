// Disposable CPU-only service experiment; not linked into the application.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	_ "golang.org/x/image/bmp"
	"golang.org/x/sys/windows/svc"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func run(root string, requireGrid bool) error {
	var manifest map[string]string
	b, e := os.ReadFile(filepath.Join(root, "fixtures.json"))
	if e != nil {
		return e
	}
	if e = json.Unmarshal(b, &manifest); e != nil {
		return e
	}
	exe := filepath.Join(root, "tools", "ffmpeg.exe")
	probe := filepath.Join(root, "tools", "ffprobe.exe")
	version, e := command(exe, "-version")
	if e != nil {
		return errors.New("candidate cannot start")
	}
	if strings.Contains(string(version), "--enable-gpl") || strings.Contains(string(version), "--enable-nonfree") {
		return errors.New("unexpected license configuration")
	}
	license, e := command(exe, "-L")
	if e != nil || !bytes.Contains(license, []byte("Lesser General Public License")) {
		return errors.New("license not confirmed")
	}
	// Verify PE import audit does not silently add an installer prerequisite.
	imports, e := os.ReadFile(filepath.Join(root, "tools", "pe-imports.txt"))
	if e != nil {
		return e
	}
	allowed := map[string]bool{"kernel32.dll": true, "msvcrt.dll": true, "bcrypt.dll": true, "user32.dll": true, "advapi32.dll": true, "shell32.dll": true, "ws2_32.dll": true, "ole32.dll": true}
	for _, line := range strings.Split(strings.TrimSpace(string(imports)), "\n") {
		name := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "DLL: ")))
		if !allowed[name] {
			return fmt.Errorf("unexpected native import: %s", name)
		}
	}
	protocols, e := command(exe, "-protocols")
	if e != nil {
		return e
	}
	for _, line := range strings.Split(string(protocols), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 1 && !strings.HasSuffix(fields[0], ":") && fields[0] != "file" && fields[0] != "pipe" {
			return errors.New("unexpected enabled protocol")
		}
	}
	self, e := os.Executable()
	if e != nil {
		return e
	}
	started := time.Now()
	if _, e = commandBounded(250*time.Millisecond, self, "-probe-stall"); !errors.Is(e, context.DeadlineExceeded) || time.Since(started) > 5*time.Second {
		return errors.New("actual child timeout termination failed")
	}
	if _, e = commandBounded(5*time.Second, self, "-probe-flood"); e == nil {
		return errors.New("actual child output limit not enforced")
	}
	if e = containmentChecks(root); e != nil {
		return e
	}
	for _, name := range []string{"h264.mp4", "hevc.mp4", "rotated.mov", "synthetic.heic"} {
		path := filepath.Join(root, name)
		original, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		hash := sha256.Sum256(original)
		if hex.EncodeToString(hash[:]) != manifest[name] {
			return errors.New("fixture checksum mismatch")
		}
		metadata, e := command(probe, "-v", "error", "-protocol_whitelist", "file,pipe", "-select_streams", "v:0", "-show_entries", "stream=codec_name,width,height:stream_side_data=rotation", "-of", "json", path)
		if e != nil {
			return fmt.Errorf("%s metadata failed", name)
		}
		var info struct {
			Streams []struct {
				Codec         string `json:"codec_name"`
				Width, Height int
				SideData      []struct {
					Rotation int `json:"rotation"`
				} `json:"side_data_list"`
			}
		}
		if e = json.Unmarshal(metadata, &info); e != nil || len(info.Streams) != 1 {
			return errors.New("stream inspection failed")
		}
		if name == "rotated.mov" && (len(info.Streams[0].SideData) == 0 || info.Streams[0].SideData[0].Rotation != 90) {
			return errors.New("rotation fixture display matrix missing")
		}
		expectedCodec := "hevc"
		if name == "h264.mp4" || name == "rotated.mov" {
			expectedCodec = "h264"
		}
		if info.Streams[0].Codec != expectedCodec || info.Streams[0].Width != 160 || info.Streams[0].Height != 120 {
			return fmt.Errorf("%s unexpected stream", name)
		}
		out, e := command(exe, "-hide_banner", "-loglevel", "error", "-nostdin", "-max_alloc", "67108864", "-threads", "1", "-protocol_whitelist", "file,pipe", "-i", path, "-map", "0:v:0", "-an", "-sn", "-dn", "-vf", "scale=160:160:force_original_aspect_ratio=decrease", "-frames:v", "1", "-threads", "1", "-c:v", "bmp", "-f", "image2pipe", "pipe:1")
		if e != nil {
			return fmt.Errorf("%s actual frame extraction failed", name)
		}
		cfg, _, e := image.DecodeConfig(bytes.NewReader(out))
		if e != nil || cfg.Width*cfg.Height > 160*160 {
			return errors.New("thumbnail output invalid")
		}
		img, _, e := image.Decode(bytes.NewReader(out))
		if e != nil {
			return e
		}
		w, h := 160, 120
		if name == "rotated.mov" {
			w, h = 120, 160
		}
		if img.Bounds().Dx() != w || img.Bounds().Dy() != h {
			return fmt.Errorf("%s rotation/dimensions incorrect: got %dx%d", name, img.Bounds().Dx(), img.Bounds().Dy())
		}
		r, g, bl, _ := img.At(w/2, h/2).RGBA()
		if r < 55000 || g > 8000 || bl > 8000 {
			return fmt.Errorf("%s decoded pixel mismatch", name)
		}
		f, e := os.Create(filepath.Join(root, name+".preview.png"))
		if e != nil {
			return e
		}
		e = png.Encode(f, img)
		f.Close()
		if e != nil {
			return e
		}
		after, e := os.ReadFile(path)
		if e != nil || sha256.Sum256(after) != hash {
			return errors.New("original modified")
		}
	}
	// A grid must either reconstruct every tile correctly or fail explicitly.
	gridFile := filepath.Join(root, "grid.heic")
	original, e := os.ReadFile(gridFile)
	if e != nil {
		return e
	}
	before := sha256.Sum256(original)
	if hex.EncodeToString(before[:]) != manifest["grid.heic"] {
		return errors.New("grid fixture hash mismatch")
	}
	gridOut, gridErr := command(exe, "-hide_banner", "-loglevel", "error", "-nostdin", "-max_alloc", "67108864", "-threads", "1", "-protocol_whitelist", "file,pipe", "-i", gridFile, "-filter_complex", "[0:g:0]scale=160:160:force_original_aspect_ratio=decrease[thumb]", "-map", "[thumb]", "-an", "-sn", "-dn", "-frames:v", "1", "-threads", "1", "-c:v", "bmp", "-f", "image2pipe", "pipe:1")
	if requireGrid {
		if gridErr != nil {
			return errors.New("required HEIC grid reconstruction failed")
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(gridOut))
		if err != nil || cfg.Width != 160 || cfg.Height != 120 {
			return errors.New("HEIC grid dimensions wrong")
		}
		img, _, err := image.Decode(bytes.NewReader(gridOut))
		if err != nil {
			return err
		}
		for i, p := range []image.Point{{20, 20}, {140, 20}, {20, 100}, {140, 100}} {
			r, g, b, _ := img.At(p.X, p.Y).RGBA()
			values := []uint32{r, g, b}
			for c, value := range values {
				bright := i == 3 || c == i
				if (bright && value < 55000) || (!bright && value > 8000) {
					return errors.New("HEIC grid quadrant pixels wrong")
				}
			}
		}
	} else if gridErr == nil {
		return errors.New("older candidate unexpectedly accepted grid; verify all tiles before claiming support")
	}
	after, e := os.ReadFile(gridFile)
	if e != nil || sha256.Sum256(after) != before {
		return errors.New("grid original modified")
	}
	bad := filepath.Join(root, "malformed.heic")
	if e = os.WriteFile(bad, []byte("not a valid HEIF container"), 0600); e != nil {
		return e
	}
	if _, e = command(probe, "-v", "error", "-show_format", "-of", "json", bad); e == nil {
		return errors.New("malformed accepted")
	}
	if _, e = command(probe, "-v", "error", "-show_format", "-of", "json", "http://127.0.0.1:9/disabled"); e == nil {
		return errors.New("network protocol unexpectedly enabled")
	}
	return nil
}

type service struct {
	root        string
	requireGrid bool
}

func (p service) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending, WaitHint: 120000}
	result := make(chan error, 1)
	go func() { result <- run(p.root, p.requireGrid) }()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	checkpoint := uint32(1)
	for {
		select {
		case e := <-result:
			if e != nil {
				_ = os.WriteFile(filepath.Join(p.root, "failed.txt"), []byte(e.Error()), 0600)
				return false, 1
			}
			if os.WriteFile(filepath.Join(p.root, "passed.txt"), []byte(fmt.Sprintf("actual media decode and hashes passed; job limit %d; observed peak committed bytes %d", jobMemoryLimit, peakCommit.Load())), 0600) != nil {
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
		case <-ticker.C:
			checkpoint++
			status <- svc.Status{State: svc.StartPending, CheckPoint: checkpoint, WaitHint: 120000}
		}
	}
}
func main() {
	root := flag.String("root", "", "disposable fixture directory")
	stall := flag.Bool("probe-stall", false, "synthetic child timeout fixture")
	flood := flag.Bool("probe-flood", false, "synthetic output bound fixture")
	requireGrid := flag.Bool("require-grid", false, "require full grid reconstruction")
	helper := flag.String("probe-helper", "", "synthetic containment operation")
	flag.Parse()
	if *helper != "" {
		helperMode(*helper, *root)
		return
	}
	if *stall {
		time.Sleep(10 * time.Minute)
		os.Exit(0)
	}
	if *flood {
		b := make([]byte, 4096)
		for i := 0; i < 512; i++ {
			if _, e := os.Stdout.Write(b); e != nil {
				os.Exit(1)
			}
		}
		os.Exit(0)
	}
	if *root == "" {
		os.Exit(2)
	}
	if svc.Run("ImmichCloneMediaProbe", service{*root, *requireGrid}) != nil {
		os.Exit(1)
	}
}
