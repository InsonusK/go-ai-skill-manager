package detectors

import (
	"context"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/sourcing"
)

// FlatSkill finds flat skills: a "*.skill.md" file. Inside a folder skill
// such a file is never shown to it -- a Scanner doesn't search a selected
// folder -- so there it is the folder's marker or a "pattern-conflict".
type FlatSkill struct {
	Catalog *sourcing.SkillCatalog
}

var _ sourcing.Detector = FlatSkill{}

func (d FlatSkill) Name() string { return "flat-skill" }

// Примеры:
//   - файл "b.skill.md"       -> true
//   - файл "f/f.skill.md"     -> true
//   - файл "notes.md"         -> false
//   - папка "x.skill.md"      -> false: flat-скил -- файл
func (d FlatSkill) IsSelected(e sourcing.Entry) bool {
	return !e.IsDir && strings.HasSuffix(e.Path, ".skill.md")
}

// Select builds the skill of file e; an invalid name is "invalid-name".
func (d FlatSkill) Select(ctx context.Context, e sourcing.Entry) (sourcing.ScanResult, issues.SkillIssues) {
	return selectSkill(d.Catalog, e, issues.CodeInvalidName, func() (*entity.Skill, error) {
		return entity.MakeSkill(e.Repo, e.Path, "", entity.FlatSkill)
	})
}

func (d FlatSkill) IsValid(ctx context.Context, selected sourcing.ScanResult) issues.SkillIssues {
	return validateSkills(d.Catalog, selected)
}
