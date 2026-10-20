package drt

import (
	"encoding/binary"
	"encoding/hex"
	"math"
	"math/big"
	"strings"
	"unicode"
	"unicode/utf16"

	"orcaton/internal/num"
)

var zstdMagic = []byte{0x28, 0xb5, 0x2f, 0xfd}

func le64(b []byte) float64 { return math.Float64frombits(binary.LittleEndian.Uint64(b)) }

func DecodeTime(value *string) num.Q {
	if value == nil {
		return nil
	}
	v := strings.TrimSpace(*value)
	if v == "" {
		return nil
	}
	if whole, fracHex, ok := strings.Cut(v, "|"); ok {
		if len(fracHex) > 16 {
			fracHex = fracHex[:16]
		}
		b, err := hex.DecodeString(fracHex)
		if err != nil || len(b) < 8 {
			return nil
		}
		w, ok := new(big.Int).SetString(whole, 10)
		if !ok {
			return nil
		}
		return num.Add(new(big.Rat).SetInt(w), num.FloatLimit(le64(b), 1_000_000))
	}
	q, ok := num.ParseFraction(v)
	if !ok {
		return nil
	}
	return q
}

func EncodeTime(frames num.Q) string {
	if frames.IsInt() {
		return frames.Num().String()
	}
	whole := num.Floor(frames)
	frac := num.Float(num.Sub(frames, new(big.Rat).SetInt(whole)))
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], math.Float64bits(frac))
	return whole.String() + "|" + hex.EncodeToString(b[:])
}

func DecodeExtents(value *string) (float64, float64, bool) {
	if value == nil {
		return 0, 0, false
	}
	v := strings.TrimSpace(*value)
	if len(v) < 32 {
		return 0, 0, false
	}
	b, err := hex.DecodeString(v[:32])
	if err != nil {
		return 0, 0, false
	}
	return le64(b[:8]), le64(b[8:16]), true
}

func EncodeExtents(start, dur float64) string {
	var b [16]byte
	binary.LittleEndian.PutUint64(b[:8], math.Float64bits(start))
	binary.LittleEndian.PutUint64(b[8:], math.Float64bits(dur))
	return hex.EncodeToString(b[:])
}

func DecodeDoubleLE(value *string) (float64, bool) {
	if value == nil {
		return 0, false
	}
	v := strings.TrimSpace(*value)
	if len(v) < 16 {
		return 0, false
	}
	b, err := hex.DecodeString(v[:16])
	if err != nil {
		return 0, false
	}
	return le64(b), true
}

func DecodeResolution(value *string) (int, int, bool) {
	if value == nil {
		return 0, 0, false
	}
	v := strings.TrimSpace(*value)
	if len(v) < 32 {
		return 0, 0, false
	}
	b, err := hex.DecodeString(v[:32])
	if err != nil {
		return 0, 0, false
	}
	return int(int64(binary.BigEndian.Uint64(b[:8]))), int(int64(binary.BigEndian.Uint64(b[8:]))), true
}

func IsZstdBlob(value string) bool {
	v := strings.TrimSpace(value)
	if len(v) > 64 {
		v = v[:64]
	}
	b, err := hex.DecodeString(v)
	if err != nil {
		return false
	}
	if len(b) > 16 {
		b = b[:16]
	}
	return strings.Contains(string(b), string(zstdMagic))
}

func UTF16BEStrings(value string, minLen int) []string {
	b, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return nil
	}
	var out []string
	i := 0
	for i+4 <= len(b) {
		n := int(binary.BigEndian.Uint32(b[i : i+4]))
		if 2*minLen <= n && n <= 512 && n%2 == 0 && i+4+n <= len(b) {
			if s, ok := decodeUTF16BE(b[i+4 : i+4+n]); ok && isPrintable(s) {
				out = append(out, s)
				i += 4 + n
				continue
			}
		}
		i++
	}
	return out
}

func decodeUTF16BE(b []byte) (string, bool) {
	u := make([]uint16, len(b)/2)
	for k := range u {
		u[k] = binary.BigEndian.Uint16(b[2*k:])
	}
	for k := 0; k < len(u); k++ {
		c := u[k]
		if c >= 0xD800 && c <= 0xDBFF {
			if k+1 >= len(u) || u[k+1] < 0xDC00 || u[k+1] > 0xDFFF {
				return "", false
			}
			k++
		} else if c >= 0xDC00 && c <= 0xDFFF {
			return "", false
		}
	}
	return string(utf16.Decode(u)), true
}

func isPrintable(s string) bool {
	for _, r := range s {
		if r == ' ' {
			continue
		}
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}
