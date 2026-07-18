package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestProposalDraftingFeatures runs the executable acceptance for the Proposal
// Drafting Path (067). Like the sibling build-side suites (062–066) its Paths name
// ONLY this spec's feature file and it runs with the ~@wip filter, so only the
// scenarios implemented so far execute.
//
// The deliverable is a pair of declarative Claude-plugin artifacts — a thin
// `proposal-drafting` skill that delegates to a write-capable-but-fenced
// `proposal-drafter` agent — plus a single-sourced composed-leaf registry and a
// best-effort drift guard. The artifacts carry no runtime Go path of their own, so
// the executable scenarios assert against their content: the skill's when +
// workflow + delegation + gated-write note, the agent's fenced grant +
// isolated-execution + draft-record output contract + defensive drafting, the
// gated-side-of-the-guardrail boundary (exercised against 063's real gate script),
// and the drift guard that keeps the composed leaves truthful to the shipped CLI
// and pins the create's membership in 063's gated set (the guard's standalone test
// is T002).
func TestProposalDraftingFeatures(t *testing.T) {
	w := &proposalDraftingWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.register(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unequipped-agent-operators/proposal-drafting-path.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: proposal-drafting-path feature scenarios failed")
	}
}

// proposalDraftingWorld is the per-scenario state: the loaded skill + agent
// content, the single-sourced composed leaves, 063's gated set, the CLI's live
// tension/proposal subcommand surfaces, the parsed agent tool grant, and the drift
// findings.
type proposalDraftingWorld struct {
	// skill/agent hold the artifacts with whitespace collapsed to single spaces, so
	// phrase assertions are resilient to markdown line-wrapping (a hard-wrapped
	// "before judging\nduplicates" still matches "before judging duplicates").
	skill string
	agent string
	// skillRaw/agentRaw keep the verbatim content for structural checks that depend
	// on line breaks (frontmatter delimiters, the tools list).
	skillRaw     string
	agentRaw     string
	composed     []string
	gated        []string
	liveTension  []string
	liveProposal []string
	tools        []string
	hasTools     bool
	drift        []string
}

