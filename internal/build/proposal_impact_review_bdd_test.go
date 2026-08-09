package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestProposalImpactReviewFeatures runs the executable acceptance for the
// Proposal Impact Review Path (069). Like the sibling build-side suites
// (062–068) its Paths name ONLY this spec's feature file and it runs with the
// ~@wip filter, so only the scenarios implemented so far execute.
//
// The deliverable is a pair of declarative Claude-plugin artifacts — a thin
// `proposal-impact-review` skill that delegates the review to a pure-read
// `proposal-impact-reviewer` agent — plus a single-sourced composed-leaf
// registry and a best-effort drift guard. The artifacts carry no runtime Go
// path of their own, so the executable scenarios assert against their content:
// the skill's when + workflow + delegation + decision-and-respond note, the
// agent's read-posture grant + isolated-execution + impact-picture output
// contract + footprint honesty + defensive reviewing, the split write locus
// (review in the agent; the one gated respond in the caller's context — the
// respond exercised against 063's real gate script), the
// reviews-inform-never-decide discipline, and the drift guard that keeps the
// composed leaves truthful to the shipped CLI and pins the respond's
// membership in 063's gated set (the guard's standalone test is T002).
func TestProposalImpactReviewFeatures(t *testing.T) {
	w := &proposalImpactReviewWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.register(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unequipped-agent-operators/proposal-impact-review-path.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: proposal-impact-review-path feature scenarios failed")
	}
}

// proposalImpactReviewWorld is the per-scenario state: the loaded skill + agent
// content, the single-sourced composed leaves, 063's gated set, the parsed
// agent tool grant, and the drift findings.
type proposalImpactReviewWorld struct {
	// skill/agent hold the artifacts with whitespace collapsed to single spaces,
	// so phrase assertions are resilient to markdown line-wrapping (a hard-wrapped
	// "confirmed write\nflow" still matches "confirmed write flow").
	skill string
	agent string
	// skillRaw/agentRaw keep the verbatim content for structural checks that
	// depend on line breaks (frontmatter delimiters, the tools list).
	skillRaw     string
	agentRaw     string
	composed     []string
	gated        []string
	liveTop      []string
	liveMe       []string
	liveProposal []string
	tools        []string
	hasTools     bool
	drift        []string
}

