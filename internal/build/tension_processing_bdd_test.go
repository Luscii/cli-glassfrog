package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestTensionProcessingFeatures runs the executable acceptance for the Tension
// Processing Path (066). Like the sibling build-side suites (062/063/064) its
// Paths name ONLY this spec's feature file and it runs with the ~@wip filter, so
// only the scenarios implemented so far execute.
//
// The deliverable is a pair of declarative Claude-plugin artifacts — a thin
// `tension-processing` skill that delegates to a write-capable-but-fenced
// `tension-processor` agent — plus a single-sourced composed-leaf registry and a
// best-effort drift guard. The artifacts carry no runtime Go path of their own,
// so the executable scenarios assert against their content: the skill's when +
// workflow + delegation + write-boundary, the agent's fenced grant +
// isolated-execution + tension-record output contract + defensive processing,
// the ungated-side-of-the-guardrail boundary (exercised against 063's real gate
// script), and the drift guard that keeps the composed tension leaves truthful
// to the shipped CLI and disjoint from 063's gated set (the guard's standalone
// test is T002).
func TestTensionProcessingFeatures(t *testing.T) {
	w := &tensionProcessingWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.register(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unequipped-agent-operators/tension-processing-path.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: tension-processing-path feature scenarios failed")
	}
}

// tensionProcessingWorld is the per-scenario state: the loaded skill + agent
// content, the single-sourced composed leaves, 063's gated set, the CLI's live
// tension subcommand surface, the parsed agent tool grant, and the drift
// findings.
type tensionProcessingWorld struct {
	// skill/agent hold the artifacts with whitespace collapsed to single spaces, so
	// phrase assertions are resilient to markdown line-wrapping (a hard-wrapped
	// "before judging\nduplicates" still matches "before judging duplicates").
	skill string
	agent string
	// skillRaw/agentRaw keep the verbatim content for structural checks that depend
	// on line breaks (frontmatter delimiters, the tools list).
	skillRaw string
	agentRaw string
	composed []string
	gated    []string
	live     []string
	tools    []string
	hasTools bool
	drift    []string
}

