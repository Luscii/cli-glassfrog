package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/rcfile"
	"github.com/Luscii/cli-glassfrog/internal/render"
	"github.com/cucumber/godog"
)

// TestOutputFormatSelectionFeatures runs the executable acceptance for Output
// Format Selection (020): the --output precedence resolver exercised directly
// through output.ResolveFormat, plus the four reads driven through their seams over
// fakes so each format routes to its renderer and an invalid selector fails fast
// with no request — every scenario runs offline. Its Paths name ONLY this spec's
// feature file, so un-@wip-ping these scenarios cannot disturb another suite. The
// three @validation scenarios stay @wip (held for the validate skill) and are
// skipped by the ~@wip filter.
func TestOutputFormatSelectionFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeOutputFormatScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unconsumable-output/output-format-selection.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: output-format-selection feature scenarios failed")
	}
}

// myRolesBodyThree is a GET /me/roles 2xx body carrying three roles, used by the
// compact-selector scenario to assert one line per record.
const myRolesBodyThree = `{"data":[
  {"id":"role_0000000000000000000000000000000a","name":"Lead"},
  {"id":"role_0000000000000000000000000000000b","name":"Rep"},
  {"id":"role_0000000000000000000000000000000c","name":"Treasurer"}
]}`

// outputFmtWorld is the per-scenario state for the output-format-selection suite.
// A scenario sets the resolution sources (flag/env/.glassfrogrc) and which read to
// drive in its Givens; the When both resolves the format directly (for the
// resolution-level Thens) and runs the read over a fake seam (for the command-level
// Thens) — the fake's resolveFormat uses the SAME injected sources, so the two
// stay consistent.
type outputFmtWorld struct {
	which     string // "me" (default) or "roles"
	flag      string
	env       string
	fileVal   string
	filePath  string
	fileFound bool
	fileErr   error

	transport *cannedTransport

	resolved   output.OutputFormat
	resolveErr error
	outcome    Outcome
	exitCode   int
	stdout     string
	stderr     string
}

func initializeOutputFormatScenario(sc *godog.ScenarioContext) {
	w := &outputFmtWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = outputFmtWorld{
			which:     "me",
			transport: &cannedTransport{status: 200, body: meBodyAlice},
		}
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^the authenticated read "glassfrog me" had produced a successful payload$`, w.givenMePayload)
	sc.Step(`^the read "glassfrog me" had produced a successful payload$`, w.givenMePayload)
	sc.Step(`^the read "glassfrog me roles" had produced several roles$`, w.givenRolesPayload)
	sc.Step(`^the invocation passed "--output ([^"]*)"$`, w.givenFlag)
	sc.Step(`^no --output flag, GLASSFROG_OUTPUT value, or \.glassfrogrc output value was present$`, w.givenAllAbsent)
	sc.Step(`^GLASSFROG_OUTPUT held "([^"]*)"$`, w.givenEnv)
	sc.Step(`^the \.glassfrogrc output value held "([^"]*)"$`, w.givenFile)
	sc.Step(`^no --output flag and no GLASSFROG_OUTPUT value were present$`, w.givenFlagEnvAbsent)
	sc.Step(`^the nearest \.glassfrogrc on the walk-up held output "([^"]*)"$`, w.givenFile)
	sc.Step(`^no --output flag was present$`, w.givenFlagAbsent)
	sc.Step(`^a \.glassfrogrc on the walk-up could not be read or parsed$`, w.givenUnreadableFile)

	// --- Whens (all resolve the format AND run the read, so any Then can assert) ---
	sc.Step(`^the result is rendered$`, w.whenRun)
	sc.Step(`^the command is run$`, w.whenRun)
	sc.Step(`^the format is resolved$`, w.whenRun)

	// --- Thens ---
	sc.Step(`^the result will be routed to the structured JSON encoder$`, w.thenRoutedStructuredJSON)
	sc.Step(`^stdout will carry a single JSON document of the raw payload$`, w.thenStdoutSingleJSONDoc)
	sc.Step(`^the JSON format will be selected$`, w.thenJSONSelected)
	sc.Step(`^the result will be routed to the JSON encoder exactly as lowercase "json" would$`, w.thenStdoutValidJSON)
	sc.Step(`^the command will report a usage error naming the value "([^"]*)"$`, w.thenUsageNamingValue)
	sc.Step(`^it will make no API request$`, w.thenNoRequest)
	sc.Step(`^it will exit with the usage exit code (\d+)$`, w.thenUsageExit)
	sc.Step(`^the default format full will be resolved$`, w.thenFullResolved)
	sc.Step(`^the result will be routed to the full human template$`, w.thenRoutedFull)
	sc.Step(`^json will be used$`, w.thenJSONUsed)
	sc.Step(`^no lower-precedence source will be consulted$`, w.thenNoLowerSource)
	sc.Step(`^compact will be selected from that file$`, w.thenCompactSelected)
	sc.Step(`^the result will be routed to the compact human template$`, w.thenRoutedCompact)
	sc.Step(`^the command will report a usage error naming the GLASSFROG_OUTPUT source and the value "([^"]*)"$`, w.thenUsageNamingEnvSource)
	sc.Step(`^it will not fall through to the config file or the default$`, w.thenDidNotFallThrough)
	sc.Step(`^the command will report the config read error naming the file$`, w.thenReportsConfigReadError)
	sc.Step(`^each role will appear on a single line — the rendering previously reachable from no command-line surface$`, w.thenOneLinePerRecord)
}

