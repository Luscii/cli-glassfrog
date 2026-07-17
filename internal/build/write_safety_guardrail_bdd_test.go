package build

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestWriteSafetyGuardrailFeatures runs the executable acceptance for the
// Write-Safety Guardrail (063). Like the sibling build-side suites (021/022/036/
// 062) its Paths name ONLY this spec's feature file and it runs with the ~@wip
// filter, so only the scenarios implemented so far execute.
//
// The deliverable is a Claude plugin PreToolUse hook (a bash gate script) plus a
// single-sourced gated-command registry and a drift tripwire, so the executable
// scenarios drive the REAL gate script — feeding it tool-call JSON on stdin and
// asserting the permission decision it emits — and model the host's confirmation
// loop (a decision of `ask` is sent only if the practitioner confirms). The five
// @validation scenarios (no ungated write path, no invented surface, states the
// change not its merits, no blind retry, tension edits stay outside the gate)
// stay @wip, held for /score:validate. The drift scenario is implemented in T003.
func TestWriteSafetyGuardrailFeatures(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		// The guardrail's runtime is pinned to bash; its absence is a genuine
		// environment failure for this feature, not a reason to skip silently.
		t.Fatalf("bash not found on PATH — the write-safety gate is a bash hook: %v", err)
	}
	w := &gateWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.register(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unequipped-agent-operators/write-safety-guardrail.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: write-safety-guardrail feature scenarios failed")
	}
}

// gateWorld is the per-scenario state. It drives the real gate script and models a
// minimal plugin host: it holds the command under evaluation, the decision the
// gate emitted, and — via the confirmation model — whether the modelled host
// would send the command.
type gateWorld struct {
	hookInstalled bool // false models a host without the hook (fallback scenario)
	command       string
	decision      string // "ask" (gate escalated) or "" (pass-through / allow)
	message       string
	didEval       bool

	confirmed     bool // practitioner's confirmation decision for an `ask`
	sent          bool // did the modelled host run the command?
	sentCommand   string
	recordChanged bool

	staleExitCode int    // the exit code of a prior confirmed write (7 == stale)
	unlistedLeaf  string // the concrete unrecognized proposal leaf under test

	// Drift tripwire scenario (T003).
	registryLeaves []string
	liveSurface    []string
	droppedLeaf    string
	driftFindings  []string
}

