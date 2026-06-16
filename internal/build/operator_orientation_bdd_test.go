package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestOperatorOrientationFeatures runs the executable acceptance for Operator
// Orientation (062). Like the other build-side suites (021/022/036) its Paths
// name ONLY this spec's feature file and it runs with the ~@wip filter, so only
// the scenarios implemented so far execute.
//
// The deliverable is a committed Claude plugin (a manifest + one orientation
// skill), so the executable scenarios assert against that artifact rather than a
// runtime: the plugin is well-formed and discoverable, its content carries the
// required cross-cutting knowledge, and a best-effort drift guard keeps the
// skill's enumerable facts truthful to the shipped CLI. The four @validation
// scenarios (no invented surface, no Holacracy coaching, no distribution
// machinery, describes-but-never-enforces gating) stay @wip, held for
// /score:validate.
func TestOperatorOrientationFeatures(t *testing.T) {
	w := &orientationWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.register(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unequipped-agent-operators/operator-orientation.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: operator-orientation feature scenarios failed")
	}
}

// orientationWorld is the per-scenario state: the loaded manifest + skill, a
// parse error (for the malformed-manifest path), and the drift findings.
type orientationWorld struct {
	manifestRaw []byte
	manifest    OrientationManifest
	manifestErr error
	skill       string
	facts       OrientationFacts
	drift       []string
}

func (w *orientationWorld) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = orientationWorld{}
		return ctx, nil
	})

	// --- Scenario: Orientation is consultable once the plugin is present ---
	sc.Step(`^the plugin was present in an agent's environment$`, w.givenPluginPresent)
	sc.Step(`^the agent looks for operating knowledge$`, w.whenLooksForKnowledge)
	sc.Step(`^the orientation knowledge will be available to consult$`, w.thenKnowledgeConsultable)
	sc.Step(`^no configuration beyond the CLI's existing credential setup will be required$`, w.thenNoExtraConfig)

	// --- Scenario: Malformed manifest leaves the plugin unloadable ---
	sc.Step(`^the plugin manifest at "([^"]*)" was malformed$`, w.givenManifestMalformed)
	sc.Step(`^the plugin host attempts to load the plugin$`, w.whenHostLoads)
	sc.Step(`^the host will not register the orientation skill$`, w.thenHostDoesNotRegister)
	sc.Step(`^the agent will fall back to rediscovery with no command in the CLI broken$`, w.thenCLIUnbroken)

	// --- Content scenarios (T002): every "consult" Given just makes the skill
	// available; the assertions read its body. ---
	sc.Step(`^the glassfrog-operator skill was available to the agent$`, w.givenSkillAvailable)
	sc.Step(`^a list command returned more results than one response held$`, w.givenSkillAvailable)
	sc.Step(`^a glassfrog command had just exited with a non-zero code$`, w.givenSkillAvailable)
	sc.Step(`^the agent had supplied no credential$`, w.givenSkillAvailable)
	sc.Step(`^the agent needed the exact flags for one specific command$`, w.givenSkillAvailable)
	sc.Step(`^the Write-Safety Guardrail did not yet exist$`, w.givenSkillAvailable)
	// Context "And" steps that add no machine state.
	sc.Step(`^the agent needed to read a practitioner's roles for downstream parsing$`, w.noop)
	sc.Step(`^a command failed for lack of authentication$`, w.noop)
	sc.Step(`^the agent was about to run a command that writes to the governance record$`, w.noop)
	// "When the agent consults the orientation …" — all variants just ensure the
	// skill is loaded.
	sc.Step(`^the agent consults the orientation for machine-parseable output$`, w.whenConsult)
	sc.Step(`^the agent consults the orientation on pagination$`, w.whenConsult)
	sc.Step(`^the agent consults the orientation for that exit code$`, w.whenConsult)
	sc.Step(`^the agent consults the orientation$`, w.whenConsult)

	sc.Step(`^the orientation will name "([^"]*)" and "([^"]*)" as the parseable formats$`, w.thenNamesParseableFormats)
	sc.Step(`^it will instruct the agent to pass "([^"]*)" rather than parse human-rendered text$`, w.thenInstructOutputFlag)
	sc.Step(`^the orientation will explain how to detect that more pages exist$`, w.thenDetectMorePages)
	sc.Step(`^it will explain how to fetch the subsequent pages$`, w.thenFetchSubsequentPages)
	sc.Step(`^the orientation will state the meaning of each code in the 0–7 convention$`, w.thenMeaningEachCode)
	sc.Step(`^it will state the appropriate reaction for the code received$`, w.thenReactionForCode)
	sc.Step(`^the orientation will direct the agent to "([^"]*)" for the X-Auth-Token key$`, w.thenDirectToAuthLogin)
	sc.Step(`^it will introduce no credential mechanism beyond the CLI's own$`, w.thenNoNewCredentialMechanism)
	sc.Step(`^the orientation will direct the agent to "([^"]*)" for that command$`, w.thenDirectToHelp)
	sc.Step(`^it will not itself enumerate the command's flags$`, w.thenNotEnumerateFlags)
	sc.Step(`^the orientation will state the expectation to confirm before writing$`, w.thenConfirmBeforeWriting)
	sc.Step(`^it will state that a 412 stale-write refusal means re-read and re-confirm, not blind retry$`, w.then412ReReadReconfirm)
	sc.Step(`^it will not block or gate the write itself$`, w.thenNotGate)

	// --- Drift-guard scenarios (T003) ---
	sc.Step(`^the CLI's exit-code or output-format behavior had changed$`, w.givenCliBehaviourChanged)
	sc.Step(`^the orientation is checked against the shipped CLI$`, w.whenDriftChecked)
	sc.Step(`^the mismatch will be treated as a defect to fix$`, w.thenDriftIsDefect)
	sc.Step(`^it will not be accepted as a tolerable difference$`, w.thenDriftNotTolerated)
	sc.Step(`^the orientation documented an output-format token that the CLI no longer supported$`, w.givenSkillDocumentsDroppedToken)
	sc.Step(`^the internal/build drift guard runs$`, w.whenDriftChecked)
	sc.Step(`^the guard will fail$`, w.thenDriftIsDefect)
	sc.Step(`^it will report which documented anchor no longer matches the shipped CLI$`, w.thenDriftNamesFormatAnchor)
}

