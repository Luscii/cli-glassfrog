package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestProposalCirculationFeatures runs the executable acceptance for the Proposal
// Circulation Path (068). Like the sibling build-side suites (062–067) its Paths
// name ONLY this spec's feature file and it runs with the ~@wip filter, so only
// the scenarios implemented so far execute.
//
// The deliverable is a pair of declarative Claude-plugin artifacts — a thin
// `proposal-circulation` skill that delegates to a write-capable-but-fenced
// `proposal-circulator` agent — plus a single-sourced composed-leaf registry and a
// best-effort drift guard. The artifacts carry no runtime Go path of their own, so
// the executable scenarios assert against their content: the skill's when +
// workflow + delegation + gated-writes note, the agent's fenced grant +
// isolated-execution + circulation-record output contract + defensive
// circulation, the two-writes-through-the-guardrail boundary (exercised against
// 063's real gate script), the reads-inform-never-gate discipline, and the drift
// guard that keeps the composed leaves truthful to the shipped CLI and pins both
// transitions' membership in 063's gated set (the guard's standalone test is
// T002).
func TestProposalCirculationFeatures(t *testing.T) {
	w := &proposalCirculationWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.register(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unequipped-agent-operators/proposal-circulation-path.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: proposal-circulation-path feature scenarios failed")
	}
}

// proposalCirculationWorld is the per-scenario state: the loaded skill + agent
// content, the single-sourced composed leaves, 063's gated set, the CLI's live
// proposal subcommand surface, the parsed agent tool grant, and the drift
// findings.
type proposalCirculationWorld struct {
	// skill/agent hold the artifacts with whitespace collapsed to single spaces, so
	// phrase assertions are resilient to markdown line-wrapping (a hard-wrapped
	// "confirmed write\nflow" still matches "confirmed write flow").
	skill string
	agent string
	// skillRaw/agentRaw keep the verbatim content for structural checks that depend
	// on line breaks (frontmatter delimiters, the tools list).
	skillRaw     string
	agentRaw     string
	composed     []string
	gated        []string
	liveProposal []string
	tools        []string
	hasTools     bool
	drift        []string
}

