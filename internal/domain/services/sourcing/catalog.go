package sourcing

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
)

func isNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

// SkillCatalog is the skills found in acquired repositories, looked up by
// path. Get* only look at skills already loaded. Skills at or below a path
// are searched for by a Scanner and remembered here with Add; the skill
// whose folder holds a path is found by FetchByPathUp (GetOrFetchByPathUp
// also remembers it). What a skill is decide the detectors of Scanner, for
// both.
type SkillCatalog struct {
	Manager *Manager
	// Scanner recognizes a skill by its file or folder (see the detectors
	// package); it is given the skill detectors only. FetchByPathUp needs
	// it; a catalog only answering from its cache doesn't.
	Scanner *Scanner
	// AddRelations selects what TryGetOrFetchByPathUp does on a cache miss: true fetches the skill, false fails with
	// entity.ErrSkillNotCached -- mirrors the add_relations setting.
	AddRelations bool
	// ExcludeFromChecks maps a source to the top-level folders of its
	// skills that are loaded and copied with the skill but never checked:
	// no nested-skill check here, no link checks in validators. A source
	// missing from the map excludes nothing. See IsExcludedFromChecks.
	ExcludeFromChecks map[model.SourceKey][]string
	// cachedByPath maps entity.GetSkillKey(repo, p) to the skills a lookup
	// by path p answers: every skill at or below a requested path p, and
	// for each loaded skill its own folder (or, for a flat skill, its
	// marker file) -> that one skill. See addPath.
	cachedByPath map[string][]*entity.Skill
	// loaded is every loaded skill once, in the order it was loaded.
	loaded []*entity.Skill
}

var _ entity.SkillResolver = (*SkillCatalog)(nil)

// GetByPath returns the skills a complete search remembered (Add) at or
// below start, or the skill whose own folder (or flat marker file) start
// is. It never acquires a Repository nor reads a file; a start never
// requested before -- even one with loaded skills below it -- fails with
// entity.ErrSkillNotCached, since only a fetch knows it found them all.
//
// Примеры (репозиторий: скил guide в папке "a/guide", flat-скил
// "b.skill.md"; ранее полный поиск по "." запомнен через Add, см. addPath):
//   - GetByPath(".")          -> [guide, flat] (запрошенный ранее путь)
//   - GetByPath("a/guide")    -> [guide]       (папка скила)
//   - GetByPath("b.skill.md") -> [flat]        (файл-маркер flat-скила)
//   - GetByPath("a")          -> ErrSkillNotCached: "a" не запрашивали,
//     и это не папка скила, хотя guide лежит ниже него
func (c *SkillCatalog) GetByPath(ctx context.Context, key model.SourceKey, start string) ([]*entity.Skill, error) {
	start, err := c.cachedRepoPath(ctx, key, start)
	if err != nil {
		return nil, err
	}
	if skills, ok := c.cachedByPath[entity.GetSkillKey(key, start)]; ok {
		return skills, nil
	}
	return nil, fmt.Errorf("%w: %s in %s", entity.ErrSkillNotCached, start, key)
}

