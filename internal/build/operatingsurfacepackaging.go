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

// SetupSkillPath is the glassfrog-setup provisioning skill (plan ADR-3/-4):
// a caller-context skill — deliberately NOT a thin-skill+subagent path; setup
// provisions the environment, it is not an operator path — that instructs a
// presence check and an auth check and routes each failure to the CLI's
// existing fix. Added additively beside the seven sibling skills; the plugin
// manifest is untouched (skills stay directory-discovered).
const SetupSkillPath = "plugin/skills/glassfrog-setup/SKILL.md"

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

// --- Setup-skill anchors (T003) ---------------------------------------------
//
// The glassfrog-setup skill is instructed knowledge, not shipped code (plan
// ADR-4): it names an auth-check command and three install channels the repo
// ships elsewhere. Those are the skill's ENUMERABLE facts, and each is
// anchored best-effort to its in-repo source (plan ADR-5) so the hand-authored
// content cannot silently direct an operator at a dead channel or a dropped
// command. Detail beyond the names — flags, script options, formula internals
// — deliberately defers to `glassfrog <command> --help`, the README, and the
// orientation skill, keeping the drift surface small.

// In-repo sources the setup skill's enumerable facts are anchored against.
const (
	// setupInstallScriptSource is the install-script channel's artifact (027).
	setupInstallScriptSource = "install.sh"
	// setupHomebrewSource carries the Homebrew tap's formula publisher — the
	// `brews` section (036).
	setupHomebrewSource = ".goreleaser.yaml"
	// setupNPMSource is the npm wrapper package whose `name` is the install
	// coordinate the skill quotes (037).
	setupNPMSource = "npm/package.json"
)

// SetupAuthCheckCommand is the low-cost authenticated identity read the setup
// skill instructs as its auth check (plan ADR-4, command confirmed against the
// shipped CLI: the bare `me` identity read). Pinned as a checked-in contract
// fact — the skill's instruction and the registry anchor must name the same
// leaf — and resolved against the CLI's command registry by the guard, so a
// dropped or renamed `me` surfaces as drift.
const SetupAuthCheckCommand = "glassfrog me"

// ReadSetupSkill reads the committed glassfrog-setup SKILL.md
// (frontmatter + body).
func ReadSetupSkill() (string, error) {
	raw, err := readRepoFile(SetupSkillPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// SetupSkillName is the frontmatter name the skill must carry, derived from
// its directory (the host discovers skills by directory convention, so the
// directory is the identity's single source).
func SetupSkillName() string {
	return path.Base(path.Dir(SetupSkillPath))
}

// SkillFrontmatterField extracts a top-level scalar field (`name:`,
// `description:`) from a skill's YAML frontmatter. The second return is false
// when the frontmatter block or the field is absent.
func SkillFrontmatterField(content, field string) (string, bool) {
	front, ok := frontmatterBlock(content)
	if !ok {
		return "", false
	}
	for _, line := range strings.Split(front, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, field+":") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, field+":")), true
		}
	}
	return "", false
}

// SetupSkillAnchors is the in-repo state the setup skill's enumerable facts
// are checked against, extracted from the repository at test time.
type SetupSkillAnchors struct {
	// InstallScript reports whether the install-script channel's artifact
	// (install.sh) is still shipped at the repo root.
	InstallScript bool
	// HomebrewTap reports whether the Homebrew formula publisher (the
	// .goreleaser.yaml `brews` section) is still configured.
	HomebrewTap bool
	// NPMPackage is the npm wrapper's package name (from npm/package.json) —
	// the exact install coordinate the skill must quote.
	NPMPackage string
	// MeResolves reports whether the auth-check leaf (`me`) still resolves in
	// the CLI command registry. The bare `me` is variable-wired on root, so a
	// non-empty me-subcommand surface is the proof of registration
	// (LiveMeSubcommands errors when the root registration is gone).
	MeResolves bool
}