func (w *tensionProcessingWorld) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = tensionProcessingWorld{}
		return ctx, nil
	})

	// Rule: Turn a voiced tension into a well-formed record ------------------
	sc.Step(`^a practitioner had voiced a tension and the sensing role it belongs to$`, w.givenLoaded)
	sc.Step(`^the tension-processor captures the tension against that role$`, w.whenLoaded)
	sc.Step(`^it will return the created tension including its ten_ id$`, w.thenReturnsCreatedWithId)
	sc.Step(`^the record will carry that id so the tension can be refined or fed onward$`, w.thenIdCarriesOnward)

	sc.Step(`^a tension already on the record that needed a clearer body or better routing$`, w.givenLoaded)
	sc.Step(`^the tension-processor refines it through the update command$`, w.whenRefinesViaUpdate)
	sc.Step(`^it will return the updated tension$`, w.thenReturnsUpdated)
	sc.Step(`^it will not recapture the tension as a second entry$`, w.thenNoRecapture)

	sc.Step(`^a tension the practitioner decided was moot$`, w.givenLoaded)
	sc.Step(`^the tension-processor processes that decision$`, w.whenLoaded)
	sc.Step(`^it will retire the tension through the discard command$`, w.thenRetiresViaDiscard)
	sc.Step(`^it will not push the tension toward a proposal$`, w.thenNotPushedToProposal)

	sc.Step(`^a capture attempt with an unknown sensing role or a blank body$`, w.givenLoaded)
	sc.Step(`^the tension-processor runs the capture command$`, w.whenLoaded)
	sc.Step(`^it will surface the usage or API failure by name$`, w.thenSurfacesRejectionByName)
	sc.Step(`^it will record nothing$`, w.thenRecordsNothing)
	sc.Step(`^it will fabricate no ten_ id the record does not contain$`, w.thenFabricatesNoId)

	sc.Step(`^the plugin was present with the tension-processor agent registered$`, w.givenAgentRegistered)
	sc.Step(`^the tension-processing skill delegates a voiced tension for processing$`, w.whenSkillDelegates)
	sc.Step(`^the processor will run the workflow in its own context$`, w.thenRunsOwnContext)
	sc.Step(`^it will return only the tension record to the caller$`, w.thenReturnsOnlyRecord)

	sc.Step(`^the plugin was present but the tension-processor agent was absent or unregistered$`, w.givenAgentAbsent)
	sc.Step(`^the tension-processing skill is consulted for a tension$`, w.whenSkillConsulted)
	sc.Step(`^its workflow will remain readable as guidance the caller can follow by hand$`, w.thenWorkflowStandsAsGuidance)
	sc.Step(`^no command in the CLI will be broken by the agent's absence$`, w.thenNoCLIBroken)

	sc.Step(`^the record the tension-processor returned$`, w.givenLoaded)
	sc.Step(`^it is compared against the raw command output$`, w.whenLoaded)
	sc.Step(`^it will be a drawn-together tension record carrying ids$`, w.thenDrawnTogetherRecord)
	sc.Step(`^it will not be a concatenation of unsynthesized dumps$`, w.thenNotConcatenation)

	// Rule: Show what's already sensed before capturing a duplicate ----------
	sc.Step(`^a sensing role whose already-sensed tensions and rolled-up sub-role tensions span several pages$`, w.givenLoaded)
	sc.Step(`^the tension-processor situates the voiced tension against them$`, w.whenSituates)
	sc.Step(`^it will page through the full result set before judging duplicates$`, w.thenPagesBeforeJudging)
	sc.Step(`^it will present the tensions drawn together so the practitioner can see what is already on the record$`, w.thenPresentsDrawnTogether)
	sc.Step(`^it will treat the new capture as a deliberate addition rather than a blind duplicate$`, w.thenDeliberateAddition)

	sc.Step(`^a voiced tension that matched one already on the record$`, w.givenLoaded)
	sc.Step(`^the tension-processor situates it before capturing$`, w.whenLoaded)
	sc.Step(`^it will surface the existing tension with its id$`, w.thenSurfacesExisting)
	sc.Step(`^it will let the practitioner refine that one$`, w.thenLetsRefineExisting)
	sc.Step(`^it will not silently record a duplicate$`, w.thenNoSilentDuplicate)

	sc.Step(`^a situating step in which the sub-role roll-up failed while the other reads succeeded$`, w.givenLoaded)
	sc.Step(`^the tension-processor continues$`, w.whenLoaded)
	sc.Step(`^it will surface what the failure was$`, w.thenSurfacesFailure)
	sc.Step(`^it will present the tensions the reads that succeeded returned$`, w.thenPresentsSucceededReads)
	sc.Step(`^it will not invent the missing tensions or abandon the whole record$`, w.thenNotInventNorAbandon)

	// Rule: Move a processed tension toward a governance change --------------
	sc.Step(`^a captured, well-formed tension the practitioner wanted to turn into a proposal$`, w.givenLoaded)
	sc.Step(`^the tension-processor completes its processing$`, w.whenLoaded)
	sc.Step(`^it will hand the ten_ id to the Proposal Drafting Path$`, w.thenHandsOffId)
	sc.Step(`^it will not draft, create, or circulate the proposal itself$`, w.thenNeverDrafts)

	sc.Step(`^the tension-processing skill and agent content$`, w.givenLoaded)
	sc.Step(`^it is inspected for any proposal create, advance, withdraw, or response step$`, w.whenInspectForProposalStep)
	sc.Step(`^it will contain none$`, w.thenContainsNoProposalStep)
	sc.Step(`^it will hand the proposal work to the Proposal Drafting Path$`, w.thenHandsProposalWork)

	sc.Step(`^the path's treatment of its writes$`, w.givenLoaded)
	sc.Step(`^it is inspected against the Write-Safety Guardrail$`, w.whenInspectAgainstGuardrail)
	sc.Step(`^it will neither gate the operational tension edits behind the governance-write confirmation$`, w.thenTensionEditsUngated)
	sc.Step(`^it will perform no governance write that would require it$`, w.thenNoGovernanceWrite)

	sc.Step(`^it is inspected for an authority verdict or Holacracy coaching$`, w.whenLoaded)
	sc.Step(`^it will neither rule on whether a tension needs a proposal$`, w.thenNoAuthorityVerdict)
	sc.Step(`^it will not advise on governance craft$`, w.thenNoCoaching)

	// Drift guard (standalone test is T002) ----------------------------------
	sc.Step(`^the produced tension-processing-path content$`, w.givenProcessingContent)
	sc.Step(`^every command it composes is checked against the shipped CLI$`, w.whenCheckedAgainstCLI)
	sc.Step(`^each one will exist$`, w.thenEachExists)
	sc.Step(`^the path will name no command the CLI does not expose$`, w.thenNamesNoLackingCommand)
}

