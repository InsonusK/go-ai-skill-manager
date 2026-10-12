package skill_selector

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"slices"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
)

// SourceLoader loads the configured sources and, after them, every source
// they name themselves in source manifests -- and the ones those name, and
// so on.
type SourceLoader struct {
	// Selector loads one source; its Scanner decides whether manifests are
	// read at all (see detectors.SubSource).
	Selector SkillSelector
	// ExcludeFromChecks are the folders excluded from checks for every
	// source; a source a manifest lists adds its own to them.
	ExcludeFromChecks []string
}

// repository is what a source loads from, whatever the tree.
type repository struct{ kind, path string }

// Load selects roots in order, then the sources their manifests list in
// the order found, level by level, and returns every problem. A problem
// doesn't stop the other sources, a canceled context does.
//
// A subpath of a source is loaded once, whoever asks for it and however
// many times: a source two manifests list is not loaded twice, sources
// listing each other end, and a source listed again with more subpaths is
// loaded for the new ones only.
//
// A listed source asking for another tree of a repository than the one
// already known -- from roots or from a manifest read earlier -- is a
// "conflicting-tree" issue at its manifest and is not loaded: two trees of
// one repository would give every skill twice. (roots themselves are not
// compared with each other: that is the configuration's own business.)
//
// The folders excluded from checks of a listed source are
// ExcludeFromChecks and its own -- unless its repository already has a
// list, from roots or an earlier manifest: a repository's skills are
// loaded once, the first list stays.
//
// Примеры (roots: [A]; манифест в A перечисляет B и C; манифест в B --
// A и C):
//   - порядок загрузки: A, B, C; A и C во второй раз не загружаются
//   - B перечислен как B@v2, а в roots есть B@master -> conflicting-tree
//     на манифесте A, B@v2 не загружается
//   - roots: [A subpath a]; манифест перечисляет A subpath [a, b] ->
//     дополнительно загружается только "b"
func (l SourceLoader) Load(ctx context.Context, roots []model.SourceSpec) issues.SkillIssues {
	catalog := l.Selector.SkillCatalog
	var problems issues.SkillIssues
	loaded := map[string]bool{}
	trees := map[repository][]string{}
	for _, spec := range roots {
		id := repository{spec.Type, spec.Path}
		trees[id] = append(trees[id], spec.Tree)
	}
	queue := slices.Clone(roots)
	for len(queue) > 0 {
		spec := queue[0]
		queue = queue[1:]
		if spec.Subpaths = notLoaded(loaded, spec); len(spec.Subpaths) == 0 {
			continue
		}
		selected, found := l.Selector.Select(ctx, spec)
		problems = append(problems, found...)
		if ctx.Err() != nil {
			// Select has reported it.
			break
		}
		for _, listed := range selected.Sources {
			id := repository{listed.Spec.Type, listed.Spec.Path}
			if known := trees[id]; len(known) > 0 && !slices.Contains(known, listed.Spec.Tree) {
				problems = append(problems, issues.SkillIssue{
					Code: issues.CodeConflictingTree, Source: listed.ListedBy.String(), File: listed.Manifest,
					Message: fmt.Sprintf("%s: the repository is already loaded at tree %q, a second tree of it would give every skill twice", listed.Spec.Key(), known[0]),
				})
				continue
			}
			trees[id] = append(trees[id], listed.Spec.Tree)
			if _, set := catalog.ExcludeFromChecks[listed.Spec.Key()]; !set {
				if catalog.ExcludeFromChecks == nil {
					catalog.ExcludeFromChecks = map[model.SourceKey][]string{}
				}
				catalog.ExcludeFromChecks[listed.Spec.Key()] = union(l.ExcludeFromChecks, listed.Spec.ExcludeFromChecks)
			}
			queue = append(queue, listed.Spec)
		}
	}
	return problems
}

// notLoaded returns the subpaths of spec ("." when it has none) nobody
// asked for yet, and remembers them as asked for.
//
// Пример: уже загружены "a" источника S; spec S с subpath ["./a", "b",
// "b/"] -> ["b"].
func notLoaded(loaded map[string]bool, spec model.SourceSpec) []string {
	subpaths := spec.Subpaths
	if len(subpaths) == 0 {
		subpaths = []string{"."}
	}
	out := []string{}
	for _, p := range subpaths {
		key := spec.Key().String() + "\x00" + path.Clean(filepath.ToSlash(p))
		if !loaded[key] {
			loaded[key] = true
			out = append(out, p)
		}
	}
	return out
}

// union is a followed by what b adds, each once.
func union(a, b []string) []string {
	out := slices.Clone(a)
	for _, v := range b {
		if !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}
