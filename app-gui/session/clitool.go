package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	posixpath "path"
	"path/filepath"
	"runtime"
	"strings"

	"orcaton/api"
)

type CLITool struct {
	Available bool   `json:"available"`
	Installed bool   `json:"installed"`
	Link      string `json:"link"`
	Target    string `json:"target"`
	Other     string `json:"other,omitempty"`
	Problem   string `json:"problem,omitempty"`
}

type CLIToolReply struct {
	Tool  CLITool        `json:"tool"`
	Error *api.ErrorInfo `json:"error,omitempty"`
}

var (
	cliLink       = "/usr/local/bin/orcaton"
	cliExecutable = os.Executable
	cliRunAdmin   = runAdmin
	cliGOOS       = runtime.GOOS
)

func cliTool(goos string) CLITool {
	t := CLITool{Link: cliLink}
	if goos != "darwin" {
		return t
	}
	t.Available = true
	exe, err := cliExecutable()
	if err != nil {
		t.Problem = "no_binary"
		return t
	}
	if r, err := filepath.EvalSymlinks(exe); err == nil {
		exe = r
	}
	t.Target = filepath.Join(filepath.Dir(exe), "orcaton")
	if strings.HasPrefix(exe, "/Volumes/") || strings.Contains(exe, "/AppTranslocation/") {
		t.Problem = "not_in_applications"
	} else if fi, err := os.Stat(t.Target); err != nil || fi.IsDir() {
		t.Problem = "no_binary"
	}
	fi, err := os.Lstat(t.Link)
	switch {
	case err != nil:
	case fi.Mode()&os.ModeSymlink == 0:
		if t.Problem == "" {
			t.Problem = "occupied"
		}
	default:
		dest, _ := os.Readlink(t.Link)
		if dest != "" && !filepath.IsAbs(dest) {
			dest = filepath.Join(filepath.Dir(t.Link), dest)
		}
		if samePath(dest, t.Target) {
			t.Installed = true
		} else {
			t.Other = dest
		}
	}
	return t
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if ra, err := filepath.EvalSymlinks(a); err == nil {
		a = ra
	}
	if rb, err := filepath.EvalSymlinks(b); err == nil {
		b = rb
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func (s *Session) CLITool() CLITool { return cliTool(cliGOOS) }

func (s *Session) InstallCLITool(prompt string) CLIToolReply {
	return s.changeCLITool("install", prompt)
}

func (s *Session) RemoveCLITool(prompt string) CLIToolReply {
	return s.changeCLITool("remove", prompt)
}

func (s *Session) changeCLITool(action, prompt string) (r CLIToolReply) {
	defer s.recoverTo(&r.Error)
	ctx, end, busy := s.begin("cli-tool")
	if busy != nil {
		r.Tool, r.Error = s.CLITool(), busy
		return r
	}
	defer end()
	t := s.CLITool()
	sh, err := cliShell(action, t)
	if err != nil {
		r.Tool, r.Error = t, &api.ErrorInfo{Kind: api.ErrFailed, Message: err.Error()}
		return r
	}
	s.log.Printf("cli-tool %s: %s", action, sh)
	out, err := cliRunAdmin(ctx, sh, prompt)
	r.Tool = s.CLITool()
	switch {
	case err != nil && (ctx.Err() != nil || strings.Contains(out, "-128")):
		r.Error = &api.ErrorInfo{Kind: api.ErrCancelled, Message: "the password window was closed — nothing changed"}
	case err != nil:
		r.Error = &api.ErrorInfo{Kind: api.ErrFailed, Message: fmt.Sprintf("%v: %s", err, strings.TrimSpace(out)),
			Hint: "In Terminal: sudo " + sh}
	case action == "install" && !r.Tool.Installed, action == "remove" && r.Tool.Installed:
		r.Error = &api.ErrorInfo{Kind: api.ErrFailed, Message: "the command ran, but " + t.Link + " did not change",
			Hint: "In Terminal: sudo " + sh}
	}
	s.log.Printf("cli-tool %s: installed=%v", action, r.Tool.Installed)
	return r
}

func cliShell(action string, t CLITool) (string, error) {
	if !t.Available {
		return "", errors.New("the orcaton command is installed this way only on macOS")
	}
	switch action {
	case "install":
		if t.Problem != "" {
			return "", errors.New("cannot install: " + t.Problem)
		}
		return "mkdir -p " + shQuote(posixpath.Dir(t.Link)) + " && ln -sfn " + shQuote(t.Target) + " " + shQuote(t.Link), nil
	case "remove":
		if !t.Installed {
			return "", errors.New(t.Link + " does not point to this program — leaving it alone")
		}
		return "rm -f " + shQuote(t.Link), nil
	}
	return "", errors.New("unknown action: " + action)
}

func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func asQuote(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
}

func adminScript(sh, prompt string) string {
	s := "do shell script " + asQuote(sh)
	if prompt != "" {
		s += " with prompt " + asQuote(prompt)
	}
	return s + " with administrator privileges"
}

func runAdmin(ctx context.Context, sh, prompt string) (string, error) {
	out, err := exec.CommandContext(ctx, "/usr/bin/osascript", "-e", adminScript(sh, prompt)).CombinedOutput()
	return string(out), err
}
