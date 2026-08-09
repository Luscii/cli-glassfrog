package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestGovernanceNavigationFeatures runs the executable acceptance for the
// Governance Navigation Path (064). Like the sibling build-side suites
// (021/022/036/062/063) its Paths name ONLY this spec's feature file and it runs
// with the ~@wip filter, so only the scenarios implemented so far execute.
//
// The deliverable is a pair of declarative Claude-plugin artifacts — a thin
// `governance-navigation` skill that delegates to a read-only
// `governance-navigator` agent — plus a single-sourced composed-read registry and
// a best-effort drift guard. The artifacts carry no runtime Go path of their own,
// so the executable scenarios assert against their content: the skill's when +
// workflow + delegation, the agent's read-only grant + isolated-execution +
// synthesized-picture output contract + defensive traversal, and the drift guard
// that keeps the composed read leaves truthful to the shipped CLI (the drift
// scenario is implemented in T002).
func TestGovernanceNavigationFeatures(t *testing.T) {
	w := &navigationWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.register(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unequipped-agent-operators/governance-navigation-path.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: governance-navigation-path feature scenarios failed")
	}
}

// navigationWorld is the per-scenario state: the loaded skill + agent content, the
// single-sourced composed reads, the CLI's live top-level surface, the parsed
// agent tool grant, and the drift findings.
type navigationWorld struct {
	// skill/agent hold the artifacts with whitespace collapsed to single spaces, so
	// phrase assertions are resilient to markdown line-wrapping (a hard-wrapped
	// "nothing relevant\nwas found" still matches "nothing relevant was found").
	skill string
	agent string
	// skillRaw/agentRaw keep the verbatim content for structural checks that depend
	// on line breaks (frontmatter delimiters, the tools list).
	skillRaw string
	agentRaw string
	composed []string
	live     []string
	tools    []string
	hasTools bool
	drift    []string
}

