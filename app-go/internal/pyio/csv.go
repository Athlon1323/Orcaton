package pyio

import (
	"bufio"
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const BOM = "\ufeff"

type CSVWriter struct {
	w     *bufio.Writer
	delim byte
}

func NewCSV(w io.Writer, delim byte) *CSVWriter {
	return &CSVWriter{w: bufio.NewWriter(w), delim: delim}
}

func (c *CSVWriter) Write(row []string) error {
	for i, f := range row {
		if i > 0 {
			c.w.WriteByte(c.delim)
		}
		need := strings.ContainsAny(f, string([]byte{c.delim, '"', '\r', '\n'}))
		if len(row) == 1 && f == "" {
			need = true
		}
		if need {
			c.w.WriteByte('"')
			c.w.WriteString(strings.ReplaceAll(f, `"`, `""`))
			c.w.WriteByte('"')
		} else {
			c.w.WriteString(f)
		}
	}
	_, err := c.w.WriteString("\r\n")
	return err
}

func (c *CSVWriter) Flush() error { return c.w.Flush() }

func WriteCSVFile(path string, head []string, rows [][]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "" {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(BOM); err != nil {
		return err
	}
	w := NewCSV(f, ';')
	if head != nil {
		if err := w.Write(head); err != nil {
			return err
		}
	}
	for _, r := range rows {
		if err := w.Write(r); err != nil {
			return err
		}
	}
	return w.Flush()
}

func ReadCSVDicts(path string) ([]map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s := strings.TrimPrefix(string(data), BOM)
	r := csv.NewReader(strings.NewReader(s))
	r.Comma = ';'
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	recs, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, nil
	}
	head := recs[0]
	var out []map[string]string
	for _, rec := range recs[1:] {
		if len(rec) == 0 || (len(rec) == 1 && rec[0] == "") {
			continue
		}
		m := map[string]string{}
		for i, h := range head {
			if i < len(rec) {
				m[h] = rec[i]
			}
		}
		out = append(out, m)
	}
	return out, nil
}
