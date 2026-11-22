package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"strings"
	"sync"

	"orcaton/internal/app"
	"orcaton/internal/core"
	"orcaton/internal/engine"
	"orcaton/internal/formats"
	"orcaton/internal/num"
	"orcaton/internal/pyio"
)

type devOpt struct {
	after string
	o     opt
}

var devSyncOpts []devOpt

func syncCommand() *command {
	c := syncBase()
	for _, d := range devSyncOpts {
		for i, o := range c.opts {
			if o.names[0] == d.after {
				c.opts = append(c.opts[:i+1], append([]opt{d.o}, c.opts[i+1:]...)...)
				break
			}
		}
	}
	return c
}

func syncBase() *command {
	return &command{name: "sync",
		help: "Synchronize a timeline by audio: IN (xml/drt) → OUT (xml/drt).\n\n" +
			"  OUT .drt — only from IN .drt: the source timeline is patched (clip positions change),\n" +
			"  everything else stays as it was. Labels are not written to DRT yet — statuses go to the CSV report\n" +
			"  (for DRT it is always created: OUT_report.csv).",
		args: []arg{{name: "src", file: true}, {name: "dst"}},
		opts: []opt{mapOpt, searchOpt, depthOpt, yesOpt,
			{names: []string{"--keep-paths"}, neg: "--no-keep-paths", kind: kBool, def: false,
				help: "--map/--search are only for reading media; OUT keeps the original paths"},
			{names: []string{"--unsynced"}, kind: kStr, def: "order_time",
				help: "Unpaired: order_time (in camera order, also inside segments) | edges (never inside a segment) | end | remove. Offline — always at the very end"},
			{names: []string{"--order-by"}, kind: kStr, def: "name",
				help: "Camera file order: name (number in the name, then shooting time) | time (shooting time / TC)"},
			{names: []string{"--singles"}, kind: kStr, def: "sequential",
				help: "Unpaired files: sequential (do not run at the same time as other unpaired files) | stacked (each on its own camera)"},
			{names: []string{"--label"}, kind: kStr, def: "Rose", help: "Color label for unsynced clips"},
			{names: []string{"--min-conf"}, kind: kFloat, def: 12.0, min: bound(0),
				help: "Min. match significance (Fisher z); the threshold rises to the project noise"},
			{names: []string{"--min-ratio"}, kind: kFloat, def: 1.5, min: bound(0), help: "Uniqueness: peak z / second z"},
			{names: []string{"--auto-threshold"}, neg: "--no-auto-threshold", kind: kBool, def: true,
				help: "Threshold from project noise (median + 6·MAD over all pairs)"},
			{names: []string{"--min-overlap"}, kind: kFloat, def: 2.0, min: bound(0), help: "Min. audio overlap, s"},
			{names: []string{"--max-offset"}, kind: kFloat, min: bound(0.001), help: "Search window ±s"},
			{names: []string{"--audio-mode"}, kind: kStr, def: "mix",
				help: "Audio for analysis: mix | first_stream | first_channel"},
			{names: []string{"--file-order"}, neg: "--no-file-order", kind: kBool, def: true,
				help: "Files of one camera (track+folder) follow TC/number order"},
			{names: []string{"--intra-lane"}, neg: "--no-intra-lane", kind: kBool, def: false,
				help: "Compare clips on the same track (if devices are mixed on it)"},
			{names: []string{"--jobs", "-j"}, kind: kInt, def: 0, min: bound(0), max: bound(256),
				help: "Parallel ffmpeg processes; 0 — auto: half the cores (2…16), up to 4 on HDD"},
			{names: []string{"--threads"}, kind: kInt, def: 0, min: bound(0), max: bound(4096),
				help: "Correlation threads; 0 — all cores"},
			{names: []string{"--mem-limit"}, kind: kFloat, def: 0.0, min: bound(0),
				help: "Memory cap, GB; 0 — auto: ¾ of RAM, 1 to 7 GB"},
			{names: []string{"--cache"}, kind: kStr, help: "Feature cache folder"},
			{names: []string{"--report"}, kind: kStr, help: "Per-clip CSV report"},
			{names: []string{"--to"}, kind: kStr, help: "Output format"},
			{names: []string{"--originals"}, kind: kList,
				help: "Folder with camera originals/sidecars (recording time for proxies); can be repeated"},
			{names: []string{"--record-time"}, neg: "--no-record-time", kind: kBool, def: true,
				help: "Use recording time: order, gaps, placement by camera clock"},
			{names: []string{"--time-label"}, kind: kStr, def: "Mango",
				help: "Color label for files placed by camera clock only"},
			{names: []string{"--max-gap"}, kind: kFloat, def: 0.0, min: bound(0),
				help: "Gaps longer than this (s) where no file plays are removed; 0 — all"},
			{names: []string{"--keep-gap"}, kind: kFloat, def: 0.0, min: bound(0),
				help: "Length to shrink such gaps to, s (0 — butt-joined)"},
			progressOpt, configOpt, noConfigOpt, briefOpt, jsonOpt},
		run: cmdSync}
}

