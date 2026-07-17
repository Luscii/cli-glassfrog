package build

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Governance Navigation Path (064) is the first operator *path* on the Claude
// plugin 062 established: a thin `governance-navigation` skill that delegates to a
// read-only `governance-navigator` agent under plugin/agents/. Like Operator
// Orientation (062) and the Write-Safety Guardrail (063) it adds NO code to the Go
// CLI — the artifacts are declarative (a skill, an agent, and a single-sourced
// composed-read registry). This file gives internal/build read + validation access
// so the BDD suite can assert the artifacts are well-formed and the drift guard can
// keep the composed read leaves truthful to the shipped CLI's command surface.
//
// internal/build stays cli-free by deliberate convention (see VersionInjectionTarget
// / operatororientation.go): the CLI's command surface is matched as strings against
// the CLI sources rather than importing internal/cli and inverting the dependency.

// Repo-relative locations of the navigation-path artifacts (forward-slash; joined
// through filepath so the reads are OS-agnostic).
const (
	// NavigationSkillPath is the thin, discoverable entry point the host loads on
	// demand. Its frontmatter description is the trigger surface.
	NavigationSkillPath = "plugin/skills/governance-navigation/SKILL.md"

	// NavigatorAgentPath is the read-only subagent the skill delegates traversal
	// to. It is auto-discovered from plugin/agents/ by directory convention — no
	// `agents` key is added to plugin.json (063's hooks.json confirmed the plugin
	// uses directory auto-discovery; ManifestDemandsNoSetup still forbids the key).
	NavigatorAgentPath = "plugin/agents/governance-navigator.md"

	// ComposedReadsPath is the single source of the read leaves the navigator
	// composes, read by BOTH the agent artifact (which names exactly these leaves)
	// and the drift guard (which checks each still resolves in the CLI). Mirrors
	// 063's gated-commands.txt single-sourcing (plan ADR-4).
	ComposedReadsPath = "plugin/agents/composed-reads.txt"

	// cliWiringSource is the single, explicit command-wiring site (app.go's
	// Assemble) the live top-level command surface is extracted from.
	cliWiringSource = "internal/cli/app.go (Assemble)"
)

