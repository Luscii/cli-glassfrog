package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/rcfile"
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
	// resolveFormat resolves the effective output format (020) from the inherited
	// --output flag value plus the real environment and .glassfrogrc walk. Production
	// binds output.ResolveFormatFromOS over os.Getwd/os.UserHomeDir; tests bind a
	// fake over hand-built sources. Kept off assemble (009) — output format is a
	// presentation concern resolved on the render path, independent of the
	// connection context. A present-but-invalid value returns a *output.FormatError
	// (or an internal/rcfile read/format error) the read maps to UsageError(2).
	resolveFormat(flagValue string) (output.OutputFormat, error)
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

// resolveFormat binds the real OS to Output Format Selection's resolver (020): it
// derives the working and home directories and delegates to
// output.ResolveFormatFromOS, which binds os.Getenv and the internal/rcfile walk
// over the output key. A working directory that cannot be determined errors; a home
// directory that cannot be determined simply drops the home fallback (the
// ResolveBaseURLFromOS shape). Tests bind a fake over hand-built sources, so no
// suite reads the real ~/.glassfrogrc.
func (productionSeam) resolveFormat(flagValue string) (output.OutputFormat, error) {
	startDir, err := os.Getwd()
	if err != nil {
		return output.DefaultFormat, fmt.Errorf("could not determine the working directory: %w", err)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "" // no home → skip the home fallback rather than fail
	}
	return output.ResolveFormatFromOS(flagValue, startDir, homeDir)
}

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
	// 1. Resolve the output format FIRST (020, ADR-4): a present-but-invalid
	//    --output/GLASSFROG_OUTPUT/.glassfrogrc output value fails fast as a usage
	//    error before any assembly or request — no wasted call (the validateInclude
	//    fail-fast shape, pinned by a tripwire transport). Resolution is independent
	//    of connection assembly (009).
	format, ferr := cfg.seam.resolveFormat(cfg.outputFlag)
	if ferr != nil {
		return reportFormatResolutionError(cfg.stderr, ferr)
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
		return reportClientError(cfg.stderr, err)
	}

	// 4. Build GET /me and the 017 retry executor, then dispatch on the resolved
	//    format (020 ADR-3): the dispatch picks the decode target (raw bytes for
	//    json/yaml, the typed projection for full/compact) and the renderer, sends
	//    through the executor, and writes to stdout. include=roles is added only when
	//    requested. A transient 429 is ridden out within bounded caps; a capped-out
	//    429 surfaces unchanged (017 ADR-5) and 015's classifyClientError maps it to
	//    RateLimited(5). The seam renders only response-side fields, so the token
	//    never appears. me carries no incompleteness signal, so no note.
	req := apiclient.Request{Method: http.MethodGet, Path: "/me"}
	if includeRoles {
		req.Query = url.Values{"include": []string{"roles"}}
	}
	exec := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, cfg.seam.sleep(), cfg.stderr)
	return renderResult[glassfrog.MeResponse](cfg.stdout, cfg.stderr, format, render.ResourceMe, exec, cfg.reqCtx, req, nil)
}

// reportFormatResolutionError writes a resolved-format failure to stderr and returns
// the Outcome the shared classifier assigns to it. The two expected causes —
// an invalid selector (*output.FormatError) and an unreadable/malformed .glassfrogrc
// surfaced while reading the output key (*rcfile.{Read,Format}Error) — both classify
// as UsageError(2) (correctable input). Any other resolution error (e.g. a
// working-directory failure from productionSeam.resolveFormat's os.Getwd) matches no
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

