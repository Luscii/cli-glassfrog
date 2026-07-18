package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestConstraintDiscoveryFeatures runs the executable acceptance for the
// Constraint Discovery Path (065). Like the sibling build-side suites
// (021/022/036/062/063/064) its Paths name ONLY this spec's feature file and it
// runs with the ~@wip filter, so only the scenarios implemented so far execute.
//
// The deliverable is a pair of declarative Claude-plugin artifacts — a thin
// `constraint-discovery` skill (which also owns the clarify-when-vague branch,
// ADR-3) that delegates to a read-only, non-interactive `constraint-navigator`
// agent — plus a single-sourced composed-read registry and a best-effort drift
// guard. The artifacts carry no runtime Go path of their own, so the executable
// scenarios assert against their content: the skill's when + workflow + clarify
// + delegation, the agent's read-only grant + isolated-execution +
// synthesized-picture output contract with its composed characterization +
// defensive traversal, and the drift guard that keeps the composed read leaves
// truthful to the shipped CLI (the drift scenario is implemented in T002).
func TestConstraintDiscoveryFeatures(t *testing.T) {
	w := &constraintWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.register(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unequipped-agent-operators/constraint-discovery-path.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: constraint-discovery-path feature scenarios failed")
	}
}

// constraintWorld is the per-scenario state: the loaded skill + agent content,
// the parsed agent tool grant, and the inspection mode the shared "it will
// contain none" step dispatches on.
type constraintWorld struct {
	// skill/agent hold the artifacts with whitespace collapsed to single spaces,
	// so phrase assertions are resilient to markdown line-wrapping (a hard-wrapped
	// "does not clearly\nanswer" still matches "does not clearly answer").
	skill string
	agent string
	// skillRaw/agentRaw keep the verbatim content for structural checks that
	// depend on line breaks (frontmatter delimiters, the tools list).
	skillRaw string
	agentRaw string
	tools    []string
	hasTools bool
	// inspect records which inspection the When step performed — "verdict" (a
	// permission verdict computed from local logic) or "write" (any write,
	// confirm, or gate step) — so the shared "it will contain none" Then can
	// dispatch on world state instead of guessing (two scenarios share the text).
	inspect string
	// Drift-guard state (T002): the single-sourced composed reads, the CLI's
	// live top-level and `me`-subcommand surfaces, and the drift findings.
	composed []string
	liveTop  []string
	liveMe   []string
	drift    []string
}

