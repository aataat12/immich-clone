// Package recognition runs the optional, pinned local CPU provider.
package recognition

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/image/draw"
	"image"
	"immichclone/internal/diagnostics"
	"immichclone/internal/media"
	"io"
	"math"
	"os"
	"path/filepath"
	"time"
)

const ProviderID = "opencv-4.12.0-nr01-zlib-1.3.2-15d1c2ca7ef3"
const AnalysisID = ProviderID + "-bgr960-v1"

var ErrUnavailable = errors.New("local AI provider unavailable; originals remain available")
var hashes = map[string]string{
	"ai-worker.exe": "15d1c2ca7ef3f526b34ac4823f9337c51280ad171ddf43df4f3af5d8128a7c58",
	"yunet.onnx":    "8f2383e4dd3cfbb4553ea8718107fc0423210dc964f9f4280604804ed2552fa4",
	"sface.onnx":    "0ba9fbfa01b5270c96627c4ef784da859931e02f04419c829e83484087c34e79",
	"yolox.onnx":    "c5c2d13e59ae883e6af3b45daea64af4833a4951c92d116ec270d9ddbe998063",
}

type Face struct {
	Box        []float64 `json:"box"`
	Confidence float64   `json:"confidence"`
	Embedding  []float64 `json:"embedding"`
}
type Object struct {
	Class      int       `json:"class"`
	Confidence float64   `json:"confidence"`
	Box        []float64 `json:"box"`
}
type Result struct {
	Protocol int      `json:"protocol"`
	Faces    []Face   `json:"faces"`
	Objects  []Object `json:"objects"`
}
type Provider struct {
	dir       string
	available bool
}

func verify(ctx context.Context, dir string) bool {
	for name, want := range hashes {
		f, e := os.Open(filepath.Join(dir, name))
		if e != nil {
			return false
		}
		h := sha256.New()
		buf := make([]byte, 64<<10)
		var total int64
		for {
			if ctx.Err() != nil {
				f.Close()
				return false
			}
			n, e := f.Read(buf)
			total += int64(n)
			if total > 45<<20 {
				f.Close()
				return false
			}
			h.Write(buf[:n])
			if e == io.EOF {
				break
			}
			if e != nil {
				f.Close()
				return false
			}
		}
		f.Close()
		if hex.EncodeToString(h.Sum(nil)) != want {
			return false
		}
	}
	return true
}
func New(dir string) *Provider {
	ctx, c := context.WithTimeout(context.Background(), 10*time.Second)
	defer c()
	p := &Provider{dir: dir, available: verify(ctx, dir)}
	var e error
	if !p.available {
		e = ErrUnavailable
	}
	diagnostics.Record("recognition", "ai_provider_verify", e, 0, 0, 0)
	return p
}
func (p *Provider) Available() bool { return p != nil && p.available }

