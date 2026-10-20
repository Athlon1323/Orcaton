package api

import (
	"orcaton/internal/engine"
)

type Options struct {
	Unsynced      string  `json:"unsynced"`
	OrderBy       string  `json:"orderBy"`
	Singles       string  `json:"singles"`
	UnsyncedLabel string  `json:"unsyncedLabel"`
	TimeLabel     string  `json:"timeLabel"`
	MinConfidence float64 `json:"minConfidence"`
	MinRatio      float64 `json:"minRatio"`
	AutoThreshold bool    `json:"autoThreshold"`
	MinOverlapSec float64 `json:"minOverlapSec"`
	MaxOffsetSec  float64 `json:"maxOffsetSec"`
	AudioMode     string  `json:"audioMode"`
	FileOrder     bool    `json:"fileOrder"`
	IntraLane     bool    `json:"intraLane"`
	RecordTime    bool    `json:"recordTime"`
	MaxGapSec     float64 `json:"maxGapSec"`
	KeepGapSec    float64 `json:"keepGapSec"`
	Jobs          int     `json:"jobs"`
	Threads       int     `json:"threads"`
	MemLimitMB    int     `json:"memLimitMB"`
	CacheDir      string  `json:"cacheDir"`
}

var (
	UnsyncedModes = []string{string(engine.PlaceOrderTime), string(engine.PlaceEdges),
		string(engine.PlaceEnd), string(engine.PlaceRemove)}
	AudioModes   = append([]string(nil), engine.AudioModes...)
	OrderModes   = append([]string(nil), engine.OrderModes...)
	SinglesModes = append([]string(nil), engine.SinglesModes...)
)

func DefaultOptions() Options {
	d := engine.DefaultOptions()
	return Options{Unsynced: string(d.Unsynced), OrderBy: d.OrderBy, Singles: d.Singles, UnsyncedLabel: d.UnsyncedLabel,
		TimeLabel: d.TimeLabel, MinConfidence: d.MinConfidence, MinRatio: d.MinRatio,
		AutoThreshold: d.AutoThreshold, MinOverlapSec: d.MinOverlapSec, AudioMode: d.AudioMode,
		FileOrder: d.FileOrder, IntraLane: d.IntraLane, RecordTime: d.RecordTime,
		MaxGapSec: d.IdleGapSec, KeepGapSec: d.IdleKeepSec, Jobs: d.Jobs, Threads: d.Threads, MemLimitMB: d.MemLimitMB,
		CacheDir: d.CacheDir}
}

func (o Options) Validate() error {
	_, err := o.engine()
	return err
}

func (o Options) engine() (engine.Options, error) {
	e := engine.DefaultOptions()
	pl, ok := engine.ParsePlacement(o.Unsynced)
	if !ok {
		return e, &engine.OptionError{Name: "Unsynced",
			Msg: "допустимо edges | order_time | end | remove, получено " + quote(o.Unsynced)}
	}
	if o.MaxOffsetSec < 0 {
		return e, &engine.OptionError{Name: "MaxOffsetSec", Msg: "нужно ≥ 0 (0 — без ограничения)"}
	}
	e.Unsynced, e.UnsyncedLabel, e.TimeLabel = pl, o.UnsyncedLabel, o.TimeLabel
	e.OrderBy, e.Singles = o.OrderBy, o.Singles
	e.MinConfidence, e.MinRatio, e.AutoThreshold = o.MinConfidence, o.MinRatio, o.AutoThreshold
	e.MinOverlapSec, e.AudioMode = o.MinOverlapSec, o.AudioMode
	e.MaxOffsetSec = -1
	if o.MaxOffsetSec > 0 {
		e.MaxOffsetSec = o.MaxOffsetSec
	}
	e.FileOrder, e.IntraLane, e.RecordTime = o.FileOrder, o.IntraLane, o.RecordTime
	e.IdleGapSec, e.IdleKeepSec, e.Jobs, e.CacheDir = o.MaxGapSec, o.KeepGapSec, o.Jobs, o.CacheDir
	e.Threads, e.MemLimitMB = o.Threads, o.MemLimitMB
	return e, e.Validate()
}

func quote(s string) string { return "«" + s + "»" }
