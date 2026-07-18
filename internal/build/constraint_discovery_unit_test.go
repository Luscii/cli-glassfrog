package build

import "testing"

// TestParseComposedReads_CommandPathLeaf pins the one parsing property 065
// adds over 064's registry: a two-token command-path leaf ("me roles") keeps
// its single-space form through the whitespace collapse, so the drift guard
// and the agent artifact key on the same leaf identity.
func TestParseComposedReads_CommandPathLeaf(t *testing.T) {
	got := parseComposedReads("# header\nsearch\n  me   roles  \n")
	want := []string{"search", "me roles"}
	if len(got) != len(want) {
		t.Fatalf("parseComposedReads returned %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseComposedReads returned %v, want %v", got, want)
		}
	}
}

// TestCheckConstraintDrift proves the guard is not fail-open — it reports the
// offending leaf when a composed read no longer exists in the CLI (top-level
// and `me <sub>` alike), when the registry carries a command path the guard
// has no anchor for, and when the agent artifact stops naming a composed leaf
// (the prose drifting from the single source). A guard that stayed silent here
// would let a stale artifact ship as authoritative.
func TestCheckConstraintDrift(t *testing.T) {
	liveTop := []string{"search", "roles", "tree", "domains", "policies", "policy"}
	liveMe := []string{"actions", "projects", "roles"}
	composed := []string{"search", "roles", "tree", "domains", "policies", "policy", "me roles"}
	agentNamesAll := "reads: `search` `roles` `tree` `domains` `policies` `policy` `me roles`"

	// Truthful: every composed leaf exists and is named.
	if d := CheckConstraintDrift(composed, liveTop, liveMe, agentNamesAll); len(d) != 0 {
		t.Fatalf("expected no drift for a truthful path, got: %v", d)
	}

	// (a) a top-level composed leaf the CLI dropped is reported by name.
	d := CheckConstraintDrift(append([]string{"gone"}, composed...), liveTop, liveMe, agentNamesAll+" `gone`")
	if len(d) == 0 {
		t.Fatal("drift guard stayed silent when a top-level composed leaf no longer exists in the CLI")
	}
	if !containsFold(joinDrift(d), "gone") {
		t.Fatalf("drift finding did not name the offending leaf: %v", d)
	}

	// (b) a dropped `me` subcommand leaf is reported by name.
	d = CheckConstraintDrift(composed, liveTop, []string{"actions", "projects"}, agentNamesAll)
	if len(d) == 0 {
		t.Fatal("drift guard stayed silent when the `me roles` leaf no longer exists in the CLI")
	}
	if !containsFold(joinDrift(d), "me roles") {
		t.Fatalf("drift finding did not name the offending `me roles` leaf: %v", d)
	}

	// An unanchorable command path is reported, not silently skipped.
	d = CheckConstraintDrift([]string{"proposal list"}, liveTop, liveMe, "`proposal list`")
	if len(d) == 0 {
		t.Fatal("drift guard stayed silent on a command path it cannot anchor")
	}
	if !containsFold(joinDrift(d), "cannot anchor") {
		t.Fatalf("unanchorable-path finding does not say the leaf cannot be anchored: %v", d)
	}

	// (c) the agent prose dropping a composed leaf is reported.
	d = CheckConstraintDrift(composed, liveTop, liveMe, "reads: only `search` here")
	if len(d) == 0 {
		t.Fatal("drift guard stayed silent when the agent stopped naming composed leaves")
	}
}

// TestLiveMeSubcommands_RolesResolves is a focused positive check that the
// `me`-subcommand extraction resolves the exact leaf the constraint discovery
// path composes ("me roles") — a regression in the app.go/cli-source parsing
// that dropped it would otherwise surface only as a confusing drift-guard
// failure.
func TestLiveMeSubcommands_RolesResolves(t *testing.T) {
	liveMe, err := LiveMeSubcommands()
	if err != nil {
		t.Fatalf("LiveMeSubcommands: %v", err)
	}
	for _, sub := range liveMe {
		if sub == "roles" {
			return
		}
	}
	t.Errorf("LiveMeSubcommands did not resolve the `roles` subcommand; got %v", liveMe)
}

// TestLiveTopLevelCommands_ConstraintLeavesResolve is the 065 sibling of the
// 064 positive check: every single-token leaf the constraint discovery path
// composes resolves to a real top-level command (notably `policy`, which 064's
// registry does not carry).
func TestLiveTopLevelCommands_ConstraintLeavesResolve(t *testing.T) {
	live, err := LiveTopLevelCommands()
	if err != nil {
		t.Fatalf("LiveTopLevelCommands: %v", err)
	}
	set := map[string]bool{}
	for _, r := range live {
		set[r] = true
	}
	for _, leaf := range []string{"search", "roles", "tree", "domains", "policies", "policy"} {
		if !set[leaf] {
			t.Errorf("LiveTopLevelCommands did not resolve the composed read leaf %q; got %v", leaf, live)
		}
	}
}
