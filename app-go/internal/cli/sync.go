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
		help: "Синхронизировать таймлайн по звуку: IN (xml/drt) → OUT (xml/drt).\n\n" +
			"  OUT .drt — только из IN .drt: исходный таймлайн патчится (меняются позиции клипов),\n" +
			"  всё остальное остаётся как было. Метки в DRT пока не пишутся — статусы в CSV-отчёте\n" +
			"  (для DRT он создаётся всегда: OUT_report.csv).",
		args: []arg{{name: "src", file: true}, {name: "dst"}},
		opts: []opt{mapOpt, searchOpt, depthOpt, yesOpt,
			{names: []string{"--keep-paths"}, neg: "--no-keep-paths", kind: kBool, def: false,
				help: "--map/--search только для чтения медиа; в OUT остаются исходные пути"},
			{names: []string{"--unsynced"}, kind: kStr, def: "order_time",
				help: "Без пары: order_time (по порядку камеры, и внутри отрезков) | edges (внутрь отрезка — никогда) | end | remove. Нет на диске — всегда в самый конец"},
			{names: []string{"--order-by"}, kind: kStr, def: "name",
				help: "Порядок файлов камеры: name (номер в имени, потом время съёмки) | time (время съёмки / TC)"},
			{names: []string{"--singles"}, kind: kStr, def: "sequential",
				help: "Файлы без пары: sequential (не идут одновременно с другими файлами без пары по звуку) | stacked (каждый по своей камере)"},
			{names: []string{"--label"}, kind: kStr, def: "Rose", help: "Цветовая метка несинхронизированных"},
			{names: []string{"--min-conf"}, kind: kFloat, def: 12.0, min: bound(0),
				help: "Мин. значимость совпадения (z Фишера); порог поднимается до шума проекта"},
			{names: []string{"--min-ratio"}, kind: kFloat, def: 1.5, min: bound(0), help: "Однозначность: z пика / z второго"},
			{names: []string{"--auto-threshold"}, neg: "--no-auto-threshold", kind: kBool, def: true,
				help: "Порог по шуму проекта (медиана + 6·MAD по всем парам)"},
			{names: []string{"--min-overlap"}, kind: kFloat, def: 2.0, min: bound(0), help: "Мин. перекрытие звука, сек"},
			{names: []string{"--max-offset"}, kind: kFloat, min: bound(0.001), help: "Окно поиска ±сек"},
			{names: []string{"--audio-mode"}, kind: kStr, def: "mix",
				help: "Звук для анализа: mix | first_stream | first_channel"},
			{names: []string{"--file-order"}, neg: "--no-file-order", kind: kBool, def: true,
				help: "Файлы одной камеры (дорожка+папка) идут по порядку TC/номера"},
			{names: []string{"--intra-lane"}, neg: "--no-intra-lane", kind: kBool, def: false,
				help: "Сравнивать клипы одной дорожки (если на ней смешаны устройства)"},
			{names: []string{"--jobs", "-j"}, kind: kInt, def: 0, min: bound(0), max: bound(256),
				help: "Параллельных ffmpeg; 0 — авто: половина ядер (2…16), на HDD — до 4"},
			{names: []string{"--threads"}, kind: kInt, def: 0, min: bound(0), max: bound(4096),
				help: "Потоков корреляции; 0 — все ядра"},
			{names: []string{"--mem-limit"}, kind: kFloat, def: 0.0, min: bound(0),
				help: "Потолок памяти, ГБ; 0 — авто: ¾ ОЗУ, от 1 до 7 ГБ"},
			{names: []string{"--cache"}, kind: kStr, help: "Папка кэша признаков"},
			{names: []string{"--report"}, kind: kStr, help: "CSV-отчёт по клипам"},
			{names: []string{"--to"}, kind: kStr, help: "Формат вывода"},
			{names: []string{"--originals"}, kind: kList,
				help: "Папка с оригиналами/сайдкарами камер (время записи для прокси); можно несколько раз"},
			{names: []string{"--record-time"}, neg: "--no-record-time", kind: kBool, def: true,
				help: "Использовать время записи: порядок, паузы, постановка по часам камеры"},
			{names: []string{"--time-label"}, kind: kStr, def: "Mango",
				help: "Цветовая метка файлов, поставленных только по часам камеры"},
			{names: []string{"--max-gap"}, kind: kFloat, def: 0.0, min: bound(0),
				help: "Пустоты длиннее (сек), где не идёт ни один файл, убираются; 0 — все"},
			{names: []string{"--keep-gap"}, kind: kFloat, def: 0.0, min: bound(0),
				help: "До какой длины сжимать такие пустоты, сек (0 — встык)"},
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
		c.printf("Настройки: %s\n", strings.Join(files, "; "))
	}
	p, code := c.loadWithMedia(src)
	if code != 0 {
		return code
	}
	outFmt, report, err := app.CheckOutput(p, dst, c.Str("to"), c.Str("report"))
	if err != nil {
		c.printf("Ошибка: %v\n", err)
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
		c.printf("Ошибка записи: %v\n", err)
		return 2
	}
	if report != "" {
		if err := app.WriteReport(report, run.Report); err != nil {
			c.printf("Ошибка: %v\n", err)
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
		c.println("--unsynced: допустимо edges | order_time | end | remove")
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
		c.printf("Ошибка: %v\n", err)
		return o, 2
	}
	return o, 0
}