func (w *proposalDraftingWorld) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = proposalDraftingWorld{}
		return ctx, nil
	})

	// Rule: Turn a ready tension into a submittable draft --------------------
	sc.Step(`^a well-formed anchor tension's ten_ id and an assembled set of governance changes$`, w.givenLoaded)
	sc.Step(`^the proposal-drafter surfaces them for confirmation and runs the create through the confirmed write flow$`, w.whenCreatesThroughConfirmedFlow)
	sc.Step(`^it will return the created proposal including its prp_ id and draft status$`, w.thenReturnsCreatedWithId)
	sc.Step(`^the record will carry that id so the draft can be advanced or withdrawn$`, w.thenIdCarriesForward)

	sc.Step(`^a set of governance changes held in a JSON file$`, w.givenLoaded)
	sc.Step(`^the proposal-drafter assembles the change set from that source$`, w.whenAssembles)
	sc.Step(`^it will pass the array through verbatim above the type floor as the proposal's changes$`, w.thenPassesVerbatim)
	sc.Step(`^it will not interpret or validate any change's command-specific keys$`, w.thenNoInterpret)

	sc.Step(`^a create attempt whose anchor tension was unknown to the record$`, w.givenLoaded)
	sc.Step(`^the proposal-drafter runs the create through the confirmed write flow$`, w.whenCreatesThroughConfirmedFlow)
	sc.Step(`^it will surface the API failure by name$`, w.thenSurfacesAPIFailure)
	sc.Step(`^it will create nothing$`, w.thenCreatesNothing)
	sc.Step(`^it will fabricate no prp_ id the record does not contain$`, w.thenFabricatesNoId)

	sc.Step(`^an assembled anchor and change set ready to create$`, w.givenLoaded)
	sc.Step(`^the proposal-drafter reaches the proposal create$`, w.whenReachesCreate)
	sc.Step(`^it will route the write through the Write-Safety Guardrail's confirmed flow, surfacing the change set first$`, w.thenRoutesThroughGuardrail)
	sc.Step(`^no proposal will be created when the write is not confirmed$`, w.thenNoProposalWhenDeclined)

	sc.Step(`^the plugin was present with the proposal-drafter agent registered$`, w.givenAgentRegistered)
	sc.Step(`^the proposal-drafting skill delegates a ready tension for drafting$`, w.whenSkillDelegates)
	sc.Step(`^the drafter will run the workflow in its own context$`, w.thenRunsOwnContext)
	sc.Step(`^it will return only the draft record to the caller$`, w.thenReturnsOnlyRecord)

	sc.Step(`^the plugin was present but the proposal-drafter agent was absent or unregistered$`, w.givenAgentAbsent)
	sc.Step(`^the proposal-drafting skill is consulted for a ready tension$`, w.whenSkillConsulted)
	sc.Step(`^its workflow will remain readable as guidance the caller can follow by hand$`, w.thenWorkflowStandsAsGuidance)
	sc.Step(`^no command in the CLI will be broken by the agent's absence$`, w.thenNoCLIBroken)

	sc.Step(`^the record the proposal-drafter returned$`, w.givenLoaded)
	sc.Step(`^it is compared against the raw command output$`, w.whenLoaded)
	sc.Step(`^it will be a drawn-together draft carrying its prp_ id$`, w.thenDrawnTogetherRecord)
	sc.Step(`^it will not be a concatenation of unsynthesized dumps$`, w.thenNotConcatenation)

	// Rule: Show what's already in flight before opening a duplicate draft ----
	sc.Step(`^a circle whose in-flight proposals span several pages$`, w.givenLoaded)
	sc.Step(`^the proposal-drafter surfaces the proposals already in flight there$`, w.whenSituates)
	sc.Step(`^it will page through the full result set before judging duplicates$`, w.thenPagesBeforeJudging)
	sc.Step(`^it will present them drawn together so the practitioner can see what is already circulating$`, w.thenPresentsDrawnTogether)
	sc.Step(`^it will treat the new draft as a deliberate addition rather than a blind duplicate$`, w.thenDeliberateAddition)

	sc.Step(`^a change that matched a draft already circulating in the circle$`, w.givenLoaded)
	sc.Step(`^the proposal-drafter situates before creating$`, w.whenLoaded)
	sc.Step(`^it will surface the existing proposal with its prp_ id$`, w.thenSurfacesExisting)
	sc.Step(`^it will let the practitioner decide$`, w.thenLetsDecide)
	sc.Step(`^it will not silently create a duplicate draft$`, w.thenNoSilentDuplicate)

	sc.Step(`^a situating step in which the proposal list read failed mid-walk$`, w.givenLoaded)
	sc.Step(`^the proposal-drafter continues$`, w.whenLoaded)
	sc.Step(`^it will surface what the failure was$`, w.thenSurfacesFailure)
	sc.Step(`^it will present the proposals the read gathered so far, flagged incomplete$`, w.thenPresentsGathered)
	sc.Step(`^it will not invent the missing proposals or abandon the whole result$`, w.thenNotInventNorAbandon)

	// Rule: Move a created draft toward circulation carrying its id -----------
	sc.Step(`^a created draft proposal the practitioner wanted to advance$`, w.givenLoaded)
	sc.Step(`^the proposal-drafter completes its drafting$`, w.whenLoaded)
	sc.Step(`^it will hand the prp_ id to the Proposal Circulation Path$`, w.thenHandsOffId)
	sc.Step(`^it will not advance, withdraw, or circulate the proposal itself$`, w.thenNeverCirculates)

	sc.Step(`^the path's treatment of the proposal create$`, w.givenLoaded)
	sc.Step(`^it is inspected against the Write-Safety Guardrail$`, w.whenInspectAgainstGuardrail)
	sc.Step(`^the create will run through the confirmed write flow$`, w.thenCreateGatedRunsConfirmed)
	sc.Step(`^the path will not issue the gated proposal write as if it were ungated$`, w.thenNotUngated)

	// Shared Given for the three "skill and agent content" validation scenarios.
	sc.Step(`^the proposal-drafting skill and agent content$`, w.givenLoaded)

	sc.Step(`^it is inspected for any advance, withdraw, response, or circulate step$`, w.whenInspectForCirculationStep)
	sc.Step(`^it will contain none$`, w.thenContainsNoCirculationStep)
	sc.Step(`^it will hand the prp_ id to the Proposal Circulation Path$`, w.thenHandsProposalWorkToCirculation)

	sc.Step(`^it is inspected for per-change interpretation$`, w.whenLoaded)
	sc.Step(`^it will assemble and pass the array through verbatim above a type floor$`, w.thenAssemblesVerbatim)
	sc.Step(`^it will validate no change's type value or command-specific keys$`, w.thenValidatesNoKeys)
	sc.Step(`^it will build no typed constructor$`, w.thenNoTypedConstructor)

	sc.Step(`^it is inspected for an authority verdict or Holacracy coaching$`, w.whenLoaded)
	sc.Step(`^it will neither rule on whether the tension needs a proposal$`, w.thenNoAuthorityVerdict)
	sc.Step(`^it will not advise on governance craft$`, w.thenNoCoaching)

	// Drift guard (standalone test is T002) ----------------------------------
	sc.Step(`^the produced proposal-drafting-path content$`, w.givenDraftingContent)
	sc.Step(`^every command it composes is checked against the shipped CLI$`, w.whenCheckedAgainstCLI)
	sc.Step(`^each one will exist$`, w.thenEachExists)
	sc.Step(`^the path will name no command the CLI does not expose$`, w.thenNamesNoLackingCommand)
}

