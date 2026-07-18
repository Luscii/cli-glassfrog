package build

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Tension Processing Path (066) is the second operator *path* on the Claude
// plugin 062 established, and the write-side counterpart to the read-only
// Governance Navigation Path (064): a thin `tension-processing` skill that
// delegates to a write-capable-but-fenced `tension-processor` agent under
// plugin/agents/. Like 062/063/064 it adds NO code to the Go CLI — the artifacts
// are declarative (a skill, an agent, and a single-sourced composed-leaf
// registry). This file gives internal/build read + validation access so the BDD
// suite can assert the artifacts are well-formed and the drift guard can keep the
// composed tension leaves truthful to the shipped CLI's `tension` subcommand
// surface AND disjoint from 063's gated proposal-write set (the ungated-writes
// invariant plan ADR-3 depends on).
//
// internal/build stays cli-free by deliberate convention (see VersionInjectionTarget
// / operatororientation.go): the CLI's command surface is matched as strings against
// the CLI sources rather than importing internal/cli and inverting the dependency.

// Repo-relative locations of the tension-processing-path artifacts (forward-slash;
// joined through filepath so the reads are OS-agnostic).
const (
	// TensionSkillPath is the thin, discoverable entry point the host loads on
	// demand. Its frontmatter description is the trigger surface.
	TensionSkillPath = "plugin/skills/tension-processing/SKILL.md"

	// TensionProcessorAgentPath is the write-capable-but-fenced subagent the skill
	// delegates processing to. It is auto-discovered from plugin/agents/ by
	// directory convention — no `agents` key is added to plugin.json (063's
	// hooks.json and 064's navigator confirmed directory auto-discovery;
	// ManifestDemandsNoSetup still forbids the key).
	TensionProcessorAgentPath = "plugin/agents/tension-processor.md"

	// TensionCommandsPath is the single source of the `tension <sub>` leaves the
	// processor composes, read by BOTH the agent artifact (which names exactly
	// these leaves) and the drift guard (which checks each still resolves in the
	// CLI and is absent from 063's gated set). Mirrors 063's gated-commands.txt
	// and 064's composed-reads.txt single-sourcing (plan ADR-5).
	TensionCommandsPath = "plugin/agents/tension-processing-commands.txt"

	// tensionSurfaceSource is the CLI source the tension subcommand surface is
	// extracted from — the group constructor that registers every leaf.
	tensionSurfaceSource = "internal/cli/tension.go (newTensionCommand)"
)