func (w *constraintWorld) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = constraintWorld{}
		return ctx, nil
	})

	// Rule: Follow a guided path from a wanted action to the governing
	// domains and policies --------------------------------------------------
	sc.Step(`^a practitioner's free-form wanted action with no role or domain in hand$`, w.givenLoaded)
	sc.Step(`^the constraint-navigator traverses the action$`, w.whenLoaded)
	sc.Step(`^it will search the governance record for what the action touches$`, w.thenSearchesRecord)
	sc.Step(`^it will surface the domain that governs the action and the role that holds it$`, w.thenSurfacesDomainAndOwner)
	sc.Step(`^it will characterize the action as falling under that role's authority, needing its permission or a proposal$`, w.thenCharacterizesOtherRole)
	sc.Step(`^each domain and role will carry the id needed to read it again$`, w.thenDomainRoleCarryId)

	sc.Step(`^a wanted action governed by a domain that a role the caller fills holds$`, w.givenLoaded)
	sc.Step(`^the constraint-navigator reads the caller's own roles and the governing domain$`, w.whenLoaded)
	sc.Step(`^it will find the governing domain belongs to a role the caller fills$`, w.thenFindsOwnDomain)
	sc.Step(`^it will characterize the action as within the caller's own authority$`, w.thenCharacterizesOwnAuthority)
	sc.Step(`^it will not frame the action as needing another role's permission$`, w.thenNotFrameOtherPermission)

	sc.Step(`^the caller's own-roles read returned an incomplete list with an incompleteness note$`, w.givenLoaded)
	sc.Step(`^the constraint-navigator cannot confirm whether the governing domain's role is one the caller fills$`, w.whenLoaded)
	sc.Step(`^it will mark the owned-by-caller finding as uncertain rather than false$`, w.thenMarksUncertain)
	sc.Step(`^it will surface that the roles list was incomplete$`, w.thenSurfacesIncomplete)
	sc.Step(`^it will not characterize the action as another role's domain on the unconfirmed match$`, w.thenNotMisattribute)

	sc.Step(`^a wanted action described too vaguely to search for its governing governance$`, w.givenSkillOnly)
	sc.Step(`^the constraint-discovery skill begins$`, w.whenSkillOnly)
	sc.Step(`^the skill will ask the operator to sharpen the action before delegating$`, w.thenAsksSharpen)
	sc.Step(`^it will not guess a meaning and traverse on the guess$`, w.thenNotGuess)
	sc.Step(`^the constraint-navigator will not be invoked until the action is well-formed$`, w.thenNotInvokedUntilWellFormed)

	sc.Step(`^an action so broad that the search matched many roles, domains, and policies across several pages$`, w.givenLoaded)
	sc.Step(`^the constraint-navigator assembles the picture$`, w.whenLoaded)
	sc.Step(`^it will page through the full result set before choosing what is most relevant$`, w.thenPagesFullSet)
	sc.Step(`^it will present the most relevant governing constraints rather than every match$`, w.thenPresentsMostRelevant)
	sc.Step(`^it will note that the picture was narrowed so the practitioner can refine$`, w.thenNotesNarrowed)

	sc.Step(`^a traversal in which one read failed while the others succeeded$`, w.givenLoaded)
	sc.Step(`^it will surface what the failure was$`, w.thenSurfacesFailure)
	sc.Step(`^it will return the picture built from the reads that succeeded$`, w.thenReturnsSucceeded)
	sc.Step(`^it will not invent the missing piece$`, w.thenNotInvent)

	sc.Step(`^the plugin was present with the constraint-navigator agent registered$`, w.givenAgentRegistered)
	sc.Step(`^the constraint-discovery skill delegates a well-formed action for traversal$`, w.whenSkillDelegates)
	sc.Step(`^the navigator will run the traversal in its own context$`, w.thenRunsOwnContext)
	sc.Step(`^it will return only the synthesized picture to the caller$`, w.thenReturnsOnlyPicture)

	sc.Step(`^the plugin was present but the constraint-navigator agent was absent or unregistered$`, w.givenSkillOnly)
	sc.Step(`^the constraint-discovery skill is consulted for an action$`, w.whenSkillOnly)
	sc.Step(`^its workflow will remain readable as guidance the caller can follow by hand$`, w.thenWorkflowStandsAsGuidance)
	sc.Step(`^no command in the CLI will be broken by the agent's absence$`, w.thenNoCLIBroken)

	// Rule: See what governs the action, drawn from the record not the
	// tool's opinion ---------------------------------------------------------
	sc.Step(`^an action the constraint-navigator had located in the governance record$`, w.givenLoaded)
	sc.Step(`^it traverses the policies bearing on the action$`, w.whenLoaded)
	sc.Step(`^it will surface the policy that grants or limits the action$`, w.thenSurfacesPolicy)
	sc.Step(`^it will present it as the constraint to observe, drawn together with any governing domain$`, w.thenConstraintToObserve)
	sc.Step(`^it will not present it as a concatenation of separate dumps$`, w.thenNotSeparateDumps)

	sc.Step(`^an action for which no domain in view governs it and no policy in view limits it$`, w.givenLoaded)
	sc.Step(`^the constraint-navigator completes the traversal$`, w.whenLoaded)
	sc.Step(`^it will surface that the record shows nothing constraining the action$`, w.thenSurfacesAbsence)
	sc.Step(`^it will report that absence plainly$`, w.thenReportsPlainly)
	sc.Step(`^it will not assert that the operator is permitted$`, w.thenNotAssertPermitted)

	sc.Step(`^an action for which the match is ambiguous and no domain plainly owns it$`, w.givenLoaded)
	sc.Step(`^it will characterize the situation as one the record does not clearly answer$`, w.thenCharacterizesUnclear)
	sc.Step(`^it will surface what it found$`, w.thenSurfacesWhatFound)
	sc.Step(`^it will not fabricate an authority ruling to resolve the ambiguity$`, w.thenNotFabricateRuling)

	sc.Step(`^the constraint-navigator's treatment of the wanted action$`, w.givenLoaded)
	sc.Step(`^it is inspected for a permission verdict computed from local logic$`, w.whenInspectForVerdict)
	sc.Step(`^it will contain none$`, w.thenContainsNone)
	sc.Step(`^it will only surface the governing domains and policies drawn from the record$`, w.thenOnlySurfacesFromRecord)
	sc.Step(`^it will nowhere reimplement permission rules or rule on whether the action is allowed$`, w.thenNowhereReimplements)

	sc.Step(`^the path's handling of an action the record does not clearly constrain$`, w.givenLoaded)
	sc.Step(`^its result is inspected$`, w.whenLoaded)
	sc.Step(`^it will state what is unclear and surface what it found$`, w.thenStatesUnclear)
	sc.Step(`^it will nowhere assert a permitted or forbidden verdict it cannot ground in the record$`, w.thenNowhereAssertsUngrounded)

	sc.Step(`^the picture the constraint-navigator returned$`, w.givenLoaded)
	sc.Step(`^it is compared against the raw command output$`, w.whenLoaded)
	sc.Step(`^it will be a drawn-together picture of the governing domains and policies$`, w.thenDrawnTogether)
	sc.Step(`^it will not be a concatenation of unsynthesized dumps$`, w.thenNotConcatenation)

	// Rule: Move from a surfaced constraint to action with every element
	// carrying its id ---------------------------------------------------------
	sc.Step(`^a picture the constraint-navigator had returned for a wanted action$`, w.givenLoaded)
	sc.Step(`^the caller inspects each domain, policy, and role in it$`, w.whenLoaded)
	sc.Step(`^every element will carry the id needed to read it again$`, w.thenEveryElementCarriesId)
	sc.Step(`^the caller will be able to act on any element without re-running the search$`, w.thenActWithoutReRunning)

	sc.Step(`^the constraint-discovery skill and agent content$`, w.givenLoaded)
	sc.Step(`^it is inspected for any write, confirm, or gate step$`, w.whenInspectForWrite)
	sc.Step(`^the path will only read$`, w.thenOnlyReads)

	// Drift guard (T002) -----------------------------------------------------
	sc.Step(`^the produced constraint-discovery-path content$`, w.givenConstraintContent)
	sc.Step(`^every command and read it composes is checked against the shipped CLI$`, w.whenCheckedAgainstCLI)
	sc.Step(`^each one will exist$`, w.thenEachExists)
	sc.Step(`^the path will name no read the CLI does not expose$`, w.thenNamesNoLackingRead)
}

