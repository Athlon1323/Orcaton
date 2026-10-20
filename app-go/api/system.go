package api

import (
	"math"

	"orcaton/internal/core"
	"orcaton/internal/media"
	"orcaton/internal/num"
)

func FormatTC(sec float64, seq SequenceInfo) string {
	if seq.FPSNum <= 0 || seq.FPSDen <= 0 || math.IsNaN(sec) || math.IsInf(sec, 0) {
		return "--:--:--:--"
	}
	r := core.RateQ(num.F(seq.FPSNum, seq.FPSDen))
	fr := math.RoundToEven((sec + seq.StartSec) * r.Float())
	return core.FramesToTC(int64(fr), r, false)
}

type FFmpegInfo struct {
	FFmpeg  string `json:"ffmpeg"`
	FFprobe string `json:"ffprobe"`
	Version string `json:"version"`
	Old     bool   `json:"old"`
	Min     string `json:"min"`
}

func GetFFmpegInfo() (FFmpegInfo, error) {
	var fi FFmpegInfo
	var err error
	if fi.FFmpeg, err = media.Bin("ffmpeg"); err != nil {
		return fi, err
	}
	if fi.FFprobe, err = media.Bin("ffprobe"); err != nil {
		return fi, err
	}
	if v, verr := media.ToolVersion(fi.FFmpeg); verr == nil {
		fi.Version = v
	}
	fi.Old, fi.Min = media.FFmpegVersionOld(fi.Version), media.MinFFmpeg
	return fi, nil
}

type CacheInfo struct {
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	Entries int    `json:"entries"`
	Bytes   int64  `json:"bytes"`
	Partial int    `json:"partial"`
}

func GetCacheInfo(dir string) CacheInfo {
	if dir == "" {
		dir = media.DefaultCacheDir()
	}
	st := media.Stats(dir)
	return CacheInfo{Path: st.Path, Exists: st.Exists, Entries: st.Entries, Bytes: st.TotalBytes(),
		Partial: st.Tmp}
}

type CacheCleared struct {
	Removed int      `json:"removed"`
	Bytes   int64    `json:"bytes"`
	Errors  []string `json:"errors"`
}

func ClearCache(dir string) CacheCleared {
	if dir == "" {
		dir = media.DefaultCacheDir()
	}
	r := media.Clear(dir, false)
	return CacheCleared{Removed: r.Removed, Bytes: r.RemovedBytes, Errors: append([]string{}, r.Errors...)}
}
