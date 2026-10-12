package skill_selector

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/sourcing"
)

// SkillSelector fills SkillCatalog with the skills a source selects: the
// catalog's Scanner searches the source's folders, the catalog remembers
// what it found.
type SkillSelector struct {
	SkillCatalog *sourcing.SkillCatalog
}

// LoadPath returns the skills at or below path p of source key, loading
// them into the catalog first when it has no complete answer for p yet
// (catalog.GetByPath): the Repository is acquired (lazily, cached by the
// catalog's Manager), p is resolved inside it, the catalog's Scanner
// searches from it down, and the catalog remembers the skills found (catalog.Add). Invalid
// candidates are returned as issues.SkillIssues next to the valid skills;
// such a search is not complete, so a repeated call searches -- and
// reports the same issues -- again.
//
// Примеры (репозиторий: скилы one в "a", two в "b"):
//   - LoadPath("a")  -> [one]: репозиторий получен, "a" обойдена
//   - LoadPath("a") ещё раз -> [one] из каталога, ничего не читается
//   - LoadPath(".") после "a" -> [one, two]: "." шире, обходится заново,
//     one берётся уже загруженный
func (d SkillSelector) LoadPath(ctx context.Context, key model.SourceKey, p string) ([]*entity.Skill, error) {
	skills, err := d.SkillCatalog.GetByPath(ctx, key, p)
	if !errors.Is(err, entity.ErrSkillNotCached) {
		return skills, err
	}
	repo, start, err := d.SkillCatalog.Acquire(ctx, key, p)
	if err != nil {
		return nil, err
	}
	found, problems := d.SkillCatalog.Scanner.Scan(ctx, repo, start)
	skills = found.Skills
	d.SkillCatalog.Add(repo.Key, start, skills, len(problems) == 0)
	if len(problems) > 0 {
		return skills, problems
	}
	return skills, nil
}

// Select loads into the catalog the skills spec selects by path: for each
// of its subpaths (defaulting to ".") it calls LoadPath with spec's
// SourceKey. Select returns the skills, each once.
//
// spec.Tags are not applied yet: selecting by tags is postponed (see
// AGENTS.md), every skill under the subpaths is selected.
//
// spec must have passed config/validator.Validate: a subpath leading out
// of the source reaching Select is a bug in the caller, and Select panics
// on it.
//
// Problems never stop the other subpaths, except those that stop the whole
// source: a source that can't be acquired ("source-acquire") and a
// canceled context. Every problem carries the source it belongs to.
//
// Примеры (источник local:repo, subpaths ["a", "missing", "b"]):
//   - всё есть -> скилы из "a" и "b", проблем нет
//   - нет папки "missing" -> скилы из "a" и "b" + missing-subpath (File
//     "missing")
//   - провайдер не отдал репозиторий -> ни одного скила, одна проблема
//     source-acquire (а не по одной на каждый subpath)
func (d SkillSelector) Select(ctx context.Context, spec model.SourceSpec) ([]*entity.Skill, issues.SkillIssues) {
	source := spec.Key().String()
	var problems issues.SkillIssues
	selected := []*entity.Skill{}
	seen := map[string]bool{}
	paths := spec.Subpaths
	if len(paths) == 0 {
		paths = []string{"."}
	}
	for _, p := range paths {
		found, err := d.LoadPath(ctx, spec.Key(), p)
		if err != nil {
			list := asIssues(err, p)
			for _, i := range list {
				if i.Code == issues.CodeUnsafeSubpath {
					panic(fmt.Sprintf("skill_selector: subpath %q of source %s leads out of it, config/validator.Validate must reject it: %s", p, source, i.Message))
				}
			}
			problems = append(problems, list...)
			if stopsSource(ctx, list) {
				break
			}
		}
		for _, s := range found {
			if !seen[s.Key()] {
				selected = append(selected, s)
				seen[s.Key()] = true
			}
		}
	}
	for i := range problems {
		if problems[i].Source == "" {
			problems[i].Source = source
		}
	}
	return selected, problems
}

// asIssues turns a LoadPath error for subpath p into issues: the
// catalog's own issues as they are, a canceled context as "canceled", any
// other error as "source-read".
func asIssues(err error, p string) issues.SkillIssues {
	var list issues.SkillIssues
	if errors.As(err, &list) {
		return list
	}
	var one issues.SkillIssue
	if errors.As(err, &one) {
		return issues.SkillIssues{one}
	}
	code := issues.CodeSourceRead
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		code = issues.CodeCanceled
	}
	return issues.SkillIssues{{Code: code, File: p, Message: err.Error()}}
}

// stopsSource reports whether the remaining subpaths of the source are
// not worth trying: the context is done or the source can't be acquired.
func stopsSource(ctx context.Context, list issues.SkillIssues) bool {
	if ctx.Err() != nil {
		return true
	}
	for _, i := range list {
		if i.Code == issues.CodeSourceAcquire {
			return true
		}
	}
	return false
}

func isNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

func Tags(s *entity.Skill) []string {
	raw := s.Document.Properties["tags"]
	if text, ok := raw.(string); ok {
		return []string{strings.TrimSpace(text)}
	}
	out := []string{}
	if list, ok := raw.([]any); ok {
		for _, v := range list {
			if text, ok := v.(string); ok {
				out = append(out, strings.TrimSpace(text))
			}
		}
	}
	return out
}