func (w *gateWorld) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = gateWorld{hookInstalled: true}
		return ctx, nil
	})

	// Keep a human decision in the loop --------------------------------------
	sc.Step(`^the guardrail hook was active over the Bash tool$`, w.givenHookActive)
	sc.Step(`^the agent was about to run "([^"]*)"$`, w.givenAboutToRun)
	sc.Step(`^the PreToolUse hook evaluates the command$`, w.whenHookEvaluates)
	sc.Step(`^the hook will return permissionDecision "([^"]*)"$`, w.thenDecisionIs)
	sc.Step(`^its message will name the command, the target "([^"]*)", and that it advances the proposal into circulation$`, w.thenMessageNamesProposeInto)
	sc.Step(`^the write will not be sent until the practitioner explicitly confirms it$`, w.thenNotSentUntilConfirm)

	sc.Step(`^the practitioner explicitly confirmed "([^"]*)"$`, w.givenConfirmed)
	sc.Step(`^the agent executes the confirmed write$`, w.whenExecuteConfirmed)
	sc.Step(`^exactly the confirmed draft proposal will be created from tension "([^"]*)"$`, w.thenExactlyCreatedFrom)
	sc.Step(`^the agent will not broaden, substitute, or bundle any additional write into the action$`, w.thenNoBroaden)

	sc.Step(`^the hook returned permissionDecision "([^"]*)"$`, w.andHookReturnedDecision)
	sc.Step(`^the practitioner does not confirm the write$`, w.whenNotConfirm)
	sc.Step(`^the command will not run$`, w.thenCommandNotRun)
	sc.Step(`^the governance record will remain unchanged$`, w.thenRecordUnchanged)

	sc.Step(`^the agent invoked "([^"]*)" that the registry did not list$`, w.givenInvokedUnlisted)
	sc.Step(`^a future proposal write will be gated by default until the registry is updated$`, w.thenFutureGatedByDefault)

	// Re-read and re-confirm a stale write -----------------------------------
	sc.Step(`^a confirmed write was refused as a stale write with exit code (\d+)$`, w.givenStaleRefusal)
	sc.Step(`^the agent re-reads the resource for its current version and retries the write$`, w.whenReReadAndRetry)
	sc.Step(`^the retry will itself be a proposal write the hook gates again$`, w.thenRetryGatedAgain)
	sc.Step(`^the practitioner will be asked to confirm against the now-current state before it is sent$`, w.thenAskedToConfirmCurrent)

	sc.Step(`^a stale-write refusal had prompted a re-read of the current state$`, w.givenStalePromptedReRead)
	sc.Step(`^the practitioner does not re-confirm against that current state$`, w.whenNotReConfirm)
	sc.Step(`^the agent will not retry the write$`, w.thenNotRetry)
	sc.Step(`^the resource will remain as the concurrent change last set it$`, w.thenResourceUnchanged)

	sc.Step(`^a confirmed write failed with a permission outcome rather than the stale-write category$`, w.givenNonStaleFailure)
	sc.Step(`^the agent observes the outcome$`, w.whenObserveOutcome)
	sc.Step(`^it will not invoke the re-read and re-confirm recovery$`, w.thenNoRecoveryInvoked)
	sc.Step(`^the failure will flow through the CLI's normal failure handling unchanged$`, w.thenNormalFailureHandling)

	// Reads and operational tension edits pass ungated -----------------------
	sc.Step(`^the hook will not require confirmation$`, w.thenNotRequireConfirmation)
	sc.Step(`^the read will proceed immediately$`, w.thenReadProceeds)
	sc.Step(`^the tension will be captured immediately$`, w.thenTensionCaptured)

	sc.Step(`^the guardrail hook was not installed in the agent's host$`, w.givenHookNotInstalled)
	sc.Step(`^the agent runs "([^"]*)"$`, w.whenAgentRuns)
	sc.Step(`^the write will proceed under Operator Orientation's guidance only with no enforcement$`, w.thenProceedUnderGuidance)
	sc.Step(`^nothing in the CLI will break$`, w.thenNothingBreaks)

	// Drift tripwire (T003) --------------------------------------------------
	sc.Step(`^the registry gated a proposal-write leaf the CLI's proposal surface no longer exposed$`, w.givenRegistryGatesDroppedLeaf)
	sc.Step(`^the internal/build drift tripwire runs$`, w.whenDriftTripwireRuns)
	sc.Step(`^the tripwire will fail$`, w.thenTripwireFails)
	sc.Step(`^it will report that the proposal subcommand surface changed without the registry$`, w.thenReportsSurfaceChanged)
}

// --- Gate driver + host model ----------------------------------------------

