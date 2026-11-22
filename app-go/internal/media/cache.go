package media

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"orcaton/internal/pyio"
)

var (
	entryRe = regexp.MustCompile(`^[0-9a-f]{40}\.npy$`)
	tmpRe   = regexp.MustCompile(`^[0-9a-f]{40}\.tmp\.npy$`)
	ageRe   = regexp.MustCompile(`^\s*(\d+(?:\.\d+)?)\s*([smhdw]?)\s*$`)
	timeNow = time.Now
)

const TmpGraceSec = 3600.0

var units = map[string]float64{"s": 1, "m": 60, "h": 3600, "d": 86400, "w": 7 * 86400}

type CacheStats struct {
	Path         string
	Exists       bool
	Entries      int
	EntriesBytes int64
	Tmp          int
	TmpBytes     int64
	Other        int
	Oldest       *float64
	Newest       *float64
}

func (s *CacheStats) TotalBytes() int64 { return s.EntriesBytes + s.TmpBytes }

func (s *CacheStats) AsDict() *pyio.OMap {
	m := pyio.NewOMap()
	m.Set("path", s.Path).Set("exists", s.Exists).Set("entries", s.Entries).
		Set("entries_bytes", s.EntriesBytes).Set("tmp", s.Tmp).Set("tmp_bytes", s.TmpBytes).
		Set("total_bytes", s.TotalBytes()).Set("other", s.Other)
	if s.Oldest != nil {
		m.Set("oldest", pyio.Float(*s.Oldest))
	} else {
		m.Set("oldest", nil)
	}
	if s.Newest != nil {
		m.Set("newest", pyio.Float(*s.Newest))
	} else {
		m.Set("newest", nil)
	}
	return m
}

type CleanResult struct {
	Removed      int
	RemovedBytes int64
	Kept         int
	Errors       []string
	DryRun       bool
}

func (r *CleanResult) AsDict() *pyio.OMap {
	errs := make([]any, len(r.Errors))
	for i, e := range r.Errors {
		errs[i] = e
	}
	return pyio.NewOMap().Set("removed", r.Removed).Set("removed_bytes", r.RemovedBytes).
		Set("kept", r.Kept).Set("errors", errs).Set("dry_run", r.DryRun)
}

func ParseAge(text string) (float64, error) {
	m := ageRe.FindStringSubmatch(strings.ToLower(text))
	if m == nil {
		return 0, fmt.Errorf("Cannot parse age '%s': expected e.g. 30d, 12h, 2w", text)
	}
	v, _ := strconv.ParseFloat(m[1], 64)
	u := m[2]
	if u == "" {
		u = "d"
	}
	return v * units[u], nil
}

type cfile struct {
	path  string
	name  string
	kind  string
	size  int64
	mtime float64
}

func mtimeOf(st os.FileInfo) float64 {
	t := st.ModTime()
	return float64(t.Unix()) + float64(t.Nanosecond())*1e-9
}

func listCache(path string) []cfile {
	ents, err := os.ReadDir(path)
	if err != nil {
		return nil
	}
	var out []cfile
	for _, e := range ents {
		full := filepath.Join(path, e.Name())
		st, err := os.Stat(full)
		if err != nil || !st.Mode().IsRegular() {
			continue
		}
		kind := "other"
		if tmpRe.MatchString(e.Name()) {
			kind = "tmp"
		} else if entryRe.MatchString(e.Name()) {
			kind = "entry"
		}
		out = append(out, cfile{full, e.Name(), kind, st.Size(), mtimeOf(st)})
	}
	return out
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func Stats(path string) *CacheStats {
	s := &CacheStats{Path: path, Exists: isDir(path)}
	for _, f := range listCache(path) {
		switch f.kind {
		case "entry":
			s.Entries++
			s.EntriesBytes += f.size
			mt := f.mtime
			if s.Oldest == nil || mt < *s.Oldest {
				s.Oldest = &mt
			}
			mt2 := f.mtime
			if s.Newest == nil || mt2 > *s.Newest {
				s.Newest = &mt2
			}
		case "tmp":
			s.Tmp++
			s.TmpBytes += f.size
		default:
			s.Other++
		}
	}
	return s
}

func remove(files []cfile, res *CleanResult) {
	for _, f := range files {
		if !res.DryRun {
			if err := os.Remove(f.path); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", f.name, err))
				continue
			}
		}
		res.Removed++
		res.RemovedBytes += f.size
	}
}

func Clear(path string, dryRun bool) *CleanResult {
	res := &CleanResult{DryRun: dryRun}
	var victims []cfile
	for _, f := range listCache(path) {
		if f.kind != "other" {
			victims = append(victims, f)
		}
	}
	remove(victims, res)
	return res
}

func Prune(path string, olderThanSec float64, dryRun bool, now float64) *CleanResult {
	if now < 0 {
		t := timeNow()
		now = float64(t.Unix()) + float64(t.Nanosecond())*1e-9
	}
	res := &CleanResult{DryRun: dryRun}
	var victims []cfile
	for _, f := range listCache(path) {
		age := now - f.mtime
		if (f.kind == "entry" && age > olderThanSec) || (f.kind == "tmp" && age > TmpGraceSec) {
			victims = append(victims, f)
		} else if f.kind == "entry" {
			res.Kept++
		}
	}
	remove(victims, res)
	return res
}

func HumanBytes(n int64) string {
	size := float64(n)
	for _, unit := range []string{"B", "KB", "MB", "GB"} {
		if size < 1024 || unit == "GB" {
			if unit == "B" {
				return strconv.FormatFloat(size, 'f', 0, 64) + " " + unit
			}
			return strconv.FormatFloat(size, 'f', 1, 64) + " " + unit
		}
		size /= 1024
	}
	return fmt.Sprintf("%d B", n)
}
