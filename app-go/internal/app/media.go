package app

import (
	"context"
	"os"

	"orcaton/internal/core"
	"orcaton/internal/engine"
	"orcaton/internal/formats"
	"orcaton/internal/media"
)

type Action string

const (
	AnswerFolder Action = "folder"
	AnswerSkip   Action = "skip"
	AnswerCancel Action = "cancel"
)

type Answer struct {
	Action Action
	Folder string
}

type Asker interface {
	AskMissing(ctx context.Context, missing []*core.MediaRef) (Answer, error)
}

type AskFunc func(ctx context.Context, missing []*core.MediaRef) (Answer, error)

func (f AskFunc) AskMissing(ctx context.Context, missing []*core.MediaRef) (Answer, error) {
	return f(ctx, missing)
}

type Ambiguity struct {
	Media      *core.MediaRef
	Candidates []string
}

type SearchReport struct {
	Answer    bool
	Roots     []string
	BadRoots  []string
	Before    int
	Found     int
	Ambiguous []Ambiguity
	NotFound  []*core.MediaRef
}

type FindRequest struct {
	Roots    []string
	Depth    int
	Ask      Asker
	OnSearch func(SearchReport)
	Progress engine.Progress
}

func FindMedia(ctx context.Context, p *core.Project, req FindRequest) ([]*core.MediaRef, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	missing := media.MissingMedia(p)
	if len(missing) == 0 {
		return nil, nil
	}
	if len(req.Roots) > 0 {
		missing = search(p, req, req.Roots, false)
	}
	if req.Ask == nil {
		return missing, nil
	}
	for len(missing) > 0 {
		if err := ctx.Err(); err != nil {
			return missing, err
		}
		ans, err := req.Ask.AskMissing(ctx, missing)
		if err != nil {
			return missing, err
		}
		switch ans.Action {
		case AnswerSkip:
			return missing, nil
		case AnswerCancel:
			return missing, wrap(KindCancelled, errUserCancel)
		}
		dir := ans.Folder
		if !isDir(dir) {
			if st, e := os.Stat(dir); e == nil && st.Mode().IsRegular() {
				dir = formats.ParentPathOS(dir)
			} else {
				report(req, SearchReport{Answer: true, BadRoots: []string{ans.Folder},
					Before: len(missing), NotFound: missing})
				continue
			}
		}
		missing = search(p, req, []string{dir}, true)
	}
	return nil, nil
}

func search(p *core.Project, req FindRequest, roots []string, answer bool) []*core.MediaRef {
	r := SearchReport{Answer: answer}
	for _, d := range roots {
		if isDir(d) {
			r.Roots = append(r.Roots, d)
		} else {
			r.BadRoots = append(r.BadRoots, d)
		}
	}
	emit(req.Progress, engine.StageSearch, 0, 1, "")
	res := media.SearchMedia(p, r.Roots, req.Depth)
	emit(req.Progress, engine.StageSearch, 1, 1, "")
	r.Before, r.Found, r.NotFound = len(res.Missing), len(res.Found), res.NotFound
	for _, m := range res.Missing {
		if c, ok := res.Ambiguous[m]; ok {
			r.Ambiguous = append(r.Ambiguous, Ambiguity{m, c})
		}
	}
	report(req, r)
	return res.NotFound
}

func report(req FindRequest, r SearchReport) {
	if req.OnSearch != nil {
		req.OnSearch(r)
	}
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func emit(pr engine.Progress, st engine.Stage, done, total int, file string) {
	if pr != nil {
		pr(engine.Event{Stage: st, Title: engine.StageTitles[st], Done: done, Total: total, File: file})
	}
}
