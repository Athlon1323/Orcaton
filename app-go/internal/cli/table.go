package cli

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

type table struct {
	head []string
	rows [][]string
}

func newTable(head ...string) *table { return &table{head: head} }

func (t *table) add(row ...string) { t.rows = append(t.rows, row) }

func (t *table) print(w io.Writer) {
	width := make([]int, len(t.head))
	for i, h := range t.head {
		width[i] = utf8.RuneCountInString(h)
	}
	for _, r := range t.rows {
		for i, c := range r {
			if i < len(width) {
				width[i] = max(width[i], utf8.RuneCountInString(c))
			}
		}
	}
	line := func(cells []string) {
		var b strings.Builder
		for i := range width {
			c := ""
			if i < len(cells) {
				c = cells[i]
			}
			if i > 0 {
				b.WriteString(" │ ")
			}
			b.WriteString(c)
			b.WriteString(strings.Repeat(" ", width[i]-utf8.RuneCountInString(c)))
		}
		fmt.Fprintln(w, strings.TrimRight(b.String(), " "))
	}
	line(t.head)
	var sep []string
	for _, n := range width {
		sep = append(sep, strings.Repeat("─", n))
	}
	fmt.Fprintln(w, strings.Join(sep, "─┼─"))
	for _, r := range t.rows {
		line(r)
	}
}
