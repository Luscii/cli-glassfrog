package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/Luscii/cli-glassfrog/internal/render"
	"github.com/cucumber/godog"
)

// TestPostCreateValidityReadFeatures runs the executable acceptance for the
// Post-Create Validity Read (074): the create's read-back stage, its four
// verdict states across the output formats, and the id-preserving failure
// isolation — driven through the shared proposalSeam over the scripted
// two-exchange transport so every scenario runs offline. Its Paths name ONLY
// this spec's feature file, so the suite reports its own scenario count and
// cannot disturb another suite. The 4 @validation scenarios stay @wip (held for
// the validate skill) and are skipped by the ~@wip filter.
//
// ~@deprecated is the second filter, added by Invalid-Create Outcome (078): four
// scenarios here describe the intermediate state in which an accepted-but-invalid
// create still exited 0, and 078 makes that a failure with exit 8. Their
// @deprecate tag was inert until now — nothing in the repo read it — so the tag
// finally carries the exclusion its documented lifecycle already promised, in the
// same commit as the behaviour that supersedes them. The superseding scenarios
// live in invalid-create-outcome.feature.
func TestPostCreateValidityReadFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializePostCreateValidityScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/success-reported-for-a-dead-proposal/post-create-validity-read.feature"},
			Tags:     "~@wip && ~@deprecated",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: post-create-validity-read feature scenarios failed")
	}
}

// readBackBodyWith assembles a read-back 200 {data: Proposal} body from the
// scenario-varying parts: the server-given status, the verdict fields (a
// `"valid":…,"validation_alerts":…,` fragment, or empty for a body carrying no
// verdict at all), and the available transitions.
func readBackBodyWith(status, verdictFields, transitions string) string {
	return `{"data":{"id":"prp_0123","type":"proposal","status":"` + status + `",` +
		`"tension_id":"ten_0123","circle_id":"role_0123","proposer_id":"per_0123",` +
		`"changes":[{"id":"chg_1","type":"CreateRole","name":"Scribe"}],` +
		`"response_summary":{"total":0,"no_objection":0,"bring_to_meeting":0},` +
		`"expected_response_count":0,"received_response_count":0,` +
		verdictFields +
		`"available_transitions":` + transitions + `,` +
		`"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}}`
}

// postCreateValidityWorld is the per-scenario state: the connection context, the
// scripted create + read-back exchanges the Givens configure, the injected user
// template files, and the captured outcome/exit-code/streams of the When run.
// Everything is injected — no step touches the real network, env, home, pipe,
// or filesystem. Parameterised expectations (the alert message, the conflicted
// status) are stashed by the Given and read back by the Then, never re-derived.
type postCreateValidityWorld struct {
	ctx        apiclient.ConnectionContext
	createStep proposalSeqStep
	readStep   proposalSeqStep
	transport  *proposalSeqTransport
	tmplFiles  map[string]string
	secret     string

	wantAlertMessage string // the server alert message a Given stashed
	wantStatus       string // the server-given status a Given stashed
	wantValidity     string // the validity token the read-back Given implies ("valid" / "not valid")
	wantAlertCount   int    // how many alerts that Given attached

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string

	siblingOutputs map[string]string // command → stdout, for the sibling-commands scenario
	siblingWant    string            // the shared proposal rendering the siblings must match
}

