package build

import (
	"strings"
	"testing"
)

// validConfigYAML is the exact-four-targets, cgo-disabled config plus the 022
// release sections (archives/checksum/release) and the 036 Homebrew additions
// (archives.builds_info.mtime, release.disable, the brews entry) the guard must
// accept. The drift cases below mutate copies of this baseline so each test
// changes exactly one thing. It mirrors the shipped .goreleaser.yaml
// structurally; TestConfigGuard_RealConfig pins the real file separately.
const validConfigYAML = `
version: 2
project_name: glassfrog
builds:
  - id: glassfrog
    main: .
    binary: glassfrog
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64
    flags:
      - -trimpath
    ldflags:
      - ""
archives:
  - id: glassfrog
    ids:
      - glassfrog
    formats:
      - tar.gz
    name_template: "glassfrog_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    builds_info:
      mtime: "{{ .CommitDate }}"
checksum:
  algorithm: sha256
  name_template: "glassfrog_{{ .Version }}_checksums.txt"
release:
  disable: true
brews:
  - name: glassfrog
    repository:
      owner: Luscii
      name: homebrew-cli-glassfrog
      branch: main
      token: '{{ .Env.HOMEBREW_TAP_TOKEN }}'
`

// TestConfigGuard_RealConfig is the change-detector against the shipped
// .goreleaser.yaml: the guard must pass on the actual repository config. This
// is the task's "passes on exactly the four supported targets with cgo
// disabled" acceptance, run against the real file rather than a fixture, so a
// future edit to .goreleaser.yaml that drifts the matrix fails here.
func TestConfigGuard_RealConfig(t *testing.T) {
	cfg, path, err := LoadConfig()
	if err != nil {
		t.Fatalf("loading the build config: %v", err)
	}
	if violations := CheckConfigGuard(cfg); len(violations) != 0 {
		t.Fatalf("the shipped %s must pass the config guard, got violations:\n  %s",
			path, strings.Join(violations, "\n  "))
	}
}