// --- Loading -----------------------------------------------------------------

func (w *proposalDraftingWorld) ensureSkillLoaded() error {
	if w.skillRaw == "" {
		skill, err := ReadProposalDraftingSkill()
		if err != nil {
			return fmt.Errorf("proposal-drafting skill did not load: %w", err)
		}
		w.skillRaw, w.skill = skill, normalizeWS(skill)
	}
	return nil
}

func (w *proposalDraftingWorld) ensureAgentLoaded() error {
	if w.agentRaw == "" {
		agent, err := ReadProposalDrafterAgent()
		if err != nil {
			return fmt.Errorf("proposal-drafter agent did not load: %w", err)
		}
		w.agentRaw, w.agent = agent, normalizeWS(agent)
	}
	return nil
}

// ensureLoaded loads both artifacts — used by every scenario except the
// missing-drafter degradation path, which loads only the skill (see
// givenAgentAbsent) so it faithfully models the agent being absent.
func (w *proposalDraftingWorld) ensureLoaded() error {
	if err := w.ensureSkillLoaded(); err != nil {
		return err
	}
	return w.ensureAgentLoaded()
}

func (w *proposalDraftingWorld) givenLoaded() error { return w.ensureLoaded() }
func (w *proposalDraftingWorld) whenLoaded() error  { return w.ensureLoaded() }

// combined returns the (normalized) skill and agent content together — most
// behavioral scenarios inspect "the drafter's behavior" as described across both
// artifacts.
func (w *proposalDraftingWorld) combined() string { return w.skill + " " + w.agent }

// --- Rule: ready tension → created draft ---------------------------------------

func (w *proposalDraftingWorld) whenCreatesThroughConfirmedFlow() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !mentionsToken(w.combined(), "proposal create") {
		return fmt.Errorf("the workflow does not create through the proposal create command")
	}
	if !containsFold(w.combined(), "confirmed write flow") {
		return fmt.Errorf("the workflow does not run the create through the confirmed write flow")
	}
	return nil
}

func (w *proposalDraftingWorld) thenReturnsCreatedWithId() error {
	if !mentionsToken(w.combined(), "proposal create") {
		return fmt.Errorf("the workflow does not create through the proposal create command")
	}
	if !containsFold(w.agent, "prp_") {
		return fmt.Errorf("the output contract does not return the created proposal's prp_ id")
	}
	if !containsFold(w.agent, "draft") {
		return fmt.Errorf("the output contract does not state the created proposal's draft status")
	}
	return nil
}

