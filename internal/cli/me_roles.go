package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
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
	seam    meSeam
	baseURL string // the inherited persistent --base-url flag value (may be empty)
	reqCtx  context.Context
	stdout  io.Writer
	stderr  io.Writer
}

// runMeRoles is the pure orchestration the `me roles` leaf delegates to: assemble
// the context once, build the client once, send one GET /me/roles, then render
// the projection on success (with the incompleteness note to stderr when the API
// signals a next page) or classify + report the typed error via 011's shared
// helpers. It emits no exit code, never retries, and never reads the token — the
// projection renders response-side fields only.
func runMeRoles(cfg meRolesConfig) (Outcome, error) {
	// Resolve the connection and build the client once. A base-URL error surfaces
	// here (no doomed send); classify + report it via 011's shared helper.
	ctx := cfg.seam.assemble(cfg.baseURL)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportClientError(cfg.stderr, err)
	}

	// Send exactly one GET /me/roles, decoding the 2xx body into the shared
	// response type. The endpoint takes no positional args and no filters.
	var resp glassfrog.MyRolesResponse
	if _, err := client.Execute(cfg.reqCtx, apiclient.Request{Method: http.MethodGet, Path: "/me/roles"}, &resp); err != nil {
		return reportClientError(cfg.stderr, err)
	}

	// Render the reshaped projection to stdout (never the token). When the API
	// reports more roles than this page carried, write the incompleteness note to
	// stderr so the partial list is never read as complete — still exit 0.
	fmt.Fprint(cfg.stdout, formatMeRoles(resp))
	if incomplete(resp) {
		fmt.Fprintln(cfg.stderr, incompleteRolesNote)
	}
	return Success, nil
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
			outcome, oerr := runMeRoles(meRolesConfig{
				seam:    seam,
				baseURL: baseURL,
				reqCtx:  cmd.Context(),
				stdout:  cmd.OutOrStdout(),
				stderr:  cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
}

// formatMeRoles renders the reshaped `glassfrog me roles` projection (never raw
// JSON; --output json is the deferred Unconsumable Output capability). It is
// pure (MyRolesResponse → string) and surfaces only response-side fields, so the
// token never appears. One block per role, blocks separated by a blank line:
//
//	<Role Name> (role_…)
//	  Purpose: <purpose>          // (no purpose set) when null/empty
//	  Domains:                    // header always renders
//	    - <domain description>    //   (none) when the role has none
//	  Accountabilities:           // header always renders, Domains-before-Accountabilities
//	    - <accountability description>
//
// An empty list is the valid "you fill no roles" answer and renders exactly
// `No roles.` (interface-cli). The role's fillers, tags, and classification
// flags are never rendered (spec Non-Behaviors) — they are not even fields on
// the shared Role type.
func formatMeRoles(resp glassfrog.MyRolesResponse) string {
	if len(resp.Data) == 0 {
		return "No roles.\n"
	}

	blocks := make([]string, 0, len(resp.Data))
	for _, r := range resp.Data {
		var b strings.Builder
		fmt.Fprintf(&b, "%s (%s)\n", r.Name, r.ID)

		purpose := strings.TrimSpace(r.Purpose)
		if purpose == "" {
			purpose = "(no purpose set)"
		}
		fmt.Fprintf(&b, "  Purpose: %s\n", purpose)

		domains := make([]string, len(r.Domains))
		for i, d := range r.Domains {
			domains[i] = d.Description
		}
		accountabilities := make([]string, len(r.Accountabilities))
		for i, a := range r.Accountabilities {
			accountabilities[i] = a.Description
		}
		writeRoleSection(&b, "Domains", domains)
		writeRoleSection(&b, "Accountabilities", accountabilities)
		blocks = append(blocks, b.String())
	}
	// Each block already ends in "\n"; joining with "\n" inserts one blank line
	// between blocks and leaves no trailing blank line after the last.
	return strings.Join(blocks, "\n")
}

// writeRoleSection writes a role's Domains/Accountabilities section: the header
// always renders (uniform, agent-parseable structure), followed by the item
// descriptions or `    (none)` when the role has none.
func writeRoleSection(b *strings.Builder, header string, items []string) {
	fmt.Fprintf(b, "  %s:\n", header)
	if len(items) == 0 {
		b.WriteString("    (none)\n")
		return
	}
	for _, d := range items {
		fmt.Fprintf(b, "    - %s\n", d)
	}
}

// incomplete reports whether the API signalled that more roles exist than this
// (first) response carried — i.e. Meta.Pagination.HasNextPage. The command turns
// a true result into the single stderr incompleteness note (still exit 0); this
// command never follows pagination (deferred to 016).
func incomplete(resp glassfrog.MyRolesResponse) bool {
	return resp.Meta.Pagination.HasNextPage
}
