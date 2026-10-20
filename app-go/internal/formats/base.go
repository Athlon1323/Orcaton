package formats

import (
	"fmt"
	"os"
	"strings"

	"orcaton/internal/core"
)

type FormatError struct{ Msg string }

func (e *FormatError) Error() string { return e.Msg }

func Errorf(format string, a ...any) error { return &FormatError{fmt.Sprintf(format, a...)} }

type Importer interface {
	Name() string
	Title() string
	Extensions() []string
	Sniff(path string) (bool, error)
	Read(path string) (*core.Project, error)
}

type Exporter interface {
	Name() string
	Title() string
	Extensions() []string
	Write(p *core.Project, path string) error
}

var (
	importers []Importer
	exporters []Exporter
)

func RegisterImporter(i Importer) { importers = append(importers, i) }
func RegisterExporter(e Exporter) { exporters = append(exporters, e) }

func Importers() []Importer { return append([]Importer(nil), importers...) }
func Exporters() []Exporter { return append([]Exporter(nil), exporters...) }

func DetectImporter(path, fmtName string) (Importer, error) {
	if fmtName != "" {
		for _, i := range importers {
			if i.Name() == fmtName {
				return i, nil
			}
		}
		return nil, Errorf("Неизвестный формат импорта: %s", fmtName)
	}
	for _, i := range importers {
		ok, err := i.Sniff(path)
		if err != nil {
			continue
		}
		if ok {
			return i, nil
		}
	}
	ext := strings.ToLower(Suffix(path))
	for _, i := range importers {
		for _, e := range i.Extensions() {
			if e == ext {
				return i, nil
			}
		}
	}
	return nil, Errorf("Не удалось определить формат файла: %s", path)
}

func ExporterFor(path, fmtName string) (Exporter, error) {
	if fmtName != "" {
		for _, e := range exporters {
			if e.Name() == fmtName {
				return e, nil
			}
		}
		return nil, Errorf("Неизвестный формат экспорта: %s", fmtName)
	}
	ext := strings.ToLower(Suffix(path))
	for _, e := range exporters {
		for _, x := range e.Extensions() {
			if x == ext {
				return e, nil
			}
		}
	}
	return nil, Errorf("Нет экспортёра для расширения '%s'", ext)
}

func ReadProject(path, fmtName string, pathMap *string) (*core.Project, error) {
	imp, err := DetectImporter(path, fmtName)
	if err != nil {
		return nil, err
	}
	p, err := imp.Read(path)
	if err != nil {
		return nil, err
	}
	if len(p.Sequences) == 0 {
		return nil, Errorf("%s: в проекте нет ни одной секвенции (таймлайна)", path)
	}
	spec := os.Getenv("ORCATON_PATH_MAP")
	if pathMap != nil {
		spec = *pathMap
	}
	if rules := ParsePathMap(spec); len(rules) > 0 {
		for _, m := range p.MediaList() {
			if m.Path != nil && *m.Path != "" {
				if m.OriginalPath == nil {
					orig := *m.Path
					m.OriginalPath = &orig
				}
				np := RemapPath(*m.Path, rules)
				m.Path = &np
			}
		}
	}
	return p, nil
}

func WriteProject(p *core.Project, path, fmtName string) error {
	e, err := ExporterFor(path, fmtName)
	if err != nil {
		return err
	}
	return e.Write(p, path)
}

func RestoreOriginalPaths(p *core.Project) {
	for _, m := range p.MediaList() {
		if m.OriginalPath != nil {
			orig := *m.OriginalPath
			m.Path = &orig
			m.OriginalPath = nil
		}
	}
}
