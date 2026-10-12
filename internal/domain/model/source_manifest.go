package model

import "strings"

// SourceManifestFile is the name of a source manifest: a file inside a
// source, outside its skills' folders, listing further sources to load.
// "<any>.skill-sources.yaml" is a manifest too, so one folder can hold
// several.
const SourceManifestFile = "skill-sources.yaml"

// IsSourceManifest reports whether a file named name (no folders) is a
// source manifest.
//
// Примеры:
//   - "skill-sources.yaml"       -> true
//   - "team.skill-sources.yaml"  -> true
//   - "skill-sources.yml"        -> false (только .yaml)
//   - "myskill-sources.yaml"     -> false (перед именем нужна точка)
//   - ".skill-sources.yaml"      -> false (пустое имя набора)
//   - "ai-skills.yaml"           -> false (конфиг проекта -- не манифест)
func IsSourceManifest(name string) bool {
	if name == SourceManifestFile {
		return true
	}
	prefix, ok := strings.CutSuffix(name, "."+SourceManifestFile)
	return ok && prefix != ""
}

// SourceManifest is a source manifest file as found in a source.
type SourceManifest struct {
	// Source is the source the manifest lies in.
	Source SourceKey
	// Path is the manifest's path from the repository folder, e.g.
	// "docs/skills/team.skill-sources.yaml".
	Path string
	// RepoDirOsPath is the absolute OS path of the folder the repository
	// lies in or was downloaded to; empty when the provider gives none.
	RepoDirOsPath string
	Content       []byte
}
