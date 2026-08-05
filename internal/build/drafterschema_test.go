package build

import (
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// validDraftingWorkflowYAML is the baseline drafting workflow the coupling
// guard must accept: one step pinning the drafter action at a major at the
// schema floor. Drift cases mutate copies so each test changes exactly one
// thing; TestDrafterSchemaCoupling_RealConfig pins the real files separately.
const validDraftingWorkflowYAML = `
name: Release Drafting
on:
  push:
    branches: [main]
jobs:
  draft:
    runs-on: ubuntu-latest
    steps:
      - name: Draft the next release
        uses: release-drafter/release-drafter@v7.7.0
`

// TestDrafterSchemaCoupling_RealConfig is the change-detector against the
// shipped .github/release-drafter.yml and .github/workflows/release-drafting.yml:
// the coupling must pass on the actual repository files. A future edit that
// moves the config schema without the pinned action major (or vice versa)
// fails here — the silent-degradation window 071 exists to close.
func TestDrafterSchemaCoupling_RealConfig(t *testing.T) {
	rd, wf, err := LoadDrafterSchemaCoupling()
	if err != nil {
		t.Fatalf("loading the drafter schema-coupling inputs: %v", err)
	}
	if violations := CheckDrafterSchemaCoupling(rd, wf); len(violations) != 0 {
		t.Fatalf("the shipped config and drafting workflow must satisfy the coupling, got violations:\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// TestDrafterSchemaCoupling_Drift exercises the coupling guard against
// in-memory mutated fixtures. Each failing case asserts the violation NAMES the
// workflow file and the offending value, so the message is actionable.
func TestDrafterSchemaCoupling_Drift(t *testing.T) {
	cases := []struct {
		name              string
		drafter, workflow string // empty → use the valid baseline
		wantPass          bool
		wantNamed         []string // substrings the reported violation set must contain (when failing)
	}{
		{
			name:     "the current schema and a floor-satisfying pinned major pass",
			wantPass: true,
		},
		{
			// v7 and v7.7 must derive the same major as v7.7.0.
			name: "a bare major tag derives its major and passes",
			workflow: strings.Replace(validDraftingWorkflowYAML,
				"@v7.7.0", "@v7", 1),
			wantPass: true,
		},
		{
			name: "a major.minor tag derives its major and passes",
			workflow: strings.Replace(validDraftingWorkflowYAML,
				"@v7.7.0", "@v7.7", 1),
			wantPass: true,
		},
		{
			// The guarded direction: an action major behind the config schema.
			name: "a pinned major below the schema floor is rejected and named",
			workflow: strings.Replace(validDraftingWorkflowYAML,
				"@v7.7.0", "@v6.4.0", 1),
			wantPass: false,
			wantNamed: []string{"release-drafting.yml", "v6.4.0",
				"major 6", "major 7"},
		},
		{
			// A commit SHA carries no derivable major — a finding, not a pass.
			name: "a commit-SHA ref is rejected as underivable and named",
			workflow: strings.Replace(validDraftingWorkflowYAML,
				"@v7.7.0", "@1b38099d29f3144567d9aabbccddeeff00112233", 1),
			wantPass: false,
			wantNamed: []string{"release-drafting.yml",
				"1b38099d29f3144567d9aabbccddeeff00112233",
				"pinned major could not be determined"},
		},
		{
			// A branch ref is just as underivable.
			name: "a branch ref is rejected as underivable and named",
			workflow: strings.Replace(validDraftingWorkflowYAML,
				"@v7.7.0", "@main", 1),
			wantPass: false,
			wantNamed: []string{"release-drafting.yml", "main",
				"pinned major could not be determined"},
		},
		{
			// A tag without a leading vN (or with a non-numeric suffix glued to
			// the major) yields no major either.
			name: "a non-vN tag is rejected as underivable and named",
			workflow: strings.Replace(validDraftingWorkflowYAML,
				"@v7.7.0", "@release-7", 1),
			wantPass: false,
			wantNamed: []string{"release-drafting.yml", "release-7",
				"pinned major could not be determined"},
		},
		{
			// No drafter step at all: the guard's input is missing.
			name: "a workflow without the drafter action is rejected and named",
			workflow: strings.Replace(validDraftingWorkflowYAML,
				"uses: release-drafter/release-drafter@v7.7.0",
				"uses: actions/checkout@v4", 1),
			wantPass: false,
			wantNamed: []string{"release-drafting.yml",
				"release-drafter/release-drafter", "not a pass"},
		},
		{
			// A config on the superseded schema has no derivable floor — the
			// coupling verdict reports it rather than passing silently.
			name: "a superseded-schema config yields no derivable floor and is rejected",
			drafter: `
version-resolver:
  major:
    labels: [breaking]
  default: patch
categories:
  - title: "Features"
    labels: [features]
exclude-labels:
  - no-release-note
`,
			wantPass: false,
			wantNamed: []string{"release-drafter.yml",
				"cannot determine the schema floor"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rd := parseDrafter(t, orDefault(tc.drafter, validDrafterYAML))
			wf := parseDraftingWorkflow(t, orDefault(tc.workflow, validDraftingWorkflowYAML))

			violations := CheckDrafterSchemaCoupling(rd, wf)
			if tc.wantPass {
				if len(violations) != 0 {
					t.Fatalf("expected the coupling to pass, got violations:\n  %s",
						strings.Join(violations, "\n  "))
				}
				return
			}
			if len(violations) == 0 {
				t.Fatalf("expected the coupling to fail the guard, but it passed")
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

func parseDraftingWorkflow(t *testing.T, raw string) Workflow {
	t.Helper()
	var wf Workflow
	if err := yaml.Unmarshal([]byte(raw), &wf); err != nil {
		t.Fatalf("parsing fixture release-drafting.yml: %v", err)
	}
	return wf
}
