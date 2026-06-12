package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/render"
	"github.com/spf13/cobra"
)

// incompleteRolesNote is the single stderr line the command writes (still exiting
// 0) when the API reports more roles exist than this first response carried, so a
// partial list is never silently presented as complete (interface-cli).
const incompleteRolesNote = "note: more roles exist than shown; pagination is not yet supported, so this list may be incomplete"

// meRolesConfig carries everything runMeRoles needs, gathered by the command's
// RunE. As with runMe, keeping runMeRoles a function of injected values makes the
// whole read — assemble, build, send, render/classify — testable over a fake
// transport with no real network or ~/.glassfrogrc. It reuses the meSeam (the
// assemble + newClient pair Identity Read defined); My Roles needs nothing more.
type meRolesConfig struct {
	seam       meSeam
	baseURL    string // the inherited persistent --base-url flag value (may be empty)
	outputFlag string // the inherited persistent --output flag value (may be empty), resolved before any request
	reqCtx     context.Context
	stdout     io.Writer
	stderr     io.Writer
}

// runMeRoles is the pure orchestration the `me roles` leaf delegates to: assemble
// the context once, build the client once, send one GET /me/roles, then render
// the projection on success (with the incompleteness note to stderr when the API
// signals a next page) or classify + report the typed error via 011's shared
// helpers. It emits no exit code, never retries, and never reads the token — the
// projection renders response-side fields only.
func runMeRoles(cfg meRolesConfig) (Outcome, error) {
	// Resolve the render target FIRST (020 widened by 035, ADR-1/ADR-4): a
	// present-but-invalid selector — or, for a user template, a missing/unparseable
	// source or empty stdin — fails fast as a usage error before any assembly or
	// request.
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	// Resolve the connection and build the client once. A base-URL error surfaces
	// here (no doomed send); classify + report it via 011's shared helper.
	ctx := cfg.seam.assemble(cfg.baseURL)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}

	// Send exactly one GET /me/roles and dispatch on the resolved format (020
	// ADR-3): json/yaml route through 018's encoder over the verbatim bytes,
	// full/compact through 019's templates over the typed projection — the dispatch
	// picks the decode target. The endpoint takes no positional args and no filters.
	// When the API reports more roles than this page carried, the note rides stderr
	// on the human path so the partial list is never read as complete (still exit 0);
	// the structured document carries the pagination metadata in-band.
	req := apiclient.Request{Method: http.MethodGet, Path: "/me/roles"}
	return renderResult[glassfrog.MyRolesResponse](
		cfg.stdout, cfg.stderr, rt.format, rt.tmpl, render.ResourceRoles, client, cfg.reqCtx, req,
		func(resp glassfrog.MyRolesResponse) string {
			if incomplete(resp) {
				return incompleteRolesNote
			}
			return ""
		},
	)
}

// newMeRolesCommand builds the `roles` leaf attached under Identity Read's
// runnable `me` command (ADR-1): a guard-ready cobra command (no positional args,
// non-empty Short, SilenceErrors/SilenceUsage so runMeRoles owns its messages).
// Its RunE reads the persistent --base-url value the root declares (inherited,
// not re-registered), delegates to the pure runMeRoles, and maps the returned
// Outcome onto dispatch's error channel via the shared outcomeToDispatchError —
// adding no new Outcome category and no ExitCode case (ADR-2/3). The seam is
// injected so tests drive a fake one; production passes productionSeam{} from
// Assemble.
func newMeRolesCommand(seam meSeam) *cobra.Command {
	return &cobra.Command{
		Use:   "roles",
		Short: "List the roles you fill",
		Long: "roles lists the roles the authenticated practitioner fills (GET /me/roles). " +
			"The caller is identified by the stored token, not named on the command line. " +
			"Each role is shown as a reshaped projection — name, id, purpose, domains, and " +
			"accountabilities — never the raw API envelope.",
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
			outputFlag, err := cmd.Flags().GetString(output.FlagOutput)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "could not read the --output flag: %v\n", err)
				return err
			}
			outcome, oerr := runMeRoles(meRolesConfig{
				seam:       seam,
				baseURL:    baseURL,
				outputFlag: outputFlag,
				reqCtx:     cmd.Context(),
				stdout:     cmd.OutOrStdout(),
				stderr:     cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
}

// incomplete reports whether the API signalled that more roles exist than this
// (first) response carried — i.e. Meta.Pagination.HasNextPage. The command turns
// a true result into the single stderr incompleteness note (still exit 0); this
// command never follows pagination (deferred to 016).
func incomplete(resp glassfrog.MyRolesResponse) bool {
	return resp.Meta.Pagination.HasNextPage
}
