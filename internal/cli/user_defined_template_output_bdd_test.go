package cli

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestUserDefinedTemplateOutputFeatures runs the executable acceptance for
// User-Defined Template Output (035): the --output flag widened, at the flag rung,
// to a template file path or "stdin", driven through the reads over a fake seam that
// resolves the selection and reads the template source from injected content (no real
// filesystem or os.Stdin) — so every scenario runs offline and hermetically. Its
// Paths name ONLY this spec's feature file, so un-@wip-ping these scenarios cannot
// disturb another suite. The three @validation scenarios stay @wip (held for the
// validate skill) and are skipped by the ~@wip filter.
func TestUserDefinedTemplateOutputFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeUserTemplateScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unconsumable-output/user-defined-template-output.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: user-defined-template-output feature scenarios failed")
	}
}

// udtoWorld is the per-scenario state for the user-defined-template-output suite. A
// scenario sets which read to drive, the -o flag value, and the template source
// (file content via tmplFiles, or piped stdin) in its Givens; the When runs the read
// over a fake seam wired to those sources and captures the outcome, exit code, and
// streams.
type udtoWorld struct {
	which     string // "me" (default) or "roles"
	flag      string
	source    string // operator-facing source label the error is expected to name
	transport *cannedTransport

	tmplFiles      map[string]string
	tmplStdin      string
	tmplStdinPiped bool

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeUserTemplateScenario(sc *godog.ScenarioContext) {
	w := &udtoWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = udtoWorld{
			which:     "me",
			transport: &cannedTransport{status: 200, body: meBodyAlice},
			tmplFiles: map[string]string{},
		}
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^the read "glassfrog me roles" had produced several roles$`, w.givenRolesPayload)
	sc.Step(`^the invocation passed "-o ([^"]*)" naming a readable, parseable template$`, w.givenReadableTemplate)
	sc.Step(`^the invocation passed "-o ([^"]*)" naming a file that does not exist$`, w.givenMissingFile)
	sc.Step(`^the invocation passed "-o ([^"]*)" naming a file whose template cannot be parsed$`, w.givenUnparseableFile)
	sc.Step(`^the read had produced a result that omitted an embedded collection$`, w.givenMeNoRoles)
	sc.Step(`^a template that guards a reference to that collection$`, w.givenGuardedTemplate)
	sc.Step(`^the read had produced a successful result$`, w.givenMeSuccess)
	sc.Step(`^a template that references an absent field without guarding it$`, w.givenUnguardedTemplate)
	sc.Step(`^a template had been piped to the command on standard input$`, w.givenPipedTemplate)
	sc.Step(`^the invocation passed "-o stdin" to a successful "glassfrog me" read$`, w.givenStdinToMe)
	sc.Step(`^no template had been piped to standard input$`, w.givenNoPipe)
	sc.Step(`^a file named "([^"]*)" existed in the current working directory$`, w.givenSameNamedFile)
	sc.Step(`^the invocation passed "-o ([^"]*)"$`, w.givenFlag)

	// --- Whens ---
	sc.Step(`^the result is rendered$`, w.whenRun)
	sc.Step(`^the command is run$`, w.whenRun)

	// --- Thens ---
	sc.Step(`^the roles' data will be rendered through that template$`, w.thenRolesRenderedThroughTemplate)
	sc.Step(`^stdout will carry the template's output$`, w.thenStdoutCarriesTemplateOutput)
	sc.Step(`^the command will report a usage error naming the file$`, w.thenUsageNamingSource)
	sc.Step(`^the command will report a usage error naming the source$`, w.thenUsageNamingSource)
	sc.Step(`^the command will report a usage error$`, w.thenUsageError)
	sc.Step(`^it will make no API request$`, w.thenNoRequest)
	sc.Step(`^it will exit with the usage exit code (\d+)$`, w.thenUsageExit)
	sc.Step(`^an explicit absence marker will appear where the field would be$`, w.thenAbsenceMarker)
	sc.Step(`^no fabricated data value will stand in for data the API did not return$`, w.thenNoFabrication)
	sc.Step(`^the template will be read from standard input$`, w.thenRenderedFromStdin)
	sc.Step(`^the "me" result's data will be rendered through it$`, w.thenMeRenderedThroughTemplate)
	sc.Step(`^stdout will carry nothing$`, w.thenStdoutEmpty)
	sc.Step(`^the built-in full template will be selected$`, w.thenBuiltinFullSelected)
	sc.Step(`^the file named "([^"]*)" will not be read$`, w.thenFileNotRead)
}

