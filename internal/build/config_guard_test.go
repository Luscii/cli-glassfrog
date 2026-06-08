package build

import (
	"strings"
	"testing"
)

// validConfigYAML is the exact-four-targets, cgo-disabled config the guard must
// accept. The drift cases below mutate copies of this baseline so each test
// changes exactly one thing.
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
