package build

import (
	"sort"
	"testing"
)

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
