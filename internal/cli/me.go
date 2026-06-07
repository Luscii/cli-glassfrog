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

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/Luscii/cli-glassfrog/internal/rcfile"
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

// meConfig carries everything runMe needs, gathered by the command's RunE.
// Keeping runMe a function of injected values (not the seam alone) makes the
// whole read — validate, assemble, build, send, render/classify — testable over
// a fake transport with no real network or ~/.glassfrogrc.
type meConfig struct {
	seam    meSeam
	baseURL string   // the persistent --base-url flag value (may be empty)
	include []string // the raw --include targets, validated before any request
	reqCtx  context.Context
	stdout  io.Writer
	stderr  io.Writer
}

// runMe is the pure orchestration the command delegates to (ADR-5): validate the
// include targets fail-fast, assemble the context once, build the client once,
// send GET /me once, and render the projection on success or classify+report the
// typed error otherwise. It returns the code-free Outcome the command maps onto
// dispatch's error channel; it never emits an exit code, never retries, and
// never reads the token — the projection renders response-side fields only.
func runMe(cfg meConfig) (Outcome, error) {
	// 1. Validate --include BEFORE any assembly or request (fail-fast usage
	//    error, no wasted call — pinned by a tripwire transport in the tests).
	if err := validateInclude(cfg.include); err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}
	includeRoles := wantsInclude(cfg.include, "roles")

	// 2. Resolve the connection and build the client once. A base-URL error
	//    surfaces here (no doomed send); classify + report it.
	ctx := cfg.seam.assemble(cfg.baseURL)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportClientError(cfg.stderr, err)
	}

	// 3. Send exactly one GET /me, decoding the 2xx body into the projection
	//    target. include=roles is added only when requested.
	req := apiclient.Request{Method: http.MethodGet, Path: "/me"}
	if includeRoles {
		req.Query = url.Values{"include": []string{"roles"}}
	}
	var me glassfrog.MeResponse
	if _, err := client.Execute(cfg.reqCtx, req, &me); err != nil {
		return reportClientError(cfg.stderr, err)
	}

	// 4. Render the reshaped projection (never the token).
	fmt.Fprint(cfg.stdout, formatMe(me, includeRoles))
	return Success, nil
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

// formatMe renders the reshaped identity projection (not raw JSON; --output json
// is deferred). It surfaces, each on its own labelled line in a stable order:
// actor name/kind/id, organization name/id, and the membership access level —
// the id values are always present (the machine-actionable handles). The roles
// section is appended only when roles were requested AND the response carries
// them; an empty embed omits the section rather than printing an empty list. It
// is pure (MeResponse → string) and renders only response-side fields, so the
// token never appears.
func formatMe(me glassfrog.MeResponse, includeRoles bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "actor:        %s (%s) %s\n", me.Actor.Name, me.Actor.Kind, me.Actor.ID)
	fmt.Fprintf(&b, "organization: %s (%s)\n", me.Organization.Name, me.Organization.ID)
	fmt.Fprintf(&b, "access:       %s\n", me.Membership.AccessLevel)
	if includeRoles && len(me.Roles) > 0 {
		b.WriteString("roles:\n")
		for _, r := range me.Roles {
			fmt.Fprintf(&b, "  - %s (%s)\n", r.Name, r.ID)
		}
	}
	return b.String()
}

// reportClientError writes a controlled, token-free, next-step message to stderr
// and returns the Outcome the shared classifier assigns to err. The Outcome
// always comes from classifyClientError (the single classification chain, reused
// by 012–017); this function only adds the operator-facing presentation, so the
// category and the message can never disagree about which error occurred.
func reportClientError(stderr io.Writer, err error) (Outcome, error) {
	fmt.Fprintln(stderr, formatClientErrorMessage(err))
	return classifyClientError(err), err
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
	var responseErr *apiclient.ResponseError
	if errors.As(err, &responseErr) {
		// Name the status (the cause) AND a GENERIC next step. The next step is
		// deliberately not tailored to the status — a per-status meaning (401 →
		// re-authenticate, 429 → back off) is API Error Extraction (015) / Rate-Limit
		// Handling (017)'s concern, and interpreting the status here would breach the
		// spec Non-Behavior. A generic "check access and retry, or consult the status"
		// hint satisfies Action Transparency (cause + next step) without that
		// interpretation. Shared by `me` and `me roles`.
		return fmt.Sprintf("the API returned a non-2xx response: status %d — the API rejected the read; check that the token has access and retry, or consult the status code", responseErr.StatusCode)
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
			outcome, oerr := runMe(meConfig{
				seam:    seam,
				baseURL: baseURL,
				include: include,
				reqCtx:  cmd.Context(),
				stdout:  cmd.OutOrStdout(),
				stderr:  cmd.ErrOrStderr(),
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
