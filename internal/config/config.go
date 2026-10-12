package config

import (
	"fmt"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	"go.yaml.in/yaml/v3"
)

// Config is a parsed configuration: the request, and what in the file the
// user should fix though it doesn't stop the work (deprecated settings).
type Config struct {
	Request  model.Request
	Warnings issues.ConfigIssues
}

// deprecations collects the "deprecated-setting" warnings of one Parse.
type deprecations struct{ list issues.ConfigIssues }

func (d *deprecations) add(setting, message string) {
	d.list = append(d.list, issues.ConfigIssue{Code: issues.CodeDeprecatedSetting, Setting: setting, Message: message})
}

type Overrides struct {
	Target                      string
	DryRun                      bool
	RemoveOrphans, AddRelations *bool
}

func Parse(data []byte) (Config, error) {
	req := model.Request{RemoveOrphans: true}
	warnings := &deprecations{}
	root, node, err := decodeRoot(data, "config")
	if err != nil {
		return Config{}, err
	}
	settings, err := mapping(root["settings"], "settings")
	if err != nil {
		return Config{}, err
	}
	if req.TempDir, err = stringValue(settings, "temp_dir", ""); err != nil {
		return Config{}, err
	}
	if req.DryRun, err = boolean(settings, "dry_run", false); err != nil {
		return Config{}, err
	}
	if req.RemoveOrphans, err = boolean(settings, "remove_orphans", true); err != nil {
		return Config{}, err
	}
	if req.AddRelations, err = boolean(settings, "add_relations", false); err != nil {
		return Config{}, err
	}
	if _, set := settings["on_conflict"]; set {
		warnings.add("settings.on_conflict", "deprecated setting, remove it: skills with the same name are always an error")
	}
	if req.ExcludeFromChecks, err = globalExcludeFromChecks(settings, warnings); err != nil {
		return Config{}, err
	}
	target, rootTarget := root["target"]
	targetNode := fieldNode(node.Content[0], "target")
	legacyTarget, settingsTarget := settings["target"]
	if rootTarget && settingsTarget {
		return Config{}, fmt.Errorf("target cannot be defined both at root and in settings")
	}
	targetSetting := "target"
	if settingsTarget {
		targetSetting = "settings.target"
		target = legacyTarget
		targetNode = fieldNode(fieldNode(node.Content[0], "settings"), "target")
	}
	if req.Targets, err = parseTargets(target, targetNode, targetSetting, warnings); err != nil {
		return Config{}, err
	}
	nested, err := boolean(settings, "nested_sources", true)
	if err != nil {
		return Config{}, err
	}
	if req.Sources, err = parseSources(root["sources"], nested, warnings); err != nil {
		return Config{}, err
	}
	return Config{Request: req, Warnings: warnings.list}, nil
}

// decodeRoot reads data as a YAML (or JSON) mapping; what names the file
// in errors, e.g. "config".
func decodeRoot(data []byte, what string) (map[string]any, *yaml.Node, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", what, err)
	}
	if len(node.Content) == 0 {
		return nil, nil, fmt.Errorf("%s must be a mapping", what)
	}
	literalTags(&node)
	var root map[string]any
	if err := node.Decode(&root); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", what, err)
	}
	if root == nil {
		return nil, nil, fmt.Errorf("%s must be a mapping", what)
	}
	return root, &node, nil
}

// parseSources reads the "sources" list; a missing one is empty. nested
// is the "nested_sources" of a source that doesn't set it.
func parseSources(sources any, nested bool, warnings *deprecations) ([]model.SourceSpec, error) {
	if sources == nil {
		sources = []any{}
	}
	list, ok := sources.([]any)
	if !ok {
		return nil, fmt.Errorf("sources must be a list")
	}
	out := []model.SourceSpec{}
	for i, raw := range list {
		m, e := mapping(raw, "source")
		if e != nil {
			return nil, e
		}
		s, e := parseSource(m, fmt.Sprintf("sources[%d]", i), nested, warnings)
		if e != nil {
			return nil, e
		}
		out = append(out, s)
	}
	return out, nil
}