func (w *proposalDraftingWorld) thenIdCarriesForward() error {
	if !containsFold(w.agent, "carries the id") {
		return fmt.Errorf("the output contract does not state every element carries the id needed to feed it onward")
	}
	if !containsFold(w.agent, "prp_") {
		return fmt.Errorf("the record does not carry the prp_ id so the draft can be advanced or withdrawn")
	}
	return nil
}

func (w *proposalDraftingWorld) whenAssembles() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !containsFold(w.combined(), "assemble") || !containsFold(w.combined(), "change set") {
		return fmt.Errorf("the workflow does not assemble the change set")
	}
	return nil
}

func (w *proposalDraftingWorld) thenPassesVerbatim() error {
	if !containsFold(w.combined(), "verbatim") || !containsFold(w.combined(), "type floor") {
		return fmt.Errorf("the workflow does not pass the array through verbatim above the type floor")
	}
	return nil
}

func (w *proposalDraftingWorld) thenNoInterpret() error {
	if !containsFold(w.combined(), "command-specific keys") {
		return fmt.Errorf("the workflow does not state it leaves change command-specific keys uninterpreted")
	}
	if !containsFold(w.combined(), "does not interpret") && !containsFold(w.combined(), "not interpret or validate") {
		return fmt.Errorf("the workflow does not state it neither interprets nor validates a change's keys")
	}
	return nil
}

func (w *proposalDraftingWorld) whenReachesCreate() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !mentionsToken(w.combined(), "proposal create") {
		return fmt.Errorf("the workflow does not reach the proposal create command")
	}
	return nil
}

func (w *proposalDraftingWorld) thenRoutesThroughGuardrail() error {
	if !containsFold(w.combined(), "Write-Safety Guardrail") {
		return fmt.Errorf("the workflow does not route the write through the Write-Safety Guardrail")
	}
	if !containsFold(w.combined(), "confirmed write flow") {
		return fmt.Errorf("the workflow does not route the create through the confirmed write flow")
	}
	if !containsFold(w.combined(), "inline") {
		return fmt.Errorf("the workflow does not surface the change set inline so the confirmation shows the exact payload")
	}
	return nil
}

func (w *proposalDraftingWorld) thenNoProposalWhenDeclined() error {
	if !containsFold(w.combined(), "declined") {
		return fmt.Errorf("the workflow does not treat a declined confirmation as an outcome")
	}
	if !containsFold(w.combined(), "no proposal is created") {
		return fmt.Errorf("the workflow does not state no proposal is created when the write is not confirmed")
	}
	return nil
}

func (w *proposalDraftingWorld) thenSurfacesAPIFailure() error {
	if !containsFold(w.agent, "API failure by name") {
		return fmt.Errorf("the agent does not surface a rejected create's API failure by name")
	}
	return nil
}

func (w *proposalDraftingWorld) thenCreatesNothing() error {
	if !containsFold(w.agent, "creates nothing") {
		return fmt.Errorf("the agent does not state a rejected create creates nothing")
	}
	return nil
}

func (w *proposalDraftingWorld) thenFabricatesNoId() error {
	if !containsFold(w.agent, "fabricate no") || !containsFold(w.agent, "prp_") {
		return fmt.Errorf("the agent does not state it fabricates no prp_ id on a rejected create")
	}
	return nil
}

// --- Rule: reachable / degrades -----------------------------------------------

func (w *proposalDraftingWorld) givenAgentRegistered() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	// Discoverability: the agent carries the frontmatter (name/description/tools)
	// the host needs, and — like the skill — it is auto-discovered from
	// plugin/agents/, so the manifest declares NO `agents` key.
	for _, field := range []string{"name", "description", "tools"} {
		if !hasFrontmatterField(w.agentRaw, field) {
			return fmt.Errorf("proposal-drafter frontmatter lacks the %q field the host needs to register it", field)
		}
	}
	_, raw, err := ReadOrientationManifest()
	if err != nil {
		return fmt.Errorf("plugin manifest did not load: %w", err)
	}
	if !ManifestDemandsNoSetup(raw) {
		return fmt.Errorf("plugin.json declares a setup-forcing key (e.g. `agents`) — the drafter must be auto-discovered from plugin/agents/, not registered via a manifest key")
	}
	return nil
}