func (w *orientationWorld) noop() error { return nil }

// --- Scenario: Orientation is consultable once the plugin is present -------

func (w *orientationWorld) givenPluginPresent() error {
	m, raw, err := ReadOrientationManifest()
	if err != nil {
		return fmt.Errorf("plugin manifest did not load: %w", err)
	}
	skill, err := ReadOrientationSkill()
	if err != nil {
		return fmt.Errorf("orientation skill did not load: %w", err)
	}
	w.manifest, w.manifestRaw, w.skill = m, raw, skill
	return nil
}

func (w *orientationWorld) whenLooksForKnowledge() error {
	// Looking for operating knowledge is consulting the skill the host discovered;
	// it was loaded in the Given. Nothing to drive.
	return nil
}

func (w *orientationWorld) thenKnowledgeConsultable() error {
	if w.manifest.Name != "glassfrog-operator" {
		return fmt.Errorf("manifest name is %q, want glassfrog-operator — the host could not identify the plugin", w.manifest.Name)
	}
	// The skill is consultable iff it is discoverable: a frontmatter block that
	// carries the name and the description trigger surface.
	if !hasFrontmatterField(w.skill, "name") || !hasFrontmatterField(w.skill, "description") {
		return fmt.Errorf("SKILL.md lacks the frontmatter name/description that makes it discoverable")
	}
	return nil
}

func (w *orientationWorld) thenNoExtraConfig() error {
	if !ManifestDemandsNoSetup(w.manifestRaw) {
		return fmt.Errorf("manifest declares a key (mcpServers/hooks/commands/agents) that would force setup beyond the CLI's credential flow")
	}
	return nil
}

// --- Scenario: Malformed manifest leaves the plugin unloadable -------------

func (w *orientationWorld) givenManifestMalformed(path string) error {
	if path != OrientationManifestPath {
		return fmt.Errorf("scenario names manifest path %q, guard reads %q", path, OrientationManifestPath)
	}
	// A truncated object — valid JSON start, no close — models a manifest the host
	// cannot parse.
	w.manifestRaw = []byte(`{"name": "glassfrog-operator", "version": "0.1.0"`)
	return nil
}

func (w *orientationWorld) whenHostLoads() error {
	w.manifest, w.manifestErr = ParseOrientationManifest(w.manifestRaw)
	return nil
}

func (w *orientationWorld) thenHostDoesNotRegister() error {
	if w.manifestErr == nil {
		return fmt.Errorf("malformed manifest parsed without error — the host would wrongly register the skill")
	}
	return nil
}

