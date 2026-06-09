// Package build holds the executable verification of the project's build and
// release contract:
//   - the GoReleaser configuration declares exactly the four supported,
//     cgo-disabled targets and the 022 release sections (archives/checksum/
//     release) — the config-guard;
//   - a produced glassfrog binary runs on a clean host of its target depending
//     only on OS-provided libraries — the self-containment check;
//   - the release workflow triggers, gates, and publishes as specified — the
//     release-workflow guard (022).
//
// There is no runtime component here — the "system" under test is the build and
// release pipeline itself (021 build matrix + self-containment; 022 packaging,
// release workflow, and cross-target verification gate).
package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

// ConfigFileName is the GoReleaser configuration the guard reads. GoReleaser v2
// uses the .yaml extension by default.
const ConfigFileName = ".goreleaser.yaml"

// SupportedGoos and SupportedGoarch are the exact, closed sets the build matrix
// must declare — their cross product is the four supported targets
// (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64). The guard treats a
// missing entry as loudly as an extra one (change-detector rigor): adding
// windows fails, dropping linux fails.
var (
	SupportedGoos   = []string{"darwin", "linux"}
	SupportedGoarch = []string{"amd64", "arm64"}
)

// ArchiveFormat is the single archive format the guard requires (one tar.gz per
// target). ChecksumAlgorithm is the single checksum algorithm required. Both are
// closed: a drift away from these values fails the guard as loudly as a missing
// section, mirroring the build matrix's change-detector rigor.
const (
	ArchiveFormat     = "tar.gz"
	ChecksumAlgorithm = "sha256"
	// ReleaseMode is the only accepted `release.mode`: keep-existing never
	// replaces an existing release's body, so a direct `goreleaser release` can
	// not clobber #30's notes or the publisher's pre-release/latest status.
	ReleaseMode = "keep-existing"

	// ArchiveNameTemplate and ChecksumNameTemplate are the exact name templates
	// downstream consumers (#27 install script, #36 Homebrew, #37 npm) depend on.
	// The guard pins them verbatim so a rename — which would silently break those
	// consumers — fails here. ArchiveBuildID is the build the archive must draw
	// from (021's single build); pinning it stops an archive from being pointed at
	// a different (or future) build.
	ArchiveNameTemplate  = "glassfrog_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
	ChecksumNameTemplate = "glassfrog_{{ .Version }}_checksums.txt"
	ArchiveBuildID       = "glassfrog"
)

// Config is the subset of the GoReleaser schema the guard inspects. Fields the
// guard does not assert on (project_name, flags, ldflags) are kept so a round
// trip is lossless enough for debugging and so the ldflags 023 seam is visible.
//
// 022 adds Archives/Checksum/Release; the guard asserts those sections are
// present and unchanged alongside 021's build matrix.
type Config struct {
	Version     int       `json:"version"`
	ProjectName string    `json:"project_name"`
	Builds      []Build   `json:"builds"`
	Archives    []Archive `json:"archives"`
	Checksum    Checksum  `json:"checksum"`
	Release     Release   `json:"release"`
}

// Archive mirrors a single GoReleaser archives entry. GoReleaser v2 renamed the
// single `format` field to a `formats` list; both are captured so the guard can
// read either spelling and fail clearly if neither yields tar.gz.
type Archive struct {
	ID           string   `json:"id"`
	IDs          []string `json:"ids"`
	Formats      []string `json:"formats"`
	Format       string   `json:"format"`
	NameTemplate string   `json:"name_template"`
}

// Checksum mirrors the GoReleaser checksum section. Disable is captured so the
// guard fails if checksums are turned off (the integrity mechanism is the only
// one — there is no signing).
type Checksum struct {
	Algorithm    string `json:"algorithm"`
	NameTemplate string `json:"name_template"`
	Disable      bool   `json:"disable"`
}

// Release mirrors the GoReleaser release section. The guard pins Mode to
// keep-existing and Draft to false, and requires Prerelease/MakeLatest to be
// absent, so a direct `goreleaser release` honors the existing release's body
// AND its pre-release/latest status (022's honor-not-decide contract).
//
// Prerelease/MakeLatest are modeled as interface{} so the guard can distinguish
// "absent" (nil — the only acceptable state) from "set to anything" (a non-nil
// value, including the string "auto" or a bool). They are NOT typed bool/string:
// a typed field could not tell an explicit `prerelease: false` from an omitted
// one, and the contract is that 022 sets neither.
type Release struct {
	Mode       string      `json:"mode"`
	Draft      bool        `json:"draft"`
	Prerelease interface{} `json:"prerelease"`
	MakeLatest interface{} `json:"make_latest"`
}

