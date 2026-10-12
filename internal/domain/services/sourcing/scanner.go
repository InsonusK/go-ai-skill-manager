package sourcing

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
)

// Entry is one file or folder a Scanner shows to its detectors. Path is
// from the repository folder; Children are a folder's own entries (one
// level, already read -- a detector needn't read the folder again).
type Entry struct {
	Repo     *entity.Repository
	Path     string
	IsDir    bool
	Children []fs.DirEntry
}

// Detector recognizes one kind of object -- an agent-dir skill, say -- by
// the file or folder it lives at. A Scanner asks every detector about
// every entry it walks over. The implementations are in the detectors
// package.
type Detector interface {
	// Name tells the detectors of a Scanner apart.
	Name() string
	// IsSelected reports whether e is an object of the detector's kind. It
	// only looks at e: no reading, no judging whether the object is a
	// correct one.
	IsSelected(e Entry) bool
	// Select builds what is at e; what can't be built is reported as
	// issues.
	Select(ctx context.Context, e Entry) (ScanResult, issues.SkillIssues)
	// IsValid checks what Select built: no issues means valid.
	IsValid(ctx context.Context, selected ScanResult) issues.SkillIssues
}

// ScanResult is what a search found, by kind, each in the order found.
type ScanResult struct {
	Skills []*entity.Skill
	// Sources are the sources the source manifests list, a manifest's
	// sources together, in the order written.
	Sources []ListedSource
}

// ListedSource is a source a source manifest lists.
type ListedSource struct {
	Spec model.SourceSpec
	// ListedBy is the source the manifest lies in, Manifest the manifest's
	// path from that source's repository folder.
	ListedBy model.SourceKey
	Manifest string
}

// add appends what other holds.
func (r *ScanResult) add(other ScanResult) {
	r.Skills = append(r.Skills, other.Skills...)
	r.Sources = append(r.Sources, other.Sources...)
}

// Scanner walks a repository's folders once, for all of its detectors.
type Scanner struct {
	detectors []Detector
}

// NewScanner builds a Scanner asking detectors in this order. Two
// detectors with one name are refused.
func NewScanner(detectors ...Detector) (*Scanner, error) {
	seen := map[string]bool{}
	for _, d := range detectors {
		if seen[d.Name()] {
			return nil, fmt.Errorf("scanner: detector %q is registered twice", d.Name())
		}
		seen[d.Name()] = true
	}
	return &Scanner{detectors: detectors}, nil
}

// Scan finds everything at or below start inside repo -- start is a path
// from the repository folder, already resolved (see normalizePath). Each
// file and folder is shown to every detector:
//   - nobody selects it: a folder is searched into, a file is skipped;
//   - one detector selects it: the object is built (Select) and checked
//     (IsValid). Valid, it is found; either way a selected folder is the
//     detector's and is not searched into;
//   - several detectors select it: a "pattern-conflict" issue, nothing is
//     built, a folder is not searched into.
//
// Issues never stop the rest of the search, a canceled context does. The
// ".git" folder is not searched. Nothing is remembered: that is up to the
// caller (SkillCatalog.Add for skills).
//
// Примеры (детекторы скилов; в репозитории: "a/guide/SKILL.md",
// "a/guide/docs/x.md", "b.skill.md", "x/notes.md"):
//   - Scan(".")            -> [guide, flat] ("x" обойдена, скилов нет)
//   - Scan("a")            -> [guide]
//   - Scan("a/guide")      -> [guide] (start сам -- папка скила)
//   - Scan("b.skill.md")   -> [flat]
//   - Scan("a/guide/docs") -> [] -- ищет только вниз; скил, в папке
//     которого лежит start, ищет SkillCatalog.FetchByPathUp
//   - Scan("missing")      -> [] (существование start проверяет вызывающий)
func (s *Scanner) Scan(ctx context.Context, repo *entity.Repository, start string) (ScanResult, issues.SkillIssues) {
	var out ScanResult
	var problems issues.SkillIssues
	var scan func(string)
	scan = func(p string) {
		entry, found, selected, list := s.probe(ctx, repo, p)
		problems = append(problems, list...)
		out.add(found)
		if selected || len(list) > 0 {
			return
		}
		for _, child := range entry.Children {
			if child.Name() != ".git" {
				scan(path.Join(p, child.Name()))
			}
		}
	}
	scan(path.Clean(start))
	return out, problems
}

// Probe shows the one file or folder at p to the detectors, by the rules
// of Scan, and searches no further: an empty result without issues means
// nobody selected p.
//
// Примеры (тот же репозиторий): Probe("a/guide") -> [guide];
// Probe("a") -> [] (папка "a" -- не скил, внутрь не смотрит).
func (s *Scanner) Probe(ctx context.Context, repo *entity.Repository, p string) (ScanResult, issues.SkillIssues) {
	_, found, _, problems := s.probe(ctx, repo, path.Clean(p))
	return found, problems
}

// probe is Probe, returning also the entry it showed to the detectors and
// whether one of them selected it.
func (s *Scanner) probe(ctx context.Context, repo *entity.Repository, p string) (Entry, ScanResult, bool, issues.SkillIssues) {
	entry := Entry{Repo: repo, Path: p}
	failed := func(code issues.Code, message string) (Entry, ScanResult, bool, issues.SkillIssues) {
		return entry, ScanResult{}, false, issues.SkillIssues{{Code: code, Source: repo.Key.String(), File: p, Message: message}}
	}
	if err := ctx.Err(); err != nil {
		return failed(issues.CodeCanceled, err.Error())
	}
	info, err := fs.Stat(repo.FS, p)
	if err != nil {
		if isNotExist(err) {
			return entry, ScanResult{}, false, nil
		}
		return failed(issues.CodeSourceRead, err.Error())
	}
	if entry.IsDir = info.IsDir(); entry.IsDir {
		if entry.Children, err = fs.ReadDir(repo.FS, p); err != nil {
			return failed(issues.CodeSourceRead, err.Error())
		}
	}
	var selected []Detector
	for _, d := range s.detectors {
		if d.IsSelected(entry) {
			selected = append(selected, d)
		}
	}
	switch len(selected) {
	case 0:
		return entry, ScanResult{}, false, nil
	case 1:
		found, problems := selected[0].Select(ctx, entry)
		if len(problems) == 0 {
			problems = selected[0].IsValid(ctx, found)
		}
		if len(problems) > 0 {
			return entry, ScanResult{}, true, problems
		}
		return entry, found, true, nil
	default:
		names := make([]string, len(selected))
		for i, d := range selected {
			names[i] = d.Name()
		}
		return failed(issues.CodePatternConflict, "selected by several detectors: "+strings.Join(names, ", "))
	}
}