func (w *proposalCirculationWorld) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = proposalCirculationWorld{}
		return ctx, nil
	})

	// Rule: Advance a prepared draft without hand-assembling the transition ---
	sc.Step(`^a draft proposal's prp_ id whose available transitions included propose$`, w.givenLoaded)
	sc.Step(`^the proposal-circulator surfaces it for confirmation and runs the propose through the confirmed write flow$`, w.whenProposesThroughConfirmedFlow)
	sc.Step(`^it will return the proposal now in proposed_outside_meeting, carrying its response deadline and the implicit no-objection$`, w.thenReturnsAdvanced)
	sc.Step(`^the record will carry the prp_ id so the proposal can be monitored or withdrawn$`, w.thenIdCarriesForward)

	sc.Step(`^a proposal ready to propose or withdraw$`, w.givenLoaded)
	sc.Step(`^the proposal-circulator reaches the transition$`, w.whenReachesTransition)
	sc.Step(`^it will route the write through the Write-Safety Guardrail's confirmed flow, surfacing the proposal first$`, w.thenRoutesThroughGuardrail)
	sc.Step(`^no transition will happen when the write is not confirmed$`, w.thenNoTransitionWhenDeclined)

	sc.Step(`^a propose attempt whose transition the server did not allow$`, w.givenLoaded)
	sc.Step(`^the proposal-circulator runs the transition through the confirmed write flow$`, w.whenProposesThroughConfirmedFlow)
	sc.Step(`^it will surface the API failure by name$`, w.thenSurfacesAPIFailure)
	sc.Step(`^it will transition nothing$`, w.thenTransitionsNothing)
	sc.Step(`^it will fabricate no state the record does not contain$`, w.thenFabricatesNoState)

	sc.Step(`^a proposal whose available transitions the proposal-circulator had read$`, w.givenLoaded)
	sc.Step(`^it advances or withdraws the proposal$`, w.whenAdvancesOrWithdraws)
	sc.Step(`^it will issue the transition and let the server authorize$`, w.thenIssuesAndServerAuthorizes)
	sc.Step(`^it will surface a 422 refusal plainly when the server refuses$`, w.thenSurfaces422Plainly)
	sc.Step(`^it will not pre-gate the call on the read snapshot$`, w.thenNoPreGateOnSnapshot)

	sc.Step(`^the plugin was present with the proposal-circulator agent registered$`, w.givenAgentRegistered)
	sc.Step(`^the proposal-circulation skill delegates a circulation act$`, w.whenSkillDelegates)
	sc.Step(`^the circulator will run the workflow in its own context$`, w.thenRunsOwnContext)
	sc.Step(`^it will return only the circulation record to the caller$`, w.thenReturnsOnlyRecord)

	sc.Step(`^the plugin was present but the proposal-circulator agent was absent or unregistered$`, w.givenAgentAbsent)
	sc.Step(`^the proposal-circulation skill is consulted for a circulation act$`, w.whenSkillConsulted)
	sc.Step(`^its workflow will remain readable as guidance the caller can follow by hand$`, w.thenWorkflowStandsAsGuidance)
	sc.Step(`^no command in the CLI will be broken by the agent's absence$`, w.thenNoCLIBroken)

	sc.Step(`^the path's treatment of the propose and withdraw transitions$`, w.givenLoaded)
	sc.Step(`^each is inspected against the Write-Safety Guardrail$`, w.whenInspectAgainstGuardrail)
	sc.Step(`^both will run through the confirmed write flow$`, w.thenBothGatedRunConfirmed)
	sc.Step(`^the path will not issue either gated transition as if it were ungated$`, w.thenNotUngated)

	sc.Step(`^the path's use of available transitions$`, w.givenLoaded)
	sc.Step(`^it is inspected for a client-side transition gate$`, w.whenLoaded)
	sc.Step(`^it will read to show the proposer where the proposal stands$`, w.thenReadsToShowProposer)
	sc.Step(`^it will not pre-gate the call client-side$`, w.thenNoPreGateClientSide)

	// Rule: See where a circulating proposal stands, drawn together -----------
	sc.Step(`^the prp_ id of a proposal already circulating$`, w.givenLoaded)
	sc.Step(`^the proposal-circulator reads it back$`, w.whenReadsBack)
	sc.Step(`^it will surface the response summary, response deadline, and available transitions drawn together$`, w.thenSurfacesDrawnTogether)
	sc.Step(`^it will not compute acceptance itself$`, w.thenNotComputeAcceptance)

	sc.Step(`^a monitoring step in which the proposal list read failed mid-walk$`, w.givenLoaded)
	sc.Step(`^the proposal-circulator continues$`, w.whenLoaded)
	sc.Step(`^it will surface what the failure was$`, w.thenSurfacesFailure)
	sc.Step(`^it will present what it gathered so far, flagged incomplete$`, w.thenPresentsGathered)
	sc.Step(`^it will not invent the missing data or abandon the whole result$`, w.thenNotInventNorAbandon)

	sc.Step(`^a circulating proposal awaiting consent$`, w.givenLoaded)
	sc.Step(`^a member wants to record a no-objection or bring-to-meeting response$`, w.whenLoaded)
	sc.Step(`^the proposal-circulator will name the response side as where that act belongs$`, w.thenNamesResponseSide)
	sc.Step(`^it will not record the response itself$`, w.thenNotRecordResponse)

	// Shared Given for the two "skill and agent content" validation scenarios.
	sc.Step(`^the proposal-circulation skill and agent content$`, w.givenLoaded)

	sc.Step(`^it is inspected for any no-objection or bring-to-meeting record step$`, w.whenInspectForResponseStep)
	sc.Step(`^it will contain none$`, w.thenContainsNoResponseStep)
	sc.Step(`^recording a response will remain the response side's act$`, w.thenResponseRemainsResponseSide)

	sc.Step(`^the record the proposal-circulator returned$`, w.givenLoaded)
	sc.Step(`^it is compared against the raw command output$`, w.whenLoaded)
	sc.Step(`^it will be a drawn-together circulation picture carrying the prp_ id$`, w.thenDrawnTogetherRecord)
	sc.Step(`^it will not be a concatenation of unsynthesized dumps$`, w.thenNotConcatenation)

	// Rule: Pull a circulating proposal back to draft without losing my place -
	sc.Step(`^a circulating proposal the proposer wanted to amend$`, w.givenLoaded)
	sc.Step(`^the proposal-circulator surfaces it for confirmation and runs the withdraw through the confirmed write flow$`, w.whenWithdrawsThroughConfirmedFlow)
	sc.Step(`^it will return the proposal now back in draft$`, w.thenReturnsBackInDraft)
	sc.Step(`^it will hand the prp_ id back to the Proposal Drafting Path for re-editing$`, w.thenHandsBackToDrafting)

	sc.Step(`^a session that advanced a draft and later withdrew the circulating proposal$`, w.givenLoaded)
	sc.Step(`^the proposal-circulator runs each transition$`, w.whenRunsEachTransition)
	sc.Step(`^each transition will pass through its own confirmed write flow$`, w.thenEachOwnConfirmedFlow)
	sc.Step(`^neither confirmation will batch or pre-authorize the other$`, w.thenNeitherBatchNorPreauthorize)

	sc.Step(`^it is inspected for an authority verdict or Holacracy coaching$`, w.whenLoaded)
	sc.Step(`^it will not rule on whether the change is within authority$`, w.thenNoAuthorityVerdict)
	sc.Step(`^it will not advise on governance craft$`, w.thenNoCoaching)

	// Drift guard (standalone test is T002) ----------------------------------
	sc.Step(`^the produced proposal-circulation-path content$`, w.givenCirculationContent)
	sc.Step(`^every command it composes is checked against the shipped CLI$`, w.whenCheckedAgainstCLI)
	sc.Step(`^each one will exist$`, w.thenEachExists)
	sc.Step(`^the path will name no command the CLI does not expose$`, w.thenNamesNoLackingCommand)
}

