package sysres

import (
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	psapi                    = syscall.NewLazyDLL("psapi.dll")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
	procDeviceIoControl      = kernel32.NewProc("DeviceIoControl")
	procGetProcessMemoryInfo = psapi.NewProc("GetProcessMemoryInfo")
)

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func TotalRAM() uint64 {
	var m memoryStatusEx
	m.Length = uint32(unsafe.Sizeof(m))
	if r, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&m))); r == 0 {
		return 0
	}
	return m.TotalPhys
}

func DiskKind(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return DiskUnknown
	}
	vol := filepath.VolumeName(abs)
	if len(vol) != 2 || vol[1] != ':' || strings.HasPrefix(abs, `\\`) {
		return DiskUnknown
	}
	name, err := syscall.UTF16PtrFromString(`\\.\` + vol)
	if err != nil {
		return DiskUnknown
	}
	h, err := syscall.CreateFile(name, 0, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil,
		syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		return DiskUnknown
	}
	defer syscall.CloseHandle(h)
	const ioctlStorageQueryProperty = 0x002D1400
	query := [3]uint32{7, 0, 0}
	var desc struct {
		Version, Size uint32
		SeekPenalty   byte
		_             [3]byte
	}
	var got uint32
	r, _, _ := procDeviceIoControl.Call(uintptr(h), ioctlStorageQueryProperty,
		uintptr(unsafe.Pointer(&query[0])), unsafe.Sizeof(query),
		uintptr(unsafe.Pointer(&desc)), unsafe.Sizeof(desc), uintptr(unsafe.Pointer(&got)), 0)
	if r == 0 || got < 9 {
		return DiskUnknown
	}
	if desc.SeekPenalty != 0 {
		return DiskHDD
	}
	return DiskSSD
}

type processMemoryCounters struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
}

func PeakRSS() uint64 {
	var c processMemoryCounters
	c.CB = uint32(unsafe.Sizeof(c))
	h, err := syscall.GetCurrentProcess()
	if err != nil {
		return 0
	}
	if r, _, _ := procGetProcessMemoryInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&c)),
		uintptr(c.CB)); r == 0 {
		return 0
	}
	return uint64(c.PeakWorkingSetSize)
}
