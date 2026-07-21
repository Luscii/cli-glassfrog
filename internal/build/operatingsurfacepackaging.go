package build

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"
)

// Operating-Surface Packaging (070) is the distribution vehicle for the Claude
// plugin that 062–069 built: a repo-root marketplace manifest
// (.claude-plugin/marketplace.json) a Claude plugin host reads when the repo is
// added as a marketplace, plus a `glassfrog-setup` provisioning skill inside
// the plugin. Like every deliverable in the family it adds NO code to the Go
// CLI — the artifacts are committed, hand-authored data. This file gives
// internal/build read + validation access so the BDD suite can assert the
// artifacts are well-formed and the drift guard can keep the marketplace entry
// truthful to the plugin it ships (plan ADR-5): entry identity is compared
// against `plugin/.claude-plugin/plugin.json` (OrientationManifestPath) with
// both sides read from disk at test time, never against a hard-coded copy.
//
// The marketplace is general in *shape* (plan ADR-2): a `plugins` list with
// one entry today, a future sibling appended rather than restructured. The
// helpers therefore locate the glassfrog entry by name within the list
// (existence, never "is the only entry") so additional entries cannot break
// the guard.

// Repo-relative locations of the packaging artifacts (forward-slash; joined
// through filepath so the reads are OS-agnostic).
const (
	// MarketplaceManifestPath is the marketplace manifest at the REPO ROOT —
	// where a Claude plugin host looks when the repository itself is added as
	// a marketplace (`/plugin marketplace add Luscii/cli-glassfrog`). The
	// plugin's own `plugin/.claude-plugin/` directory stays marketplace-free
	// (062's "deliberately absent" holds for that path).
	MarketplaceManifestPath = ".claude-plugin/marketplace.json"
)

// MarketplacePluginName is the identity operators install against —
// `/plugin install glassfrog@glassfrog` (plan ADR-1) — pinned as a checked-in
// contract fact because the documented install command depends on it. It is
// cross-checked against the plugin manifest's own `name` by the consistency
// guard (both read from disk), so this anchor can never silently diverge from
// the plugin it names.
const MarketplacePluginName = "glassfrog"

// MarketplaceManifest is the subset of the Claude marketplace manifest this
// guard validates: identity, ownership, and the plugins list. The host's full
// schema is deliberately not mirrored here — the guard checks in-repo
// consistency, not the host's marketplace schema.
type MarketplaceManifest struct {
	Schema  string                   `json:"$schema"`
	Name    string                   `json:"name"`
	Owner   MarketplaceOwner         `json:"owner"`
	Plugins []MarketplacePluginEntry `json:"plugins"`
}

// MarketplaceOwner is the marketplace owner object ({name, email?}), matching
// the installed-marketplace convention of an object rather than a string.
type MarketplaceOwner struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

// MarketplacePluginEntry is one entry of the manifest's `plugins` list. It is
// decoded through a custom UnmarshalJSON so the guard can see whether the raw
// entry carried a `version` key at all — the KEY'S PRESENCE is the defect
// (plan ADR-2: the source is in-repo, so the installed version is the
// checkout's and `plugin.json` stays the single version source), independent
// of whatever value it holds.
type MarketplacePluginEntry struct {
	Name        string
	Source      string
	Description string
	// HasVersion records whether the raw entry carried a "version" key.
	HasVersion bool
}

