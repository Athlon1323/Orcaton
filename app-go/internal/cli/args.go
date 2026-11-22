package cli

import (
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

type optKind int

const (
	kBool optKind = iota
	kStr
	kFloat
	kInt
	kList
)

type opt struct {
	names []string
	neg   string
	kind  optKind
	def   any
	help  string
	meta  string
	min   *float64
	max   *float64
}

func bound(v float64) *float64 { return &v }

func (o opt) rangeText() string {
	g := func(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }
	switch {
	case o.min != nil && o.max != nil:
		return g(*o.min) + "<=x<=" + g(*o.max)
	case o.min != nil:
		return "x>=" + g(*o.min)
	case o.max != nil:
		return "x<=" + g(*o.max)
	}
	return ""
}

func (o opt) checkRange(name, val string, v float64) error {
	if (o.min != nil && v < *o.min) || (o.max != nil && v > *o.max) {
		return &usageError{fmt.Sprintf("Invalid value for '%s': %s is not in the range %s.", name, val, o.rangeText())}
	}
	return nil
}

type arg struct {
	name     string
	help     string
	file     bool
	optional bool
}

type command struct {
	name  string
	help  string
	args  []arg
	opts  []opt
	run   func(c *ctx) int
	group map[string]*command
	order []string
	deflt string
}

type ctx struct {
	cmd  *command
	pos  map[string]string
	vals map[string]any
	set  map[string]bool
	out  io.Writer
	err  io.Writer
}

func (c *ctx) key(o opt) string { return strings.TrimLeft(o.names[0], "-") }

func (c *ctx) Bool(name string) bool     { v, _ := c.vals[name].(bool); return v }
func (c *ctx) Str(name string) string    { v, _ := c.vals[name].(string); return v }
func (c *ctx) Float(name string) float64 { v, _ := c.vals[name].(float64); return v }
func (c *ctx) Int(name string) int       { v, _ := c.vals[name].(int); return v }
func (c *ctx) List(name string) []string {
	v, _ := c.vals[name].([]string)
	return v
}

func (c *ctx) Has(name string) bool { return c.vals[name] != nil }

func (c *ctx) StrPtr(name string) *string {
	if v, ok := c.vals[name].(string); ok {
		return &v
	}
	return nil
}

func (c *ctx) Arg(name string) string { return c.pos[name] }

type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

func parse(cmd *command, argv []string) (*ctx, error) {
	c := &ctx{cmd: cmd, pos: map[string]string{}, vals: map[string]any{}, set: map[string]bool{}}
	for _, o := range cmd.opts {
		c.vals[c.key(o)] = o.def
	}
	find := func(name string) (*opt, bool, bool) {
		for i := range cmd.opts {
			o := &cmd.opts[i]
			for _, n := range o.names {
				if n == name {
					return o, false, true
				}
			}
			if o.neg != "" && o.neg == name {
				return o, true, true
			}
		}
		return nil, false, false
	}
	var positional []string
	onlyPos := false
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		if onlyPos || !strings.HasPrefix(a, "-") || a == "-" || isNumber(a) {
			positional = append(positional, a)
			continue
		}
		if a == "--" {
			onlyPos = true
			continue
		}
		name, val, hasVal := strings.Cut(a, "=")
		o, neg, ok := find(name)
		if !ok {
			return nil, &usageError{fmt.Sprintf("No such option: %s", name)}
		}
		k := c.key(*o)
		c.set[k] = true
		if o.kind == kBool {
			if hasVal {
				return nil, &usageError{fmt.Sprintf("Option '%s' does not take a value.", name)}
			}
			c.vals[k] = !neg
			continue
		}
		if !hasVal {
			if i+1 >= len(argv) {
				return nil, &usageError{fmt.Sprintf("Option '%s' requires an argument.", name)}
			}
			i++
			val = argv[i]
		}
		switch o.kind {
		case kStr:
			c.vals[k] = val
		case kFloat:
			f, err := strconv.ParseFloat(val, 64)
			if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
				return nil, &usageError{fmt.Sprintf("Invalid value for '%s': '%s' is not a valid float.", name, val)}
			}
			if err := o.checkRange(name, val, f); err != nil {
				return nil, err
			}
			c.vals[k] = f
		case kInt:
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, &usageError{fmt.Sprintf("Invalid value for '%s': '%s' is not a valid integer.", name, val)}
			}
			if err := o.checkRange(name, val, float64(n)); err != nil {
				return nil, err
			}
			c.vals[k] = n
		case kList:
			l, _ := c.vals[k].([]string)
			c.vals[k] = append(append([]string(nil), l...), val)
		}
	}
	need := 0
	for _, a := range cmd.args {
		if !a.optional {
			need++
		}
	}
	if len(positional) < need {
		missing := cmd.args[len(positional)].name
		return nil, &usageError{fmt.Sprintf("Missing argument '%s'.", strings.ToUpper(missing))}
	}
	if len(positional) > len(cmd.args) {
		return nil, &usageError{fmt.Sprintf("Got unexpected extra argument (%s)", positional[len(cmd.args)])}
	}
	for i, p := range positional {
		a := cmd.args[i]
		if a.file {
			st, err := os.Stat(p)
			if err != nil {
				return nil, &usageError{fmt.Sprintf("Invalid value for '%s': File '%s' does not exist.",
					strings.ToUpper(a.name), p)}
			}
			if st.IsDir() {
				return nil, &usageError{fmt.Sprintf("Invalid value for '%s': File '%s' is a directory.",
					strings.ToUpper(a.name), p)}
			}
		}
		c.pos[a.name] = p
	}
	return c, nil
}

func isNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func usageLine(prog string, cmd *command) string {
	var b strings.Builder
	b.WriteString("Usage: " + prog)
	if len(cmd.opts) > 0 {
		b.WriteString(" [OPTIONS]")
	}
	for _, a := range cmd.args {
		if a.optional {
			b.WriteString(" [" + strings.ToUpper(a.name) + "]")
		} else {
			b.WriteString(" " + strings.ToUpper(a.name))
		}
	}
	return b.String()
}

func printHelp(w io.Writer, prog string, cmd *command) {
	if cmd.group != nil {
		fmt.Fprintf(w, "Usage: %s [OPTIONS] COMMAND [ARGS]...\n\n  %s\n\nCommands:\n", prog, cmd.help)
		names := cmd.order
		if names == nil {
			for n := range cmd.group {
				names = append(names, n)
			}
			sort.Strings(names)
		}
		for _, n := range names {
			mark := ""
			if n == cmd.deflt {
				mark = " (default)"
			}
			fmt.Fprintf(w, "  %-18s %s%s\n", n, firstLine(cmd.group[n].help), mark)
		}
		return
	}
	fmt.Fprintln(w, usageLine(prog, cmd))
	fmt.Fprintf(w, "\n  %s\n", cmd.help)
	if len(cmd.args) > 0 {
		fmt.Fprintln(w, "\nArguments:")
		for _, a := range cmd.args {
			fmt.Fprintf(w, "  %-28s %s\n", strings.ToUpper(a.name), a.help)
		}
	}
	fmt.Fprintln(w, "\nOptions:")
	for _, o := range cmd.opts {
		n := strings.Join(o.names, ", ")
		if o.neg != "" {
			n += " / " + o.neg
		}
		switch o.kind {
		case kStr, kList:
			n += " TEXT"
		case kFloat:
			n += " FLOAT"
		case kInt:
			n += " INTEGER"
		}
		if r := o.rangeText(); r != "" {
			n += " [" + r + "]"
		}
		def := ""
		if o.def != nil && o.kind != kList {
			if b, ok := o.def.(bool); !ok || b {
				def = fmt.Sprintf(" [default: %v]", o.def)
			}
		}
		fmt.Fprintf(w, "  %-28s %s%s\n", n, o.help, def)
	}
	fmt.Fprintf(w, "  %-28s %s\n", "--help", "Show this help")
}

func firstLine(s string) string {
	l, _, _ := strings.Cut(s, "\n")
	return l
}
