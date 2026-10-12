package manifest_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/config/manifest"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manager/tools/testsupport"
	"github.com/cucumber/godog"
)

func initialize(sc *godog.ScenarioContext) {
	var given model.SourceManifest
	var specs []model.SourceSpec
	var problems issues.ConfigIssues

	// A local source lies in its own path; any other is downloaded to
	// "/tmp/downloaded".
	sc.Step(`^a source manifest "([^"]*)" in (local|github) source "([^"]*)"$`, func(ctx context.Context, p, kind, source string, d *godog.DocString) error {
		given = model.SourceManifest{Source: model.SourceKey{Type: kind, Path: source}, Path: p, RepoDirOsPath: source, Content: []byte(d.Content)}
		if kind != "local" {
			given.Source.Tree = "master"
			given.RepoDirOsPath = "/tmp/downloaded"
		}
		specs, problems = nil, nil
		testsupport.Log("manifest=%s in %s\n%s", p, given.Source, d.Content)
		return nil
	})
	sc.Step(`^I read the source manifest$`, func(ctx context.Context) error {
		specs, problems = manifest.Reader{}.ReadSourceManifest(ctx, given)
		testsupport.Log("issues=%v", problems)
		return nil
	})
	// the manifest sources are: a JSON list of {type, path, tree, subpaths,
	// tags, exclude}.
	sc.Step(`^the manifest sources are$`, func(ctx context.Context, d *godog.DocString) error {
		if len(problems) > 0 {
			return fmt.Errorf("unexpected issues:\n%v", problems)
		}
		got := []map[string]any{}
		for _, s := range specs {
			got = append(got, map[string]any{"type": s.Type, "path": s.Path, "tree": s.Tree, "subpaths": s.Subpaths, "tags": s.Tags, "exclude": s.ExcludeFromChecks})
		}
		return testsupport.JSON(got, d)
	})
	// the manifest issues are: a JSON list of [code, source, setting]; no
	// source is returned next to issues.
	sc.Step(`^the manifest issues are$`, func(ctx context.Context, d *godog.DocString) error {
		got := [][]string{}
		for _, i := range problems {
			got = append(got, []string{string(i.Code), i.Source, i.Setting})
		}
		if err := testsupport.JSON(got, d); err != nil {
			return err
		}
		return testsupport.Equal(len(specs), 0)
	})
	sc.Step(`^the manifest sources read nested sources "([^"]*)"$`, func(ctx context.Context, want string) error {
		flags := []string{}
		for _, s := range specs {
			flags = append(flags, strconv.FormatBool(!s.SkipNestedSources))
		}
		return testsupport.Equal(strings.Join(flags, ","), want)
	})
	sc.Step(`^the manifest is refused as "([^"]*)" with "(.*)"$`, func(ctx context.Context, code, text string) error {
		text = strings.ReplaceAll(text, `\"`, `"`)
		if len(problems) != 1 {
			return fmt.Errorf("%d issues, want 1: %v", len(problems), problems)
		}
		if err := testsupport.Equal([]string{string(problems[0].Code), problems[0].Setting}, []string{code, ""}); err != nil {
			return err
		}
		if !strings.Contains(problems[0].Message, text) {
			return fmt.Errorf("message=%q; want one with %q", problems[0].Message, text)
		}
		return testsupport.Equal(len(specs), 0)
	})
	sc.Step(`^the manifest issue (\d+) says "(.*)"$`, func(ctx context.Context, n int, text string) error {
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