func (w *proposalDraftingWorld) whenSkillDelegates() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !containsFold(w.skill, "proposal-drafter") || !containsFold(w.skill, "delegat") {
		return fmt.Errorf("the skill does not delegate drafting to the proposal-drafter agent")
	}
	return nil
}

func (w *proposalDraftingWorld) thenRunsOwnContext() error {
	if !containsFold(w.agent, "isolated context") {
		return fmt.Errorf("the agent does not state it runs the drafting in its own isolated context")
	}
	return nil
}

func (w *proposalDraftingWorld) thenReturnsOnlyRecord() error {
	if !containsFold(w.agent, "draft record") {
		return fmt.Errorf("the agent output contract never names the draft record it returns")
	}
	if !containsFold(w.agent, "only the draft record") && !containsFold(w.agent, "return **only**") {
		return fmt.Errorf("the agent does not state it returns ONLY the draft record (raw output stays in its context)")
	}
	return nil
}

// givenAgentAbsent models the drafter agent being absent/unregistered: it loads
// ONLY the skill. The degradation scenario asserts the skill's workflow stands
// alone as guidance and no CLI command breaks — neither needs the agent, and
// loading it would both contradict the premise and fail outright if the agent file
// were genuinely gone.
func (w *proposalDraftingWorld) givenAgentAbsent() error { return w.ensureSkillLoaded() }

func (w *proposalDraftingWorld) whenSkillConsulted() error { return w.ensureSkillLoaded() }

func (w *proposalDraftingWorld) thenWorkflowStandsAsGuidance() error {
	if !containsFold(w.skill, "the workflow") {
		return fmt.Errorf("the skill does not carry the workflow steps as consultable guidance")
	}
	for _, step := range []string{"tension get", "proposal list", "proposal get", "proposal create"} {
		if !mentionsToken(w.skill, step) {
			return fmt.Errorf("the skill workflow is not self-contained guidance — it omits the %q step", step)
		}
	}
	if !containsFold(w.skill, "follow by hand") && !containsFold(w.skill, "degrades to guidance") {
		return fmt.Errorf("the skill does not state it remains usable as guidance when the agent is absent")
	}
	return nil
}

func (w *proposalDraftingWorld) thenNoCLIBroken() error {
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

func (w *proposalDraftingWorld) thenDrawnTogetherRecord() error {
	if !containsFold(w.agent, "drawn-together") {
		return fmt.Errorf("the output contract is not framed as a drawn-together draft record")
	}
	for _, el := range []string{"draft", "anchor", "situating", "action", "handoff", "notes"} {
		if !mentionsToken(w.agent, el) {
			return fmt.Errorf("the drawn-together record omits the %q element", el)
		}
	}
	if !containsFold(w.agent, "prp_") || !containsFold(w.agent, "ten_") {
		return fmt.Errorf("the record does not carry the concrete prp_/ten_ ids per element")
	}
	return nil
}

func (w *proposalDraftingWorld) thenNotConcatenation() error {
	if !containsFold(w.agent, "concatenation") {
		return fmt.Errorf("the agent does not state the record is not a concatenation of unsynthesized dumps")
	}
	return nil
}

// --- Rule: situate before creating ---------------------------------------------

func (w *proposalDraftingWorld) whenSituates() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !mentionsToken(w.combined(), "proposal list") {
		return fmt.Errorf("the workflow does not situate through proposal list")
	}
	return nil
}

func (w *proposalDraftingWorld) thenPagesBeforeJudging() error {
	if !containsFold(w.combined(), "full result set") || !containsFold(w.combined(), "before judging duplicates") {
		return fmt.Errorf("the workflow does not page through the full result set before judging duplicates")
	}
	return nil
}

func (w *proposalDraftingWorld) thenPresentsDrawnTogether() error {
	if !containsFold(w.agent, "drawn together") || !containsFold(w.agent, "already circulating") {
		return fmt.Errorf("the agent does not present the situating proposals drawn together so the practitioner sees what is already circulating")
	}
	return nil
}