func (w *navigationWorld) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = navigationWorld{}
		return ctx, nil
	})

	// Rule: Work a tension through a guided path -----------------------------
	sc.Step(`^a practitioner had voiced a free-form concern with no role in hand$`, w.givenLoaded)
	sc.Step(`^the governance-navigator traverses the concern$`, w.whenLoaded)
	sc.Step(`^it will search the governance record for what the concern touches$`, w.thenSearchesRecord)
	sc.Step(`^it will return the relevant roles and who fills them$`, w.thenReturnsRolesAndFillers)
	sc.Step(`^each role and filler will carry the id needed to read it again$`, w.thenRoleFillerCarryId)

	sc.Step(`^the plugin was present with the governance-navigator agent registered$`, w.givenAgentRegistered)
	sc.Step(`^the governance-navigation skill delegates a concern for traversal$`, w.whenSkillDelegates)
	sc.Step(`^the navigator will run the traversal in its own context$`, w.thenRunsOwnContext)
	sc.Step(`^it will return only the synthesized picture to the caller$`, w.thenReturnsOnlyPicture)

	sc.Step(`^the plugin was present but the governance-navigator agent was absent or unregistered$`, w.givenAgentAbsent)
	sc.Step(`^the governance-navigation skill is consulted for a concern$`, w.whenSkillConsulted)
	sc.Step(`^its workflow will remain readable as guidance the caller can follow by hand$`, w.thenWorkflowStandsAsGuidance)
	sc.Step(`^no command in the CLI will be broken by the agent's absence$`, w.thenNoCLIBroken)

	sc.Step(`^a concern so broad that the search matched many roles, domains, and policies across several pages$`, w.givenLoaded)
	sc.Step(`^the governance-navigator assembles the picture$`, w.whenLoaded)
	sc.Step(`^it will page through the full result set before choosing what is most relevant$`, w.thenPagesFullSet)
	sc.Step(`^it will present the most relevant results rather than every match$`, w.thenPresentsMostRelevant)
	sc.Step(`^it will note that the picture was narrowed so the practitioner can refine$`, w.thenNotesNarrowed)

	sc.Step(`^a concern for which the search returned no results$`, w.givenLoaded)
	sc.Step(`^the governance-navigator completes the traversal$`, w.whenLoaded)
	sc.Step(`^it will report that nothing relevant was found$`, w.thenReportsNothingFound)
	sc.Step(`^it will suggest refining the concern$`, w.thenSuggestsRefining)
	sc.Step(`^it will fabricate no roles or governance$`, w.thenFabricatesNothing)

	sc.Step(`^a traversal in which one read failed while the others succeeded$`, w.givenLoaded)
	sc.Step(`^it will surface what the failure was$`, w.thenSurfacesFailure)
	sc.Step(`^it will return the picture built from the reads that succeeded$`, w.thenReturnsSucceeded)
	sc.Step(`^it will not invent the missing piece$`, w.thenNotInvent)

	sc.Step(`^the picture the governance-navigator returned$`, w.givenLoaded)
	sc.Step(`^it is compared against the raw command output$`, w.whenLoaded)
	sc.Step(`^it will be a drawn-together picture of roles, fillers, domains, and policies$`, w.thenDrawnTogether)
	sc.Step(`^it will not be a concatenation of unsynthesized dumps$`, w.thenNotConcatenation)

	// Rule: See the roles, fillers, and governing domains and policies -------
	sc.Step(`^a role the governance-navigator had identified as relevant to the concern$`, w.givenLoaded)
	sc.Step(`^it traverses that role's governance$`, w.whenLoaded)
	sc.Step(`^it will draw in the domains the role controls that bear on the concern$`, w.thenDrawsInDomains)
	sc.Step(`^it will draw in the policies on the role's interior that bear on the concern$`, w.thenDrawsInPolicies)
	sc.Step(`^it will present them as part of one picture$`, w.thenPartOfOnePicture)

	sc.Step(`^the concern touched a role that is a circle$`, w.givenLoaded)
	sc.Step(`^the governance-navigator judges the sub-roles relevant$`, w.whenLoaded)
	sc.Step(`^it will follow into those sub-roles and their fillers as far as the concern warrants$`, w.thenFollowsSubRoles)
	sc.Step(`^it will stop short of walking the whole tree$`, w.thenStopsShortOfTree)

	sc.Step(`^a concern phrased as whether the practitioner may take an action$`, w.givenLoaded)
	sc.Step(`^the governance-navigator surfaces the domains and policies that govern it$`, w.whenLoaded)
	sc.Step(`^it will present that governing governance$`, w.thenPresentsGoverning)
	sc.Step(`^it will defer the authority verdict to the Constraint Discovery Path$`, w.thenDefersVerdict)
	sc.Step(`^it will not rule on whether the action is permitted$`, w.thenNotRulePermitted)

	sc.Step(`^the governance-navigator's treatment of domains and policies$`, w.givenLoaded)
	sc.Step(`^it is inspected for an authority or permission verdict$`, w.whenLoaded)
	sc.Step(`^it will only surface the governing governance$`, w.thenOnlySurfaces)
	sc.Step(`^it will nowhere rule on whether an action is allowed$`, w.thenNowhereRules)

	// Rule: Move from understanding to action with every element carrying id -
	sc.Step(`^a picture the governance-navigator had returned for a concern$`, w.givenLoaded)
	sc.Step(`^the caller inspects each role, filler, domain, and policy in it$`, w.whenLoaded)
	sc.Step(`^every element will carry the id needed to read it again$`, w.thenEveryElementCarriesId)
	sc.Step(`^the caller will be able to act on any element without re-running the search$`, w.thenActWithoutReRunning)

	sc.Step(`^the governance-navigation skill and agent content$`, w.givenLoaded)
	sc.Step(`^it is inspected for any write, confirm, or gate step$`, w.whenInspectForWrite)
	sc.Step(`^it will contain none$`, w.thenContainsNoWriteStep)
	sc.Step(`^the path will only read$`, w.thenOnlyReads)

	// Drift guard (T002) -----------------------------------------------------
	sc.Step(`^the produced navigation-path content$`, w.givenNavigationContent)
	sc.Step(`^every command and read it composes is checked against the shipped CLI$`, w.whenCheckedAgainstCLI)
	sc.Step(`^each one will exist$`, w.thenEachExists)
	sc.Step(`^the path will name no read the CLI does not expose$`, w.thenNamesNoLackingRead)
}

