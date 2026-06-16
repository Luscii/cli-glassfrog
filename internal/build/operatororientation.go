package build

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

// --- Drift guard (T003) ----------------------------------------------------
//
// The drift guard pins the orientation's *enumerable* facts to their source of
// truth in the CLI, so the hand-authored skill cannot silently drift as the CLI
// evolves. It is best-effort and explicitly partial (spec assumption; plan
// ADR-4): it covers the closed format-token set, the published exit-code numbers,
// and the existence of `auth login`. It deliberately does NOT verify the prose
// reactions, the pagination mechanics, or the write-safety guidance — those have
// no machine source to anchor against and stay review concerns. That gap is
// stated here rather than left silent (no silent caps).
//
// build stays cli-free (see VersionInjectionTarget), so the anchors are read as
// strings from the CLI sources rather than by importing internal/cli /
// internal/output and inverting the dependency.

// CLI source files the orientation's enumerable facts are anchored against.
const (
	orientationFormatSource   = "internal/output/format.go"
	orientationExitCodeSource = "internal/cli/exitcode.go"
	orientationAuthSource     = "internal/cli/authcmd.go"
)

// OrientationFacts is the slice of CLI source-of-truth the orientation must
// mirror, extracted from the CLI sources.
type OrientationFacts struct {
	// FormatsRaw is the verbatim supportedFormats literal (e.g. "full, compact,
	// json, yaml"); Formats is the same set, tokenized.
	FormatsRaw string
	Formats    []string
	// ExitCodes maps each published code number to its constant label
	// (7 -> "StaleWrite"). Drawn from the exitcode.go const block — the single
	// source ExitCode maps an Outcome onto.
	ExitCodes map[int]string
	// AuthLogin reports whether the CLI still wires the `auth login` command.
	AuthLogin bool
}

// LiveOrientationFacts extracts the current CLI facts from source.
func LiveOrientationFacts() (OrientationFacts, error) {
	var f OrientationFacts
	f.ExitCodes = map[int]string{}

	fmtSrc, err := readRepoFile(orientationFormatSource)
	if err != nil {
		return f, err
	}
	m := regexp.MustCompile(`supportedFormats\s*=\s*"([^"]*)"`).FindSubmatch(fmtSrc)
	if m == nil {
		return f, fmt.Errorf("could not locate supportedFormats in %s", orientationFormatSource)
	}
	f.FormatsRaw = string(m[1])
	for _, tok := range strings.Split(f.FormatsRaw, ",") {
		if t := strings.TrimSpace(tok); t != "" {
			f.Formats = append(f.Formats, t)
		}
	}

	ecSrc, err := readRepoFile(orientationExitCodeSource)
	if err != nil {
		return f, err
	}
	for _, mm := range regexp.MustCompile(`code(\w+)\s*=\s*(\d+)`).FindAllStringSubmatch(string(ecSrc), -1) {
		n, convErr := strconv.Atoi(mm[2])
		if convErr != nil {
			continue
		}
		f.ExitCodes[n] = mm[1]
	}
	if len(f.ExitCodes) == 0 {
		return f, fmt.Errorf("could not locate exit-code constants in %s", orientationExitCodeSource)
	}

	authSrc, err := readRepoFile(orientationAuthSource)
	if err != nil {
		return f, err
	}
	f.AuthLogin = regexp.MustCompile(`Use:\s*"auth"`).Match(authSrc) &&
		regexp.MustCompile(`Use:\s*"login`).Match(authSrc)

	return f, nil
}