// --- Loading -----------------------------------------------------------------

func (w *proposalCirculationWorld) ensureSkillLoaded() error {
	if w.skillRaw == "" {
		skill, err := ReadProposalCirculationSkill()
		if err != nil {
			return fmt.Errorf("proposal-circulation skill did not load: %w", err)
		}
		w.skillRaw, w.skill = skill, normalizeWS(skill)
	}
	return nil
}

func (w *proposalCirculationWorld) ensureAgentLoaded() error {
	if w.agentRaw == "" {
		agent, err := ReadProposalCirculatorAgent()
		if err != nil {
			return fmt.Errorf("proposal-circulator agent did not load: %w", err)
		}
		w.agentRaw, w.agent = agent, normalizeWS(agent)
	}
	return nil
}

// ensureLoaded loads both artifacts — used by every scenario except the
// missing-circulator degradation path, which loads only the skill (see
// givenAgentAbsent) so it faithfully models the agent being absent.
func (w *proposalCirculationWorld) ensureLoaded() error {
	if err := w.ensureSkillLoaded(); err != nil {
		return err
	}
	return w.ensureAgentLoaded()
}

func (w *proposalCirculationWorld) givenLoaded() error { return w.ensureLoaded() }
func (w *proposalCirculationWorld) whenLoaded() error  { return w.ensureLoaded() }

// combined returns the (normalized) skill and agent content together — most
// behavioral scenarios inspect "the circulator's behavior" as described across
// both artifacts.
func (w *proposalCirculationWorld) combined() string { return w.skill + " " + w.agent }

// --- Rule: advance a prepared draft ---------------------------------------------

func (w *proposalCirculationWorld) whenProposesThroughConfirmedFlow() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !mentionsToken(w.combined(), "proposal propose") {
		return fmt.Errorf("the workflow does not advance through the proposal propose command")
	}
	if !containsFold(w.combined(), "confirmed write flow") {
		return fmt.Errorf("the workflow does not run the transition through the confirmed write flow")
	}
	if !containsFold(w.agent, "narrate the proposal") {
		return fmt.Errorf("the agent does not surface (narrate) the proposal before the transition")
	}
	return nil
}

func (w *proposalCirculationWorld) thenReturnsAdvanced() error {
	if !containsFold(w.combined(), "proposed_outside_meeting") {
		return fmt.Errorf("the artifacts do not return the advanced proposal in proposed_outside_meeting")
	}
	if !containsFold(w.combined(), "response_deadline") {
		return fmt.Errorf("the advanced proposal does not carry its response_deadline")
	}
	if !containsFold(w.combined(), "implicit") || !containsFold(w.combined(), "no_objection") {
		return fmt.Errorf("the advanced proposal does not carry the proposer's implicit no_objection")
	}
	return nil
}

func (w *proposalCirculationWorld) thenIdCarriesForward() error {
	if !containsFold(w.agent, "carries the id") {
		return fmt.Errorf("the output contract does not state every element carries the id needed to act on it next")
	}
	if !containsFold(w.agent, "prp_") {
		return fmt.Errorf("the record does not carry the prp_ id so the proposal can be monitored or withdrawn")
	}
	return nil
}

func (w *proposalCirculationWorld) whenReachesTransition() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	for _, leaf := range []string{"proposal propose", "proposal withdraw"} {
		if !mentionsToken(w.combined(), leaf) {
			return fmt.Errorf("the workflow does not reach the %s transition", leaf)
		}
	}
	return nil
}