// ReadNavigationSkill reads the committed governance-navigation SKILL.md
// (frontmatter + body).
func ReadNavigationSkill() (string, error) {
	raw, err := readRepoFile(NavigationSkillPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ReadNavigatorAgent reads the committed governance-navigator agent (frontmatter
// + body).
func ReadNavigatorAgent() (string, error) {
	raw, err := readRepoFile(NavigatorAgentPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ReadComposedReads reads the single-sourced composed-read registry, returning the
// top-level `glassfrog` read leaves it lists (search, roles, tree, …). Comment (#)
// and blank lines are ignored; each remaining line is one read leaf, whitespace
// trimmed. This is the same source the agent artifact enumerates, so the drift
// guard anchors against exactly what the navigator is told it may compose.
func ReadComposedReads() ([]string, error) {
	raw, err := readRepoFile(ComposedReadsPath)
	if err != nil {
		return nil, err
	}
	return parseComposedReads(string(raw)), nil
}

// parseComposedReads extracts the read leaves from registry content. Split out so
// the comment/blank-line skipping is unit-testable without a filesystem read.
func parseComposedReads(content string) []string {
	var leaves []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Each leaf is a single top-level command token; collapse any stray
		// interior whitespace so the leaf identity is the token itself.
		leaves = append(leaves, strings.Join(strings.Fields(trimmed), " "))
	}
	return leaves
}

// AgentTools returns the tools the navigator agent's frontmatter grants, and false
// when no `tools:` field is present. The inline comma form (`tools: Bash, Read`)
// is the official installed-plugin convention (feature-dev agents); a YAML block
// list is also accepted so the grant can be read either way.
func AgentTools(agent string) ([]string, bool) {
	front, ok := frontmatterBlock(agent)
	if !ok {
		return nil, false
	}
	lines := strings.Split(front, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		rest, isTools := strings.CutPrefix(trimmed, "tools:")
		if !isTools {
			continue
		}
		rest = strings.TrimSpace(rest)
		if rest != "" {
			// Inline form: `tools: Bash, Read, Grep, Glob` (also tolerates a
			// bracketed `[Bash, Read]`).
			rest = strings.Trim(rest, "[]")
			return splitToolTokens(rest), true
		}
		// Block form: subsequent `- Tool` lines.
		var tools []string
		for _, bl := range lines[i+1:] {
			bt := strings.TrimSpace(bl)
			item, isItem := strings.CutPrefix(bt, "-")
			if !isItem {
				break
			}
			if t := strings.TrimSpace(strings.Trim(item, `"'`)); t != "" {
				tools = append(tools, t)
			}
		}
		return tools, true
	}
	return nil, false
}

// splitToolTokens splits a comma-separated tool list, trimming quotes/whitespace.
func splitToolTokens(s string) []string {
	var tools []string
	for _, tok := range strings.Split(s, ",") {
		if t := strings.TrimSpace(strings.Trim(tok, `"'`)); t != "" {
			tools = append(tools, t)
		}
	}
	return tools
}

// frontmatterBlock returns the leading YAML frontmatter (between the first two
// `---` delimiters) of a plugin markdown file. The second return is false when no
// frontmatter block is present.
func frontmatterBlock(content string) (string, bool) {
	trimmed := strings.TrimLeft(content, " \t\r\n")
	if !strings.HasPrefix(trimmed, "---") {
		return "", false
	}
	rest := trimmed[len("---"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", false
	}
	return rest[:end], true
}

// --- Drift guard (T002) ----------------------------------------------------
//
// The drift guard anchors the single-sourced composed-read registry to the CLI's
// actual top-level command surface, so a renamed or dropped read cannot leave the
// navigation artifacts naming a command the CLI no longer exposes (plan ADR-4). It
// is best-effort and explicitly PARTIAL (plan ADR-4, stated not silent): it pins
// the EXISTENCE of the composed command leaves only. It deliberately does NOT
// verify their flags (deferred to `glassfrog <command> --help`), the
// synthesized-picture prose, or the traversal's relevance judgment — those have no
// machine source to anchor against and stay review concerns. That gap is stated
// here rather than left silent (no silent caps).
//
// build stays cli-free (see VersionInjectionTarget / operatororientation.go), so
// the command surface is read as strings from the CLI sources rather than by
// importing internal/cli and inverting the dependency.

// LiveTopLevelCommands extracts the CLI's top-level command surface from the
// wiring site: it reads the constructors Assemble registers directly on the root
// via the `MustRegister(root, new<X>Command(…))` pattern, then resolves each
// constructor's cobra `Use:` token to its command word. Returned sorted.
// Extracting the real `Use:` token (not the constructor name) means a rename of
// the command word is caught even if the constructor keeps its name.
//
// It is NOT exhaustive: a command registered via a variable rather than a direct
// `new<X>Command(` call — e.g. `me`, wired as `MustRegister(root, meCmd)` — is not
// matched. That is fine for the drift guard, whose purpose is only to confirm the
// composed READ leaves still exist, and every one of those is wired with the
// `new<X>Command(` pattern. The matched surface still spans reads AND writes
// (`proposal`, `tension`, …); it is deliberately not read-filtered, so callers
// must assume neither "only reads" nor "every top-level command".
func LiveTopLevelCommands() ([]string, error) {
	appSrc, err := readRepoFile("internal/cli/app.go")
	if err != nil {
		return nil, err
	}
	// The composed read leaves — and most top-level commands — are wired as
	// `MustRegister(root, new<Suffix>Command(…)` at the single explicit wiring site
	// (app.go Assemble). A command registered via a variable (e.g. `me`, wired as
	// `MustRegister(root, meCmd)`) does not match this pattern and is intentionally
	// out of scope: every read leaf the navigation path composes is wired directly.
	registered := regexp.MustCompile(`MustRegister\(root,\s*new(\w+)Command\(`).FindAllStringSubmatch(string(appSrc), -1)
	if len(registered) == 0 {
		return nil, fmt.Errorf("found no top-level commands registered on root in %s", cliWiringSource)
	}
	sources, err := readCLISources()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var words []string
	for _, m := range registered {
		suffix := m[1]
		ctor := regexp.MustCompile(`func new` + regexp.QuoteMeta(suffix) + `Command\(`)
		loc := ctor.FindStringIndex(sources)
		if loc == nil {
			// A matched `new<X>Command(` whose constructor definition is not found
			// in the cli sources is skipped rather than erroring. (Variable-wired
			// commands like `me` never reach this branch — they don't match the
			// `new<X>Command(` pattern above, so they're excluded before the lookup.)
			continue
		}
		use := regexp.MustCompile(`Use:\s*"([a-zA-Z][\w-]*)`).FindStringSubmatch(sources[loc[1]:])
		if use == nil {
			continue
		}
		if !seen[use[1]] {
			seen[use[1]] = true
			words = append(words, use[1])
		}
	}
	if len(words) == 0 {
		return nil, fmt.Errorf("resolved no top-level command words from the constructors registered in %s", cliWiringSource)
	}
	sort.Strings(words)
	return words, nil
}

// CheckNavigationDrift returns one finding per way the composed-read registry and
// the CLI's top-level command surface have diverged. Empty means truthful. Each
// finding names the offending leaf so a CI failure points straight at it.
//
// Both sides are derived from source — the leaves from composed-reads.txt, the
// live surface from app.go — so the guard hard-codes neither (a hard-coded
// expectation would become a second source edited on every legitimate change).
//
//	(a) every composed read leaf must still exist as a top-level command on the CLI
//	    — else the artifacts name a read the CLI dropped or renamed;
//	(b) the navigator agent must name every composed leaf — so the artifact prose
//	    stays a genuine consumer of the single source and cannot silently drop one.
func CheckNavigationDrift(composedLeaves, liveCommands []string, agent string) []string {
	var findings []string

	liveSet := map[string]bool{}
	for _, r := range liveCommands {
		liveSet[r] = true
	}

	// (a) composed leaves must still exist on the CLI's top-level surface.
	for _, leaf := range composedLeaves {
		if !liveSet[leaf] {
			findings = append(findings, fmt.Sprintf("composed read leaf %q no longer exists as a top-level command in the CLI — the navigation artifacts name a read the CLI dropped or renamed; fix the artifact or restore the command (anchor: %s)", leaf, cliWiringSource))
		}
	}

	// (b) the agent artifact must name every composed leaf — the prose is a genuine
	// consumer of composed-reads.txt, not a divergent copy.
	for _, leaf := range composedLeaves {
		if !mentionsToken(agent, leaf) {
			findings = append(findings, fmt.Sprintf("the governance-navigator agent no longer names the composed read leaf %q that %s lists — the artifact prose drifted from the single source", leaf, ComposedReadsPath))
		}
	}

	return findings
}

// readCLISources concatenates the non-test .go sources in internal/cli so a
// command constructor defined in any of them can be found without importing the
// package (build stays cli-free).
func readCLISources() (string, error) {
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
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
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