// --- Loading -----------------------------------------------------------------

func (w *tensionProcessingWorld) ensureSkillLoaded() error {
	if w.skillRaw == "" {
		skill, err := ReadTensionSkill()
		if err != nil {
			return fmt.Errorf("tension-processing skill did not load: %w", err)
		}
		w.skillRaw, w.skill = skill, normalizeWS(skill)
	}
	return nil
}

func (w *tensionProcessingWorld) ensureAgentLoaded() error {
	if w.agentRaw == "" {
		agent, err := ReadTensionProcessorAgent()
		if err != nil {
			return fmt.Errorf("tension-processor agent did not load: %w", err)
		}
		w.agentRaw, w.agent = agent, normalizeWS(agent)
	}
	return nil
}

// ensureLoaded loads both artifacts — used by every scenario except the
// missing-processor degradation path, which loads only the skill (see
// givenAgentAbsent) so it faithfully models the agent being absent.
func (w *tensionProcessingWorld) ensureLoaded() error {
	if err := w.ensureSkillLoaded(); err != nil {
		return err
	}
	return w.ensureAgentLoaded()
}

func (w *tensionProcessingWorld) givenLoaded() error { return w.ensureLoaded() }
func (w *tensionProcessingWorld) whenLoaded() error  { return w.ensureLoaded() }

// combined returns the (normalized) skill and agent content together — most
// behavioral scenarios inspect "the processor's behavior" as described across
// both artifacts.
func (w *tensionProcessingWorld) combined() string { return w.skill + " " + w.agent }

// --- Rule: voiced tension → well-formed record --------------------------------

func (w *tensionProcessingWorld) thenReturnsCreatedWithId() error {
	if !mentionsToken(w.combined(), "tension create") {
		return fmt.Errorf("the workflow does not capture through the tension create command")
	}
	if !containsFold(w.agent, "ten_") {
		return fmt.Errorf("the output contract does not return the created tension's ten_ id")
	}
	return nil
}

func (w *tensionProcessingWorld) thenIdCarriesOnward() error {
	if !containsFold(w.agent, "refined or fed onward") {
		return fmt.Errorf("the output contract does not state the record carries the id so the tension can be refined or fed onward")
	}
	return nil
}

func (w *tensionProcessingWorld) whenRefinesViaUpdate() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !mentionsToken(w.combined(), "tension update") {
		return fmt.Errorf("the workflow does not refine through the tension update command")
	}
	return nil
}

func (w *tensionProcessingWorld) thenReturnsUpdated() error {
	if !mentionsToken(w.agent, "refined") {
		return fmt.Errorf("the output contract has no refined action for an updated tension")
	}
	return nil
}

func (w *tensionProcessingWorld) thenNoRecapture() error {
	if !containsFold(w.combined(), "recaptur") || !containsFold(w.combined(), "second entry") {
		return fmt.Errorf("the workflow does not refine in place rather than recapturing a second entry")
	}
	return nil
}

func (w *tensionProcessingWorld) thenRetiresViaDiscard() error {
	if !mentionsToken(w.combined(), "tension discard") || !containsFold(w.combined(), "retire") {
		return fmt.Errorf("the workflow does not retire a moot tension through the tension discard command")
	}
	return nil
}

func (w *tensionProcessingWorld) thenNotPushedToProposal() error {
	if !containsFold(w.combined(), "rather than pushing it toward a proposal") {
		return fmt.Errorf("the workflow does not state a moot tension is retired rather than pushed toward a proposal")
	}
	return nil
}

func (w *tensionProcessingWorld) thenSurfacesRejectionByName() error {
	if !containsFold(w.agent, "usage or API failure by name") {
		return fmt.Errorf("the agent does not surface a rejected capture's usage or API failure by name")
	}
	return nil
}

