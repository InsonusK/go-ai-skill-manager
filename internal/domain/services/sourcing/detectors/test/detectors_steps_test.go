package detectors_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/sourcing/detectors"
	"github.com/InsonusK/go-ai-skill-manager/tools/testsupport"
	"github.com/cucumber/godog"
)

var repoKey = model.SourceKey{Type: "local", Path: "repo"}

// countingFS counts every Open, so a scenario can prove nothing more was
// read.
type countingFS struct {
	files fstest.MapFS
	opens *int
}

// Open opens a file; one named "unreadable..." opens but can't be read.
func (f countingFS) Open(name string) (fs.File, error) {
	*f.opens++
	file, err := f.files.Open(name)
	if err == nil && strings.HasPrefix(path.Base(name), "unreadable") {
		return unreadableFile{file}, nil
	}
	return file, err
}

type unreadableFile struct{ fs.File }

func (unreadableFile) Read([]byte) (int, error) { return 0, errors.New("disk failure") }

type treeProvider struct{ fs fs.FS }

func (p treeProvider) Acquire(ctx context.Context, key model.SourceKey, options model.AcquisitionOptions) (*entity.Repository, error) {
	return &entity.Repository{Key: key, RootPath: "/repos/" + key.Path, FS: p.fs}, nil
}

// lineReader is a fake interfaces.SourceManifestReader. Every line of a
// manifest is one source, "source TYPE PATH", or one problem, "problem
// CODE SETTING MESSAGE" (SETTING "-" for none). It remembers the manifests
// it was given.
type lineReader struct{ given *[]model.SourceManifest }

func (r lineReader) ReadSourceManifest(ctx context.Context, manifest model.SourceManifest) ([]model.SourceSpec, issues.ConfigIssues) {
	*r.given = append(*r.given, manifest)
	var specs []model.SourceSpec
	var problems issues.ConfigIssues
	for _, line := range strings.Split(string(manifest.Content), "\n") {
		f := strings.SplitN(line, " ", 4)
		switch {
		case len(f) == 3 && f[0] == "source":
			specs = append(specs, model.SourceSpec{Type: f[1], Path: f[2]})
		case len(f) == 4 && f[0] == "problem":
			problems = append(problems, issues.ConfigIssue{Code: issues.Code(f[1]), Setting: strings.TrimPrefix(f[2], "-"), Message: f[3]})
		}
	}
	return specs, problems
}