func (w *proposalCirculationWorld) thenRoutesThroughGuardrail() error {
	if !containsFold(w.combined(), "Write-Safety Guardrail") {
		return fmt.Errorf("the workflow does not route the writes through the Write-Safety Guardrail")
	}
	if !containsFold(w.combined(), "confirmed write flow") {
		return fmt.Errorf("the workflow does not route the transitions through the confirmed write flow")
	}
	if !containsFold(w.agent, "narrate the proposal") {
		return fmt.Errorf("the agent does not surface the proposal before each transition")
	}
	return nil
}

func (w *proposalCirculationWorld) thenNoTransitionWhenDeclined() error {
	if !containsFold(w.combined(), "declined") {
		return fmt.Errorf("the workflow does not treat a declined confirmation as an outcome")
	}
	if !containsFold(w.combined(), "no transition") {
		return fmt.Errorf("the workflow does not state no transition happens when the write is not confirmed")
	}
	return nil
}

func (w *proposalCirculationWorld) thenSurfacesAPIFailure() error {
	if !containsFold(w.agent, "API failure by name") {
		return fmt.Errorf("the agent does not surface a rejected transition's API failure by name")
	}
	return nil
}

func (w *proposalCirculationWorld) thenTransitionsNothing() error {
	if !containsFold(w.agent, "transitions nothing") {
		return fmt.Errorf("the agent does not state a rejected transition transitions nothing")
	}
	return nil
}

func (w *proposalCirculationWorld) thenFabricatesNoState() error {
	if !containsFold(w.agent, "fabricate no state") || !containsFold(w.agent, "record does not contain") {
		return fmt.Errorf("the agent does not state it fabricates no state the record does not contain")
	}
	return nil
}

// --- Rule: reads inform, never gate ----------------------------------------------

func (w *proposalCirculationWorld) whenAdvancesOrWithdraws() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	for _, leaf := range []string{"proposal propose", "proposal withdraw"} {
		if !mentionsToken(w.combined(), leaf) {
			return fmt.Errorf("the workflow does not advance/withdraw through the %s command", leaf)
		}
	}
	return nil
}

func (w *proposalCirculationWorld) thenIssuesAndServerAuthorizes() error {
	if !containsFold(w.combined(), "issue the intended transition") {
		return fmt.Errorf("the artifacts do not state the intended transition is issued regardless of the snapshot")
	}
	if !containsFold(w.combined(), "let the server authorize") {
		return fmt.Errorf("the artifacts do not let the server authorize the transition")
	}
	return nil
}

func (w *proposalCirculationWorld) thenSurfaces422Plainly() error {
	if !containsFold(w.combined(), "422") || !containsFold(w.combined(), "plainly") {
		return fmt.Errorf("the artifacts do not surface a 422 refusal plainly")
	}
	return nil
}

func (w *proposalCirculationWorld) thenNoPreGateOnSnapshot() error {
	if !containsFold(w.combined(), "pre-gate") || !containsFold(w.combined(), "read snapshot") {
		return fmt.Errorf("the artifacts do not forbid pre-gating the call on the read snapshot")
	}
	if !containsFold(w.combined(), "client-side precondition") {
		return fmt.Errorf("the artifacts do not state the snapshot is never a client-side precondition")
	}
	return nil
}

func (w *proposalCirculationWorld) thenReadsToShowProposer() error {
	if !containsFold(w.combined(), "available_transitions") {
		return fmt.Errorf("the artifacts do not read available_transitions to show the proposer the state")
	}
	if !containsFold(w.combined(), "where the proposal stands") {
		return fmt.Errorf("the artifacts do not read to show the proposer where the proposal stands")
	}
	return nil
}

func (w *proposalCirculationWorld) thenNoPreGateClientSide() error {
	if !containsFold(w.combined(), "client-side precondition") || !containsFold(w.combined(), "pre-gate") {
		return fmt.Errorf("the artifacts do not forbid a client-side transition gate")
	}
	if !containsFold(w.combined(), "only transition authority") {
		return fmt.Errorf("the artifacts do not name the server as the only transition authority")
	}
	return nil
}

// --- Rule: reachable / degrades --------------------------------------------------