func initializePostCreateValidityScenario(sc *godog.ScenarioContext) {
	w := &postCreateValidityWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = postCreateValidityWorld{
			createStep: proposalSeqStep{status: 201, body: proposalCreatedBody},
			// The default read-back carries no validity field; per-scenario
			// Givens override it with the verdict the scenario states.
			readStep: proposalSeqStep{status: 200, body: proposalCreatedBody},
			secret:   meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^the tension "([^"]*)" exists$`, w.tensionExists)
	sc.Step(`^the created proposal reads back as valid with no validation alerts$`, w.readsBackValidNoAlerts)
	sc.Step(`^the created proposal reads back as not valid with the alert "([^"]*)" and no available transitions$`, w.readsBackNotValidWithAlertNoTransitions)
	sc.Step(`^the created proposal reads back as not valid with one validation alert$`, w.readsBackNotValidOneAlert)
	sc.Step(`^the created proposal reads back carrying no validity field$`, w.readsBackNoValidityField)
	sc.Step(`^the created proposal reads back as valid with no available transitions$`, w.readsBackValidNoTransitions)
	sc.Step(`^the created proposal reads back with status "([^"]*)" and as valid$`, w.readsBackWithStatusAndValid)
	sc.Step(`^the created proposal reads back as valid with one alert of severity "([^"]*)"$`, w.readsBackValidWithAlertSeverity)
	sc.Step(`^the proposals endpoint rejects the create$`, w.createRejected)
	sc.Step(`^the create succeeds but the read of the created proposal cannot reach the server$`, w.readBackUnreachable)
	sc.Step(`^the create succeeds but the read of the created proposal is rate-limited after its retries$`, w.readBackRateLimited)
	sc.Step(`^the create answers with a success body carrying no "([^"]*)" id$`, w.createBodyCarriesNoID)
	sc.Step(`^a user template referencing only the proposal fields the create rendered before the verdict existed$`, w.preVerdictUserTemplate)

	// --- Whens ---
	sc.Step(`^an agent runs "glassfrog (.+)"$`, w.runCommand)
	sc.Step(`^an agent creates a proposal with that template selected$`, w.runCreateWithTemplate)
	sc.Step(`^an agent reads, advances, and withdraws a proposal whose read carries a validity field$`, w.runSiblingCommands)

	// --- Thens ---
	sc.Step(`^the created proposal will be printed with its "([^"]*)" id and "([^"]*)" status$`, w.createdPrintedWithIDAndStatus)
	sc.Step(`^the created proposal will be printed with its "([^"]*)" id$`, w.createdPrintedWithID)
	sc.Step(`^the result will report the validity as "([^"]*)"$`, w.validityReportedAs)
	sc.Step(`^the create will have been followed by exactly one read of the created proposal$`, w.exactlyOneReadFollowed)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the result will carry the server's alert message with its severity and path$`, w.alertCarriedWithSeverityAndPath)
	sc.Step(`^the result will carry the alert with its severity, path, and message$`, w.alertCarriedWithSeverityAndPath)
	sc.Step(`^the result will report that no transitions are available$`, w.noTransitionsReported)
	sc.Step(`^stderr will report that the create failed and name the HTTP status$`, w.stderrNamesCreateFailure)
	sc.Step(`^no read of any proposal will be attempted$`, w.noProposalReadAttempted)
	sc.Step(`^the result will report that the server stated no verdict on the draft$`, w.noVerdictStatedReported)
	sc.Step(`^the result will describe the draft as neither valid nor not valid$`, w.neitherValidNorNotValid)
	sc.Step(`^neither fact will be restated as the other$`, w.factsStayDistinct)
	sc.Step(`^the result will report the status "([^"]*)" as the server gave it$`, w.statusReportedAsGiven)
	sc.Step(`^neither will be adjusted to agree with the other$`, w.statusAndVerdictUnadjusted)
	sc.Step(`^the alert's presence will not be reported as an unfavourable verdict$`, w.alertNotUnfavourable)
	sc.Step(`^the structured result will contain the created proposal's "([^"]*)" id$`, w.structuredContainsID)
	sc.Step(`^the structured result will carry "([^"]*)", "([^"]*)", and "([^"]*)"$`, w.structuredCarriesKeys)
	sc.Step(`^the advisory will be rendered in the selected machine format, carrying "([^"]*)" as (true|false)$`, w.machineAdvisoryCarries)
	sc.Step(`^the advisory will carry the reason and the remedy naming "([^"]*)"$`, w.advisoryCarriesReasonAndRemedy)
	sc.Step(`^no part of the four verdict states will require reading prose to identify$`, w.noProseNeeded)
	sc.Step(`^the compact line will carry the created "([^"]*)" id, its status, and the change count$`, w.compactCarriesIDStatusCount)
	sc.Step(`^it will carry the validity token and the alert count$`, w.compactCarriesValidityAndAlertCount)
	sc.Step(`^every field path in the template will still resolve$`, w.templateFieldsResolve)
	sc.Step(`^the template's output will carry the read-back's values through those same paths$`, w.templateOutputCarriesReadBack)
	sc.Step(`^none of those results will render a validity, alert, or verdict-source line$`, w.siblingsRenderNoVerdict)
	sc.Step(`^their output will be unchanged from before the create gained its verdict$`, w.siblingsOutputUnchanged)
	sc.Step(`^the result will report that the verdict could not be obtained and name the cause$`, w.verdictUnobtainableWithCause)
	sc.Step(`^the result will report that the verdict could not be obtained because the request budget was exhausted$`, w.verdictUnobtainableBudget)
	sc.Step(`^the result will report that the created proposal's id could not be determined$`, w.idUndeterminableReported)
	sc.Step(`^the create response will still be reported$`, w.createResponseStillReported)
}

