package build

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Operator Orientation (062) ships a Claude plugin — a manifest plus one skill —
// that packages the cross-cutting knowledge an agent needs to drive the glassfrog
// CLI (output formats, pagination, exit codes, credentials, write-safety). The
// plugin is committed, hand-authored content, not generated; it adds no code to
// the CLI. This file gives internal/build read + validation access to that
// artifact so the BDD suite can assert it is well-formed and the drift guard can
// keep its enumerable facts truthful to the shipped CLI.
//
// internal/build stays cli-free by deliberate convention (see VersionInjectionTarget):
// the orientation's CLI anchors are matched as strings against the CLI sources,
// the same discipline the version-injection guard uses, rather than importing
// internal/cli / internal/output and inverting the dependency.

// Repo-relative locations of the plugin artifact (forward-slash; joined through
// filepath so the guard is OS-agnostic).
const (
	// OrientationManifestPath is the plugin manifest the host loads to discover
	// the skill. A malformed manifest leaves the plugin unloadable.
	OrientationManifestPath = "plugin/.claude-plugin/plugin.json"

	// OrientationSkillPath is the one orientation skill the agent consults on
	// demand. Its frontmatter description is the trigger surface.
	OrientationSkillPath = "plugin/skills/glassfrog-operator/SKILL.md"

	// OrientationPluginDir is the plugin package root. Distribution (#70) extends
	// this directory; it is not added here.
	OrientationPluginDir = "plugin"
)

// OrientationManifest is the subset of the Claude plugin manifest this guard
// validates. Skills are auto-discovered from skills/, so the manifest carries no
// `skills` array — only identity and discovery metadata.
type OrientationManifest struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	Author      *ManifestAuthor `json:"author"`
	Keywords    []string        `json:"keywords"`
}

// ManifestAuthor is the plugin author object ({name, url?}), matching the
// sibling-plugin convention (score/prelude) of an object rather than a string.
type ManifestAuthor struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// ParseOrientationManifest decodes raw plugin.json bytes into the validated
// manifest shape. A decode error is exactly the "malformed manifest → unloadable
// plugin" condition: the host cannot register the skill, and — because the plugin
// tree carries no Go code (see OrientationPluginHasNoGoCode) — nothing in the CLI
// is affected; the agent simply falls back to rediscovery.
func ParseOrientationManifest(raw []byte) (OrientationManifest, error) {
	var m OrientationManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return OrientationManifest{}, err
	}
	return m, nil
}

// ReadOrientationManifest reads and decodes the committed plugin manifest.
func ReadOrientationManifest() (OrientationManifest, []byte, error) {
	raw, err := readRepoFile(OrientationManifestPath)
	if err != nil {
		return OrientationManifest{}, nil, err
	}
	m, err := ParseOrientationManifest(raw)
	return m, raw, err
}

// ReadOrientationSkill reads the committed SKILL.md content (frontmatter + body).
func ReadOrientationSkill() (string, error) {
	raw, err := readRepoFile(OrientationSkillPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ManifestDemandsNoSetup reports whether the manifest avoids every key that would
// force configuration beyond the CLI's existing credential setup — it declares no
// MCP servers, hooks, or commands of its own. The orientation is knowledge +
// packaging: once the plugin is present, the only thing an agent needs is the
// CLI's own `auth login`, nothing more. The raw bytes are inspected (not the
// decoded struct) so an unknown demanding key cannot slip through the lenient
// decode.
func ManifestDemandsNoSetup(raw []byte) bool {
	var keyed map[string]json.RawMessage
	if json.Unmarshal(raw, &keyed) != nil {
		return false
	}
	for _, forbidden := range []string{"mcpServers", "hooks", "commands", "agents"} {
		if _, present := keyed[forbidden]; present {
			return false
		}
	}
	return true
}

// OrientationPluginHasNoGoCode reports whether the plugin tree is pure data — it
// contains no .go file. This is what makes a malformed manifest inert to the CLI:
// nothing under plugin/ participates in the CLI build, so a load failure there
// can never break a glassfrog command.
func OrientationPluginHasNoGoCode() (bool, error) {
	root, err := RepoRoot()
	if err != nil {
		return false, err
	}
	clean := true
	walkErr := filepath.WalkDir(filepath.Join(root, OrientationPluginDir), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".go") {
			clean = false
		}
		return nil
	})
	return clean, walkErr
}

// readRepoFile reads a repo-relative file, resolving the repo root the same way
// the other build guards do (walk up to go.mod).
func readRepoFile(rel string) ([]byte, error) {
	root, err := RepoRoot()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
}
