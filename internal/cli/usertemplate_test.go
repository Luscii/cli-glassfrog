package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/render"
)

// TestClassifyClientError_UserTemplateError pins T003: both stages of a
// *render.UserTemplateError classify as UsageError (symmetric with the
// *output.FormatError arm), distinct from a built-in *render.RenderError which stays
// a RuntimeError code defect.
func TestClassifyClientError_UserTemplateError(t *testing.T) {
	for _, stage := range []render.UserTemplateStage{render.StageParse, render.StageExecute} {
		err := &render.UserTemplateError{Stage: stage, Source: "./t.tmpl", Err: errors.New("boom")}
		if got := classifyClientError(err); got != UsageError {
			t.Errorf("stage %v: classifyClientError = %v, want UsageError", stage, got)
		}
	}
	// A built-in render defect stays RuntimeError (the fail-safe), unchanged by 035.
	re := &render.RenderError{Resource: render.ResourceMe, Format: render.FormatFull, Err: errors.New("defect")}
	if got := classifyClientError(re); got != RuntimeError {
		t.Errorf("a built-in *RenderError should stay RuntimeError, got %v", got)
	}
}

// TestWriteHuman_UserTemplateExecuteError pins the writeHuman boundary used by the
// dispatch (T004): a user-template execution failure leaves stdout empty
// (buffer-then-write) and maps to UsageError, while a built-in defect maps to
// RuntimeError.
func TestWriteHuman_UserTemplateExecuteError(t *testing.T) {
	tmpl, err := render.ParseUserTemplate("{{.NoSuchField}}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tmpl.Source = "./t.tmpl"
	var out, errb bytes.Buffer
	outcome, rerr := writeHuman(&out, &errb, tmpl, render.ResourceMe, output.DefaultFormat, struct{}{})
	if outcome != UsageError {
		t.Errorf("a user-template execution failure should be UsageError, got %v", outcome)
	}
	if rerr == nil {
		t.Error("writeHuman should return the execution error")
	}
	if out.String() != "" {
		t.Errorf("buffer-then-write: stdout must stay empty, got %q", out.String())
	}
	if !bytesContains(errb.Bytes(), "./t.tmpl") {
		t.Errorf("the stderr message should name the source, got %q", errb.String())
	}
}

// TestReportTemplateError_IsUsage pins that the pre-request template-source reporter
// returns UsageError (a missing file / unparseable template / empty stdin), the
// known-usage fail-fast class.
func TestReportTemplateError_IsUsage(t *testing.T) {
	var errb bytes.Buffer
	outcome, err := reportTemplateError(&errb, errors.New("could not read template file \"x\""))
	if outcome != UsageError {
		t.Errorf("reportTemplateError should return UsageError, got %v", outcome)
	}
	if err == nil || errb.Len() == 0 {
		t.Error("reportTemplateError should write the error to stderr and return it")
	}
}

// TestReadTemplateSourceFrom_StdinOverflowNamesTemplate pins that the `-o stdin`
// path's overflow error names "template", not "token" (the bounded reader is shared
// with the auth token path but parameterized per-caller).
func TestReadTemplateSourceFrom_StdinOverflowNamesTemplate(t *testing.T) {
	big := strings.NewReader(strings.Repeat("x", maxPipedTemplateBytes+1))
	ref := output.TemplateRef{Kind: output.TemplateStdin}
	_, err := readTemplateSourceFrom(ref, nil, false, big)
	if err == nil {
		t.Fatal("an over-cap piped template should error")
	}
	if !strings.Contains(err.Error(), "template") || strings.Contains(err.Error(), "token") {
		t.Errorf("the overflow error should name the template, not a token, got %q", err.Error())
	}
}

func bytesContains(b []byte, sub string) bool {
	return bytes.Contains(b, []byte(sub))
}