// Build mirrors a single GoReleaser builds entry.
type Build struct {
	ID      string   `json:"id"`
	Main    string   `json:"main"`
	Binary  string   `json:"binary"`
	Env     []string `json:"env"`
	Goos    []string `json:"goos"`
	Goarch  []string `json:"goarch"`
	Flags   []string `json:"flags"`
	Ldflags []string `json:"ldflags"`
}

// VersionInjectionTarget is the linker symbol the build must stamp the version
// into (spec 023): the package-level var in internal/cli that resolveVersion
// reads as its highest-precedence source. The config-guard below fails if
// builds.ldflags no longer injects it, so a blanked ldflags seam or a drifted
// symbol path is caught at config level rather than silently shipping a release
// that reports the build-info/placeholder value. Importing internal/cli here is
// avoided deliberately (it would invert the dependency and the build package
// stays cli-free); the symbol is matched as a string, the same way the linker
// consumes it.
const VersionInjectionTarget = "github.com/Luscii/cli-glassfrog/internal/cli.version"

// CheckVersionInjection returns the list of guard violations for version
// embedding: an empty result means the single build's ldflags inject
// VersionInjectionTarget via a syntactically valid `-X` flag. It does not pin
// the value (that is GoReleaser's template, not ours) — only that the seam
// targets the right symbol through a real `-X` form, which is the regression
// that matters (the 021/022 config-guard ignores ldflags entirely).
func CheckVersionInjection(cfg Config) []string {
	if len(cfg.Builds) != 1 {
		return []string{fmt.Sprintf(
			"build matrix must be a single builds entry, found %d", len(cfg.Builds))}
	}
	for _, entry := range cfg.Builds[0].Ldflags {
		if ldflagsInjectVersion(entry) {
			return nil
		}
	}
	return []string{fmt.Sprintf(
		"builds.ldflags must inject the version via -X %s=…, but no such flag is present "+
			"(the 023 seam is blank or the symbol path drifted)", VersionInjectionTarget)}
}

// ldflagsInjectVersion reports whether a single ldflags entry contains a
// syntactically valid `-X` flag stamping VersionInjectionTarget. It tokenizes
// the entry (entries may bundle several space-separated flags, e.g.
// "-s -w -X sym=val") and accepts only the two forms the Go linker actually
// parses for `-X`:
//
//	-X <sym>=<val>   (the flag and its argument as separate tokens)
//	-X=<sym>=<val>   (the flag and its argument joined by '=')
//
// The concatenated `-X<sym>=<val>` form is rejected — the linker does not parse
// it as `-X`, so accepting it would let a broken seam pass the guard. Matching
// the `-X` token structure (rather than a bare substring) also avoids a false
// positive from the symbol appearing in some unrelated flag.
func ldflagsInjectVersion(entry string) bool {
	want := VersionInjectionTarget + "="
	tokens := strings.Fields(entry)
	for i, tok := range tokens {
		if arg, ok := strings.CutPrefix(tok, "-X="); ok {
			if strings.HasPrefix(arg, want) {
				return true
			}
			continue
		}
		if tok == "-X" && i+1 < len(tokens) && strings.HasPrefix(tokens[i+1], want) {
			return true
		}
	}
	return false
}

// RepoRoot walks up from the current working directory to the directory holding
// go.mod — the repository root, where .goreleaser.yaml lives. Tests in this
// package run with their package directory as the working directory, so the
// walk is what lets the guard find the config regardless of the caller's depth.
func RepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("locating repo root: %w", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("locating repo root: no go.mod found walking up from working directory")
		}
		dir = parent
	}
}

