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
// keep-existing and Draft to false so a direct `goreleaser release` honors the
// existing release's body and status (022's honor-not-decide contract).
type Release struct {
	Mode  string `json:"mode"`
	Draft bool   `json:"draft"`
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

// checkArchives requires exactly one archives entry producing a single tar.gz.
// One archive per build target is GoReleaser's per-target fan-out of this one
// entry, so the entry count is one (not four). The format is read from either
// the v2 `formats` list or the legacy singular `format`; anything other than a
// lone tar.gz fails, naming the archives section.
func checkArchives(archives []Archive) []string {
	if len(archives) != 1 {
		return []string{fmt.Sprintf(
			"archives section must declare exactly one archive entry (one tar.gz per target), found %d", len(archives))}
	}
	formats := archives[0].Formats
	if len(formats) == 0 && archives[0].Format != "" {
		formats = []string{archives[0].Format}
	}
	if len(formats) != 1 || formats[0] != ArchiveFormat {
		return []string{fmt.Sprintf(
			"archives format must be exactly %q, got %v", ArchiveFormat, formats)}
	}
	return nil
}

// checkChecksum requires the checksum section to be enabled (not disabled) and
// to use sha256. An empty algorithm is GoReleaser's sha256 default, so it is
// accepted; any other explicit value fails. The message names the checksum
// section so drift is actionable.
func checkChecksum(c Checksum) []string {
	if c.Disable {
		return []string{"checksum section must not be disabled — the checksums file is the only integrity mechanism"}
	}
	if c.Algorithm != "" && c.Algorithm != ChecksumAlgorithm {
		return []string{fmt.Sprintf(
			"checksum algorithm must be %q, got %q", ChecksumAlgorithm, c.Algorithm)}
	}
	return nil
}

// checkRelease pins the release section to keep-existing/draft:false so a direct
// `goreleaser release` honors the existing release's body and status rather than
// authoring or replacing them.
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
