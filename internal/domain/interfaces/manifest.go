package interfaces

import (
	"context"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
)

// SourceManifestReader reads the sources a source manifest lists. It
// parses and checks them by the rules of the project's own configuration,
// so the sources it returns are as valid as the configured ones; what is
// wrong in the manifest is returned as issues, each with the Setting it is
// about (e.g. "sources[0].subpath"), next to the sources that are right.
// The domain owns the port because it doesn't know the configuration
// format.
type SourceManifestReader interface {
	ReadSourceManifest(ctx context.Context, manifest model.SourceManifest) ([]model.SourceSpec, issues.ConfigIssues)
}