// LoadConfig reads and parses .goreleaser.yaml from the repository root. It
// returns the parsed config and its absolute path. A missing or unparseable
// config is an error — the RED state before T002 adds the file.
func LoadConfig() (Config, string, error) {
	root, err := RepoRoot()
	if err != nil {
		return Config{}, "", err
	}
	path := filepath.Join(root, ConfigFileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, path, fmt.Errorf("reading %s: %w", path, err)
	}
	cfg, err := ParseConfig(raw)
	if err != nil {
		return Config{}, path, err
	}
	return cfg, path, nil
}

// ParseConfig decodes GoReleaser YAML into Config. Split from LoadConfig so the
// config-guard and the BDD suite can exercise the guard against in-memory
// mutated configs without touching the filesystem.
func ParseConfig(raw []byte) (Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing %s: %w", ConfigFileName, err)
	}
	return cfg, nil
}

// CheckConfigGuard returns the list of guard violations for a parsed config. An
// empty result means the matrix is exactly the four supported targets with cgo
// disabled. Each message names the offending element so a reviewer (or the
// model) can correct the drift without re-reading the config.
//
// The guard enforces, with change-detector rigor (presence AND absence):
//
// 021 build matrix:
//   - exactly one builds entry (the single glassfrog build),
//   - CGO_ENABLED=0 present in that entry's env,
//   - the goos set is exactly SupportedGoos (no windows, none missing),
//   - the goarch set is exactly SupportedGoarch.
//
// 022 release sections (a missing section fails as loudly as an extra one):
//   - exactly one archives entry, format tar.gz,
//   - the checksum section is enabled with algorithm sha256,
//   - the release section is mode keep-existing, draft false.
func CheckConfigGuard(cfg Config) []string {
	var violations []string

	if len(cfg.Builds) != 1 {
		violations = append(violations, fmt.Sprintf(
			"build matrix must be a single builds entry, found %d", len(cfg.Builds)))
		// The build matrix is the precondition for everything else; bail here so
		// the message set stays focused on the structural break.
		return violations
	}
	b := cfg.Builds[0]

	violations = append(violations, checkCgoDisabled(b.Env)...)
	violations = append(violations, diffTargetSet("OS target", b.Goos, SupportedGoos)...)
	violations = append(violations, diffTargetSet("architecture target", b.Goarch, SupportedGoarch)...)

	violations = append(violations, checkArchives(cfg.Archives)...)
	violations = append(violations, checkChecksum(cfg.Checksum)...)
	violations = append(violations, checkRelease(cfg.Release)...)

	return violations
}

// checkArchives requires exactly one archives entry producing a single tar.gz,
// drawing from the glassfrog build, with the pinned name template. One archive
// per build target is GoReleaser's per-target fan-out of this one entry, so the
// entry count is one (not four). The format is read from either the v2 `formats`
// list or the legacy singular `format`. The name_template and ids are pinned
// because downstream consumers depend on the asset names and on the archive
// carrying 021's build — a rename or a re-pointed build would otherwise pass.
func checkArchives(archives []Archive) []string {
	if len(archives) != 1 {
		return []string{fmt.Sprintf(
			"archives section must declare exactly one archive entry (one tar.gz per target), found %d", len(archives))}
	}
	a := archives[0]
	var violations []string

	formats := a.Formats
	if len(formats) == 0 && a.Format != "" {
		formats = []string{a.Format}
	}
	if len(formats) != 1 || formats[0] != ArchiveFormat {
		violations = append(violations, fmt.Sprintf(
			"archives format must be exactly %q, got %v", ArchiveFormat, formats))
	}
	if a.NameTemplate != ArchiveNameTemplate {
		violations = append(violations, fmt.Sprintf(
			"archives name_template must be exactly %q (downstream consumers depend on the asset names), got %q",
			ArchiveNameTemplate, a.NameTemplate))
	}
	// The archive must draw from 021's single build. GoReleaser accepts the
	// build reference under `ids:` (v2) or the legacy singular `id:`; require the
	// glassfrog build via whichever is set, and reject any other/extra reference.
	ids := a.IDs
	if len(ids) == 0 && a.ID != "" {
		ids = []string{a.ID}
	}
	if len(ids) != 1 || ids[0] != ArchiveBuildID {
		violations = append(violations, fmt.Sprintf(
			"archives must draw from exactly the %q build (ids/id), got %v", ArchiveBuildID, ids))
	}
	return violations
}