func (w *proposalImpactReviewWorld) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = proposalImpactReviewWorld{}
		return ctx, nil
	})

	// Rule: See what a circulating proposal would change for my own work ------
	sc.Step(`^the prp_ id of a proposal circulating for consent$`, w.givenLoaded)
	sc.Step(`^the proposal-impact-reviewer reads the change set back and draws it against the operator's me roles, actions, and projects$`, w.whenReadsAndDraws)
	sc.Step(`^it will present a drawn-together impact picture showing which of the operator's roles the change touches and how$`, w.thenPresentsDrawnTogetherPicture)
	sc.Step(`^it will record no response$`, w.thenRecordsNoResponse)

	sc.Step(`^a review in which an affected-role read failed$`, w.givenLoaded)
	sc.Step(`^the proposal-impact-reviewer continues$`, w.whenLoaded)
	sc.Step(`^it will surface what the failure was$`, w.thenSurfacesFailure)
	sc.Step(`^it will present what it gathered so far, flagged incomplete$`, w.thenPresentsGathered)
	sc.Step(`^it will not invent the missing data or abandon the whole review$`, w.thenNotInventNorAbandonReview)

	sc.Step(`^the plugin was present with the proposal-impact-reviewer agent registered$`, w.givenAgentRegistered)
	sc.Step(`^the proposal-impact-review skill delegates a review$`, w.whenSkillDelegates)
	sc.Step(`^the reviewer will run the read traversal in its own context$`, w.thenRunsOwnContext)
	sc.Step(`^it will return only the impact picture to the caller$`, w.thenReturnsOnlyPicture)

	sc.Step(`^the plugin was present but the proposal-impact-reviewer agent was absent or unregistered$`, w.givenAgentAbsent)
	sc.Step(`^the proposal-impact-review skill is consulted for a review$`, w.whenSkillConsulted)
	sc.Step(`^its workflow will remain readable as guidance the caller can follow by hand$`, w.thenWorkflowStandsAsGuidance)
	sc.Step(`^no command in the CLI will be broken by the agent's absence$`, w.thenNoCLIBroken)

	sc.Step(`^the picture the proposal-impact-reviewer returned$`, w.givenLoaded)
	sc.Step(`^it is compared against the raw command output$`, w.whenLoaded)
	sc.Step(`^it will be a drawn-together impact picture relating the change set to the operator's footprint$`, w.thenDrawnTogetherPicture)
	sc.Step(`^it will not be a concatenation of unsynthesized dumps$`, w.thenNotConcatenation)

	// Rule: Judge for myself whether the change creates a tension for my work -
	sc.Step(`^a circulating proposal whose change set fell outside the operator's roles, actions, and projects$`, w.givenLoaded)
	sc.Step(`^the proposal-impact-reviewer draws the impact picture$`, w.whenDrawsPicture)
	sc.Step(`^it will report that the change does not touch the operator's current governance$`, w.thenReportsNoTouch)
	sc.Step(`^it will still show what the proposal would change overall$`, w.thenStillShowsOverall)

	sc.Step(`^an operator who had reviewed a proposal's impact but was not ready to answer$`, w.givenLoaded)
	sc.Step(`^the path finishes the review$`, w.whenLoaded)
	sc.Step(`^it will present the impact picture and record no response$`, w.thenPresentsPictureNoResponse)
	sc.Step(`^the review will be a useful result on its own$`, w.thenUsefulOnItsOwn)

	sc.Step(`^a footprint read in which me roles signalled that more roles exist than shown$`, w.givenLoaded)
	sc.Step(`^it will carry the incompleteness forward as footprint coverage$`, w.thenCarriesCoverageForward)
	sc.Step(`^a no-impact conclusion will read as not found in the roles visible to this read$`, w.thenQualifiedNoImpact)
	sc.Step(`^it will never state an unqualified negative over a known-incomplete list$`, w.thenNoUnqualifiedNegative)

	sc.Step(`^the path's use of the change set and the operator's footprint$`, w.givenLoaded)
	sc.Step(`^it is inspected for a fabricated objection verdict or an auto-chosen response$`, w.whenLoaded)
	sc.Step(`^it will draw the impact together for the operator to judge$`, w.thenForOperatorToJudge)
	sc.Step(`^it will neither rule that an objection is required nor answer on the operator's behalf$`, w.thenNoVerdictNoAutoAnswer)

	// Shared Given for the two "skill and agent content" validation scenarios.
	sc.Step(`^the proposal-impact-review skill and agent content$`, w.givenLoaded)

	sc.Step(`^it is inspected for a proposer-side authority verdict or Holacracy coaching$`, w.whenLoaded)
	sc.Step(`^it will not rule on whether the change is within the proposer's authority$`, w.thenNoAuthorityVerdict)
	sc.Step(`^it will not advise on how to weigh an objection$`, w.thenNoObjectionCoaching)

	// Rule: Answer a reviewed proposal without hand-assembling the response ---
	sc.Step(`^a reviewed proposal whose change did not create a tension for the operator's work$`, w.givenLoaded)
	sc.Step(`^the path surfaces the proposal and no-objection for confirmation and runs the respond through the confirmed write flow$`, w.whenRespondsNoObjection)
	sc.Step(`^it will return the recorded response with its prr_ id and the parent proposal's status at the time of response$`, w.thenReturnsRecordedResponse)
	sc.Step(`^the returned parent status will read accepted when that response completed the expected set$`, w.thenAcceptedWhenSetComplete)

	sc.Step(`^a reviewed proposal whose change landed on a role the operator fills in a way they wanted discussed$`, w.givenLoaded)
	sc.Step(`^the path surfaces the proposal and bring-to-meeting for confirmation and runs the respond through the confirmed write flow$`, w.whenRespondsBringToMeeting)
	sc.Step(`^it will return the recorded response, which persists on the proposal and blocks auto-acceptance$`, w.thenPersistsAndBlocksAutoAcceptance)
	sc.Step(`^the path will stop, leaving advancing or withdrawing to the Proposal Circulation Path$`, w.thenLeavesCirculationToPath)

	sc.Step(`^a proposal the operator was ready to answer$`, w.givenLoaded)
	sc.Step(`^the path reaches the respond write$`, w.whenReachesRespond)
	sc.Step(`^it will route the write through the Write-Safety Guardrail's confirmed flow, surfacing the proposal and the chosen response first$`, w.thenRoutesThroughGuardrail)
	sc.Step(`^no response will be recorded when the write is not confirmed$`, w.thenNoResponseWhenDeclined)

	sc.Step(`^a respond attempt the server did not allow$`, w.givenLoaded)
	sc.Step(`^the path runs the response through the confirmed write flow$`, w.whenRespondThroughFlow)
	sc.Step(`^it will surface the API failure by name$`, w.thenSurfacesAPIFailure)
	sc.Step(`^it will record nothing$`, w.thenRecordsNothing)
	sc.Step(`^it will not treat any non-2xx as success or fabricate a state the record does not contain$`, w.thenNoFabrication)

	sc.Step(`^a caller who asked the proposal-impact-reviewer to record the response itself$`, w.givenLoaded)
	sc.Step(`^the reviewer answers$`, w.whenLoaded)
	sc.Step(`^it will refuse and name the respond as the skill's caller-context step, taken after the operator decides$`, w.thenRefusesAndNamesHandoff)
	sc.Step(`^no respond will be run by the reviewer$`, w.thenNoRespondByReviewer)

	sc.Step(`^the path's treatment of the respond write$`, w.givenLoaded)
	sc.Step(`^it is inspected against the Write-Safety Guardrail$`, w.whenInspectAgainstGuardrail)
	sc.Step(`^it will run through the confirmed write flow$`, w.thenRespondRunsConfirmed)
	sc.Step(`^the path will not record the response as if it were an ungated write$`, w.thenNotUngated)
	sc.Step(`^the path will perform no other governance write$`, w.thenNoOtherGovernanceWrite)

	sc.Step(`^it is inspected for any propose, withdraw, or create step$`, w.whenInspectForCirculationOrCreate)
	sc.Step(`^it will contain none$`, w.thenContainsNoCirculationOrCreate)
	sc.Step(`^advancing and withdrawing will remain the Proposal Circulation Path's acts$`, w.thenCirculationRemainsCirculations)
	sc.Step(`^creation will remain the Proposal Drafting Path's act$`, w.thenCreationRemainsDraftings)

	// Drift guard (standalone test is T002) ----------------------------------
	sc.Step(`^the produced proposal-impact-review-path content$`, w.givenImpactReviewContent)
	sc.Step(`^every command it composes is checked against the shipped CLI$`, w.whenCheckedAgainstCLI)
	sc.Step(`^each one will exist$`, w.thenEachExists)
	sc.Step(`^the path will name no command the CLI does not expose$`, w.thenNamesNoLackingCommand)
}

