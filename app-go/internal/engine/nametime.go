package engine

import "math"

func resolveNameDates(units []*Unit) {
	var all []float64
	for _, u := range units {
		if u.RecTime != nil && u.recAlt == nil {
			all = append(all, *u.RecTime)
		}
	}
	devs, by := byDevice(units)
	for _, d := range devs {
		var own []float64
		var amb []*Unit
		for _, u := range by[d] {
			switch {
			case u.recAlt != nil:
				amb = append(amb, u)
			case u.RecTime != nil:
				own = append(own, *u.RecTime)
			}
		}
		if len(amb) == 0 {
			continue
		}
		ref := own
		if len(ref) == 0 {
			ref = all
		}
		if f, ok := nearestFormat(amb, ref); ok {
			for _, u := range amb {
				for _, c := range u.recAlt {
					if c.Format == f {
						v := c.T
						u.RecTime = &v
					}
				}
			}
		}
		for _, u := range amb {
			u.recAlt = nil
		}
	}
}

func nearestFormat(amb []*Unit, ref []float64) (string, bool) {
	if len(ref) == 0 {
		return "", false
	}
	r := median(ref)
	best, cost := "", math.Inf(1)
	for _, c0 := range amb[0].recAlt {
		sum, ok := 0.0, true
		for _, u := range amb {
			found := false
			for _, c := range u.recAlt {
				if c.Format == c0.Format {
					sum += math.Abs(c.T - r)
					found = true
				}
			}
			ok = ok && found
		}
		if ok && sum < cost {
			best, cost = c0.Format, sum
		}
	}
	return best, best != ""
}