// checkChecksum requires the checksum section to be present, enabled, sha256, and
// to carry the pinned name template. The algorithm is pinned to a non-empty
// "sha256" (NOT accepting empty-as-default): a completely missing `checksum:`
// section parses as the zero-value Checksum{}, and accepting an empty algorithm
// would let that missing section pass — exactly the change-detector hole the
// guard exists to close. The shipped config sets the algorithm explicitly.
func checkChecksum(c Checksum) []string {
	if c.Disable {
		return []string{"checksum section must not be disabled — the checksums file is the only integrity mechanism"}
	}
	var violations []string
	if c.Algorithm != ChecksumAlgorithm {
		violations = append(violations, fmt.Sprintf(
			"checksum section must be present with algorithm %q, got %q (an empty algorithm means the checksum section is missing)",
			ChecksumAlgorithm, c.Algorithm))
	}
	if c.NameTemplate != ChecksumNameTemplate {
		violations = append(violations, fmt.Sprintf(
			"checksum name_template must be exactly %q, got %q", ChecksumNameTemplate, c.NameTemplate))
	}
	return violations
}

// checkRelease pins the release section to keep-existing/draft:false and rejects
// any prerelease/make_latest override, so a direct `goreleaser release` honors
// the existing release's body AND its pre-release/latest status rather than
// authoring, replacing, or flipping them (022's honor-not-decide contract). A
// non-nil prerelease/make_latest means the field was set to something — the
// contract is that 022 sets neither, so any value (including "auto" or false) is
// a violation.
func checkRelease(r Release) []string {
	var violations []string
	if r.Mode != ReleaseMode {
		violations = append(violations, fmt.Sprintf(
			"release mode must be %q (never replace an existing release body), got %q", ReleaseMode, r.Mode))
	}
	if r.Draft {
		violations = append(violations,
			"release draft must be false — 022 attaches to an already-published release, it does not draft one")
	}
	if r.Prerelease != nil {
		violations = append(violations, fmt.Sprintf(
			"release must not set prerelease — it honors the published release's status (got %v)", r.Prerelease))
	}
	if r.MakeLatest != nil {
		violations = append(violations, fmt.Sprintf(
			"release must not set make_latest — it honors the published release's latest status (got %v)", r.MakeLatest))
	}
	return violations
}

// checkCgoDisabled enforces CGO_ENABLED=0. A self-contained binary is the whole
// point of the build (CONSTITUTION XII); a re-enabled cgo silently pulls in a C
// library dependency. The message says cgo must remain disabled so the
// config-drift scenario's expectation is met verbatim.
//
// It scans every CGO_ENABLED entry rather than stopping at the first: a config
// carrying both CGO_ENABLED=0 and a later CGO_ENABLED=1 would re-enable cgo (a
// duplicate env key resolves to the last value), so the guard fails on ANY
// non-zero assignment and passes only when at least one entry exists and all
// are 0.
func checkCgoDisabled(env []string) []string {
	seen := false
	for _, e := range env {
		name, value, found := strings.Cut(e, "=")
		if !found || strings.TrimSpace(name) != "CGO_ENABLED" {
			continue
		}
		seen = true
		if strings.TrimSpace(value) != "0" {
			return []string{fmt.Sprintf(
				"cgo must remain disabled: CGO_ENABLED must be 0, got %q", strings.TrimSpace(value))}
		}
	}
	if !seen {
		return []string{"cgo must remain disabled: CGO_ENABLED=0 is absent from the build env"}
	}
	return nil
}

// diffTargetSet compares a declared target dimension against its closed
// supported set, emitting a violation for every unsupported (extra) value and
// every missing required value. label distinguishes the OS and architecture
// dimensions in the message.
func diffTargetSet(label string, declared, supported []string) []string {
	var violations []string

	supportedSet := make(map[string]bool, len(supported))
	for _, s := range supported {
		supportedSet[s] = true
	}
	declaredSet := make(map[string]bool, len(declared))
	for _, d := range declared {
		declaredSet[d] = true
		if !supportedSet[d] {
			violations = append(violations, fmt.Sprintf("unsupported %s declared: %q", label, d))
		}
	}
	for _, s := range supported {
		if !declaredSet[s] {
			violations = append(violations, fmt.Sprintf("required %s missing: %q", label, s))
		}
	}
	return violations
}