// --- Loading -----------------------------------------------------------------

func (w *proposalImpactReviewWorld) ensureSkillLoaded() error {
	if w.skillRaw == "" {
		skill, err := ReadProposalImpactReviewSkill()
		if err != nil {
			return fmt.Errorf("proposal-impact-review skill did not load: %w", err)
		}
		w.skillRaw, w.skill = skill, normalizeWS(skill)
	}
	return nil
}

func (w *proposalImpactReviewWorld) ensureAgentLoaded() error {
	if w.agentRaw == "" {
		agent, err := ReadProposalImpactReviewerAgent()
		if err != nil {
			return fmt.Errorf("proposal-impact-reviewer agent did not load: %w", err)
		}
		w.agentRaw, w.agent = agent, normalizeWS(agent)
	}
	return nil
}

// ensureLoaded loads both artifacts — used by every scenario except the
// missing-reviewer degradation path, which loads only the skill (see
// givenAgentAbsent) so it faithfully models the agent being absent.
func (w *proposalImpactReviewWorld) ensureLoaded() error {
	if err := w.ensureSkillLoaded(); err != nil {
		return err
	}
	return w.ensureAgentLoaded()
}

func (w *proposalImpactReviewWorld) givenLoaded() error { return w.ensureLoaded() }
func (w *proposalImpactReviewWorld) whenLoaded() error  { return w.ensureLoaded() }

// combined returns the (normalized) skill and agent content together — most
// behavioral scenarios inspect "the path's behavior" as described across both
// artifacts.
func (w *proposalImpactReviewWorld) combined() string { return w.skill + " " + w.agent }

// --- Rule: see what the proposal would change for my own work -----------------

