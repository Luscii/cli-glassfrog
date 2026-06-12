package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/render"
)

// fakeExecutor decodes a canned 2xx body into the dispatch's chosen target (a
// *json.RawMessage for the structured path, a typed *T for the human path), so
// renderResult's decode-target selection and renderer routing are exercised without
// a real transport. It records the concrete out type the dispatch passed.
type fakeExecutor struct {
	body    string
	outType string // "json.RawMessage" or the typed struct, recorded for routing assertions
}

func (f *fakeExecutor) Execute(_ context.Context, _ apiclient.Request, out any) (*apiclient.Response, error) {
	switch out.(type) {
	case *json.RawMessage:
		f.outType = "raw"
	default:
		f.outType = "typed"
	}
	if err := json.Unmarshal([]byte(f.body), out); err != nil {
		return nil, err // not hit for the valid canned bodies these tests use
	}
	return &apiclient.Response{StatusCode: 200}, nil
}

// TestRenderResult_StructuredRoutesToOutputEncoder pins that json/yaml decode the
// verbatim bytes and route through 018's encoder (ADR-3): stdout is a single
// machine document carrying the raw payload, and the dispatch decoded into a
// json.RawMessage (not the typed struct).
func TestRenderResult_StructuredRoutesToOutputEncoder(t *testing.T) {
	for _, tc := range []struct {
		format  output.OutputFormat
		wantSub string // a substring proving the machine encoding (not the human projection)
		isJSON  bool
	}{
		{output.FormatJSON, "Alice Smith", true},
		{output.FormatYAML, "name: Alice Smith", false},
	} {
		t.Run(tc.format.String(), func(t *testing.T) {
			exec := &fakeExecutor{body: meBodyAlice}
			var out, errb bytes.Buffer
			outcome, err := renderResult[glassfrog.MeResponse](
				&out, &errb, tc.format, nil, render.ResourceMe, exec, context.Background(),
				apiclient.Request{}, nil,
			)
			if outcome != Success || err != nil {
				t.Fatalf("outcome=%v err=%v, want Success/nil; stderr=%s", outcome, err, errb.String())
			}
			if exec.outType != "raw" {
				t.Errorf("structured path should decode into json.RawMessage, decoded into %q", exec.outType)
			}
			if tc.isJSON && !json.Valid(out.Bytes()) {
				t.Errorf("json output is not valid:\n%s", out.String())
			}
			if !strings.Contains(out.String(), tc.wantSub) {
				t.Errorf("%v output should carry the raw payload (%q):\n%s", tc.format, tc.wantSub, out.String())
			}
			// The structured document must NOT be the labelled human projection.
			if strings.Contains(out.String(), "actor:        Alice Smith") {
				t.Errorf("%v routed to the human renderer instead of the encoder:\n%s", tc.format, out.String())
			}
		})
	}
}

// TestRenderResult_HumanRoutesToRenderTemplates pins that full/compact decode the
// typed struct and route through 019's templates — the labelled projection, not a
// machine document.
func TestRenderResult_HumanRoutesToRenderTemplates(t *testing.T) {
	for _, format := range []output.OutputFormat{output.FormatFull, output.FormatCompact} {
		t.Run(format.String(), func(t *testing.T) {
			exec := &fakeExecutor{body: meBodyAlice}
			var out, errb bytes.Buffer
			outcome, err := renderResult[glassfrog.MeResponse](
				&out, &errb, format, nil, render.ResourceMe, exec, context.Background(),
				apiclient.Request{}, nil,
			)
			if outcome != Success || err != nil {
				t.Fatalf("outcome=%v err=%v, want Success/nil; stderr=%s", outcome, err, errb.String())
			}
			if exec.outType != "typed" {
				t.Errorf("human path should decode into the typed struct, decoded into %q", exec.outType)
			}
			if !strings.Contains(out.String(), "Alice Smith") || !strings.Contains(out.String(), "per_") {
				t.Errorf("%v output should render the human projection:\n%s", format, out.String())
			}
			// The human projection must NOT be a JSON document.
			if json.Valid(out.Bytes()) {
				t.Errorf("%v routed to the structured encoder instead of the template:\n%s", format, out.String())
			}
		})
	}
}

// TestRenderResult_HumanRenderFailureExitsOneNoPartialStdout pins the
// buffer-then-write contract (ADR-3 / 019 ADR-4): a render failure leaves stdout
// empty and maps to RuntimeError(1).
func TestRenderResult_HumanRenderFailureExitsOneNoPartialStdout(t *testing.T) {
	orig := renderFn
	defer func() { renderFn = orig }()
	renderFn = func(render.Resource, render.Format, any) (string, error) {
		return "partial output that must not reach stdout", &render.RenderError{
			Resource: render.ResourceMe, Format: render.FormatFull, Err: errors.New("template defect"),
		}
	}

	exec := &fakeExecutor{body: meBodyAlice}
	var out, errb bytes.Buffer
	outcome, err := renderResult[glassfrog.MeResponse](
		&out, &errb, output.FormatFull, nil, render.ResourceMe, exec, context.Background(),
		apiclient.Request{}, nil,
	)
	if outcome != RuntimeError {
		t.Fatalf("outcome=%v, want RuntimeError", outcome)
	}
	if ExitCode(outcome) != 1 {
		t.Errorf("exit code = %d, want 1", ExitCode(outcome))
	}
	if err == nil {
		t.Error("a render failure should return the error")
	}
	if out.Len() != 0 {
		t.Errorf("a render failure must leave stdout empty, got:\n%s", out.String())
	}
}

// TestClassifyClientError_FormatErrorIsUsage pins the T004 arm: a present-but-invalid
// selector maps to UsageError(2), symmetric with the base-URL arms, with no new exit
// code introduced.
func TestClassifyClientError_FormatErrorIsUsage(t *testing.T) {
	err := &output.FormatError{Source: "--output", Value: "xml"}
	if got := classifyClientError(err); got != UsageError {
		t.Fatalf("classifyClientError(*output.FormatError) = %v, want UsageError", got)
	}
	if ExitCode(UsageError) != 2 {
		t.Errorf("UsageError exit code = %d, want 2", ExitCode(UsageError))
	}
}