func (w *tensionProcessingWorld) thenRecordsNothing() error {
	if !containsFold(w.agent, "records nothing") {
		return fmt.Errorf("the agent does not state a rejected capture records nothing")
	}
	return nil
}

func (w *tensionProcessingWorld) thenFabricatesNoId() error {
	if !containsFold(w.agent, "fabricate no") || !containsFold(w.agent, "ten_") {
		return fmt.Errorf("the agent does not state it fabricates no ten_ id on a rejected capture")
	}
	return nil
}

// --- Rule: reachable / degrades -----------------------------------------------

func (w *tensionProcessingWorld) givenAgentRegistered() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	// Discoverability: the agent carries the frontmatter (name/description/tools)
	// the host needs, and — like the skill — it is auto-discovered from
	// plugin/agents/, so the manifest declares NO `agents` key.
	for _, field := range []string{"name", "description", "tools"} {
		if !hasFrontmatterField(w.agentRaw, field) {
			return fmt.Errorf("tension-processor frontmatter lacks the %q field the host needs to register it", field)
		}
	}
	_, raw, err := ReadOrientationManifest()
	if err != nil {
		return fmt.Errorf("plugin manifest did not load: %w", err)
	}
	if !ManifestDemandsNoSetup(raw) {
		return fmt.Errorf("plugin.json declares a setup-forcing key (e.g. `agents`) — the processor must be auto-discovered from plugin/agents/, not registered via a manifest key")
	}
	return nil
}

func (w *tensionProcessingWorld) whenSkillDelegates() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !containsFold(w.skill, "tension-processor") || !containsFold(w.skill, "delegat") {
		return fmt.Errorf("the skill does not delegate processing to the tension-processor agent")
	}
	return nil
}

func (w *tensionProcessingWorld) thenRunsOwnContext() error {
	if !containsFold(w.agent, "isolated context") {
		return fmt.Errorf("the agent does not state it runs the processing in its own isolated context")
	}
	return nil
}

func (w *tensionProcessingWorld) thenReturnsOnlyRecord() error {
	if !containsFold(w.agent, "tension record") {
		return fmt.Errorf("the agent output contract never names the tension record it returns")
	}
	if !containsFold(w.agent, "only the tension record") && !containsFold(w.agent, "return **only**") {
		return fmt.Errorf("the agent does not state it returns ONLY the tension record (raw output stays in its context)")
	}
	return nil
}

// givenAgentAbsent models the processor agent being absent/unregistered: it
// loads ONLY the skill. The degradation scenario asserts the skill's workflow
// stands alone as guidance and no CLI command breaks — neither needs the agent,
// and loading it would both contradict the premise and fail outright if the
// agent file were genuinely gone.
func (w *tensionProcessingWorld) givenAgentAbsent() error { return w.ensureSkillLoaded() }

func (w *tensionProcessingWorld) whenSkillConsulted() error { return w.ensureSkillLoaded() }

func (w *tensionProcessingWorld) thenWorkflowStandsAsGuidance() error {
	// The skill body carries the processing steps itself (usable standalone) and
	// states the degradation explicitly.
	if !containsFold(w.skill, "the workflow") {
		return fmt.Errorf("the skill does not carry the workflow steps as consultable guidance")
	}
	for _, step := range []string{"tension list", "tension subroles", "tension create", "tension update", "tension discard"} {
		if !mentionsToken(w.skill, step) {
			return fmt.Errorf("the skill workflow is not self-contained guidance — it omits the %q step", step)
		}
	}
	if !containsFold(w.skill, "follow by hand") && !containsFold(w.skill, "degrades to guidance") {
		return fmt.Errorf("the skill does not state it remains usable as guidance when the agent is absent")
	}
	return nil
}

func (w *tensionProcessingWorld) thenNoCLIBroken() error {
	// The plugin tree is pure data: nothing under plugin/ compiles into the CLI, so
	// an absent/unregistered agent cannot break a glassfrog command.
	clean, err := OrientationPluginHasNoGoCode()
	if err != nil {
		return fmt.Errorf("could not inspect the plugin tree: %w", err)
	}
	if !clean {
		return fmt.Errorf("plugin tree carries Go code — the agent's absence could no longer be isolated from the CLI")
	}
	return nil
}