// --- Loading -----------------------------------------------------------------

func (w *constraintWorld) ensureSkillLoaded() error {
	if w.skillRaw == "" {
		skill, err := ReadConstraintSkill()
		if err != nil {
			return fmt.Errorf("constraint-discovery skill did not load: %w", err)
		}
		w.skillRaw, w.skill = skill, normalizeWS(skill)
	}
	return nil
}

func (w *constraintWorld) ensureAgentLoaded() error {
	if w.agentRaw == "" {
		agent, err := ReadConstraintNavigatorAgent()
		if err != nil {
			return fmt.Errorf("constraint-navigator agent did not load: %w", err)
		}
		w.agentRaw, w.agent = agent, normalizeWS(agent)
	}
	return nil
}

// ensureLoaded loads both artifacts — used by every scenario except the
// clarify-when-vague and missing-navigator paths, which load only the skill:
// the clarify branch lives in the skill alone (ADR-3), and the degradation
// scenario models the agent being absent (loading it would contradict the
// premise and fail outright if the agent file were genuinely gone).
func (w *constraintWorld) ensureLoaded() error {
	if err := w.ensureSkillLoaded(); err != nil {
		return err
	}
	return w.ensureAgentLoaded()
}

func (w *constraintWorld) givenLoaded() error    { return w.ensureLoaded() }
func (w *constraintWorld) whenLoaded() error     { return w.ensureLoaded() }
func (w *constraintWorld) givenSkillOnly() error { return w.ensureSkillLoaded() }
func (w *constraintWorld) whenSkillOnly() error  { return w.ensureSkillLoaded() }

