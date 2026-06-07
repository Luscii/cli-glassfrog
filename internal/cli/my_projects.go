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

// incompleteProjectsNote is the single stderr line the command writes (still
// exiting 0) when the API reports more projects exist than this first response
// carried, so a partial list is never silently presented as complete. It follows
// My Roles (012)'s signal convention — the note rides stderr while the projection
// rides stdout — rather than inventing a second signal shape (interface-cli). It
// is the My Actions (013) note worded for projects.
const incompleteProjectsNote = "note: more projects exist than shown; pagination is not yet supported, so this list may be incomplete"

// noRoleMarker is the explicit owning-role placeholder for a non-role-owned
// project (a null role_id — an individual initiative). The projection renders it
// in place of a blank role so the reader sees "no owning role" rather than a
// missing field (interface-cli — explicit no-role marker).
const noRoleMarker = "—"

// myProjectsConfig carries everything runMyProjects needs, gathered by the
// command's RunE. As with runMyActions, keeping runMyProjects a function of
// injected values makes the whole read — validate, assemble, build, send,
// render/classify — testable over a fake transport with no real network or
// ~/.glassfrogrc. It reuses the meSeam (the assemble + newClient pair Identity
// Read defined); My Projects needs nothing more. status is the raw --status flag
// value (may be empty), validated before any request.
type myProjectsConfig struct {
	seam    meSeam
	baseURL string // the inherited persistent --base-url flag value (may be empty)
	status  string // the raw --status flag value (may be empty); validated before any I/O
	reqCtx  context.Context
	stdout  io.Writer
	stderr  io.Writer
}

// runMyProjects is the pure orchestration the `me projects` leaf delegates to,
// the near-mechanical twin of runMyActions: validate --status fail-fast (an
// unsupported value is a usage error with no request issued), assemble the
// context once, build the client once, send one GET /me/projects (carrying
// ?status= when filtered), then render the projection on success (with the
// more-available note to stderr when the API signals a next page) or classify +
// report the typed error via 011's shared helpers. It emits no exit code, never
// retries, and never reads the token — the projection renders response-side
// fields only. There is no --include (ADR-2): /me/projects offers no include.
func runMyProjects(cfg myProjectsConfig) (Outcome, error) {
	// 1. Validate --status BEFORE any assembly or request (fail-fast usage error,
	//    no wasted call — pinned by a tripwire transport in the tests). Reuses
	//    013's shared validateStatus + status set; no second validator.
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

	// 3. Send exactly one GET /me/projects, decoding the 2xx body into the shared
	//    response type. ?status= is added only when a filter was supplied.
	req := apiclient.Request{Method: http.MethodGet, Path: "/me/projects"}
	if cfg.status != "" {
		req.Query = url.Values{"status": []string{cfg.status}}
	}
	var resp glassfrog.MyProjectsResponse
	if _, err := client.Execute(cfg.reqCtx, req, &resp); err != nil {
		return reportClientError(cfg.stderr, err)
	}

	// 4. Render the reshaped projection to stdout (never the token). When the API
	//    reports more projects than this page carried, write the more-available
	//    note to stderr so the partial list is never read as complete — still
	//    exit 0. The next page is signalled, never fetched (Pagination 016).
	fmt.Fprint(cfg.stdout, formatMyProjects(resp))
	if incompleteProjects(resp) {
		fmt.Fprintln(cfg.stderr, incompleteProjectsNote)
	}
	return Success, nil
}

// newMyProjectsCommand builds the `projects` leaf attached under Identity Read's
// runnable `me` command — a sibling of `roles` (My Roles 012) and `actions` (My
// Actions 013). (The 014 spec prose says "my projects"; the implemented
// convention is `me projects` under `me`, mirroring `me roles`/`me actions` —
// see .score/memory/LEARNINGS.md.) It is a guard-ready cobra command (no
// positional args, non-empty Short, SilenceErrors/SilenceUsage so runMyProjects
// owns its messages) with a local --status flag and NO --include (ADR-2). Its
// RunE reads the persistent --base-url value the root declares (inherited, not
// re-registered), delegates to the pure runMyProjects, and maps the returned
// Outcome onto dispatch's error channel via the shared outcomeToDispatchError —
// adding no new Outcome category and no ExitCode case. The seam is injected so
// tests drive a fake one; production passes productionSeam{} from Assemble.
func newMyProjectsCommand(seam meSeam) *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List the projects you are responsible for",
		Long: "projects lists the projects owned by the roles the authenticated practitioner " +
			"fills (GET /me/projects), optionally filtered by --status. The caller is identified " +
			"by the stored token, not named on the command line. Each project is shown as a " +
			"reshaped projection — id, status, description, owning role (or a no-role marker), " +
			"whether it has sub-projects and actions, and tags — never the raw API envelope. It " +
			"fetches the first page and signals when more results exist.",
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
			outcome, oerr := runMyProjects(myProjectsConfig{
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
	cmd.Flags().StringVar(&status, "status", "", "Filter by project status (one of: "+strings.Join(supportedStatusNames(), ", ")+")")
	return cmd
}

// formatMyProjects renders the reshaped `glassfrog me projects` projection (never
// raw JSON; --output json is the deferred Unconsumable Output capability). It is
// pure (MyProjectsResponse → string) and surfaces only response-side fields, so
// the token never appears. One entry per project, in the order the API returned
// them; the id is always present (the machine-actionable handle). The layout:
//
//	<proj_…>  [<status>]  <description>                              // description → — when null/empty
//	  role: <role_… or —>   sub-projects: yes|no   actions: yes|no   tags: <t1>, <t2>
//
// A null role_id (a non-role-owned project) renders the explicit no-role marker
// in the role slot rather than a blank. An empty list is the valid "you own no
// matching projects" answer and renders exactly `no projects` (interface-cli).
// The more-available signal is the command's concern (stderr, 012's convention)
// and never appears in this body — the project's timestamps, link, note, and
// other fields are never rendered (spec Non-Behaviors).
func formatMyProjects(resp glassfrog.MyProjectsResponse) string {
	if len(resp.Data) == 0 {
		return "no projects\n"
	}

	var b strings.Builder
	for _, p := range resp.Data {
		description := strings.TrimSpace(p.Description)
		if description == "" {
			description = "—"
		}
		fmt.Fprintf(&b, "%s  [%s]  %s\n", p.ID, p.Status, description)

		role := p.RoleID
		if strings.TrimSpace(role) == "" {
			role = noRoleMarker
		}
		line := fmt.Sprintf("  role: %s   sub-projects: %s   actions: %s", role, yesNo(p.HasSubProjects), yesNo(p.HasActions))
		if len(p.Tags) > 0 {
			line += "   tags: " + strings.Join(p.Tags, ", ")
		}
		fmt.Fprintf(&b, "%s\n", line)
	}
	return b.String()
}

// yesNo renders a presence-signal boolean (has_sub_projects / has_actions) as the
// reader-facing yes/no the projection shows.
func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

// incompleteProjects reports whether the API signalled that more projects exist
// than this (first) response carried — i.e. Meta.Pagination.HasNextPage. The
// command turns a true result into the single stderr more-available note (still
// exit 0); this command never follows pagination (deferred to 016).
func incompleteProjects(resp glassfrog.MyProjectsResponse) bool {
	return resp.Meta.Pagination.HasNextPage
}
