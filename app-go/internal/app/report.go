package app

import (
	"fmt"
	"sort"

	"orcaton/internal/core"
	"orcaton/internal/engine"
	"orcaton/internal/num"
	"orcaton/internal/pyio"
)

const Version = "0.0.47"

type Partner struct {
	Name               string
	Score, Ratio, Peak float64
}

func BestPartners(r *engine.Report) map[int]Partner {
	partner := map[int]Partner{}
	for _, e := range r.TreeEdges {
		for _, pr := range [][2]int{{e.A, e.B}, {e.B, e.A}} {
			me, other := pr[0], pr[1]
			cur, ok := partner[me]
			if !ok || e.Score() > cur.Score {
				partner[me] = Partner{r.Units[other].Name(), e.Score(), e.Res.Ratio, e.Res.Peak}
			}
		}
	}
	for _, u := range r.Units {
		if u.DupOf != nil && u.Status == core.StatusSynced {
			if pt, ok := partner[u.DupOf.UID]; ok {
				partner[u.UID] = pt
			}
		}
	}
	return partner
}

func ReportOrder(r *engine.Report) []*engine.Unit {
	units := append([]*engine.Unit(nil), r.Units...)
	sort.SliceStable(units, func(i, j int) bool {
		if units[i].Lane != units[j].Lane {
			return units[i].Lane < units[j].Lane
		}
		return units[i].Order.Less(units[j].Order)
	})
	return units
}

func Reason(u *engine.Unit) string {
	if u.Status == core.StatusByTimecode {
		return "by camera clock (audio did not confirm)"
	}
	if u.Error != nil {
		return *u.Error
	}
	return ""
}

func ReasonCode(u *engine.Unit) string {
	if u.Status == core.StatusByTimecode {
		return engine.ReasonByClock
	}
	if u.Error != nil {
		return u.Reason
	}
	return ""
}

func WriteReport(path string, r *engine.Report) error {
	return wrap(KindWrite, writeReport(path, r))
}

func writeReport(path string, r *engine.Report) error {
	partner := BestPartners(r)
	var rows [][]string
	for _, u := range ReportOrder(r) {
		pos, grp := "", ""
		if u.Position != nil {
			pos = num.Fixed(num.Float(u.Position), 3)
		}
		if u.Component >= 0 {
			grp = fmt.Sprint(u.Component)
		}
		pn, ps, pr, pp := "", "", "", ""
		if pt, ok := partner[u.UID]; ok {
			pn, ps, pr, pp = pt.Name, num.Fixed(pt.Score, 1), num.Fixed(pt.Ratio, 2), num.Fixed(pt.Peak, 2)
		}
		rec := ""
		if u.RecTime != nil {
			rec = pyio.ISOUTC(*u.RecTime)
		}
		rows = append(rows, []string{u.Lane, u.Name(), string(u.Status), pos,
			num.Fixed(num.Float(u.Duration), 3), num.Fixed(num.Float(u.SourceIn), 3), grp,
			pn, ps, pr, pp, rec, Reason(u)})
	}
	return pyio.WriteCSVFile(path, []string{"lane", "file", "status", "position_s", "duration_s",
		"source_in_s", "group", "best_partner", "confidence", "ratio", "peak", "record_start",
		"reason"}, rows)
}
