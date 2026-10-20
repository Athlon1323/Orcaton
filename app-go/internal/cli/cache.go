package cli

import (
	"fmt"

	"orcaton/internal/media"
	"orcaton/internal/pyio"
)

func cacheGroup() *command {
	cacheOpt := opt{names: []string{"--cache"}, kind: kStr,
		help: "Папка кэша (по умолчанию — системная, или $ORCATON_CACHE_DIR)"}
	dry := opt{names: []string{"--dry-run"}, kind: kBool, def: false, help: "Только показать, что будет удалено"}
	g := &command{name: "cache", help: "Кэш признаков звука: где лежит, сколько занимает, очистка.\n\n" +
		"  Из любой папки: orcaton cache (= cache info) · orcaton cache clear · " +
		"orcaton cache prune --older-than 30d",
		group: map[string]*command{}, order: []string{"info", "clear", "prune"}, deflt: "info"}
	g.group["info"] = &command{name: "info", help: "Где кэш, сколько места занимает, сколько записей и обрывков.",
		opts: []opt{cacheOpt, jsonOpt}, run: cmdCacheInfo}
	g.group["clear"] = &command{name: "clear", help: "Очистить кэш целиком (признаки пересчитаются при следующей синхронизации).",
		opts: []opt{cacheOpt, dry, jsonOpt}, run: cmdCacheClear}
	g.group["prune"] = &command{name: "prune", help: "Удалить давно не использованные записи и обрывки *.tmp прерванных прогонов.",
		opts: []opt{{names: []string{"--older-than"}, kind: kStr, def: "30d",
			help: "Удалить записи, не использованные дольше: 30d, 12h, 2w"}, cacheOpt, dry, jsonOpt},
		run: cmdCachePrune}
	return g
}

func cacheDir(c *ctx) string {
	if d := c.Str("cache"); d != "" {
		return d
	}
	return media.DefaultCacheDir()
}

func cmdCacheInfo(c *ctx) int {
	st := media.Stats(cacheDir(c))
	if c.Bool("json") {
		c.emitJSON(st.AsDict())
		return 0
	}
	day := func(t *float64) string {
		if t == nil || *t == 0 {
			return "—"
		}
		return pyio.LocalDay(*t)
	}
	created := ""
	if !st.Exists {
		created = "  (ещё не создан)"
	}
	c.printf("Кэш: %s%s\n", st.Path, created)
	other := ""
	if st.Other > 0 {
		other = fmt.Sprintf(" | чужих файлов: %d (не трогаем)", st.Other)
	}
	c.printf("Записей: %d (%s), обрывков *.tmp: %d (%s), всего %s | использовались: %s … %s%s\n",
		st.Entries, media.HumanBytes(st.EntriesBytes), st.Tmp, media.HumanBytes(st.TmpBytes),
		media.HumanBytes(st.TotalBytes()), day(st.Oldest), day(st.Newest), other)
	return 0
}

func printClean(c *ctx, res *media.CleanResult, what string) {
	c.printf("Кэш: %s\n", cacheDir(c))
	verb := "Удалено"
	if res.DryRun {
		verb = "Будет удалено"
	}
	kept := ""
	if res.Kept > 0 {
		kept = fmt.Sprintf(", осталось записей: %d", res.Kept)
	}
	c.printf("%s %s: %d (%s)%s\n", verb, what, res.Removed, media.HumanBytes(res.RemovedBytes), kept)
	for _, e := range res.Errors[:min(10, len(res.Errors))] {
		c.printf("  не удалось: %s\n", e)
	}
}

func cmdCacheClear(c *ctx) int {
	res := media.Clear(cacheDir(c), c.Bool("dry-run"))
	if c.Bool("json") {
		c.emitJSON(res.AsDict())
	} else {
		printClean(c, res, "файлов кэша")
	}
	if len(res.Errors) > 0 {
		return 1
	}
	return 0
}

func cmdCachePrune(c *ctx) int {
	ot := c.Str("older-than")
	age, err := media.ParseAge(ot)
	if err != nil {
		c.printf("--older-than: %v\n", err)
		return 2
	}
	res := media.Prune(cacheDir(c), age, c.Bool("dry-run"), -1)
	if c.Bool("json") {
		c.emitJSON(res.AsDict())
	} else {
		printClean(c, res, "старше "+ot+" и обрывков")
	}
	if len(res.Errors) > 0 {
		return 1
	}
	return 0
}