// combined returns the (normalized) skill and agent content together — most
// behavioral scenarios inspect "the path's behavior" as described across both
// artifacts.
func (w *constraintWorld) combined() string { return w.skill + " " + w.agent }

// --- Rule: guided path --------------------------------------------------------

func (w *constraintWorld) thenSearchesRecord() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !mentionsToken(w.combined(), "search") || !containsFold(w.combined(), "discover") {
		return fmt.Errorf("the workflow does not describe searching the record to discover what the action touches")
	}
	return nil
}

func (w *constraintWorld) thenSurfacesDomainAndOwner() error {
	if !containsFold(w.combined(), "governing domain") || !containsFold(w.combined(), "role that holds") {
		return fmt.Errorf("the artifacts do not surface the governing domain together with the role that holds it")
	}
	return nil
}

func (w *constraintWorld) thenCharacterizesOtherRole() error {
	if !containsFold(w.agent, "under that role's authority") || !containsFold(w.agent, "permission or a proposal") {
		return fmt.Errorf("the characterization does not name the action as falling under the owning role's authority, needing its permission or a proposal")
	}
	return nil
}

func (w *constraintWorld) thenDomainRoleCarryId() error {
	if !containsFold(w.agent, "role_") || !containsFold(w.agent, "id needed to read it again") {
		return fmt.Errorf("the output contract does not give each domain and role the id needed to read it again")
	}
	return nil
}

// --- Rule: own-vs-other and the incomplete roles list --------------------------

func (w *constraintWorld) thenFindsOwnDomain() error {
	if !containsFold(w.combined(), "a role the caller fills") {
		return fmt.Errorf("the artifacts never determine whether the governing domain belongs to a role the caller fills")
	}
	if !mentionsToken(w.combined(), "me roles") {
		return fmt.Errorf("the own-vs-other determination does not read the caller's own roles via `me roles`")
	}
	return nil
}

func (w *constraintWorld) thenCharacterizesOwnAuthority() error {
	if !containsFold(w.agent, "within the caller's own authority") {
		return fmt.Errorf("the characterization does not name an own-domain action as within the caller's own authority")
	}
	return nil
}

func (w *constraintWorld) thenNotFrameOtherPermission() error {
	if !containsFold(w.agent, "do not frame it as needing another role's permission") {
		return fmt.Errorf("the agent does not rule out framing an own-domain action as needing another role's permission")
	}
	return nil
}

func (w *constraintWorld) thenMarksUncertain() error {
	if !containsFold(w.agent, "uncertain") || !containsFold(w.agent, "never a definite") {
		return fmt.Errorf("the agent does not mark the owned-by-caller finding uncertain (never a definite false) on an incomplete roles list")
	}
	return nil
}

func (w *constraintWorld) thenSurfacesIncomplete() error {
	if !containsFold(w.agent, "incomplete") {
		return fmt.Errorf("the agent does not surface that the roles list was incomplete")
	}
	return nil
}

func (w *constraintWorld) thenNotMisattribute() error {
	if !containsFold(w.agent, "unconfirmed match") || !containsFold(w.agent, "do not characterize") {
		return fmt.Errorf("the agent does not rule out characterizing the action as another role's domain on an unconfirmed match")
	}
	return nil
}

// --- Rule: clarify-when-vague (skill only, ADR-3) -------------------------------

func (w *constraintWorld) thenAsksSharpen() error {
	if !containsFold(w.skill, "sharpen") || !containsFold(w.skill, "before delegating") {
		return fmt.Errorf("the skill does not ask the operator to sharpen a too-vague action before delegating")
	}
	if !containsFold(w.skill, "ask") {
		return fmt.Errorf("the skill does not describe asking the operator")
	}
	return nil
}