// --- Given implementations ---

func (w *postCreateValidityWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *postCreateValidityWorld) tensionExists(_ string) error { return nil }

func (w *postCreateValidityWorld) readsBackValidNoAlerts() error {
	w.readStep = proposalSeqStep{status: 200, body: readBackBodyWith("draft", `"valid":true,"validation_alerts":[],`, `["propose","withdraw"]`)}
	return nil
}

func (w *postCreateValidityWorld) readsBackNotValidWithAlertNoTransitions(message string) error {
	w.wantAlertMessage = message
	w.wantValidity, w.wantAlertCount = "not valid", 1
	alerts := fmt.Sprintf(`"valid":false,"validation_alerts":[{"severity":"error","path":"name","message":%q}],`, message)
	w.readStep = proposalSeqStep{status: 200, body: readBackBodyWith("draft", alerts, `[]`)}
	return nil
}

func (w *postCreateValidityWorld) readsBackNotValidOneAlert() error {
	return w.readsBackNotValidWithAlertNoTransitions("Can't update the Cloud Foundations role during this meeting.")
}

func (w *postCreateValidityWorld) readsBackNoValidityField() error {
	w.readStep = proposalSeqStep{status: 200, body: readBackBodyWith("draft", ``, `["propose"]`)}
	return nil
}

func (w *postCreateValidityWorld) readsBackValidNoTransitions() error {
	w.readStep = proposalSeqStep{status: 200, body: readBackBodyWith("draft", `"valid":true,"validation_alerts":[],`, `[]`)}
	return nil
}

func (w *postCreateValidityWorld) readsBackWithStatusAndValid(status string) error {
	w.wantStatus = status
	w.readStep = proposalSeqStep{status: 200, body: readBackBodyWith(status, `"valid":true,"validation_alerts":[],`, `["propose"]`)}
	return nil
}

func (w *postCreateValidityWorld) readsBackValidWithAlertSeverity(severity string) error {
	w.wantAlertMessage = "advisory only"
	w.wantValidity, w.wantAlertCount = "valid", 1
	alerts := fmt.Sprintf(`"valid":true,"validation_alerts":[{"severity":%q,"path":"changes[0]","message":"advisory only"}],`, severity)
	w.readStep = proposalSeqStep{status: 200, body: readBackBodyWith("draft", alerts, `["propose"]`)}
	return nil
}

func (w *postCreateValidityWorld) createRejected() error {
	w.createStep = proposalSeqStep{status: 422, body: `{"detail":"the change set was rejected"}`}
	return nil
}

func (w *postCreateValidityWorld) readBackUnreachable() error {
	w.readStep = proposalSeqStep{netErr: errors.New("network unreachable")}
	return nil
}

