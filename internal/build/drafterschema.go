package build

import (
	"fmt"
	"strconv"
	"strings"
)

// DraftingWorkflowFileName is the release-drafting workflow the coupling guard
// reads, relative to the repository root. NOT WorkflowFileName — that is 022's
// release.yml, the publish pipeline; this is 030's drafting workflow. The two
// must not be confused.
const DraftingWorkflowFileName = ".github/workflows/release-drafting.yml"

// DrafterActionRepo is the release-drafter action's repository. The drafting
// step is located by this `uses:` prefix; the pinned ref after the `@` is the
// guard's derived input.
const DrafterActionRepo = "release-drafter/release-drafter"

// DrafterSchemaMinMajor is the lowest release-drafter action major that reads
// the schema .github/release-drafter.yml is written in (071): typed categories
// with `when` conditions, a `pre-exclude` category, and `version-resolver`
// categories in place of the top-level key. The property this pins: an action
// major BEHIND the config's schema does not reject the config — it accepts it
// and silently ignores the positions it does not recognise, so drafting runs
// green while producing miscategorised notes and a wrong bump. Nothing at
// runtime reveals that, so this floor is the only signal. Hand-edited at the
// next schema migration, deliberately — updating it is the same act as writing
// that migration (the GoReleaserVersion precedent in workflow.go: write the
// reason next to the pin so the next person re-derives instead of copying).
const DrafterSchemaMinMajor = 7

// LoadDrafterSchemaCoupling reads the drafter config and the drafting workflow
// from the repository root — the two halves of the coupling verdict. It shares
// labelcontract.go's ReleaseDrafterConfig parse and workflow.go's Workflow
// type (whose `On` field is tagged json:"true" because YAML coerces the `on:`
// key — already handled and documented there). A missing or unparseable file
// is an error.
func LoadDrafterSchemaCoupling() (ReleaseDrafterConfig, Workflow, error) {
	root, err := RepoRoot()
	if err != nil {
		return ReleaseDrafterConfig{}, Workflow{}, err
	}
	var rd ReleaseDrafterConfig
	if err := loadYAML(root, ReleaseDrafterFileName, &rd); err != nil {
		return ReleaseDrafterConfig{}, Workflow{}, err
	}
	var wf Workflow
	if err := loadYAML(root, DraftingWorkflowFileName, &wf); err != nil {
		return ReleaseDrafterConfig{}, Workflow{}, err
	}
	return rd, wf, nil
}

// CheckDrafterSchemaCoupling returns the coupling violations: the drafting
// workflow's pinned action major must be at or above the floor the config's
// schema requires. An empty result means the pinned action understands the
// committed config. Both sides are derived — the floor from the config's own
// shape, the major from the workflow's pinned ref — so neither is a
// hand-editable literal standing in for a derivable value (071 ADR-5). The
// guard is deliberately one-directional: it catches a config NEWER than the
// action (silent miscategorisation), not the reverse (which degrades noisily
// through the action's compatibility warnings).
func CheckDrafterSchemaCoupling(rd ReleaseDrafterConfig, wf Workflow) []string {
	var violations []string

	floor, floorOK := drafterSchemaFloor(rd)
	if !floorOK {
		violations = append(violations, fmt.Sprintf(
			"%s: cannot determine the schema floor the config requires — the contract is not declared through typed/conditioned categories (a config on the superseded schema must be migrated before the coupling verdict can hold)",
			ReleaseDrafterFileName))
	}

	step, found := drafterStep(wf)
	if !found {
		violations = append(violations, fmt.Sprintf(
			"%s has no step using %s — the coupling guard's input is missing, which is a violation, not a pass",
			DraftingWorkflowFileName, DrafterActionRepo))
		return violations
	}

	ref := strings.TrimPrefix(step.Uses, DrafterActionRepo+"@")
	major, majorOK := parsePinnedMajor(ref)
	if !majorOK {
		violations = append(violations, fmt.Sprintf(
			"%s pins %s at %q: the pinned major could not be determined (a branch, commit SHA, or non-vN ref carries no derivable major) — underivable is a finding, not a pass",
			DraftingWorkflowFileName, DrafterActionRepo, ref))
		return violations
	}

	if floorOK && major < floor {
		violations = append(violations, fmt.Sprintf(
			"%s pins %s at %q (major %d), below the floor the release-drafter.yml schema requires (major %d) — an action major behind the schema accepts the config and silently ignores the positions it does not recognise",
			DraftingWorkflowFileName, DrafterActionRepo, ref, major, floor))
	}

	return violations
}

// drafterSchemaFloor derives the action-major floor the config's schema
// requires from the config's own shape: a config declaring its contract
// through typed or conditioned categories is on the 071 schema, whose floor
// DrafterSchemaMinMajor pins. A config still on the superseded shape — or one
// declaring no contract at all — has no derivable floor, which the caller
// reports rather than passing silently.
func drafterSchemaFloor(rd ReleaseDrafterConfig) (int, bool) {
	if len(drafterLegacyShape(rd)) > 0 {
		return 0, false
	}
	for _, c := range rd.Categories {
		if c.When != nil || categoryType(c) != CategoryTypeChangelog {
			return DrafterSchemaMinMajor, true
		}
	}
	return 0, false
}

// drafterStep locates the step whose `uses:` pins the drafter action, across
// all jobs.
func drafterStep(wf Workflow) (Step, bool) {
	for _, job := range wf.Jobs {
		for _, s := range job.Steps {
			if strings.HasPrefix(s.Uses, DrafterActionRepo+"@") {
				return s, true
			}
		}
	}
	return Step{}, false
}

// parsePinnedMajor derives the major version from a pinned ref: a leading `v`
// followed by digits, terminated by the end of the ref or a `.` — so v7, v7.7,
// and v7.7.0 all yield 7. Anything else (a branch name, a commit SHA, a tag
// without a leading vN) yields no major: ok is false and the caller must treat
// that as a violation, never a default or a pass.
func parsePinnedMajor(ref string) (major int, ok bool) {
	if !strings.HasPrefix(ref, "v") {
		return 0, false
	}
	rest := ref[1:]
	i := 0
	for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, false
	}
	if i < len(rest) && rest[i] != '.' {
		return 0, false
	}
	major, err := strconv.Atoi(rest[:i])
	if err != nil {
		return 0, false
	}
	return major, true
}