func (w *constraintWorld) thenNotGuess() error {
	if !containsFold(w.skill, "never guesses a meaning") {
		return fmt.Errorf("the skill does not rule out guessing a meaning and traversing on the guess")
	}
	return nil
}

func (w *constraintWorld) thenNotInvokedUntilWellFormed() error {
	if !containsFold(w.skill, "never invoked on a guess") || !containsFold(w.skill, "well-formed") {
		return fmt.Errorf("the skill does not keep the constraint-navigator uninvoked until the action is well-formed")
	}
	return nil
}

// --- Rule: narrowing / partial failure ------------------------------------------

func (w *constraintWorld) thenPagesFullSet() error {
	if !containsFold(w.combined(), "full result set") || !containsFold(w.combined(), "most relevant") {
		return fmt.Errorf("the workflow does not page through the full result set before choosing the most relevant")
	}
	return nil
}

func (w *constraintWorld) thenPresentsMostRelevant() error {
	if !containsFold(w.agent, "most relevant") || !containsFold(w.agent, "rather than every match") {
		return fmt.Errorf("the agent does not present the most relevant governing constraints rather than every match")
	}
	return nil
}

func (w *constraintWorld) thenNotesNarrowed() error {
	if !containsFold(w.agent, "narrowed") || !containsFold(w.agent, "refine") {
		return fmt.Errorf("the agent does not note that the picture was narrowed so the practitioner can refine")
	}
	return nil
}

func (w *constraintWorld) thenSurfacesFailure() error {
	if !containsFold(w.agent, "surface what the failure was") {
		return fmt.Errorf("the agent does not surface what the failure was on a partial read")
	}
	return nil
}

func (w *constraintWorld) thenReturnsSucceeded() error {
	if !containsFold(w.agent, "reads that") || !containsFold(w.agent, "succeeded") {
		return fmt.Errorf("the agent does not return the picture built from the reads that succeeded")
	}
	return nil
}

func (w *constraintWorld) thenNotInvent() error {
	if !containsFold(w.agent, "invent") {
		return fmt.Errorf("the agent does not state it will not invent the missing piece")
	}
	return nil
}

// --- Rule: reachable / degrades --------------------------------------------------

func (w *constraintWorld) givenAgentRegistered() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	// Discoverability: the agent carries the frontmatter (name/description/tools)
	// the host needs, and — like the skill — it is auto-discovered from
	// plugin/agents/, so the manifest declares NO `agents` key.
	for _, field := range []string{"name", "description", "tools"} {
		if !hasFrontmatterField(w.agentRaw, field) {
			return fmt.Errorf("constraint-navigator frontmatter lacks the %q field the host needs to register it", field)
		}
	}
	_, raw, err := ReadOrientationManifest()
	if err != nil {
		return fmt.Errorf("plugin manifest did not load: %w", err)
	}
	if !ManifestDemandsNoSetup(raw) {
		return fmt.Errorf("plugin.json declares a setup-forcing key (e.g. `agents`) — the navigator must be auto-discovered from plugin/agents/, not registered via a manifest key")
	}
	return nil
}

func (w *constraintWorld) whenSkillDelegates() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !containsFold(w.skill, "constraint-navigator") || !containsFold(w.skill, "delegat") {
		return fmt.Errorf("the skill does not delegate traversal to the constraint-navigator agent")
	}
	if !containsFold(w.skill, "well-formed") {
		return fmt.Errorf("the skill does not scope delegation to a well-formed action")
	}
	return nil
}

func (w *constraintWorld) thenRunsOwnContext() error {
	if !containsFold(w.agent, "isolated context") {
		return fmt.Errorf("the agent does not state it runs the traversal in its own isolated context")
	}
	return nil
}

func (w *constraintWorld) thenReturnsOnlyPicture() error {
	if !containsFold(w.agent, "synthesized picture") {
		return fmt.Errorf("the agent output contract never names the synthesized picture it returns")
	}
	if !containsFold(w.agent, "only the synthesized picture") && !containsFold(w.agent, "return **only**") {
		return fmt.Errorf("the agent does not state it returns ONLY the synthesized picture (raw output stays in its context)")
	}
	return nil
}

