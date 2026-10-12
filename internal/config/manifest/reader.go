// Package manifest reads source manifests -- files inside a source that
// list further sources -- by the rules of the project's own configuration:
// config.ParseSourceManifest for the shape, config/validator for the
// meaning.
package manifest

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/InsonusK/go-ai-skill-manager/internal/config"
	"github.com/InsonusK/go-ai-skill-manager/internal/config/validator"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
)

// Reader is the interfaces.SourceManifestReader of the YAML/JSON
// configuration format.
type Reader struct{}

var _ interfaces.SourceManifestReader = Reader{}

// ReadSourceManifest returns the sources manifest lists, as valid as the
// sources of a configuration that passed config/validator.Validate -- or
// the problems, and then no source:
//   - "invalid-manifest": the file is not a source manifest (see
//     config.ParseSourceManifest);
//   - "local-sub-source": a local source in a manifest that doesn't lie in
//     a local source itself. A repository somebody else wrote must not make
//     the run copy files from the user's machine;
//   - whatever config/validator.Validate reports for the sources, with the
//     same Setting ("sources[1].subpath[0]").
//
// The path of a local source is made absolute from the manifest's own
// folder, as a configuration's is from the project folder.
//
// Примеры (манифест "docs/skill-sources.yaml"):
//   - в local-источнике "/work/repo", "path: ../shared" -> local
//     "/work/repo/shared"; "path: /abs/x" -> local "/abs/x"
//   - в github-источнике, "type: local" -> local-sub-source, Setting
//     "sources[0].type"
//   - "subpath: [../x]" -> unsafe-subpath, Setting "sources[0].subpath[0]"
func (Reader) ReadSourceManifest(ctx context.Context, manifest model.SourceManifest) ([]model.SourceSpec, issues.ConfigIssues) {
	specs, err := config.ParseSourceManifest(manifest.Content)
	if err != nil {
		return nil, issues.ConfigIssues{{Code: issues.CodeInvalidManifest, Message: err.Error()}}
	}
	var problems issues.ConfigIssues
	dir := filepath.Join(manifest.RepoDirOsPath, filepath.FromSlash(path.Dir(manifest.Path)))
	for i := range specs {
		if specs[i].Type != "local" {
			continue
		}
		if manifest.Source.Type != "local" || manifest.RepoDirOsPath == "" {
			problems = append(problems, issues.ConfigIssue{
				Code: issues.CodeLocalSubSource, Source: specs[i].Key().String(), Setting: fmt.Sprintf("sources[%d].type", i),
				Message: fmt.Sprintf("a local source %q is allowed only in a manifest of a local source, this one lies in %s", specs[i].Path, manifest.Source),
			})
			continue
		}
		if filepath.IsAbs(specs[i].Path) {
			specs[i].Path = filepath.Clean(specs[i].Path)
		} else {
			specs[i].Path = filepath.Join(dir, specs[i].Path)
		}
	}
	problems = append(problems, validator.Validate(ctx, model.Request{Sources: specs})...)
	if len(problems) > 0 {
		return nil, problems
	}
	return specs, nil
}