func (w *postCreateValidityWorld) readBackRateLimited() error {
	w.readStep = proposalSeqStep{status: 429, body: `{"detail":"rate limited"}`}
	return nil
}

func (w *postCreateValidityWorld) createBodyCarriesNoID(_ string) error {
	w.createStep = proposalSeqStep{status: 201, body: `{"data":{"type":"proposal","status":"draft"}}`}
	return nil
}

// preVerdictUserTemplate builds a template over ONLY pre-074 field paths. It
// includes .Proposal.AvailableTransitions deliberately: that is the field on
// which the create response and the read-back DISAGREE, so the Then step can
// tell which document supplied the values. A projection limited to id, status,
// and the change count renders identically either way, which would make the
// assertion inert (validate.md Round 2, F-1 — the same defect the unit-level
// pinning test carried).
func (w *postCreateValidityWorld) preVerdictUserTemplate() error {
	w.tmplFiles = map[string]string{
		"pre074.tmpl": "{{.Proposal.ID}} {{.Proposal.Status}} {{len .Proposal.Changes}} {{len .Proposal.AvailableTransitions}}",
	}
	return nil
}

// --- When implementations ---

// runCommand parses the invocation with the suite's single-quote-aware splitter
// (proposal_creation_bdd_test) and dispatches it through a real root with the
// `proposal` group attached over a fake seam whose transport scripts the create
// and read-back exchanges. It asserts the secret token never leaks into output.
func (w *postCreateValidityWorld) runCommand(invocation string) error {
	args := splitArgsPOSIX(strings.ReplaceAll(invocation, `\"`, `"`))
	root := NewRootCommand()
	w.transport = &proposalSeqTransport{steps: []proposalSeqStep{w.createStep, w.readStep}}
	seam := &fakeProposalSeam{
		fakeMeSeam: &fakeMeSeam{ctx: w.ctx, transport: w.transport, tmplFiles: w.tmplFiles},
	}
	MustRegister(root, newProposalCommand(seam))

	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	w.outcome, _ = Run(root, args)
	w.exitCode = ExitCode(w.outcome)
	w.stdout, w.stderr = out.String(), errb.String()

	if w.secret != "" && strings.Contains(w.stdout+w.stderr, w.secret) {
		return fmt.Errorf("the token leaked into output: stdout=%q stderr=%q", w.stdout, w.stderr)
	}
	return nil
}

func (w *postCreateValidityWorld) runCreateWithTemplate() error {
	if len(w.tmplFiles) == 0 {
		return fmt.Errorf("no user template was set up by the Given")
	}
	if w.readStep.body == proposalCreatedBody {
		// A favourable read-back that DIFFERS from the create response on one
		// pre-074 field. The transition set is the discriminator: the create
		// response carries ["propose"], this carries both, so the Then step can
		// tell which document the template rendered. Rigging the two to agree
		// (as this fixture originally did) makes the assertion unfalsifiable —
		// validate.md Round 2, F-1.
		w.readStep = proposalSeqStep{status: 200, body: readBackBodyWith("draft", `"valid":true,"validation_alerts":[],`, `["propose","withdraw"]`)}
	}
	return w.runCommand(`proposal create ten_0123 --changes '[{"type":"CreateRole","name":"Scribe"}]' --output pre074.tmpl`)
}

