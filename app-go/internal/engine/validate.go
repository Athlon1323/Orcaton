package engine

import (
	"fmt"
	"math"
)

type OptionError struct {
	Name string
	Msg  string
}

func (e *OptionError) Error() string { return fmt.Sprintf("опция %s: %s", e.Name, e.Msg) }

var AudioModes = []string{"mix", "first_stream", "first_channel"}

func (o Options) Validate() error {
	if o.SampleRate < 1000 || o.SampleRate > 192000 {
		return &OptionError{"SampleRate", fmt.Sprintf("частота анализа %d Гц вне 1000…192000", o.SampleRate)}
	}
	if o.Hop < 1 || o.Hop > o.SampleRate {
		return &OptionError{"Hop", fmt.Sprintf("шаг %d вне 1…%d отсчётов", o.Hop, o.SampleRate)}
	}
	if o.CoarseFactor < 1 {
		return &OptionError{"CoarseFactor", fmt.Sprintf("%d < 1", o.CoarseFactor)}
	}
	if o.Jobs < 0 || o.Jobs > 256 {
		return &OptionError{"Jobs", fmt.Sprintf("параллельных ffmpeg %d вне 0…256 (0 — авто)", o.Jobs)}
	}
	if o.Threads < 0 || o.Threads > 4096 {
		return &OptionError{"Threads", fmt.Sprintf("потоков %d вне 0…4096 (0 — все ядра)", o.Threads)}
	}
	if o.MemLimitMB != 0 && (o.MemLimitMB < 256 || o.MemLimitMB > 1<<20) {
		return &OptionError{"MemLimitMB", fmt.Sprintf("потолок памяти %d МБ вне 256…1048576 (0 — авто)", o.MemLimitMB)}
	}
	if o.OrderBy != OrderByName && o.OrderBy != OrderByTime {
		return &OptionError{"OrderBy", fmt.Sprintf("неизвестный порядок %q (допустимо: name | time)", o.OrderBy)}
	}
	if o.Singles != SinglesSequential && o.Singles != SinglesStacked {
		return &OptionError{"Singles", fmt.Sprintf("неизвестная раскладка %q (допустимо: sequential | stacked)", o.Singles)}
	}
	if !knownMode(o.AudioMode) {
		return &OptionError{"AudioMode", fmt.Sprintf("неизвестный режим %q (допустимо: mix | first_stream | first_channel)", o.AudioMode)}
	}
	nonNeg := []struct {
		name string
		v    float64
	}{
		{"MinConfidence", o.MinConfidence}, {"NoiseK", o.NoiseK}, {"MinRatio", o.MinRatio},
		{"Dominance", o.Dominance}, {"AmbiguityFloor", o.AmbiguityFloor},
		{"CompetitorRatio", o.CompetitorRatio}, {"MinOverlapSec", o.MinOverlapSec},
		{"RefineRadiusSec", o.RefineRadiusSec}, {"GroupGapSec", o.GroupGapSec},
		{"WindowMinConf", o.WindowMinConf}, {"WindowRatio", o.WindowRatio},
		{"TimeSearchSec", o.TimeSearchSec}, {"TimeMinConf", o.TimeMinConf}, {"TimeRatio", o.TimeRatio},
		{"TimeMinActivity", o.TimeMinActivity}, {"ClockToleranceSec", o.ClockToleranceSec},
		{"MaxTimeGapSec", o.MaxTimeGapSec}, {"IdleGapSec", o.IdleGapSec}, {"IdleKeepSec", o.IdleKeepSec},
		{"SpannedTolSec", o.SpannedTolSec},
	}
	for _, f := range nonNeg {
		if math.IsNaN(f.v) || math.IsInf(f.v, 0) || f.v < 0 {
			return &OptionError{f.name, fmt.Sprintf("нужно конечное число ≥ 0, получено %v", f.v)}
		}
	}
	if math.IsNaN(o.MaxOffsetSec) || math.IsInf(o.MaxOffsetSec, 0) {
		return &OptionError{"MaxOffsetSec", fmt.Sprintf("нужно конечное число, получено %v", o.MaxOffsetSec)}
	}
	return nil
}

func knownMode(m string) bool {
	for _, x := range AudioModes {
		if m == x {
			return true
		}
	}
	return false
}