func (w *constraintWorld) thenWorkflowStandsAsGuidance() error {
	// The skill body carries the traversal steps itself (usable standalone) and
	// states the degradation explicitly.
	if !containsFold(w.skill, "the workflow") {
		return fmt.Errorf("the skill does not carry the workflow steps as consultable guidance")
	}
	for _, step := range []string{"search", "roles", "tree", "domains", "policies", "policy", "me roles", "synthesize"} {
		if !mentionsToken(w.skill, step) {
			return fmt.Errorf("the skill workflow is not self-contained guidance — it omits the %q step", step)
		}
	}
	if !containsFold(w.skill, "follow by hand") && !containsFold(w.skill, "degrades to guidance") {
		return fmt.Errorf("the skill does not state it remains usable as guidance when the agent is absent")
	}
	return nil
}

func (w *constraintWorld) thenNoCLIBroken() error {
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

// --- Rule: drawn from the record, not the tool's opinion -------------------------

func (w *constraintWorld) thenSurfacesPolicy() error {
	if !containsFold(w.combined(), "grants or limits") {
		return fmt.Errorf("the artifacts do not surface the policy that grants or limits the action")
	}
	return nil
}

func (w *constraintWorld) thenConstraintToObserve() error {
	if !containsFold(w.agent, "constraint to observe") || !containsFold(w.agent, "drawn together with any governing domain") {
		return fmt.Errorf("the agent does not present the policy as the constraint to observe, drawn together with any governing domain")
	}
	return nil
}

func (w *constraintWorld) thenNotSeparateDumps() error {
	if !containsFold(w.agent, "concatenation") {
		return fmt.Errorf("the agent does not rule out presenting the picture as a concatenation of separate dumps")
	}
	return nil
}

func (w *constraintWorld) thenSurfacesAbsence() error {
	if !containsFold(w.agent, "nothing constraining") {
		return fmt.Errorf("the agent does not surface that the record shows nothing constraining the action")
	}
	return nil
}

func (w *constraintWorld) thenReportsPlainly() error {
	if !containsFold(w.agent, "plainly") || !containsFold(w.agent, "absence") {
		return fmt.Errorf("the agent does not report the absence plainly, as an absence in the record")
	}
	return nil
}

func (w *constraintWorld) thenNotAssertPermitted() error {
	if !containsFold(w.agent, `not a "you are permitted" verdict`) {
		return fmt.Errorf("the agent does not rule out turning an unconstrained action into a permitted verdict")
	}
	return nil
}

func (w *constraintWorld) thenCharacterizesUnclear() error {
	if !containsFold(w.combined(), "does not clearly answer") {
		return fmt.Errorf("the characterization never names the record-does-not-clearly-answer outcome")
	}
	return nil
}

func (w *constraintWorld) thenSurfacesWhatFound() error {
	if !containsFold(w.combined(), "what it found") {
		return fmt.Errorf("the artifacts do not surface what was found when the record is unclear")
	}
	return nil
}

func (w *constraintWorld) thenNotFabricateRuling() error {
	if !containsFold(w.agent, "never fabricate an authority ruling") {
		return fmt.Errorf("the agent does not rule out fabricating an authority ruling to resolve ambiguity")
	}
	return nil
}

// --- Validation: surface-and-characterize, never rule ----------------------------

func (w *constraintWorld) whenInspectForVerdict() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	w.inspect = "verdict"
	return nil
}

func (w *constraintWorld) whenInspectForWrite() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	w.inspect = "write"
	w.tools, w.hasTools = AgentTools(w.agentRaw)
	return nil
}

// thenContainsNone is shared by two validation scenarios whose When steps
// perform different inspections; it dispatches on the mode the When recorded in
// world state (godog matches step text, and both scenarios say "it will contain
// none").
func (w *constraintWorld) thenContainsNone() error {
	switch w.inspect {
	case "verdict":
		return w.containsNoLocalVerdict()
	case "write":
		return w.containsNoWriteStep()
	default:
		return fmt.Errorf("no inspection was performed before asserting the result is empty")
	}
}

