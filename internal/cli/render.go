package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/render"
)

// renderFn is the human-render seam (render.Render by default). It is a package
// var so a test can override it to exercise the buffer-then-write failure path —
// a built-in-template defect — without contriving a result struct that fails to
// render (the typed result structs always carry every field their template reads,
// so render never fails for real read output). The render-dispatch routes the
// human formats (full/compact) through it; the structured formats (json/yaml) go
// to output.RenderSuccess instead.
var renderFn = render.Render

// executor is the minimal send surface the render-dispatch needs: a bare
// *apiclient.Client (me roles / me actions / me projects) or a 017
// *apiclient.RetryExecutor (me). Both decode the 2xx body into out — a
// *json.RawMessage (structured) or a typed *T (human) — so the dispatch alone picks
// the decode target from the resolved format (018 ADR-2).
type executor interface {
	Execute(reqCtx context.Context, req apiclient.Request, out any) (*apiclient.Response, error)
}

// humanFormat maps the human members of output.OutputFormat onto render.Format —
// the single site translating the 020 selection vocabulary to 019's template-engine
// names (ADR-3), so internal/output and internal/render stay non-importing siblings.
// The structured members never reach here: the dispatch routes them to
// output.RenderSuccess via MachineFormat before consulting a human renderer.
func humanFormat(f output.OutputFormat) render.Format {
	if f == output.FormatCompact {
		return render.FormatCompact
	}
	return render.FormatFull
}

// renderResult is the shared success dispatch (020 ADR-3) — the only site that
// imports both internal/output (machine) and internal/render (human). Given the
// resolved format it selects the decode target and the renderer, sends req through
// exec, and writes the rendered document to stdout:
//
//   - structured (json/yaml): decode the verbatim 2xx body into a json.RawMessage
//     and write output.RenderSuccess(machineFmt, raw), preserving every field and
//     number exactly (018 fidelity). The structured document carries the API's own
//     pagination metadata in-band, so the human incompleteness note is not emitted
//     on this path — selection shapes presentation, and machine consumers read
//     completeness from the document itself.
//   - human (full/compact): decode the typed *T and write render.Render(resource,
//     humanFmt, v) (019), then append the optional incompleteness note to stderr.
//
// A client/transport/API error from the send is reported via the shared
// reportClientError (unchanged category and cause-plus-next-step message; 032 owns
// format-aware failures). A render error from either renderer is buffer-then-write:
// nothing reaches stdout and it maps to RuntimeError(1) (018/019 contract). On
// success it returns Success after writing the document and any note. note may be
// nil (a read with no incompleteness signal, e.g. me).
func renderResult[T any](
	stdout, stderr io.Writer,
	format output.OutputFormat,
	resource render.Resource,
	exec executor,
	reqCtx context.Context,
	req apiclient.Request,
	note func(T) string,
) (Outcome, error) {
	if machineFmt, ok := format.MachineFormat(); ok {
		var raw json.RawMessage
		if _, err := exec.Execute(reqCtx, req, &raw); err != nil {
			return reportClientError(stderr, err)
		}
		doc, rerr := output.RenderSuccess(machineFmt, raw)
		if rerr != nil {
			// Buffer-then-write: a render failure leaves stdout empty and maps to
			// RuntimeError(1). The error is token-free (018 contract).
			fmt.Fprintln(stderr, rerr.Error())
			return RuntimeError, rerr
		}
		_, _ = stdout.Write(doc)
		return Success, nil
	}

	var v T
	if _, err := exec.Execute(reqCtx, req, &v); err != nil {
		return reportClientError(stderr, err)
	}
	text, rerr := renderFn(resource, humanFormat(format), v)
	if rerr != nil {
		// Buffer-then-write: a built-in-template defect leaves stdout empty and maps
		// to RuntimeError(1) (019 ADR-4).
		fmt.Fprintln(stderr, rerr.Error())
		return RuntimeError, rerr
	}
	fmt.Fprint(stdout, text)
	if note != nil {
		if msg := note(v); msg != "" {
			fmt.Fprintln(stderr, msg)
		}
	}
	return Success, nil
}