// GetByPathUp returns the loaded skill whose folder holds repo-relative
// path p -- unlike GetByPath, which answers for a path and below, it
// searches up: p is usually a file inside some skill. It checks p and each
// of its parent folders by key, without reading the filesystem. Nothing
// found fails with entity.ErrSkillNotCached.
//
// Примеры (загружены guide с папкой "a/guide" и flat-скил "b.skill.md";
// репозиторий в ОС лежит в "/home/u/skills"):
//   - GetByPathUp("a/guide/docs/x.md") -> guide: проверены ключи
//     "a/guide/docs/x.md", "a/guide/docs", "a/guide" -- найден на последнем
//   - GetByPathUp("a/guide/SKILL.md")  -> guide (файл-маркер в папке скила)
//   - GetByPathUp("a/guide")           -> guide (сама папка скила)
//   - GetByPathUp("b.skill.md")        -> flat (файл-маркер flat-скила)
//   - GetByPathUp("/home/u/skills/a/guide/docs/x.md") -> guide (путь ОС
//     сначала переводится в "a/guide/docs/x.md", см. cleanRelative)
//   - GetByPathUp("a")          -> ErrSkillNotCached ("a" выше скила, а не в нём)
//   - GetByPathUp("x/notes.md") -> ErrSkillNotCached (ни одна папка-родитель
//     не папка загруженного скила)
func (c *SkillCatalog) GetByPathUp(ctx context.Context, key model.SourceKey, p string) (*entity.Skill, error) {
	p, err := c.cachedRepoPath(ctx, key, p)
	if err != nil {
		return nil, err
	}
	for dir := p; ; dir = path.Dir(dir) {
		if skill := c.SkillAt(key, dir); skill != nil && ownsPath(skill, p) {
			return skill, nil
		}
		if dir == "." {
			return nil, fmt.Errorf("%w: no skill holds %s in %s", entity.ErrSkillNotCached, p, key)
		}
	}
}

// GetOrFetchByPathUp is GetByPathUp; on a cache miss it acquires key's
// Repository, fetches the skill whose folder holds p (FetchByPathUp) and
// remembers it (addPath).
func (c *SkillCatalog) GetOrFetchByPathUp(ctx context.Context, key model.SourceKey, p string) (*entity.Skill, error) {
	skill, err := c.GetByPathUp(ctx, key, p)
	if !errors.Is(err, entity.ErrSkillNotCached) {
		return skill, err
	}
	repo, p, err := c.Acquire(ctx, key, p)
	if err != nil {
		return nil, err
	}
	skill, err = c.FetchByPathUp(ctx, repo, p)
	if err != nil {
		return nil, err
	}
	c.addPath(repo.Key, skill.DirOrMarkerPath(), []*entity.Skill{skill})
	return skill, nil
}

// TryGetOrFetchByPathUp is GetOrFetchByPathUp when c.AddRelations is set,
// and GetByPathUp -- failing with entity.ErrSkillNotCached on a miss --
// otherwise.
func (c *SkillCatalog) TryGetOrFetchByPathUp(ctx context.Context, key model.SourceKey, p string) (*entity.Skill, error) {
	if !c.AddRelations {
		return c.GetByPathUp(ctx, key, p)
	}
	return c.GetOrFetchByPathUp(ctx, key, p)
}

// cachedRepoPath cleans p against key's already-acquired Repository
// without acquiring it; a Repository not acquired yet holds no loaded
// skill, so it fails with entity.ErrSkillNotCached.
//
// Примеры: для загруженного репозитория -- как cleanRelative
// ("./a//b/" -> "a/b"); для незагруженного -- на любой путь
// ErrSkillNotCached "source local:repo is not acquired".
func (c *SkillCatalog) cachedRepoPath(ctx context.Context, key model.SourceKey, p string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	repo, ok := c.Manager.LookupKey(ctx, key)
	if !ok {
		return "", fmt.Errorf("%w: source %s is not acquired", entity.ErrSkillNotCached, key)
	}
	return cleanRelative(repo, p)
}

// Acquire gets key's Repository via Manager (fetched once, then cached)
// and resolves p inside it (see normalizePath). A Repository the provider
// can't give is a "source-acquire" issue.
//
// Пример: Acquire(key, "/home/u/skills/a/guide") -> (репозиторий, "a/guide")
// -- первый вызов для key загружает репозиторий, следующие берут его из
// кэша Manager.
func (c *SkillCatalog) Acquire(ctx context.Context, key model.SourceKey, p string) (*entity.Repository, string, error) {
	repo, err := c.Manager.Get(ctx, key)
	if err != nil {
		return nil, "", issues.SkillIssue{Code: issues.CodeSourceAcquire, Source: key.String(), Message: err.Error()}
	}
	p, err = normalizePath(repo, p)
	if err != nil {
		return nil, "", err
	}
	return repo, p, nil
}