func (w *orientationWorld) thenCLIUnbroken() error {
	// The plugin tree is pure data: nothing under plugin/ is compiled into the
	// CLI, so a manifest the host rejects cannot break a glassfrog command. The
	// agent just falls back to rediscovery.
	clean, err := OrientationPluginHasNoGoCode()
	if err != nil {
		return fmt.Errorf("could not inspect the plugin tree: %w", err)
	}
	if !clean {
		return fmt.Errorf("plugin tree carries Go code — a manifest failure could no longer be isolated from the CLI")
	}
	return nil
}

// --- Content scenarios (T002) ----------------------------------------------

func (w *orientationWorld) ensureSkill() error {
	if w.skill != "" {
		return nil
	}
	skill, err := ReadOrientationSkill()
	if err != nil {
		return fmt.Errorf("orientation skill did not load: %w", err)
	}
	w.skill = skill
	return nil
}

func (w *orientationWorld) givenSkillAvailable() error { return w.ensureSkill() }
func (w *orientationWorld) whenConsult() error         { return w.ensureSkill() }

func (w *orientationWorld) thenNamesParseableFormats(a, b string) error {
	if !mentionsToken(w.skill, a) || !mentionsToken(w.skill, b) {
		return fmt.Errorf("skill does not name both %q and %q as formats", a, b)
	}
	if !containsFold(w.skill, "machine-parseable") && !containsFold(w.skill, "parse") {
		return fmt.Errorf("skill names %q/%q but never frames them as the parseable formats", a, b)
	}
	return nil
}

func (w *orientationWorld) thenInstructOutputFlag(flag string) error {
	if !strings.Contains(w.skill, flag) {
		return fmt.Errorf("skill never instructs the agent to pass %q", flag)
	}
	if !containsFold(w.skill, "human") {
		return fmt.Errorf("skill instructs %q but does not contrast it against scraping human-rendered text", flag)
	}
	return nil
}

func (w *orientationWorld) thenDetectMorePages() error {
	if !strings.Contains(w.skill, "--first-page") || !containsFold(w.skill, "signal") {
		return fmt.Errorf("skill does not explain how to detect that more pages exist (expected --first-page and a signal)")
	}
	return nil
}

func (w *orientationWorld) thenFetchSubsequentPages() error {
	if !containsFold(w.skill, "subsequent") && !containsFold(w.skill, "every page") {
		return fmt.Errorf("skill does not explain how to fetch the subsequent pages")
	}
	return nil
}

func (w *orientationWorld) thenMeaningEachCode() error {
	for code := 0; code <= 7; code++ {
		if !mentionsExitCode(w.skill, code) {
			return fmt.Errorf("skill does not document exit code %d in the 0–7 convention", code)
		}
	}
	return nil
}

func (w *orientationWorld) thenReactionForCode() error {
	// The exit-code table pairs each code with a reaction column.
	if !containsFold(w.skill, "react by") {
		return fmt.Errorf("skill states code meanings but no per-code reaction")
	}
	return nil
}

func (w *orientationWorld) thenDirectToAuthLogin(cmd string) error {
	if !strings.Contains(w.skill, cmd) {
		return fmt.Errorf("skill does not direct the agent to %q", cmd)
	}
	if !strings.Contains(w.skill, "X-Auth-Token") {
		return fmt.Errorf("skill directs to %q but never names the X-Auth-Token key", cmd)
	}
	return nil
}

func (w *orientationWorld) thenNoNewCredentialMechanism() error {
	if !containsFold(w.skill, "no separate credential") && !containsFold(w.skill, "no other credential") {
		return fmt.Errorf("skill does not make clear it introduces no credential mechanism beyond the CLI's own")
	}
	return nil
}

func (w *orientationWorld) thenDirectToHelp(cmd string) error {
	if !strings.Contains(w.skill, cmd) {
		return fmt.Errorf("skill does not route per-command detail to %q", cmd)
	}
	return nil
}

func (w *orientationWorld) thenNotEnumerateFlags() error {
	if !containsFold(w.skill, "enumerate") {
		return fmt.Errorf("skill does not state it leaves per-command flag enumeration to the CLI")
	}
	return nil
}

func (w *orientationWorld) thenConfirmBeforeWriting() error {
	if !containsFold(w.skill, "confirm") {
		return fmt.Errorf("skill does not state the confirm-before-writing expectation")
	}
	return nil
}