var progressOpt = opt{names: []string{"--progress"}, kind: kStr, def: "auto",
	help: "Прогресс в stderr: auto (строкой, если stderr — терминал и нет --brief/--json) | text | " +
		"json (событие JSON на строку: stage, title, done, total, file) | none"}

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
	c.println("--progress: допустимо auto | text | json | none")
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
	c.printf("sync → %s: звук %s/%s, часы %s, без пары %d, нет на диске %s | пар %s, рёбер %s, "+
		"порог %s | %s с\n", formats.BaseName(dst), sv(sm, "synced"), sv(sm, "units"),
		sv(sm, "by_time"), loose, sv(sm, "offline"), sv(sm, "tried_pairs"), sv(sm, "edges"),
		sv(sm, "threshold"), sv(sm, "seconds"))
}

func printSyncText(c *ctx, r *engine.Report, drtOut bool, dst, report string) {
	c.printf("Единиц: %d  синхронизировано: %d  нет: %d  пар проверено: %d  совпадений: %d  "+
		"порог z: %s (шум %s)  неоднозначных: %d  отброшено: наложение %d, порядок файлов %d  "+
		"в окне камеры: %d  время записи: %d файлов, по часам+звук: %d (сдвигом %d), только по часам: %d, "+
		"сшито групп: %d  несогласованных: %d  конфликтов: %d\n",
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
		where := "стоят вместе с оригиналом"
		if apart > 0 {
			where += fmt.Sprintf(", кроме %d: в DRT на общей с оригиналом дорожке — без пары, по порядку камеры", apart)
		}
		c.printf("Копий файлов (тот же звук): %d — %s\n", r.Duplicates, where)
	}
	c.printf("Время, c: %s\n", strings.Join(tparts, ", "))
	c.printf("Ресурсы: %s; пик памяти %s\n", r.Resources, peakText(r.PeakMem))
	why := newCounter()
	for _, u := range r.Units {
		if u.Error != nil && *u.Error != "" {
			why.inc(*u.Error)
		}
	}
	if len(why.keys) > 0 {
		c.printf("Причины несинхронизации: %s\n", why.String())
	}
	if report != "" {
		c.printf("Отчёт: %s\n", report)
	}
	if drtOut {
		c.println("DRT: цветовые метки (без пары / по часам) в .drt пока не пишутся — статусы клипов в отчёте")
	}
	c.printf("Записано: %s\n", dst)
}

func peakText(b uint64) string {
	if b == 0 {
		return "?"
	}
	return fmt.Sprintf("%.1f ГБ", float64(b)/(1<<30))
}
