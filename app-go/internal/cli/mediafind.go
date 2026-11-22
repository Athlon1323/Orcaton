package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"orcaton/internal/app"
	"orcaton/internal/core"
	"orcaton/internal/formats"
	"orcaton/internal/media"
)

var (
	promptIn  io.Reader = os.Stdin
	isTermFn            = func() bool { return isTerminal(os.Stdin) && isTerminal(os.Stderr) }
	promptMax           = 15
)

func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

var (
	searchOpt = opt{names: []string{"--search"}, kind: kList,
		help: "Folder to search for missing media by file name (like --map FOLDER); can be repeated"}
	depthOpt = opt{names: []string{"--search-depth"}, kind: kInt, def: media.DefaultSearchDepth, min: bound(0), max: bound(64),
		help: "Search depth in subfolders"}
	yesOpt = opt{names: []string{"--yes", "-y"}, kind: kBool, def: false,
		help: "Do not ask: files not found go straight to the end of the timeline (offline)"}
)

func trimPath(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "& ")
	for len(s) >= 2 && (s[0] == '"' && s[len(s)-1] == '"' || s[0] == '\'' && s[len(s)-1] == '\'') {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	if strings.Count(s, `"`) == 1 && strings.HasSuffix(s, `"`) {
		s = strings.TrimSpace(strings.TrimSuffix(s, `"`))
	}
	return s
}

func searchRoots(c *ctx) []string {
	spec := os.Getenv("ORCATON_PATH_MAP")
	if p := c.StrPtr("map"); p != nil {
		spec = *p
	}
	var roots []string
	for _, part := range strings.Split(spec, ";") {
		if strings.Contains(part, "=") {
			continue
		}
		if s := trimPath(part); s != "" {
			roots = append(roots, s)
		}
	}
	for _, s := range c.List("search") {
		if s = trimPath(s); s != "" {
			roots = append(roots, s)
		}
	}
	return roots
}

func (c *ctx) eprintf(format string, a ...any) { fmt.Fprintf(c.err, format, a...) }

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func (c *ctx) loadWithMedia(path string) (*core.Project, int) {
	p, code := c.load(path, "", c.StrPtr("map"))
	if code != 0 {
		return nil, code
	}
	if findMissingMedia(c, p) {
		return nil, 1
	}
	return p, 0
}

func origOf(m *core.MediaRef) string {
	if m.OriginalPath != nil {
		return *m.OriginalPath
	}
	return *m.Path
}

func searchDepth(c *ctx) int {
	if !c.Has("search-depth") {
		return media.DefaultSearchDepth
	}
	return c.Int("search-depth")
}

func printSearch(c *ctx, r app.SearchReport) {
	for _, b := range r.BadRoots {
		if r.Answer {
			c.eprintf("Folder not found: %s\n", b)
		} else {
			c.eprintf("Search folder not found: %s\n", b)
		}
	}
	if len(r.Roots) == 0 || r.Before == 0 {
		return
	}
	c.eprintf("Media search (%s): found %d of %d missing\n",
		strings.Join(r.Roots, "; "), r.Found, r.Before)
	for _, a := range r.Ambiguous {
		c.eprintf("  %s — several identical files, none chosen: %s\n",
			formats.BaseName(*a.Media.Path), strings.Join(a.Candidates[:min(3, len(a.Candidates))], " | "))
	}
}

func listMissing(c *ctx, missing []*core.MediaRef) {
	for _, m := range missing[:min(promptMax, len(missing))] {
		c.eprintf("  %s   (%s)\n", formats.BaseName(*m.Path), origOf(m))
	}
	if len(missing) > promptMax {
		c.eprintf("  … and %d more\n", len(missing)-promptMax)
	}
}

type termAsker struct {
	c     *ctx
	in    *bufio.Reader
	shown int
	asked bool
}

func (a *termAsker) AskMissing(_ context.Context, missing []*core.MediaRef) (app.Answer, error) {
	c := a.c
	a.asked = true
	if len(missing) != a.shown {
		c.eprintf("\nNot found on disk: %d file(s):\n", len(missing))
		listMissing(c, missing)
		c.eprintf("Enter a folder where they may be (search by name in subfolders),\n" +
			"Enter — skip (they go to the end of the timeline as missing, offline in the editing program), " +
			"q — cancel:\n")
		a.shown = len(missing)
	}
	c.eprintf("> ")
	line, err := a.in.ReadString('\n')
	ans := trimPath(line)
	if ans == "" {
		if err != nil {
			c.eprintf("\n")
		}
		c.eprintf("Skipped: %d file(s) — to the end of the timeline (offline)\n", len(missing))
		return app.Answer{Action: app.AnswerSkip}, nil
	}
	switch strings.ToLower(ans) {
	case "q", "й", "quit", "exit":
		c.eprintf("Cancelled\n")
		return app.Answer{Action: app.AnswerCancel}, nil
	}
	return app.Answer{Action: app.AnswerFolder, Folder: ans}, nil
}

func findMissingMedia(c *ctx, p *core.Project) (cancelled bool) {
	if len(c.List("features")) > 0 {
		return false
	}
	quiet := c.Bool("brief") || c.Bool("json")
	interactive := !quiet && !c.Bool("yes") && isTermFn()
	req := app.FindRequest{Roots: searchRoots(c), Depth: searchDepth(c),
		OnSearch: func(r app.SearchReport) { printSearch(c, r) }}
	var asker *termAsker
	if interactive {
		asker = &termAsker{c: c, in: bufio.NewReader(promptIn), shown: -1}
		req.Ask = asker
	}
	missing, err := app.FindMedia(context.Background(), p, req)
	if err != nil {
		return true
	}
	switch {
	case len(missing) > 0 && !quiet && !interactive:
		c.eprintf("Not found on disk: %d file(s) — they go to the end of the timeline as missing "+
			"(offline in the editing program). Folder with sources: --map \"FOLDER\"\n", len(missing))
		listMissing(c, missing)
	case len(missing) == 0 && asker != nil && asker.asked:
		c.eprintf("All files found\n")
	}
	return false
}
