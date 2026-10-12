package detectors

import (
	"context"
	"path"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/sourcing"
)

// AgentSkill finds agent-dir skills: a folder holding the marker file
// "SKILL.md".
type AgentSkill struct {
	Catalog *sourcing.SkillCatalog
}

var _ sourcing.Detector = AgentSkill{}

func (d AgentSkill) Name() string { return "agent-skill" }

// Примеры:
//   - папка "a/guide" с "SKILL.md"       -> true
//   - папка "g.skill" с "SKILL.md"       -> true (имя папки не важно)
//   - папка "p" только с "q/SKILL.md"    -> false: вглубь не смотрит
//   - папка "d" с папкой "SKILL.md"      -> false: маркер -- файл
//   - файл "a/guide/SKILL.md"            -> false: скил -- папка, не файл
func (d AgentSkill) IsSelected(e sourcing.Entry) bool {
	return e.IsDir && hasFile(e, "SKILL.md")
}

// Select builds the skill of folder e; an invalid name is "invalid-skill".
func (d AgentSkill) Select(ctx context.Context, e sourcing.Entry) (sourcing.ScanResult, issues.SkillIssues) {
	return selectSkill(d.Catalog, e, issues.CodeInvalidSkill, func() (*entity.Skill, error) {
		return entity.MakeSkill(e.Repo, path.Join(e.Path, "SKILL.md"), e.Path, entity.AgentDirSkill)
	})
}

func (d AgentSkill) IsValid(ctx context.Context, selected sourcing.ScanResult) issues.SkillIssues {
	return validateSkills(d.Catalog, selected)
}
