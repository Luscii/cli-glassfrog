package build

import "testing"

// TestParseTensionCommands pins the single-source parser: comment (#) and blank
// lines are skipped, interior whitespace collapses, and each remaining line is
// one composed leaf. The drift guard and the agent artifact both key on this
// list, so a parse regression would mis-scope what the guard checks.
func TestParseTensionCommands(t *testing.T) {
	in := "# header comment\n\ntension list\n  tension   create  \n\n# another\ntension subroles\n"
	got := parseTensionCommands(in)
	want := []string{"tension list", "tension create", "tension subroles"}
	if len(got) != len(want) {
		t.Fatalf("parseTensionCommands returned %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseTensionCommands returned %v, want %v", got, want)
		}
	}
}

// TestCheckTensionProcessingDrift proves the guard is not fail-open — for each
// way the composed set can drift (a leaf the CLI dropped, a leaf that collides
// with 063's gated set, a gated tension leaf, a non-tension leaf leaking in, the
// agent prose dropping a leaf) it reports a finding that names the offender. A
// guard that stayed silent on any of these would let a stale or boundary-breaking
// artifact ship as authoritative.
func TestCheckTensionProcessingDrift(t *testing.T) {
	live := []string{"create", "discard", "get", "list", "subroles", "update"}
	gated := []string{"proposal create", "proposal propose", "proposal respond", "proposal withdraw"}
	composed := []string{"tension list", "tension get", "tension subroles", "tension create", "tension update", "tension discard"}
	agentNamesAll := "leaves: `tension list` `tension get` `tension subroles` `tension create` `tension update` `tension discard`"

	// Truthful: every composed leaf exists, none is gated, all are named.
	if d := CheckTensionProcessingDrift(composed, live, gated, agentNamesAll); len(d) != 0 {
		t.Fatalf("expected no drift for a truthful path, got: %v", d)
	}

	t.Run("a leaf the CLI dropped is reported by name", func(t *testing.T) {
		withGone := append([]string{"tension gone"}, composed...)
		d := CheckTensionProcessingDrift(withGone, live, gated, agentNamesAll+" `tension gone`")
		if len(d) == 0 {
			t.Fatal("drift guard stayed silent when a composed leaf no longer exists in the CLI")
		}
		if !containsFold(joinDrift(d), "tension gone") {
			t.Fatalf("drift finding did not name the offending leaf: %v", d)
		}
	})

	t.Run("a composed leaf entering the gated set is reported", func(t *testing.T) {
		gatedPlusTension := append([]string{"tension create"}, gated...)
		d := CheckTensionProcessingDrift(composed, live, gatedPlusTension, agentNamesAll)
		if len(d) == 0 {
			t.Fatal("drift guard stayed silent when a composed leaf appeared in 063's gated set — the ungated invariant broke unnoticed")
		}
		if !containsFold(joinDrift(d), "tension create") {
			t.Fatalf("drift finding did not name the gated collision: %v", d)
		}
	})

	t.Run("a gated tension leaf outside the composed set is still reported", func(t *testing.T) {
		// Even a tension leaf the artifacts do not compose must never be gated —
		// 063's Behavioral Accord keeps ALL operational tension edits ungated.
		gatedPlusOther := append([]string{"tension archive"}, gated...)
		d := CheckTensionProcessingDrift(composed, live, gatedPlusOther, agentNamesAll)
		if len(d) == 0 {
			t.Fatal("drift guard stayed silent when a tension leaf entered the gated registry")
		}
		if !containsFold(joinDrift(d), "tension archive") {
			t.Fatalf("drift finding did not name the gated tension leaf: %v", d)
		}
	})

	t.Run("a non-tension leaf leaking into the composed set is reported", func(t *testing.T) {
		withProposal := append([]string{"proposal propose"}, composed...)
		d := CheckTensionProcessingDrift(withProposal, live, gated, agentNamesAll)
		if len(d) == 0 {
			t.Fatal("drift guard stayed silent when a proposal leaf leaked into the composed set")
		}
		if !containsFold(joinDrift(d), "proposal propose") {
			t.Fatalf("drift finding did not name the leaked leaf: %v", d)
		}
	})

	t.Run("the agent prose dropping a composed leaf is reported", func(t *testing.T) {
		d := CheckTensionProcessingDrift(composed, live, gated, "leaves: only `tension list` here")
		if len(d) == 0 {
			t.Fatal("drift guard stayed silent when the agent stopped naming composed leaves")
		}
	})
}

// TestLiveTensionSubcommands_ComposedLeavesResolve is a focused positive check
// that the extraction resolves the exact subcommands the processing path
// composes to real leaves of the CLI's tension command — a regression in the
// tension.go/cli-source parsing that dropped one would otherwise surface only as
// a confusing drift-guard failure.
func TestLiveTensionSubcommands_ComposedLeavesResolve(t *testing.T) {
	live, err := LiveTensionSubcommands()
	if err != nil {
		t.Fatalf("LiveTensionSubcommands: %v", err)
	}
	set := map[string]bool{}
	for _, s := range live {
		set[s] = true
	}
	for _, sub := range []string{"list", "get", "subroles", "create", "update", "discard"} {
		if !set[sub] {
			t.Errorf("LiveTensionSubcommands did not resolve the composed subcommand %q; got %v", sub, live)
		}
	}
}