// runSiblingCommands drives `proposal get`, `proposal propose`, and
// `proposal withdraw` over responses that DO carry the validity fields — the
// exact leak surface ADR-4 routes around — and captures each command's stdout
// plus the shared proposal rendering they must all still match.
func (w *postCreateValidityWorld) runSiblingCommands() error {
	body := readBackBodyWith("draft", `"valid":false,"validation_alerts":[{"severity":"error","path":"name","message":"refused"}],`, `["propose"]`)

	var doc glassfrog.Document[glassfrog.Proposal]
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		return fmt.Errorf("fixture decode: %v", err)
	}
	want, err := render.Render(render.ResourceProposal, render.FormatFull, render.ProposalView{Proposal: doc.Data})
	if err != nil {
		return fmt.Errorf("shared rendering: %v", err)
	}
	w.siblingWant = want

	w.siblingOutputs = map[string]string{}
	for _, invocation := range []string{
		"proposal get prp_0123",
		"proposal propose prp_0123",
		"proposal withdraw prp_0123",
	} {
		root := NewRootCommand()
		tr := &tensionTransport{status: 200, body: body}
		seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: w.ctx, transport: tr}}
		MustRegister(root, newProposalCommand(seam))
		var out, errb bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&errb)
		outcome, _ := Run(root, splitArgsPOSIX(invocation))
		if outcome != Success {
			return fmt.Errorf("%q should succeed, got %v\nstderr: %s", invocation, outcome, errb.String())
		}
		w.siblingOutputs[invocation] = out.String()
	}
	return nil
}

// --- Then implementations ---

func (w *postCreateValidityWorld) createdPrintedWithIDAndStatus(idPrefix, status string) error {
	for _, want := range []string{idPrefix, status} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the created proposal should print %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *postCreateValidityWorld) createdPrintedWithID(idPrefix string) error {
	if !strings.Contains(w.stdout, idPrefix) {
		return fmt.Errorf("the created proposal should print its %s id:\n%s", idPrefix, w.stdout)
	}
	return nil
}

func (w *postCreateValidityWorld) validityReportedAs(label string) error {
	want := "  Validity:       " + label + "\n"
	if !strings.Contains(w.stdout, want) {
		return fmt.Errorf("the result should report the validity as %q:\n%s", label, w.stdout)
	}
	return nil
}

func (w *postCreateValidityWorld) exactlyOneReadFollowed() error {
	if w.transport == nil {
		return fmt.Errorf("no command was run")
	}
	if w.transport.calls != 2 {
		return fmt.Errorf("a successful create is exactly two exchanges (POST + one read), got %d", w.transport.calls)
	}
	if w.transport.methods[1] != "GET" || !strings.HasSuffix(w.transport.paths[1], "/proposals/prp_0123") {
		return fmt.Errorf("the second exchange should be GET /proposals/prp_0123, got %s %s", w.transport.methods[1], w.transport.paths[1])
	}
	return nil
}

func (w *postCreateValidityWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d\nstderr: %s", w.exitCode, code, w.stderr)
	}
	return nil
}

func (w *postCreateValidityWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *postCreateValidityWorld) alertCarriedWithSeverityAndPath() error {
	if w.wantAlertMessage == "" {
		return fmt.Errorf("no alert message was set up by the Given")
	}
	if !strings.Contains(w.stdout, w.wantAlertMessage) {
		return fmt.Errorf("the result should carry the server's message %q:\n%s", w.wantAlertMessage, w.stdout)
	}
	// The alert line renders `- [<severity>] <path>: <message>` — severity badge
	// and path present on the same line as the message.
	for _, line := range strings.Split(w.stdout, "\n") {
		if strings.Contains(line, w.wantAlertMessage) {
			if !strings.Contains(line, "[") || !strings.Contains(line, "]") || !strings.Contains(line, ":") {
				return fmt.Errorf("the alert line should carry its severity and path: %q", line)
			}
			return nil
		}
	}
	return fmt.Errorf("no alert line found:\n%s", w.stdout)
}

func (w *postCreateValidityWorld) noTransitionsReported() error {
	if !strings.Contains(w.stdout, "  Transitions:    (none)\n") {
		return fmt.Errorf("the result should report no available transitions:\n%s", w.stdout)
	}
	return nil
}

func (w *postCreateValidityWorld) stderrNamesCreateFailure() error {
	want := fmt.Sprintf("%d", w.createStep.status)
	if !strings.Contains(w.stderr, want) {
		return fmt.Errorf("stderr should name the HTTP status (%s):\n%s", want, w.stderr)
	}
	return nil
}