// containsNoLocalVerdict asserts the artifacts nowhere compute a permission
// verdict from local logic: they explicitly disclaim it AND carry none of the
// verdict phrasings a permission ruling would use. The absence check is the
// structural proof of "surface and characterize, never rule" (spec @validation;
// plan ADR-4).
func (w *constraintWorld) containsNoLocalVerdict() error {
	if !containsFold(w.combined(), "permission verdict") || !containsFold(w.combined(), "local rules") {
		return fmt.Errorf("the artifacts do not explicitly disclaim computing a permission verdict from local rules")
	}
	for _, verdict := range []string{
		"you are allowed to",
		"you are not allowed to",
		"you are permitted to",
		"you are forbidden",
		"permission granted",
		"permission denied",
		"you may proceed with",
	} {
		if containsFold(w.combined(), verdict) {
			return fmt.Errorf("the artifacts carry a permission verdict %q — the path must surface and characterize, never rule", verdict)
		}
	}
	return nil
}

// containsNoWriteStep mirrors 064's read-only assertion: the tool grant
// withholds Write/Edit and the artifacts state they carry no write, confirm, or
// gate step.
func (w *constraintWorld) containsNoWriteStep() error {
	if !w.hasTools {
		return fmt.Errorf("the agent declares no tools grant — read-only cannot be asserted structurally")
	}
	for _, forbidden := range []string{"Write", "Edit"} {
		for _, t := range w.tools {
			if strings.EqualFold(t, forbidden) {
				return fmt.Errorf("the agent tool grant includes %q — the path must not be able to write", forbidden)
			}
		}
	}
	if !containsFold(w.skill, "no write, confirm, or gate step") && !containsFold(w.agent, "no confirm") {
		return fmt.Errorf("the artifacts do not state they carry no write, confirm, or gate step")
	}
	return nil
}

func (w *constraintWorld) thenOnlySurfacesFromRecord() error {
	if !containsFold(w.combined(), "surface") || !containsFold(w.combined(), "drawn from the record") {
		return fmt.Errorf("the artifacts do not scope themselves to surfacing the governing governance drawn from the record")
	}
	return nil
}

func (w *constraintWorld) thenNowhereReimplements() error {
	if !containsFold(w.combined(), "reimplement") {
		return fmt.Errorf("the artifacts do not disclaim reimplementing permission rules")
	}
	if !containsFold(w.combined(), "rule on whether the action is allowed") {
		return fmt.Errorf("the artifacts do not disclaim ruling on whether the action is allowed")
	}
	return nil
}

func (w *constraintWorld) thenStatesUnclear() error {
	if !containsFold(w.agent, "state what is unclear and surface what it found") {
		return fmt.Errorf("the agent does not state what is unclear and surface what it found under uncertainty")
	}
	return nil
}

func (w *constraintWorld) thenNowhereAssertsUngrounded() error {
	if !containsFold(w.agent, "cannot ground in the record") {
		return fmt.Errorf("the agent does not rule out asserting a verdict it cannot ground in the record")
	}
	// The same verdict-phrase sweep as the local-verdict inspection: an
	// ungrounded permitted/forbidden assertion would use exactly these shapes.
	w.inspect = "verdict"
	return w.containsNoLocalVerdict()
}

// --- Validation: synthesized, not raw ---------------------------------------------

func (w *constraintWorld) thenDrawnTogether() error {
	if !containsFold(w.agent, "drawn-together") {
		return fmt.Errorf("the output contract is not framed as a drawn-together picture")
	}
	for _, el := range []string{"domains", "policies"} {
		if !mentionsToken(w.agent, el) {
			return fmt.Errorf("the drawn-together picture omits %q", el)
		}
	}
	return nil
}

func (w *constraintWorld) thenNotConcatenation() error {
	if !containsFold(w.agent, "concatenation") || !containsFold(w.agent, "unsynthesized") {
		return fmt.Errorf("the agent does not state the picture is not a concatenation of unsynthesized dumps")
	}
	return nil
}

// --- Rule: every element carries its id --------------------------------------------