func (w *proposalDraftingWorld) thenDeliberateAddition() error {
	if !containsFold(w.combined(), "deliberate addition") {
		return fmt.Errorf("the workflow does not treat the new draft as a deliberate addition")
	}
	return nil
}

func (w *proposalDraftingWorld) thenSurfacesExisting() error {
	if !containsFold(w.combined(), "existing proposal") || !containsFold(w.combined(), "with its") {
		return fmt.Errorf("the workflow does not surface the existing proposal with its id on a duplicate")
	}
	if !containsFold(w.combined(), "prp_") {
		return fmt.Errorf("the surfaced existing proposal does not carry its prp_ id")
	}
	return nil
}

func (w *proposalDraftingWorld) thenLetsDecide() error {
	if !containsFold(w.combined(), "let the practitioner decide") {
		return fmt.Errorf("the workflow does not let the practitioner decide on a matching draft")
	}
	return nil
}

func (w *proposalDraftingWorld) thenNoSilentDuplicate() error {
	if !containsFold(w.combined(), "silently create a duplicate") {
		return fmt.Errorf("the workflow does not state it never silently creates a duplicate draft")
	}
	return nil
}

func (w *proposalDraftingWorld) thenSurfacesFailure() error {
	if !containsFold(w.agent, "surface what the failure was") {
		return fmt.Errorf("the agent does not surface what the failure was on a partial situating walk")
	}
	return nil
}

func (w *proposalDraftingWorld) thenPresentsGathered() error {
	if !containsFold(w.agent, "gathered so far") || !containsFold(w.agent, "incomplete") {
		return fmt.Errorf("the agent does not present the proposals gathered so far, flagged incomplete")
	}
	return nil
}

func (w *proposalDraftingWorld) thenNotInventNorAbandon() error {
	if !containsFold(w.agent, "invent") || !containsFold(w.agent, "whole result") {
		return fmt.Errorf("the agent does not state it neither invents the missing proposals nor abandons the whole result")
	}
	return nil
}

// --- Rule: hand off, gated write, no judging -----------------------------------

func (w *proposalDraftingWorld) thenHandsOffId() error {
	if !containsFold(w.combined(), "Proposal Circulation Path") || !containsFold(w.combined(), "068") {
		return fmt.Errorf("the workflow does not hand the created draft to the Proposal Circulation Path (068)")
	}
	if !containsFold(w.agent, "handoff") || !containsFold(w.agent, "prp_") {
		return fmt.Errorf("the record has no handoff element carrying the prp_ id for 068")
	}
	return nil
}

func (w *proposalDraftingWorld) thenNeverCirculates() error {
	for _, verb := range []string{"advance", "withdraw", "circulate"} {
		if !containsFold(w.agent, verb) {
			return fmt.Errorf("the agent does not state it never %ss the proposal itself", verb)
		}
	}
	return nil
}

// whenInspectAgainstGuardrail exercises the path's one write against 063's REAL
// PreToolUse gate script and pins the fenced tool grant + gate membership: the
// composed create must be gated (asked), the situating reads must not, and the
// agent tool grant must keep Bash while withholding Write/Edit.
func (w *proposalDraftingWorld) whenInspectAgainstGuardrail() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	w.tools, w.hasTools = AgentTools(w.agentRaw)
	gated, err := ReadGatedRegistry()
	if err != nil {
		return fmt.Errorf("could not read 063's gated-command registry: %w", err)
	}
	w.gated = gated
	composed, err := ReadProposalDraftingCommands()
	if err != nil {
		return fmt.Errorf("could not read the composed-leaf registry: %w", err)
	}
	w.composed = composed
	return nil
}

func (w *proposalDraftingWorld) thenCreateGatedRunsConfirmed() error {
	// The create is asked (gated) at 063's real gate — the confirmed write flow.
	dec, _, err := runGateScript(`glassfrog proposal create ten_1 --changes '[{"type":"add_role"}]'`)
	if err != nil {
		return fmt.Errorf("063's gate errored on the proposal create: %v", err)
	}
	if dec != "ask" {
		return fmt.Errorf("proposal create was not gated (decision %q) — the path's one write must run through the confirmed write flow", dec)
	}
	if !containsFold(w.combined(), "confirmed write flow") {
		return fmt.Errorf("the artifacts do not state the create runs through the confirmed write flow")
	}
	return nil
}