// --- Rule: synthesized record --------------------------------------------------

func (w *tensionProcessingWorld) thenDrawnTogetherRecord() error {
	if !containsFold(w.agent, "drawn-together") {
		return fmt.Errorf("the output contract is not framed as a drawn-together tension record")
	}
	for _, el := range []string{"tension", "situating", "action", "handoff", "notes"} {
		if !mentionsToken(w.agent, el) {
			return fmt.Errorf("the drawn-together record omits the %q element", el)
		}
	}
	if !containsFold(w.agent, "ten_") || !containsFold(w.agent, "role_") {
		return fmt.Errorf("the record does not carry the concrete ten_/role_ ids per element")
	}
	return nil
}

func (w *tensionProcessingWorld) thenNotConcatenation() error {
	if !containsFold(w.agent, "concatenation") {
		return fmt.Errorf("the agent does not state the record is not a concatenation of unsynthesized dumps")
	}
	return nil
}

// --- Rule: situate before capturing --------------------------------------------

func (w *tensionProcessingWorld) whenSituates() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !mentionsToken(w.combined(), "tension list") || !mentionsToken(w.combined(), "tension subroles") {
		return fmt.Errorf("the workflow does not situate through tension list and tension subroles")
	}
	return nil
}

func (w *tensionProcessingWorld) thenPagesBeforeJudging() error {
	if !containsFold(w.combined(), "full result set") || !containsFold(w.combined(), "before judging duplicates") {
		return fmt.Errorf("the workflow does not page through the full result set before judging duplicates")
	}
	return nil
}

func (w *tensionProcessingWorld) thenPresentsDrawnTogether() error {
	if !containsFold(w.agent, "drawn together") || !containsFold(w.agent, "already on the record") {
		return fmt.Errorf("the agent does not present the situating tensions drawn together so the practitioner sees what is already on the record")
	}
	return nil
}

func (w *tensionProcessingWorld) thenDeliberateAddition() error {
	if !containsFold(w.combined(), "deliberate") {
		return fmt.Errorf("the workflow does not treat the new capture as a deliberate addition")
	}
	return nil
}

func (w *tensionProcessingWorld) thenSurfacesExisting() error {
	if !containsFold(w.combined(), "existing tension") || !containsFold(w.combined(), "with its id") {
		return fmt.Errorf("the workflow does not surface the existing tension with its id on a duplicate")
	}
	return nil
}

func (w *tensionProcessingWorld) thenLetsRefineExisting() error {
	if !containsFold(w.combined(), "refine that one") {
		return fmt.Errorf("the workflow does not let the practitioner refine the existing tension")
	}
	return nil
}

func (w *tensionProcessingWorld) thenNoSilentDuplicate() error {
	if !containsFold(w.combined(), "silently record a") {
		return fmt.Errorf("the workflow does not state it never silently records a duplicate")
	}
	return nil
}

func (w *tensionProcessingWorld) thenSurfacesFailure() error {
	if !containsFold(w.agent, "surface what the failure was") {
		return fmt.Errorf("the agent does not surface what the failure was on a partial situating read")
	}
	return nil
}

func (w *tensionProcessingWorld) thenPresentsSucceededReads() error {
	if !containsFold(w.agent, "reads that succeeded") {
		return fmt.Errorf("the agent does not return the record built from the reads that succeeded")
	}
	return nil
}

func (w *tensionProcessingWorld) thenNotInventNorAbandon() error {
	if !containsFold(w.agent, "invent") || !containsFold(w.agent, "whole record") {
		return fmt.Errorf("the agent does not state it neither invents the missing tensions nor abandons the whole record")
	}
	return nil
}

// --- Rule: handoff, operational writes only, ungated side, no judging ----------

func (w *tensionProcessingWorld) thenHandsOffId() error {
	if !containsFold(w.combined(), "Proposal Drafting Path") {
		return fmt.Errorf("the workflow does not hand the ready tension to the Proposal Drafting Path by its in-plugin name")
	}
	if !containsFold(w.agent, "handoff") || !containsFold(w.agent, "ten_") {
		return fmt.Errorf("the record has no handoff element carrying the ten_ id for the drafting path")
	}
	return nil
}