func (w *proposalCirculationWorld) givenAgentRegistered() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	// Discoverability: the agent carries the frontmatter (name/description/tools)
	// the host needs, and — like the skill — it is auto-discovered from
	// plugin/agents/, so the manifest declares NO `agents` key.
	for _, field := range []string{"name", "description", "tools"} {
		if !hasFrontmatterField(w.agentRaw, field) {
			return fmt.Errorf("proposal-circulator frontmatter lacks the %q field the host needs to register it", field)
		}
	}
	_, raw, err := ReadOrientationManifest()
	if err != nil {
		return fmt.Errorf("plugin manifest did not load: %w", err)
	}
	if !ManifestDemandsNoSetup(raw) {
		return fmt.Errorf("plugin.json declares a setup-forcing key (e.g. `agents`) — the circulator must be auto-discovered from plugin/agents/, not registered via a manifest key")
	}
	return nil
}

func (w *proposalCirculationWorld) whenSkillDelegates() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !containsFold(w.skill, "proposal-circulator") || !containsFold(w.skill, "delegat") {
		return fmt.Errorf("the skill does not delegate circulation to the proposal-circulator agent")
	}
	return nil
}

func (w *proposalCirculationWorld) thenRunsOwnContext() error {
	if !containsFold(w.agent, "isolated context") {
		return fmt.Errorf("the agent does not state it runs the circulation in its own isolated context")
	}
	return nil
}

func (w *proposalCirculationWorld) thenReturnsOnlyRecord() error {
	if !containsFold(w.agent, "circulation record") {
		return fmt.Errorf("the agent output contract never names the circulation record it returns")
	}
	if !containsFold(w.agent, "only the circulation record") && !containsFold(w.agent, "return **only**") {
		return fmt.Errorf("the agent does not state it returns ONLY the circulation record (raw output stays in its context)")
	}
	return nil
}

// givenAgentAbsent models the circulator agent being absent/unregistered: it
// loads ONLY the skill. The degradation scenario asserts the skill's workflow
// stands alone as guidance and no CLI command breaks — neither needs the agent,
// and loading it would both contradict the premise and fail outright if the agent
// file were genuinely gone.
func (w *proposalCirculationWorld) givenAgentAbsent() error { return w.ensureSkillLoaded() }

func (w *proposalCirculationWorld) whenSkillConsulted() error { return w.ensureSkillLoaded() }

func (w *proposalCirculationWorld) thenWorkflowStandsAsGuidance() error {
	if !containsFold(w.skill, "the workflow") {
		return fmt.Errorf("the skill does not carry the workflow steps as consultable guidance")
	}
	for _, step := range []string{"proposal get", "proposal list", "proposal propose", "proposal withdraw"} {
		if !mentionsToken(w.skill, step) {
			return fmt.Errorf("the skill workflow is not self-contained guidance — it omits the %q step", step)
		}
	}
	if !containsFold(w.skill, "follow by hand") && !containsFold(w.skill, "degrades to guidance") {
		return fmt.Errorf("the skill does not state it remains usable as guidance when the agent is absent")
	}
	return nil
}

func (w *proposalCirculationWorld) thenNoCLIBroken() error {
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

// --- Rule: both writes through the guardrail --------------------------------------

// whenInspectAgainstGuardrail exercises the path's two writes against 063's REAL
// PreToolUse gate script and pins the fenced tool grant + gate membership: both
// composed transitions must be gated (asked), the reads must not, and the agent
// tool grant must keep Bash while withholding Write/Edit.
func (w *proposalCirculationWorld) whenInspectAgainstGuardrail() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	w.tools, w.hasTools = AgentTools(w.agentRaw)
	gated, err := ReadGatedRegistry()
	if err != nil {
		return fmt.Errorf("could not read 063's gated-command registry: %w", err)
	}
	w.gated = gated
	composed, err := ReadProposalCirculationCommands()
	if err != nil {
		return fmt.Errorf("could not read the composed-leaf registry: %w", err)
	}
	w.composed = composed
	return nil
}

func (w *proposalCirculationWorld) thenBothGatedRunConfirmed() error {
	// Each transition is asked (gated) at 063's real gate — the confirmed write
	// flow. Concrete bodyless command lines, exactly as the circulator would run
	// them.
	for _, command := range []string{
		"glassfrog proposal propose prp_0123",
		"glassfrog proposal withdraw prp_0123",
	} {
		dec, _, err := runGateScript(command)
		if err != nil {
			return fmt.Errorf("063's gate errored on %q: %v", command, err)
		}
		if dec != "ask" {
			return fmt.Errorf("%q was not gated (decision %q) — both transitions must run through the confirmed write flow", command, dec)
		}
	}
	if !containsFold(w.combined(), "confirmed write flow") {
		return fmt.Errorf("the artifacts do not state the transitions run through the confirmed write flow")
	}
	return nil
}

