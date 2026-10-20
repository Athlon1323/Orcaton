package xmldom

import "strings"

type W struct {
	Tag      string
	Attrs    []Attr
	Text     *string
	Children []*W
}

func NewW(tag string, attrs ...Attr) *W { return &W{Tag: tag, Attrs: attrs} }

func (w *W) Sub(tag string, attrs ...Attr) *W {
	c := &W{Tag: tag, Attrs: attrs}
	w.Children = append(w.Children, c)
	return c
}

func (w *W) SubText(tag, text string) *W {
	c := w.Sub(tag)
	c.Text = &text
	return c
}

func (w *W) SetText(s string) *W {
	w.Text = &s
	return w
}

func Serialize(root *W, doctype string) []byte {
	var b strings.Builder
	b.WriteString("<?xml version='1.0' encoding='UTF-8'?>\n")
	if doctype != "" {
		b.WriteString(doctype)
		b.WriteString("\n")
	}
	write(&b, root, 0, "  ")
	return []byte(b.String())
}

func SerializeIndented(root *W, unit string) []byte {
	var b strings.Builder
	b.WriteString("<?xml version='1.0' encoding='UTF-8'?>\n")
	write(&b, root, 0, unit)
	s := strings.TrimSuffix(b.String(), "\n")
	return []byte(s)
}

func write(b *strings.Builder, n *W, level int, unit string) {
	ind := strings.Repeat(unit, level)
	b.WriteString(ind)
	b.WriteString("<")
	b.WriteString(n.Tag)
	for _, a := range n.Attrs {
		b.WriteString(" ")
		b.WriteString(a.Name)
		b.WriteString(`="`)
		b.WriteString(EscapeAttr(a.Value))
		b.WriteString(`"`)
	}
	switch {
	case len(n.Children) == 0 && n.Text == nil:
		b.WriteString("/>\n")
	case len(n.Children) == 0:
		b.WriteString(">")
		b.WriteString(EscapeText(*n.Text))
		b.WriteString("</" + n.Tag + ">\n")
	default:
		b.WriteString(">")
		if n.Text != nil {
			b.WriteString(EscapeText(*n.Text))
		}
		b.WriteString("\n")
		for _, c := range n.Children {
			write(b, c, level+1, unit)
		}
		b.WriteString(ind + "</" + n.Tag + ">\n")
	}
}

var textEsc = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\r", "&#13;")
var attrEsc = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;",
	"\n", "&#10;", "\r", "&#13;", "\t", "&#9;")

func EscapeText(s string) string { return textEsc.Replace(s) }

func EscapeAttr(s string) string { return attrEsc.Replace(s) }