// CheckOrientationDrift returns one finding per enumerable fact in the skill that
// no longer matches the CLI facts. An empty result means the skill is truthful.
// Each finding names the offending anchor so a CI failure points straight at it.
//
// The comparison is bidirectional for the closed vocabularies: a token/code the
// CLI added but the skill omits, AND a token/code the skill documents but the CLI
// dropped, both count as drift.
func CheckOrientationDrift(skill string, facts OrientationFacts) []string {
	var drift []string

	// Output formats: compare the set the skill enumerates against the CLI set.
	skillFormats, ok := orientationSkillFormats(skill)
	if !ok {
		drift = append(drift, fmt.Sprintf("output formats: skill no longer carries the canonical %q enumeration (anchor: %s supportedFormats)", "tokens are exactly: …", orientationFormatSource))
	} else if missing, extra := diffSets(facts.Formats, skillFormats); len(missing) > 0 || len(extra) > 0 {
		drift = append(drift, fmt.Sprintf("output formats: skill set %v does not match the CLI's %v (missing %v, extra %v) (anchor: %s supportedFormats)", skillFormats, facts.Formats, missing, extra, orientationFormatSource))
	}

	// Exit codes: compare the code numbers the skill documents (backticked, e.g.
	// `7`) against the CLI's published code set.
	skillCodes := orientationSkillExitCodes(skill)
	cliCodes := make([]string, 0, len(facts.ExitCodes))
	for n := range facts.ExitCodes {
		cliCodes = append(cliCodes, strconv.Itoa(n))
	}
	if missing, extra := diffSets(cliCodes, skillCodes); len(missing) > 0 || len(extra) > 0 {
		drift = append(drift, fmt.Sprintf("exit codes: skill documents codes %v but the CLI publishes %v (missing %v, extra %v) (anchor: %s)", skillCodes, cliCodes, missing, extra, orientationExitCodeSource))
	}
	// The 412 anchor the write-safety guidance leans on: code 7 must be StaleWrite
	// and the skill must tie it to the 412.
	if label, has7 := facts.ExitCodes[7]; has7 && strings.EqualFold(label, "StaleWrite") && !strings.Contains(skill, "412") {
		drift = append(drift, fmt.Sprintf("exit codes: skill documents code 7 (%s) but never references the 412 it maps (anchor: %s)", label, orientationExitCodeSource))
	}

	// Credentials: the CLI must still wire `auth login`, and the skill must route
	// to it.
	if !facts.AuthLogin {
		drift = append(drift, fmt.Sprintf("credentials: the CLI no longer wires `auth login` (anchor: %s)", orientationAuthSource))
	}
	if !strings.Contains(skill, "glassfrog auth login") {
		drift = append(drift, fmt.Sprintf("credentials: skill never routes to `glassfrog auth login` (anchor: %s)", orientationAuthSource))
	}

	return drift
}

// orientationSkillFormats extracts the format tokens the skill presents as the
// supported set, from its canonical "supported tokens are exactly: a, b, c"
// enumeration. The second return is false when that anchor line is absent.
func orientationSkillFormats(skill string) ([]string, bool) {
	m := regexp.MustCompile(`(?i)tokens are exactly:\s*([a-z0-9, ]+)`).FindStringSubmatch(skill)
	if m == nil {
		return nil, false
	}
	var toks []string
	for _, t := range strings.Split(m[1], ",") {
		if t = strings.TrimSpace(t); t != "" {
			toks = append(toks, t)
		}
	}
	return toks, true
}

// orientationSkillExitCodes returns the distinct single-digit code numbers the
// skill documents in backticked form (`` `7` ``) — the exit-code table and its
// cross-references. The backtick form keeps the digit from colliding with an HTTP
// status or version number in prose.
func orientationSkillExitCodes(skill string) []string {
	seen := map[string]bool{}
	var codes []string
	for _, mm := range regexp.MustCompile("`(\\d)`").FindAllStringSubmatch(skill, -1) {
		if !seen[mm[1]] {
			seen[mm[1]] = true
			codes = append(codes, mm[1])
		}
	}
	return codes
}

// diffSets returns the elements of want missing from got, and the elements of got
// not in want — order-independent, used for the closed-vocabulary comparisons.
func diffSets(want, got []string) (missing, extra []string) {
	wantSet, gotSet := map[string]bool{}, map[string]bool{}
	for _, w := range want {
		wantSet[w] = true
	}
	for _, g := range got {
		gotSet[g] = true
	}
	for w := range wantSet {
		if !gotSet[w] {
			missing = append(missing, w)
		}
	}
	for g := range gotSet {
		if !wantSet[g] {
			extra = append(extra, g)
		}
	}
	return missing, extra
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
