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
		help: "Кратко: одна строка (экономно для логов и ИИ)"}
	jsonOpt = opt{names: []string{"--json"}, kind: kBool, def: false,
		help: "Машиночитаемый вывод (JSON)"}
	fmtOpt = opt{names: []string{"--format", "-f"}, kind: kStr, def: nil, help: "Явно указать формат"}
	mapOpt = opt{names: []string{"--map"}, kind: kStr, def: nil,
		help: "Где медиа на этом компьютере: папка с исходниками (поиск по имени) и/или подмена " +
			"префикса 'D:/=/Volumes/Media/'; несколько — через ';' (по умолчанию $ORCATON_PATH_MAP)"}
)

func root() *command {
	r := &command{name: prog, help: "Orcaton — синхронизация видео/аудио по звуку (MVP, Go).",
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
	add(&command{name: "version", help: "Версия.", run: cmdVersion})
	add(&command{name: "formats", help: "Список поддерживаемых форматов.", run: cmdFormats})
	add(&command{name: "inspect", help: "Сводка по проекту: секвенции, дорожки, клипы, FPS, метки, разрывы.",
		args: []arg{{name: "path", file: true}}, opts: []opt{fmtOpt}, run: cmdInspect})
	add(&command{name: "clips", help: "Список клипов (позиции в секундах и TC).",
		args: []arg{{name: "path", file: true}},
		opts: []opt{{names: []string{"--track", "-t"}, kind: kStr, help: "Напр. V1 или A3"},
			{names: []string{"--csv"}, kind: kStr, help: "Сохранить в CSV"}, fmtOpt}, run: cmdClips})
	add(&command{name: "convert", help: "Конвертировать проект (через доменную модель); --map переносит пути на другую ОС.",
		args: []arg{{name: "src", file: true}, {name: "dst"}},
		opts: []opt{fmtOpt, {names: []string{"--to"}, kind: kStr, help: "Формат выхода"}, mapOpt},
		run:  cmdConvert})
	add(&command{name: "probe", help: "ffprobe медиафайла: потоки, fps, таймкод.",
		args: []arg{{name: "media", file: true}}, run: cmdProbe})
	add(&command{name: "offset", help: "Смещение звука файла B относительно A (кросс-корреляция).",
		args: []arg{{name: "a", file: true}, {name: "b", file: true}},
		opts: []opt{{names: []string{"--sr"}, kind: kInt, def: 8000, min: bound(1000), max: bound(192000), help: "Частота анализа, Гц"},
			{names: []string{"--max-offset"}, kind: kFloat, min: bound(0.001), help: "Окно поиска ±сек"}}, run: cmdOffset})
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
			fmt.Fprintf(stderr, "Внутренняя ошибка Orcaton: %v\nЭто ошибка программы, а не ваших данных — "+
				"пришлите, пожалуйста, команду и текст ниже.\n\n%s", rec, debug.Stack())
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
		c.printf("Ошибка: %v\n", err)
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
