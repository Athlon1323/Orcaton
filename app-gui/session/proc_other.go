//go:build !windows

package session

import "os/exec"

func hideConsole(*exec.Cmd) {}
