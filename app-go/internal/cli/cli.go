package cli

import (
	"errors"
	"fmt"
	"io"
	"runtime/debug"
	"strings"

	"orcaton/internal/app"
	"orcaton/internal/core"
	"orcaton/internal/formats"
	_ "orcaton/internal/formats/all"
	"orcaton/internal/pyio"
)

var devCommands []devCommand

type devCommand struct {
	after string
	mk    func() *command
}

const Version = app.Version

const prog = "orcaton"

var (
	briefOpt = opt{names: []string{"--brief"}, kind: kBool, def: false,
		help: "Brief: one line (compact for logs and AI)"}
	jsonOpt = opt{names: []string{"--json"}, kind: kBool, def: false,
		help: "Machine-readable output (JSON)"}
	fmtOpt = opt{names: []string{"--format", "-f"}, kind: kStr, def: nil, help: "Set the format explicitly"}
	mapOpt = opt{names: []string{"--map"}, kind: kStr, def: nil,
		help: "Where the media are on this computer: a folder with sources (search by name) and/or a " +
			"prefix replacement 'D:/=/Volumes/Media/'; several — separated by ';' (default $ORCATON_PATH_MAP)"}
)

func root() *command {
	r := &command{name: prog, help: "Orcaton — video/audio synchronization by sound (MVP, Go).",
		group: map[string]*command{}}
	var add func(c *command)
	add = func(c *command) {
		r.group[c.name] = c
		r.order = append(r.order, c.name)
		for _, d := range devCommands {
			if d.after == c.name {
				add(d.mk())
			}
		}
	}
	add(&command{name: "version", help: "Version.", run: cmdVersion})
	add(&command{name: "formats", help: "List of supported formats.", run: cmdFormats})
	add(&command{name: "inspect", help: "Project summary: sequences, tracks, clips, FPS, labels, gaps.",
		args: []arg{{name: "path", file: true}}, opts: []opt{fmtOpt}, run: cmdInspect})
	add(&command{name: "clips", help: "List of clips (positions in seconds and TC).",
		args: []arg{{name: "path", file: true}},
		opts: []opt{{names: []string{"--track", "-t"}, kind: kStr, help: "E.g. V1 or A3"},
			{names: []string{"--csv"}, kind: kStr, help: "Save to CSV"}, fmtOpt}, run: cmdClips})
	add(&command{name: "convert", help: "Convert a project (via the domain model); --map moves paths to another OS.",
		args: []arg{{name: "src", file: true}, {name: "dst"}},
		opts: []opt{fmtOpt, {names: []string{"--to"}, kind: kStr, help: "Output format"}, mapOpt},
		run:  cmdConvert})
	add(&command{name: "probe", help: "ffprobe of a media file: streams, fps, timecode.",
		args: []arg{{name: "media", file: true}}, run: cmdProbe})
	add(&command{name: "offset", help: "Audio offset of file B relative to A (cross-correlation).",
		args: []arg{{name: "a", file: true}, {name: "b", file: true}},
		opts: []opt{{names: []string{"--sr"}, kind: kInt, def: 8000, min: bound(1000), max: bound(192000), help: "Analysis sample rate, Hz"},
			{names: []string{"--max-offset"}, kind: kFloat, min: bound(0.001), help: "Search window ±s"}}, run: cmdOffset})
	add(syncCommand())
	add(configCommand())
	add(cacheGroup())
	return r
}

func Run(argv []string, stdout, stderr io.Writer) int {
	r := root()
	cur := r
	path := prog
	i := 0
	for cur.group != nil {
		if cur.deflt != "" && (i >= len(argv) || strings.HasPrefix(argv[i], "-") &&
			argv[i] != "--help" && argv[i] != "-h") {
			path += " " + cur.deflt
			cur = cur.group[cur.deflt]
			break
		}
		if i >= len(argv) || argv[i] == "--help" || argv[i] == "-h" {
			printHelp(stdout, path, cur)
			if i >= len(argv) {
				return 2
			}
			return 0
		}
		sub, ok := cur.group[argv[i]]
		if !ok {
			fmt.Fprintln(stderr, usageLine(path, cur)+" COMMAND [ARGS]...")
			fmt.Fprintf(stderr, "Error: No such command '%s'.\n", argv[i])
			return 2
		}
		path += " " + argv[i]
		cur = sub
		i++
	}
	for _, a := range argv[i:] {
		if a == "--help" || a == "-h" {
			printHelp(stdout, path, cur)
			return 0
		}
	}
	c, err := parse(cur, argv[i:])
	if err != nil {
		fmt.Fprintln(stderr, usageLine(path, cur))
		fmt.Fprintf(stderr, "Try '%s --help' for help.\n\nError: %v\n", path, err)
		return 2
	}
	c.out, c.err = stdout, stderr
	return runSafe(cur, c, stderr)
}

func runSafe(cur *command, c *ctx, stderr io.Writer) (code int) {
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Fprintf(stderr, "Orcaton internal error: %v\nThis is a bug in the program, not in your data — "+
				"please send the command and the text below.\n\n%s", rec, debug.Stack())
			code = 70
		}
	}()
	return cur.run(c)
}

func (c *ctx) printf(format string, a ...any) { fmt.Fprintf(c.out, format, a...) }
func (c *ctx) println(a ...any)               { fmt.Fprintln(c.out, a...) }

func (c *ctx) emitJSON(v any) { fmt.Fprintln(c.out, pyio.Dumps(v, -1)) }

func (c *ctx) load(path, fmtName string, pathMap *string) (*core.Project, int) {
	p, err := formats.ReadProject(path, fmtName, pathMap)
	if err != nil {
		c.printf("Error: %v\n", err)
		var fe *formats.FormatError
		if errors.As(err, &fe) {
			return nil, 2
		}
		return nil, 1
	}
	return p, 0
}

func (c *ctx) loadOpt(path string) (*core.Project, int) {
	if path == "" {
		return nil, 0
	}
	return c.load(path, "", nil)
}
