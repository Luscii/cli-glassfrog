package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/spf13/cobra"
)

// subroleActorsConfig carries everything runSubroleActors needs, gathered by the
// command's RunE. Keeping the read a function of injected values makes the whole
// flow — resolve, validate, assemble, build, walk, render/classify — testable over a
// fake transport with no real network or ~/.glassfrogrc. `subrole-actors` takes a
// REQUIRED positional role id (cobra.ExactArgs(1)) and surfaces ONLY --kind among
// the list filters: the /roles/{role_id}/subroles/actors endpoint offers no role_id
// and no q (those are 048's directory filters — plan ADR-2). It reuses the landed
// actorsWalkConfig/runActorsListWalk shape with the request path swapped (plan
// ADR-3) and never reads ctx.Cred.Token.
type subroleActorsConfig struct {
	seam           actorsSeam
	baseURL        string // inherited persistent --base-url (may be empty)
	baseURLPresent bool   // whether --base-url was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	outputFlag     string // inherited persistent --output (may be empty), resolved before any request
	outputPresent  bool   // whether --output was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	roleID         string // the required positional anchor role id (ExactArgs(1)); passed through to a clean 404

	kind    string // --kind filter, validated against {human, agent} before any request
	kindSet bool   // whether --kind was provided (Changed); kind is sent only when set AND non-empty

	firstPage  bool
	perPage    int
	perPageSet bool // whether --per-page was provided (Changed); presence, not value

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runSubroleActors is the pure orchestration the `subrole-actors` leaf delegates to:
// resolve the output format (020) FIRST, then validate the one closed-enum input
// (--kind via the landed validateKind, 048) — both pure, pre-assembly checks,
// output-first so error precedence matches the sibling reads (an invalid --output is
// reported even when --kind is also invalid; interface § Interactions). A failure
// here is a fail-fast usage error with NO request sent (a transport tripwire asserts
// this). Then assemble the connection, build the retrying executor once, and walk
// GET /roles/{role_id}/subroles/actors to completion via the shared actors list-walk
// (runActorsListWalk) — only the request path differs from the `actors` directory
// (plan ADR-3). The anchor role id is PathEscaped to one opaque segment and passed
// through unvalidated, so a leaf anchor's 404 (or an unknown id) surfaces verbatim
// through the shared classifier with NO "this role has no sub-roles" interpretation
// (plan ADR-3, VISION Exclusion 1). It adds no new Outcome/ExitCode and never reads
// the token.
func runSubroleActors(cfg subroleActorsConfig) (Outcome, error) {
	// 1. Resolve the render target FIRST (020 widened by 035): a present-but-invalid
	//    selector — or, for a user template, a missing/unparseable source or empty
	//    stdin — fails fast as a usage error before any assembly or request.
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.outputPresent, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	// 2. Validate the one closed-enum input BEFORE any request (a bad value would be
	//    silently ignored at the API, returning a result indistinguishable from a real
	//    empty answer — plan ADR-2/048). --kind reuses the landed validateKind over
	//    {human, agent}; the anchor role id is a free value passed through. The check
	//    is pure and pre-assembly, so the no-request-on-rejection tripwire holds.
	if err := validateKind(cfg.kind); err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}

	// 3. Resolve the connection and build the client + retrying executor once. A
	//    base-URL error surfaces here (no doomed send); classify + report it.
	ctx := cfg.seam.assemble(cfg.baseURL, cfg.baseURLPresent)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	exec := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, cfg.seam.sleep(), cfg.stderr)

	// 4. Walk GET /roles/{role_id}/subroles/actors via the shared list-walk. The id is
	//    PathEscaped to one opaque segment so a raw `/` cannot redirect the request to
	//    a different endpoint (the 025 runRoleGet discipline). The roll-up is ONE LEVEL
	//    ONLY: this is the single paginated read — no recursion into grand-child roles.
	req := apiclient.Request{
		Method: http.MethodGet,
		Path:   "/roles/" + url.PathEscape(cfg.roleID) + "/subroles/actors",
		Query:  subroleActorsQuery(cfg),
	}
	return runActorsListWalk(cfg.walk(), exec, rt, req)
}

// walk projects the roll-up config down to the shared list-walk context, so
// `subrole-actors` drives the same page loop / render branch / completeness note as
// the `actors` directory (plan ADR-3).
func (cfg subroleActorsConfig) walk() actorsWalkConfig {
	return actorsWalkConfig{
		firstPage:  cfg.firstPage,
		perPage:    cfg.perPage,
		perPageSet: cfg.perPageSet,
		reqCtx:     cfg.reqCtx,
		stdout:     cfg.stdout,
		stderr:     cfg.stderr,
	}
}

