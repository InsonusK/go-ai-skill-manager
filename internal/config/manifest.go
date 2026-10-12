package config

import (
	"fmt"
	"slices"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
)

// ParseSourceManifest reads a source manifest -- a file inside a source
// listing further sources (see model.IsSourceManifest). It is the "sources"
// list of a configuration and nothing else: the sources are read exactly
// as Parse reads them (same keys, same defaults), any other top-level key
// is refused, and so is a missing "sources". Like Parse it only checks
// the shape; what the sources mean is checked by config/validator.
//
// A manifest is a new format, so the settings Parse still reads with a
// "deprecated-setting" warning are errors here.
//
// Примеры:
//   - "sources: [{type: github, path: o/r}]" -> один источник, tree
//     "master", subpath ["skills"] (умолчания как в конфиге)
//   - "sources: []"                           -> ни одного источника
//   - "settings: {dry_run: true}\nsources: []" -> ошибка: ключ "settings"
//   - "target: {default: {}}"                 -> ошибка: ключ "target"
//   - ""                                      -> ошибка: не mapping
//   - "sources: [{path: a, name: x}]"         -> ошибка: sources[0].name
func ParseSourceManifest(data []byte) ([]model.SourceSpec, error) {
	root, _, err := decodeRoot(data, "source manifest")
	if err != nil {
		return nil, err
	}
	extra := []string{}
	for key := range root {
		if key != "sources" {
			extra = append(extra, key)
		}
	}
	if len(extra) > 0 {
		slices.Sort(extra)
		return nil, fmt.Errorf("a source manifest holds only \"sources\", found: %s", strings.Join(extra, ", "))
	}
	if _, ok := root["sources"]; !ok {
		return nil, fmt.Errorf("a source manifest must hold \"sources\"")
	}
	warnings := &deprecations{}
	specs, err := parseSources(root["sources"], warnings)
	if err != nil {
		return nil, err
	}
	if len(warnings.list) > 0 {
		w := warnings.list[0]
		return nil, fmt.Errorf("%s: %s", w.Setting, w.Message)
	}
	return specs, nil
}
