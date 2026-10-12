package sourcing_test

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manager/tools/testsupport"
	"github.com/cucumber/godog"
)

// suffixDetector is a test sourcing.Detector: it selects every entry whose
// name ends with suffix and finds a skill named "<detector>=<path>". See
// scanner.feature.
type suffixDetector struct{ name, suffix string }

func (d suffixDetector) Name() string { return d.name }

func (d suffixDetector) IsSelected(e sourcing.Entry) bool {
	return strings.HasSuffix(path.Base(e.Path), d.suffix)
}

func (d suffixDetector) Select(ctx context.Context, e sourcing.Entry) (sourcing.ScanResult, issues.SkillIssues) {
	if strings.HasPrefix(path.Base(e.Path), "unbuildable") {
		return sourcing.ScanResult{}, issues.SkillIssues{{Code: issues.CodeInvalidSkill, File: e.Path, Message: "can't build"}}
	}
	return sourcing.ScanResult{Skills: []*entity.Skill{{Name: d.name + "=" + e.Path, MainFilePath: e.Path}}}, nil
}

func (d suffixDetector) IsValid(ctx context.Context, selected sourcing.ScanResult) issues.SkillIssues {
	var problems issues.SkillIssues
	for _, s := range selected.Skills {
		if strings.HasPrefix(path.Base(s.MainFilePath), "invalid") {
			problems = append(problems, issues.SkillIssue{Code: issues.CodeNestedSkill, File: s.MainFilePath, Message: "not valid"})
		}
	}
	return problems
}

// scannerSteps scans the files of the "a source tree" step (catalogSteps),
// which leaves them in tree.
func scannerSteps(sc *godog.ScenarioContext, tree *catalogTreeFS) {
	var canceled bool
	var found sourcing.ScanResult
	var problems issues.SkillIssues
	var refused error

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		canceled, found, problems, refused = false, sourcing.ScanResult{}, nil, nil
		return ctx, nil
	})
	sc.Step(`^the scan is canceled$`, func(ctx context.Context) error { canceled = true; return nil })
	sc.Step(`^I scan "([^"]*)" with detectors "([^"]*)"$`, func(ctx context.Context, start, spec string) error {
		var detectors []sourcing.Detector
		for _, one := range strings.Split(spec, ",") {
			name, suffix, _ := strings.Cut(one, ":")
			detectors = append(detectors, suffixDetector{name: name, suffix: suffix})
		}
		scanner, err := sourcing.NewScanner(detectors...)
		if err != nil {
			refused = err
			return nil
		}
		if canceled {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			cancel()
		}
		repo := &entity.Repository{Key: model.SourceKey{Type: "local", Path: "repo"}, FS: *tree}
		found, problems = scanner.Scan(ctx, repo, start)
		return nil
	})
	sc.Step(`^the scan finds "([^"]*)"$`, func(ctx context.Context, want string) error {
		if refused != nil {
			return refused
		}
		got := []string{}
		for _, s := range found.Skills {
			got = append(got, s.Name)
		}
		return testsupport.Equal(strings.Join(got, ","), want)
	})
	sc.Step(`^the scan issues are "([^"]*)"$`, func(ctx context.Context, want string) error {
		got := []string{}
		for _, i := range problems {
			got = append(got, string(i.Code)+"="+i.File)
		}
		return testsupport.Equal(strings.Join(got, ","), want)
	})
	sc.Step(`^the scan issue at "([^"]*)" says "([^"]*)"$`, func(ctx context.Context, file, text string) error {
		for _, i := range problems {
			if i.File == file {
				testsupport.Log("message=%s source=%s", i.Message, i.Source)
				if !strings.Contains(i.Message, text) {
					return fmt.Errorf("message=%q; want one with %q", i.Message, text)
				}
				return testsupport.Equal(i.Source, "local:repo")
			}
		}
		return fmt.Errorf("no issue at %q", file)
	})
	sc.Step(`^the scanner is refused with "(.*)"$`, func(ctx context.Context, text string) error {
		text = strings.ReplaceAll(text, `\"`, `"`)
		if refused == nil || !strings.Contains(refused.Error(), text) {
			return fmt.Errorf("error=%v; want one with %q", refused, text)
		}
		return nil
	})
}
