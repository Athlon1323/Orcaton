package sysres

import (
	"fmt"
	"runtime"
)

const (
	MiB = 1 << 20
	GiB = 1 << 30
)

const (
	budgetMin  = 1 * GiB
	budgetMax  = 7 * GiB
	unknownRAM = 4 * GiB
)

func MemoryBudget(total uint64) uint64 {
	if total == 0 {
		total = unknownRAM
	}
	b := total / 4 * 3
	return min(max(b, budgetMin), budgetMax)
}

const (
	DiskSSD     = "ssd"
	DiskHDD     = "hdd"
	DiskUnknown = ""
)

const jobMem = 256 * MiB

func AutoJobs(cpus int, disk string, memLimit uint64) int {
	n := min(max(cpus/2, 2), 16)
	if disk == DiskHDD {
		n = min(n, 4)
	}
	if memLimit > 0 {
		n = min(n, max(int(memLimit/jobMem), 1))
	}
	return n
}

type Resources struct {
	RAM      uint64
	MemLimit uint64
	Threads  int
	Jobs     int
	Disk     string
	FFmpeg   string
}

func Resolve(memLimitMB, threads, jobs int, mediaPath string) Resources {
	r := Resources{RAM: TotalRAM(), Threads: threads, Jobs: jobs}
	if memLimitMB > 0 {
		r.MemLimit = uint64(memLimitMB) * MiB
	} else {
		r.MemLimit = MemoryBudget(r.RAM)
	}
	if r.Threads <= 0 {
		r.Threads = max(runtime.NumCPU(), 1)
	}
	if mediaPath != "" {
		r.Disk = DiskKind(mediaPath)
	}
	if r.Jobs <= 0 {
		r.Jobs = AutoJobs(runtime.NumCPU(), r.Disk, r.MemLimit)
	}
	return r
}

func (r Resources) String() string {
	ram := "?"
	if r.RAM > 0 {
		ram = fmt.Sprintf("%.1f ГБ", float64(r.RAM)/GiB)
	}
	disk := r.Disk
	if disk == "" {
		disk = "диск ?"
	}
	s := fmt.Sprintf("ОЗУ %s → потолок %.1f ГБ, потоков %d, ffmpeg %d (%s)", ram,
		float64(r.MemLimit)/GiB, r.Threads, r.Jobs, disk)
	if r.FFmpeg != "" {
		s += ", FFmpeg " + r.FFmpeg
	}
	return s
}