const sameNamedFileSentinel = "SHOULD-NOT-BE-READ"

// --- Given implementations -------------------------------------------------

func (w *udtoWorld) givenRolesPayload() error {
	w.which = "roles"
	w.transport = &cannedTransport{status: 200, body: myRolesBodyThree}
	return nil
}

func (w *udtoWorld) givenReadableTemplate(path string) error {
	w.flag, w.source = path, path
	// A template over the MyRolesResponse value (one role name per line).
	w.tmplFiles[path] = "{{range .Data}}{{.Name}}\n{{end}}"
	return nil
}

func (w *udtoWorld) givenMissingFile(path string) error {
	w.flag, w.source = path, path // not registered in tmplFiles → reads as not-found
	return nil
}

func (w *udtoWorld) givenUnparseableFile(path string) error {
	w.flag, w.source = path, path
	w.tmplFiles[path] = "{{.Unclosed" // a syntax error
	return nil
}

func (w *udtoWorld) givenMeNoRoles() error {
	w.which = "me"
	w.transport = &cannedTransport{status: 200, body: meBodyAlice} // no roles embed
	return nil
}

func (w *udtoWorld) givenGuardedTemplate() error {
	w.flag, w.source = "./guard.tmpl", "./guard.tmpl"
	w.tmplFiles[w.flag] = "{{if .Roles}}{{range .Roles}}{{.Name}} {{end}}{{else}}(no roles){{end}}"
	return nil
}

func (w *udtoWorld) givenMeSuccess() error {
	w.which = "me"
	w.transport = &cannedTransport{status: 200, body: meBodyAlice}
	return nil
}

func (w *udtoWorld) givenUnguardedTemplate() error {
	w.flag, w.source = "./exec-fail.tmpl", "./exec-fail.tmpl"
	w.tmplFiles[w.flag] = "{{.NoSuchField}}" // unguarded absent struct field → execution error
	return nil
}

func (w *udtoWorld) givenPipedTemplate() error {
	w.tmplStdinPiped = true
	w.tmplStdin = "{{.Actor.Name}}"
	return nil
}

func (w *udtoWorld) givenStdinToMe() error {
	w.which = "me"
	w.flag, w.source = "stdin", "stdin"
	w.transport = &cannedTransport{status: 200, body: meBodyAlice}
	return nil
}

func (w *udtoWorld) givenNoPipe() error {
	w.tmplStdinPiped = false
	w.flag, w.source = "stdin", "stdin"
	return nil
}

func (w *udtoWorld) givenSameNamedFile(name string) error {
	w.tmplFiles[name] = sameNamedFileSentinel
	return nil
}

func (w *udtoWorld) givenFlag(value string) error {
	w.flag = value
	w.source = value
	return nil
}

// --- When implementation ---------------------------------------------------