// --- Loading -----------------------------------------------------------------

func (w *navigationWorld) ensureSkillLoaded() error {
	if w.skillRaw == "" {
		skill, err := ReadNavigationSkill()
		if err != nil {
			return fmt.Errorf("governance-navigation skill did not load: %w", err)
		}
		w.skillRaw, w.skill = skill, normalizeWS(skill)
	}
	return nil
}

func (w *navigationWorld) ensureAgentLoaded() error {
	if w.agentRaw == "" {
		agent, err := ReadNavigatorAgent()
		if err != nil {
			return fmt.Errorf("governance-navigator agent did not load: %w", err)
		}
		w.agentRaw, w.agent = agent, normalizeWS(agent)
	}
	return nil
}

// ensureLoaded loads both artifacts — used by every scenario except the
// missing-navigator degradation path, which loads only the skill (see
// givenAgentAbsent) so it faithfully models the agent being absent.
func (w *navigationWorld) ensureLoaded() error {
	if err := w.ensureSkillLoaded(); err != nil {
		return err
	}
	return w.ensureAgentLoaded()
}

func (w *navigationWorld) givenLoaded() error { return w.ensureLoaded() }
func (w *navigationWorld) whenLoaded() error  { return w.ensureLoaded() }

// combined returns the (normalized) skill and agent content together — most
// behavioral scenarios inspect "the navigator's behavior" as described across both
// artifacts.
func (w *navigationWorld) combined() string { return w.skill + " " + w.agent }

// normalizeWS collapses every run of whitespace to a single space, so a phrase the
// markdown wrapped across a line break still reads as one contiguous string.
func normalizeWS(s string) string { return strings.Join(strings.Fields(s), " ") }

// --- Rule: guided path -------------------------------------------------------

func (w *navigationWorld) thenSearchesRecord() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !mentionsToken(w.combined(), "search") || !containsFold(w.combined(), "discover") {
		return fmt.Errorf("the workflow does not describe searching the record to discover what the concern touches")
	}
	return nil
}

func (w *navigationWorld) thenReturnsRolesAndFillers() error {
	if !mentionsToken(w.combined(), "roles") || !mentionsToken(w.combined(), "fillers") {
		return fmt.Errorf("the output contract does not return both the relevant roles and who fills them")
	}
	return nil
}

func (w *navigationWorld) thenRoleFillerCarryId() error {
	c := w.agent
	if !containsFold(c, "role_") || (!containsFold(c, "per_") && !containsFold(c, "agt_")) {
		return fmt.Errorf("the output contract does not give each role its role_ id and each filler its actor id")
	}
	if !containsFold(c, "id") {
		return fmt.Errorf("the output contract does not state each element carries the id to read it again")
	}
	return nil
}

// --- Rule: reachable / degrades ---------------------------------------------

