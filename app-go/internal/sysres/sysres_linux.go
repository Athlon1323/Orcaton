package sysres

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func TotalRAM() uint64 {
	var total uint64
	if f, err := os.Open("/proc/meminfo"); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			if v, ok := strings.CutPrefix(sc.Text(), "MemTotal:"); ok {
				kb, _ := strconv.ParseUint(strings.TrimSuffix(strings.TrimSpace(v), " kB"), 10, 64)
				total = kb * 1024
				break
			}
		}
		f.Close()
	}
	for _, p := range []string{"/sys/fs/cgroup/memory.max", "/sys/fs/cgroup/memory/memory.limit_in_bytes"} {
		if b, err := os.ReadFile(p); err == nil {
			if v, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64); err == nil &&
				v > 0 && (total == 0 || v < total) {
				total = v
			}
		}
	}
	return total
}

func DiskKind(path string) string {
	var st syscall.Stat_t
	if err := syscall.Stat(path, &st); err != nil {
		return DiskUnknown
	}
	dev64 := uint64(st.Dev)
	maj := (dev64&0x00000000000fff00)>>8 | (dev64&0xfffff00000000000)>>32
	mnr := dev64&0xff | (dev64&0x00000ffffff00000)>>12
	dev, err := filepath.EvalSymlinks("/sys/dev/block/" + strconv.FormatUint(maj, 10) + ":" +
		strconv.FormatUint(mnr, 10))
	if err != nil {
		return DiskUnknown
	}
	for _, d := range []string{dev, filepath.Dir(dev)} {
		if b, err := os.ReadFile(filepath.Join(d, "queue", "rotational")); err == nil {
			if strings.TrimSpace(string(b)) == "1" {
				return DiskHDD
			}
			return DiskSSD
		}
	}
	return DiskUnknown
}

func PeakRSS() uint64 {
	var ru syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &ru) != nil {
		return 0
	}
	return uint64(ru.Maxrss) * 1024
}
