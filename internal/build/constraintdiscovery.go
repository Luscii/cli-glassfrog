package build

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