func cmdSync(c *ctx) int {
	files, code := applySettings(c, c.Arg("src"))
	if code != 0 {
		return code
	}
	o, code := syncOptions(c)
	if code != 0 {
		return code
	}
	quiet := c.Bool("brief") || c.Bool("json")
	prog, code := syncProgress(c, quiet)
	if code != 0 {
		return code
	}
	src, dst := c.Arg("src"), c.Arg("dst")
	if len(files) > 0 && !quiet {
		c.printf("Settings: %s\n", strings.Join(files, "; "))
	}
	p, code := c.loadWithMedia(src)
	if code != 0 {
		return code
	}
	outFmt, report, err := app.CheckOutput(p, dst, c.Str("to"), c.Str("report"))
	if err != nil {
		c.printf("Error: %v\n", err)
		return 2
	}
	sctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	run, err := app.Sync(sctx, p, app.SyncRequest{Options: o, Features: c.List("features"),
		Originals: c.List("originals"), Times: c.List("times")}, prog)
	if err != nil {
		return ffmpegFail(c, err)
	}
	if err := app.Save(run, dst, c.Str("to"), c.Bool("keep-paths"), prog); err != nil {
		c.printf("Write error: %v\n", err)
		return 2
	}
	if report != "" {
		if err := app.WriteReport(report, run.Report); err != nil {
			c.printf("Error: %v\n", err)
			return 1
		}
	}
	if quiet {
		printSyncBrief(c, run.Report, dst, report)
		return 0
	}
	printSyncText(c, run.Report, outFmt == "drt", dst, report)
	return 0
}

func syncOptions(c *ctx) (engine.Options, int) {
	o := engine.DefaultOptions()
	placement, ok := engine.ParsePlacement(c.Str("unsynced"))
	if !ok {
		c.println("--unsynced: allowed values are edges | order_time | end | remove")
		return o, 2
	}
	o.AudioMode = c.Str("audio-mode")
	o.MinConfidence = c.Float("min-conf")
	o.MinRatio = c.Float("min-ratio")
	o.MinOverlapSec = c.Float("min-overlap")
	if c.Has("max-offset") {
		o.MaxOffsetSec = c.Float("max-offset")
	}
	o.AutoThreshold = c.Bool("auto-threshold")
	o.FileOrder = c.Bool("file-order")
	o.IntraLane = c.Bool("intra-lane")
	o.Unsynced = placement
	o.OrderBy = c.Str("order-by")
	o.Singles = c.Str("singles")
	o.UnsyncedLabel = c.Str("label")
	o.Jobs = c.Int("jobs")
	o.Threads = c.Int("threads")
	o.MemLimitMB = int(math.Round(c.Float("mem-limit") * 1024))
	o.CacheDir = c.Str("cache")
	o.RecordTime = c.Bool("record-time")
	o.TimeLabel = c.Str("time-label")
	o.IdleGapSec = c.Float("max-gap")
	o.IdleKeepSec = c.Float("keep-gap")
	if err := o.Validate(); err != nil {
		c.printf("Error: %v\n", err)
		return o, 2
	}
	return o, 0
}

var progressOpt = opt{names: []string{"--progress"}, kind: kStr, def: "auto",
	help: "Progress to stderr: auto (one line if stderr is a terminal and there is no --brief/--json) | text | " +
		"json (one JSON event per line: stage, title, done, total, file) | none"}

func syncProgress(c *ctx, quiet bool) (engine.Progress, int) {
	mode := c.Str("progress")
	switch mode {
	case "auto":
		if quiet || !isTermWriter(c.err) {
			return nil, 0
		}
		return textProgress(c), 0
	case "text":
		return textProgress(c), 0
	case "json":
		return jsonProgress(c), 0
	case "none":
		return nil, 0
	}
	c.println("--progress: allowed values are auto | text | json | none")
	return nil, 2
}

