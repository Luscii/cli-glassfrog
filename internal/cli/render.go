package cli

import (
	"fmt"
	"io"

	"github.com/Luscii/cli-glassfrog/internal/render"
)

// renderFn is the human-render seam (render.Render by default). It is a package
// var so a test can override it to exercise the buffer-then-write failure path —
// a built-in-template defect — without contriving a result struct that fails to
// render (the typed result structs always carry every field their template
// reads, so render never fails for real read output). 020 Output Format
// Selection replaces the hardcoded FormatFull at the call sites; until then full
// is the only format the reads select.
var renderFn = render.Render

// renderResult renders a read's decoded result through the human-render seam with
// the standing full format (019) and writes the text to stdout on success. A
// render failure is an internal built-in-template defect (ADR-4): nothing is
// written to stdout (buffer-then-write — no partial output), the token-free error
// goes to stderr, and the command maps the RuntimeError outcome to exit code 1
// through the existing Outcome→ExitCode registry. On success it returns Success
// and the caller appends any post-render note (e.g. a list read's incompleteness
// note) itself.
func renderResult(stdout, stderr io.Writer, resource render.Resource, data any) (Outcome, error) {
	out, err := renderFn(resource, render.FormatFull, data)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return RuntimeError, err
	}
	fmt.Fprint(stdout, out)
	return Success, nil
}
