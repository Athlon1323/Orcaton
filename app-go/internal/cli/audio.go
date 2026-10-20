package cli

import (
	"errors"
	"fmt"

	"orcaton/internal/dsp"
	"orcaton/internal/formats"
	"orcaton/internal/media"
	"orcaton/internal/num"
)

func cmdProbe(c *ctx) int {
	m := c.Arg("media")
	info, err := media.Probe(m)
	if err != nil {
		var nf *media.FFmpegNotFound
		if errors.As(err, &nf) {
			c.println(err.Error())
			return 3
		}
		c.printf("Ошибка: %v\n", err)
		return 1
	}
	fm, _ := info["format"].(map[string]any)
	g := func(m map[string]any, k string) string {
		if v, ok := m[k]; ok && v != nil {
			return fmt.Sprint(v)
		}
		return "None"
	}
	tags := "{}"
	if fm != nil {
		if tg, ok := fm["tags"]; ok {
			tags = fmt.Sprint(tg)
		}
	}
	c.printf("%s длительность=%s c  tags=%s\n", formats.BaseName(m), g(fm, "duration"), tags)
	sts, _ := info["streams"].([]any)
	for _, s := range sts {
		st, _ := s.(map[string]any)
		tc := "None"
		if tg, ok := st["tags"].(map[string]any); ok {
			if v, ok := tg["timecode"]; ok {
				tc = fmt.Sprint(v)
			}
		}
		c.printf("  #%s %s %s fps=%s avg=%s sr=%s ch=%s tc=%s\n", g(st, "index"), g(st, "codec_type"),
			g(st, "codec_name"), g(st, "r_frame_rate"), g(st, "avg_frame_rate"), g(st, "sample_rate"),
			g(st, "channels"), tc)
	}
	return 0
}

func cmdOffset(c *ctx) int {
	sr := c.Int("sr")
	xa, err := media.ExtractAudio(c.Arg("a"), sr, media.AudioOpts{})
	if err != nil {
		return ffmpegFail(c, err)
	}
	xb, err := media.ExtractAudio(c.Arg("b"), sr, media.AudioOpts{})
	if err != nil {
		return ffmpegFail(c, err)
	}
	maxOff := -1.0
	if c.Has("max-offset") {
		maxOff = c.Float("max-offset")
	}
	r := dsp.FindOffset(xa, xb, sr, 80, maxOff, 2.0)
	c.printf("B начинается в A на %s c  уверенность=%s  пик=%s\n", num.Signed(r.OffsetSec, 3),
		num.Fixed(r.Confidence, 1), num.Fixed(r.Peak, 3))
	return 0
}