func (w *tensionProcessingWorld) thenNeverDrafts() error {
	if !containsFold(w.agent, "draft") || !containsFold(w.agent, "circulate") {
		return fmt.Errorf("the agent does not state it never drafts, creates, or circulates the proposal itself")
	}
	return nil
}

// whenInspectForProposalStep loads 063's gated registry — the source-derived
// enumeration of what a proposal-write step IS — alongside the artifacts, so the
// contains-none check keys on the guardrail's own list rather than a hand-coded
// copy of it.
func (w *tensionProcessingWorld) whenInspectForProposalStep() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	gated, err := ReadGatedRegistry()
	if err != nil {
		return fmt.Errorf("could not read 063's gated-command registry: %w", err)
	}
	if len(gated) == 0 {
		return fmt.Errorf("063's gated registry lists no leaves — nothing to inspect the artifacts against")
	}
	w.gated = gated
	return nil
}

func (w *tensionProcessingWorld) thenContainsNoProposalStep() error {
	// No gated proposal-write leaf ("proposal create", "proposal propose", …) may
	// appear anywhere in the artifacts — naming one as a step would put the path
	// on the gated governance-write side it must never cross into.
	for _, leaf := range w.gated {
		if containsFold(w.combined(), leaf) {
			return fmt.Errorf("the artifacts contain the gated proposal-write step %q — the path must perform only operational tension writes", leaf)
		}
	}
	return nil
}

func (w *tensionProcessingWorld) thenHandsProposalWork() error {
	if !containsFold(w.combined(), "Proposal Drafting Path") {
		return fmt.Errorf("the artifacts do not hand the proposal work to the Proposal Drafting Path")
	}
	return nil
}

// whenInspectAgainstGuardrail exercises the path's writes against 063's REAL
// PreToolUse gate script: every composed tension write must pass ungated, so
// the boundary claim is checked against the guardrail itself, not just prose.
func (w *tensionProcessingWorld) whenInspectAgainstGuardrail() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	w.tools, w.hasTools = AgentTools(w.agentRaw)
	gated, err := ReadGatedRegistry()
	if err != nil {
		return fmt.Errorf("could not read 063's gated-command registry: %w", err)
	}
	w.gated = gated
	composed, err := ReadTensionCommands()
	if err != nil {
		return fmt.Errorf("could not read the composed-leaf registry: %w", err)
	}
	w.composed = composed
	return nil
}

func (w *tensionProcessingWorld) thenTensionEditsUngated() error {
	// The operational tension writes pass 063's gate without the governance-write
	// confirmation — asked (gated) would contradict 063's Behavioral Accord.
	for _, cmd := range []string{
		"glassfrog tension create role_1 --body 'a gap'",
		"glassfrog tension update ten_1 --body 'sharper'",
		"glassfrog tension discard ten_1",
	} {
		dec, _, err := runGateScript(cmd)
		if err != nil {
			return fmt.Errorf("063's gate errored on %q: %v", cmd, err)
		}
		if dec == "ask" {
			return fmt.Errorf("%q was gated (ask) — the operational tension edits must pass the guardrail ungated", cmd)
		}
	}
	// And the artifacts describe that side correctly: ungated by design, no
	// invented confirmation gate.
	if !containsFold(w.combined(), "ungated") {
		return fmt.Errorf("the artifacts do not state the operational tension writes run ungated")
	}
	if !containsFold(w.combined(), "must not invent one") {
		return fmt.Errorf("the artifacts do not state the path must not invent a confirmation gate of its own")
	}
	return nil
}

func (w *tensionProcessingWorld) thenNoGovernanceWrite() error {
	// Structurally: the tool grant withholds Write/Edit (no workspace mutation)
	// while keeping Bash (the writes it DOES run are the ungated tension edits).
	if !w.hasTools {
		return fmt.Errorf("the agent declares no tools grant — the fenced write capability cannot be asserted structurally")
	}
	hasBash := false
	for _, tool := range w.tools {
		if strings.EqualFold(tool, "Bash") {
			hasBash = true
		}
		for _, forbidden := range []string{"Write", "Edit"} {
			if strings.EqualFold(tool, forbidden) {
				return fmt.Errorf("the agent tool grant includes %q — the processor must not be able to mutate the workspace", forbidden)
			}
		}
	}
	if !hasBash {
		return fmt.Errorf("the agent tool grant omits Bash — it could not invoke glassfrog at all (a misconfiguration, not a safety win)")
	}
	// The composed set stays disjoint from the gated governance-write set, and no
	// gated leaf appears in the artifacts as a step.
	gatedSet := map[string]bool{}
	for _, g := range w.gated {
		gatedSet[g] = true
	}
	for _, leaf := range w.composed {
		if gatedSet[leaf] {
			return fmt.Errorf("composed leaf %q is in 063's gated registry — the path would perform a governance write requiring confirmation", leaf)
		}
	}
	for _, leaf := range w.gated {
		if containsFold(w.combined(), leaf) {
			return fmt.Errorf("the artifacts contain the gated governance write %q", leaf)
		}
	}
	return nil
}