// UnmarshalJSON decodes the typed fields and records `version`-key presence
// from the raw key set, so a lenient struct decode cannot hide the pin.
func (e *MarketplacePluginEntry) UnmarshalJSON(raw []byte) error {
	var fields struct {
		Name        string `json:"name"`
		Source      string `json:"source"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	var keyed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keyed); err != nil {
		return err
	}
	_, hasVersion := keyed["version"]
	*e = MarketplacePluginEntry{
		Name:        fields.Name,
		Source:      fields.Source,
		Description: fields.Description,
		HasVersion:  hasVersion,
	}
	return nil
}

// ParseMarketplaceManifest decodes raw marketplace.json bytes into the
// validated manifest shape. A decode error — or a missing required field — is
// exactly the "host cannot add the marketplace" condition: nothing installs,
// and the guard fails first in CI.
func ParseMarketplaceManifest(raw []byte) (MarketplaceManifest, error) {
	var m MarketplaceManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return MarketplaceManifest{}, err
	}
	if strings.TrimSpace(m.Name) == "" {
		return MarketplaceManifest{}, fmt.Errorf("marketplace manifest is missing the required %q field", "name")
	}
	if strings.TrimSpace(m.Owner.Name) == "" {
		return MarketplaceManifest{}, fmt.Errorf("marketplace manifest is missing the required %q field", "owner.name")
	}
	if len(m.Plugins) == 0 {
		return MarketplaceManifest{}, fmt.Errorf("marketplace manifest carries no %q entries", "plugins")
	}
	return m, nil
}

// ReadMarketplaceManifest reads and decodes the committed repo-root
// marketplace manifest, returning the raw bytes alongside for shape checks
// that inspect the JSON itself.
func ReadMarketplaceManifest() (MarketplaceManifest, []byte, error) {
	raw, err := readRepoFile(MarketplaceManifestPath)
	if err != nil {
		return MarketplaceManifest{}, nil, err
	}
	m, err := ParseMarketplaceManifest(raw)
	return m, raw, err
}

// FindMarketplacePlugin locates the named entry within the manifest's plugins
// list. The lookup is existence-based — never "is the only entry" — so the
// marketplace's ADR-2 generality (a future sibling is one appended entry) is
// preserved by construction.
func FindMarketplacePlugin(m MarketplaceManifest, name string) (MarketplacePluginEntry, bool) {
	for _, entry := range m.Plugins {
		if entry.Name == name {
			return entry, true
		}
	}
	return MarketplacePluginEntry{}, false
}

// MarketplaceSourcePluginManifest resolves a marketplace entry's relative
// `source` against the repo root and reads the plugin manifest that directory
// must contain (`<source>/.claude-plugin/plugin.json`). An error is exactly
// the "entry no longer resolves to a plugin definition" defect: install fails
// at the host, and the guard goes red first.
func MarketplaceSourcePluginManifest(source string) (OrientationManifest, error) {
	if strings.TrimSpace(source) == "" {
		return OrientationManifest{}, fmt.Errorf("marketplace entry carries an empty source")
	}
	rel := path.Join(source, ".claude-plugin", "plugin.json")
	raw, err := readRepoFile(rel)
	if err != nil {
		return OrientationManifest{}, fmt.Errorf("marketplace entry source %q does not resolve to a directory containing a plugin manifest: %w", source, err)
	}
	m, err := ParseOrientationManifest(raw)
	if err != nil {
		return OrientationManifest{}, fmt.Errorf("marketplace entry source %q resolves to an invalid plugin manifest: %w", source, err)
	}
	return m, nil
}

// CheckMarketplaceEntryConsistency returns one finding per way the marketplace
// entry has drifted from the plugin manifest it ships (plan ADR-5). Empty
// means consistent. Both sides come in from disk reads at the call site —
// nothing here restates the guarded values — so a finding is always a real
// divergence between the two committed files. Each finding names the offending
// field so a CI failure points straight at it.
func CheckMarketplaceEntryConsistency(entry MarketplacePluginEntry, plugin OrientationManifest) []string {
	var findings []string

	if entry.Name != plugin.Name {
		findings = append(findings, fmt.Sprintf("marketplace entry name %q does not match the plugin manifest's name %q (%s ↔ %s) — the install identity drifted; reconcile the two manifests", entry.Name, plugin.Name, MarketplaceManifestPath, OrientationManifestPath))
	}
	if entry.Description != plugin.Description {
		findings = append(findings, fmt.Sprintf("marketplace entry description is not verbatim-equal to the plugin manifest's description (%s ↔ %s) — the listing would describe a different plugin than it ships; copy the plugin manifest's description", MarketplaceManifestPath, OrientationManifestPath))
	}
	if entry.HasVersion {
		findings = append(findings, fmt.Sprintf("marketplace entry %q carries a %q key — the plugin version is single-sourced in the plugin manifest (%s); the in-repo source installs the checkout's version, so remove the pin", entry.Name, "version", OrientationManifestPath))
	}

	return findings
}
