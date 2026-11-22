package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"orcaton/internal/num"
	"orcaton/internal/settings"
)

var (
	configOpt = opt{names: []string{"--config"}, kind: kStr,
		help: "Settings file instead of orcaton.toml next to the project (the user file is read anyway)"}
	noConfigOpt = opt{names: []string{"--no-config"}, kind: kBool, def: false,
		help: "Do not read orcaton.toml: only flags and defaults"}
)

func (c *ctx) optOf(key string) (opt, bool) {
	for _, o := range c.cmd.opts {
		if c.key(o) == key {
			return o, true
		}
	}
	return opt{}, false
}

func applySettings(c *ctx, project string) ([]string, int) {
	if c.Bool("no-config") {
		return nil, 0
	}
	s, err := settings.Load(project, c.Str("config"))
	if err != nil {
		c.printf("Settings error: %v\n", err)
		return nil, 2
	}
	for _, name := range s.Names() {
		o, ok := c.optOf(name)
		if !ok || c.set[name] {
			continue
		}
		if name == "map" && os.Getenv("ORCATON_PATH_MAP") != "" {
			continue
		}
		v, l := s.Get(name)
		val, err := optValue(o, v)
		if err != nil {
			c.printf("Settings error: %v\n", &settings.Error{Path: l.Path, Key: name, Msg: err.Error()})
			return nil, 2
		}
		if val != nil {
			c.vals[name] = val
		}
	}
	return s.Files(), 0
}

func optValue(o opt, v any) (any, error) {
	switch x := v.(type) {
	case int:
		if err := o.checkRange(o.names[0], strconv.Itoa(x), float64(x)); err != nil {
			return nil, err
		}
	case float64:
		if x == 0 && o.names[0] == "--max-offset" {
			return nil, nil
		}
		if err := o.checkRange(o.names[0], num.Repr(x), x); err != nil {
			return nil, err
		}
	}
	return v, nil
}

func configCommand() *command {
	return &command{name: "config",
		help: "Settings file orcaton.toml: where it is and which sync option values are in effect.\n\n" +
			"  Defaults for sync: first the user file, on top of it — orcaton.toml next to the\n" +
			"  timeline (or --config); a command-line flag wins over the file. --init creates a template.",
		args: []arg{{name: "project", file: true, optional: true}},
		opts: []opt{configOpt,
			{names: []string{"--init"}, kind: kBool, def: false,
				help: "Create a template: --config FILE, otherwise next to PROJECT, otherwise the user file"}},
		run: cmdConfig}
}

func cmdConfig(c *ctx) int {
	project, explicit := c.Arg("project"), c.Str("config")
	if c.Bool("init") {
		return configInit(c, project, explicit)
	}
	user := settings.UserPath()
	state := func(p string) string {
		if _, err := os.Stat(p); err == nil {
			return p + " (exists)"
		}
		return p + " (missing)"
	}
	if user == "" {
		c.printf("User file:    not read (%s=%s)\n", settings.EnvVar, os.Getenv(settings.EnvVar))
	} else {
		c.printf("User file:    %s\n", state(user))
	}
	if p := explicit; p != "" || project != "" {
		if p == "" {
			p = settings.ProjectPath(project)
		}
		c.printf("Project file: %s\n", state(p))
	}
	s, err := settings.Load(project, explicit)
	if err != nil {
		c.printf("Settings error: %v\n", err)
		return 2
	}
	names := s.Names()
	if len(names) == 0 {
		c.println("Values: none set — sync uses its defaults.")
		return 0
	}
	c.printf("Values [%s] (a command-line flag wins):\n", settings.Section)
	sync := syncCommand()
	var bad []error
	for _, n := range names {
		v, l := s.Get(n)
		k, _ := settings.Lookup(n)
		f, _ := settings.Format(k, v)
		from := "user"
		if l.Path != user {
			from = "project"
		}
		c.printf("  %s = %s   ← %s\n", n, f, from)
		for _, o := range sync.opts {
			if o.names[0] == "--"+n {
				if _, err := optValue(o, v); err != nil {
					bad = append(bad, &settings.Error{Path: l.Path, Key: n, Msg: err.Error()})
				}
			}
		}
	}
	for _, err := range bad {
		c.printf("Settings error: %v\n", err)
	}
	if len(bad) > 0 {
		return 2
	}
	return 0
}

func configInit(c *ctx, project, explicit string) int {
	path := explicit
	switch {
	case path != "":
	case project != "":
		path = settings.ProjectPath(project)
	default:
		path = settings.UserPath()
	}
	if path == "" {
		c.printf("The user file is disabled (%s) — pass --config FILE\n", settings.EnvVar)
		return 2
	}
	if _, err := os.Stat(path); err == nil {
		c.printf("Already exists: %s — not overwriting\n", path)
		return 1
	}
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err == nil {
		err = os.WriteFile(path, []byte(configTemplate()), 0o644)
	}
	if err != nil {
		c.printf("Error: %v\n", err)
		return 1
	}
	c.printf("Created: %s\n", path)
	return 0
}

func configTemplate() string {
	var b strings.Builder
	b.WriteString(`# orcaton.toml — default values of "orcaton sync" options (docs/go/usage.md, "Settings file").
# The user file applies to all projects, orcaton.toml next to the timeline goes on top of it,
# a command-line flag wins over both. The key is the flag name without "--". To set a value,
# remove the "# " at the start of the line. Windows paths go in single quotes: 'D:\media\set_a'. Relative
# paths (search, originals, cache, folders in map) are relative to this file's folder.

[sync]
`)
	sync := syncCommand()
	for _, k := range settings.Keys {
		for _, o := range sync.opts {
			if strings.TrimLeft(o.names[0], "-") != k.Name {
				continue
			}
			help := strings.TrimSuffix(o.help, "; can be repeated")
			if k.Name == "max-offset" {
				help += "; 0 — no limit"
			}
			fmt.Fprintf(&b, "%-30s # %s\n", "# "+k.Name+" = "+templateValue(k, o), help)
		}
	}
	return b.String()
}

func templateValue(k settings.Key, o opt) string {
	v := o.def
	switch k.Kind {
	case settings.List:
		v = []string{}
	case settings.String:
		if v == nil {
			v = ""
		}
	case settings.Float:
		if v == nil {
			v = 0.0
		}
	}
	s, err := settings.Format(k, v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return s
}
