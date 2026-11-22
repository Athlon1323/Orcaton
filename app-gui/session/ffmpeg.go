package session

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"orcaton/api"
)

const EventFFmpeg = "ffmpeg:install"

type FFmpegPlan struct {
	OS      string `json:"os"`
	Tool    string `json:"tool"`
	Command string `json:"command"`
	Button  bool   `json:"button"`
	Admin   bool   `json:"admin"`
	Problem string `json:"problem"`
	argv    []string
}

type lookFn func(string) (string, error)

var brewDirs = []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"}

func planFFmpeg(goos string, look lookFn) FFmpegPlan {
	p := FFmpegPlan{OS: goos}
	has := func(n string) string {
		if path, err := look(n); err == nil {
			return path
		}
		return ""
	}
	switch goos {
	case "windows":
		p.Tool, p.Command = "winget", "winget install --id Gyan.FFmpeg -e"
		if w := has("winget"); w != "" {
			p.Button = true
			p.argv = []string{w, "install", "--id", "Gyan.FFmpeg", "-e", "--source", "winget",
				"--accept-source-agreements", "--accept-package-agreements", "--disable-interactivity"}
		} else {
			p.Problem = "no_winget"
		}
	case "darwin":
		p.Tool, p.Command = "brew", "brew install ffmpeg"
		b := has("brew")
		for _, d := range brewDirs {
			if b == "" {
				b = has(d)
			}
		}
		if b != "" {
			p.Button, p.argv = true, []string{b, "install", "ffmpeg"}
		} else {
			p.Problem = "no_brew"
		}
	default:
		managers := []struct {
			tool, bin string
			args      []string
		}{
			{"apt", "apt-get", []string{"install", "-y", "ffmpeg"}},
			{"dnf", "dnf", []string{"install", "-y", "/usr/bin/ffmpeg"}},
			{"pacman", "pacman", []string{"-S", "--noconfirm", "ffmpeg"}},
			{"zypper", "zypper", []string{"--non-interactive", "install", "ffmpeg"}},
		}
		for _, m := range managers {
			bin := has(m.bin)
			if bin == "" {
				continue
			}
			p.Tool = m.tool
			p.Command = "sudo " + m.bin + " " + strings.Join(m.args, " ")
			if pk := has("pkexec"); pk != "" {
				p.Button, p.Admin = true, true
				p.argv = append([]string{pk, bin}, m.args...)
			} else {
				p.Problem = "no_pkexec"
			}
			return p
		}
		p.Problem = "no_manager"
	}
	return p
}

func (s *Session) FFmpegPlan() FFmpegPlan { return planFFmpeg(runtime.GOOS, exec.LookPath) }

type InstallReply struct {
	FFmpeg *api.ErrorInfo `json:"ffmpeg,omitempty"`
	Error  *api.ErrorInfo `json:"error,omitempty"`
}

func (s *Session) InstallFFmpeg() (r InstallReply) {
	defer s.recoverTo(&r.Error)
	ctx, end, busy := s.begin("ffmpeg")
	if busy != nil {
		r.Error = busy
		return r
	}
	defer end()
	p := s.FFmpegPlan()
	if !p.Button {
		r.Error = &api.ErrorInfo{Kind: api.ErrNoFFmpeg, Message: "cannot install with the button (" + p.Problem + "): " + p.Command}
		r.FFmpeg = s.CheckFFmpeg()
		return r
	}
	s.log.Printf("ffmpeg install: %s", strings.Join(p.argv, " "))
	if err := runLines(ctx, p.argv, func(line string) { s.log.Printf("ffmpeg> %s", line); s.ev.em.Emit(EventFFmpeg, line) }); err != nil {
		if ctx.Err() != nil {
			r.Error = &api.ErrorInfo{Kind: api.ErrCancelled, Message: "FFmpeg installation stopped"}
		} else {
			r.Error = &api.ErrorInfo{Kind: api.ErrFailed, Message: fmt.Sprintf("%s: %v", p.Tool, err),
				Hint: "Run the command in a terminal: " + p.Command}
		}
	}
	r.FFmpeg = s.CheckFFmpeg()
	if r.FFmpeg == nil {
		r.Error = nil
	}
	s.log.Printf("ffmpeg install: found=%v", r.FFmpeg == nil)
	return r
}

func runLines(ctx context.Context, argv []string, line func(string)) error {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	hideConsole(cmd)
	pr, pw, err := os.Pipe()
	if err != nil {
		return err
	}
	cmd.Stdout, cmd.Stderr = pw, pw
	if err := cmd.Start(); err != nil {
		pr.Close()
		pw.Close()
		return err
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		sc.Split(splitCRLF)
		for sc.Scan() {
			if t := strings.TrimSpace(sc.Text()); t != "" && !spinnerOnly(t) {
				line(t)
			}
		}
	}()
	werr := cmd.Wait()
	pw.Close()
	wg.Wait()
	pr.Close()
	var ee *exec.ExitError
	if errors.As(werr, &ee) {
		return fmt.Errorf("exit code %d", ee.ExitCode())
	}
	return werr
}

func splitCRLF(data []byte, atEOF bool) (int, []byte, error) {
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func spinnerOnly(s string) bool { return strings.Trim(s, `-\|/ `) == "" }