func (w *postCreateValidityWorld) noProposalReadAttempted() error {
	if w.transport == nil {
		return fmt.Errorf("no command was run")
	}
	for i, m := range w.transport.methods {
		if m == "GET" && strings.Contains(w.transport.paths[i], "/proposals/") {
			return fmt.Errorf("a proposal read was attempted: GET %s", w.transport.paths[i])
		}
	}
	return nil
}

func (w *postCreateValidityWorld) noVerdictStatedReported() error {
	if !strings.Contains(w.stdout, "  Validity:       not reported by the server\n") {
		return fmt.Errorf("the result should report the server stated no verdict:\n%s", w.stdout)
	}
	return nil
}

// neitherValidNorNotValid pins that the reported state is not one of the two
// server verdicts: the Validity line's value neither reads `valid` nor
// `not valid` (the unreported/unavailable states have their own labels).
func (w *postCreateValidityWorld) neitherValidNorNotValid() error {
	for _, forbidden := range []string{"  Validity:       valid\n", "  Validity:       not valid\n"} {
		if strings.Contains(w.stdout, forbidden) {
			return fmt.Errorf("the draft must not be described as %q:\n%s", strings.TrimSpace(forbidden), w.stdout)
		}
	}
	return nil
}

// factsStayDistinct pins that validity and transitions are reported as two
// separate lines, neither derived from the other: the valid verdict stands next
// to an empty transition set.
func (w *postCreateValidityWorld) factsStayDistinct() error {
	if !strings.Contains(w.stdout, "  Validity:       valid\n") {
		return fmt.Errorf("the validity fact should stand as given:\n%s", w.stdout)
	}
	if !strings.Contains(w.stdout, "  Transitions:    (none)\n") {
		return fmt.Errorf("the transitions fact should stand as given:\n%s", w.stdout)
	}
	return nil
}

func (w *postCreateValidityWorld) statusReportedAsGiven(status string) error {
	if !strings.Contains(w.stdout, "["+status+"]") {
		return fmt.Errorf("the status %q should be reported as the server gave it:\n%s", status, w.stdout)
	}
	return nil
}

func (w *postCreateValidityWorld) statusAndVerdictUnadjusted() error {
	if w.wantStatus == "" {
		return fmt.Errorf("no conflicted status was set up by the Given")
	}
	if !strings.Contains(w.stdout, "["+w.wantStatus+"]") {
		return fmt.Errorf("the status should stand unadjusted:\n%s", w.stdout)
	}
	if !strings.Contains(w.stdout, "  Validity:       valid\n") {
		return fmt.Errorf("the verdict should stand unadjusted:\n%s", w.stdout)
	}
	return nil
}

func (w *postCreateValidityWorld) alertNotUnfavourable() error {
	if !strings.Contains(w.stdout, "  Validity:       valid\n") {
		return fmt.Errorf("the alert's presence must not turn the verdict unfavourable:\n%s", w.stdout)
	}
	for _, forbidden := range []string{"not valid", "unavailable"} {
		if strings.Contains(w.stdout, "  Validity:       "+forbidden) {
			return fmt.Errorf("the verdict must stay %q, got %q:\n%s", "valid", forbidden, w.stdout)
		}
	}
	return nil
}

func (w *postCreateValidityWorld) structuredContainsID(idPrefix string) error {
	if !strings.Contains(w.stdout, idPrefix) {
		return fmt.Errorf("the structured result should contain the %s id:\n%s", idPrefix, w.stdout)
	}
	if strings.Contains(w.stdout, "Transitions:") {
		return fmt.Errorf("structured output must not render the human projection:\n%s", w.stdout)
	}
	return nil
}

