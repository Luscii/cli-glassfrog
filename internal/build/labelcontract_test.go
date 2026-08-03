package build

import (
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// The three baseline configs the label-contract guard must accept (spec 030,
// ADR-6). They mirror the shipped files structurally; the drift cases below
// mutate copies so each test changes exactly one thing. TestLabelContract_RealConfig
// pins the real files separately.
const (
	validDrafterYAML = `
categories:
  - type: "pre-exclude"
    when:
      labels: [no-release-note]
  - title: "Breaking"
    when:
      labels: [breaking]
  - title: "Features"
    when:
      labels: [features]
  - title: "Fixes"
    when:
      labels: [fixes]
  - title: "Documentation"
    when:
      labels: [docs]
  - title: "Infrastructure"
    when:
      labels: [infrastructure]
  - title: "Dependencies"
    when:
      labels: [dependencies]
  - title: "Internal"
    when:
      labels: [internal]
  - type: "version-resolver"
    semver-increment: "major"
    when:
      labels: [breaking]
  - type: "version-resolver"
    semver-increment: "minor"
    when:
      labels: [features]
  - type: "version-resolver"
    semver-increment: "patch"
    when:
      labels: [fixes]
  - type: "version-resolver"
    semver-increment: "patch"
`

	validLabelerYAML = `
version: 1
labels:
  - label: breaking
  - label: features
  - label: fixes
  - label: docs
  - label: infrastructure
  - label: dependencies
  - label: internal
  - label: no-release-note
    negate: true
`

	validSettingsYAML = `
labels:
  - name: breaking
  - name: features
  - name: fixes
  - name: docs
  - name: infrastructure
  - name: dependencies
  - name: internal
  - name: no-release-note
`
)

// TestLabelContract_RealConfig is the change-detector against the shipped
// .github/release-drafter.yml, labeler.yml, and settings.yml: the guard must
// pass on the actual repository files. A future edit that drifts the label
// contract in one file without the others fails here.
func TestLabelContract_RealConfig(t *testing.T) {
	rd, labeler, settings, err := LoadLabelContract()
	if err != nil {
		t.Fatalf("loading the label-contract configs: %v", err)
	}
	if violations := CheckLabelContract(rd, labeler, settings); len(violations) != 0 {
		t.Fatalf("the shipped label contract must pass the guard, got violations:\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// TestLabelContract_Drift exercises the guard against in-memory mutated configs.
// Change-detector rigor: a missing label fails as loudly as an extra one, in any
// of the three files. Each failing case asserts the violation NAMES the offending
// file/section + label so the message is actionable.
func TestLabelContract_Drift(t *testing.T) {
	cases := []struct {
		name                       string
		drafter, labeler, settings string // empty → use the valid baseline
		wantPass                   bool
		wantNamed                  []string // substrings the reported violation set must contain (when failing)
	}{
		{
			name:     "all three files agree on the eight-label contract passes",
			wantPass: true,
		},
		{
			// A category label renamed in release-drafter but not the others.
			name: "a renamed release-drafter category label is rejected and named",
			drafter: strings.Replace(validDrafterYAML,
				"  - title: \"Features\"\n    when:\n      labels: [features]\n",
				"  - title: \"Features\"\n    when:\n      labels: [feature]\n", 1),
			wantPass:  false,
			wantNamed: []string{"release-drafter.yml category", "feature", "features"},
		},
		{
			// A category label dropped from labeler — must fail as loudly as an extra.
			// (Expected value is non-zero, but a zero-valued expectation would still
			// be caught: the guard uses set-difference, not a map-by-key lookup.)
			name: "a category label dropped from labeler is rejected and named",
			labeler: strings.Replace(validLabelerYAML,
				"  - label: internal\n", "", 1),
			wantPass:  false,
			wantNamed: []string{"labeler.yml category", "internal", "missing"},
		},
		{
			// An extra ninth label in settings — caught by both the category diff
			// (it's neither a category nor the exclusion label) and the count check.
			name: "an extra ninth label in settings is rejected and named",
			settings: strings.Replace(validSettingsYAML,
				"  - name: no-release-note\n", "  - name: no-release-note\n  - name: surprise\n", 1),
			wantPass:  false,
			wantNamed: []string{"settings.yml", "surprise", "exactly 8"},
		},
		{
			// The version-resolver major bucket drifts off `breaking`.
			name: "a drifted version-resolver major bucket is rejected and named",
			drafter: strings.Replace(validDrafterYAML,
				"    semver-increment: \"major\"\n    when:\n      labels: [breaking]\n",
				"    semver-increment: \"major\"\n    when:\n      labels: [major-change]\n", 1),
			wantPass:  false,
			wantNamed: []string{"version-resolver major", "major-change", "breaking"},
		},
		{
			// spec 030 requires the fallback bump to be patch; a drifted default
			// (the condition-less version-resolver category, since 071) must
			// fail as loudly as a drifted bucket. The anchor includes the patch
			// bucket above it — `semver-increment: "patch"` alone matches the
			// bucket entry first.
			name: "a drifted version-resolver default is rejected and named",
			drafter: strings.Replace(validDrafterYAML,
				"      labels: [fixes]\n  - type: \"version-resolver\"\n    semver-increment: \"patch\"\n",
				"      labels: [fixes]\n  - type: \"version-resolver\"\n    semver-increment: \"minor\"\n", 1),
			wantPass:  false,
			wantNamed: []string{"version-resolver default", "patch", "minor"},
		},
		{
			name: "no-release-note missing from settings is rejected and named",
			settings: strings.Replace(validSettingsYAML,
				"  - name: no-release-note\n", "", 1),
			wantPass:  false,
			wantNamed: []string{"settings.yml must define", "no-release-note"},
		},
		{
			name: "no-release-note missing from labeler is rejected and named",
			labeler: strings.Replace(validLabelerYAML,
				"  - label: no-release-note\n    negate: true\n", "", 1),
			wantPass:  false,
			wantNamed: []string{"labeler.yml must define", "no-release-note"},
		},
		{
			name: "no-release-note missing from the release-drafter pre-exclude category is rejected and named",
			drafter: strings.Replace(validDrafterYAML,
				"      labels: [no-release-note]\n", "      labels: []\n", 1),
			wantPass:  false,
			wantNamed: []string{"pre-exclude category must exclude", "no-release-note"},
		},
		{
			// Dropping a label entirely shrinks the managed set below eight.
			name: "a settings catalog of fewer than eight labels is rejected and named",
			settings: strings.Replace(validSettingsYAML,
				"  - name: dependencies\n", "", 1),
			wantPass:  false,
			wantNamed: []string{"settings.yml", "exactly 8"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rd := parseDrafter(t, orDefault(tc.drafter, validDrafterYAML))
			labeler := parseLabeler(t, orDefault(tc.labeler, validLabelerYAML))
			settings := parseSettings(t, orDefault(tc.settings, validSettingsYAML))

			violations := CheckLabelContract(rd, labeler, settings)
			if tc.wantPass {
				if len(violations) != 0 {
					t.Fatalf("expected the contract to pass, got violations:\n  %s",
						strings.Join(violations, "\n  "))
				}
				return
			}
			if len(violations) == 0 {
				t.Fatalf("expected the contract to fail the guard, but it passed")
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

func orDefault(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func parseDrafter(t *testing.T, raw string) ReleaseDrafterConfig {
	t.Helper()
	var rd ReleaseDrafterConfig
	if err := yaml.Unmarshal([]byte(raw), &rd); err != nil {
		t.Fatalf("parsing fixture release-drafter.yml: %v", err)
	}
	return rd
}

func parseLabeler(t *testing.T, raw string) LabelerConfig {
	t.Helper()
	var l LabelerConfig
	if err := yaml.Unmarshal([]byte(raw), &l); err != nil {
		t.Fatalf("parsing fixture labeler.yml: %v", err)
	}
	return l
}

func parseSettings(t *testing.T, raw string) SettingsConfig {
	t.Helper()
	var s SettingsConfig
	if err := yaml.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatalf("parsing fixture settings.yml: %v", err)
	}
	return s
}