func (w *proposalCirculationWorld) thenNotUngated() error {
	// Structurally: both gated writes are composed leaves AND members of 063's
	// gated set, and every other composed leaf (the reads) is absent from it.
	// Anchored per write leaf (ProposalCirculationGatedWrites) — deriving the
	// writes from the gated set alone, or counting "exactly two gated composed
	// leaves", would pass if a read were swapped in for a transition (both share
	// the proposal group and a swap preserves the count), the exact regression
	// this check exists to catch.
	gatedSet := setOf(w.gated)
	composedSet := setOf(w.composed)
	writeSet := setOf(ProposalCirculationGatedWrites)
	for _, write := range ProposalCirculationGatedWrites {
		if !composedSet[write] {
			return fmt.Errorf("the composed set no longer names the gated write %q — a guardrail-crossing transition is missing from what the path composes", write)
		}
		if !gatedSet[write] {
			return fmt.Errorf("the gated write %q is not a member of 063's gated set — that transition would ship unconfirmed", write)
		}
	}
	for _, leaf := range w.composed {
		if !writeSet[leaf] && gatedSet[leaf] {
			return fmt.Errorf("composed leaf %q is a read but is gated by 063 — monitoring would start prompting", leaf)
		}
	}
	// A grounding read passes 063's gate ungated (would-be prompting is the defect).
	dec, _, err := runGateScript("glassfrog proposal get prp_0123")
	if err != nil {
		return fmt.Errorf("063's gate errored on the grounding read: %v", err)
	}
	if dec == "ask" {
		return fmt.Errorf("the grounding read proposal get was gated (ask) — monitoring must not start prompting")
	}
	// The tool grant withholds Write/Edit (no workspace mutation) while keeping
	// Bash (to invoke the composed reads and the two gated transitions).
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
				return fmt.Errorf("the agent tool grant includes %q — the circulator must not be able to mutate the workspace", forbidden)
			}
		}
	}
	if !hasBash {
		return fmt.Errorf("the agent tool grant omits Bash — it could not invoke the gated transitions (a misconfiguration, not a safety win)")
	}
	return nil
}

// --- Rule: monitor drawn together -------------------------------------------------

func (w *proposalCirculationWorld) whenReadsBack() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !mentionsToken(w.combined(), "proposal get") {
		return fmt.Errorf("the workflow does not read the proposal back through proposal get")
	}
	return nil
}

func (w *proposalCirculationWorld) thenSurfacesDrawnTogether() error {
	for _, field := range []string{"response_summary", "response_deadline", "available_transitions"} {
		if !containsFold(w.combined(), field) {
			return fmt.Errorf("the monitoring picture omits %s", field)
		}
	}
	if !containsFold(w.agent, "drawn together") {
		return fmt.Errorf("the agent does not surface the monitoring fields drawn together")
	}
	return nil
}

func (w *proposalCirculationWorld) thenNotComputeAcceptance() error {
	if !containsFold(w.combined(), "computes no acceptance") {
		return fmt.Errorf("the artifacts do not state the path computes no acceptance itself")
	}
	return nil
}

func (w *proposalCirculationWorld) thenSurfacesFailure() error {
	if !containsFold(w.agent, "surface what the failure was") {
		return fmt.Errorf("the agent does not surface what the failure was on a partial monitoring walk")
	}
	return nil
}

func (w *proposalCirculationWorld) thenPresentsGathered() error {
	if !containsFold(w.agent, "gathered so far") || !containsFold(w.agent, "incomplete") {
		return fmt.Errorf("the agent does not present what it gathered so far, flagged incomplete")
	}
	return nil
}

func (w *proposalCirculationWorld) thenNotInventNorAbandon() error {
	if !containsFold(w.agent, "invent") || !containsFold(w.agent, "whole result") {
		return fmt.Errorf("the agent does not state it neither invents the missing data nor abandons the whole result")
	}
	return nil
}

// --- Rule: responses belong to the response side ----------------------------------

func (w *proposalCirculationWorld) thenNamesResponseSide() error {
	if !containsFold(w.combined(), "response side") || !containsFold(w.combined(), "069") {
		return fmt.Errorf("the artifacts do not name the response side (069) as where a response is recorded")
	}
	return nil
}

func (w *proposalCirculationWorld) thenNotRecordResponse() error {
	if !containsFold(w.agent, "record no response") && !containsFold(w.agent, "do not record it yourself") {
		return fmt.Errorf("the agent does not state it records no response itself")
	}
	return nil
}

