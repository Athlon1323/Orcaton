package media

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"

	"orcaton/internal/dsp"
	"orcaton/internal/formats"
)

func DefaultCacheDir() string {
	if env := os.Getenv("ORCATON_CACHE_DIR"); env != "" {
		return env
	}
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, "orcaton", "cache")
	case "darwin":
		return filepath.Join(home, "Library", "Caches", "orcaton")
	}
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "orcaton")
}

type FeatureParams struct {
	SampleRate int
	Hop        int
	AudioMode  string
}

func DefaultParams() FeatureParams { return FeatureParams{8000, 80, "mix"} }

func (p FeatureParams) FPS() float64 { return float64(p.SampleRate) / float64(p.Hop) }

func (p FeatureParams) Tag() string {
	return fmt.Sprintf("v%d-sr%d-h%d-%s", dsp.FeatureVersion, p.SampleRate, p.Hop, p.AudioMode)
}

type OnDone func(path string, err error)

type FeatureSource interface {
	Params() FeatureParams
	Has(path string) bool
	Get(path string) ([]float32, error)
	Prefetch(ctx context.Context, paths []string, jobs int, onDone OnDone) map[string]error
}

type FeatureStore struct {
	params   FeatureParams
	CacheDir string
	mu       sync.Mutex
	mem      map[string][]float32
}

func NewFeatureStore(cacheDir string, params *FeatureParams) (*FeatureStore, error) {
	p := DefaultParams()
	if params != nil {
		p = *params
	}
	if cacheDir == "" {
		cacheDir = DefaultCacheDir()
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, err
	}
	return &FeatureStore{params: p, CacheDir: cacheDir, mem: map[string][]float32{}}, nil
}

func (s *FeatureStore) Params() FeatureParams { return s.params }

func (s *FeatureStore) Has(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Mode().IsRegular()
}

func absResolved(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	if r, err := filepath.EvalSymlinks(a); err == nil {
		return r
	}
	return a
}

func (s *FeatureStore) key(path string) (string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	raw := fmt.Sprintf("%s|%d|%d|%s-go", absResolved(path), st.Size(), st.ModTime().Unix(),
		s.params.Tag())
	h := sha1.Sum([]byte(raw))
	return hex.EncodeToString(h[:]), nil
}

func (s *FeatureStore) Get(path string) ([]float32, error) {
	return s.getCtx(context.Background(), path)
}

func (s *FeatureStore) getCtx(ctx context.Context, path string) ([]float32, error) {
	key, err := s.key(path)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if a, ok := s.mem[key]; ok {
		s.mu.Unlock()
		return a, nil
	}
	s.mu.Unlock()
	f := filepath.Join(s.CacheDir, key+".npy")
	var arr []float32
	if data, err := os.ReadFile(f); err == nil {
		a, err := ReadNpy(data)
		if err != nil || a.F32 == nil {
			return nil, fmt.Errorf("кэш %s повреждён: %v", f, err)
		}
		arr = a.F32
		now := timeNow()
		_ = os.Chtimes(f, now, now)
	} else {
		arr, err = s.compute(ctx, path)
		if err != nil {
			return nil, err
		}
		tmp := filepath.Join(s.CacheDir, key+".tmp.npy")
		if err := os.WriteFile(tmp, EncodeNpyF32(arr), 0o644); err == nil {
			if err := os.Rename(tmp, f); err != nil {
				os.Remove(tmp)
			}
		}
	}
	s.mu.Lock()
	s.mem[key] = arr
	s.mu.Unlock()
	return arr, nil
}

func (s *FeatureStore) compute(ctx context.Context, path string) ([]float32, error) {
	var o AudioOpts
	if s.params.AudioMode == "first_stream" || s.params.AudioMode == "first_channel" {
		z := 0
		o.Stream = &z
	}
	if s.params.AudioMode == "first_channel" {
		z := 0
		o.Channel = &z
	}
	st := dsp.NewOnsetStream(s.params.SampleRate, s.params.Hop)
	if err := StreamAudioCtx(ctx, path, s.params.SampleRate, o, st.Write); err != nil {
		return nil, err
	}
	return st.Finish(), nil
}