// subroleActorsQuery builds the GET /roles/{role_id}/subroles/actors query. The
// endpoint's ONLY filter is `kind` (no role_id, no q — plan ADR-2), sent ONLY when
// --kind was provided (Changed) AND non-empty (the 033/038 optional-flag discipline):
// an omitted flag and `--kind ""` both behave as no filter. The value was already
// validated by runSubroleActors against the closed {human, agent} set. A nil return
// leaves the request unparameterised (every sub-role filler requested). paging.All
// preserves this query across every page of the walk (plan Risk).
func subroleActorsQuery(cfg subroleActorsConfig) url.Values {
	if cfg.kindSet && cfg.kind != "" {
		return url.Values{"kind": {cfg.kind}}
	}
	return nil
}

// newSubroleActorsCommand builds the runnable `subrole-actors` leaf (plan ADR-1): a
// guard-ready cobra command with a REQUIRED positional anchor role id (Args:
// cobra.ExactArgs(1) — zero or more than one positional is a fail-fast usage error
// via cobra's own arg validator, no hand-rolled guard), a non-empty Short, and
// SilenceErrors/SilenceUsage so runSubroleActors owns its messages. It rolls up the
// actors filling the anchor role's DIRECT sub-roles (one level, not transitive) by
// reading GET /roles/{role_id}/subroles/actors, narrowed only by the optional --kind
// filter (the endpoint offers no role_id/q). It is its OWN top-level read leaf — NOT
// a subcommand of the positional-bearing `actors` command (plan ADR-1: the 026/050
// own-command precedent) — and reuses the landed Actor model, the `actors` render
// path, validateKind, and the actors list-walk unchanged (plan ADR-2/ADR-3). It reads
// the inherited persistent --base-url/--output flags, then delegates to the pure
// runSubroleActors. The seam is injected so tests drive a fake one; production passes
// productionSeam{} (reusing the actorsSeam shape).
func newSubroleActorsCommand(seam actorsSeam) *cobra.Command {
	var (
		kind      string
		firstPage bool
		perPage   int
	)
	cmd := &cobra.Command{
		Use:   "subrole-actors <role-id>",
		Short: "Roll up the actors filling a role's direct sub-roles",
		Long: "subrole-actors rolls up the actors filling the anchor role's direct " +
			"sub-roles (one level, not transitive) by reading " +
			"GET /roles/{role_id}/subroles/actors, walking every page to completion by " +
			"default. It is the cross-role counterpart of `actors --role-id` (which lists " +
			"the actors filling one role) and the actor-shaped twin of `tension subroles`. " +
			"Each row is a bare actor (id, name, kind) — not an assignment, so no focus or " +
			"election is shown. Narrow the roll-up to people or agents with --kind; stop at " +
			"one page with --first-page. A leaf anchor (no sub-roles) is surfaced as the API's " +
			"404 read failure, distinct from a sub-role set that simply carries no fillers.",
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL, err := cmd.Flags().GetString(apiclient.FlagBaseURL)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "could not read the --base-url flag: %v\n", err)
				return err
			}
			outputFlag, err := cmd.Flags().GetString(output.FlagOutput)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "could not read the --output flag: %v\n", err)
				return err
			}
			outcome, oerr := runSubroleActors(subroleActorsConfig{
				seam:           seam,
				baseURL:        baseURL,
				baseURLPresent: cmd.Flags().Changed(apiclient.FlagBaseURL),
				outputFlag:     outputFlag,
				outputPresent:  cmd.Flags().Changed(output.FlagOutput),
				roleID:         args[0],
				kind:           kind,
				// Presence, not value: --kind is sent only when Changed AND non-empty, so
				// `--kind ""` behaves as no filter (ADR-2).
				kindSet:   cmd.Flags().Changed("kind"),
				firstPage: firstPage,
				perPage:   perPage,
				// Presence, not value: a provided 0/negative --per-page must reach the API
				// rather than be silently ignored (paging's no-clamp contract).
				perPageSet: cmd.Flags().Changed("per-page"),
				reqCtx:     cmd.Context(),
				stdout:     cmd.OutOrStdout(),
				stderr:     cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "", "Filter the roll-up by actor type (one of: "+strings.Join(supportedKindNames(), ", ")+"), sent as the kind parameter; omitted when empty")
	cmd.Flags().BoolVar(&firstPage, "first-page", false, "Fetch only the first page and signal if more actors exist")
	cmd.Flags().IntVar(&perPage, "per-page", 0, "Page size for the walk (the API owns the valid range)")
	return cmd
}