func (w *tensionProcessingWorld) thenNoAuthorityVerdict() error {
	if !containsFold(w.combined(), "needs a proposal") || !containsFold(w.combined(), "do not rule") && !containsFold(w.combined(), "does not rule") {
		return fmt.Errorf("the artifacts do not explicitly disclaim ruling on whether a tension needs a proposal")
	}
	if !containsFold(w.combined(), "Constraint Discovery Path") {
		return fmt.Errorf("the artifacts do not hand the authority question to the Constraint Discovery Path by its in-plugin name")
	}
	// No authority verdict may appear: a ruling would read as "you are (not)
	// allowed / permitted" or "this needs / does not need a proposal" stated as a
	// verdict.
	for _, verdict := range []string{
		"you are allowed to",
		"you are not allowed to",
		"you are permitted to",
		"permission granted",
		"permission denied",
		"you may proceed with",
	} {
		if containsFold(w.combined(), verdict) {
			return fmt.Errorf("the artifacts carry an authority verdict %q — the path must process, not judge", verdict)
		}
	}
	return nil
}

func (w *tensionProcessingWorld) thenNoCoaching() error {
	if !containsFold(w.combined(), "advise on governance craft") || !containsFold(w.combined(), "coach") {
		return fmt.Errorf("the artifacts do not disclaim advising on governance craft or coaching Holacracy practice")
	}
	return nil
}

// --- Drift guard (standalone test is T002) -------------------------------------

func (w *tensionProcessingWorld) givenProcessingContent() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	composed, err := ReadTensionCommands()
	if err != nil {
		return fmt.Errorf("could not read the composed-leaf registry: %w", err)
	}
	if len(composed) == 0 {
		return fmt.Errorf("the composed-leaf registry is empty — nothing to check against the CLI")
	}
	w.composed = composed
	return nil
}

func (w *tensionProcessingWorld) whenCheckedAgainstCLI() error {
	live, err := LiveTensionSubcommands()
	if err != nil {
		return fmt.Errorf("could not extract the CLI's tension subcommand surface: %w", err)
	}
	if len(live) == 0 {
		return fmt.Errorf("extracted no tension subcommands — the surface anchor could not be read")
	}
	w.live = live
	gated, err := ReadGatedRegistry()
	if err != nil {
		return fmt.Errorf("could not read 063's gated-command registry: %w", err)
	}
	w.gated = gated
	w.drift = CheckTensionProcessingDrift(w.composed, w.live, w.gated, w.agent)
	return nil
}

func (w *tensionProcessingWorld) thenEachExists() error {
	if len(w.drift) != 0 {
		return fmt.Errorf("a composed tension leaf no longer resolves in the shipped CLI (or violates the ungated invariant):\n  - %s", joinDrift(w.drift))
	}
	live := map[string]bool{}
	for _, s := range w.live {
		live[s] = true
	}
	for _, leaf := range w.composed {
		fields := strings.Fields(leaf)
		if len(fields) != 2 || fields[0] != "tension" || !live[fields[1]] {
			return fmt.Errorf("composed leaf %q does not exist as a subcommand of the CLI's tension command", leaf)
		}
	}
	return nil
}

func (w *tensionProcessingWorld) thenNamesNoLackingCommand() error {
	// The same drift result proves the path names no command the CLI does not
	// expose.
	if len(w.drift) != 0 {
		return fmt.Errorf("the path names a command the CLI does not expose:\n  - %s", joinDrift(w.drift))
	}
	return nil
}