func (w *proposalDraftingWorld) thenNotUngated() error {
	// Structurally: the one gated write (the create) is a composed leaf AND a member
	// of 063's gated set, and every other composed leaf (the situating reads) is
	// absent from it. Anchored on the write leaf — deriving it as "the single gated
	// composed leaf" would pass if a read were swapped in for the create (both share
	// the proposal group), the exact regression this check exists to catch.
	gatedSet := setOf(w.gated)
	composedSet := setOf(w.composed)
	if !composedSet[ProposalDraftingGatedWrite] {
		return fmt.Errorf("the composed set no longer names the one gated write %q — the guardrail-crossing write is missing from what the path composes", ProposalDraftingGatedWrite)
	}
	if !gatedSet[ProposalDraftingGatedWrite] {
		return fmt.Errorf("the one gated write %q is not a member of 063's gated set — the create would ship ungated", ProposalDraftingGatedWrite)
	}
	for _, leaf := range w.composed {
		if leaf != ProposalDraftingGatedWrite && gatedSet[leaf] {
			return fmt.Errorf("composed leaf %q is a situating read but is gated by 063 — situating would start prompting", leaf)
		}
	}
	// A situating read passes 063's gate ungated (would-be prompting is the defect).
	dec, _, err := runGateScript(`glassfrog proposal list --role-id circ_1 --status draft`)
	if err != nil {
		return fmt.Errorf("063's gate errored on the situating read: %v", err)
	}
	if dec == "ask" {
		return fmt.Errorf("the situating read proposal list was gated (ask) — situating must not start prompting")
	}
	// The tool grant withholds Write/Edit (no workspace mutation, blocking change-set
	// temp files) while keeping Bash (to invoke the one gated create).
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
				return fmt.Errorf("the agent tool grant includes %q — the drafter must not be able to mutate the workspace", forbidden)
			}
		}
	}
	if !hasBash {
		return fmt.Errorf("the agent tool grant omits Bash — it could not invoke the one gated create (a misconfiguration, not a safety win)")
	}
	return nil
}

// whenInspectForCirculationStep loads 063's gated registry and the composed leaf
// list — the source-derived enumeration of what a circulation write IS
// (propose/respond/withdraw are the gated writes the path does NOT compose).
func (w *proposalDraftingWorld) whenInspectForCirculationStep() error {
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
	composed, err := ReadProposalDraftingCommands()
	if err != nil {
		return fmt.Errorf("could not read the composed-leaf registry: %w", err)
	}
	w.composed = composed
	return nil
}

func (w *proposalDraftingWorld) thenContainsNoCirculationStep() error {
	// The path RUNS exactly one gated write — the create — and no circulation write
	// (propose/respond/withdraw). The gated composed leaves must be EXACTLY the
	// create anchor: any other gated leaf in the composed set is a circulation write
	// the path would run, and the create missing means it stops short of its write.
	gatedSet := setOf(w.gated)
	if !gatedSet[ProposalDraftingGatedWrite] {
		return fmt.Errorf("the one gated write %q is not gated by 063 — the path's create would ship ungated", ProposalDraftingGatedWrite)
	}
	for _, leaf := range w.composed {
		if gatedSet[leaf] && leaf != ProposalDraftingGatedWrite {
			return fmt.Errorf("the composed set runs the gated write %q beyond the create — the path must stop at the created draft, never a circulation write", leaf)
		}
	}
	if !setOf(w.composed)[ProposalDraftingGatedWrite] {
		return fmt.Errorf("the composed set no longer names the create %q — the path must create the draft it stops at", ProposalDraftingGatedWrite)
	}
	return nil
}

func (w *proposalDraftingWorld) thenHandsProposalWorkToCirculation() error {
	if !containsFold(w.combined(), "Proposal Circulation Path") {
		return fmt.Errorf("the artifacts do not hand the proposal work onward to the Proposal Circulation Path")
	}
	return nil
}

