package build

import "testing"

// TestParseComposedReads pins the single-source parser: comment (#) and blank
// lines are skipped, interior whitespace collapses, and each remaining line is one
// read leaf. The drift guard and the agent artifact both key on this list, so a
// parse regression would mis-scope what the guard checks.
func TestParseComposedReads(t *testing.T) {
	in := "# header comment\n\nsearch\n  roles  \n\n# another\nsubrole-actors\n"
	got := parseComposedReads(in)
	want := []string{"search", "roles", "subrole-actors"}
	if len(got) != len(want) {
		t.Fatalf("parseComposedReads returned %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseComposedReads returned %v, want %v", got, want)
		}
	}
}

// TestAgentTools pins the tool-grant parser across the two conventions the
// interface allows — the inline comma form (feature-dev agents) and a YAML block
// list — so the read-only assertion (Bash present, Write/Edit absent) rests on a
// correctly parsed grant.
func TestAgentTools(t *testing.T) {
	cases := []struct {
		name  string
		front string
		want  []string
		ok    bool
	}{
		{"inline", "---\nname: n\ntools: Bash, Read, Grep, Glob\n---\nbody", []string{"Bash", "Read", "Grep", "Glob"}, true},
		{"bracketed", "---\ntools: [Write, Read]\n---\n", []string{"Write", "Read"}, true},
		{"block list", "---\nname: n\ntools:\n  - Bash\n  - Read\nmodel: inherit\n---\n", []string{"Bash", "Read"}, true},
		{"absent", "---\nname: n\ndescription: d\n---\n", nil, false},
		{"no frontmatter", "no front here\ntools: Bash\n", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := AgentTools(c.front)
			if ok != c.ok {
				t.Fatalf("AgentTools ok=%v, want %v", ok, c.ok)
			}
			if len(got) != len(c.want) {
				t.Fatalf("AgentTools = %v, want %v", got, c.want)
			}
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Fatalf("AgentTools = %v, want %v", got, c.want)
				}
			}
		})
	}
}

// TestCheckNavigationDrift proves the guard is not fail-open — it reports the
// offending leaf when a composed read no longer exists in the CLI, and when the
// agent artifact stops naming a composed leaf (the prose drifting from the single
// source). A guard that stayed silent here would let a stale artifact ship as
// authoritative.
func TestCheckNavigationDrift(t *testing.T) {
	live := []string{"search", "roles", "tree", "fillers", "subrole-actors", "domains", "policies"}
	agentNamesAll := "reads: `search` `roles` `tree` `fillers` `subrole-actors` `domains` `policies`"

	// Truthful: every composed leaf exists and is named.
	if d := CheckNavigationDrift(live, live, agentNamesAll); len(d) != 0 {
		t.Fatalf("expected no drift for a truthful path, got: %v", d)
	}

	// (a) a composed leaf the CLI dropped is reported by name.
	composed := append([]string{"gone"}, live...)
	d := CheckNavigationDrift(composed, live, agentNamesAll+" `gone`")
	if len(d) == 0 {
		t.Fatal("drift guard stayed silent when a composed leaf no longer exists in the CLI")
	}
	if !containsFold(joinDrift(d), "gone") {
		t.Fatalf("drift finding did not name the offending leaf: %v", d)
	}

	// (b) the agent prose dropping a composed leaf is reported.
	d = CheckNavigationDrift(live, live, "reads: only `search` here")
	if len(d) == 0 {
		t.Fatal("drift guard stayed silent when the agent stopped naming composed leaves")
	}
}

// TestLiveTopLevelCommands_ComposedLeavesResolve is a focused positive check that
// the extraction resolves the exact read leaves the navigation path composes to
// real top-level commands — a regression in the app.go/cli-source parsing that
// dropped one would otherwise surface only as a confusing drift-guard failure.
func TestLiveTopLevelCommands_ComposedLeavesResolve(t *testing.T) {
	live, err := LiveTopLevelCommands()
	if err != nil {
		t.Fatalf("LiveTopLevelCommands: %v", err)
	}
	set := map[string]bool{}
	for _, r := range live {
		set[r] = true
	}
	for _, leaf := range []string{"search", "roles", "tree", "fillers", "subrole-actors", "domains", "policies"} {
		if !set[leaf] {
			t.Errorf("LiveTopLevelCommands did not resolve the composed read leaf %q; got %v", leaf, live)
		}
	}
}