// whenInspectForResponseStep loads 063's gated registry and the composed leaf
// list — the source-derived enumeration of what a proposal write IS
// (create/respond are the gated writes the path must NOT compose).
func (w *proposalCirculationWorld) whenInspectForResponseStep() error {
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
	composed, err := ReadProposalCirculationCommands()
	if err != nil {
		return fmt.Errorf("could not read the composed-leaf registry: %w", err)
	}
	w.composed = composed
	return nil
}

func (w *proposalCirculationWorld) thenContainsNoResponseStep() error {
	// The path RUNS exactly two gated writes — the transitions — and no response
	// or create step. The gated composed leaves must be EXACTLY the two write
	// anchors: any other gated leaf in the composed set is a proposal write the
	// path must not run (respond is 069's, create is 067's), and a missing anchor
	// means a transition stopped being composed.
	gatedSet := setOf(w.gated)
	writeSet := setOf(ProposalCirculationGatedWrites)
	composedSet := setOf(w.composed)
	for _, write := range ProposalCirculationGatedWrites {
		if !composedSet[write] {
			return fmt.Errorf("the composed set no longer names the gated transition %q — the path must compose both transitions", write)
		}
		if !gatedSet[write] {
			return fmt.Errorf("the gated transition %q is not gated by 063 — the path's write would ship unconfirmed", write)
		}
	}
	for _, leaf := range w.composed {
		if gatedSet[leaf] && !writeSet[leaf] {
			return fmt.Errorf("the composed set runs the gated write %q beyond the two transitions — recording a response (or creating a draft) is never this path's act", leaf)
		}
	}
	return nil
}

func (w *proposalCirculationWorld) thenResponseRemainsResponseSide() error {
	if !containsFold(w.combined(), "response side") || !containsFold(w.combined(), "069") {
		return fmt.Errorf("the artifacts do not leave recording a response to the response side (069)")
	}
	return nil
}

// --- Rule: synthesized record ------------------------------------------------------

func (w *proposalCirculationWorld) thenDrawnTogetherRecord() error {
	if !containsFold(w.agent, "drawn-together") {
		return fmt.Errorf("the output contract is not framed as a drawn-together circulation picture")
	}
	for _, el := range []string{"proposal", "situating", "action", "handoff", "notes"} {
		if !mentionsToken(w.agent, el) {
			return fmt.Errorf("the drawn-together record omits the %q element", el)
		}
	}
	if !containsFold(w.agent, "prp_") {
		return fmt.Errorf("the record does not carry the concrete prp_ id per element")
	}
	return nil
}

func (w *proposalCirculationWorld) thenNotConcatenation() error {
	if !containsFold(w.agent, "concatenation") {
		return fmt.Errorf("the agent does not state the record is not a concatenation of unsynthesized dumps")
	}
	return nil
}

// --- Rule: withdraw back to draft ---------------------------------------------------

func (w *proposalCirculationWorld) whenWithdrawsThroughConfirmedFlow() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !mentionsToken(w.combined(), "proposal withdraw") {
		return fmt.Errorf("the workflow does not withdraw through the proposal withdraw command")
	}
	if !containsFold(w.combined(), "confirmed write flow") {
		return fmt.Errorf("the workflow does not run the withdraw through the confirmed write flow")
	}
	if !containsFold(w.agent, "narrate the proposal") {
		return fmt.Errorf("the agent does not surface (narrate) the proposal before the withdraw")
	}
	return nil
}

func (w *proposalCirculationWorld) thenReturnsBackInDraft() error {
	if !containsFold(w.agent, "back in") || !mentionsToken(w.agent, "draft") {
		return fmt.Errorf("the artifacts do not return the withdrawn proposal back in draft")
	}
	return nil
}

func (w *proposalCirculationWorld) thenHandsBackToDrafting() error {
	if !containsFold(w.combined(), "Proposal Drafting Path") || !containsFold(w.combined(), "067") {
		return fmt.Errorf("the workflow does not hand the withdrawn draft back to the Proposal Drafting Path (067)")
	}
	if !containsFold(w.agent, "handoff") || !containsFold(w.agent, "prp_") {
		return fmt.Errorf("the record has no handoff element carrying the prp_ id back to 067")
	}
	if !containsFold(w.combined(), "re-editing") {
		return fmt.Errorf("the artifacts do not hand the prp_ id back for re-editing")
	}
	return nil
}

