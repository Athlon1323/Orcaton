package media

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type FFmpegNotFound struct{ Msg string }

func (e *FFmpegNotFound) Error() string { return e.Msg }

var fallbackDirs = defaultFallbackDirs(runtime.GOOS, os.Getenv)

func defaultFallbackDirs(goos string, env func(string) string) []string {
	if goos != "windows" {
		return []string{"/opt/homebrew/bin", "/usr/local/bin", "/usr/bin"}
	}
	var dirs []string
	add := func(base string, rel ...string) {
		if base != "" {
			dirs = append(dirs, filepath.Join(append([]string{base}, rel...)...))
		}
	}
	add(env("LOCALAPPDATA"), "Microsoft", "WinGet", "Links")
	add(env("ProgramFiles"), "WinGet", "Links")
	add(env("USERPROFILE"), "scoop", "shims")
	add(env("ProgramData"), "chocolatey", "bin")
	return dirs
}

const MinFFmpeg = "4.4"

func FFmpegVersionOld(v string) bool {
	m := versionRe.FindStringSubmatch(v)
	if m == nil {
		return false
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	return major < 4 || major == 4 && minor < 4
}

var versionRe = regexp.MustCompile(`^n?(\d+)\.(\d+)`)

var installHint = map[string]string{
	"windows": "winget install Gyan.FFmpeg",
	"darwin":  "brew install ffmpeg",
	"linux":   "sudo apt install ffmpeg  (or dnf/pacman)",
}

var executable = os.Executable

func bundledDirs() []string {
	var dirs []string
	if exe, err := executable(); err == nil {
		if r, err := filepath.EvalSymlinks(exe); err == nil {
			exe = r
		}
		d := filepath.Dir(exe)
		dirs = append(dirs, d)
		if runtime.GOOS == "darwin" {
			dirs = append(dirs, filepath.Join(filepath.Dir(d), "Resources"))
		}
	}
	if runtime.GOOS == "linux" {
		if a := os.Getenv("APPDIR"); a != "" {
			dirs = append(dirs, filepath.Join(a, "usr", "bin"))
		}
	}
	return dirs
}

func findIn(d, name string) string {
	for _, cand := range []string{filepath.Join(d, name), filepath.Join(d, name+".exe")} {
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			return cand
		}
	}
	return ""
}

func Bin(name string) (string, error) {
	if d := os.Getenv("ORCATON_FFMPEG_DIR"); d != "" {
		if c := findIn(d, name); c != "" {
			return c, nil
		}
	}
	for _, d := range bundledDirs() {
		if c := findIn(d, name); c != "" {
			return c, nil
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	for _, d := range fallbackDirs {
		if c := findIn(d, name); c != "" {
			return c, nil
		}
	}
	hint, ok := installHint[runtime.GOOS]
	if !ok {
		hint = "install FFmpeg"
	}
	return "", &FFmpegNotFound{fmt.Sprintf("%s not found. Install: %s; or set ORCATON_FFMPEG_DIR.", name, hint)}
}

func Probe(path string) (map[string]any, error) {
	b, err := Bin("ffprobe")
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(b, "-v", "error", "-print_format", "json", "-show_format",
		"-show_streams", path)
	hideWindow(cmd)
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffprobe: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	var res map[string]any
	if err := json.Unmarshal(bytes.ToValidUTF8(out.Bytes(), []byte("�")), &res); err != nil {
		return nil, err
	}
	return res, nil
}

type AudioOpts struct {
	Stream   *int
	Channel  *int
	Start    *float64
	Duration *float64
}

func ExtractAudio(path string, sampleRate int, o AudioOpts) ([]float32, error) {
	return ExtractAudioCtx(context.Background(), path, sampleRate, o)
}

func ExtractAudioCtx(ctx context.Context, path string, sampleRate int, o AudioOpts) ([]float32, error) {
	var x []float32
	err := StreamAudioCtx(ctx, path, sampleRate, o, func(p []float32) { x = append(x, p...) })
	if err != nil {
		return nil, err
	}
	return x, nil
}

func audioArgs(path string, sampleRate int, o AudioOpts) []string {
	args := []string{"-v", "error", "-nostdin"}
	if o.Start != nil {
		args = append(args, "-ss", fmt.Sprintf("%.6f", *o.Start))
	}
	args = append(args, "-i", path)
	if o.Duration != nil {
		args = append(args, "-t", fmt.Sprintf("%.6f", *o.Duration))
	}
	args = append(args, "-vn", "-sn", "-dn")
	if o.Stream != nil {
		args = append(args, "-map", fmt.Sprintf("0:a:%d", *o.Stream))
	}
	if o.Channel != nil {
		args = append(args, "-af", fmt.Sprintf("pan=mono|c0=c%d", *o.Channel))
	} else {
		args = append(args, "-ac", "1")
	}
	return append(args, "-ar", fmt.Sprint(sampleRate), "-f", "f32le", "-acodec", "pcm_f32le", "pipe:1")
}

const streamChunk = 64 << 10

func StreamAudioCtx(ctx context.Context, path string, sampleRate int, o AudioOpts,
	sink func([]float32)) error {
	b, err := Bin("ffmpeg")
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, b, audioArgs(path, sampleRate, o)...)
	hideWindow(cmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("ffmpeg: %v", err)
	}
	ch := make(chan []float32, 32)
	var rerr error
	go func() {
		defer close(ch)
		raw := make([]byte, streamChunk)
		have := 0
		for {
			n, e := out.Read(raw[have:])
			have += n
			if k := have / 4; k > 0 {
				x := make([]float32, k)
				for i := range x {
					x[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[4*i:]))
				}
				ch <- x
				have = copy(raw, raw[4*k:have])
			}
			if e != nil {
				if e != io.EOF {
					rerr = e
				}
				return
			}
		}
	}()
	for x := range ch {
		sink(x)
	}
	werr := cmd.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if werr != nil {
		return fmt.Errorf("ffmpeg: %v: %s", werr, strings.TrimSpace(stderr.String()))
	}
	return rerr
}

func RunFFmpeg(args ...string) error {
	b, err := Bin("ffmpeg")
	if err != nil {
		return err
	}
	cmd := exec.Command(b, args...)
	hideWindow(cmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

var ffmpegVer struct {
	sync.Mutex
	v string
}

func FFmpegVersion() string {
	ffmpegVer.Lock()
	defer ffmpegVer.Unlock()
	if ffmpegVer.v == "" {
		if b, err := Bin("ffmpeg"); err == nil {
			ffmpegVer.v, _ = ToolVersion(b)
		}
	}
	return ffmpegVer.v
}

func ToolVersion(bin string) (string, error) {
	cmd := exec.Command(bin, "-version")
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s -version: %v", filepath.Base(bin), err)
	}
	line, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	line = strings.TrimSpace(line)
	if f := strings.Fields(line); len(f) >= 3 && f[1] == "version" {
		return f[2], nil
	}
	return line, nil
}
