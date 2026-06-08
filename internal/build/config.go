// Package build holds the executable verification of the project's build
// contract: that the GoReleaser configuration declares exactly the four
// supported, cgo-disabled targets (the config-guard), and that a produced
// glassfrog binary runs on a clean host of its target depending only on
// OS-provided libraries (the self-containment check). There is no runtime
// component here — the "system" under test is the build itself (021).
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

// Config is the subset of the GoReleaser schema the guard inspects. Fields the
// guard does not assert on (project_name, flags, ldflags) are kept so a round
// trip is lossless enough for debugging and so the ldflags 023 seam is visible.
type Config struct {
	Version     int     `json:"version"`
	ProjectName string  `json:"project_name"`
	Builds      []Build `json:"builds"`
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
		return Config{}, path, fmt.Errorf("reading %s: %w", ConfigFileName, err)
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
//   - exactly one builds entry (the single glassfrog build),
//   - CGO_ENABLED=0 present in that entry's env,
//   - the goos set is exactly SupportedGoos (no windows, none missing),
//   - the goarch set is exactly SupportedGoarch.
func CheckConfigGuard(cfg Config) []string {
	var violations []string

	if len(cfg.Builds) != 1 {
		violations = append(violations, fmt.Sprintf(
			"build matrix must be a single builds entry, found %d", len(cfg.Builds)))
		return violations
	}
	b := cfg.Builds[0]

	violations = append(violations, checkCgoDisabled(b.Env)...)
	violations = append(violations, diffTargetSet("OS target", b.Goos, SupportedGoos)...)
	violations = append(violations, diffTargetSet("architecture target", b.Goarch, SupportedGoarch)...)

	return violations
}

// checkCgoDisabled enforces CGO_ENABLED=0. A self-contained binary is the whole
// point of the build (CONSTITUTION XII); a re-enabled cgo silently pulls in a C
// library dependency. The message says cgo must remain disabled so the
// config-drift scenario's expectation is met verbatim.
func checkCgoDisabled(env []string) []string {
	for _, e := range env {
		name, value, found := strings.Cut(e, "=")
		if !found || strings.TrimSpace(name) != "CGO_ENABLED" {
			continue
		}
		if strings.TrimSpace(value) == "0" {
			return nil
		}
		return []string{fmt.Sprintf(
			"cgo must remain disabled: CGO_ENABLED must be 0, got %q", strings.TrimSpace(value))}
	}
	return []string{"cgo must remain disabled: CGO_ENABLED=0 is absent from the build env"}
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