func (w *proposalImpactReviewWorld) whenReadsAndDraws() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !mentionsToken(w.agent, "proposal get") {
		return fmt.Errorf("the review does not read the proposal's change set back through proposal get")
	}
	for _, leaf := range []string{"me roles", "me actions", "me projects"} {
		if !mentionsToken(w.agent, leaf) {
			return fmt.Errorf("the review does not draw the operator's footprint through %s", leaf)
		}
	}
	if !containsFold(w.agent, "change set") {
		return fmt.Errorf("the review does not relate the proposal's change set to the footprint")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenPresentsDrawnTogetherPicture() error {
	if !containsFold(w.agent, "drawn-together") || !containsFold(w.agent, "impact picture") {
		return fmt.Errorf("the agent does not present a drawn-together impact picture")
	}
	if !containsFold(w.agent, "which of the operator's roles the change touches") {
		return fmt.Errorf("the picture does not show which of the operator's roles the change touches and how")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenRecordsNoResponse() error {
	if !containsFold(w.agent, "record no response") {
		return fmt.Errorf("the agent does not state the review records no response")
	}
	if !containsFold(w.agent, "zero writes") {
		return fmt.Errorf("the agent does not state its zero-writes posture")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenSurfacesFailure() error {
	if !containsFold(w.agent, "surface what the failure was") {
		return fmt.Errorf("the agent does not surface what the failure was on a partial review")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenPresentsGathered() error {
	if !containsFold(w.agent, "gathered so far") || !containsFold(w.agent, "flagged incomplete") {
		return fmt.Errorf("the agent does not present what it gathered so far, flagged incomplete")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenNotInventNorAbandonReview() error {
	if !containsFold(w.agent, "invent") || !containsFold(w.agent, "abandon the whole review") {
		return fmt.Errorf("the agent does not state it neither invents the missing data nor abandons the whole review")
	}
	return nil
}

// --- Rule: reachable / degrades ------------------------------------------------

func (w *proposalImpactReviewWorld) givenAgentRegistered() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	// Discoverability: the agent carries the frontmatter (name/description/tools)
	// the host needs, and — like the skill — it is auto-discovered from
	// plugin/agents/, so the manifest declares NO `agents` key.
	for _, field := range []string{"name", "description", "tools"} {
		if !hasFrontmatterField(w.agentRaw, field) {
			return fmt.Errorf("proposal-impact-reviewer frontmatter lacks the %q field the host needs to register it", field)
		}
	}
	_, raw, err := ReadOrientationManifest()
	if err != nil {
		return fmt.Errorf("plugin manifest did not load: %w", err)
	}
	if !ManifestDemandsNoSetup(raw) {
		return fmt.Errorf("plugin.json declares a setup-forcing key (e.g. `agents`) — the reviewer must be auto-discovered from plugin/agents/, not registered via a manifest key")
	}
	return nil
}

func (w *proposalImpactReviewWorld) whenSkillDelegates() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !containsFold(w.skill, "proposal-impact-reviewer") || !containsFold(w.skill, "delegat") {
		return fmt.Errorf("the skill does not delegate the review to the proposal-impact-reviewer agent")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenRunsOwnContext() error {
	if !containsFold(w.agent, "isolated context") {
		return fmt.Errorf("the agent does not state it runs the review in its own isolated context")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenReturnsOnlyPicture() error {
	if !containsFold(w.agent, "impact picture") {
		return fmt.Errorf("the agent output contract never names the impact picture it returns")
	}
	if !containsFold(w.agent, "only the impact picture") && !containsFold(w.agent, "return **only**") {
		return fmt.Errorf("the agent does not state it returns ONLY the impact picture (raw output stays in its context)")
	}
	return nil
}

// givenAgentAbsent models the reviewer agent being absent/unregistered: it loads
// ONLY the skill. The degradation scenario asserts the skill's workflow stands
// alone as guidance and no CLI command breaks — neither needs the agent, and
// loading it would both contradict the premise and fail outright if the agent
// file were genuinely gone.
func (w *proposalImpactReviewWorld) givenAgentAbsent() error { return w.ensureSkillLoaded() }

func (w *proposalImpactReviewWorld) whenSkillConsulted() error { return w.ensureSkillLoaded() }

func (w *proposalImpactReviewWorld) thenWorkflowStandsAsGuidance() error {
	if !containsFold(w.skill, "the workflow") {
		return fmt.Errorf("the skill does not carry the workflow steps as consultable guidance")
	}
	// Self-contained by-hand guidance names every leaf the path composes: the
	// review reads and the one gated respond.
	for _, step := range []string{
		"proposal get", "proposal list",
		"me", "me roles", "me actions", "me projects",
		"roles", "domains", "policies",
		"proposal respond",
	} {
		if !mentionsToken(w.skill, step) {
			return fmt.Errorf("the skill workflow is not self-contained guidance — it omits the %q step", step)
		}
	}
	if !containsFold(w.skill, "follow by hand") && !containsFold(w.skill, "degrades to guidance") {
		return fmt.Errorf("the skill does not state it remains usable as guidance when the agent is absent")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenNoCLIBroken() error {
	// The plugin tree is pure data: nothing under plugin/ compiles into the CLI,
	// so an absent/unregistered agent cannot break a glassfrog command.
	clean, err := OrientationPluginHasNoGoCode()
	if err != nil {
		return fmt.Errorf("could not inspect the plugin tree: %w", err)
	}
	if !clean {
		return fmt.Errorf("plugin tree carries Go code — the agent's absence could no longer be isolated from the CLI")
	}
	return nil
}

// --- Rule: synthesized picture ---------------------------------------------------

func (w *proposalImpactReviewWorld) thenDrawnTogetherPicture() error {
	if !containsFold(w.agent, "drawn-together") {
		return fmt.Errorf("the output contract is not framed as a drawn-together impact picture")
	}
	for _, el := range []string{"proposal", "changes", "footprint", "footprint_coverage", "intersections", "pending", "notes"} {
		if !mentionsToken(w.agent, el) {
			return fmt.Errorf("the drawn-together picture omits the %q element", el)
		}
	}
	if !containsFold(w.agent, "prp_") {
		return fmt.Errorf("the picture does not carry the concrete prp_ id per element")
	}
	if !containsFold(w.agent, "every element carries the id") {
		return fmt.Errorf("the output contract does not state every element carries the id needed to act on it")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenNotConcatenation() error {
	if !containsFold(w.agent, "concatenation") || !containsFold(w.agent, "unsynthesized") {
		return fmt.Errorf("the agent does not state the picture is not a concatenation of unsynthesized dumps")
	}
	return nil
}

// --- Rule: judge for myself ------------------------------------------------------

func (w *proposalImpactReviewWorld) whenDrawsPicture() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !containsFold(w.agent, "impact picture") || !containsFold(w.agent, "synthesize") {
		return fmt.Errorf("the agent does not draw (synthesize) the impact picture")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenReportsNoTouch() error {
	if !containsFold(w.agent, "does not touch the operator's current governance") {
		return fmt.Errorf("the agent does not report plainly that the change does not touch the operator's current governance")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenStillShowsOverall() error {
	if !containsFold(w.agent, "what the proposal would change overall") {
		return fmt.Errorf("the no-impact picture does not still show what the proposal would change overall")
	}
	if !containsFold(w.agent, "load-bearing") {
		return fmt.Errorf("the agent does not treat the no-impact review as a load-bearing result")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenPresentsPictureNoResponse() error {
	if !containsFold(w.skill, "first-class exit") {
		return fmt.Errorf("the skill does not treat the not-yet exit as first-class")
	}
	if !containsFold(w.skill, "record no response") {
		return fmt.Errorf("the skill does not state the review-only exit records no response")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenUsefulOnItsOwn() error {
	if !containsFold(w.skill, "useful result on its own") && !containsFold(w.skill, "stands on its own") {
		return fmt.Errorf("the skill does not state the review is a useful result on its own")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenCarriesCoverageForward() error {
	if !mentionsToken(w.agent, "footprint_coverage") {
		return fmt.Errorf("the picture carries no footprint_coverage element")
	}
	if !containsFold(w.agent, "tri-state") {
		return fmt.Errorf("footprint_coverage is not tri-state")
	}
	if !containsFold(w.agent, "never a silent complete") {
		return fmt.Errorf("the agent does not forbid a silent complete coverage")
	}
	if !containsFold(w.agent, "incompleteness signal") {
		return fmt.Errorf("the agent does not read the me roles incompleteness signal (stderr note / pagination metadata)")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenQualifiedNoImpact() error {
	if !containsFold(w.agent, "not in the roles visible to this read (list incomplete)") {
		return fmt.Errorf("a no-impact conclusion over an incomplete footprint is not qualified as \"not in the roles visible to this read (list incomplete)\"")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenNoUnqualifiedNegative() error {
	if !containsFold(w.agent, "unqualified negative over a known-incomplete list") {
		return fmt.Errorf("the agent does not forbid an unqualified negative over a known-incomplete list")
	}
	return nil
}

// --- Rule: reviews inform, never decide --------------------------------------------

func (w *proposalImpactReviewWorld) thenForOperatorToJudge() error {
	if !containsFold(w.agent, "for the operator to judge") {
		return fmt.Errorf("the agent does not draw the impact together for the operator to judge")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenNoVerdictNoAutoAnswer() error {
	if !containsFold(w.agent, "no verdict field") || !containsFold(w.agent, "no recommended response") {
		return fmt.Errorf("the output contract does not exclude a verdict field and a recommended response")
	}
	if !containsFold(w.agent, "never rule that an objection is required") {
		return fmt.Errorf("the agent does not state it never rules that an objection is required")
	}
	if !containsFold(w.agent, "on the operator's behalf") {
		return fmt.Errorf("the agent does not state it never answers on the operator's behalf")
	}
	if !containsFold(w.skill, "never infers or defaults the value") {
		return fmt.Errorf("the skill does not forbid inferring or defaulting the response value from the review")
	}
	if !containsFold(w.skill, "not an instruction to answer") {
		return fmt.Errorf("the skill does not state that \"no objections found\" is not an instruction to answer no_objection")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenNoAuthorityVerdict() error {
	if !containsFold(w.combined(), "within the proposer's authority") {
		return fmt.Errorf("the artifacts do not disclaim ruling on whether the change is within the proposer's authority")
	}
	if !containsFold(w.combined(), "do not rule") && !containsFold(w.combined(), "does not rule") {
		return fmt.Errorf("the artifacts do not state they do not rule on the authority question")
	}
	if !containsFold(w.combined(), "Constraint Discovery Path") {
		return fmt.Errorf("the artifacts do not hand the authority question to the Constraint Discovery Path by its in-plugin name")
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
			return fmt.Errorf("the artifacts carry an authority verdict %q — the path must review, not judge", verdict)
		}
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenNoObjectionCoaching() error {
	if !containsFold(w.combined(), "advise on how to weigh an objection") {
		return fmt.Errorf("the artifacts do not disclaim advising on how to weigh an objection")
	}
	if !containsFold(w.combined(), "coach") {
		return fmt.Errorf("the artifacts do not disclaim coaching Holacracy practice")
	}
	return nil
}

// --- Rule: answer without hand-assembling the response ------------------------------

func (w *proposalImpactReviewWorld) whenRespondsNoObjection() error {
	if err := w.whenReachesRespond(); err != nil {
		return err
	}
	if !mentionsToken(w.skill, "no_objection") {
		return fmt.Errorf("the skill does not carry the no_objection response value")
	}
	return nil
}

func (w *proposalImpactReviewWorld) whenRespondsBringToMeeting() error {
	if err := w.whenReachesRespond(); err != nil {
		return err
	}
	if !mentionsToken(w.skill, "bring_to_meeting") {
		return fmt.Errorf("the skill does not carry the bring_to_meeting response value")
	}
	return nil
}

func (w *proposalImpactReviewWorld) whenReachesRespond() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !mentionsToken(w.skill, "proposal respond") {
		return fmt.Errorf("the workflow does not record the response through the proposal respond command")
	}
	if !containsFold(w.skill, "confirmed write flow") {
		return fmt.Errorf("the workflow does not run the respond through the confirmed write flow")
	}
	if !containsFold(w.skill, "narrate the proposal") {
		return fmt.Errorf("the skill does not surface (narrate) the proposal before the respond")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenReturnsRecordedResponse() error {
	if !containsFold(w.skill, "prr_") {
		return fmt.Errorf("the recorded response does not carry its prr_ id")
	}
	if !containsFold(w.skill, "status at the time of response") {
		return fmt.Errorf("the recorded response does not carry the parent proposal's status at the time of response")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenAcceptedWhenSetComplete() error {
	if !containsFold(w.skill, "accepted") || !containsFold(w.skill, "completed the expected set") {
		return fmt.Errorf("the skill does not state accepted means the response completed the expected set")
	}
	if !containsFold(w.skill, "closed the consent window") {
		return fmt.Errorf("the skill does not read accepted as the response closing the consent window")
	}
	if !containsFold(w.skill, "never computed client-side") {
		return fmt.Errorf("the skill does not state acceptance is surfaced from the record, never computed client-side")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenPersistsAndBlocksAutoAcceptance() error {
	if !containsFold(w.skill, "persists on the proposal") || !containsFold(w.skill, "blocks auto-acceptance") {
		return fmt.Errorf("the skill does not state a recorded bring_to_meeting persists on the proposal and blocks auto-acceptance")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenLeavesCirculationToPath() error {
	if !containsFold(w.skill, "stops") {
		return fmt.Errorf("the skill does not stop after the recorded response")
	}
	if !containsFold(w.combined(), "Proposal Circulation Path") {
		return fmt.Errorf("the artifacts do not leave advancing or withdrawing to the Proposal Circulation Path by its in-plugin name")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenRoutesThroughGuardrail() error {
	if !containsFold(w.skill, "Write-Safety Guardrail") {
		return fmt.Errorf("the skill does not route the respond through the Write-Safety Guardrail")
	}
	if !containsFold(w.skill, "confirmed write flow") {
		return fmt.Errorf("the skill does not route the respond through the confirmed write flow")
	}
	if !containsFold(w.skill, "narrate the proposal and the chosen value") {
		return fmt.Errorf("the skill does not surface the proposal and the chosen response before issuing")
	}
	if !containsFold(w.skill, "complete payload") {
		return fmt.Errorf("the skill does not state the inline value makes the confirmation show the complete payload")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenNoResponseWhenDeclined() error {
	if !containsFold(w.skill, "declined") {
		return fmt.Errorf("the skill does not treat a declined confirmation as an outcome")
	}
	if !containsFold(w.skill, "no response is recorded") {
		return fmt.Errorf("the skill does not state no response is recorded when the write is not confirmed")
	}
	return nil
}

func (w *proposalImpactReviewWorld) whenRespondThroughFlow() error { return w.whenReachesRespond() }

func (w *proposalImpactReviewWorld) thenSurfacesAPIFailure() error {
	if !containsFold(w.skill, "API failure by name") {
		return fmt.Errorf("the skill does not surface a rejected respond's API failure by name")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenRecordsNothing() error {
	if !containsFold(w.skill, "records nothing") {
		return fmt.Errorf("the skill does not state a rejected respond records nothing")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenNoFabrication() error {
	if !containsFold(w.skill, "non-2xx") {
		return fmt.Errorf("the skill does not state it never treats a non-2xx as success")
	}
	if !containsFold(w.skill, "fabricates no state") || !containsFold(w.skill, "record does not contain") {
		return fmt.Errorf("the skill does not state it fabricates no state the record does not contain")
	}
	return nil
}

// --- Rule: the split write locus ----------------------------------------------------

func (w *proposalImpactReviewWorld) thenRefusesAndNamesHandoff() error {
	if !containsFold(w.agent, "refuse") {
		return fmt.Errorf("the agent does not refuse when asked to record the response itself")
	}
	if !containsFold(w.agent, "caller-context step") || !containsFold(w.agent, "after the operator decides") {
		return fmt.Errorf("the agent does not name the respond as the skill's caller-context step, taken after the operator decides")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenNoRespondByReviewer() error {
	if !containsFold(w.agent, "never run") || !mentionsToken(w.agent, "proposal respond") {
		return fmt.Errorf("the agent does not state it never runs proposal respond")
	}
	if !containsFold(w.agent, "is ever run by you") {
		return fmt.Errorf("the agent does not state no proposal respond is ever run by it")
	}
	return nil
}

// whenInspectAgainstGuardrail exercises the path's one write against 063's REAL
// PreToolUse gate script and pins the read-posture tool grant + gate membership:
// the composed respond must be gated (asked), the nine reads must not, and the
// agent tool grant must keep Bash while withholding Write/Edit.
func (w *proposalImpactReviewWorld) whenInspectAgainstGuardrail() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	w.tools, w.hasTools = AgentTools(w.agentRaw)
	gated, err := ReadGatedRegistry()
	if err != nil {
		return fmt.Errorf("could not read 063's gated-command registry: %w", err)
	}
	w.gated = gated
	composed, err := ReadProposalImpactReviewCommands()
	if err != nil {
		return fmt.Errorf("could not read the composed-leaf registry: %w", err)
	}
	w.composed = composed
	return nil
}

func (w *proposalImpactReviewWorld) thenRespondRunsConfirmed() error {
	// The respond is asked (gated) at 063's real gate — the confirmed write flow.
	// A concrete command line, exactly as the caller would run it: the one-token
	// value rides inline so the confirmation shows the complete payload.
	dec, _, err := runGateScript("glassfrog proposal respond prp_0123 --response no_objection")
	if err != nil {
		return fmt.Errorf("063's gate errored on the respond: %v", err)
	}
	if dec != "ask" {
		return fmt.Errorf("the respond was not gated (decision %q) — the response must run through the confirmed write flow", dec)
	}
	if !containsFold(w.skill, "confirmed write flow") {
		return fmt.Errorf("the skill does not state the respond runs through the confirmed write flow")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenNotUngated() error {
	// Structurally: the one gated write is a composed leaf AND a member of 063's
	// gated set, and every other composed leaf (the nine reads) is absent from
	// it. Anchored per write leaf (ProposalImpactReviewGatedWrite) — deriving the
	// write from the gated set alone, or counting "exactly one gated composed
	// leaf", would pass if a read were swapped in for the respond (the
	// grounding/situating reads share the proposal group and a swap preserves
	// the count), the exact regression this check exists to catch.
	gatedSet := setOf(w.gated)
	composedSet := setOf(w.composed)
	if !composedSet[ProposalImpactReviewGatedWrite] {
		return fmt.Errorf("the composed set no longer names the gated write %q — the guardrail-crossing respond is missing from what the path composes", ProposalImpactReviewGatedWrite)
	}
	if !gatedSet[ProposalImpactReviewGatedWrite] {
		return fmt.Errorf("the gated write %q is not a member of 063's gated set — the response would ship unconfirmed", ProposalImpactReviewGatedWrite)
	}
	for _, leaf := range w.composed {
		if leaf != ProposalImpactReviewGatedWrite && gatedSet[leaf] {
			return fmt.Errorf("composed leaf %q is a read but is gated by 063 — the review would start prompting", leaf)
		}
	}
	// A grounding read passes 063's gate ungated (would-be prompting is the
	// defect).
	dec, _, err := runGateScript("glassfrog proposal get prp_0123")
	if err != nil {
		return fmt.Errorf("063's gate errored on the grounding read: %v", err)
	}
	if dec == "ask" {
		return fmt.Errorf("the grounding read proposal get was gated (ask) — the review must not start prompting")
	}
	// The tool grant withholds Write/Edit (no workspace mutation) while keeping
	// Bash (to invoke the composed reads).
	if !w.hasTools {
		return fmt.Errorf("the agent declares no tools grant — the read posture cannot be asserted structurally")
	}
	hasBash := false
	for _, tool := range w.tools {
		if strings.EqualFold(tool, "Bash") {
			hasBash = true
		}
		for _, forbidden := range []string{"Write", "Edit"} {
			if strings.EqualFold(tool, forbidden) {
				return fmt.Errorf("the agent tool grant includes %q — the reviewer must not be able to mutate the workspace", forbidden)
			}
		}
	}
	if !hasBash {
		return fmt.Errorf("the agent tool grant omits Bash — it could not invoke the composed reads (a misconfiguration, not a safety win)")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenNoOtherGovernanceWrite() error {
	// The path RUNS exactly one gated write — the respond. Any other gated leaf
	// in the composed set is a proposal write the path must not run (create is
	// 067's, propose/withdraw are 068's).
	gatedSet := setOf(w.gated)
	for _, leaf := range w.composed {
		if gatedSet[leaf] && leaf != ProposalImpactReviewGatedWrite {
			return fmt.Errorf("the composed set runs the gated write %q beyond the respond — creating, advancing, or withdrawing a proposal is never this path's act", leaf)
		}
	}
	return nil
}

// whenInspectForCirculationOrCreate loads 063's gated registry and the composed
// leaf list — the source-derived enumeration of what a proposal write IS
// (create/propose/withdraw are the gated writes the path must NOT compose).
func (w *proposalImpactReviewWorld) whenInspectForCirculationOrCreate() error {
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
	composed, err := ReadProposalImpactReviewCommands()
	if err != nil {
		return fmt.Errorf("could not read the composed-leaf registry: %w", err)
	}
	w.composed = composed
	return nil
}

func (w *proposalImpactReviewWorld) thenContainsNoCirculationOrCreate() error {
	// The composed set carries no propose, withdraw, or create step, and the
	// gated composed leaves are EXACTLY the one write anchor: any other gated
	// leaf in the composed set is a proposal write the path must not run.
	composedSet := setOf(w.composed)
	for _, foreign := range []string{"proposal propose", "proposal withdraw", "proposal create"} {
		if composedSet[foreign] {
			return fmt.Errorf("the composed set names %q — advancing, withdrawing, or creating a proposal is never this path's step", foreign)
		}
	}
	gatedSet := setOf(w.gated)
	if !composedSet[ProposalImpactReviewGatedWrite] || !gatedSet[ProposalImpactReviewGatedWrite] {
		return fmt.Errorf("the path's one gated write %q is no longer composed and gated — the respond would ship unconfirmed", ProposalImpactReviewGatedWrite)
	}
	for _, leaf := range w.composed {
		if gatedSet[leaf] && leaf != ProposalImpactReviewGatedWrite {
			return fmt.Errorf("the composed set runs the gated write %q beyond the respond — a circulation or creation step must not enter this path", leaf)
		}
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenCirculationRemainsCirculations() error {
	if !containsFold(w.combined(), "Proposal Circulation Path") {
		return fmt.Errorf("the artifacts do not leave advancing and withdrawing to the Proposal Circulation Path by its in-plugin name")
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenCreationRemainsDraftings() error {
	if !containsFold(w.combined(), "Proposal Drafting Path") {
		return fmt.Errorf("the artifacts do not leave creation to the Proposal Drafting Path by its in-plugin name")
	}
	return nil
}

// --- Drift guard (standalone test is T002) -------------------------------------

func (w *proposalImpactReviewWorld) givenImpactReviewContent() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	composed, err := ReadProposalImpactReviewCommands()
	if err != nil {
		return fmt.Errorf("could not read the composed-leaf registry: %w", err)
	}
	if len(composed) == 0 {
		return fmt.Errorf("the composed-leaf registry is empty — nothing to check against the CLI")
	}
	w.composed = composed
	return nil
}

func (w *proposalImpactReviewWorld) whenCheckedAgainstCLI() error {
	liveTop, err := LiveTopLevelCommands()
	if err != nil {
		return fmt.Errorf("could not extract the CLI's top-level command surface: %w", err)
	}
	if len(liveTop) == 0 {
		return fmt.Errorf("extracted no top-level commands — the surface anchor could not be read")
	}
	w.liveTop = liveTop
	liveMe, err := LiveMeSubcommands()
	if err != nil {
		return fmt.Errorf("could not extract the CLI's `me` subcommand surface: %w", err)
	}
	if len(liveMe) == 0 {
		return fmt.Errorf("extracted no `me` subcommands — the me surface anchor could not be read")
	}
	w.liveMe = liveMe
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
	w.drift = CheckProposalImpactReviewDrift(w.composed, w.liveTop, w.liveMe, w.liveProposal, w.gated, w.skillRaw, w.agentRaw)
	return nil
}

func (w *proposalImpactReviewWorld) thenEachExists() error {
	if len(w.drift) != 0 {
		return fmt.Errorf("a composed leaf no longer resolves in the shipped CLI (or violates the gate-membership invariant):\n  - %s", joinDrift(w.drift))
	}
	// Positive per-leaf existence, independent of the drift check's own loop:
	// every leaf resolves against the anchor its shape names.
	topSet := setOf(w.liveTop)
	meSet := setOf(w.liveMe)
	proposalSet := setOf(w.liveProposal)
	for _, leaf := range w.composed {
		fields := strings.Fields(leaf)
		switch {
		case leaf == "me":
			if len(w.liveMe) == 0 {
				return fmt.Errorf("composed leaf %q could not be anchored to the CLI's `me` wiring", leaf)
			}
		case len(fields) == 1:
			if !topSet[leaf] {
				return fmt.Errorf("composed leaf %q does not exist as a top-level command in the CLI", leaf)
			}
		case len(fields) == 2 && fields[0] == "me":
			if !meSet[fields[1]] {
				return fmt.Errorf("composed leaf %q does not exist as a subcommand of the CLI's me command", leaf)
			}
		case len(fields) == 2 && fields[0] == "proposal":
			if !proposalSet[fields[1]] {
				return fmt.Errorf("composed leaf %q does not exist as a subcommand of the CLI's proposal command", leaf)
			}
		default:
			return fmt.Errorf("composed leaf %q has a shape the guard cannot anchor", leaf)
		}
	}
	return nil
}

func (w *proposalImpactReviewWorld) thenNamesNoLackingCommand() error {
	// The same drift result proves the path names no command the CLI does not
	// expose.
	if len(w.drift) != 0 {
		return fmt.Errorf("the path names a command the CLI does not expose:\n  - %s", joinDrift(w.drift))
	}
	return nil
}
