//go:build !linux && !darwin && !windows

package sysres

func TotalRAM() uint64 { return 0 }

func DiskKind(string) string { return DiskUnknown }

func PeakRSS() uint64 { return 0 }