// LiveSetupSkillAnchors extracts the current in-repo anchors.
func LiveSetupSkillAnchors() (SetupSkillAnchors, error) {
	var a SetupSkillAnchors

	if _, err := readRepoFile(setupInstallScriptSource); err == nil {
		a.InstallScript = true
	}

	brews, err := readRepoFile(setupHomebrewSource)
	if err != nil {
		return a, fmt.Errorf("could not read %s: %w", setupHomebrewSource, err)
	}
	a.HomebrewTap = strings.Contains(string(brews), "brews:")

	npmRaw, err := readRepoFile(setupNPMSource)
	if err != nil {
		return a, fmt.Errorf("could not read %s: %w", setupNPMSource, err)
	}
	var npmPkg struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(npmRaw, &npmPkg); err != nil {
		return a, fmt.Errorf("could not decode %s: %w", setupNPMSource, err)
	}
	a.NPMPackage = npmPkg.Name

	subs, err := LiveMeSubcommands()
	a.MeResolves = err == nil && len(subs) > 0

	return a, nil
}

// CheckSetupSkillDrift returns one finding per way the setup skill's
// enumerable facts have diverged from their in-repo sources. Empty means
// truthful. It checks the facts' presence and resolution only — the journey
// prose (check → fix → verify, the two failure classes kept distinct) has no
// machine source and stays a review + BDD concern.
func CheckSetupSkillDrift(skill string, anchors SetupSkillAnchors) []string {
	var findings []string

	// Frontmatter: the host triggers the skill through its description; the
	// name must match the discovered directory.
	name, ok := SkillFrontmatterField(skill, "name")
	if !ok || name != SetupSkillName() {
		findings = append(findings, fmt.Sprintf("setup skill frontmatter name %q does not match its directory-derived identity %q (%s)", name, SetupSkillName(), SetupSkillPath))
	}
	if desc, ok := SkillFrontmatterField(skill, "description"); !ok || strings.TrimSpace(desc) == "" {
		findings = append(findings, fmt.Sprintf("setup skill frontmatter carries no non-empty description — the skill would never trigger (%s)", SetupSkillPath))
	}

	// The three install channels: each must still exist in-repo AND be named
	// by the skill — a channel dropped from either side is drift.
	if !anchors.InstallScript {
		findings = append(findings, fmt.Sprintf("the install-script channel's artifact (%s) is gone from the repo — the setup skill directs operators at a dead channel", setupInstallScriptSource))
	}
	if !strings.Contains(skill, setupInstallScriptSource) {
		findings = append(findings, fmt.Sprintf("the setup skill no longer names the install-script channel (%s)", setupInstallScriptSource))
	}
	if !anchors.HomebrewTap {
		findings = append(findings, fmt.Sprintf("the Homebrew formula publisher (`brews` section in %s) is gone — the setup skill directs operators at a dead channel", setupHomebrewSource))
	}
	if !strings.Contains(strings.ToLower(skill), "homebrew") {
		findings = append(findings, "the setup skill no longer names the Homebrew tap channel")
	}
	if strings.TrimSpace(anchors.NPMPackage) == "" {
		findings = append(findings, fmt.Sprintf("the npm wrapper package name could not be read from %s — the npm channel cannot be anchored", setupNPMSource))
	} else if !strings.Contains(skill, anchors.NPMPackage) {
		findings = append(findings, fmt.Sprintf("the setup skill no longer quotes the npm wrapper's install coordinate %q (%s)", anchors.NPMPackage, setupNPMSource))
	}

	// The auth-check command: the skill must instruct the pinned identity
	// read, and its leaf must still resolve in the CLI command registry.
	if !strings.Contains(skill, SetupAuthCheckCommand) {
		findings = append(findings, fmt.Sprintf("the setup skill no longer instructs the auth check %q", SetupAuthCheckCommand))
	}
	if !anchors.MeResolves {
		findings = append(findings, fmt.Sprintf("the auth-check leaf (`me`) no longer resolves in the CLI command registry (anchor: %s) — the setup skill instructs a command the CLI dropped", cliWiringSource))
	}

	return findings
}