// TestConfigGuard_Drift exercises the guard against in-memory mutated configs.
// Change-detector rigor: a missing target must fail as loudly as an extra one,
// and a re-enabled cgo must fail. Each case asserts the violation NAMES the
// offending element so the message is actionable.
func TestConfigGuard_Drift(t *testing.T) {
	cases := []struct {
		name      string
		yaml      string
		wantPass  bool
		wantNamed []string // substrings every reported violation set must contain (when failing)
	}{
		{
			name:     "exactly the four supported targets with cgo disabled passes",
			yaml:     validConfigYAML,
			wantPass: true,
		},
		{
			name: "an unsupported OS target (windows) is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"      - darwin\n      - linux\n",
				"      - darwin\n      - linux\n      - windows\n", 1),
			wantPass:  false,
			wantNamed: []string{"windows"},
		},
		{
			name:      "cgo re-enabled (CGO_ENABLED=1) is rejected",
			yaml:      strings.Replace(validConfigYAML, "CGO_ENABLED=0", "CGO_ENABLED=1", 1),
			wantPass:  false,
			wantNamed: []string{"cgo must remain disabled"},
		},
		{
			// A later CGO_ENABLED=1 wins (duplicate env key → last value), so the
			// guard must scan every entry, not stop at the first CGO_ENABLED=0.
			name: "a later CGO_ENABLED=1 after CGO_ENABLED=0 is rejected",
			yaml: strings.Replace(validConfigYAML,
				"      - CGO_ENABLED=0\n", "      - CGO_ENABLED=0\n      - CGO_ENABLED=1\n", 1),
			wantPass:  false,
			wantNamed: []string{"cgo must remain disabled"},
		},
		{
			name: "CGO_ENABLED absent entirely is rejected",
			yaml: strings.Replace(validConfigYAML,
				"    env:\n      - CGO_ENABLED=0\n", "", 1),
			wantPass:  false,
			wantNamed: []string{"cgo must remain disabled"},
		},
		{
			name: "a missing OS target (linux dropped) fails as loudly as an extra",
			yaml: strings.Replace(validConfigYAML,
				"      - darwin\n      - linux\n", "      - darwin\n", 1),
			wantPass:  false,
			wantNamed: []string{"linux", "missing"},
		},
		{
			name: "a missing architecture target (arm64 dropped) is named",
			yaml: strings.Replace(validConfigYAML,
				"      - amd64\n      - arm64\n", "      - amd64\n", 1),
			wantPass:  false,
			wantNamed: []string{"arm64", "missing"},
		},
		{
			// 022 release sections — a missing section must fail as loudly as an
			// extra target. Dropping the entire archives section is the loudest miss.
			name: "a missing archives section is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"archives:\n  - id: glassfrog\n    ids:\n      - glassfrog\n    formats:\n      - tar.gz\n    name_template: \"glassfrog_{{ .Version }}_{{ .Os }}_{{ .Arch }}\"\n",
				"", 1),
			wantPass:  false,
			wantNamed: []string{"archives"},
		},
		{
			name: "an archive format other than tar.gz is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"    formats:\n      - tar.gz\n", "    formats:\n      - zip\n", 1),
			wantPass:  false,
			wantNamed: []string{"archives format", "tar.gz"},
		},
		{
			name: "a disabled checksum section is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"checksum:\n  algorithm: sha256\n", "checksum:\n  disable: true\n  algorithm: sha256\n", 1),
			wantPass:  false,
			wantNamed: []string{"checksum"},
		},
		{
			name: "a non-sha256 checksum algorithm is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"  algorithm: sha256\n", "  algorithm: sha512\n", 1),
			wantPass:  false,
			wantNamed: []string{"algorithm", "sha256"},
		},
		{
			// 036 — the release must be disabled (strict form of keep-existing); a
			// false disable fails as loudly as a missing section.
			name: "release disable: false is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"release:\n  disable: true\n", "release:\n  disable: false\n", 1),
			wantPass:  false,
			wantNamed: []string{"disable: true"},
		},
		{
			// 036 — a completely missing release section parses as the zero value
			// (Disable false); it must fail as loudly as an explicit disable: false.
			name: "a missing release section entirely is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"release:\n  disable: true\n", "", 1),
			wantPass:  false,
			wantNamed: []string{"disable: true"},
		},
		{
			// 036 reproducibility — an absent builds_info (no pinned mtime) must
			// fail; an unpinned mtime makes the tap job's rebuilt archive's sha256
			// diverge from the published asset's.
			name: "an unpinned archive mtime (builds_info absent) is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"    builds_info:\n      mtime: \"{{ .CommitDate }}\"\n", "", 1),
			wantPass:  false,
			wantNamed: []string{"builds_info.mtime"},
		},
		{
			// 036 — an empty mtime string is the same hole as an absent builds_info.
			name: "an empty archive mtime is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"      mtime: \"{{ .CommitDate }}\"\n", "      mtime: \"\"\n", 1),
			wantPass:  false,
			wantNamed: []string{"builds_info.mtime"},
		},
		{
			// 036 — a non-empty but NON-deterministic mtime (build-time {{ .Now }})
			// passes a presence-only check yet breaks cross-job reproducibility; the
			// guard must reject anything not commit-anchored.
			name: "a non-deterministic build-time archive mtime ({{ .Now }}) is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"      mtime: \"{{ .CommitDate }}\"\n", "      mtime: \"{{ .Now }}\"\n", 1),
			wantPass:  false,
			wantNamed: []string{"builds_info.mtime", "commit-anchored"},
		},
		{
			// #2 — a name_template rename silently breaks downstream consumers; pin it.
			name: "an archive name_template rename is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				`name_template: "glassfrog_{{ .Version }}_{{ .Os }}_{{ .Arch }}"`,
				`name_template: "glassfrog-{{ .Os }}-{{ .Arch }}"`, 1),
			wantPass:  false,
			wantNamed: []string{"archives name_template"},
		},
		{
			// #2 — pointing the archive at a different build must fail.
			name: "an archive drawing from a different build id is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"    ids:\n      - glassfrog\n", "    ids:\n      - other\n", 1),
			wantPass:  false,
			wantNamed: []string{"archives must draw from", "glassfrog"},
		},
		{
			// #1 — a completely missing checksum section parses as the zero value;
			// it must fail as loudly as an explicit bad algorithm.
			name: "a missing checksum section entirely is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"checksum:\n  algorithm: sha256\n  name_template: \"glassfrog_{{ .Version }}_checksums.txt\"\n",
				"", 1),
			wantPass:  false,
			wantNamed: []string{"checksum section must be present"},
		},
		{
			// #1 — the checksum filename is part of the consumer contract.
			name: "a checksum name_template rename is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				`name_template: "glassfrog_{{ .Version }}_checksums.txt"`,
				`name_template: "checksums.txt"`, 1),
			wantPass:  false,
			wantNamed: []string{"checksum name_template"},
		},
		{
			// 036 — a blanked brews block ships no Homebrew formula; it must fail as
			// loudly as a missing release section (feature scenario: "Config-guard
			// fails when the brews block is blanked or retargeted").
			name: "a blanked brews block (no entry) is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"brews:\n  - name: glassfrog\n    repository:\n      owner: Luscii\n      name: homebrew-cli-glassfrog\n      branch: main\n      token: '{{ .Env.HOMEBREW_TAP_TOKEN }}'\n",
				"", 1),
			wantPass:  false,
			wantNamed: []string{"brews section must declare exactly one"},
		},
		{
			// 036 — the v0.2.2 failure. GoReleaser's brew publisher validates the
			// token template's SHAPE, so the functionally identical `index` form is
			// rejected at publish time. It survived three releases only because the
			// tap job had never run (publish failed first), which is exactly why the
			// guard has to hold it rather than the comment.
			name: "a brews token using the index .Env form is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"token: '{{ .Env.HOMEBREW_TAP_TOKEN }}'",
				`token: '{{ index .Env "HOMEBREW_TAP_TOKEN" }}'`, 1),
			wantPass:  false,
			wantNamed: []string{"repository.token must be exactly"},
		},
		{
			// A blanked token would make the brew push fail to authenticate.
			name: "a brews entry with no token at all is rejected",
			yaml: strings.Replace(validConfigYAML,
				"      token: '{{ .Env.HOMEBREW_TAP_TOKEN }}'\n", "", 1),
			wantPass:  false,
			wantNamed: []string{"repository.token must be exactly"},
		},
		{
			// 036 — a retargeted tap repo would push the formula to the wrong place.
			name: "a retargeted brews tap repo name is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"      name: homebrew-cli-glassfrog\n", "      name: homebrew-something-else\n", 1),
			wantPass:  false,
			wantNamed: []string{"brews repository.name", "homebrew-cli-glassfrog"},
		},
		{
			// 036 — a retargeted tap owner is the same hazard as a wrong repo name.
			name: "a retargeted brews tap owner is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"      owner: Luscii\n", "      owner: SomeoneElse\n", 1),
			wantPass:  false,
			wantNamed: []string{"brews repository.owner", "Luscii"},
		},
		{
			// 036 — a wrong formula name breaks `brew install glassfrog`.
			name: "a wrong brews formula name is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"brews:\n  - name: glassfrog\n", "brews:\n  - name: notglassfrog\n", 1),
			wantPass:  false,
			wantNamed: []string{"brews formula name", "glassfrog"},
		},
		{
			// 036 — a tap branch other than main would push the formula where
			// Homebrew never reads it; a missing branch (zero value) fails the same.
			name: "a retargeted brews tap branch is rejected and named",
			yaml: strings.Replace(validConfigYAML,
				"      branch: main\n", "      branch: develop\n", 1),
			wantPass:  false,
			wantNamed: []string{"brews repository.branch", "main"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := ParseConfig([]byte(tc.yaml))
			if err != nil {
				t.Fatalf("parsing fixture config: %v", err)
			}
			violations := CheckConfigGuard(cfg)
			if tc.wantPass {
				if len(violations) != 0 {
					t.Fatalf("expected the config to pass, got violations:\n  %s",
						strings.Join(violations, "\n  "))
				}
				return
			}
			if len(violations) == 0 {
				t.Fatalf("expected the config to fail the guard, but it passed")
			}
			joined := strings.Join(violations, "\n")
			for _, want := range tc.wantNamed {
				if !strings.Contains(joined, want) {
					t.Fatalf("guard violation must name %q, got:\n%s", want, joined)
				}
			}
		})
	}
}
