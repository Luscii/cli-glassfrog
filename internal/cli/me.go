package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/render"
	"github.com/spf13/cobra"
)

// supportedIncludeTargets is the spec's GET /me ?include set — today just
// "roles" (the embed that lets `me` return identity + roles in one read). An
// unsupported target is rejected before any request (validateInclude). Adding a
// target tracks the spec's include enum (a one-line change; the planning risk).
var supportedIncludeTargets = map[string]bool{
	"roles": true,
}

// meSeam supplies everything the me command reads from the outside, so runMe is
// pure over injected values and every branch runs offline (ADR-5, the loginSeam
// shape). Production binds apiclient.AssembleFromOS + apiclient.NewClientFromOS
// (the real base transport); tests bind a fake transport returning canned
// GET /me responses. me never reads ctx.Cred.Token — the token rides 007's
// AuthTransport, wired inside the client.
type meSeam interface {
	// assemble resolves the ConnectionContext once from the --base-url flag value
	// (009; resolution already happened, identity + endpoint are stable).
	assemble(baseURL string) apiclient.ConnectionContext
	// newClient builds the request client once over the assembled context (010).
	// A context with no usable endpoint returns its base-URL error verbatim.
	newClient(ctx apiclient.ConnectionContext) (*apiclient.Client, error)
	// sleep is the backoff clock Rate-Limit Handling (017) waits on between bounded
	// 429 retries: production binds time.Sleep; tests bind a recording fake that
	// never blocks, so the retry caps are asserted in milliseconds (CONSTITUTION
	// IV). Injected here alongside the other OS-touching resolvers.
	sleep() func(time.Duration)
	// resolveSelection resolves the effective output SELECTION (020 widened by 035,
	// ADR-1) from the inherited --output flag value plus the real environment and
	// .glassfrogrc walk: a built-in format, or — only at the flag rung — a
	// user-template source (file path / stdin). Production binds
	// output.ResolveSelectionFromOS over os.Getwd/os.UserHomeDir; tests bind a fake
	// over hand-built sources. A present-but-invalid value returns a
	// *output.FormatError (or an internal/rcfile read/format error) the read maps to
	// UsageError(2). readTemplateSource reads a selected template's bytes behind the
	// seam (file via os.ReadFile, stdin via the bounded isTTY-guarded reader, 035
	// ADR-4) — both methods declared so the shared selectionSeam is satisfied.
	resolveSelection(flagValue string) (output.Selection, error)
	readTemplateSource(ref output.TemplateRef) (string, error)
}

// productionSeam (defined for loginSeam in authlogin_seam.go) also satisfies
// meSeam by binding the real OS resolvers and the real base transport. One
// MustRegister(root, newMeCommand(productionSeam{})) line in Assemble wires it.
func (productionSeam) assemble(baseURL string) apiclient.ConnectionContext {
	return apiclient.AssembleFromOS(baseURL)
}

func (productionSeam) newClient(ctx apiclient.ConnectionContext) (*apiclient.Client, error) {
	return apiclient.NewClientFromOS(ctx)
}

// sleep binds the real clock for 017's backoff between retries (ADR-4). Tests
// bind a recording fake via their own seam, so no suite ever waits real seconds.
func (productionSeam) sleep() func(time.Duration) { return time.Sleep }

// productionSeam.resolveSelection / .readTemplateSource (the 035 widening of 020's
// resolveFormat) live in usertemplate.go alongside the shared selectionSeam and the
// read helpers.

// meConfig carries everything runMe needs, gathered by the command's RunE.
// Keeping runMe a function of injected values (not the seam alone) makes the
// whole read — validate, assemble, build, send, render/classify — testable over
// a fake transport with no real network or ~/.glassfrogrc.
type meConfig struct {
	seam       meSeam
	baseURL    string   // the persistent --base-url flag value (may be empty)
	outputFlag string   // the persistent --output flag value (may be empty), resolved before any request
	include    []string // the raw --include targets, validated before any request
	reqCtx     context.Context
	stdout     io.Writer
	stderr     io.Writer
}