func (w *udtoWorld) whenRun() error {
	seam := &fakeMeSeam{
		ctx:            validMeContext(),
		transport:      w.transport,
		tmplFiles:      w.tmplFiles,
		tmplStdin:      w.tmplStdin,
		tmplStdinPiped: w.tmplStdinPiped,
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

func (w *udtoWorld) thenRolesRenderedThroughTemplate() error {
	if w.outcome != Success {
		return fmt.Errorf("expected Success, got %v\nstderr: %s", w.outcome, w.stderr)
	}
	// The injected template renders one role NAME per line; the three roles in
	// myRolesBodyThree must all appear, and the built-in projection's labels must not.
	for _, name := range []string{"Lead", "Rep", "Treasurer"} {
		if !strings.Contains(w.stdout, name) {
			return fmt.Errorf("the template output should carry the role %q:\n%s", name, w.stdout)
		}
	}
	if strings.Contains(w.stdout, "role_") {
		return fmt.Errorf("the template emitted names only — the built-in projection (ids) leaked:\n%s", w.stdout)
	}
	return nil
}

func (w *udtoWorld) thenStdoutCarriesTemplateOutput() error {
	if strings.TrimSpace(w.stdout) == "" {
		return fmt.Errorf("stdout should carry the template's output, got empty")
	}
	return nil
}

func (w *udtoWorld) thenUsageNamingSource() error {
	if w.outcome != UsageError {
		return fmt.Errorf("expected UsageError, got %v\nstderr: %s", w.outcome, w.stderr)
	}
	if !strings.Contains(w.stderr, w.source) {
		return fmt.Errorf("the usage error should name the source %q:\n%s", w.source, w.stderr)
	}
	return nil
}

func (w *udtoWorld) thenUsageError() error {
	if w.outcome != UsageError {
		return fmt.Errorf("expected UsageError, got %v\nstderr: %s", w.outcome, w.stderr)
	}
	return nil
}

func (w *udtoWorld) thenNoRequest() error {
	if w.transport.calls != 0 {
		return fmt.Errorf("no API request should be made, but the transport was called %d times", w.transport.calls)
	}
	return nil
}

func (w *udtoWorld) thenUsageExit(code int) error {
	if w.outcome != UsageError {
		return fmt.Errorf("expected UsageError, got %v", w.outcome)
	}
	if w.exitCode != code {
		return fmt.Errorf("expected exit code %d, got %d\nstderr: %s", code, w.exitCode, w.stderr)
	}
	return nil
}

func (w *udtoWorld) thenAbsenceMarker() error {
	if w.outcome != Success {
		return fmt.Errorf("a guarded absence should render successfully, got %v\nstderr: %s", w.outcome, w.stderr)
	}
	if !strings.Contains(w.stdout, "(no roles)") {
		return fmt.Errorf("the guarded template should render its explicit-absence marker:\n%s", w.stdout)
	}
	return nil
}

func (w *udtoWorld) thenNoFabrication() error {
	// The result carried no roles; the marker stands in, and no role name is invented.
	if strings.Contains(w.stdout, "role_") {
		return fmt.Errorf("a data value was fabricated for data the API did not return:\n%s", w.stdout)
	}
	return nil
}

func (w *udtoWorld) thenRenderedFromStdin() error {
	if w.outcome != Success {
		return fmt.Errorf("expected Success rendering from stdin, got %v\nstderr: %s", w.outcome, w.stderr)
	}
	// The piped template was {{.Actor.Name}} — its output proves stdin was the source.
	if !strings.Contains(w.stdout, "Alice Smith") {
		return fmt.Errorf("stdout should carry the piped template's output:\n%s", w.stdout)
	}
	return nil
}

func (w *udtoWorld) thenMeRenderedThroughTemplate() error {
	if !strings.Contains(w.stdout, "Alice Smith") {
		return fmt.Errorf("the me result should be rendered through the template:\n%s", w.stdout)
	}
	// The template emitted only the actor name, not the built-in labelled projection.
	if strings.Contains(w.stdout, "actor:") {
		return fmt.Errorf("the built-in projection leaked instead of the template output:\n%s", w.stdout)
	}
	return nil
}

func (w *udtoWorld) thenStdoutEmpty() error {
	if w.stdout != "" {
		return fmt.Errorf("an execution failure must leave stdout empty (buffer-then-write), got:\n%s", w.stdout)
	}
	return nil
}

func (w *udtoWorld) thenBuiltinFullSelected() error {
	if w.outcome != Success {
		return fmt.Errorf("expected Success for the built-in full path, got %v\nstderr: %s", w.outcome, w.stderr)
	}
	// The built-in full me projection prints the labelled identity facts.
	if !strings.Contains(w.stdout, "actor:") {
		return fmt.Errorf("the built-in full template should render the labelled projection:\n%s", w.stdout)
	}
	return nil
}

func (w *udtoWorld) thenFileNotRead(name string) error {
	if strings.Contains(w.stdout+w.stderr, sameNamedFileSentinel) {
		return fmt.Errorf("the same-named file %q was read instead of the reserved format:\n%s", name, w.stdout+w.stderr)
	}
	return nil
}
