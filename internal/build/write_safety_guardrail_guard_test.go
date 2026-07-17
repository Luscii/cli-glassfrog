package build

import (
	"sort"
	"strings"
	"testing"
)

// TestGateHelpPathAndRegistryDriven locks in the two refinements from PR #150
// review triage: (1) a bare `proposal` or a help/usage path such as
// `proposal --help` writes nothing, so it must pass ungated rather than being
// fail-closed gated on the absence of a subcommand leaf; and (2) the gated
// classification is driven by the single-sourced registry — a registry-listed
// write leaf gates with its specific effect wording, while an unrecognized
// proposal subcommand token still gates fail-closed but with the generic wording,
// proving `gated-commands.txt` actually influences the decision (not dead data).
func TestGateHelpPathAndRegistryDriven(t *testing.T) {
	t.Run("help and bare proposal pass ungated", func(t *testing.T) {
		for _, cmd := range []string{
			"glassfrog proposal --help",
			"glassfrog proposal",
			"glassfrog proposal -h",
		} {
			dec, _, err := runGateScript(cmd)
			if err != nil {
				t.Fatalf("gate errored on %q: %v", cmd, err)
			}
			if dec == "ask" {
				t.Errorf("%q was gated (ask) — a help/usage path writes nothing and must pass ungated", cmd)
			}
		}
	})

	t.Run("registry-listed write gates with its specific effect", func(t *testing.T) {
		dec, msg, err := runGateScript("glassfrog proposal propose prp_0123")
		if err != nil {
			t.Fatal(err)
		}
		if dec != "ask" {
			t.Fatalf("a registry-listed write was not gated: decision %q", dec)
		}
		if !strings.Contains(msg, "into circulation") {
			t.Errorf("gated write message lost its specific effect wording: %q", msg)
		}
	})

	t.Run("unrecognized proposal subcommand gates fail-closed with generic wording", func(t *testing.T) {
		// A concrete token that is not in the registry and is not a recognized read.
		leaf := "escalate"
		leaves, err := ReadGatedRegistry()
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range leaves {
			if l == "proposal "+leaf {
				t.Fatalf("%q is in the registry — pick a token that is not, to exercise fail-closed", leaf)
			}
		}
		dec, msg, err := runGateScript("glassfrog proposal " + leaf + " prp_9")
		if err != nil {
			t.Fatal(err)
		}
		if dec != "ask" {
			t.Fatalf("an unrecognized proposal subcommand was not gated fail-closed: decision %q", dec)
		}
		if !strings.Contains(msg, "unrecognized proposal subcommand") {
			t.Errorf("fail-closed message did not use the generic wording (registry not consulted?): %q", msg)
		}
	})
}

// TestGateRegistrationWellFormed pins T001's contract: the PreToolUse hook that
// wires the write-safety gate to the Bash tool is present, deterministic, and
// bounded. A malformed or mis-shaped registration leaves the guardrail
// unenforced, so it is a build failure, not a runtime surprise.
func TestGateRegistrationWellFormed(t *testing.T) {
	cfg, _, err := ReadHooksConfig()
	if err != nil {
		t.Fatalf("could not read/parse the hook registration %s: %v", GateHooksPath, err)
	}
	if problems := ValidateGateRegistration(cfg); len(problems) != 0 {
		t.Fatalf("hook registration violates the interface contract:\n  - %s", joinDrift(problems))
	}
}

