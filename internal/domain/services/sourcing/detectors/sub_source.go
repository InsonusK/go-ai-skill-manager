package detectors

import (
	"context"
	"io/fs"
	"path"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/sourcing"
)

// SubSource finds the sources a source names itself: a source manifest
// file (see model.IsSourceManifest) lists them, Reader reads it. A
// manifest inside a skill's folder is never shown to it -- a Scanner
// doesn't search a selected folder -- there it is just one of the skill's
// files.
type SubSource struct {
	Reader interfaces.SourceManifestReader
}

var _ sourcing.Detector = SubSource{}

func (d SubSource) Name() string { return "sub-source" }

// Примеры:
//   - файл "skill-sources.yaml"            -> true
//   - файл "docs/team.skill-sources.yaml"  -> true
//   - файл "ai-skills.yaml"                -> false
//   - папка "skill-sources.yaml"           -> false: манифест -- файл
func (d SubSource) IsSelected(e sourcing.Entry) bool {
	return !e.IsDir && model.IsSourceManifest(path.Base(e.Path))
}

// Select reads the manifest at e and returns the sources it lists. A file
// that can't be read is "source-read". What Reader finds wrong is reported
// with its own code, at the manifest file, the message starting with the
// setting -- and then none of the manifest's sources is returned.
//
// Пример: Reader вернул проблему {unsafe-subpath, Setting
// "sources[0].subpath[1]", Message "leads out"} для "a/skill-sources.yaml"
// -> {unsafe-subpath, File "a/skill-sources.yaml", Message
// "sources[0].subpath[1]: leads out"}.
func (d SubSource) Select(ctx context.Context, e sourcing.Entry) (sourcing.ScanResult, issues.SkillIssues) {
	source := e.Repo.Key.String()
	content, err := fs.ReadFile(e.Repo.FS, e.Path)
	if err != nil {
		return sourcing.ScanResult{}, issues.SkillIssues{{Code: issues.CodeSourceRead, Source: source, File: e.Path, Message: err.Error()}}
	}
	specs, found := d.Reader.ReadSourceManifest(ctx, model.SourceManifest{Source: e.Repo.Key, Path: e.Path, RepoDirOsPath: e.Repo.RootPath, Content: content})
	if len(found) > 0 {
		problems := make(issues.SkillIssues, len(found))
		for i, f := range found {
			message := f.Message
			if f.Setting != "" {
				message = f.Setting + ": " + message
			}
			problems[i] = issues.SkillIssue{Code: f.Code, Source: source, File: e.Path, Message: message}
		}
		return sourcing.ScanResult{}, problems
	}
	listed := make([]sourcing.ListedSource, len(specs))
	for i, spec := range specs {
		listed[i] = sourcing.ListedSource{Spec: spec, ListedBy: e.Repo.Key, Manifest: e.Path}
	}
	return sourcing.ScanResult{Sources: listed}, nil
}

// IsValid has nothing to add: Reader checks the sources while reading.
func (d SubSource) IsValid(ctx context.Context, selected sourcing.ScanResult) issues.SkillIssues {
	return nil
}
