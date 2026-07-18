package build

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Constraint Discovery Path (065) is the second operator *path* on the Claude
// plugin 062 established: a thin `constraint-discovery` skill (which also owns
// the clarify-when-vague exchange, ADR-3) that delegates to a read-only
// `constraint-navigator` agent under plugin/agents/. Like its sibling paths
// (062/063/064) it adds NO code to the Go CLI — the artifacts are declarative
// (a skill, an agent, and a single-sourced composed-read registry). This file
// gives internal/build read + validation access so the BDD suite can assert
// the artifacts are well-formed and the drift guard can keep the composed read
// leaves truthful to the shipped CLI's command surface.
//
// internal/build stays cli-free by deliberate convention (see
// VersionInjectionTarget / operatororientation.go): the CLI's command surface
// is matched as strings against the CLI sources rather than importing
// internal/cli and inverting the dependency.

// Repo-relative locations of the constraint-discovery-path artifacts
// (forward-slash; joined through filepath so the reads are OS-agnostic).
const (
	// ConstraintSkillPath is the thin, discoverable entry point the host loads
	// on demand. Its frontmatter description is the trigger surface; its body
	// carries the single-sourced workflow including the clarify-when-vague
	// branch the skill (not the agent) owns.
	ConstraintSkillPath = "plugin/skills/constraint-discovery/SKILL.md"

	// ConstraintNavigatorAgentPath is the read-only, non-interactive subagent
	// the skill delegates traversal to. It is auto-discovered from
	// plugin/agents/ by directory convention — no `agents` key is added to
	// plugin.json (064's navigator confirmed directory auto-discovery;
	// ManifestDemandsNoSetup still forbids the key).
	ConstraintNavigatorAgentPath = "plugin/agents/constraint-navigator.md"

	// ConstraintComposedReadsPath is the single source of the read leaves the
	// constraint-navigator composes, read by BOTH the agent artifact (which
	// names exactly these leaves) and the drift guard (which checks each still
	// resolves in the CLI). Mirrors 064's composed-reads.txt and 063's
	// gated-commands.txt single-sourcing (plan ADR-2). Unlike 064's registry,
	// a leaf here may be a command path ("me roles" — the `roles` subcommand
	// of `me`), not only a top-level command word.
	ConstraintComposedReadsPath = "plugin/agents/constraint-discovery-composed-reads.txt"
)

