package build

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Write-Safety Guardrail (063) extends the Claude plugin 062 established with an
// enforcing layer: a PreToolUse hook over the `Bash` tool that recognizes a
// governance write on the proposal write path and routes it to the host's
// human-confirmation prompt before it runs. Like Operator Orientation it adds NO
// code to the Go CLI — the artifacts are declarative (a hook registration, a gate
// script, and a single-sourced gated-command registry) plus this build-side
// read/validation access so the BDD suite can drive the real gate and the drift
// tripwire can keep the registry truthful to the shipped CLI's proposal surface.
//
// internal/build stays cli-free by deliberate convention (see VersionInjectionTarget
// / operatororientation.go): the proposal-surface anchors are matched as strings
// against the CLI sources rather than importing internal/cli and inverting the
// dependency.

// Repo-relative locations of the guardrail artifact (forward-slash; joined through
// filepath so the reads are OS-agnostic).
const (
	// GateHooksPath is the hook registration the host auto-discovers at the
	// plugin's default hooks path. It wires the PreToolUse gate to the Bash tool.
	// The registration lives at the convention path rather than as a `hooks` key
	// in plugin.json so 062's manifest stays free of setup-forcing keys
	// (ManifestDemandsNoSetup) — the hook rides on directory convention, exactly
	// as the orientation skill rides on skills/.
	GateHooksPath = "plugin/hooks/hooks.json"

	// GateScriptPath is the recognizer + permission-decision emitter the host runs
	// on each matched Bash call. Working name (interface [ASSUMED]); nothing keys
	// on it but hooks.json (which invokes it) and the in-repo tests.
	GateScriptPath = "plugin/hooks/glassfrog-write-gate.sh"

	// GatedRegistryPath is the single source of the gated proposal-write leaves,
	// read by BOTH the gate script (to classify) and the drift tripwire (to anchor).
	GatedRegistryPath = "plugin/hooks/gated-commands.txt"

	// GatePluginRootVar is the token the host expands to the installed plugin
	// directory; the hook command must be rooted at it so it resolves wherever the
	// plugin is installed.
	GatePluginRootVar = "${CLAUDE_PLUGIN_ROOT}"
)

// --- Hook registration (hooks.json) ----------------------------------------

// HooksConfig is the subset of the Claude plugin hooks file this guard validates:
// a map of event name -> matcher groups, each carrying command hooks.
type HooksConfig struct {
	Description string                   `json:"description"`
	Hooks       map[string][]HookMatcher `json:"hooks"`
}

// HookMatcher scopes a group of hooks to the tool calls whose name matches
// Matcher (e.g. "Bash").
type HookMatcher struct {
	Matcher string      `json:"matcher"`
	Hooks   []HookEntry `json:"hooks"`
}

// HookEntry is one hook the host runs. Type is "command" (a deterministic script)
// — never "prompt" (an LLM-judged decision), which would make enforcement depend
// on the very agent judgment the guardrail backstops.
type HookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

// ParseHooksConfig decodes raw hooks.json bytes. A decode error is the
// "malformed registration → host cannot load the hook" condition; because the
// plugin tree carries no Go code, nothing in the CLI is affected — the write
// simply falls back to 062's guidance-only behavior.
func ParseHooksConfig(raw []byte) (HooksConfig, error) {
	var c HooksConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return HooksConfig{}, err
	}
	return c, nil
}

// ReadHooksConfig reads and decodes the committed hook registration.
func ReadHooksConfig() (HooksConfig, []byte, error) {
	raw, err := readRepoFile(GateHooksPath)
	if err != nil {
		return HooksConfig{}, nil, err
	}
	c, err := ParseHooksConfig(raw)
	return c, raw, err
}

// PreToolUseBashGate returns the single command hook the config registers for the
// PreToolUse/Bash path. The second return is false when no such matcher exists.
// It requires exactly one command hook under the Bash matcher — the gate is a
// single deterministic script, not a chain — so an accidental second hook (or a
// missing one) is caught rather than silently accepted.
func PreToolUseBashGate(c HooksConfig) (HookEntry, bool) {
	for _, m := range c.Hooks["PreToolUse"] {
		if m.Matcher != "Bash" {
			continue
		}
		if len(m.Hooks) != 1 {
			return HookEntry{}, false
		}
		return m.Hooks[0], true
	}
	return HookEntry{}, false
}

// ValidateGateRegistration checks the registration meets the interface contract:
// a PreToolUse hook, scoped to Bash, that is a deterministic `command` (never a
// `prompt`), rooted at ${CLAUDE_PLUGIN_ROOT}, invoked through `bash`, and bounded
// by a positive timeout so a hung gate cannot stall the agent. It returns one
// finding per violated clause (empty means well-formed) so a failure points
// straight at the offending field.
func ValidateGateRegistration(c HooksConfig) []string {
	var problems []string
	gate, ok := PreToolUseBashGate(c)
	if !ok {
		return []string{fmt.Sprintf("no single PreToolUse hook scoped to matcher %q found in %s", "Bash", GateHooksPath)}
	}
	if gate.Type != "command" {
		problems = append(problems, fmt.Sprintf("PreToolUse/Bash hook type is %q, want %q — enforcement must be a deterministic script, never a %q (LLM-judged) hook", gate.Type, "command", "prompt"))
	}
	if !strings.Contains(gate.Command, GatePluginRootVar) {
		problems = append(problems, fmt.Sprintf("hook command %q is not rooted at %s — it would not resolve wherever the plugin is installed", gate.Command, GatePluginRootVar))
	}
	if !strings.Contains(gate.Command, "bash ") {
		problems = append(problems, fmt.Sprintf("hook command %q does not invoke the gate through `bash` — the runtime is pinned to bash (no other interpreter dependency)", gate.Command))
	}
	if !strings.Contains(gate.Command, "glassfrog-write-gate.sh") {
		problems = append(problems, fmt.Sprintf("hook command %q does not point at the gate script %s", gate.Command, GateScriptPath))
	}
	if gate.Timeout <= 0 {
		problems = append(problems, fmt.Sprintf("hook timeout is %d, want a positive bound so a hung gate cannot stall the agent", gate.Timeout))
	}
	return problems
}

// --- Gated-command registry -------------------------------------------------

// ReadGatedRegistry reads the single-sourced gated-command registry, returning
// the proposal-write leaves it lists (e.g. "proposal create"). Comment lines (#)
// and blank lines are ignored; each remaining line is one gated leaf, whitespace
// trimmed. This is the same source the gate script parses at runtime, so the
// drift tripwire anchors against exactly what the gate enforces.
func ReadGatedRegistry() ([]string, error) {
	raw, err := readRepoFile(GatedRegistryPath)
	if err != nil {
		return nil, err
	}
	return parseGatedRegistry(string(raw)), nil
}

// parseGatedRegistry extracts the gated leaves from registry file content. Split
// out so the exact comment/blank-line skipping the gate script relies on is unit
// testable without a filesystem read.
func parseGatedRegistry(content string) []string {
	var leaves []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Collapse interior whitespace so "proposal   create" reads the same as
		// "proposal create" — the leaf identity is the token sequence.
		leaves = append(leaves, strings.Join(strings.Fields(trimmed), " "))
	}
	return leaves
}

// ReadGateScript reads the committed gate-script content, for the structural
// checks (it is pinned to bash, keys only on the command path, carries no
// stale-write recovery logic) the BDD suite asserts.
func ReadGateScript() (string, error) {
	raw, err := readRepoFile(GateScriptPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