func initialize(sc *godog.ScenarioContext) {
	var catalog *sourcing.SkillCatalog
	var repo *entity.Repository
	var opens, opensBefore int
	var found sourcing.ScanResult
	var problems issues.SkillIssues
	var loaded map[string]*entity.Skill
	var given []model.SourceManifest
	every := func() []sourcing.Detector {
		return append(detectors.Skills(catalog), detectors.SubSource{Reader: lineReader{given: &given}})
	}

	// a source tree: JSON {path: content}; content "skill:NAME" is the
	// file of a skill named NAME.
	sc.Step(`^a source tree$`, func(ctx context.Context, d *godog.DocString) error {
		var raw map[string]string
		if err := json.Unmarshal([]byte(d.Content), &raw); err != nil {
			return err
		}
		tree := fstest.MapFS{}
		for p, v := range raw {
			if name, ok := strings.CutPrefix(v, "skill:"); ok {
				v = "---\nname: " + name + "\n---\nBody"
			}
			tree[p] = &fstest.MapFile{Data: []byte(v), Mode: 0644}
		}
		opens = 0
		files := countingFS{files: tree, opens: &opens}
		catalog = &sourcing.SkillCatalog{Manager: sourcing.NewManager(map[string]interfaces.SourceProvider{"local": treeProvider{fs: files}}, "")}
		catalog.Scanner = detectors.MustScanner(catalog)
		found, problems, loaded, given = sourcing.ScanResult{}, nil, map[string]*entity.Skill{}, nil
		var err error
		repo, err = catalog.Manager.Get(ctx, repoKey)
		testsupport.Log("files=%v", raw)
		return err
	})
	sc.Step(`^folders excluded from checks of source "([^"]*)" are "([^"]*)"$`, func(ctx context.Context, source, folders string) error {
		catalog.ExcludeFromChecks = map[model.SourceKey][]string{{Type: "local", Path: source}: strings.Split(folders, ",")}
		return nil
	})
	sc.Step(`^the skills at "([^"]*)" are already loaded$`, func(ctx context.Context, start string) error {
		result, list := catalog.Scanner.Scan(ctx, repo, start)
		if len(list) > 0 {
			return list
		}
		catalog.Add(repoKey, start, result.Skills, true)
		for _, s := range result.Skills {
			loaded[s.Name] = s
		}
		return nil
	})
	scan := func(ctx context.Context, start string, list ...sourcing.Detector) error {
		scanner, err := sourcing.NewScanner(list...)
		if err != nil {
			return err
		}
		found, problems = scanner.Scan(ctx, repo, start)
		return nil
	}
	sc.Step(`^I scan "([^"]*)" with detector "([^"]*)"$`, func(ctx context.Context, start, name string) error {
		for _, d := range every() {
			if d.Name() == name {
				return scan(ctx, start, d)
			}
		}
		return fmt.Errorf("no detector %q", name)
	})
	sc.Step(`^I scan "([^"]*)" with the skill detectors$`, func(ctx context.Context, start string) error {
		return scan(ctx, start, detectors.Skills(catalog)...)
	})
	sc.Step(`^I scan "([^"]*)" with every detector$`, func(ctx context.Context, start string) error {
		return scan(ctx, start, every()...)
	})
	// the scan finds sources: "type:path", comma-separated.
	sc.Step(`^the scan finds sources "([^"]*)"$`, func(ctx context.Context, want string) error {
		got := []string{}
		for _, s := range found.Sources {
			got = append(got, s.Key().String())
		}
		return testsupport.Equal(strings.Join(got, ","), want)
	})
	// the reader was given: JSON list of {source, path, repo_dir, content}.
	sc.Step(`^the manifest reader was given$`, func(ctx context.Context, d *godog.DocString) error {
		got := []map[string]string{}
		for _, m := range given {
			got = append(got, map[string]string{"source": m.Source.String(), "path": m.Path, "repo_dir": m.RepoDirOsPath, "content": string(m.Content)})
		}
		return testsupport.JSON(got, d)
	})
	// the scan finds: "name@location:format", comma-separated.
	sc.Step(`^the scan finds "([^"]*)"$`, func(ctx context.Context, want string) error {
		got := []string{}
		for _, s := range found.Skills {
			got = append(got, s.Name+"@"+s.DirOrMarkerPath()+":"+string(s.Format))
		}
		return testsupport.Equal(strings.Join(got, ","), want)
	})
	// the scan issues are: "code=file", comma-separated.
	sc.Step(`^the scan issues are "([^"]*)"$`, func(ctx context.Context, want string) error {
		got := []string{}
		for _, i := range problems {
			testsupport.Log("issue=%s", i.Error())
			got = append(got, string(i.Code)+"="+i.File)
		}
		return testsupport.Equal(strings.Join(got, ","), want)
	})
	// the scan issue N is: JSON {code, source, skill, skill_path, file, message}.
	sc.Step(`^the scan issue (\d+) is$`, func(ctx context.Context, n int, d *godog.DocString) error {
		if n < 1 || n > len(problems) {
			return fmt.Errorf("%d issues, no issue %d: %v", len(problems), n, problems)
		}
		i := problems[n-1]
		return testsupport.JSON(map[string]string{"code": string(i.Code), "source": i.Source, "skill": i.Skill, "skill_path": i.SkillPath, "file": i.File, "message": i.Message}, d)
	})
	sc.Step(`^the found skill "([^"]*)" is the loaded one$`, func(ctx context.Context, name string) error {
		for _, s := range found.Skills {
			if s.Name == name {
				if s != loaded[name] {
					return fmt.Errorf("skill %q was built again, not reused", name)
				}
				return nil
			}
		}
		return fmt.Errorf("skill %q not found", name)
	})
	sc.Step(`^listing the files of the found skill "([^"]*)" reads nothing more$`, func(ctx context.Context, name string) error {
		for _, s := range found.Skills {
			if s.Name == name {
				opensBefore = opens
				files, err := s.FilesByPath("")
				if err != nil {
					return err
				}
				testsupport.Log("files=%d", len(files))
				return testsupport.Equal(opens, opensBefore)
			}
		}
		return fmt.Errorf("skill %q not found", name)
	})
	sc.Step(`^the catalog has nothing loaded at "([^"]*)"$`, func(ctx context.Context, p string) error {
		skills, err := catalog.GetByPath(ctx, repoKey, p)
		testsupport.Log("skills=%d error=%v", len(skills), err)
		if !errors.Is(err, entity.ErrSkillNotCached) {
			return fmt.Errorf("error=%v, want ErrSkillNotCached", err)
		}
		return nil
	})
}