func (w *orientationWorld) then412ReReadReconfirm() error {
	for _, want := range []string{"412", "re-read", "re-confirm", "blind"} {
		if !containsFold(w.skill, want) {
			return fmt.Errorf("skill's 412 guidance is missing %q (expected re-read + re-confirm, not blind retry)", want)
		}
	}
	return nil
}

func (w *orientationWorld) thenNotGate() error {
	if !containsFold(w.skill, "guidance") {
		return fmt.Errorf("skill does not mark write-safety as guidance")
	}
	if !containsFold(w.skill, "does not gate") && !containsFold(w.skill, "not enforcement") {
		return fmt.Errorf("skill does not state it neither blocks nor gates the write")
	}
	return nil
}

// --- Drift-guard scenarios (T003) ------------------------------------------

func (w *orientationWorld) loadLiveFactsAndSkill() error {
	if err := w.ensureSkill(); err != nil {
		return err
	}
	facts, err := LiveOrientationFacts()
	if err != nil {
		return fmt.Errorf("could not extract CLI facts: %w", err)
	}
	w.facts = facts
	return nil
}

// givenCliBehaviourChanged models the CLI growing a new output-format token the
// hand-authored skill does not yet document — a behaviour change the guard must
// catch.
func (w *orientationWorld) givenCliBehaviourChanged() error {
	if err := w.loadLiveFactsAndSkill(); err != nil {
		return err
	}
	w.facts.Formats = append(append([]string{}, w.facts.Formats...), "xml")
	return nil
}

// givenSkillDocumentsDroppedToken models the reverse: the CLI dropped a format
// token the skill still documents.
func (w *orientationWorld) givenSkillDocumentsDroppedToken() error {
	if err := w.loadLiveFactsAndSkill(); err != nil {
		return err
	}
	kept := w.facts.Formats[:0:0]
	for _, t := range w.facts.Formats {
		if t != "yaml" {
			kept = append(kept, t)
		}
	}
	w.facts.Formats = kept
	return nil
}

func (w *orientationWorld) whenDriftChecked() error {
	w.drift = CheckOrientationDrift(w.skill, w.facts)
	return nil
}

func (w *orientationWorld) thenDriftIsDefect() error {
	if len(w.drift) == 0 {
		return fmt.Errorf("guard found no drift, but the CLI facts diverged from the skill")
	}
	return nil
}

func (w *orientationWorld) thenDriftNotTolerated() error {
	// The guard surfaces drift as findings (a CI failure), never as an accepted
	// difference — a non-empty result is exactly that.
	if len(w.drift) == 0 {
		return fmt.Errorf("drift was tolerated rather than reported as a defect")
	}
	return nil
}

func (w *orientationWorld) thenDriftNamesFormatAnchor() error {
	for _, d := range w.drift {
		if containsFold(d, "format") {
			return nil
		}
	}
	return fmt.Errorf("no drift finding named the offending output-format anchor; got %v", w.drift)
}

// mentionsToken reports a case-insensitive word-boundary match for a bare token
// (e.g. a format name) so "json" matches but "jsonschema" does not.
func mentionsToken(skill, token string) bool {
	low := strings.ToLower(skill)
	t := strings.ToLower(token)
	for {
		i := strings.Index(low, t)
		if i < 0 {
			return false
		}
		before := i == 0 || !isWordByte(low[i-1])
		after := i+len(t) >= len(low) || !isWordByte(low[i+len(t)])
		if before && after {
			return true
		}
		low = low[i+len(t):]
	}
}

func isWordByte(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

// mentionsExitCode reports whether the skill documents a code number in its
// backticked exit-code form (“ `7` “), so the digit cannot be confused with an
// HTTP status or version number in prose.
func mentionsExitCode(skill string, code int) bool {
	return strings.Contains(skill, fmt.Sprintf("`%d`", code))
}

func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

// hasFrontmatterField reports whether the SKILL.md leading YAML frontmatter block
// (delimited by the first two `---` lines) declares the given field.
func hasFrontmatterField(skill, field string) bool {
	trimmed := strings.TrimSpace(skill)
	if !strings.HasPrefix(trimmed, "---") {
		return false
	}
	rest := trimmed[len("---"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return false
	}
	front := rest[:end]
	for _, line := range strings.Split(front, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), field+":") {
			return true
		}
	}
	return false
}
