package detectors

import (
	"context"
	"path"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/sourcing"
)

// HumanSkill finds human-dir skills: a folder "x.skill" holding the marker
// file named after it, "x.skill.md".
type HumanSkill struct {
	Catalog *sourcing.SkillCatalog
}

var _ sourcing.Detector = HumanSkill{}

func (d HumanSkill) Name() string { return "human-skill" }

// Примеры:
//   - папка "h.skill" с "h.skill.md"        -> true
//   - папка "o.skill" с "other.skill.md"    -> false: маркер должен называться
//     по папке ("o.skill.md")
//   - папка "f" с "f.skill.md"              -> false: папка не "*.skill", файл
//     в ней -- flat-скил
//   - папка "h.skill" только с "SKILL.md"   -> false: это agent-dir скил
func (d HumanSkill) IsSelected(e sourcing.Entry) bool {
	return e.IsDir && strings.HasSuffix(e.Path, ".skill") && hasFile(e, humanMarker(e.Path))
}

// Select builds the skill of folder e; an invalid name is "invalid-skill".
func (d HumanSkill) Select(ctx context.Context, e sourcing.Entry) (sourcing.ScanResult, issues.SkillIssues) {
	return selectSkill(d.Catalog, e, issues.CodeInvalidSkill, func() (*entity.Skill, error) {
		return entity.MakeSkill(e.Repo, path.Join(e.Path, humanMarker(e.Path)), e.Path, entity.HumanDirSkill)
	})
}

func (d HumanSkill) IsValid(ctx context.Context, selected sourcing.ScanResult) issues.SkillIssues {
	return validateSkills(d.Catalog, selected)
}

// humanMarker is the marker file's name for a human-dir skill's folder.
//
// Пример: "a/h.skill" -> "h.skill.md".
func humanMarker(dir string) string { return path.Base(dir) + ".md" }