// TestGatedRegistryListsExactlyTheProposalWrites pins the single-sourced registry
// (T001): it enumerates EXACTLY the four proposal-write leaves and nothing else —
// no read (`proposal get`/`list`) and no operational `tension` command, which
// pass ungated. The gate script and the drift tripwire both read this file, so a
// drift here would silently mis-scope the gate.
func TestGatedRegistryListsExactlyTheProposalWrites(t *testing.T) {
	leaves, err := ReadGatedRegistry()
	if err != nil {
		t.Fatalf("could not read the gated-command registry %s: %v", GatedRegistryPath, err)
	}
	want := []string{
		"proposal create",
		"proposal propose",
		"proposal respond",
		"proposal withdraw",
	}
	got := append([]string(nil), leaves...)
	sort.Strings(got)
	sortedWant := append([]string(nil), want...)
	sort.Strings(sortedWant)
	if len(got) != len(sortedWant) {
		t.Fatalf("registry lists %d leaves %v, want exactly the %d proposal-write leaves %v", len(got), leaves, len(sortedWant), want)
	}
	for i := range got {
		if got[i] != sortedWant[i] {
			t.Fatalf("registry leaves %v do not match the four proposal-write leaves %v", leaves, want)
		}
	}
	// Belt-and-braces on the exclusions the interface calls out explicitly: no
	// read and no tension command may leak into the gated set.
	for _, leaf := range leaves {
		for _, forbidden := range []string{"tension", "get", "list", "search", "roles", "me"} {
			if leaf == "proposal "+forbidden || leaf == forbidden {
				t.Errorf("registry contains %q — reads and tension edits must stay ungated (out of the registry)", leaf)
			}
		}
	}
}

// TestWriteSafetyRegistryDriftGuard is the best-effort drift tripwire (T003, plan
// ADR-4). It anchors the single-sourced gated-command registry to the CLI's actual
// `proposal` subcommand surface so a newly-added or renamed proposal write command
// cannot silently ship UNGATED.
//
// COVERAGE (explicitly partial, per plan R4 — stated, not silent):
//   - every gated registry leaf (create/propose/respond/withdraw) still exists on
//     the CLI's proposal command;
//   - the CLI's full proposal subcommand surface still matches the checked-in
//     expectation, so an added/renamed leaf (read OR write) breaks the build until
//     it is reclassified and — if a write — added to the registry.
//
// NOT COVERED (no machine source to anchor against; left to review + the BDD
// suite's command-variant unit coverage): the hook's command-string parsing
// robustness (chaining, quoting, aliases — plan R1). The tripwire pins the
// enumerable surface, never the parser; it is not total coverage.
func TestWriteSafetyRegistryDriftGuard(t *testing.T) {
	registry, err := ReadGatedRegistry()
	if err != nil {
		t.Fatalf("could not read the gated-command registry: %v", err)
	}
	live, err := LiveProposalSubcommands()
	if err != nil {
		t.Fatalf("could not extract the CLI's proposal subcommand surface: %v", err)
	}
	// Sanity-check the extraction itself, so a regression in LiveProposalSubcommands
	// fails loudly rather than silently reporting an empty surface as "no drift".
	if len(live) == 0 {
		t.Fatal("extracted no proposal subcommands — the surface anchor could not be read")
	}
	if drift := CheckRegistryDrift(registry, live); len(drift) != 0 {
		t.Fatalf("write-safety registry drifted from the CLI's proposal surface:\n  - %s", joinDrift(drift))
	}
}

// TestGuardrailKeepsManifestSetupFree confirms the design choice that makes T001
// safe for 062: the hook is discovered at the default hooks path, NOT declared as
// a `hooks` key in plugin.json, so the manifest stays free of every setup-forcing
// key (062's ManifestDemandsNoSetup). Adding the key would silently break
// Operator Orientation's "no configuration beyond the CLI's credential setup"
// contract; this test fails fast if a future change reaches for it.
func TestGuardrailKeepsManifestSetupFree(t *testing.T) {
	_, raw, err := ReadOrientationManifest()
	if err != nil {
		t.Fatalf("could not read the plugin manifest: %v", err)
	}
	if !ManifestDemandsNoSetup(raw) {
		t.Fatal("plugin.json now declares a setup-forcing key (mcpServers/hooks/commands/agents/skills) — 063 must wire its hook via the default hooks path, not a manifest key, to keep 062's no-setup contract")
	}
	// The plugin tree stays pure data: the gate is bash + a data registry, no Go
	// code compiled into the CLI, so a hook-load failure can never break a command.
	clean, err := OrientationPluginHasNoGoCode()
	if err != nil {
		t.Fatalf("could not inspect the plugin tree: %v", err)
	}
	if !clean {
		t.Fatal("plugin tree now carries Go code — a hook/manifest failure could no longer be isolated from the CLI")
	}
}