func textProgress(c *ctx) engine.Progress {
	var mu sync.Mutex
	return func(ev engine.Event) {
		mu.Lock()
		defer mu.Unlock()
		if ev.Total > 0 && ev.Done < ev.Total {
			fmt.Fprintf(c.err, "\r%s %d/%d   ", ev.Title, ev.Done, ev.Total)
			return
		}
		fmt.Fprint(c.err, "\r"+strings.Repeat(" ", 60)+"\r")
	}
}

func jsonProgress(c *ctx) engine.Progress {
	var mu sync.Mutex
	return func(ev engine.Event) {
		b, _ := json.Marshal(ev)
		mu.Lock()
		fmt.Fprintf(c.err, "%s\n", b)
		mu.Unlock()
	}
}

func isTermWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && isTerminal(f)
}

func printSyncBrief(c *ctx, r *engine.Report, dst, report string) {
	sm := r.Summary()
	if c.Bool("json") {
		m := pyio.NewOMap().Set("out", pathStr(dst))
		if report != "" {
			m.Set("report", pathStr(report))
		} else {
			m.Set("report", nil)
		}
		for _, k := range sm.Keys() {
			v, _ := sm.Get(k)
			m.Set(k, v)
		}
		c.emitJSON(m)
		return
	}
	loose := int(sf(sm, "unsynced")) - int(sf(sm, "offline"))
	c.printf("sync → %s: audio %s/%s, clock %s, unpaired %d, offline %s | pairs %s, edges %s, "+
		"threshold %s | %s s\n", formats.BaseName(dst), sv(sm, "synced"), sv(sm, "units"),
		sv(sm, "by_time"), loose, sv(sm, "offline"), sv(sm, "tried_pairs"), sv(sm, "edges"),
		sv(sm, "threshold"), sv(sm, "seconds"))
}

func printSyncText(c *ctx, r *engine.Report, drtOut bool, dst, report string) {
	c.printf("Units: %d  synced: %d  not: %d  pairs checked: %d  matches: %d  "+
		"z threshold: %s (noise %s)  ambiguous: %d  rejected: overlap %d, file order %d  "+
		"in camera window: %d  recording time: %d files, clock+audio: %d (by shift %d), clock only: %d, "+
		"groups joined: %d  inconsistent: %d  conflicts: %d\n",
		len(r.Units), r.Count(core.StatusSynced), r.Count(core.StatusUnsynced), r.TriedPairs,
		len(r.Edges), num.Fixed(r.Threshold, 1), num.Fixed(r.NoiseLevel, 1), r.Ambiguous,
		r.RejectedOverlap, r.RejectedOrder, r.WindowSynced, r.WithRecTime, r.TimeAudio,
		r.TimeShiftTested, r.TimePlaced, r.ClockMerged, r.InconsistentEdges, r.Conflicts)
	var tparts []string
	for _, t := range r.Timings {
		tparts = append(tparts, t.Name+"="+num.Fixed(t.Sec, 1))
	}
	if r.Duplicates > 0 {
		apart := 0
		for _, u := range r.Units {
			if u.Reason == engine.ReasonDuplicate {
				apart++
			}
		}
		where := "placed together with the original"
		if apart > 0 {
			where += fmt.Sprintf(", except %d: in DRT on a track shared with the original — unpaired, in camera order", apart)
		}
		c.printf("File copies (same audio): %d — %s\n", r.Duplicates, where)
	}
	c.printf("Time, s: %s\n", strings.Join(tparts, ", "))
	c.printf("Resources: %s; peak memory %s\n", r.Resources, peakText(r.PeakMem))
	why := newCounter()
	for _, u := range r.Units {
		if u.Error != nil && *u.Error != "" {
			why.inc(*u.Error)
		}
	}
	if len(why.keys) > 0 {
		c.printf("Reasons not synced: %s\n", why.String())
	}
	if report != "" {
		c.printf("Report: %s\n", report)
	}
	if drtOut {
		c.println("DRT: color labels (unpaired / by clock) are not written to .drt yet — clip statuses are in the report")
	}
	c.printf("Written: %s\n", dst)
}

func peakText(b uint64) string {
	if b == 0 {
		return "?"
	}
	return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
}