// --- Given implementations -------------------------------------------------

func (w *outputFmtWorld) givenMePayload() error {
	w.which = "me"
	w.transport = &cannedTransport{status: 200, body: meBodyAlice}
	return nil
}

func (w *outputFmtWorld) givenRolesPayload() error {
	w.which = "roles"
	w.transport = &cannedTransport{status: 200, body: myRolesBodyThree}
	return nil
}

func (w *outputFmtWorld) givenFlag(value string) error { w.flag = value; return nil }

func (w *outputFmtWorld) givenAllAbsent() error     { return nil } // defaults: every source absent
func (w *outputFmtWorld) givenFlagEnvAbsent() error { return nil } // flag + env absent (file may follow)
func (w *outputFmtWorld) givenFlagAbsent() error    { return nil }

func (w *outputFmtWorld) givenEnv(value string) error { w.env = value; return nil }

func (w *outputFmtWorld) givenFile(value string) error {
	w.fileVal = value
	w.filePath = "/work/.glassfrogrc"
	w.fileFound = true
	return nil
}

func (w *outputFmtWorld) givenUnreadableFile() error {
	w.fileErr = &rcfile.ReadError{Path: "/work/.glassfrogrc", Err: fmt.Errorf("permission denied")}
	return nil
}

// --- When implementation ---------------------------------------------------

// whenRun resolves the format over the injected sources (for resolution-level
// assertions) and drives the named read over a fake seam whose resolveFormat uses
// the same sources (for command-level assertions), capturing the outcome, exit
// code, and streams.
func (w *outputFmtWorld) whenRun() error {
	w.resolved, w.resolveErr = output.ResolveFormat(w.flag, w.env, w.fileVal, w.filePath, w.fileFound, w.fileErr)

	seam := &fakeMeSeam{
		ctx:        validMeContext(),
		transport:  w.transport,
		envOutput:  w.env,
		fileOutput: w.fileVal,
		filePath:   w.filePath,
		fileFound:  w.fileFound,
		fileErr:    w.fileErr,
	}
	var out, errb bytes.Buffer
	switch w.which {
	case "roles":
		w.outcome, _ = runMeRoles(meRolesConfig{seam: seam, outputFlag: w.flag, outputPresent: w.flag != "", reqCtx: context.Background(), stdout: &out, stderr: &errb})
	default:
		w.outcome, _ = runMe(meConfig{seam: seam, outputFlag: w.flag, outputPresent: w.flag != "", reqCtx: context.Background(), stdout: &out, stderr: &errb})
	}
	w.exitCode = ExitCode(w.outcome)
	w.stdout, w.stderr = out.String(), errb.String()

	// Secret-hygiene invariant: the token never appears in any produced output.
	if strings.Contains(w.stdout+w.stderr, meSecretToken) {
		return fmt.Errorf("the token leaked into output: stdout=%q stderr=%q", w.stdout, w.stderr)
	}
	return nil
}

// --- Then implementations --------------------------------------------------

func (w *outputFmtWorld) thenRoutedStructuredJSON() error {
	if w.resolveErr != nil {
		return fmt.Errorf("resolution unexpectedly failed: %v", w.resolveErr)
	}
	if w.resolved != output.FormatJSON || !w.resolved.IsStructured() {
		return fmt.Errorf("expected the structured JSON encoder, resolved %v (structured=%v)", w.resolved, w.resolved.IsStructured())
	}
	return nil
}

func (w *outputFmtWorld) thenStdoutSingleJSONDoc() error {
	if !json.Valid([]byte(w.stdout)) {
		return fmt.Errorf("stdout is not a valid JSON document:\n%s", w.stdout)
	}
	// "of the raw payload": the document carries the API's fields verbatim.
	if !strings.Contains(w.stdout, "Alice Smith") || !strings.Contains(w.stdout, "per_") {
		return fmt.Errorf("the JSON document should carry the raw payload's fields:\n%s", w.stdout)
	}
	return nil
}

func (w *outputFmtWorld) thenJSONSelected() error {
	if w.resolved != output.FormatJSON {
		return fmt.Errorf("expected the JSON format selected, resolved %v", w.resolved)
	}
	return nil
}

func (w *outputFmtWorld) thenStdoutValidJSON() error {
	if !json.Valid([]byte(w.stdout)) {
		return fmt.Errorf("uppercase --output JSON should route to the JSON encoder exactly as lowercase, but stdout is not valid JSON:\n%s", w.stdout)
	}
	return nil
}