// runMe is the pure orchestration the command delegates to (ADR-5): validate the
// include targets fail-fast, assemble the context once, build the client once,
// send GET /me through the retry executor, and render the projection on success
// or classify+report the typed error otherwise. It returns the code-free Outcome
// the command maps onto dispatch's error channel; it never emits an exit code and
// never reads the token — the projection renders response-side fields only. The
// send routes through 017's RetryExecutor, so a transient 429 is ridden out within
// bounded caps; a capped-out 429 surfaces as the unchanged *ResponseError. 017 adds
// no classification of its own (ADR-5); now that API Error Extraction (015) has
// landed, the shared classifyClientError maps that surfaced 429 to RateLimited(5).
func runMe(cfg meConfig) (Outcome, error) {
	// 1. Resolve the render target FIRST (020 widened by 035, ADR-1/ADR-4): a
	//    present-but-invalid selector — or, when a user template is selected, a
	//    missing file / unparseable template / empty stdin — fails fast as a usage
	//    error before any assembly or request (no wasted call, pinned by a tripwire
	//    transport). Resolution is independent of connection assembly (009).
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	// 2. Validate --include BEFORE any assembly or request (fail-fast usage
	//    error, no wasted call — pinned by a tripwire transport in the tests).
	if err := validateInclude(cfg.include); err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}
	includeRoles := wantsInclude(cfg.include, "roles")

	// 3. Resolve the connection and build the client once. A base-URL error
	//    surfaces here (no doomed send); classify + report it.
	ctx := cfg.seam.assemble(cfg.baseURL)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}

	// 4. Build GET /me and the 017 retry executor, then dispatch on the resolved
	//    target (020 ADR-3 / 035): the dispatch picks the decode target (raw bytes for
	//    json/yaml, the typed projection for full/compact or a user template) and the
	//    renderer, sends through the executor, and writes to stdout. include=roles is
	//    added only when requested. A transient 429 is ridden out within bounded caps;
	//    a capped-out 429 surfaces unchanged (017 ADR-5) and 015's classifyClientError
	//    maps it to RateLimited(5). The seam renders only response-side fields, so the
	//    token never appears. me carries no incompleteness signal, so no note.
	req := apiclient.Request{Method: http.MethodGet, Path: "/me"}
	if includeRoles {
		req.Query = url.Values{"include": []string{"roles"}}
	}
	exec := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, cfg.seam.sleep(), cfg.stderr)
	return renderResult[glassfrog.MeResponse](cfg.stdout, cfg.stderr, rt.format, rt.tmpl, render.ResourceMe, exec, cfg.reqCtx, req, nil)
}

// reportFormatResolutionError writes a resolved-format failure to stderr and returns
// the Outcome the shared classifier assigns to it. The two expected causes —
// an invalid selector (*output.FormatError) and an unreadable/malformed .glassfrogrc
// surfaced while reading the output key (*rcfile.{Read,Format}Error) — both classify
// as UsageError(2) (correctable input). Any other resolution error (e.g. a
// working-directory failure from productionSeam.resolveSelection's os.Getwd) matches no
// usage-class arm and falls to classifyClientError's RuntimeError(1) fail-safe — the
// correct treatment for an unexpected internal failure, not a usage error. The
// message is the error's own text (a FormatError names the source + value with the
// supported list; an rcfile error names the file), NOT routed through
// formatClientErrorMessage, whose base-URL wording would misdescribe an output-key
// rcfile error. Category and message derive from the same value (the
// classifyClientError contract).
func reportFormatResolutionError(stderr io.Writer, err error) (Outcome, error) {
	fmt.Fprintln(stderr, err.Error())
	return classifyClientError(err), err
}

// validateInclude rejects unsupported --include targets against the spec's
// include set, returning a usage error NAMING each offending target — before any
// context assembly or request (the 002 invalid-input convention). An empty list
// (the flag absent) is valid. --include is a string slice, so more than one
// unsupported target can be supplied at once; each is quoted individually and the
// noun agrees in number, so a multi-target rejection reads cleanly rather than
// quoting the whole comma-joined list as a single bogus target. Targets are
// reported in stable (sorted) order so the message is deterministic.
func validateInclude(targets []string) error {
	var unsupported []string
	for _, t := range targets {
		if !supportedIncludeTargets[t] {
			unsupported = append(unsupported, t)
		}
	}
	if len(unsupported) == 0 {
		return nil
	}
	sort.Strings(unsupported)
	quoted := make([]string, len(unsupported))
	for i, t := range unsupported {
		quoted[i] = fmt.Sprintf("%q", t)
	}
	noun := "target"
	if len(unsupported) > 1 {
		noun = "targets"
	}
	return fmt.Errorf(
		"unsupported --include %s %s — supported: %s",
		noun, strings.Join(quoted, ", "), strings.Join(supportedIncludeNames(), ", "),
	)
}

