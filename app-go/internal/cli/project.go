package cli

import (
	"fmt"
	"strings"

	"orcaton/internal/formats"
	"orcaton/internal/media"
	"orcaton/internal/num"
	"orcaton/internal/pyio"
)

func cmdVersion(c *ctx) int {
	c.printf("orcaton %s\n", Version)
	p, err := media.Bin("ffmpeg")
	if err != nil {
		c.printf("FFmpeg: %v\n", err)
		return 0
	}
	v, _ := media.ToolVersion(p)
	old := ""
	if media.FFmpegVersionOld(v) {
		old = fmt.Sprintf(" — older than %s: should work but is untested, better update", media.MinFFmpeg)
	}
	c.printf("FFmpeg %s (%s)%s\n", v, p, old)
	return 0
}

func cmdFormats(c *ctx) int {
	t := newTable("Direction", "Key", "Name", "Extensions")
	for _, i := range formats.Importers() {
		t.add("import", i.Name(), i.Title(), strings.Join(i.Extensions(), ", "))
	}
	for _, e := range formats.Exporters() {
		t.add("export", e.Name(), e.Title(), strings.Join(e.Extensions(), ", "))
	}
	t.print(c.out)
	return 0
}

func cmdInspect(c *ctx) int {
	p, code := c.load(c.Arg("path"), c.Str("format"), nil)
	if code != 0 {
		return code
	}
	c.printf("%s  format=%s  media files=%d\n", p.Name, p.SourceFormat, len(p.Media))
	for _, s := range p.Sequences {
		c.printf("\n%s  %s fps  %sx%s  duration=%s s  start TC=%s\n", s.Name, s.Rate,
			intOrNone(s.Width), intOrNone(s.Height), num.Fixed(num.Float(s.Duration()), 1), tcOf(s, 0))
		t := newTable("Track", "Clips", "Gaps", "Source FPS", "Labels", "Folders")
		for _, tr := range s.Tracks {
			fps, labels, dirs := newCounter(), newCounter(), newCounter()
			for _, cl := range tr.Clips {
				k := "?"
				if cl.Media != nil && cl.Media.Rate != nil {
					k = cl.Media.Rate.String()
				}
				fps.inc(k)
				lb := "—"
				if cl.Label != nil && *cl.Label != "" {
					lb = *cl.Label
				}
				labels.inc(lb)
				d := ""
				if cl.Media != nil {
					d = formats.ParentNameOS(cl.Media.PathOr())
				}
				if d == "" {
					d = "?"
				}
				dirs.inc(d)
			}
			t.add(tr.Label(), fmt.Sprint(len(tr.Clips)), fmt.Sprint(tr.Gaps()), fps.String(),
				labels.String(), dirs.String())
		}
		t.print(c.out)
	}
	return 0
}

func cmdClips(c *ctx) int {
	p, code := c.load(c.Arg("path"), c.Str("format"), nil)
	if code != 0 {
		return code
	}
	s := p.Main()
	var rows [][]string
	track := c.Str("track")
	for _, tr := range s.Tracks {
		if track != "" && strings.ToUpper(tr.Label()) != strings.ToUpper(track) {
			continue
		}
		for _, cl := range tr.SortedClips() {
			fps, ftc := "", ""
			if cl.Media != nil {
				if cl.Media.Rate != nil {
					fps = cl.Media.Rate.String()
				}
				if cl.Media.TCString != nil {
					ftc = *cl.Media.TCString
				}
			}
			start := num.Float(cl.Start)
			rows = append(rows, []string{tr.Label(), cl.Name, num.Fixed(start, 3), tcOf(s, start),
				num.Fixed(num.Float(cl.Duration), 3), num.Fixed(num.Float(cl.SourceIn), 3), fps, ftc,
				cl.LabelOr()})
		}
	}
	head := []string{"track", "name", "start_s", "start_tc", "dur_s", "src_in_s", "fps", "file_tc", "label"}
	if out := c.Str("csv"); out != "" {
		if err := pyio.WriteCSVFile(out, head, rows); err != nil {
			c.printf("Error: %v\n", err)
			return 1
		}
		c.printf("Saved: %s (%d rows)\n", out, len(rows))
		return 0
	}
	t := newTable(head...)
	for _, r := range rows {
		t.add(r...)
	}
	t.print(c.out)
	return 0
}

func cmdConvert(c *ctx) int {
	p, code := c.load(c.Arg("src"), c.Str("format"), c.StrPtr("map"))
	if code != 0 {
		return code
	}
	if err := formats.WriteProject(p, c.Arg("dst"), c.Str("to")); err != nil {
		c.printf("Error: %v\n", err)
		return 2
	}
	c.printf("Written: %s\n", c.Arg("dst"))
	return 0
}