// Add remembers what a search at or below path start of repository key
// found (see Scanner.Scan). A complete search -- one without problems --
// becomes the answer GetByPath gives for start. After an incomplete one
// each skill is remembered at its own location only, so start stays
// unanswered and the search is run -- and its problems reported -- again.
//
// Примеры (поиск по "." нашёл guide в папке "a/guide"):
//   - Add(key, ".", [guide], true)  -> GetByPath(".") = [guide],
//     GetByPath("a/guide") = [guide]
//   - Add(key, ".", [guide], false) -> GetByPath(".") = ErrSkillNotCached,
//     GetByPath("a/guide") = [guide]
func (c *SkillCatalog) Add(key model.SourceKey, start string, skills []*entity.Skill, complete bool) {
	if complete {
		c.addPath(key, start, skills)
		return
	}
	for _, skill := range skills {
		c.addSkill(skill)
	}
}

// addPath remembers skills as the answer for path p of repository key, and
// each skill as the answer for its own location (DirOrMarkerPath) -- so GetByPath
// finds them by p or by a skill's folder, and GetByPathUp by any path
// inside a skill's folder.
//
// Пример: addPath(key, ".", [guide (папка "a/guide"), flat ("b.skill.md")])
// записывает три ключа:
//   - "."          -> [guide, flat] (ответ на запрошенный путь)
//   - "a/guide"    -> [guide]       (собственное место guide)
//   - "b.skill.md" -> [flat]        (собственное место flat-скила)
func (c *SkillCatalog) addPath(key model.SourceKey, p string, skills []*entity.Skill) {
	if c.cachedByPath == nil {
		c.cachedByPath = map[string][]*entity.Skill{}
	}
	for _, skill := range skills {
		if c.SkillAt(key, skill.DirOrMarkerPath()) == nil {
			c.loaded = append(c.loaded, skill)
		}
		c.cachedByPath[entity.GetSkillKey(key, skill.DirOrMarkerPath())] = []*entity.Skill{skill}
	}
	c.cachedByPath[entity.GetSkillKey(key, p)] = skills
}

// addSkill remembers skill at its own location only (see addPath).
func (c *SkillCatalog) addSkill(skill *entity.Skill) {
	c.addPath(skill.Repo.Key, skill.DirOrMarkerPath(), []*entity.Skill{skill})
}

// Skills returns every loaded skill once, in the order it was loaded: a
// skill loaded later -- e.g. fetched while following a link -- comes after
// all skills loaded before it, so a caller walking the list by index while
// loading more sees the new ones too.
func (c *SkillCatalog) Skills() []*entity.Skill {
	return append([]*entity.Skill(nil), c.loaded...)
}

// SkillAt returns the loaded skill whose own location (DirOrMarkerPath) is
// path p of repository key, or nil -- an entry for a requested path above skills is not one.
//
// Примеры (после addPath из примера выше):
//   - SkillAt("a/guide")      -> guide
//   - SkillAt("b.skill.md")   -> flat
//   - SkillAt(".")            -> nil: ключ есть, но это ответ на запрошенный
//     путь, а не место какого-то скила
//   - SkillAt("a")            -> nil: ключа нет
//   - SkillAt("a/guide/docs") -> nil: папка внутри скила -- не его место
//     (для этого есть GetByPathUp)
func (c *SkillCatalog) SkillAt(key model.SourceKey, p string) *entity.Skill {
	for _, skill := range c.cachedByPath[entity.GetSkillKey(key, p)] {
		if skill.DirOrMarkerPath() == p {
			return skill
		}
	}
	return nil
}

