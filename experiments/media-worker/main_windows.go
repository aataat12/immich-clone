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
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type limitedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.Len() {
		return 0, errors.New("output limit")
	}
	return b.Buffer.Write(p)
}
func command(exe string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Env = []string{"SystemRoot=" + os.Getenv("SystemRoot"), "PATH=" + filepath.Dir(exe)}
	stdout := &limitedBuffer{limit: 1 << 20}
	stderr := &limitedBuffer{limit: 64 << 10}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}
func run(root string) error {
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
		metadata, e := command(probe, "-v", "error", "-protocol_whitelist", "file,pipe", "-select_streams", "v:0", "-show_entries", "stream=codec_name,width,height", "-of", "json", path)
		if e != nil {
			return fmt.Errorf("%s metadata failed", name)
		}
		var info struct {
			Streams []struct {
				Codec         string `json:"codec_name"`
				Width, Height int
			}
		}
		if e = json.Unmarshal(metadata, &info); e != nil || len(info.Streams) != 1 {
			return errors.New("stream inspection failed")
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
			return fmt.Errorf("%s rotation/dimensions incorrect", name)
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

type service struct{ root string }

func (p service) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending, WaitHint: 120000}
	result := make(chan error, 1)
	go func() { result <- run(p.root) }()
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
			if os.WriteFile(filepath.Join(p.root, "passed.txt"), []byte("actual media decode and hashes passed"), 0600) != nil {
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
	flag.Parse()
	if *root == "" {
		os.Exit(2)
	}
	if svc.Run("ImmichCloneMediaProbe", service{*root}) != nil {
		os.Exit(1)
	}
}