// runGateScript feeds the real gate script a Bash tool call carrying command and
// returns the permission decision ("" when the gate passes through) and message.
func runGateScript(command string) (decision, message string, err error) {
	root, err := RepoRoot()
	if err != nil {
		return "", "", err
	}
	script := filepath.Join(root, filepath.FromSlash(GateScriptPath))
	type toolInput struct {
		Command string `json:"command"`
	}
	type toolCall struct {
		ToolName  string    `json:"tool_name"`
		ToolInput toolInput `json:"tool_input"`
	}
	raw, err := json.Marshal(toolCall{ToolName: "Bash", ToolInput: toolInput{Command: command}})
	if err != nil {
		return "", "", err
	}
	cmd := exec.Command("bash", script)
	cmd.Stdin = bytes.NewReader(raw)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if runErr := cmd.Run(); runErr != nil {
		return "", "", fmt.Errorf("gate script failed: %v (stderr: %s)", runErr, errBuf.String())
	}
	trimmed := strings.TrimSpace(out.String())
	if trimmed == "" {
		return "", "", nil // pass-through
	}
	var g struct {
		HookSpecificOutput struct {
			PermissionDecision       string `json:"permissionDecision"`
			PermissionDecisionReason string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
		SystemMessage string `json:"systemMessage"`
	}
	if uerr := json.Unmarshal([]byte(trimmed), &g); uerr != nil {
		return "", "", fmt.Errorf("gate emitted non-JSON output %q: %w", trimmed, uerr)
	}
	msg := g.SystemMessage
	if msg == "" {
		msg = g.HookSpecificOutput.PermissionDecisionReason
	}
	return g.HookSpecificOutput.PermissionDecision, msg, nil
}

// runGateRawStdin feeds the gate script arbitrary raw bytes on stdin (bypassing
// json.Marshal, so a raw control character survives into the command value) and
// returns the raw stdout. Used to assert the gate emits valid JSON even when the
// command carries control characters.
func runGateRawStdin(stdin string) (string, error) {
	root, err := RepoRoot()
	if err != nil {
		return "", err
	}
	script := filepath.Join(root, filepath.FromSlash(GateScriptPath))
	cmd := exec.Command("bash", script)
	cmd.Stdin = bytes.NewReader([]byte(stdin))
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if runErr := cmd.Run(); runErr != nil {
		return "", fmt.Errorf("gate script failed: %v (stderr: %s)", runErr, errBuf.String())
	}
	return out.String(), nil
}

func (w *gateWorld) evaluate(command string) error {
	w.command = command
	dec, msg, err := runGateScript(command)
	if err != nil {
		return err
	}
	w.decision, w.message, w.didEval = dec, msg, true
	return nil
}

func (w *gateWorld) ensureEvaluated() error {
	if w.didEval {
		return nil
	}
	return w.evaluate(w.command)
}

// execute models the plugin host: it runs the command unless the gate escalated
// to `ask` and the practitioner did not confirm. When the hook is not installed,
// the host has no gate at all and runs the command directly.
func (w *gateWorld) execute() {
	if !w.hookInstalled {
		w.sent, w.sentCommand, w.recordChanged = true, w.command, true
		return
	}
	if w.decision == "ask" && !w.confirmed {
		w.sent = false
		return
	}
	w.sent, w.sentCommand, w.recordChanged = true, w.command, true
}

// --- Keep a human decision in the loop -------------------------------------

func (w *gateWorld) givenHookActive() error { w.hookInstalled = true; return nil }

func (w *gateWorld) givenAboutToRun(command string) error { w.command = command; return nil }

func (w *gateWorld) whenHookEvaluates() error { return w.evaluate(w.command) }

func (w *gateWorld) thenDecisionIs(want string) error {
	if err := w.ensureEvaluated(); err != nil {
		return err
	}
	if w.decision != want {
		return fmt.Errorf("gate returned permissionDecision %q for %q, want %q", w.decision, w.command, want)
	}
	return nil
}

func (w *gateWorld) thenMessageNamesProposeInto(target string) error {
	if err := w.ensureEvaluated(); err != nil {
		return err
	}
	if !strings.Contains(w.message, w.command) {
		return fmt.Errorf("gate message does not name the command %q: %q", w.command, w.message)
	}
	if !strings.Contains(w.message, target) {
		return fmt.Errorf("gate message does not name the target %q: %q", target, w.message)
	}
	if !containsFold(w.message, "into circulation") {
		return fmt.Errorf("gate message does not state the effect (advances into circulation): %q", w.message)
	}
	return nil
}

func (w *gateWorld) thenNotSentUntilConfirm() error {
	if w.decision != "ask" {
		return fmt.Errorf("write was not gated (decision %q) — nothing holds it for confirmation", w.decision)
	}
	// Without confirmation the modelled host does not send the write...
	w.confirmed = false
	w.execute()
	if w.sent {
		return fmt.Errorf("write was sent without confirmation")
	}
	// ...and only an explicit confirmation lets it through.
	w.confirmed = true
	w.execute()
	if !w.sent {
		return fmt.Errorf("write was still not sent after explicit confirmation")
	}
	return nil
}

func (w *gateWorld) givenConfirmed(command string) error {
	if err := w.evaluate(command); err != nil {
		return err
	}
	if w.decision != "ask" {
		return fmt.Errorf("confirmed command %q was not gated (decision %q) — a confirmation only exists for a gated write", command, w.decision)
	}
	w.confirmed = true
	return nil
}

func (w *gateWorld) whenExecuteConfirmed() error { w.execute(); return nil }

func (w *gateWorld) thenExactlyCreatedFrom(target string) error {
	if !w.sent {
		return fmt.Errorf("confirmed write was not executed")
	}
	if !strings.Contains(w.message, target) || !containsFold(w.message, "draft proposal from tension") {
		return fmt.Errorf("gate message does not describe creating a draft proposal from tension %q: %q", target, w.message)
	}
	return nil
}

func (w *gateWorld) thenNoBroaden() error {
	// The host runs EXACTLY the evaluated command — the gate classifies, it never
	// rewrites or bundles. So the sent command is byte-identical to the confirmed one.
	if w.sentCommand != w.command {
		return fmt.Errorf("sent command %q differs from the confirmed command %q — the action was broadened/substituted", w.sentCommand, w.command)
	}
	return nil
}

func (w *gateWorld) andHookReturnedDecision(want string) error { return w.thenDecisionIs(want) }

func (w *gateWorld) whenNotConfirm() error { w.confirmed = false; w.execute(); return nil }

func (w *gateWorld) thenCommandNotRun() error {
	if w.sent {
		return fmt.Errorf("command ran despite withheld confirmation")
	}
	return nil
}

func (w *gateWorld) thenRecordUnchanged() error {
	if w.recordChanged {
		return fmt.Errorf("governance record changed despite the write not being sent")
	}
	return nil
}

// givenInvokedUnlisted models an unrecognized proposal subcommand. The feature
// text uses the placeholder "<new-write-leaf>"; we substitute a CONCRETE leaf
// that genuinely is not in the registry, so the assertion exercises fail-closed
// classification of a real token rather than a template string.
func (w *gateWorld) givenInvokedUnlisted(command string) error {
	const concreteUnlisted = "escalate"
	command = strings.Replace(command, "<new-write-leaf>", concreteUnlisted, 1)
	w.unlistedLeaf = concreteUnlisted
	return w.evaluate(command)
}

func (w *gateWorld) thenFutureGatedByDefault() error {
	leaves, err := ReadGatedRegistry()
	if err != nil {
		return err
	}
	for _, l := range leaves {
		if l == "proposal "+w.unlistedLeaf {
			return fmt.Errorf("the leaf %q used to prove fail-closed is actually listed in the registry — the test no longer exercises an unrecognized subcommand", w.unlistedLeaf)
		}
	}
	if w.decision != "ask" {
		return fmt.Errorf("an unrecognized proposal subcommand was not gated (decision %q) — fail-closed did not hold", w.decision)
	}
	return nil
}

// --- Re-read and re-confirm a stale write ----------------------------------

// A concrete proposal write used across the stale-write scenarios.
const staleProposalWrite = "glassfrog proposal propose prp_0123"

func (w *gateWorld) givenStaleRefusal(code int) error {
	w.staleExitCode = code
	w.command = staleProposalWrite
	return nil
}

func (w *gateWorld) whenReReadAndRetry() error {
	// The retry is itself the same proposal-path write; re-running it re-enters the
	// gate exactly as the first attempt did.
	return w.evaluate(w.command)
}

func (w *gateWorld) thenRetryGatedAgain() error {
	if w.decision != "ask" {
		return fmt.Errorf("the retried write was not re-gated (decision %q) — a blind retry could bypass confirmation", w.decision)
	}
	return nil
}

func (w *gateWorld) thenAskedToConfirmCurrent() error {
	if w.decision != "ask" {
		return fmt.Errorf("retry decision was %q, want ask", w.decision)
	}
	// Not sent until confirmed, same as any gated write.
	w.confirmed = false
	w.execute()
	if w.sent {
		return fmt.Errorf("retry was sent without a fresh confirmation against the current state")
	}
	return nil
}

func (w *gateWorld) givenStalePromptedReRead() error {
	w.staleExitCode = 7
	return w.evaluate(staleProposalWrite)
}

func (w *gateWorld) whenNotReConfirm() error { w.confirmed = false; w.execute(); return nil }

func (w *gateWorld) thenNotRetry() error {
	if w.sent {
		return fmt.Errorf("the write was retried despite withheld re-confirmation")
	}
	return nil
}

func (w *gateWorld) thenResourceUnchanged() error { return w.thenRecordUnchanged() }

func (w *gateWorld) givenNonStaleFailure() error {
	// A permission outcome (not the stale-write category). Exit codes other than 7
	// are not stale writes; 054 maps codeStaleWrite=7.
	w.staleExitCode = 3
	return nil
}

func (w *gateWorld) whenObserveOutcome() error { return nil }

// thenNoRecoveryInvoked asserts the guardrail interposes NO stale-write recovery.
// The gate is a pre-execution classifier: it never reads an outcome/exit code and
// carries no re-read-and-retry machinery (plan ADR-5 — the re-read guidance lives
// in Operator Orientation, and the hook only re-gates a retry). So a non-stale
// failure cannot trigger any recovery the guardrail owns.
func (w *gateWorld) thenNoRecoveryInvoked() error {
	if w.staleExitCode == 7 {
		return fmt.Errorf("scenario models a non-stale failure but exit code is 7 (stale)")
	}
	// The gate is a pre-execution classifier: its only input is the tool call on
	// stdin, and it never reads a command's exit status. A stale-write recovery
	// handler would have to branch on the outcome ($?), so the absence of any $?
	// read is the structural proof that the guardrail owns no recovery a non-stale
	// failure could trigger — recovery is not the hook's job (ADR-5); the re-read
	// guidance stays in Operator Orientation.
	script, err := ReadGateScript()
	if err != nil {
		return err
	}
	if strings.Contains(script, "$?") {
		return fmt.Errorf("gate script reads $? — it must not branch on a command's outcome; the guardrail owns no stale-write recovery (ADR-5)")
	}
	return nil
}

func (w *gateWorld) thenNormalFailureHandling() error {
	// The guardrail adds no failure-handling path at all: it registers a PreToolUse
	// gate (which runs BEFORE the command, so it has no outcome to react to) and no
	// PostToolUse hook. Any failure — stale or not — is therefore left to the CLI's
	// own handling untouched.
	cfg, _, err := ReadHooksConfig()
	if err != nil {
		return err
	}
	if _, ok := PreToolUseBashGate(cfg); !ok {
		return fmt.Errorf("no PreToolUse Bash gate found — cannot confirm the guardrail is pre-execution only")
	}
	if len(cfg.Hooks["PostToolUse"]) != 0 {
		return fmt.Errorf("registration includes a PostToolUse hook — it would interpose on outcomes rather than leaving failures to normal handling")
	}
	return nil
}

// --- Reads and operational tension edits pass ungated ----------------------

func (w *gateWorld) thenNotRequireConfirmation() error {
	if err := w.ensureEvaluated(); err != nil {
		return err
	}
	if w.decision == "ask" {
		return fmt.Errorf("command %q was gated (decision ask) but should pass ungated", w.command)
	}
	return nil
}

func (w *gateWorld) thenReadProceeds() error {
	w.execute()
	if !w.sent {
		return fmt.Errorf("read %q did not proceed immediately", w.command)
	}
	return nil
}

func (w *gateWorld) thenTensionCaptured() error {
	w.execute()
	if !w.sent {
		return fmt.Errorf("tension edit %q did not proceed immediately", w.command)
	}
	return nil
}

func (w *gateWorld) givenHookNotInstalled() error { w.hookInstalled = false; return nil }

func (w *gateWorld) whenAgentRuns(command string) error {
	w.command = command
	if !w.hookInstalled {
		// No gate fires; the host runs the command directly.
		w.decision, w.didEval = "", true
		w.execute()
		return nil
	}
	if err := w.evaluate(command); err != nil {
		return err
	}
	w.execute()
	return nil
}

func (w *gateWorld) thenProceedUnderGuidance() error {
	if w.hookInstalled {
		return fmt.Errorf("scenario models an absent hook, but hookInstalled is true")
	}
	if w.decision == "ask" {
		return fmt.Errorf("a gate decision was produced despite the hook being absent")
	}
	if !w.sent {
		return fmt.Errorf("write did not proceed under guidance-only fallback")
	}
	return nil
}

func (w *gateWorld) thenNothingBreaks() error {
	// The plugin tree is pure data — nothing under plugin/ compiles into the CLI, so
	// the hook's presence or absence can never break a glassfrog command.
	clean, err := OrientationPluginHasNoGoCode()
	if err != nil {
		return err
	}
	if !clean {
		return fmt.Errorf("plugin tree carries Go code — hook absence could no longer be isolated from the CLI")
	}
	return nil
}

// --- Drift tripwire (T003) -------------------------------------------------

// givenRegistryGatesDroppedLeaf models the CLI dropping a still-gated proposal
// write: the registry keeps every real leaf, but the live surface handed to the
// tripwire has one gated leaf removed — the "a gated leaf left the CLI" condition.
func (w *gateWorld) givenRegistryGatesDroppedLeaf() error {
	reg, err := ReadGatedRegistry()
	if err != nil {
		return err
	}
	live, err := LiveProposalSubcommands()
	if err != nil {
		return err
	}
	const dropped = "withdraw" // a real gated write leaf, removed to model the CLI dropping it
	var doctored []string
	for _, l := range live {
		if l != dropped {
			doctored = append(doctored, l)
		}
	}
	if len(doctored) == len(live) {
		return fmt.Errorf("setup error: %q was not in the live proposal surface %v, so nothing was dropped", dropped, live)
	}
	w.registryLeaves, w.liveSurface, w.droppedLeaf = reg, doctored, dropped
	return nil
}

func (w *gateWorld) whenDriftTripwireRuns() error {
	w.driftFindings = CheckRegistryDrift(w.registryLeaves, w.liveSurface)
	return nil
}

func (w *gateWorld) thenTripwireFails() error {
	if len(w.driftFindings) == 0 {
		return fmt.Errorf("drift tripwire reported no findings, but a gated leaf left the CLI's proposal surface")
	}
	return nil
}

func (w *gateWorld) thenReportsSurfaceChanged() error {
	for _, f := range w.driftFindings {
		if strings.Contains(f, w.droppedLeaf) {
			return nil
		}
	}
	return fmt.Errorf("no drift finding named the offending command %q; got %v", w.droppedLeaf, w.driftFindings)
}