// Frame uses displayed orientation, RGB composited over black, with a 960px bound.
func Frame(src image.Image) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > 960 || h > 960 {
		if w >= h {
			h = h * 960 / w
			w = 960
		} else {
			w = w * 960 / h
			h = 960
		}
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
	return dst
}
func WriteFrame(w io.Writer, img image.Image) error {
	b := img.Bounds()
	width, height := b.Dx(), b.Dy()
	if width < 32 || height < 32 || width > 960 || height > 960 {
		return errors.New("image too small or large for local analysis")
	}
	out := bufio.NewWriter(w)
	header := make([]byte, 12)
	copy(header, "ICAI")
	binary.LittleEndian.PutUint32(header[4:], uint32(width))
	binary.LittleEndian.PutUint32(header[8:], uint32(height))
	if _, e := out.Write(header); e != nil {
		return e
	}
	row := make([]byte, width*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			row[3*x] = byte(b >> 8)
			row[3*x+1] = byte(g >> 8)
			row[3*x+2] = byte(r >> 8)
		}
		if _, e := out.Write(row); e != nil {
			return e
		}
	}
	return out.Flush()
}
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func box(v []float64, w, h int) bool {
	if len(v) != 4 {
		return false
	}
	for _, x := range v {
		if !finite(x) {
			return false
		}
	}
	return v[2] > 0 && v[3] > 0 && v[0] < float64(w) && v[1] < float64(h) && v[0]+v[2] > 0 && v[1]+v[3] > 0
}
func Validate(r Result, w, h int) error {
	if r.Protocol != 1 || len(r.Faces) > 64 || len(r.Objects) > 100 {
		return errors.New("invalid AI response")
	}
	for _, f := range r.Faces {
		if !box(f.Box, w, h) || !finite(f.Confidence) || f.Confidence < .9 || f.Confidence > 1 || len(f.Embedding) != 128 {
			return errors.New("invalid face response")
		}
		norm := 0.
		for _, v := range f.Embedding {
			if !finite(v) {
				return errors.New("invalid embedding")
			}
			norm += v * v
		}
		if math.Abs(norm-1) > .02 {
			return errors.New("invalid embedding norm")
		}
	}
	for _, o := range r.Objects {
		if !box(o.Box, w, h) || !finite(o.Confidence) || o.Confidence < .5 || o.Confidence > 1 || o.Class < 0 || o.Class >= len(Classes) {
			return errors.New("invalid object response")
		}
	}
	return nil
}
func (p *Provider) Analyze(ctx context.Context, img image.Image, staging string) (result Result, retErr error) {
	started := time.Now()
	defer func() { diagnostics.Record("recognition", "ai_process", retErr, 0, 0, time.Since(started)) }()
	if !p.Available() || !verify(ctx, p.dir) {
		return result, ErrUnavailable
	}
	if img == nil || img.Bounds().Empty() {
		return result, errors.New("empty AI image")
	}
	img = Frame(img)
	f, e := os.CreateTemp(staging, "analysis-*.part")
	if e != nil {
		return result, errors.New("AI working storage unavailable")
	}
	defer os.Remove(f.Name())
	e = WriteFrame(f, img)
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return result, e
	}
	result.Protocol = 1
	for _, task := range []string{"faces", "objects"} {
		if e = ctx.Err(); e != nil {
			return Result{}, e
		}
		startedTask := time.Now()
		child, cancel := context.WithTimeout(ctx, 30*time.Second)
		r, err := media.Run(child, filepath.Join(p.dir, "ai-worker.exe"), []string{task, p.dir, f.Name()}, nil)
		cancel()
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		if err != nil {
			diagnostics.Record("recognition", "ai_"+task, err, 0, 0, time.Since(startedTask))
			return Result{}, fmt.Errorf("local AI %s task failed; check diagnostics before retrying", task)
		}
		var part Result
		if json.Unmarshal(r.Output, &part) != nil || Validate(part, img.Bounds().Dx(), img.Bounds().Dy()) != nil {
			rejected := errors.New("local AI response rejected")
			diagnostics.Record("recognition", "ai_"+task, rejected, 0, 0, time.Since(startedTask))
			return Result{}, fmt.Errorf("local AI %s response rejected", task)
		}
		diagnostics.Record("recognition", "ai_"+task, nil, 0, 0, time.Since(startedTask))
		if task == "faces" {
			result.Faces = part.Faces
		} else {
			result.Objects = part.Objects
		}
	}
	// Persist display-relative geometry, independent of the temporary analysis resolution.
	norm := func(b []float64) {
		x := math.Max(0, b[0])
		y := math.Max(0, b[1])
		right := math.Min(float64(img.Bounds().Dx()), b[0]+b[2])
		bottom := math.Min(float64(img.Bounds().Dy()), b[1]+b[3])
		b[0] = x / float64(img.Bounds().Dx())
		b[1] = y / float64(img.Bounds().Dy())
		b[2] = (right - x) / float64(img.Bounds().Dx())
		b[3] = (bottom - y) / float64(img.Bounds().Dy())
	}
	for i := range result.Faces {
		norm(result.Faces[i].Box)
	}
	for i := range result.Objects {
		norm(result.Objects[i].Box)
	}
	return result, nil
}

var Classes = []string{"person", "bicycle", "car", "motorcycle", "airplane", "bus", "train", "truck", "boat", "traffic light", "fire hydrant", "stop sign", "parking meter", "bench", "bird", "cat", "dog", "horse", "sheep", "cow", "elephant", "bear", "zebra", "giraffe", "backpack", "umbrella", "handbag", "tie", "suitcase", "frisbee", "skis", "snowboard", "sports ball", "kite", "baseball bat", "baseball glove", "skateboard", "surfboard", "tennis racket", "bottle", "wine glass", "cup", "fork", "knife", "spoon", "bowl", "banana", "apple", "sandwich", "orange", "broccoli", "carrot", "hot dog", "pizza", "donut", "cake", "chair", "couch", "potted plant", "bed", "dining table", "toilet", "tv", "laptop", "mouse", "remote", "keyboard", "cell phone", "microwave", "oven", "toaster", "sink", "refrigerator", "book", "clock", "vase", "scissors", "teddy bear", "hair drier", "toothbrush"}
