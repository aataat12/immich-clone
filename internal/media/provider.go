package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	_ "golang.org/x/image/bmp"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var executableHashes = map[string]string{
	"ffmpeg.exe":  "7eb77d88e2e97afff0844d8df1408ec10f9ca66fac58f6971112cefd25fb6a0f",
	"ffprobe.exe": "0ba24b17e5b42e02de441fb053dacca08aa52ac4f7f38e0845bda7df6b73a6e6",
}

type Provider struct {
	dir       string
	available bool
}

func New(dir string) *Provider {
	p := &Provider{dir: dir}
	for name, expected := range executableHashes {
		f, e := os.Open(filepath.Join(dir, name))
		if e != nil {
			return p
		}
		h := sha256.New()
		_, e = io.Copy(h, f)
		f.Close()
		if e != nil || hex.EncodeToString(h.Sum(nil)) != expected {
			return p
		}
	}
	p.available = true
	return p
}

type stream struct {
	Index         int
	CodecType     string `json:"codec_type"`
	Codec         string `json:"codec_name"`
	Width, Height int
	PixFmt        string `json:"pix_fmt"`
	Transfer      string `json:"color_transfer"`
	SideData      []struct {
		Rotation int `json:"rotation"`
	} `json:"side_data_list"`
}
type probeInfo struct {
	Streams []stream `json:"streams"`
	Groups  []struct {
		Index      int
		Type       string
		Components []struct {
			Width, Height int
			Tiles         int `json:"nb_tiles"`
		} `json:"components"`
	} `json:"stream_groups"`
}

func dimensions(b []byte) (w, h int, selector string, err error) {
	var info probeInfo
	if json.Unmarshal(b, &info) != nil || len(info.Streams) == 0 {
		return 0, 0, "", errors.New("media metadata unsupported")
	}
	for _, s := range info.Streams {
		if s.Transfer == "smpte2084" || s.Transfer == "arib-std-b67" || strings.Contains(s.PixFmt, "10") || strings.Contains(s.PixFmt, "12") || strings.Contains(s.PixFmt, "16") {
			return 0, 0, "", errors.New("HDR/high-bit-depth preview unsupported; original remains available")
		}
	}
	selected := -1
	for i, s := range info.Streams {
		if s.CodecType == "video" {
			selected = i
			break
		}
	}
	if selected < 0 {
		return 0, 0, "", errors.New("video stream unavailable")
	}
	s := info.Streams[selected]
	selector = fmt.Sprintf("0:%d", s.Index)
	w, h = s.Width, s.Height
	if s.Codec != "h264" && s.Codec != "hevc" {
		return 0, 0, "", errors.New("video/image codec unsupported; original remains available")
	}
	if len(s.SideData) > 0 && (s.SideData[0].Rotation%180 != 0) {
		w, h = h, w
	}
	for _, g := range info.Groups {
		if strings.EqualFold(g.Type, "Tile Grid") && len(g.Components) > 0 {
			c := g.Components[0]
			if c.Tiles <= 0 || c.Tiles > 256 {
				return 0, 0, "", errors.New("HEIC tile limit exceeded")
			}
			w, h = c.Width, c.Height
			selector = fmt.Sprintf("[0:g:%d]", g.Index)
			break
		}
	}
	if w <= 0 || h <= 0 || w > 40_000_000 || h > 40_000_000 || int64(w)*int64(h) > 40_000_000 {
		return 0, 0, "", errors.New("media exceeds 40 megapixel preview limit; original remains available")
	}
	return
}
func (p *Provider) Thumbnail(parent context.Context, path string) (image.Image, int, int, error) {
	if !p.available {
		return nil, 0, 0, ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	run := func(name string, args ...string) ([]byte, error) {
		r, e := Run(ctx, filepath.Join(p.dir, name), args, nil)
		if e != nil {
			if parent.Err() != nil {
				return nil, parent.Err()
			}
			if ctx.Err() != nil {
				return nil, errors.New("media preview time limit exceeded; original remains available")
			}
			return nil, errors.New("media preview failed; original remains available")
		}
		return r.Output, nil
	}
	b, e := run("ffprobe.exe", "-v", "error", "-max_alloc", "67108864", "-probesize", "8388608", "-analyzeduration", "5000000", "-protocol_whitelist", "file,pipe", "-show_streams", "-show_stream_groups", "-of", "json", path)
	if e != nil {
		return nil, 0, 0, e
	}
	w, h, selector, e := dimensions(b)
	if e != nil {
		return nil, 0, 0, e
	}
	args := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-max_alloc", "67108864", "-threads", "1", "-filter_threads", "1", "-filter_complex_threads", "1", "-protocol_whitelist", "file,pipe", "-i", path}
	if strings.HasPrefix(selector, "[") {
		args = append(args, "-filter_complex", selector+"scale=320:320:force_original_aspect_ratio=decrease[thumb]", "-map", "[thumb]")
	} else {
		args = append(args, "-map", selector, "-vf", "scale=320:320:force_original_aspect_ratio=decrease")
	}
	args = append(args, "-an", "-sn", "-dn", "-frames:v", "1", "-threads", "1", "-c:v", "bmp", "-f", "image2pipe", "pipe:1")
	b, e = run("ffmpeg.exe", args...)
	if e != nil {
		return nil, 0, 0, e
	}
	cfg, _, e := image.DecodeConfig(bytes.NewReader(b))
	if e != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 320 || cfg.Height > 320 {
		return nil, 0, 0, errors.New("invalid media thumbnail")
	}
	img, _, e := image.Decode(bytes.NewReader(b))
	if e != nil {
		return nil, 0, 0, errors.New("invalid media thumbnail")
	}
	return img, w, h, nil
}