// ReadConstraintSkill reads the committed constraint-discovery SKILL.md
// (frontmatter + body).
func ReadConstraintSkill() (string, error) {
	raw, err := readRepoFile(ConstraintSkillPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ReadConstraintNavigatorAgent reads the committed constraint-navigator agent
// (frontmatter + body).
func ReadConstraintNavigatorAgent() (string, error) {
	raw, err := readRepoFile(ConstraintNavigatorAgentPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ReadConstraintComposedReads reads the single-sourced composed-read registry,
// returning the `glassfrog` read leaves it lists (search, roles, …, "me
// roles"). Comment (#) and blank lines are ignored; each remaining line is one
// read leaf, whitespace trimmed and collapsed — so the two-token "me roles"
// leaf keeps its single-space command-path form. This is the same source the
// agent artifact enumerates, so the drift guard anchors against exactly what
// the navigator is told it may compose.
func ReadConstraintComposedReads() ([]string, error) {
	raw, err := readRepoFile(ConstraintComposedReadsPath)
	if err != nil {
		return nil, err
	}
	return parseComposedReads(string(raw)), nil
}

// --- Drift guard (T002) ------------------------------------------------------
//
// The drift guard anchors the single-sourced composed-read registry to the
// CLI's actual command surface, so a renamed or dropped read cannot leave the
// constraint-discovery artifacts naming a command the CLI no longer exposes
// (plan ADR-2). It is best-effort and explicitly PARTIAL (stated, not silent):
// it pins the EXISTENCE of the composed command leaves only. It deliberately
// does NOT verify their flags (deferred to `glassfrog <command> --help`), the
// synthesized-picture prose or the `characterization` wording, the read-vs-
// write classification of a leaf, or parser robustness — those have no machine
// source to anchor against and stay review + BDD concerns. That gap is stated
// here rather than left silent (no silent caps).
//
// Unlike 064's registry, a 065 leaf may be a command PATH: "me roles" is the
// `roles` subcommand of `me` (the `me` command is variable-wired in app.go, so
// LiveTopLevelCommands deliberately does not see it). The guard therefore
// anchors two surfaces: top-level command words for single-token leaves, and
// the `me` subcommand words for "me <sub>" leaves.
//
// build stays cli-free (see VersionInjectionTarget / operatororientation.go),
// so the command surface is read as strings from the CLI sources rather than
// by importing internal/cli and inverting the dependency.

// LiveMeSubcommands extracts the subcommand surface of the `me` command from
// the wiring site: it confirms `me` itself is registered on the root
// (`MustRegister(root, meCmd)`), reads the constructors registered onto it via
// the `MustRegister(meCmd, new<X>Command(…))` pattern, then resolves each
// constructor's cobra `Use:` token to its command word. Returned sorted.
// Extracting the real `Use:` token (not the constructor name) means a rename
// of the subcommand word is caught even if the constructor keeps its name.
func LiveMeSubcommands() ([]string, error) {
	appSrc, err := readRepoFile("internal/cli/app.go")
	if err != nil {
		return nil, err
	}
	if !regexp.MustCompile(`MustRegister\(root,\s*meCmd\)`).MatchString(string(appSrc)) {
		return nil, fmt.Errorf("the `me` command is no longer registered on root in %s — its subcommands cannot be anchored", cliWiringSource)
	}
	registered := regexp.MustCompile(`MustRegister\(meCmd,\s*new(\w+)Command\(`).FindAllStringSubmatch(string(appSrc), -1)
	if len(registered) == 0 {
		return nil, fmt.Errorf("found no subcommands registered on meCmd in %s", cliWiringSource)
	}
	sources, err := readCLISources()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var words []string
	for _, m := range registered {
		use, ok := resolveUseWord(sources, m[1])
		if !ok {
			// A matched constructor whose definition or Use: token is not found in
			// the cli sources is skipped rather than erroring — the same tolerance
			// LiveTopLevelCommands applies; the positive leaf checks catch a leaf
			// this loses.
			continue
		}
		if !seen[use] {
			seen[use] = true
			words = append(words, use)
		}
	}
	if len(words) == 0 {
		return nil, fmt.Errorf("resolved no subcommand words from the constructors registered on meCmd in %s", cliWiringSource)
	}
	sort.Strings(words)
	return words, nil
}

// resolveUseWord finds `func new<Suffix>Command(` in the concatenated CLI
// sources and returns the first word of the cobra `Use:` string that follows
// it. The second return is false when the constructor or its Use: token cannot
// be located.
func resolveUseWord(sources, suffix string) (string, bool) {
	ctor := regexp.MustCompile(`func new` + regexp.QuoteMeta(suffix) + `Command\(`)
	loc := ctor.FindStringIndex(sources)
	if loc == nil {
		return "", false
	}
	use := regexp.MustCompile(`Use:\s*"([a-zA-Z][\w-]*)`).FindStringSubmatch(sources[loc[1]:])
	if use == nil {
		return "", false
	}
	return use[1], true
}

// CheckConstraintDrift returns one finding per way the composed-read registry
// and the CLI's command surface have diverged. Empty means truthful. Each
// finding names the offending leaf so a CI failure points straight at it.
//
// Both sides are derived from source — the leaves from
// constraint-discovery-composed-reads.txt, the live surfaces from app.go — so
// the guard hard-codes neither (a hard-coded expectation would become a second
// source of truth edited on every legitimate change; LEARNINGS).
//
//	(a) every single-token composed leaf must still exist as a top-level command
//	    on the CLI — else the artifacts name a read the CLI dropped or renamed;
//	(b) every "me <sub>" composed leaf must still exist as a subcommand of the
//	    root-registered `me` command (the only command-path form the registry
//	    carries; any other parent is reported as unanchorable rather than
//	    silently skipped);
//	(c) the constraint-navigator agent must name every composed leaf — so the
//	    artifact prose stays a genuine consumer of the single source, not a
//	    divergent copy.
func CheckConstraintDrift(composedLeaves, liveTop, liveMe []string, agent string) []string {
	var findings []string

	topSet := map[string]bool{}
	for _, r := range liveTop {
		topSet[r] = true
	}
	meSet := map[string]bool{}
	for _, r := range liveMe {
		meSet[r] = true
	}

	for _, leaf := range composedLeaves {
		switch parts := strings.Fields(leaf); {
		case len(parts) == 1:
			// (a) top-level leaves must still exist on the CLI's top-level surface.
			if !topSet[leaf] {
				findings = append(findings, fmt.Sprintf("composed read leaf %q no longer exists as a top-level command in the CLI — the constraint-discovery artifacts name a read the CLI dropped or renamed; fix the artifact or restore the command (anchor: %s)", leaf, cliWiringSource))
			}
		case len(parts) == 2 && parts[0] == "me":
			// (b) `me` subcommand leaves must still exist on the `me` command.
			if !meSet[parts[1]] {
				findings = append(findings, fmt.Sprintf("composed read leaf %q no longer exists as a subcommand of `me` in the CLI — the constraint-discovery artifacts name a read the CLI dropped or renamed; fix the artifact or restore the command (anchor: %s)", leaf, cliWiringSource))
			}
		default:
			// The registry carries a command path the guard has no anchor for —
			// report it rather than silently skipping (no silent caps).
			findings = append(findings, fmt.Sprintf("composed read leaf %q is a command path the drift guard cannot anchor (only top-level commands and `me <sub>` are supported) — extend the guard or fix the registry", leaf))
		}

		// (c) the agent artifact must name every composed leaf — the prose is a
		// genuine consumer of the single source, not a divergent copy.
		if !mentionsToken(agent, leaf) {
			findings = append(findings, fmt.Sprintf("the constraint-navigator agent no longer names the composed read leaf %q that %s lists — the artifact prose drifted from the single source", leaf, ConstraintComposedReadsPath))
		}
	}

	return findings
}
