package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/spf13/cobra"
)

// incompleteActionsNote is the single stderr line the command writes (still
// exiting 0) when the API reports more actions exist than this first response
// carried, so a partial list is never silently presented as complete. It follows
// My Roles (012)'s signal convention — the note rides stderr while the projection
// rides stdout — rather than inventing a second signal shape (interface-cli).
const incompleteActionsNote = "note: more actions exist than shown; pagination is not yet supported, so this list may be incomplete"

// meActionsConfig carries everything runMeActions needs, gathered by the
// command's RunE. As with runMeRoles, keeping runMeActions a function of injected
// values makes the whole read — validate, assemble, build, send, render/classify
// — testable over a fake transport with no real network or ~/.glassfrogrc. It
// reuses the meSeam (the assemble + newClient pair Identity Read defined); My
// Actions needs nothing more. status is the raw --status flag value (may be
// empty), validated before any request.
type meActionsConfig struct {
	seam    meSeam
	baseURL string // the inherited persistent --base-url flag value (may be empty)
	status  string // the raw --status flag value (may be empty); validated before any I/O
	reqCtx  context.Context
	stdout  io.Writer
	stderr  io.Writer
}

// runMeActions is the pure orchestration the `me actions` leaf delegates to:
// validate --status fail-fast (an unsupported value is a usage error with no
// request issued), assemble the context once, build the client once, send one
// GET /me/actions (carrying ?status= when filtered), then render the projection
// on success (with the more-available note to stderr when the API signals a next
// page) or classify + report the typed error via 011's shared helpers. It emits
// no exit code, never retries, and never reads the token — the projection renders
// response-side fields only.
func runMeActions(cfg meActionsConfig) (Outcome, error) {
	// 1. Validate --status BEFORE any assembly or request (fail-fast usage error,
	//    no wasted call — pinned by a tripwire transport in the tests).
	if err := validateStatus(cfg.status); err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}

	// 2. Resolve the connection and build the client once. A base-URL error
	//    surfaces here (no doomed send); classify + report it via 011's helper.
	ctx := cfg.seam.assemble(cfg.baseURL)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportClientError(cfg.stderr, err)
	}

	// 3. Send exactly one GET /me/actions, decoding the 2xx body into the shared
	//    response type. ?status= is added only when a filter was supplied.
	req := apiclient.Request{Method: http.MethodGet, Path: "/me/actions"}
	if cfg.status != "" {
		req.Query = url.Values{"status": []string{cfg.status}}
	}
	var resp glassfrog.MyActionsResponse
	if _, err := client.Execute(cfg.reqCtx, req, &resp); err != nil {
		return reportClientError(cfg.stderr, err)
	}

	// 4. Render the reshaped projection to stdout (never the token). When the API
	//    reports more actions than this page carried, write the more-available
	//    note to stderr so the partial list is never read as complete — still
	//    exit 0. The next page is signalled, never fetched (Pagination 016).
	fmt.Fprint(cfg.stdout, formatMeActions(resp))
	if incompleteActions(resp) {
		fmt.Fprintln(cfg.stderr, incompleteActionsNote)
	}
	return Success, nil
}

// newMeActionsCommand builds the `actions` leaf attached under Identity Read's
// runnable `me` command — a sibling of the `roles` leaf (My Roles 012). It is a
// guard-ready cobra command (no positional args, non-empty Short,
// SilenceErrors/SilenceUsage so runMeActions owns its messages) with a local
// --status flag. Its RunE reads the persistent --base-url value the root declares
// (inherited, not re-registered), delegates to the pure runMeActions, and maps
// the returned Outcome onto dispatch's error channel via the shared
// outcomeToDispatchError — adding no new Outcome category and no ExitCode case.
// The seam is injected so tests drive a fake one; production passes
// productionSeam{} from Assemble.
func newMeActionsCommand(seam meSeam) *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use:   "actions",
		Short: "List the actions on your plate",
		Long: "actions lists the actions owned by the roles the authenticated practitioner " +
			"fills (GET /me/actions), optionally filtered by --status. The caller is identified " +
			"by the stored token, not named on the command line. Each action is shown as a " +
			"reshaped projection — id, status, description, owning role, and tags — never the raw " +
			"API envelope. It fetches the first page and signals when more results exist.",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Read the persistent --base-url value the root declares (inherited). A
			// lookup failure here is a wiring bug, not operator input.
			baseURL, err := cmd.Flags().GetString(apiclient.FlagBaseURL)
			if err != nil {
				// The entrypoint discards Run's returned error, so include the cause
				// here or this wiring bug surfaces as a bare exit code.
				fmt.Fprintf(cmd.ErrOrStderr(), "could not read the --base-url flag: %v\n", err)
				return err
			}
			outcome, oerr := runMeActions(meActionsConfig{
				seam:    seam,
				baseURL: baseURL,
				status:  status,
				reqCtx:  cmd.Context(),
				stdout:  cmd.OutOrStdout(),
				stderr:  cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "Filter by action status (one of: "+strings.Join(supportedStatusNames(), ", ")+")")
	return cmd
}

// formatMeActions renders the reshaped `glassfrog me actions` projection (never
// raw JSON; --output json is the deferred Unconsumable Output capability). It is
// pure (MyActionsResponse → string) and surfaces only response-side fields, so
// the token never appears. One entry per action, in the order the API returned
// them; the id is always present (the machine-actionable handle). The layout:
//
//	<actn_…>  [<status>]  <description>      // description → — when null/empty
//	  role: <role_…>   tags: <t1>, <t2>      // tags clause only when the action has tags
//
// An empty list is the valid "you own no matching actions" answer and renders
// exactly `No actions.` (interface-cli). The more-available signal is the
// command's concern (stderr, 012's convention) and never appears in this body —
// the action's permissions, timestamps, and other fields are never rendered (spec
// Non-Behaviors).
func formatMeActions(resp glassfrog.MyActionsResponse) string {
	if len(resp.Data) == 0 {
		return "No actions.\n"
	}

	var b strings.Builder
	for _, a := range resp.Data {
		description := strings.TrimSpace(a.Description)
		if description == "" {
			description = "—"
		}
		fmt.Fprintf(&b, "%s  [%s]  %s\n", a.ID, a.Status, description)

		line := fmt.Sprintf("  role: %s", a.RoleID)
		if len(a.Tags) > 0 {
			line += "   tags: " + strings.Join(a.Tags, ", ")
		}
		fmt.Fprintf(&b, "%s\n", line)
	}
	return b.String()
}

// incompleteActions reports whether the API signalled that more actions exist
// than this (first) response carried — i.e. Meta.Pagination.HasNextPage. The
// command turns a true result into the single stderr more-available note (still
// exit 0); this command never follows pagination (deferred to 016).
func incompleteActions(resp glassfrog.MyActionsResponse) bool {
	return resp.Meta.Pagination.HasNextPage
}