// ReadTensionSkill reads the committed tension-processing SKILL.md (frontmatter
// + body).
func ReadTensionSkill() (string, error) {
	raw, err := readRepoFile(TensionSkillPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ReadTensionProcessorAgent reads the committed tension-processor agent
// (frontmatter + body).
func ReadTensionProcessorAgent() (string, error) {
	raw, err := readRepoFile(TensionProcessorAgentPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ReadTensionCommands reads the single-sourced composed-leaf registry, returning
// the `tension <sub>` leaves it lists (tension list, tension create, …). Comment
// (#) and blank lines are ignored; each remaining line is one leaf, whitespace
// trimmed. This is the same source the agent artifact enumerates, so the drift
// guard anchors against exactly what the processor is told it may compose.
func ReadTensionCommands() ([]string, error) {
	raw, err := readRepoFile(TensionCommandsPath)
	if err != nil {
		return nil, err
	}
	return parseTensionCommands(string(raw)), nil
}

// parseTensionCommands extracts the composed leaves from registry content. Split
// out so the comment/blank-line skipping is unit-testable without a filesystem
// read. Interior whitespace collapses, so "tension   list" reads the same as
// "tension list" — the leaf identity is the token sequence (the same line format
// as 063's gated-commands.txt, so the disjointness check compares like with like).
func parseTensionCommands(content string) []string {
	var leaves []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		leaves = append(leaves, strings.Join(strings.Fields(trimmed), " "))
	}
	return leaves
}

// --- Drift guard (T002) ----------------------------------------------------
//
// The drift guard anchors the single-sourced composed-leaf registry to the CLI's
// actual `tension` subcommand surface, so a renamed or dropped tension command
// cannot leave the processing artifacts naming a command the CLI no longer
// exposes — and to 063's gated proposal-write registry, so the ungated-writes
// invariant (plan ADR-3: the processor's writes run without a confirmation gate
// BECAUSE they are not governance writes) cannot silently break in either
// direction (a tension leaf pulled into the gated set, or a proposal leaf leaking
// into the composed set).
//
// It is best-effort and explicitly PARTIAL (plan ADR-5, stated not silent): it
// pins the EXISTENCE of the composed tension leaves and their GATE-MEMBERSHIP
// only. It deliberately does NOT verify their flags (deferred to
// `glassfrog tension <sub> --help`), the tension-record prose, or the gate
// script's command-string parsing robustness — those have no machine source to
// anchor against here and stay review + sibling-suite concerns. That gap is
// stated here rather than left silent (no silent caps).
//
// build stays cli-free (see VersionInjectionTarget / operatororientation.go), so
// the command surface is read as strings from the CLI sources rather than by
// importing internal/cli and inverting the dependency.

// LiveTensionSubcommands extracts the current `tension` subcommand leaves from
// the CLI source: it reads the leaves newTensionCommand registers, then resolves
// each constructor's cobra `Use:` token. Returned sorted. Extracting the real
// `Use:` token (not the constructor name) means a rename of the command word is
// caught even if the constructor keeps its name. Mirrors LiveProposalSubcommands
// (063), which anchors the sibling `proposal` surface the same way.
func LiveTensionSubcommands() ([]string, error) {
	tensionGo, err := readRepoFile("internal/cli/tension.go")
	if err != nil {
		return nil, err
	}
	body, ok := extractFuncBody(string(tensionGo), "newTensionCommand")
	if !ok {
		return nil, fmt.Errorf("could not locate newTensionCommand in %s", tensionSurfaceSource)
	}
	suffixes := regexp.MustCompile(`MustRegister\(group,\s*newTension(\w+)Command\(`).FindAllStringSubmatch(body, -1)
	if len(suffixes) == 0 {
		return nil, fmt.Errorf("found no tension subcommands registered in newTensionCommand (%s)", tensionSurfaceSource)
	}
	sources, err := readTensionSources()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var leaves []string
	for _, m := range suffixes {
		suffix := m[1]
		ctor := regexp.MustCompile(`func newTension` + regexp.QuoteMeta(suffix) + `Command\([^)]*\)[^{]*\{`)
		loc := ctor.FindStringIndex(sources)
		if loc == nil {
			return nil, fmt.Errorf("could not find constructor newTension%sCommand for a registered tension leaf", suffix)
		}
		use := regexp.MustCompile(`Use:\s*"([a-zA-Z][\w-]*)`).FindStringSubmatch(sources[loc[1]:])
		if use == nil {
			return nil, fmt.Errorf("could not read the Use token for newTension%sCommand", suffix)
		}
		if !seen[use[1]] {
			seen[use[1]] = true
			leaves = append(leaves, use[1])
		}
	}
	sort.Strings(leaves)
	return leaves, nil
}

// CheckTensionProcessingDrift returns one finding per way the composed-leaf
// registry, the CLI's `tension` subcommand surface, and 063's gated
// proposal-write registry have diverged. Empty means truthful. Each finding names
// the offending leaf so a CI failure points straight at it.
//
// Every side is derived from source — the composed leaves from
// tension-processing-commands.txt, the live surface from newTensionCommand, the
// gated set from 063's gated-commands.txt — so the guard hard-codes none of them
// (LEARNINGS: a drift guard must not hard-code the value it guards).
//
//	(a) every composed leaf must be a `tension <sub>` pair — a leaf under any
//	    other group (e.g. a proposal leaf leaking into the composed set) is
//	    reported, not silently accepted;
//	(b) every composed leaf's subcommand must still exist on the CLI's `tension`
//	    command — else the artifacts name a command the CLI dropped or renamed;
//	(c) the composed set must be disjoint from 063's gated proposal-write set —
//	    in BOTH directions: no composed leaf may be gated, and no `tension` leaf
//	    may appear in the gated registry (the ungated-writes invariant);
//	(d) the tension-processor agent must name every composed leaf — so the
//	    artifact prose stays a genuine consumer of the single source and cannot
//	    silently drop one.
func CheckTensionProcessingDrift(composedLeaves, liveTensionSubcommands, gatedLeaves []string, agent string) []string {
	var findings []string

	liveSet := map[string]bool{}
	for _, s := range liveTensionSubcommands {
		liveSet[s] = true
	}
	gatedSet := map[string]bool{}
	for _, g := range gatedLeaves {
		gatedSet[g] = true
	}

	for _, leaf := range composedLeaves {
		fields := strings.Fields(leaf)

		// (a) the composed set holds `tension <sub>` pairs and nothing else.
		if len(fields) != 2 || fields[0] != "tension" {
			findings = append(findings, fmt.Sprintf("composed leaf %q is not a `tension <sub>` pair — a non-tension command (e.g. a proposal leaf) must not enter the processor's composed set (registry: %s)", leaf, TensionCommandsPath))
			continue
		}

		// (b) the subcommand must still exist on the CLI's tension surface.
		if !liveSet[fields[1]] {
			findings = append(findings, fmt.Sprintf("composed leaf %q no longer exists as a subcommand of the CLI's tension command — the processing artifacts name a command the CLI dropped or renamed; fix the artifact or restore the command (anchor: %s)", leaf, tensionSurfaceSource))
		}

		// (c) composed leaves must stay out of 063's gated set: the processor's
		// writes run ungated BECAUSE they are operational, not governance, writes.
		if gatedSet[leaf] {
			findings = append(findings, fmt.Sprintf("composed leaf %q appears in 063's gated registry — the processor's writes would start prompting; the composed set must stay disjoint from the gated proposal-write set (registry: %s)", leaf, GatedRegistryPath))
		}

		// (d) the agent artifact must name every composed leaf — the prose is a
		// genuine consumer of the single source, not a divergent copy.
		if !mentionsToken(agent, leaf) {
			findings = append(findings, fmt.Sprintf("the tension-processor agent no longer names the composed leaf %q that %s lists — the artifact prose drifted from the single source", leaf, TensionCommandsPath))
		}
	}

	// (c, reverse direction) no `tension` leaf may enter the gated registry — a
	// gated tension write would contradict 063's Behavioral Accord (operational
	// tension edits pass ungated) and this path's ADR-3.
	for _, g := range gatedLeaves {
		if fields := strings.Fields(g); len(fields) > 0 && fields[0] == "tension" {
			findings = append(findings, fmt.Sprintf("gated registry leaf %q gates a tension command — operational tension writes must stay ungated (063 Behavioral Accord; 066 ADR-3); remove it from %s or reclassify the command", g, GatedRegistryPath))
		}
	}

	return findings
}

// readTensionSources concatenates the non-test `tension*.go` sources in
// internal/cli so a constructor defined in any of them (tension.go holds the
// writes, tension_reads.go the situating reads) can be found without importing
// the package (build stays cli-free).
func readTensionSources() (string, error) {
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
		if e.IsDir() || !strings.HasPrefix(name, "tension") || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
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