// FetchByPathUp finds the valid skill whose folder holds p inside repo
// -- p is a path from the repository folder, already resolved (see
// normalizePath; GetOrFetchByPathUp does that before calling it): it
// walks p's parent folders up to the repository folder until the first
// skill's folder (or takes p itself when it is a flat "*.skill.md" skill),
// showing each to the detectors of Scanner (Scanner.Probe).
// Its neighbours are not loaded; nothing is remembered (see addPath).
//
// Примеры (в репозитории: "a/guide/SKILL.md", "a/guide/docs/x.md",
// "b.skill.md", "h.skill/h.skill.md", "f/f.skill.md", "x/one/SKILL.md",
// "x/notes.md"):
//   - FetchByPathUp("a/guide/docs/x.md") -> guide: проверены папки
//     "a/guide/docs" (не скил), "a/guide" (скил)
//   - FetchByPathUp("a/guide/SKILL.md") -> guide: первой проверяется папка
//     файла, "a/guide" -- agent-dir скил
//   - FetchByPathUp("a/guide")          -> guide (папка сама -- скил)
//   - FetchByPathUp("h.skill/h.skill.md") -> human-dir скил в папке
//     "h.skill": файл назван по папке, значит это её маркер, а не flat-скил
//   - FetchByPathUp("b.skill.md")   -> flat-скил "b.skill.md"
//   - FetchByPathUp("f/f.skill.md") -> flat-скил "f/f.skill.md": папка "f"
//     не "*.skill", значит "f.skill.md" в ней -- не маркер human-dir скила
//   - FetchByPathUp("x/notes.md")   -> "skill-not-found": проверены "x" и
//     "."; "x/one" -- соседняя папка, а не родитель
func (c *SkillCatalog) FetchByPathUp(ctx context.Context, repo *entity.Repository, p string) (*entity.Skill, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := fs.Stat(repo.FS, p)
	if err != nil {
		return nil, err
	}
	dir := p
	if !info.IsDir() {
		dir = path.Dir(p)
	}
	if c.Scanner == nil {
		panic("sourcing: SkillCatalog.Scanner is not set, FetchByPathUp can't recognize a skill")
	}
	for {
		found, problems := c.Scanner.Probe(ctx, repo, dir)
		// A file that isn't claimed by its own folder's marker may still
		// be a flat skill of its own.
		if len(problems) == 0 && len(found.Skills) == 0 && dir == path.Dir(p) && !info.IsDir() {
			found, problems = c.Scanner.Probe(ctx, repo, p)
		}
		if len(problems) > 0 {
			return nil, problems[0]
		}
		if len(found.Skills) > 0 {
			return found.Skills[0], nil
		}
		if dir == "." {
			return nil, issues.SkillIssue{Code: issues.CodeSkillNotFound, Source: repo.Key.String(), File: p, Message: "no skill holds this path"}
		}
		dir = path.Dir(dir)
	}
}

// IsExcludedFromChecks reports whether rel, a path from skill's folder,
// lies in a top-level folder excluded from checks for skill's source (see
// ExcludeFromChecks).
//
// Примеры (для источника скила исключена "examples"):
//   - "examples/app/README.md" -> true
//   - "examples"               -> true (сама папка)
//   - "docs/examples/page.md"  -> false: исключается только верхний уровень
//   - "examples.md"            -> false: файл, а не папка "examples"
func (c *SkillCatalog) IsExcludedFromChecks(skill *entity.Skill, rel string) bool {
	first := strings.SplitN(rel, "/", 2)[0]
	for _, folder := range c.ExcludeFromChecks[skill.Repo.Key] {
		if first == folder {
			return true
		}
	}
	return false
}

