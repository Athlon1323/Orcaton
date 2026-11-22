package media

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	descrRe = regexp.MustCompile(`'descr'\s*:\s*'([^']*)'`)
	shapeRe = regexp.MustCompile(`'shape'\s*:\s*\(([^)]*)\)`)
	fortRe  = regexp.MustCompile(`'fortran_order'\s*:\s*(True|False)`)
)

type NpyArray struct {
	F32   []float32
	Bytes []byte
	Descr string
}

func ReadNpy(data []byte) (*NpyArray, error) {
	if len(data) < 10 || !bytes.Equal(data[:6], []byte("\x93NUMPY")) {
		return nil, errors.New("not a .npy")
	}
	major := data[6]
	var hlen, off int
	if major == 1 {
		hlen, off = int(binary.LittleEndian.Uint16(data[8:10])), 10
	} else {
		if len(data) < 12 {
			return nil, errors.New("truncated .npy")
		}
		hlen, off = int(binary.LittleEndian.Uint32(data[8:12])), 12
	}
	if off+hlen > len(data) {
		return nil, errors.New("truncated .npy header")
	}
	header := string(data[off : off+hlen])
	body := data[off+hlen:]
	dm := descrRe.FindStringSubmatch(header)
	sm := shapeRe.FindStringSubmatch(header)
	if dm == nil || sm == nil {
		return nil, fmt.Errorf(".npy header not understood: %q", header)
	}
	if fm := fortRe.FindStringSubmatch(header); fm != nil && fm[1] == "True" {
		_ = fm
	}
	n := 1
	for _, s := range strings.Split(sm[1], ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		k, err := strconv.Atoi(s)
		if err != nil {
			return nil, err
		}
		if k < 0 || (k > 0 && n > len(body)/k) {
			return nil, fmt.Errorf(".npy dimension %q does not match the data (%d bytes)", sm[1], len(body))
		}
		n *= k
	}
	a := &NpyArray{Descr: dm[1]}
	switch dm[1] {
	case "<f4", "=f4":
		if len(body) < 4*n {
			return nil, errors.New("truncated .npy data")
		}
		a.F32 = make([]float32, n)
		for i := range a.F32 {
			a.F32[i] = math.Float32frombits(binary.LittleEndian.Uint32(body[4*i:]))
		}
	case "<f2", "=f2":
		if len(body) < 2*n {
			return nil, errors.New("truncated .npy data")
		}
		a.F32 = make([]float32, n)
		for i := range a.F32 {
			a.F32[i] = halfToFloat(binary.LittleEndian.Uint16(body[2*i:]))
		}
	case "<f8", "=f8":
		if len(body) < 8*n {
			return nil, errors.New("truncated .npy data")
		}
		a.F32 = make([]float32, n)
		for i := range a.F32 {
			a.F32[i] = float32(math.Float64frombits(binary.LittleEndian.Uint64(body[8*i:])))
		}
	case "|u1", "<u1", "|i1":
		if len(body) < n {
			return nil, errors.New("truncated .npy data")
		}
		a.Bytes = append([]byte(nil), body[:n]...)
	default:
		return nil, fmt.Errorf(".npy type %s is not supported", dm[1])
	}
	return a, nil
}

func npyHeader(descr string, n int) []byte {
	h := fmt.Sprintf("{'descr': '%s', 'fortran_order': False, 'shape': (%d,), }", descr, n)
	total := 10 + len(h) + 1
	pad := (64 - total%64) % 64
	h += strings.Repeat(" ", pad) + "\n"
	var b bytes.Buffer
	b.WriteString("\x93NUMPY\x01\x00")
	binary.Write(&b, binary.LittleEndian, uint16(len(h)))
	b.WriteString(h)
	return b.Bytes()
}

func EncodeNpyF32(x []float32) []byte {
	b := npyHeader("<f4", len(x))
	out := make([]byte, len(b)+4*len(x))
	copy(out, b)
	for i, v := range x {
		binary.LittleEndian.PutUint32(out[len(b)+4*i:], math.Float32bits(v))
	}
	return out
}

func EncodeNpyF16(x []float32) []byte {
	b := npyHeader("<f2", len(x))
	out := make([]byte, len(b)+2*len(x))
	copy(out, b)
	for i, v := range x {
		binary.LittleEndian.PutUint16(out[len(b)+2*i:], floatToHalf(v))
	}
	return out
}

func EncodeNpyU8(x []byte) []byte {
	b := npyHeader("|u1", len(x))
	return append(b, x...)
}

func ReadNpz(path string) ([]string, map[string]*NpyArray, error) {
	z, err := zip.OpenReader(path)
	if err != nil {
		return nil, nil, err
	}
	defer z.Close()
	var keys []string
	out := map[string]*NpyArray{}
	for _, f := range z.File {
		rc, err := f.Open()
		if err != nil {
			return nil, nil, err
		}
		d, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, nil, err
		}
		a, err := ReadNpy(d)
		if err != nil {
			return nil, nil, fmt.Errorf("%s/%s: %w", path, f.Name, err)
		}
		k := strings.TrimSuffix(f.Name, ".npy")
		if _, ok := out[k]; !ok {
			keys = append(keys, k)
		}
		out[k] = a
	}
	return keys, out, nil
}

type NpzEntry struct {
	Name string
	Data []byte
}

func WriteNpz(path string, entries []NpzEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := zip.NewWriter(f)
	for _, e := range entries {
		ww, err := w.CreateHeader(&zip.FileHeader{Name: e.Name + ".npy", Method: zip.Deflate})
		if err != nil {
			f.Close()
			return err
		}
		if _, err := ww.Write(e.Data); err != nil {
			f.Close()
			return err
		}
	}
	if err := w.Close(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func halfToFloat(h uint16) float32 {
	sign := uint32(h>>15) << 31
	exp := int(h>>10) & 0x1f
	mant := uint32(h & 0x3ff)
	switch exp {
	case 0:
		if mant == 0 {
			return math.Float32frombits(sign)
		}
		v := float32(mant) * float32(math.Pow(2, -24))
		if sign != 0 {
			v = -v
		}
		return v
	case 31:
		if mant == 0 {
			return math.Float32frombits(sign | 0x7f800000)
		}
		return math.Float32frombits(sign | 0x7fc00000 | mant<<13)
	}
	return math.Float32frombits(sign | uint32(exp-15+127)<<23 | mant<<13)
}

func floatToHalf(f float32) uint16 {
	b := math.Float32bits(f)
	sign := uint16(b>>16) & 0x8000
	exp := int(b>>23) & 0xff
	mant := b & 0x7fffff
	if exp == 0xff {
		if mant == 0 {
			return sign | 0x7c00
		}
		return sign | 0x7e00
	}
	e := exp - 127 + 15
	if e >= 31 {
		return sign | 0x7c00
	}
	if e <= 0 {
		if e < -10 {
			return sign
		}
		m := mant | 0x800000
		shift := uint(14 - e)
		half := m >> shift
		rem := m & ((1 << shift) - 1)
		mid := uint32(1) << (shift - 1)
		if rem > mid || (rem == mid && half&1 == 1) {
			half++
		}
		return sign | uint16(half)
	}
	half := uint32(e)<<10 | mant>>13
	rem := mant & 0x1fff
	if rem > 0x1000 || (rem == 0x1000 && half&1 == 1) {
		half++
	}
	return sign | uint16(half)
}