func (w *constraintWorld) thenEveryElementCarriesId() error {
	if !containsFold(w.agent, "every element carries the id") {
		return fmt.Errorf("the output contract does not state every element carries the id to read it again")
	}
	if !containsFold(w.agent, "role_") || !containsFold(w.agent, "pol_") {
		return fmt.Errorf("the output contract does not carry the concrete role_/pol_ ids per element")
	}
	return nil
}

func (w *constraintWorld) thenActWithoutReRunning() error {
	if !containsFold(w.agent, "without re-running the search") {
		return fmt.Errorf("the output contract does not let the caller act on any element without re-running the search")
	}
	return nil
}

func (w *constraintWorld) thenOnlyReads() error {
	if !w.hasTools {
		return fmt.Errorf("the agent declares no tools grant")
	}
	// Bash is required to invoke the glassfrog reads; withholding it would be a
	// misconfiguration, not a read-only win.
	hasBash := false
	for _, t := range w.tools {
		if strings.EqualFold(t, "Bash") {
			hasBash = true
		}
	}
	if !hasBash {
		return fmt.Errorf("the agent tool grant omits Bash — it could not invoke glassfrog reads at all")
	}
	if !containsFold(w.combined(), "only read") && !containsFold(w.combined(), "reads only") {
		return fmt.Errorf("the artifacts do not state the path only reads")
	}
	return nil
}

// --- Drift guard (T002) --------------------------------------------------------

func (w *constraintWorld) givenConstraintContent() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	composed, err := ReadConstraintComposedReads()
	if err != nil {
		return fmt.Errorf("could not read the composed-read registry: %w", err)
	}
	if len(composed) == 0 {
		return fmt.Errorf("the composed-read registry is empty — nothing to check against the CLI")
	}
	w.composed = composed
	return nil
}

func (w *constraintWorld) whenCheckedAgainstCLI() error {
	liveTop, err := LiveTopLevelCommands()
	if err != nil {
		return fmt.Errorf("could not extract the CLI's top-level command surface: %w", err)
	}
	if len(liveTop) == 0 {
		return fmt.Errorf("extracted no top-level commands — the surface anchor could not be read")
	}
	liveMe, err := LiveMeSubcommands()
	if err != nil {
		return fmt.Errorf("could not extract the `me` subcommand surface: %w", err)
	}
	if len(liveMe) == 0 {
		return fmt.Errorf("extracted no `me` subcommands — the surface anchor could not be read")
	}
	w.liveTop, w.liveMe = liveTop, liveMe
	w.drift = CheckConstraintDrift(w.composed, w.liveTop, w.liveMe, w.agent)
	return nil
}

func (w *constraintWorld) thenEachExists() error {
	if w.composed == nil || w.liveTop == nil || w.liveMe == nil {
		return fmt.Errorf("the composed reads were not checked against the CLI before asserting existence")
	}
	if len(w.drift) != 0 {
		return fmt.Errorf("a composed read no longer resolves in the shipped CLI:\n  - %s", joinDrift(w.drift))
	}
	top := map[string]bool{}
	for _, r := range w.liveTop {
		top[r] = true
	}
	me := map[string]bool{}
	for _, r := range w.liveMe {
		me[r] = true
	}
	for _, c := range w.composed {
		parts := strings.Fields(c)
		if len(parts) == 2 && parts[0] == "me" {
			if !me[parts[1]] {
				return fmt.Errorf("composed read %q does not exist as a subcommand of `me`", c)
			}
			continue
		}
		if !top[c] {
			return fmt.Errorf("composed read %q does not exist as a top-level CLI command", c)
		}
	}
	return nil
}

func (w *constraintWorld) thenNamesNoLackingRead() error {
	// The same drift result proves the path names no read the CLI does not
	// expose.
	if w.composed == nil || w.liveTop == nil {
		return fmt.Errorf("the composed reads were not checked against the CLI before asserting no invented read")
	}
	if len(w.drift) != 0 {
		return fmt.Errorf("the path names a read the CLI does not expose:\n  - %s", joinDrift(w.drift))
	}
	return nil
}
