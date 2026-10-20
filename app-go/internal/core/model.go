package core

import (
	"sort"

	"orcaton/internal/num"
)

type TrackKind string

const (
	Video TrackKind = "video"
	Audio TrackKind = "audio"
)

type SyncStatus string

const (
	StatusUnknown    SyncStatus = "unknown"
	StatusSynced     SyncStatus = "synced"
	StatusByTimecode SyncStatus = "by_tc"
	StatusUnsynced   SyncStatus = "unsynced"
)

type MediaRef struct {
	ID            string
	Name          string
	Path          *string
	URL           *string
	Rate          *Rate
	Duration      num.Q
	TCStart       num.Q
	TCString      *string
	Width         *int
	Height        *int
	AudioChannels *int
	SampleRate    *int
	HasVideo      bool
	HasAudio      bool

	OriginalPath    *string
	Offline         bool
	FileName        string
	TCDisplayFormat string
	DrtVideo        bool
}

func (m *MediaRef) PathOr() string {
	if m == nil || m.Path == nil {
		return ""
	}
	return *m.Path
}

type Clip struct {
	Name        string
	Media       *MediaRef
	Start       num.Q
	Duration    num.Q
	SourceIn    num.Q
	Enabled     bool
	Label       *string
	SyncStatus  SyncStatus
	LinkGroup   *string
	SourceTrack *int
	ID          *string

	MasterClipID *string
	DrtTag       string
}

func (c *Clip) End() num.Q { return num.Add(c.Start, c.Duration) }

func (c *Clip) SourceOut() num.Q { return num.Add(c.SourceIn, c.Duration) }

func (c *Clip) LabelOr() string {
	if c.Label == nil {
		return ""
	}
	return *c.Label
}

func (c *Clip) IDOr() string {
	if c.ID == nil {
		return ""
	}
	return *c.ID
}

type Track struct {
	Kind    TrackKind
	Index   int
	Clips   []*Clip
	Name    *string
	Enabled bool
	Locked  bool

	DrtType    *string
	DrtTrackID *string
}

func (t *Track) Label() string {
	p := "A"
	if t.Kind == Video {
		p = "V"
	}
	return p + itoa(t.Index)
}

func (t *Track) SortedClips() []*Clip {
	out := append([]*Clip(nil), t.Clips...)
	sort.SliceStable(out, func(i, j int) bool { return num.Lt(out[i].Start, out[j].Start) })
	return out
}

func (t *Track) Gaps() int {
	cl := t.SortedClips()
	n := 0
	for i := 1; i < len(cl); i++ {
		if !num.Eq(cl[i-1].End(), cl[i].Start) {
			n++
		}
	}
	return n
}

type Sequence struct {
	Name       string
	Rate       Rate
	Tracks     []*Track
	TCStart    num.Q
	Width      *int
	Height     *int
	SampleRate *int

	XmemlID        *string
	TCStringSrc    *string
	DrtEntry       string
	DrtSeqID       *string
	DrtTimelineID  *string
	DrtContainerID *string
}

type TrackClip struct {
	Track *Track
	Clip  *Clip
}

func (s *Sequence) Clips() []TrackClip {
	var out []TrackClip
	for _, t := range s.Tracks {
		for _, c := range t.Clips {
			out = append(out, TrackClip{t, c})
		}
	}
	return out
}

func (s *Sequence) VideoTracks() []*Track { return s.tracksOf(Video) }
func (s *Sequence) AudioTracks() []*Track { return s.tracksOf(Audio) }

func (s *Sequence) tracksOf(k TrackKind) []*Track {
	var out []*Track
	for _, t := range s.Tracks {
		if t.Kind == k {
			out = append(out, t)
		}
	}
	return out
}

func (s *Sequence) Duration() num.Q {
	d := num.I(0)
	for _, tc := range s.Clips() {
		if e := tc.Clip.End(); num.Gt(e, d) {
			d = e
		}
	}
	return d
}

type Project struct {
	Name         string
	Sequences    []*Sequence
	Media        map[string]*MediaRef
	MediaOrder   []string
	SourceFormat string
	SourcePath   string

	XmemlVersion *string
	DBVersions   [][2]string
}

func NewProject(name string) *Project {
	return &Project{Name: name, Media: map[string]*MediaRef{}}
}

func (p *Project) AddMedia(m *MediaRef) {
	if _, ok := p.Media[m.ID]; !ok {
		p.MediaOrder = append(p.MediaOrder, m.ID)
	}
	p.Media[m.ID] = m
}

func (p *Project) MediaList() []*MediaRef {
	out := make([]*MediaRef, 0, len(p.MediaOrder))
	for _, id := range p.MediaOrder {
		out = append(out, p.Media[id])
	}
	return out
}

func (p *Project) Main() *Sequence {
	if len(p.Sequences) == 0 {
		return &Sequence{Rate: RateInt(25), TCStart: num.I(0)}
	}
	return p.Sequences[0]
}

func (p *Project) Clone() *Project {
	q := *p
	q.Media = map[string]*MediaRef{}
	q.MediaOrder = append([]string(nil), p.MediaOrder...)
	mm := map[*MediaRef]*MediaRef{}
	cloneMedia := func(m *MediaRef) *MediaRef {
		if m == nil {
			return nil
		}
		if c, ok := mm[m]; ok {
			return c
		}
		c := *m
		mm[m] = &c
		return &c
	}
	for _, id := range p.MediaOrder {
		q.Media[id] = cloneMedia(p.Media[id])
	}
	q.Sequences = nil
	for _, s := range p.Sequences {
		s2 := *s
		s2.Tracks = nil
		for _, t := range s.Tracks {
			t2 := *t
			t2.Clips = nil
			for _, c := range t.Clips {
				c2 := *c
				c2.Media = cloneMedia(c.Media)
				t2.Clips = append(t2.Clips, &c2)
			}
			s2.Tracks = append(s2.Tracks, &t2)
		}
		q.Sequences = append(q.Sequences, &s2)
	}
	q.DBVersions = append([][2]string(nil), p.DBVersions...)
	return &q
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func Str(s string) *string { return &s }

func Int(n int) *int { return &n }
