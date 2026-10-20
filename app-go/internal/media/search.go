package media

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"orcaton/internal/core"
	"orcaton/internal/formats"
)

const DefaultSearchDepth = 6

type SearchResult struct {
	Missing   []*core.MediaRef
	Found     map[*core.MediaRef]string
	Ambiguous map[*core.MediaRef][]string
	NotFound  []*core.MediaRef
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular()
}

func MissingMedia(p *core.Project) []*core.MediaRef {
	var out []*core.MediaRef
	for _, m := range p.MediaList() {
		if m.Offline || m.Path == nil || *m.Path == "" {
			continue
		}
		if !fileExists(*m.Path) {
			out = append(out, m)
		}
	}
	return out
}

func nameKey(name string) string { return formats.Casefold(formats.NFC(name)) }

func pathParts(p string) []string {
	var out []string
	for _, x := range strings.Split(formats.SlashPath(p), "/") {
		if x == "" || x == "." || (len(x) == 2 && x[1] == ':') {
			continue
		}
		out = append(out, nameKey(x))
	}
	return out
}

func tailScore(a, b string) int {
	pa, pb := pathParts(a), pathParts(b)
	n := 0
	for i, j := len(pa)-2, len(pb)-2; i >= 0 && j >= 0; i, j = i-1, j-1 {
		if pa[i] != pb[j] {
			break
		}
		n++
	}
	return n
}

type Index map[string][]string

func BuildIndex(roots []string, depth int) Index {
	idx := Index{}
	for _, root := range roots {
		root = filepath.Clean(root)
		base := strings.Count(filepath.ToSlash(root), "/")
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				if p != root && strings.Count(filepath.ToSlash(p), "/")-base > depth {
					return fs.SkipDir
				}
				return nil
			}
			k := nameKey(d.Name())
			idx[k] = append(idx[k], p)
			return nil
		})
	}
	for k := range idx {
		sort.Strings(idx[k])
	}
	return idx
}

func (idx Index) Resolve(missing []*core.MediaRef) *SearchResult {
	res := &SearchResult{Missing: missing, Found: map[*core.MediaRef]string{},
		Ambiguous: map[*core.MediaRef][]string{}}
	for _, m := range missing {
		orig := *m.Path
		if m.OriginalPath != nil {
			orig = *m.OriginalPath
		}
		cands := idx[nameKey(formats.BaseName(*m.Path))]
		switch len(cands) {
		case 0:
			res.NotFound = append(res.NotFound, m)
		case 1:
			res.Found[m] = cands[0]
		default:
			best, bestScore, tie := "", -1, false
			for _, c := range cands {
				s := tailScore(orig, c)
				switch {
				case s > bestScore:
					best, bestScore, tie = c, s, false
				case s == bestScore:
					tie = true
				}
			}
			if tie {
				res.Ambiguous[m] = cands
				res.NotFound = append(res.NotFound, m)
			} else {
				res.Found[m] = best
			}
		}
	}
	return res
}

func (r *SearchResult) Apply() {
	for m, p := range r.Found {
		if m.OriginalPath == nil {
			orig := *m.Path
			m.OriginalPath = &orig
		}
		np := p
		m.Path = &np
	}
}

func SearchMedia(p *core.Project, roots []string, depth int) *SearchResult {
	missing := MissingMedia(p)
	if len(missing) == 0 || len(roots) == 0 {
		return &SearchResult{Missing: missing, Found: map[*core.MediaRef]string{},
			Ambiguous: map[*core.MediaRef][]string{}, NotFound: missing}
	}
	res := BuildIndex(roots, depth).Resolve(missing)
	res.Apply()
	return res
}
