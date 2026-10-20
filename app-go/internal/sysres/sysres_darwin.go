package sysres

import (
	"encoding/binary"
	"os/exec"
	"strings"
	"syscall"
)

func TotalRAM() uint64 {
	s, err := syscall.Sysctl("hw.memsize")
	if err != nil || len(s) > 8 {
		return 0
	}
	b := make([]byte, 8)
	copy(b, s)
	return binary.LittleEndian.Uint64(b)
}

func DiskKind(path string) string {
	out, err := exec.Command("diskutil", "info", path).Output()
	if err != nil {
		return DiskUnknown
	}
	for _, ln := range strings.Split(string(out), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(ln), "Solid State:"); ok {
			if strings.TrimSpace(v) == "Yes" {
				return DiskSSD
			}
			return DiskHDD
		}
	}
	return DiskUnknown
}

func PeakRSS() uint64 {
	var ru syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &ru) != nil {
		return 0
	}
	return uint64(ru.Maxrss)
}