func (w *postCreateValidityWorld) structuredCarriesKeys(k1, k2, k3 string) error {
	var doc struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(w.stdout), &doc); err != nil {
		return fmt.Errorf("stdout should be one structured server document: %v\n%s", err, w.stdout)
	}
	for _, key := range []string{k1, k2, k3} {
		if _, ok := doc.Data[key]; !ok {
			return fmt.Errorf("the structured result should carry %q:\n%s", key, w.stdout)
		}
	}
	return nil
}

// machineAdvisory decodes the structured stderr advisory, skipping any 017 retry
// notices that precede it.
func (w *postCreateValidityWorld) machineAdvisory() (map[string]json.RawMessage, error) {
	docStart := strings.Index(w.stderr, "{")
	if docStart < 0 {
		return nil, fmt.Errorf("no structured advisory on stderr:\n%s", w.stderr)
	}
	var advisory struct {
		VerdictSource map[string]json.RawMessage `json:"verdict_source"`
	}
	if err := json.Unmarshal([]byte(w.stderr[docStart:]), &advisory); err != nil {
		return nil, fmt.Errorf("the advisory should be structured in the selected machine format: %v\n%s", err, w.stderr)
	}
	if advisory.VerdictSource == nil {
		return nil, fmt.Errorf("the advisory should carry verdict_source:\n%s", w.stderr)
	}
	return advisory.VerdictSource, nil
}

func (w *postCreateValidityWorld) machineAdvisoryCarries(key, value string) error {
	vs, err := w.machineAdvisory()
	if err != nil {
		return err
	}
	got, ok := vs[key]
	if !ok {
		return fmt.Errorf("the advisory should carry %q:\n%s", key, w.stderr)
	}
	if string(got) != value {
		return fmt.Errorf("advisory %s = %s, want %s", key, got, value)
	}
	return nil
}

func (w *postCreateValidityWorld) advisoryCarriesReasonAndRemedy(remedyPrefix string) error {
	vs, err := w.machineAdvisory()
	if err != nil {
		return err
	}
	var reason, remedy string
	if raw, ok := vs["reason"]; !ok || json.Unmarshal(raw, &reason) != nil || reason == "" {
		return fmt.Errorf("the advisory should carry a non-empty reason:\n%s", w.stderr)
	}
	if raw, ok := vs["remedy"]; !ok || json.Unmarshal(raw, &remedy) != nil {
		return fmt.Errorf("the advisory should carry the remedy:\n%s", w.stderr)
	}
	if !strings.HasPrefix(remedy, remedyPrefix) {
		return fmt.Errorf("remedy = %q, want it to name %q", remedy, remedyPrefix)
	}
	return nil
}

// noProseNeeded pins the four-state machine accord: the state is identifiable
// from stdout's data.valid plus the advisory's read_back alone — both structured
// documents, no prose. Here (a failed read-back) that means: stdout decodes, its
// data carries no `valid` key, and read_back is false with a reason.
func (w *postCreateValidityWorld) noProseNeeded() error {
	var doc struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(w.stdout), &doc); err != nil {
		return fmt.Errorf("stdout should be structured: %v", err)
	}
	if _, present := doc.Data["valid"]; present {
		return fmt.Errorf("a failed read-back must emit the create's document (no valid key):\n%s", w.stdout)
	}
	vs, err := w.machineAdvisory()
	if err != nil {
		return err
	}
	if string(vs["read_back"]) != "false" {
		return fmt.Errorf("read_back = %s, want false", vs["read_back"])
	}
	if _, ok := vs["reason"]; !ok {
		return fmt.Errorf("the unavailable state must carry its reason:\n%s", w.stderr)
	}
	return nil
}

func (w *postCreateValidityWorld) compactCarriesIDStatusCount() error {
	line := strings.TrimRight(w.stdout, "\n")
	if strings.Contains(line, "\n") {
		return fmt.Errorf("compact output should be one line, got:\n%s", w.stdout)
	}
	for _, want := range []string{"prp_0123", "[draft]", "1 change(s)"} {
		if !strings.Contains(line, want) {
			return fmt.Errorf("the compact line should carry %q: %q", want, line)
		}
	}
	return nil
}