// ownsPath отвечает на вопрос "принадлежит ли repo-relative путь p скилу s?"
// без чтения файлов -- только по уже известным идентичности/расположению
// скила (s.MainFilePath/s.SkillDirPath/s.Format).
//
// Правила:
//   - сам главный файл или сама корневая директория скила всегда принадлежат
//     ему;
//   - для flat-скила (SkillDirPath == "") этим всё и ограничивается -- у
//     него нет директории, в которую что-то ещё могло бы попасть;
//   - для directory-скила (agent-dir/human-dir) владение распространяется на
//     всё, что лежит внутри его директории, даже если этот файл ещё ни разу
//     не читался через FilesByPath.
//
// Примеры для скила с SkillDirPath="a/guide", MainFilePath="a/guide/SKILL.md",
// Format=AgentDirSkill:
//   - ownsPath(s, "a/guide/SKILL.md")   -> true  (главный файл)
//   - ownsPath(s, "a/guide")            -> true  (сама директория)
//   - ownsPath(s, "a/guide/notes.md")   -> true  (файл внутри директории)
//   - ownsPath(s, "a/other/notes.md")   -> false (другая директория)
//
// Для flat-скила с MainFilePath="guide.skill.md", SkillDirPath=""
// (Format=FlatSkill):
//   - ownsPath(s, "guide.skill.md")     -> true  (главный файл)
//   - ownsPath(s, "guide.skill.md.bak") -> false (директории нет, значит и
//     "владения" за пределами самого файла тоже нет)
func ownsPath(s *entity.Skill, p string) bool {
	if p == s.MainFilePath || p == s.SkillDirPath {
		return true
	}
	if s.Format == entity.FlatSkill {
		return false
	}
	return s.SkillDirPath == "." || strings.HasPrefix(p, strings.TrimSuffix(s.SkillDirPath, "/")+"/")
}

// cleanRelative resolves start into a clean, valid, repo-relative path --
// pure path arithmetic, no filesystem access, so a cache hit on the result
// costs nothing beyond it.
//
// Примеры (репозиторий в ОС лежит в "/home/u/skills"):
//   - "a/b"                -> "a/b"
//   - "./a//b/"            -> "a/b"
//   - "a/../b"             -> "b"
//   - "" или "."           -> "." (папка репозитория)
//   - "/home/u/skills/a/b" -> "a/b" (путь ОС внутри репозитория)
//   - "/home/u/skills"     -> "."
//   - "../x"               -> unsafe-subpath `unsafe subpath "../x"`
//   - "/home/u/other/x"    -> unsafe-subpath `unsafe subpath "../other/x"` (вне репозитория)
//   - `a\b` (на Linux)     -> unsafe-subpath `unsafe subpath "a\\b"`
//   - "/abs" при пустом пути репозитория в ОС -> unsafe-subpath с ошибкой filepath.Rel
func cleanRelative(repo *entity.Repository, start string) (string, error) {
	if filepath.IsAbs(start) {
		rel, err := filepath.Rel(repo.RootPath, start)
		if err != nil {
			return "", issues.SkillIssue{Code: issues.CodeUnsafeSubpath, Source: repo.Key.String(), File: start, Message: err.Error()}
		}
		start = rel
	}
	start = filepath.ToSlash(filepath.Clean(start))
	if !fs.ValidPath(start) || strings.Contains(start, "\\") {
		return "", issues.SkillIssue{Code: issues.CodeUnsafeSubpath, Source: repo.Key.String(), File: start, Message: fmt.Sprintf("unsafe subpath %q", start)}
	}
	return start, nil
}

// normalizePath resolves start into a clean, valid, repo-relative path safe to
// pass to repo.FS, additionally confirming it exists.
//
// Примеры (репозиторий "local:repo" в ОС лежит в "/home/u/skills", в нём
// есть "a/guide/docs/x.md"):
//   - "a/guide"                          -> "a/guide"
//   - "/home/u/skills/a/guide/docs/x.md" -> "a/guide/docs/x.md"
//   - "a/missing" -> missing-subpath `subpath "a/missing" does not exist in repository "local:repo"`
//   - "../x"      -> unsafe-subpath `unsafe subpath "../x"` (как у cleanRelative)
func normalizePath(repo *entity.Repository, start string) (string, error) {
	start, err := cleanRelative(repo, start)
	if err != nil {
		return "", err
	}
	if _, err := fs.Stat(repo.FS, start); err != nil {
		return "", issues.SkillIssue{Code: issues.CodeMissingSubpath, Source: repo.Key.String(), File: start, Message: fmt.Sprintf("subpath %q does not exist in repository %q", start, repo.Key)}
	}
	return start, nil
}