func (w *navigationWorld) givenAgentRegistered() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	// Discoverability: the agent carries the frontmatter (name/description/tools)
	// the host needs, and — like the skill — it is auto-discovered from
	// plugin/agents/, so the manifest declares NO `agents` key.
	for _, field := range []string{"name", "description", "tools"} {
		if !hasFrontmatterField(w.agentRaw, field) {
			return fmt.Errorf("governance-navigator frontmatter lacks the %q field the host needs to register it", field)
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

func (w *navigationWorld) whenSkillDelegates() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	if !containsFold(w.skill, "governance-navigator") || !containsFold(w.skill, "delegat") {
		return fmt.Errorf("the skill does not delegate traversal to the governance-navigator agent")
	}
	return nil
}

func (w *navigationWorld) thenRunsOwnContext() error {
	if !containsFold(w.agent, "isolated context") {
		return fmt.Errorf("the agent does not state it runs the traversal in its own isolated context")
	}
	return nil
}

func (w *navigationWorld) thenReturnsOnlyPicture() error {
	if !containsFold(w.agent, "synthesized picture") {
		return fmt.Errorf("the agent output contract never names the synthesized picture it returns")
	}
	if !containsFold(w.agent, "only the synthesized picture") && !containsFold(w.agent, "return **only**") {
		return fmt.Errorf("the agent does not state it returns ONLY the synthesized picture (raw output stays in its context)")
	}
	return nil
}

// givenAgentAbsent models the navigator agent being absent/unregistered: it loads
// ONLY the skill. The degradation scenario asserts the skill's workflow stands
// alone as guidance and no CLI command breaks — neither needs the agent, and
// loading it would both contradict the premise and fail outright if the agent
// file were genuinely gone.
func (w *navigationWorld) givenAgentAbsent() error { return w.ensureSkillLoaded() }

func (w *navigationWorld) whenSkillConsulted() error { return w.ensureSkillLoaded() }

func (w *navigationWorld) thenWorkflowStandsAsGuidance() error {
	// The skill body carries the traversal steps itself (usable standalone) and
	// states the degradation explicitly.
	if !containsFold(w.skill, "the workflow") {
		return fmt.Errorf("the skill does not carry the workflow steps as consultable guidance")
	}
	for _, step := range []string{"search", "roles", "fillers", "domains", "policies", "synthesize"} {
		if !mentionsToken(w.skill, step) {
			return fmt.Errorf("the skill workflow is not self-contained guidance — it omits the %q step", step)
		}
	}
	if !containsFold(w.skill, "follow by hand") && !containsFold(w.skill, "degrades to guidance") {
		return fmt.Errorf("the skill does not state it remains usable as guidance when the agent is absent")
	}
	return nil
}

func (w *navigationWorld) thenNoCLIBroken() error {
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

// --- Rule: narrowing / empty / partial / synthesis --------------------------

func (w *navigationWorld) thenPagesFullSet() error {
	if !containsFold(w.combined(), "full result set") || !containsFold(w.combined(), "most relevant") {
		return fmt.Errorf("the workflow does not page through the full result set before choosing the most relevant")
	}
	return nil
}

func (w *navigationWorld) thenPresentsMostRelevant() error {
	if !containsFold(w.agent, "most relevant") || !containsFold(w.agent, "rather than every match") {
		return fmt.Errorf("the agent does not present the most relevant results rather than every match")
	}
	return nil
}

func (w *navigationWorld) thenNotesNarrowed() error {
	if !containsFold(w.agent, "narrowed") || !containsFold(w.agent, "refine") {
		return fmt.Errorf("the agent does not note that the picture was narrowed so the practitioner can refine")
	}
	return nil
}

func (w *navigationWorld) thenReportsNothingFound() error {
	if !containsFold(w.agent, "nothing relevant was found") {
		return fmt.Errorf("the agent does not report that nothing relevant was found")
	}
	return nil
}

func (w *navigationWorld) thenSuggestsRefining() error {
	if !containsFold(w.agent, "refin") {
		return fmt.Errorf("the agent does not suggest refining the concern")
	}
	return nil
}

func (w *navigationWorld) thenFabricatesNothing() error {
	if !containsFold(w.agent, "fabricate no") {
		return fmt.Errorf("the agent does not state it fabricates no roles or governance on an empty search")
	}
	return nil
}

func (w *navigationWorld) thenSurfacesFailure() error {
	if !containsFold(w.agent, "surface what the failure was") {
		return fmt.Errorf("the agent does not surface what the failure was on a partial read")
	}
	return nil
}

func (w *navigationWorld) thenReturnsSucceeded() error {
	if !containsFold(w.agent, "reads that") || !containsFold(w.agent, "succeeded") {
		return fmt.Errorf("the agent does not return the picture built from the reads that succeeded")
	}
	return nil
}

func (w *navigationWorld) thenNotInvent() error {
	if !containsFold(w.agent, "invent") {
		return fmt.Errorf("the agent does not state it will not invent the missing piece")
	}
	return nil
}

func (w *navigationWorld) thenDrawnTogether() error {
	if !containsFold(w.agent, "drawn-together") {
		return fmt.Errorf("the output contract is not framed as a drawn-together picture")
	}
	for _, el := range []string{"roles", "fillers", "domains", "policies"} {
		if !mentionsToken(w.agent, el) {
			return fmt.Errorf("the drawn-together picture omits %q", el)
		}
	}
	return nil
}

func (w *navigationWorld) thenNotConcatenation() error {
	if !containsFold(w.agent, "concatenation") {
		return fmt.Errorf("the agent does not state the picture is not a concatenation of unsynthesized dumps")
	}
	return nil
}

// --- Rule: domains and policies drawn together ------------------------------

func (w *navigationWorld) thenDrawsInDomains() error {
	if !mentionsToken(w.combined(), "domains") || !containsFold(w.combined(), "control") {
		return fmt.Errorf("the workflow does not draw in the domains the role controls")
	}
	return nil
}

func (w *navigationWorld) thenDrawsInPolicies() error {
	if !mentionsToken(w.combined(), "policies") || !containsFold(w.combined(), "interior") {
		return fmt.Errorf("the workflow does not draw in the policies on the role's interior")
	}
	return nil
}

func (w *navigationWorld) thenPartOfOnePicture() error {
	if !containsFold(w.combined(), "one picture") {
		return fmt.Errorf("the workflow does not present the domains and policies as part of one picture")
	}
	return nil
}

func (w *navigationWorld) thenFollowsSubRoles() error {
	if !containsFold(w.combined(), "sub-role") || !containsFold(w.combined(), "as far as the concern warrants") {
		return fmt.Errorf("the workflow does not follow into sub-roles as far as the concern warrants")
	}
	return nil
}

func (w *navigationWorld) thenStopsShortOfTree() error {
	if !containsFold(w.combined(), "whole tree") || !containsFold(w.combined(), "stop short") {
		return fmt.Errorf("the workflow does not stop short of walking the whole tree")
	}
	return nil
}

// --- Rule: authority question / surfacing not judging -----------------------

func (w *navigationWorld) thenPresentsGoverning() error {
	if !containsFold(w.combined(), "surface") || !mentionsToken(w.combined(), "domains") || !mentionsToken(w.combined(), "policies") {
		return fmt.Errorf("the workflow does not present the governing domains and policies")
	}
	return nil
}

func (w *navigationWorld) thenDefersVerdict() error {
	if !containsFold(w.combined(), "Constraint Discovery Path") {
		return fmt.Errorf("the workflow does not defer the authority verdict to the Constraint Discovery Path by its in-plugin name")
	}
	if !containsFold(w.combined(), "defer") {
		return fmt.Errorf("the workflow surfaces governance but never states it defers the verdict")
	}
	return nil
}

func (w *navigationWorld) thenNotRulePermitted() error {
	if !containsFold(w.combined(), "rule on whether the action is permitted") {
		return fmt.Errorf("the workflow does not state it will not rule on whether the action is permitted")
	}
	return nil
}

func (w *navigationWorld) thenOnlySurfaces() error {
	if !containsFold(w.combined(), "surface") || !containsFold(w.combined(), "governing") {
		return fmt.Errorf("the artifacts do not state they only surface the governing governance")
	}
	return nil
}

// thenNowhereRules asserts the artifacts nowhere carry an authority/permission
// verdict: they explicitly defer to 065 AND carry none of the verdict phrasings a
// permission ruling would use. The absence check is the structural proof of
// "surfacing, not judging" (spec @validation; plan ADR-5).
func (w *navigationWorld) thenNowhereRules() error {
	if !containsFold(w.combined(), "do not rule") && !containsFold(w.combined(), "never rule") {
		return fmt.Errorf("the artifacts do not explicitly disclaim ruling on authority")
	}
	if !containsFold(w.combined(), "Constraint Discovery Path") {
		return fmt.Errorf("the artifacts do not defer the authority verdict to the Constraint Discovery Path")
	}
	// No permission verdict may appear: a ruling would read as "you are (not)
	// allowed / permitted to …" or "permission granted/denied".
	for _, verdict := range []string{
		"you are allowed to",
		"you are not allowed to",
		"you are permitted to",
		"permission granted",
		"permission denied",
		"you may proceed with",
	} {
		if containsFold(w.combined(), verdict) {
			return fmt.Errorf("the artifacts carry an authority verdict %q — the path must surface, not judge", verdict)
		}
	}
	return nil
}

// --- Rule: every element carries its id -------------------------------------

func (w *navigationWorld) thenEveryElementCarriesId() error {
	c := w.agent
	if !containsFold(c, "every element") || !containsFold(c, "id") {
		return fmt.Errorf("the output contract does not state every element carries the id to read it again")
	}
	if !containsFold(c, "role_") || (!containsFold(c, "per_") && !containsFold(c, "agt_")) {
		return fmt.Errorf("the output contract does not carry the concrete role_/actor ids per element")
	}
	return nil
}

func (w *navigationWorld) thenActWithoutReRunning() error {
	if !containsFold(w.agent, "without re-running the") {
		return fmt.Errorf("the output contract does not let the caller act on any element without re-running the search")
	}
	return nil
}

// --- Rule: read-only, never writes ------------------------------------------

func (w *navigationWorld) whenInspectForWrite() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	w.tools, w.hasTools = AgentTools(w.agentRaw)
	return nil
}

func (w *navigationWorld) thenContainsNoWriteStep() error {
	if !w.hasTools {
		return fmt.Errorf("the agent declares no tools grant — read-only cannot be asserted structurally")
	}
	// The read-only tool grant withholds Write/Edit (no workspace mutation).
	for _, forbidden := range []string{"Write", "Edit"} {
		for _, t := range w.tools {
			if strings.EqualFold(t, forbidden) {
				return fmt.Errorf("the agent tool grant includes %q — the path must not be able to write", forbidden)
			}
		}
	}
	// The artifacts carry no write/confirm/gate STEP: they say so explicitly.
	if !containsFold(w.skill, "no write, confirm, or gate step") && !containsFold(w.agent, "no confirm") {
		return fmt.Errorf("the artifacts do not state they carry no write, confirm, or gate step")
	}
	return nil
}

func (w *navigationWorld) thenOnlyReads() error {
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

// --- Drift guard (T002) -----------------------------------------------------

func (w *navigationWorld) givenNavigationContent() error {
	if err := w.ensureLoaded(); err != nil {
		return err
	}
	composed, err := ReadComposedReads()
	if err != nil {
		return fmt.Errorf("could not read the composed-read registry: %w", err)
	}
	if len(composed) == 0 {
		return fmt.Errorf("the composed-read registry is empty — nothing to check against the CLI")
	}
	w.composed = composed
	return nil
}

func (w *navigationWorld) whenCheckedAgainstCLI() error {
	live, err := LiveTopLevelCommands()
	if err != nil {
		return fmt.Errorf("could not extract the CLI's top-level command surface: %w", err)
	}
	if len(live) == 0 {
		return fmt.Errorf("extracted no top-level commands — the surface anchor could not be read")
	}
	w.live = live
	w.drift = CheckNavigationDrift(w.composed, w.live, w.agent)
	return nil
}

func (w *navigationWorld) thenEachExists() error {
	if len(w.drift) != 0 {
		return fmt.Errorf("a composed read no longer resolves in the shipped CLI:\n  - %s", joinDrift(w.drift))
	}
	live := map[string]bool{}
	for _, r := range w.live {
		live[r] = true
	}
	for _, c := range w.composed {
		if !live[c] {
			return fmt.Errorf("composed read %q does not exist as a top-level CLI command", c)
		}
	}
	return nil
}

func (w *navigationWorld) thenNamesNoLackingRead() error {
	// The same drift result proves the path names no read the CLI does not expose.
	if len(w.drift) != 0 {
		return fmt.Errorf("the path names a read the CLI does not expose:\n  - %s", joinDrift(w.drift))
	}
	return nil
}
