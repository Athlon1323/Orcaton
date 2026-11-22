package app

import (
	"context"
	"errors"
	"path/filepath"

	"orcaton/internal/core"
	"orcaton/internal/engine"
	"orcaton/internal/formats"
	_ "orcaton/internal/formats/all"
	"orcaton/internal/media"
)

type SyncRequest struct {
	Options   engine.Options
	Features  []string
	Originals []string
	Times     []string
}

type Run struct {
	Project *core.Project
	Report  *engine.Report
}

func Sync(ctx context.Context, p *core.Project, req SyncRequest, progress engine.Progress) (*Run, error) {
	if err := req.Options.Validate(); err != nil {
		return nil, err
	}
	var bundle media.FeatureSource
	if len(req.Features) > 0 {
		b, err := media.LoadBundle(req.Features)
		if err != nil {
			return nil, err
		}
		bundle = b
	}
	var rtimes *media.RecordTimes
	if req.Options.RecordTime {
		var err error
		if rtimes, err = media.NewRecordTimes(req.Originals, req.Times, true); err != nil {
			return nil, err
		}
		rtimes.NoMediaFiles = bundle != nil
	}
	eng := engine.New(req.Options, progress, bundle, rtimes)
	out, err := eng.Sync(ctx, p, 0)
	if err != nil {
		return nil, err
	}
	return &Run{Project: out, Report: eng.Report}, nil
}

func CheckOutput(p *core.Project, dst, to, report string) (format, reportPath string, err error) {
	exp, err := formats.ExporterFor(dst, to)
	if err != nil {
		return "", "", wrap(KindBadOutput, err)
	}
	if exp.Name() == "drt" && p.SourceFormat != "drt" {
		return "", "", wrap(KindBadOutput, errors.New(".drt output is written by patching the source .drt — "+
			"the input must be .drt too. For XML input choose .xml output"))
	}
	if exp.Name() == "drt" && report == "" {
		report = filepath.Join(filepath.Dir(dst), formats.Stem(dst)+"_report.csv")
	}
	return exp.Name(), report, nil
}

func Save(r *Run, dst, to string, keepPaths bool, progress engine.Progress) error {
	if _, _, err := CheckOutput(r.Project, dst, to, ""); err != nil {
		return err
	}
	out := r.Project
	if keepPaths {
		out = out.Clone()
		formats.RestoreOriginalPaths(out)
	}
	emit(progress, engine.StageWrite, 0, 1, formats.BaseName(dst))
	if err := formats.WriteProject(out, dst, to); err != nil {
		return wrap(KindWrite, err)
	}
	emit(progress, engine.StageWrite, 1, 1, formats.BaseName(dst))
	return nil
}
