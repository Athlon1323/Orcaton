package xmldom

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"
)

type Attr struct{ Name, Value string }

type Node struct {
	Tag      string
	Attrs    []Attr
	Text     *string
	Children []*Node
	Parent   *Node
	nComment int
}

func (n *Node) Get(name string) (string, bool) {
	for _, a := range n.Attrs {
		if a.Name == name {
			return a.Value, true
		}
	}
	return "", false
}

func (n *Node) Attr(name string) string {
	v, _ := n.Get(name)
	return v
}

func (n *Node) Len() int { return len(n.Children) + n.nComment }

func (n *Node) Find(path string) *Node {
	if n == nil {
		return nil
	}
	all := n.FindAll(path)
	if len(all) == 0 {
		return nil
	}
	return all[0]
}

func (n *Node) FindAll(path string) []*Node {
	if n == nil {
		return nil
	}
	cur := []*Node{n}
	for _, step := range strings.Split(path, "/") {
		var next []*Node
		for _, c := range cur {
			for _, ch := range c.Children {
				if step == "*" || ch.Tag == step {
					next = append(next, ch)
				}
			}
		}
		cur = next
		if len(cur) == 0 {
			return nil
		}
	}
	return cur
}

func (n *Node) FindText(path string) (string, bool) {
	el := n.Find(path)
	if el == nil {
		return "", false
	}
	if el.Text == nil {
		return "", true
	}
	return *el.Text, true
}

func (n *Node) Iter(tags ...string) []*Node {
	var out []*Node
	var walk func(x *Node)
	walk = func(x *Node) {
		if len(tags) == 0 || contains(tags, x.Tag) {
			out = append(out, x)
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	if n != nil {
		walk(n)
	}
	return out
}

func (n *Node) Ancestors() []*Node {
	var out []*Node
	for p := n.Parent; p != nil; p = p.Parent {
		out = append(out, p)
	}
	return out
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

var ErrNoRoot = errors.New("no root element")

func Parse(data []byte, strict bool) (*Node, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	d.Strict = strict
	if !strict {
		d.AutoClose = xml.HTMLAutoClose
		d.Entity = xml.HTMLEntity
	}
	d.CharsetReader = func(_ string, in io.Reader) (io.Reader, error) { return in, nil }
	var root *Node
	var stack []*Node
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			if !strict && root != nil {
				break
			}
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			nd := &Node{Tag: name(t.Name)}
			for _, a := range t.Attr {
				nd.Attrs = append(nd.Attrs, Attr{name(a.Name), a.Value})
			}
			if len(stack) > 0 {
				p := stack[len(stack)-1]
				nd.Parent = p
				p.Children = append(p.Children, nd)
			} else if root == nil {
				root = nd
			}
			stack = append(stack, nd)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) == 0 {
				continue
			}
			p := stack[len(stack)-1]
			if len(p.Children) == 0 && p.nComment == 0 {
				s := string(t)
				if p.Text == nil {
					p.Text = &s
				} else {
					v := *p.Text + s
					p.Text = &v
				}
			}
		case xml.Comment, xml.ProcInst:
			if len(stack) > 0 {
				stack[len(stack)-1].nComment++
			}
		}
	}
	if root == nil {
		return nil, ErrNoRoot
	}
	return root, nil
}

func name(n xml.Name) string {
	if n.Space != "" {
		return n.Space + ":" + n.Local
	}
	return n.Local
}