// --- Rule: assembly not typed construction -------------------------------------

func (w *proposalDraftingWorld) thenAssemblesVerbatim() error {
	if !containsFold(w.combined(), "verbatim") || !containsFold(w.combined(), "type floor") {
		return fmt.Errorf("the artifacts do not assemble and pass the array through verbatim above a type floor")
	}
	return nil
}

func (w *proposalDraftingWorld) thenValidatesNoKeys() error {
	if !containsFold(w.combined(), "value or command-specific keys") {
		return fmt.Errorf("the artifacts do not state they validate no change's type value or command-specific keys")
	}
	return nil
}

func (w *proposalDraftingWorld) thenNoTypedConstructor() error {
	if !containsFold(w.combined(), "typed constructor") {
		return fmt.Errorf("the artifacts do not state they build no typed constructor")
	}
	return nil
}

// --- Rule: no authority verdict, no coaching -----------------------------------

func (w *proposalDraftingWorld) thenNoAuthorityVerdict() error {
	if !containsFold(w.combined(), "needs a proposal") {
		return fmt.Errorf("the artifacts do not disclaim ruling on whether the tension needs a proposal")
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
			return fmt.Errorf("the artifacts carry an authority verdict %q — the path must draft, not judge", verdict)
		}
	}
	return nil
}

func (w *proposalDraftingWorld) thenNoCoaching() error {
	if !containsFold(w.combined(), "advise on governance craft") || !containsFold(w.combined(), "coach") {
		return fmt.Errorf("the artifacts do not disclaim advising on governance craft or coaching Holacracy practice")
	}
	return nil
}

// --- Drift guard (standalone test is T002) -------------------------------------

func (w *proposalDraftingWorld) givenDraftingContent() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	composed, err := ReadProposalDraftingCommands()
	if err != nil {
		return fmt.Errorf("could not read the composed-leaf registry: %w", err)
	}
	if len(composed) == 0 {
		return fmt.Errorf("the composed-leaf registry is empty — nothing to check against the CLI")
	}
	w.composed = composed
	return nil
}

func (w *proposalDraftingWorld) whenCheckedAgainstCLI() error {
	liveTension, err := LiveTensionSubcommands()
	if err != nil {
		return fmt.Errorf("could not extract the CLI's tension subcommand surface: %w", err)
	}
	liveProposal, err := LiveProposalSubcommands()
	if err != nil {
		return fmt.Errorf("could not extract the CLI's proposal subcommand surface: %w", err)
	}
	if len(liveTension) == 0 || len(liveProposal) == 0 {
		return fmt.Errorf("extracted no subcommands — a surface anchor could not be read")
	}
	w.liveTension, w.liveProposal = liveTension, liveProposal
	gated, err := ReadGatedRegistry()
	if err != nil {
		return fmt.Errorf("could not read 063's gated-command registry: %w", err)
	}
	w.gated = gated
	w.drift = CheckProposalDraftingDrift(w.composed, w.liveTension, w.liveProposal, w.gated, w.agent)
	return nil
}

func (w *proposalDraftingWorld) thenEachExists() error {
	if len(w.drift) != 0 {
		return fmt.Errorf("a composed leaf no longer resolves in the shipped CLI (or violates the gated-membership invariant):\n  - %s", joinDrift(w.drift))
	}
	liveByGroup := map[string]map[string]bool{
		"tension":  setOf(w.liveTension),
		"proposal": setOf(w.liveProposal),
	}
	for _, leaf := range w.composed {
		fields := strings.Fields(leaf)
		if len(fields) != 2 || !liveByGroup[fields[0]][fields[1]] {
			return fmt.Errorf("composed leaf %q does not exist as a subcommand of the CLI's %s command", leaf, fields[0])
		}
	}
	return nil
}

func (w *proposalDraftingWorld) thenNamesNoLackingCommand() error {
	// The same drift result proves the path names no command the CLI does not
	// expose.
	if len(w.drift) != 0 {
		return fmt.Errorf("the path names a command the CLI does not expose:\n  - %s", joinDrift(w.drift))
	}
	return nil
}