func (w *outputFmtWorld) thenUsageNamingValue(value string) error {
	if w.outcome != UsageError {
		return fmt.Errorf("expected UsageError, got %v\nstderr: %s", w.outcome, w.stderr)
	}
	if !strings.Contains(w.stderr, value) {
		return fmt.Errorf("the usage error should name the value %q:\n%s", value, w.stderr)
	}
	return nil
}

func (w *outputFmtWorld) thenNoRequest() error {
	if w.transport.calls != 0 {
		return fmt.Errorf("no API request should be made, but the transport was called %d times", w.transport.calls)
	}
	return nil
}

func (w *outputFmtWorld) thenUsageExit(code int) error {
	if w.outcome != UsageError {
		return fmt.Errorf("expected UsageError, got %v", w.outcome)
	}
	if w.exitCode != code {
		return fmt.Errorf("expected exit code %d, got %d\nstderr: %s", code, w.exitCode, w.stderr)
	}
	return nil
}

func (w *outputFmtWorld) thenFullResolved() error {
	if w.resolveErr != nil {
		return fmt.Errorf("resolution unexpectedly failed: %v", w.resolveErr)
	}
	if w.resolved != output.FormatFull {
		return fmt.Errorf("expected the default FormatFull, resolved %v", w.resolved)
	}
	return nil
}

func (w *outputFmtWorld) thenRoutedFull() error {
	if w.resolved.IsStructured() || humanFormat(w.resolved) != render.FormatFull {
		return fmt.Errorf("expected routing to the full human template, resolved %v", w.resolved)
	}
	// The standing full projection still prints the labelled identity facts.
	if !strings.Contains(w.stdout, "actor:") {
		return fmt.Errorf("the full human template should print the identity facts:\n%s", w.stdout)
	}
	return nil
}

func (w *outputFmtWorld) thenJSONUsed() error {
	if w.resolveErr != nil {
		return fmt.Errorf("resolution unexpectedly failed: %v", w.resolveErr)
	}
	if w.resolved != output.FormatJSON {
		return fmt.Errorf("expected json to be used, resolved %v", w.resolved)
	}
	return nil
}

func (w *outputFmtWorld) thenNoLowerSource() error {
	// The flag short-circuits resolution: with the flag present and valid, no rcfile
	// or default value is consulted, so resolution yields the flag's format without
	// error even though env ("yaml") and file ("compact") hold other (valid) values.
	if w.resolveErr != nil {
		return fmt.Errorf("the flag should win without error, got %v", w.resolveErr)
	}
	if w.resolved != output.FormatJSON {
		return fmt.Errorf("the flag value should win over env and file, resolved %v", w.resolved)
	}
	return nil
}

func (w *outputFmtWorld) thenCompactSelected() error {
	if w.resolveErr != nil {
		return fmt.Errorf("resolution unexpectedly failed: %v", w.resolveErr)
	}
	if w.resolved != output.FormatCompact {
		return fmt.Errorf("expected compact selected from the file, resolved %v", w.resolved)
	}
	return nil
}

func (w *outputFmtWorld) thenRoutedCompact() error {
	if w.resolved != output.FormatCompact || w.resolved.IsStructured() || humanFormat(w.resolved) != render.FormatCompact {
		return fmt.Errorf("expected routing to the compact human template, resolved %v", w.resolved)
	}
	return nil
}

func (w *outputFmtWorld) thenUsageNamingEnvSource(value string) error {
	if w.outcome != UsageError {
		return fmt.Errorf("expected UsageError, got %v\nstderr: %s", w.outcome, w.stderr)
	}
	if !strings.Contains(w.stderr, output.EnvVarOutput) {
		return fmt.Errorf("the usage error should name the %s source:\n%s", output.EnvVarOutput, w.stderr)
	}
	if !strings.Contains(w.stderr, value) {
		return fmt.Errorf("the usage error should name the value %q:\n%s", value, w.stderr)
	}
	return nil
}

func (w *outputFmtWorld) thenDidNotFallThrough() error {
	// A present-but-invalid env value must surface as the usage error rather than
	// falling through to the config file or the default (which would have yielded a
	// valid format and a successful run).
	if w.outcome != UsageError {
		return fmt.Errorf("a present-but-invalid env value must not fall through; got outcome %v", w.outcome)
	}
	if w.transport.calls != 0 {
		return fmt.Errorf("a fail-fast usage error should make no request, but the transport was called %d times", w.transport.calls)
	}
	return nil
}

func (w *outputFmtWorld) thenReportsConfigReadError() error {
	if w.outcome != UsageError {
		return fmt.Errorf("an unreadable config should fail as a usage error, got %v", w.outcome)
	}
	if !strings.Contains(w.stderr, ".glassfrogrc") {
		return fmt.Errorf("the config read error should name the file:\n%s", w.stderr)
	}
	return nil
}

func (w *outputFmtWorld) thenOneLinePerRecord() error {
	lines := nonEmptyLines(w.stdout)
	if len(lines) != 3 {
		return fmt.Errorf("compact should render one line per role (3), got %d:\n%s", len(lines), w.stdout)
	}
	return nil
}
