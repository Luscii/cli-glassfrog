package build

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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

// --- Drift tripwire (T003) --------------------------------------------------
//
// The drift tripwire anchors the single-sourced gated-command registry to the
// CLI's actual `proposal` subcommand surface, so a newly-added or renamed proposal
// write command cannot silently ship UNGATED (plan ADR-4). It is best-effort and
// explicitly PARTIAL (plan R4, stated not silent): it pins the enumerable command
// surface only. It deliberately does NOT verify the hook's command-string parsing
// robustness (chaining, quoting, aliases — plan R1); that has no machine source to
// anchor against and stays a review + unit-test concern (the BDD suite's
// command-variant coverage). This gap is stated here rather than left silent.
//
// build stays cli-free (see VersionInjectionTarget / operatororientation.go), so
// the proposal surface is read as strings from the CLI sources rather than by
// importing internal/cli and inverting the dependency.

// proposalSurfaceSource is the CLI source the proposal subcommand surface is
// extracted from — the group constructor that registers every leaf.
const proposalSurfaceSource = "internal/cli/proposal.go (newProposalCommand)"

// expectedProposalSurface is the checked-in expectation of the CLI's full
// `proposal` subcommand surface (sorted). It is the second half of the drift
// guard: the gated registry (create/propose/respond/withdraw) covers the WRITES;
// this pins the whole surface so an added/renamed leaf — read OR write — breaks
// the build until a human reclassifies it and, if it is a write, adds it to the
// registry. Reads (get/list) live here but not in the registry.
var expectedProposalSurface = []string{"create", "get", "list", "propose", "respond", "withdraw"}

// LiveProposalSubcommands extracts the current `proposal` subcommand leaves from
// the CLI source: it reads the leaves newProposalCommand registers, then resolves
// each constructor's cobra `Use:` token. Returned sorted. Extracting the real
// `Use:` token (not the constructor name) means a rename of the command word is
// caught even if the constructor keeps its name.
func LiveProposalSubcommands() ([]string, error) {
	proposalGo, err := readRepoFile("internal/cli/proposal.go")
	if err != nil {
		return nil, err
	}
	body, ok := extractFuncBody(string(proposalGo), "newProposalCommand")
	if !ok {
		return nil, fmt.Errorf("could not locate newProposalCommand in %s", proposalSurfaceSource)
	}
	suffixes := regexp.MustCompile(`MustRegister\(group,\s*newProposal(\w+)Command\(`).FindAllStringSubmatch(body, -1)
	if len(suffixes) == 0 {
		return nil, fmt.Errorf("found no proposal subcommands registered in newProposalCommand (%s)", proposalSurfaceSource)
	}
	sources, err := readProposalSources()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var leaves []string
	for _, m := range suffixes {
		suffix := m[1]
		ctor := regexp.MustCompile(`func newProposal` + regexp.QuoteMeta(suffix) + `Command\([^)]*\)[^{]*\{`)
		loc := ctor.FindStringIndex(sources)
		if loc == nil {
			return nil, fmt.Errorf("could not find constructor newProposal%sCommand for a registered proposal leaf", suffix)
		}
		use := regexp.MustCompile(`Use:\s*"([a-zA-Z][\w-]*)`).FindStringSubmatch(sources[loc[1]:])
		if use == nil {
			return nil, fmt.Errorf("could not read the Use token for newProposal%sCommand", suffix)
		}
		if !seen[use[1]] {
			seen[use[1]] = true
			leaves = append(leaves, use[1])
		}
	}
	sort.Strings(leaves)
	return leaves, nil
}

// CheckRegistryDrift returns one finding per way the registry and the CLI's
// proposal surface have diverged. Empty means truthful. Each finding names the
// offending command so a CI failure points straight at it.
//
//	(a) every gated registry leaf must still exist on the CLI's proposal command —
//	    else the gate references a write the CLI dropped or renamed;
//	(b) the CLI's proposal surface must match the checked-in expectation — a leaf
//	    added or renamed beyond it (read OR write) must be consciously reclassified
//	    and, if a write, added to the registry.
func CheckRegistryDrift(registryLeaves, liveSurface []string) []string {
	var findings []string

	liveSet := map[string]bool{}
	for _, l := range liveSurface {
		liveSet[l] = true
	}

	// (a) gated leaves must still exist on the CLI.
	for _, r := range registryLeaves {
		leaf := r
		if fields := strings.Fields(r); len(fields) == 2 && fields[0] == "proposal" {
			leaf = fields[1]
		}
		if !liveSet[leaf] {
			findings = append(findings, fmt.Sprintf("gated registry leaf %q no longer exists on the CLI's proposal command — the gate references a write the CLI dropped or renamed; fix the registry or restore the command (anchor: %s)", r, proposalSurfaceSource))
		}
	}

	// (b) the proposal surface must match the checked-in expectation.
	missing, extra := diffSets(expectedProposalSurface, liveSurface)
	sort.Strings(missing)
	sort.Strings(extra)
	for _, e := range extra {
		findings = append(findings, fmt.Sprintf("the CLI's proposal command grew or renamed subcommand %q, absent from the checked-in expectation — reclassify it (read or write?) and, if a write, add it to the gated registry before updating expectedProposalSurface (anchor: %s)", e, proposalSurfaceSource))
	}
	for _, m := range missing {
		findings = append(findings, fmt.Sprintf("expected proposal subcommand %q is no longer on the CLI — the surface shrank or was renamed; update expectedProposalSurface and the registry to match (anchor: %s)", m, proposalSurfaceSource))
	}

	return findings
}

// extractFuncBody returns the text of the named top-level func — from its `func
// <name>(` header to the next top-level `func ` (or end of file). Good enough to
// scope a MustRegister sweep to one constructor without a full Go parse (build
// stays cli-free). The second return is false when the func is absent.
func extractFuncBody(src, name string) (string, bool) {
	start := strings.Index(src, "func "+name+"(")
	if start < 0 {
		return "", false
	}
	rest := src[start+1:]
	if next := strings.Index(rest, "\nfunc "); next >= 0 {
		return src[start : start+1+next], true
	}
	return src[start:], true
}

// readProposalSources concatenates the non-test `proposal*.go` sources in
// internal/cli so a constructor defined in any of them can be found.
func readProposalSources() (string, error) {
	root, err := RepoRoot()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "internal", "cli")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "proposal") || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return "", err
		}
		sb.Write(b)
		sb.WriteByte('\n')
	}
	return sb.String(), nil
}
