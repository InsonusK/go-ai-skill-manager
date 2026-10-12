// Package detectors holds the sourcing.Detector implementations: what a
// sourcing.Scanner can find in a repository.
package detectors

import (
	"fmt"
	"path"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/sourcing"
)

// A skill has three formats, each with a detector of its own: FlatSkill,
// AgentSkill, HumanSkill. A folder two of them select -- "x.skill" holding
// both "SKILL.md" and "x.skill.md" -- is a "pattern-conflict", reported by
// the Scanner. What they share is here.

// Skills returns the detectors of every skill format. catalog gives the
// skills already loaded -- one of those is reused, not built again -- and
// the folders excluded from checks.
func Skills(catalog *sourcing.SkillCatalog) []sourcing.Detector {
	return []sourcing.Detector{AgentSkill{Catalog: catalog}, HumanSkill{Catalog: catalog}, FlatSkill{Catalog: catalog}}
}

// MustScanner builds the Scanner finding skills of every format and
// whatever extra finds. It panics on a repeated detector name: the list is
// code, not configuration.
func MustScanner(catalog *sourcing.SkillCatalog, extra ...sourcing.Detector) *sourcing.Scanner {
	scanner, err := sourcing.NewScanner(append(Skills(catalog), extra...)...)
	if err != nil {
		panic(err)
	}
	return scanner
}

// selectSkill is Select of the skill detectors: the skill already loaded at
// e, or the one build makes; a skill that can't be built is an issue with
// code.
func selectSkill(catalog *sourcing.SkillCatalog, e sourcing.Entry, code issues.Code, build func() (*entity.Skill, error)) (sourcing.ScanResult, issues.SkillIssues) {
	skill := catalog.SkillAt(e.Repo.Key, e.Path)
	if skill == nil {
		var err error
		if skill, err = build(); err != nil {
			return sourcing.ScanResult{}, issues.SkillIssues{{Code: code, Source: e.Repo.Key.String(), File: e.Path, Message: err.Error()}}
		}
	}
	return sourcing.ScanResult{Skills: []*entity.Skill{skill}}, nil
}

// validateSkills is IsValid of the skill detectors: the first problem of
// each selected skill (see validateSkill).
func validateSkills(catalog *sourcing.SkillCatalog, selected sourcing.ScanResult) issues.SkillIssues {
	var problems issues.SkillIssues
	for _, skill := range selected.Skills {
		if problem := validateSkill(catalog, skill); problem != nil {
			problems = append(problems, *problem)
		}
	}
	return problems
}

// validateSkill checks a skill just built. Its files can be listed
// (FilesByPath(""), which also warms that cache), and none of them looks
// like a skill's own marker (SKILL.md or *.skill.md):
//   - one right in the skill's folder is a "pattern-conflict" -- a folder
//     skill and a flat skill in one place;
//   - one deeper is a "nested-skill", unless its top-level folder is
//     excluded from checks: that folder is still part of the skill and
//     copied along with it.
//
// Примеры (пути файлов -- от папки скила, исключена папка "examples"):
//   - ["docs/a.md", "b/SKILL.md"]   -> nested-skill "b/SKILL.md"
//   - ["docs/x.skill.md"]           -> nested-skill "docs/x.skill.md"
//   - ["other.skill.md"]            -> pattern-conflict "other.skill.md"
//   - ["b/SKILL.md", "other.skill.md"] -> pattern-conflict "other.skill.md"
//     (конфликт форматов важнее вложенного скила)
//   - ["examples/demo/SKILL.md"]    -> нет проблем (верхняя папка исключена)
//   - ["docs/examples/SKILL.md"]    -> nested-skill (исключается только
//     папка верхнего уровня)
//   - ["notSKILL.md", "a.skill.md.bak"] -> нет проблем (имена не маркеры)
func validateSkill(catalog *sourcing.SkillCatalog, skill *entity.Skill) *issues.SkillIssue {
	problem := func(code issues.Code, file, message string) *issues.SkillIssue {
		return &issues.SkillIssue{Code: code, Source: skill.Repo.Key.String(), Skill: skill.Name, SkillPath: skill.DirOrMarkerPath(), File: file, Message: message}
	}
	files, err := skill.FilesByPath("")
	if err != nil {
		return problem(issues.CodeSourceRead, skill.MainFilePath, err.Error())
	}
	nested := ""
	for _, f := range files {
		rel, err := f.Path(model.SkillRelative)
		if err != nil {
			continue
		}
		name := path.Base(rel)
		if name != "SKILL.md" && !strings.HasSuffix(name, ".skill.md") {
			continue
		}
		if name == rel {
			return problem(issues.CodePatternConflict, rel, "pattern-conflict: directory marker and flat skill")
		}
		if nested == "" && !catalog.IsExcludedFromChecks(skill, rel) {
			nested = rel
		}
	}
	if nested != "" {
		return problem(issues.CodeNestedSkill, nested, fmt.Sprintf("nested-skill: %s", nested))
	}
	return nil
}

// hasFile reports whether e is a folder holding a file (not a folder)
// named name.
func hasFile(e sourcing.Entry, name string) bool {
	for _, child := range e.Children {
		if !child.IsDir() && child.Name() == name {
			return true
		}
	}
	return false
}
