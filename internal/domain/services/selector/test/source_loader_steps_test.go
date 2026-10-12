package skill_selector_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	skill_selector "github.com/InsonusK/go-ai-skill-manager/internal/domain/services/selector"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/sourcing/detectors"
	"github.com/InsonusK/go-ai-skill-manager/tools/testsupport"
	"github.com/cucumber/godog"
)

// loaderRepositories is a fake interfaces.SourceProvider serving one
// in-memory tree per SourceKey.Path, whatever the tree asked for, and
// remembering every key it was asked for.
type loaderRepositories struct {
	trees    map[string]fstest.MapFS
	acquired *[]string
}

func (p loaderRepositories) Acquire(ctx context.Context, key model.SourceKey, options model.AcquisitionOptions) (*entity.Repository, error) {
	*p.acquired = append(*p.acquired, key.String())
	tree, ok := p.trees[key.Path]
	if !ok {
		return nil, fmt.Errorf("no repository %q", key.Path)
	}
	return &entity.Repository{Key: key, FS: tree}, nil
}

// loaderSpec reads "repository[@tree][ subpath,subpath][ !folder,folder]
// [ -nested]" as a local source; the "!" list is its exclude_from_checks,
// "-nested" its SkipNestedSources.
func loaderSpec(line string) model.SourceSpec {
	fields := strings.Fields(line)
	spec := model.SourceSpec{Type: "local"}
	spec.Path, spec.Tree, _ = strings.Cut(fields[0], "@")
	for _, f := range fields[1:] {
		if f == "-nested" {
			spec.SkipNestedSources = true
		} else if folders, ok := strings.CutPrefix(f, "!"); ok {
			spec.ExcludeFromChecks = strings.Split(folders, ",")
		} else {
			spec.Subpaths = strings.Split(f, ",")
		}
	}
	return spec
}

// loaderReader is a fake interfaces.SourceManifestReader: every line of a
// manifest is one source (see loaderSpec); a manifest holding the line
// "broken" is refused as invalid-manifest.
type loaderReader struct{}

func (loaderReader) ReadSourceManifest(ctx context.Context, manifest model.SourceManifest) ([]model.SourceSpec, issues.ConfigIssues) {
	var specs []model.SourceSpec
	for _, line := range strings.Split(string(manifest.Content), "\n") {
		if line == "broken" {
			return nil, issues.ConfigIssues{{Code: issues.CodeInvalidManifest, Message: "broken manifest"}}
		}
		if strings.TrimSpace(line) != "" {
			specs = append(specs, loaderSpec(line))
		}
	}
	return specs, nil
}

func sourceLoaderSteps(sc *godog.ScenarioContext) {
	var trees map[string]fstest.MapFS
	var acquired []string
	var catalog *sourcing.SkillCatalog
	var loader skill_selector.SourceLoader
	var problems issues.SkillIssues
	var canceled bool

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		trees, acquired, problems, canceled = map[string]fstest.MapFS{}, nil, nil, false
		catalog = &sourcing.SkillCatalog{Manager: sourcing.NewManager(map[string]interfaces.SourceProvider{"local": loaderRepositories{trees: trees, acquired: &acquired}}, "")}
		catalog.Scanner = detectors.MustScanner(catalog)
		loader = skill_selector.SourceLoader{Selector: skill_selector.SkillSelector{
			SkillCatalog: catalog,
			Scanner:      detectors.MustScanner(catalog, detectors.SubSource{Reader: loaderReader{}}),
		}}
		return ctx, nil
	})
	// repository NAME holds: JSON {path: content}; content "skill:NAME" is
	// the file of a skill named NAME.
	sc.Step(`^repository "([^"]*)" holds$`, func(ctx context.Context, name string, d *godog.DocString) error {
		var raw map[string]string
		if err := json.Unmarshal([]byte(d.Content), &raw); err != nil {
			return err
		}
		tree := fstest.MapFS{}
		for p, v := range raw {
			if skill, ok := strings.CutPrefix(v, "skill:"); ok {
				v = "---\nname: " + skill + "\n---\nBody"
			}
			tree[p] = &fstest.MapFile{Data: []byte(v), Mode: 0644}
		}
		trees[name] = tree
		testsupport.Log("repository=%s files=%v", name, raw)
		return nil
	})
	sc.Step(`^source manifests are not read$`, func(ctx context.Context) error {
		loader.Selector.Scanner = nil
		return nil
	})
	sc.Step(`^folders excluded from checks of every source are "([^"]*)"$`, func(ctx context.Context, folders string) error {
		loader.ExcludeFromChecks = strings.Split(folders, ",")
		return nil
	})
	sc.Step(`^folders excluded from checks of the configured source "([^"]*)" are "([^"]*)"$`, func(ctx context.Context, source, folders string) error {
		if catalog.ExcludeFromChecks == nil {
			catalog.ExcludeFromChecks = map[model.SourceKey][]string{}
		}
		catalog.ExcludeFromChecks[loaderSpec(source).Key()] = strings.Split(folders, ",")
		return nil
	})
	sc.Step(`^loading is canceled$`, func(ctx context.Context) error { canceled = true; return nil })
	// I load the sources: separated by ";", each as loaderSpec reads it.
	sc.Step(`^I load the sources "([^"]*)"$`, func(ctx context.Context, list string) error {
		var roots []model.SourceSpec
		for _, one := range strings.Split(list, ";") {
			roots = append(roots, loaderSpec(one))
		}
		if canceled {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			cancel()
		}
		problems = loader.Load(ctx, roots)
		testsupport.Log("issues=%v", problems)
		return nil
	})
	// the loaded skills are: "repository[@tree]:skill", in load order.
	sc.Step(`^the loaded skills are "([^"]*)"$`, func(ctx context.Context, want string) error {
		got := []string{}
		for _, s := range catalog.Skills() {
			got = append(got, strings.TrimPrefix(s.Repo.Key.String(), "local:")+":"+s.Name)
		}
		return testsupport.Equal(strings.Join(got, ","), want)
	})
	sc.Step(`^the repositories were acquired as "([^"]*)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(strings.ReplaceAll(strings.Join(acquired, ","), "local:", ""), want)
	})
	// the load issues are: JSON list of [code, source, file].
	sc.Step(`^the load issues are$`, func(ctx context.Context, d *godog.DocString) error {
		got := [][]string{}
		for _, i := range problems {
			got = append(got, []string{string(i.Code), i.Source, i.File})
		}
		return testsupport.JSON(got, d)
	})
	sc.Step(`^load issue (\d+) says "(.*)"$`, func(ctx context.Context, n int, text string) error {
		text = strings.ReplaceAll(text, `\"`, `"`)
		if n < 1 || n > len(problems) {
			return fmt.Errorf("%d issues, no issue %d", len(problems), n)
		}
		if !strings.Contains(problems[n-1].Message, text) {
			return fmt.Errorf("message=%q; want one with %q", problems[n-1].Message, text)
		}
		return nil
	})
}
