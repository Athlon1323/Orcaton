//go:build !windows

package media

import "os/exec"

func hideWindow(*exec.Cmd) {}
