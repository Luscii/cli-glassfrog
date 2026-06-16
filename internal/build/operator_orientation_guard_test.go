package build

import "testing"

// TestOperatorOrientationDriftGuard is the best-effort drift guard for Operator
// Orientation (062, plan ADR-4). It pins the orientation skill's *enumerable*
// facts to their source of truth in the CLI so the hand-authored content cannot
// silently drift as the CLI evolves: documented behaviour that no longer matches
// the shipped CLI is a defect, not a difference (spec).
//
// COVERAGE (explicitly partial, per the spec assumption — stated, not silent):
//   - output-format token set — must equal internal/output supportedFormats
//     exactly (additions, removals, renames all caught);
//   - published exit-code numbers — the skill's documented set must equal the
//     code constants in internal/cli/exitcode.go, with code 7 = StaleWrite tied
//     to the 412 the write-safety section leans on;
//   - the `auth login` command must still exist.
//
// NOT COVERED (no machine source to anchor against; left to review): the prose
// exit-code reactions, the pagination mechanics, the credential precedence, and
// the write-safety guidance wording. The guard asserts these facts are *present
// and consistent*, not that the surrounding prose is correct.
func TestOperatorOrientationDriftGuard(t *testing.T) {
	skill, err := ReadOrientationSkill()
	if err != nil {
		t.Fatalf("could not read the orientation skill: %v", err)
	}
	facts, err := LiveOrientationFacts()
	if err != nil {
		t.Fatalf("could not extract the CLI facts the orientation anchors against: %v", err)
	}

	// Sanity-pin the extracted anchors so a regression in extraction (not just in
	// the skill) also fails loudly.
	if facts.FormatsRaw != "full, compact, json, yaml" {
		t.Errorf("CLI supportedFormats is %q; the orientation guard assumed \"full, compact, json, yaml\" — update both together", facts.FormatsRaw)
	}
	if label, ok := facts.ExitCodes[7]; !ok || label != "StaleWrite" {
		t.Errorf("exit code 7 is %q (present=%v); the orientation's 412 guidance anchors on codeStaleWrite=7", label, ok)
	}
	if !facts.AuthLogin {
		t.Error("the CLI no longer wires `auth login`; the orientation's credential section points at `glassfrog auth login`")
	}

	// The integrated check: every enumerable fact the skill states still matches
	// the CLI. Each finding names its offending anchor.
	if drift := CheckOrientationDrift(skill, facts); len(drift) != 0 {
		t.Fatalf("orientation drifted from the shipped CLI:\n  - %s", joinDrift(drift))
	}
}

func joinDrift(drift []string) string {
	out := ""
	for i, d := range drift {
		if i > 0 {
			out += "\n  - "
		}
		out += d
	}
	return out
}
