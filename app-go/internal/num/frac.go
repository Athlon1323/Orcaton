package num

import (
	"math"
	"math/big"
)

type Q = *big.Rat

func I(n int64) Q { return new(big.Rat).SetInt64(n) }

func F(n, d int64) Q { return big.NewRat(n, d) }

func FromFloat(x float64) Q {
	r := new(big.Rat)
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return r
	}
	r.SetFloat64(x)
	return r
}

func Copy(a Q) Q { return new(big.Rat).Set(a) }

func Add(a, b Q) Q { return new(big.Rat).Add(a, b) }
func Sub(a, b Q) Q { return new(big.Rat).Sub(a, b) }
func Mul(a, b Q) Q { return new(big.Rat).Mul(a, b) }
func Div(a, b Q) Q { return new(big.Rat).Quo(a, b) }
func Neg(a Q) Q    { return new(big.Rat).Neg(a) }

func Cmp(a, b Q) int { return a.Cmp(b) }

func Lt(a, b Q) bool { return a.Cmp(b) < 0 }
func Le(a, b Q) bool { return a.Cmp(b) <= 0 }
func Gt(a, b Q) bool { return a.Cmp(b) > 0 }
func Ge(a, b Q) bool { return a.Cmp(b) >= 0 }
func Eq(a, b Q) bool { return a.Cmp(b) == 0 }

func IsZero(a Q) bool { return a.Sign() == 0 }

func Max(a, b Q) Q {
	if b.Cmp(a) > 0 {
		return b
	}
	return a
}

func Min(a, b Q) Q {
	if b.Cmp(a) < 0 {
		return b
	}
	return a
}

func Float(a Q) float64 {
	f, _ := a.Float64()
	return f
}

func Abs(a Q) Q { return new(big.Rat).Abs(a) }

func Floor(a Q) *big.Int {
	n, d := a.Num(), a.Denom()
	q := new(big.Int)
	m := new(big.Int)
	q.DivMod(n, d, m)
	return q
}

func RoundHalfEven(a Q) *big.Int {
	fl := Floor(a)
	rem := Sub(a, new(big.Rat).SetInt(fl))
	c := rem.Cmp(big.NewRat(1, 2))
	if c > 0 || (c == 0 && fl.Bit(0) == 1) {
		return fl.Add(fl, big.NewInt(1))
	}
	return fl
}

func RoundInt(a Q) int64 { return RoundHalfEven(a).Int64() }

func IsInt(a Q) bool { return a.IsInt() }

func LimitDenominator(a Q, maxDen int64) Q {
	md := big.NewInt(maxDen)
	if a.Denom().Cmp(md) <= 0 {
		return Copy(a)
	}
	p0, q0, p1, q1 := big.NewInt(0), big.NewInt(1), big.NewInt(1), big.NewInt(0)
	n := new(big.Int).Set(a.Num())
	d := new(big.Int).Set(a.Denom())
	for {
		a := new(big.Int)
		a.Div(n, d)
		q2 := new(big.Int).Add(q0, new(big.Int).Mul(a, q1))
		if q2.Cmp(md) > 0 {
			break
		}
		np0 := new(big.Int).Set(p1)
		nq0 := new(big.Int).Set(q1)
		np1 := new(big.Int).Add(p0, new(big.Int).Mul(a, p1))
		p0, q0, p1, q1 = np0, nq0, np1, q2
		nn := new(big.Int).Set(d)
		nd := new(big.Int).Sub(n, new(big.Int).Mul(a, d))
		n, d = nn, nd
		if d.Sign() == 0 {
			break
		}
	}
	k := new(big.Int).Sub(md, q0)
	k.Div(k, q1)
	b1 := new(big.Rat).SetFrac(new(big.Int).Add(p0, new(big.Int).Mul(k, p1)),
		new(big.Int).Add(q0, new(big.Int).Mul(k, q1)))
	b2 := new(big.Rat).SetFrac(p1, q1)
	d2 := Abs(Sub(b2, a))
	d1 := Abs(Sub(b1, a))
	if d2.Cmp(d1) <= 0 {
		return b2
	}
	return b1
}

func FloatLimit(x float64, maxDen int64) Q { return LimitDenominator(FromFloat(x), maxDen) }

func Sum(xs []Q) Q {
	s := new(big.Rat)
	for _, x := range xs {
		s.Add(s, x)
	}
	return s
}

func String(a Q) string {
	if a.IsInt() {
		return a.Num().String()
	}
	return a.Num().String() + "/" + a.Denom().String()
}

func ParseFraction(s string) (Q, bool) {
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, false
	}
	return r, true
}