// compactCarriesValidityAndAlertCount reads the expected token and count from what
// the read-back Given set up, rather than hard-coding one scenario's verdict: the
// same step text is used by scenarios whose draft is valid and whose draft is not,
// so a hard-coded "not valid" would assert the wrong thing for half of them.
func (w *postCreateValidityWorld) compactCarriesValidityAndAlertCount() error {
	if w.wantValidity == "" {
		return fmt.Errorf("no read-back verdict was set up by the Given")
	}
	line := strings.TrimRight(w.stdout, "\n")
	want := fmt.Sprintf("%s (%d alert", w.wantValidity, w.wantAlertCount)
	if !strings.Contains(line, want) {
		return fmt.Errorf("the compact line should carry the validity token and the alert count (%q): %q", want, line)
	}
	return nil
}

func (w *postCreateValidityWorld) templateFieldsResolve() error {
	if w.outcome != Success {
		return fmt.Errorf("the template render should succeed, got %v\nstderr: %s", w.outcome, w.stderr)
	}
	return nil
}

// templateOutputCarriesReadBack pins WHICH document supplied the values a
// pre-074 template projects. Where the read-back answered, it is the read-back's
// proposal (ADR-4, plan § Verdict Assembly) — so the transition count is the
// read-back's two, not the create response's one. The paths themselves are
// unchanged by the verdict's addition; the values are not promised to be, and
// asserting otherwise is what made this step inert before.
func (w *postCreateValidityWorld) templateOutputCarriesReadBack() error {
	want := "prp_0123 draft 1 2"
	if w.stdout != want {
		return fmt.Errorf("template output = %q, want %q (the read-back's transitions, not the create's)", w.stdout, want)
	}
	return nil
}

func (w *postCreateValidityWorld) siblingsRenderNoVerdict() error {
	if len(w.siblingOutputs) == 0 {
		return fmt.Errorf("no sibling commands were run")
	}
	for invocation, out := range w.siblingOutputs {
		for _, forbidden := range []string{"Validity:", "Alerts (", "Verdict source:"} {
			if strings.Contains(out, forbidden) {
				return fmt.Errorf("%q leaked a verdict line (%q):\n%s", invocation, forbidden, out)
			}
		}
	}
	return nil
}

func (w *postCreateValidityWorld) siblingsOutputUnchanged() error {
	for invocation, out := range w.siblingOutputs {
		if out != w.siblingWant {
			return fmt.Errorf("%q output diverged from the shared proposal rendering:\n got: %q\nwant: %q", invocation, out, w.siblingWant)
		}
	}
	return nil
}

func (w *postCreateValidityWorld) verdictUnobtainableWithCause() error {
	if !strings.Contains(w.stdout, "  Validity:       unavailable — ") {
		return fmt.Errorf("the result should report the verdict as unobtainable:\n%s", w.stdout)
	}
	if !strings.Contains(w.stdout, "could not be read back (") {
		return fmt.Errorf("the unavailable line should name the cause:\n%s", w.stdout)
	}
	return nil
}

func (w *postCreateValidityWorld) verdictUnobtainableBudget() error {
	if !strings.Contains(w.stdout, "  Validity:       unavailable — the read-back was rate limited (the request budget was exhausted)\n") {
		return fmt.Errorf("the result should name the exhausted request budget:\n%s", w.stdout)
	}
	return nil
}

func (w *postCreateValidityWorld) idUndeterminableReported() error {
	if !strings.Contains(w.stdout+w.stderr, "could not be determined") {
		return fmt.Errorf("the result should report the id could not be determined:\nstdout: %s\nstderr: %s", w.stdout, w.stderr)
	}
	return nil
}

func (w *postCreateValidityWorld) createResponseStillReported() error {
	if !strings.Contains(w.stdout, "[draft]") {
		return fmt.Errorf("the create response should still be reported:\n%s", w.stdout)
	}
	return nil
}