func (w *proposalCirculationWorld) whenRunsEachTransition() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	for _, leaf := range []string{"proposal propose", "proposal withdraw"} {
		if !mentionsToken(w.combined(), leaf) {
			return fmt.Errorf("the workflow does not run the %s transition", leaf)
		}
	}
	return nil
}

func (w *proposalCirculationWorld) thenEachOwnConfirmedFlow() error {
	if !containsFold(w.agent, "own confirmed write flow") {
		return fmt.Errorf("the agent does not state each transition passes through its own confirmed write flow")
	}
	if !containsFold(w.combined(), "confirms independently") {
		return fmt.Errorf("the artifacts do not state each transition confirms independently")
	}
	return nil
}

func (w *proposalCirculationWorld) thenNeitherBatchNorPreauthorize() error {
	if !containsFold(w.combined(), "batch") || !containsFold(w.combined(), "pre-authorize") {
		return fmt.Errorf("the artifacts do not state the confirmations are never batched or pre-authorized")
	}
	return nil
}

// --- Rule: no authority verdict, no coaching ----------------------------------------

func (w *proposalCirculationWorld) thenNoAuthorityVerdict() error {
	if !containsFold(w.combined(), "within authority") {
		return fmt.Errorf("the artifacts do not disclaim ruling on whether the change is within authority")
	}
	if !containsFold(w.combined(), "do not rule") && !containsFold(w.combined(), "does not rule") {
		return fmt.Errorf("the artifacts do not state they do not rule on the authority question")
	}
	if !containsFold(w.combined(), "Constraint Discovery Path") || !containsFold(w.combined(), "065") {
		return fmt.Errorf("the artifacts do not hand the authority question to the Constraint Discovery Path (065)")
	}
	// No authority verdict may appear: a ruling would read as "you are (not)
	// allowed / permitted" or "permission granted/denied" stated as a verdict.
	for _, verdict := range []string{
		"you are allowed to",
		"you are not allowed to",
		"you are permitted to",
		"permission granted",
		"permission denied",
		"you may proceed with",
	} {
		if containsFold(w.combined(), verdict) {
			return fmt.Errorf("the artifacts carry an authority verdict %q — the path must circulate, not judge", verdict)
		}
	}
	return nil
}

func (w *proposalCirculationWorld) thenNoCoaching() error {
	if !containsFold(w.combined(), "advise on governance craft") || !containsFold(w.combined(), "coach") {
		return fmt.Errorf("the artifacts do not disclaim advising on governance craft or coaching Holacracy practice")
	}
	return nil
}

// --- Drift guard (standalone test is T002) -------------------------------------

func (w *proposalCirculationWorld) givenCirculationContent() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	composed, err := ReadProposalCirculationCommands()
	if err != nil {
		return fmt.Errorf("could not read the composed-leaf registry: %w", err)
	}
	if len(composed) == 0 {
		return fmt.Errorf("the composed-leaf registry is empty — nothing to check against the CLI")
	}
	w.composed = composed
	return nil
}

func (w *proposalCirculationWorld) whenCheckedAgainstCLI() error {
	liveProposal, err := LiveProposalSubcommands()
	if err != nil {
		return fmt.Errorf("could not extract the CLI's proposal subcommand surface: %w", err)
	}
	if len(liveProposal) == 0 {
		return fmt.Errorf("extracted no proposal subcommands — the surface anchor could not be read")
	}
	w.liveProposal = liveProposal
	gated, err := ReadGatedRegistry()
	if err != nil {
		return fmt.Errorf("could not read 063's gated-command registry: %w", err)
	}
	w.gated = gated
	w.drift = CheckProposalCirculationDrift(w.composed, w.liveProposal, w.gated, w.agent)
	return nil
}

func (w *proposalCirculationWorld) thenEachExists() error {
	if len(w.drift) != 0 {
		return fmt.Errorf("a composed leaf no longer resolves in the shipped CLI (or violates the gated-membership invariant):\n  - %s", joinDrift(w.drift))
	}
	liveSet := setOf(w.liveProposal)
	for _, leaf := range w.composed {
		fields := strings.Fields(leaf)
		if len(fields) != 2 || fields[0] != "proposal" || !liveSet[fields[1]] {
			return fmt.Errorf("composed leaf %q does not exist as a subcommand of the CLI's proposal command", leaf)
		}
	}
	return nil
}

func (w *proposalCirculationWorld) thenNamesNoLackingCommand() error {
	// The same drift result proves the path names no command the CLI does not
	// expose.
	if len(w.drift) != 0 {
		return fmt.Errorf("the path names a command the CLI does not expose:\n  - %s", joinDrift(w.drift))
	}
	return nil
}