func (s *FeatureStore) Prefetch(ctx context.Context, paths []string, jobs int, onDone OnDone) map[string]error {
	uniq := uniqStrings(paths)
	if jobs < 1 {
		jobs = 1
	}
	errs := map[string]error{}
	var mu sync.Mutex
	ch := make(chan string)
	var wg sync.WaitGroup
	for w := 0; w < jobs; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range ch {
				if ctx.Err() != nil {
					continue
				}
				_, err := s.getCtx(ctx, p)
				if ctx.Err() != nil {
					continue
				}
				mu.Lock()
				if err != nil {
					errs[p] = err
				}
				if onDone != nil {
					onDone(p, err)
				}
				mu.Unlock()
			}
		}()
	}
	for _, p := range uniq {
		if ctx.Err() != nil {
			break
		}
		ch <- p
	}
	close(ch)
	wg.Wait()
	return errs
}

func uniqStrings(xs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

func BundleKey(path string) string {
	return formats.Casefold(formats.NFC(formats.BaseName(path)))
}

type FeatureBundle struct {
	params FeatureParams
	data   map[string][]float32
}

func LoadBundle(paths []string) (*FeatureBundle, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("нужен хотя бы один набор")
	}
	b := &FeatureBundle{data: map[string][]float32{}}
	var params *FeatureParams
	for _, one := range paths {
		keys, arrs, err := ReadNpz(one)
		if err != nil {
			return nil, err
		}
		meta, ok := arrs["__meta__"]
		if !ok || meta.Bytes == nil {
			return nil, fmt.Errorf("%s: нет __meta__", one)
		}
		var m map[string]any
		if err := json.Unmarshal(meta.Bytes, &m); err != nil {
			return nil, fmt.Errorf("%s: __meta__: %v", one, err)
		}
		for _, k := range keys {
			if k != "__meta__" && arrs[k].F32 != nil {
				b.data[k] = arrs[k].F32
			}
		}
		fv := toInt(m["feature_version"])
		if fv != dsp.FeatureVersion {
			return nil, fmt.Errorf("%s: признаки версии %d, движок ждёт %d — пересоздайте набор",
				one, fv, dsp.FeatureVersion)
		}
		prm := FeatureParams{toInt(m["sample_rate"]), toInt(m["hop"]), fmt.Sprint(m["audio_mode"])}
		if params != nil && prm != *params {
			return nil, fmt.Errorf("%s: параметры анализа отличаются от первого набора", one)
		}
		params = &prm
	}
	b.params = *params
	return b, nil
}

func toInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(x)
		return n
	}
	return 0
}

func (b *FeatureBundle) Params() FeatureParams { return b.params }

func (b *FeatureBundle) Has(path string) bool {
	_, ok := b.data[BundleKey(path)]
	return ok
}

func (b *FeatureBundle) Get(path string) ([]float32, error) {
	a, ok := b.data[BundleKey(path)]
	if !ok {
		return nil, fmt.Errorf("KeyError: %s", BundleKey(path))
	}
	return a, nil
}

func (b *FeatureBundle) Prefetch(ctx context.Context, paths []string, _ int, onDone OnDone) map[string]error {
	for _, p := range paths {
		if ctx.Err() != nil {
			break
		}
		if onDone != nil {
			onDone(p, nil)
		}
	}
	return map[string]error{}
}

func SaveBundle(src FeatureSource, paths []string, out string) (int, error) {
	var entries []NpzEntry
	seen := map[string]int{}
	for _, p := range uniqStrings(paths) {
		if !src.Has(p) {
			continue
		}
		a, err := src.Get(p)
		if err != nil {
			return 0, err
		}
		k := BundleKey(p)
		e := NpzEntry{Name: k, Data: EncodeNpyF16(a)}
		if i, ok := seen[k]; ok {
			entries[i] = e
		} else {
			seen[k] = len(entries)
			entries = append(entries, e)
		}
	}
	prm := src.Params()
	meta := fmt.Sprintf(`{"sample_rate": %d, "hop": %d, "audio_mode": %q, "feature_version": %d}`,
		prm.SampleRate, prm.Hop, prm.AudioMode, dsp.FeatureVersion)
	entries = append(entries, NpzEntry{Name: "__meta__", Data: EncodeNpyU8([]byte(meta))})
	if dir := filepath.Dir(out); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return 0, err
		}
	}
	if err := WriteNpz(out, entries); err != nil {
		return 0, err
	}
	return len(entries) - 1, nil
}