// reportClientError writes a controlled, token-free, next-step message to stderr
// and returns the Outcome the shared classifier assigns to err. It first refines
// a generic non-2xx *ResponseError into a typed *apiclient.ProblemError (once —
// guarded against double-refinement) so the API's own detail surfaces in the
// message and the typed error travels up the chain (015 ADR-4). The Outcome
// always comes from classifyClientError (the single classification chain, reused
// by 012–017), computed from the SAME refined value the message renders, so the
// category and the message can never disagree about which error occurred.
func reportClientError(stderr io.Writer, err error) (Outcome, error) {
	err = refineClientError(err)
	fmt.Fprintln(stderr, formatClientErrorMessage(err))
	return classifyClientError(err), err
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

// formatClientErrorMessage renders the operator-facing, token-free message for a
// client error: it names the cause AND the next step for each class me owns
// (interface-cli Error Communication). It discriminates only to choose wording —
// classifyClientError owns the category — and every branch's text is path/status
// only, never the X-Auth-Token request header.
func formatClientErrorMessage(err error) string {
	var authErr *apiclient.AuthError
	if errors.As(err, &authErr) {
		switch authErr.Kind {
		case apiclient.NoCredentials:
			return "not authenticated — run `glassfrog auth login` or set GLASSFROG_TOKEN"
		default: // CredentialError — the cause names the file, never the token
			return fmt.Sprintf("%s — fix or re-create the credentials file with `glassfrog auth login`", authErr.Error())
		}
	}
	var transportErr *apiclient.TransportError
	if errors.As(err, &transportErr) {
		return fmt.Sprintf("%s — check connectivity; the API may be unreachable", transportErr.Error())
	}
	// The refined non-2xx (015 ADR-4). reportClientError refines a *ResponseError
	// into a *ProblemError before calling here, so the message can surface the
	// API's own detail. This arm precedes the bare *ResponseError arm below
	// because a *ProblemError wraps (Unwrap → *ResponseError), so errors.As would
	// otherwise match the generic arm first. The wording keys on DetailSynthesized
	// (the provenance marker), NOT on Detail emptiness — the fallback always fills
	// Detail. Every branch names the status + a per-class next step; all text is
	// response-side (status/detail/title), never the X-Auth-Token.
	var problemErr *apiclient.ProblemError
	if errors.As(err, &problemErr) {
		hint := clientErrorNextStep(problemErr.StatusCode)
		if problemErr.DetailSynthesized {
			// The body wasn't a parseable Problem Details object: keep the original
			// generic "status N" wording (no synthesized text is presented as the
			// API's own words) plus the per-class next step.
			return fmt.Sprintf("the API returned a non-2xx response: status %d — %s", problemErr.StatusCode, hint)
		}
		// The API's own cause: surface its detail (prefixed with the title when the
		// title adds context beyond the detail) plus the per-class next step.
		cause := problemErr.Detail
		if title := strings.TrimSpace(problemErr.Title); title != "" && title != problemErr.Detail {
			cause = fmt.Sprintf("%s: %s", title, problemErr.Detail)
		}
		return fmt.Sprintf("the API returned a non-2xx response: status %d: %s — %s", problemErr.StatusCode, cause, hint)
	}
	var responseErr *apiclient.ResponseError
	if errors.As(err, &responseErr) {
		// Defensive fallback for an unrefined *ResponseError (reportClientError
		// refines before reaching here, so this is rarely hit): name the status and
		// the per-class next step.
		return fmt.Sprintf("the API returned a non-2xx response: status %d — %s", responseErr.StatusCode, clientErrorNextStep(responseErr.StatusCode))
	}
	var decodeErr *apiclient.DecodeError
	if errors.As(err, &decodeErr) {
		// Name the shape mismatch (the cause) AND the next step: a 2xx body that will
		// not decode usually means the API shape drifted, so the operator's recourse
		// is to report it. The underlying parse error is kept (path/cause only, never
		// the token) for diagnostics.
		return fmt.Sprintf("the API response did not match the expected shape — this may be an API change; report it (%s)", decodeErr.Error())
	}
	// Base-URL configuration error from client construction. This mirrors
	// classifyClientError's base-URL arms (which map all of these to UsageError),
	// so the category and the next-step hint stay symmetric: a malformed
	// configured value (*BaseURLError) AND an unreadable/malformed .glassfrogrc
	// the base-URL resolver surfaced (*rcfile.ReadError / *rcfile.FormatError) all
	// name the source and the correction step. The rcfile arms sit AFTER the
	// AuthError check above, so a credential-file rcfile error (wrapped in
	// *AuthError) keeps its credentials-file hint and only base-URL-path rcfile
	// errors reach here. Each error's text names path/source only, never the token.
	var baseURLErr *apiclient.BaseURLError
	var rcReadErr *rcfile.ReadError
	var rcFormatErr *rcfile.FormatError
	if errors.As(err, &baseURLErr) || errors.As(err, &rcReadErr) || errors.As(err, &rcFormatErr) {
		return fmt.Sprintf("%s — correct --base-url, GLASSFROG_BASE_URL, or the .glassfrogrc base_url", err.Error())
	}
	// Any other unexpected error: surface it verbatim (the apiclient contracts keep
	// these path/cause-only, never the token).
	return err.Error()
}

// clientErrorNextStep returns the per-class next-step hint for a non-2xx status
// (interface-spec Error Communication / CONSTITUTION II "…and the next step").
// The 401/403 and 429 hints match the PermissionError / RateLimited split
// classifyClientError applies; every other non-2xx keeps the original generic
// wording (which names a next step without interpreting the status). The hint is
// status-derived only — it never echoes the token.
func clientErrorNextStep(status int) string {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return "check the token's access / membership"
	case http.StatusTooManyRequests:
		return "the API is rate-limiting; retry later"
	default:
		return "the API rejected the read; check that the token has access and retry, or consult the status code"
	}
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
