package pyio

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func ParseISO(text string) (float64, error) {
	s := strings.TrimSpace(strings.ReplaceAll(text, "Z", "+00:00"))
	bad := fmt.Errorf("Invalid isoformat string: %q", text)
	if len(s) < 10 {
		return 0, bad
	}
	y, e1 := strconv.Atoi(s[0:4])
	mo, e2 := strconv.Atoi(s[5:7])
	d, e3 := strconv.Atoi(s[8:10])
	if e1 != nil || e2 != nil || e3 != nil || s[4] != '-' || s[7] != '-' {
		return 0, bad
	}
	hh, mm, ss, us := 0, 0, 0, 0
	offset := 0
	rest := ""
	if len(s) > 10 {
		rest = s[11:]
	}
	if rest != "" {
		tz := ""
		if i := strings.LastIndexAny(rest, "+-"); i >= 0 {
			tz = rest[i:]
			rest = rest[:i]
		}
		parts := strings.Split(rest, ":")
		if len(parts) < 1 || len(parts) > 3 {
			return 0, bad
		}
		var err error
		if hh, err = strconv.Atoi(parts[0]); err != nil {
			return 0, bad
		}
		if len(parts) > 1 {
			if mm, err = strconv.Atoi(parts[1]); err != nil {
				return 0, bad
			}
		}
		if len(parts) > 2 {
			sec := parts[2]
			frac := ""
			if i := strings.IndexAny(sec, ".,"); i >= 0 {
				frac = sec[i+1:]
				sec = sec[:i]
			}
			if ss, err = strconv.Atoi(sec); err != nil {
				return 0, bad
			}
			if frac != "" {
				if len(frac) > 6 {
					frac = frac[:6]
				}
				frac += strings.Repeat("0", 6-len(frac))
				if us, err = strconv.Atoi(frac); err != nil {
					return 0, bad
				}
			}
		}
		if tz != "" {
			sign := 1
			if tz[0] == '-' {
				sign = -1
			}
			t := strings.ReplaceAll(tz[1:], ":", "")
			if len(t) < 2 {
				return 0, bad
			}
			oh, err1 := strconv.Atoi(t[:2])
			om := 0
			var err2 error
			if len(t) >= 4 {
				om, err2 = strconv.Atoi(t[2:4])
			}
			if err1 != nil || err2 != nil {
				return 0, bad
			}
			offset = sign * (oh*3600 + om*60)
		}
	}
	tm := time.Date(y, time.Month(mo), d, hh, mm, ss, 0, time.UTC)
	if tm.Year() != y || int(tm.Month()) != mo || tm.Day() != d || hh > 23 || mm > 59 || ss > 59 {
		return 0, bad
	}
	micros := (tm.Unix()-int64(offset))*1_000_000 + int64(us)
	return float64(micros) / 1e6, nil
}

func ISOUTC(t float64) string {
	whole, frac := math.Modf(t)
	us := int64(math.RoundToEven(frac * 1e6))
	sec := int64(whole)
	if us >= 1_000_000 {
		sec++
		us -= 1_000_000
	} else if us < 0 {
		sec--
		us += 1_000_000
	}
	tm := time.Unix(sec, 0).UTC()
	s := tm.Format("2006-01-02T15:04:05")
	if us != 0 {
		s += fmt.Sprintf(".%06d", us)
	}
	return s + "+00:00"
}

func LocalDay(t float64) string {
	return time.Unix(int64(math.Floor(t)), 0).Local().Format("2006-01-02")
}
