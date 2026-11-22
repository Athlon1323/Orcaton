package cli

import (
	"fmt"

	"orcaton/internal/media"
	"orcaton/internal/pyio"
)

func cacheGroup() *command {
	cacheOpt := opt{names: []string{"--cache"}, kind: kStr,
		help: "Cache folder (default — the system one, or $ORCATON_CACHE_DIR)"}
	dry := opt{names: []string{"--dry-run"}, kind: kBool, def: false, help: "Only show what would be deleted"}
	g := &command{name: "cache", help: "Audio feature cache: where it is, how much space it takes, cleanup.\n\n" +
		"  From any folder: orcaton cache (= cache info) · orcaton cache clear · " +
		"orcaton cache prune --older-than 30d",
		group: map[string]*command{}, order: []string{"info", "clear", "prune"}, deflt: "info"}
	g.group["info"] = &command{name: "info", help: "Where the cache is, how much space it takes, how many entries and leftovers.",
		opts: []opt{cacheOpt, jsonOpt}, run: cmdCacheInfo}
	g.group["clear"] = &command{name: "clear", help: "Clear the whole cache (features are recomputed on the next synchronization).",
		opts: []opt{cacheOpt, dry, jsonOpt}, run: cmdCacheClear}
	g.group["prune"] = &command{name: "prune", help: "Delete entries not used for a long time and *.tmp leftovers of interrupted runs.",
		opts: []opt{{names: []string{"--older-than"}, kind: kStr, def: "30d",
			help: "Delete entries unused for longer than: 30d, 12h, 2w"}, cacheOpt, dry, jsonOpt},
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
		created = "  (not created yet)"
	}
	c.printf("Cache: %s%s\n", st.Path, created)
	other := ""
	if st.Other > 0 {
		other = fmt.Sprintf(" | foreign files: %d (left alone)", st.Other)
	}
	c.printf("Entries: %d (%s), *.tmp leftovers: %d (%s), total %s | used: %s … %s%s\n",
		st.Entries, media.HumanBytes(st.EntriesBytes), st.Tmp, media.HumanBytes(st.TmpBytes),
		media.HumanBytes(st.TotalBytes()), day(st.Oldest), day(st.Newest), other)
	return 0
}

func printClean(c *ctx, res *media.CleanResult, what string) {
	c.printf("Cache: %s\n", cacheDir(c))
	verb := "Deleted"
	if res.DryRun {
		verb = "Would delete"
	}
	kept := ""
	if res.Kept > 0 {
		kept = fmt.Sprintf(", entries left: %d", res.Kept)
	}
	c.printf("%s %s: %d (%s)%s\n", verb, what, res.Removed, media.HumanBytes(res.RemovedBytes), kept)
	for _, e := range res.Errors[:min(10, len(res.Errors))] {
		c.printf("  failed: %s\n", e)
	}
}

func cmdCacheClear(c *ctx) int {
	res := media.Clear(cacheDir(c), c.Bool("dry-run"))
	if c.Bool("json") {
		c.emitJSON(res.AsDict())
	} else {
		printClean(c, res, "cache files")
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
		printClean(c, res, "older than "+ot+" and leftovers")
	}
	if len(res.Errors) > 0 {
		return 1
	}
	return 0
}