// supportedIncludeNames lists the supported include targets in stable order for
// the usage message.
func supportedIncludeNames() []string {
	names := make([]string, 0, len(supportedIncludeTargets))
	for name := range supportedIncludeTargets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// wantsInclude reports whether target was requested among the include flags.
func wantsInclude(targets []string, target string) bool {
	for _, t := range targets {
		if t == target {
			return true
		}
	}
	return false
}

// reportFailure is the single command-execution-failure chokepoint, now
// format-aware (032, ADR-1; was reportClientError). It refines a generic non-2xx
// *ResponseError into a typed *apiclient.ProblemError (once — guarded against
// double-refinement) so the API's own detail surfaces and the typed error travels
// up the chain (015 ADR-4), then calls Diagnose ONCE (031) and hands that single
// Diagnostic to both the exit-code category and errorEnvelopeFor, so the category
// and the rendered facts are computed from the SAME value and can never disagree.
//
// The resolved format chooses the channel and shape (ADR-3):
//
//   - structured (json/yaml): write the 018 unified error envelope to stdout —
//     the channel an agent parses, so a failure reads the same way as a success.
//     The whole document is built in memory first (output.RenderError); on a render
//     error nothing reaches stdout and the outcome is RuntimeError(1) (the
//     buffer-then-write contract, ADR-4 — the failure-render path is never itself a
//     silent failure). The render error printed to stderr is token-free (018).
//   - human (full/compact): write renderDiagnostic(d) to stderr exactly as before,
//     stdout untouched.
//
// It returns d.Category (mapped to the exit code by 004, unchanged across formats)
// and the refined error. Only the presentation differs by format; the outcome does
// not. It never emits the X-Auth-Token (Diagnose and the envelope are both
// token-free).
// renderErrorFn is the structured failure-render seam (output.RenderError by
// default). Like renderFn (render.go), it is a package var so a test can override
// it to exercise the buffer-then-write failure path — a structured render that
// cannot complete — without contriving an un-encodable envelope (errorEnvelopeFor
// gates Body on json.Valid, so it never produces one for real failures). The
// production call is output.RenderError, unchanged.
var renderErrorFn = output.RenderError

func reportFailure(stdout, stderr io.Writer, format output.OutputFormat, err error) (Outcome, error) {
	err = refineClientError(err)
	d := Diagnose(err)
	if machineFmt, ok := format.MachineFormat(); ok {
		doc, rerr := renderErrorFn(machineFmt, errorEnvelopeFor(d, err))
		if rerr != nil {
			// Buffer-then-write: a render failure leaves stdout empty and maps to
			// RuntimeError(1). The error is token-free (018 contract).
			fmt.Fprintln(stderr, rerr.Error())
			return RuntimeError, rerr
		}
		_, _ = stdout.Write(doc)
		return d.Category, err
	}
	fmt.Fprintln(stderr, renderDiagnostic(d))
	return d.Category, err
}

// refineClientError refines a generic non-2xx *apiclient.ResponseError into a
// typed *apiclient.ProblemError via ExtractProblem, ONCE: an error that is
// already a *ProblemError (or carries no *ResponseError) is returned unchanged,
// so a value that funnels through reportClientError more than once is never
// re-wrapped. This is the single refinement site (015 ADR-4) — the read commands
// (011–014) need no per-command edit.
func refineClientError(err error) error {
	var problemErr *apiclient.ProblemError
	if errors.As(err, &problemErr) {
		return err
	}
	var responseErr *apiclient.ResponseError
	if errors.As(err, &responseErr) {
		return apiclient.ExtractProblem(responseErr)
	}
	return err
}

// newMeCommand builds the `me` leaf: a guard-ready cobra command (no positional
// args, non-empty Short, SilenceErrors/SilenceUsage so runMe owns its messages)
// with a local --include flag. Its RunE reads the persistent --base-url value
// (T003 / ADR-2), delegates to the pure runMe, and maps the returned Outcome
// onto dispatch's error channel (the runLogin pattern, extended for the
// operational categories). The seam is injected so tests drive a fake one;
// production passes productionSeam{} from Assemble.
func newMeCommand(seam meSeam) *cobra.Command {
	var include []string
	cmd := &cobra.Command{
		Use:   "me",
		Short: "Print the authenticated actor, organization, and membership",
		Long: "me prints who the API token resolves to — the authenticated actor " +
			"with its organization and membership, and optionally the roles it fills " +
			"(--include roles). It is the smallest end-to-end read.",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Read the persistent --base-url value the root declares (ADR-2). A
			// lookup failure here is a wiring bug, not operator input.
			baseURL, err := cmd.Flags().GetString(apiclient.FlagBaseURL)
			if err != nil {
				// The entrypoint discards Run's returned error, so include the cause
				// here or this wiring bug surfaces as a bare exit code.
				fmt.Fprintf(cmd.ErrOrStderr(), "could not read the --base-url flag: %v\n", err)
				return err
			}
			outputFlag, err := cmd.Flags().GetString(output.FlagOutput)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "could not read the --output flag: %v\n", err)
				return err
			}
			outcome, oerr := runMe(meConfig{
				seam:       seam,
				baseURL:    baseURL,
				outputFlag: outputFlag,
				include:    include,
				reqCtx:     cmd.Context(),
				stdout:     cmd.OutOrStdout(),
				stderr:     cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	cmd.Flags().StringSliceVar(&include, "include", nil, "Embed related resources in the read (supported: roles)")
	return cmd
}

// outcomeToDispatchError maps a code-free Outcome onto the error channel dispatch
// reads (the runLogin pattern, extended): Success → nil; UsageError →
// *commandUsageError (→ code 2); RuntimeError → the error as-is (dispatch's
// catch-all → code 1); the operational categories (APIError, NetworkUnavailable)
// → *outcomeError carrying the category so Exit-Code Convention maps them to 3/6
// rather than collapsing to 1.
func outcomeToDispatchError(outcome Outcome, err error) error {
	switch outcome {
	case Success:
		return nil
	case UsageError:
		return &commandUsageError{err}
	case RuntimeError:
		return err
	default:
		return &outcomeError{outcome: outcome, err: err}
	}
}
