package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/paging"
	"github.com/Luscii/cli-glassfrog/internal/render"
	"github.com/spf13/cobra"
)

// incompleteSubrolesNote is the stderr line the default walk writes when it stops
// on an error after gathering at least one page: the partial set is already on
// stdout, so this names the cause and marks the list incomplete — a partial list
// is never silently presented as complete (CONSTITUTION VI; interface-cli). The
// %s is the cause; the command exits non-zero (classifyClientError(Stop)). It is
// the subroles-worded sibling of roles.go's incompleteWalkNote.
const incompleteSubrolesNote = "note: result is incomplete — %s; the subroles shown are a partial set"

// moreSubrolesNote is the stderr line the --first-page opt-out writes when the
// first page reports more pages exist: the operator chose the boundary, so this
// is not an error (exit 0) — it just keeps a partial list from being read as
// complete (interface-cli; CONSTITUTION VI).
const moreSubrolesNote = "note: more subroles exist than shown; re-run without --first-page to fetch all"

// subrolesSeam supplies everything the `subroles` read needs from the outside, so
// runSubroles is pure over injected values and every branch runs offline. Same
// shape as rolesSeam/meSeam (assemble + newClient + sleep + resolveFormat), so
// productionSeam satisfies it unchanged and the existing test fakes drive it. It
// never reads ctx.Cred.Token — the token rides 007's AuthTransport in the client.
type subrolesSeam interface {
	assemble(baseURL string) apiclient.ConnectionContext
	newClient(ctx apiclient.ConnectionContext) (*apiclient.Client, error)
	sleep() func(time.Duration)
	resolveFormat(flagValue string) (output.OutputFormat, error)
}

// subrolesConfig carries everything runSubroles needs, gathered by the command's
// RunE. Keeping runSubroles a function of injected values makes the whole read —
// validate, assemble, build, walk, render/classify — testable over a fake
// transport with no real network or ~/.glassfrogrc.
type subrolesConfig struct {
	seam       subrolesSeam
	baseURL    string // inherited persistent --base-url (may be empty)
	outputFlag string // inherited persistent --output (may be empty), resolved before any request
	id         string // the required positional parent role id (ExactArgs(1))

	include    []string
	firstPage  bool
	perPage    int
	perPageSet bool // whether --per-page was provided (Changed); presence, not value

	// --depth is declared on `subroles` only to reject it: subroles returns one
	// level, so --depth (which bounds a recursive tree) is a usage error
	// (interface "two completeness models"). Presence, not value.
	depthSet bool

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// supportedSubrolesIncludes is the closed enum of --include values listSubroles
// accepts — exactly the getRole set (interface). A value outside it is rejected
// fail-fast (the API would otherwise silently ignore it — plan ADR-4).
var supportedSubrolesIncludes = map[string]bool{
	"assignments": true,
	"subroles":    true,
	"parent_role": true,
	"policies":    true,
	"notes":       true,
	"skills":      true,
}

// runSubroles is the pure orchestration the `subroles` leaf delegates to: resolve
// the output format, validate the flags (reject --depth) and --include values
// fail-fast before any assembly or request, assemble the connection and build the
// retrying executor, then walk GET /roles/{id}/subroles to completion (reusing
// 025's walk + --first-page opt-out verbatim — ADR-3). It adds no new
// Outcome/ExitCode and never reads the token.
func runSubroles(cfg subrolesConfig) (Outcome, error) {
	// 1. Resolve the output format FIRST (020).
	format, ferr := cfg.seam.resolveFormat(cfg.outputFlag)
	if ferr != nil {
		return reportFormatResolutionError(cfg.stderr, ferr)
	}

	// 2. Validate the flags BEFORE any assembly or request (fail-fast usage error,
	//    pinned by a tripwire transport): --depth is rejected (subroles is one
	//    level).
	if err := validateSubrolesFlags(cfg); err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}

	// 2b. Validate --include against the subroles closed set BEFORE any request.
	//     The id is NOT validated locally (ADR-4): the API 404s an unknown id.
	if err := validateIncludeSet(cfg.include, supportedSubrolesIncludes); err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}

	// 3. Resolve the connection and build the client + retrying executor.
	ctx := cfg.seam.assemble(cfg.baseURL)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, format, err)
	}
	exec := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, cfg.seam.sleep(), cfg.stderr)

	return runSubrolesList(cfg, exec, format)
}

// runSubrolesList walks GET /roles/{id}/subroles. The output format changes ONLY
// how the gathered set is rendered — never how much is fetched: every format
// walks to completion by default and signals incompleteness the same way (a
// stderr note, never a silently short list). --first-page opts out to a single
// page. This is the 025 roles-list shape with RoleDetail items and the subroles
// path. The id is escaped as one path segment (passed through unvalidated per
// ADR-4, but a raw `/`/`..` must not redirect/traverse).
func runSubrolesList(cfg subrolesConfig, exec executor, format output.OutputFormat) (Outcome, error) {
	path := "/roles/" + url.PathEscape(cfg.id) + "/subroles"
	req := apiclient.Request{Method: http.MethodGet, Path: path, Query: subrolesQuery(cfg)}

	if cfg.firstPage {
		return runSubrolesFirstPage(cfg, exec, format, req)
	}

	// Structured: walk to completion preserving each child's raw bytes, then emit
	// the aggregated {data:[…]} document (reusing the resource-neutral aggregator).
	if machineFmt, ok := format.MachineFormat(); ok {
		res := paging.All[json.RawMessage](cfg.reqCtx, exec, req, subrolesWalkOptions(cfg)...)
		if res.Stop != nil && len(res.Records) == 0 {
			return reportFailure(cfg.stdout, cfg.stderr, format, res.Stop)
		}
		doc, rerr := aggregateRawData(machineFmt, res.Records)
		if rerr != nil {
			fmt.Fprintln(cfg.stderr, rerr.Error())
			return RuntimeError, rerr
		}
		_, _ = cfg.stdout.Write(doc)
		if res.Stop != nil {
			return reportIncompleteSubrolesWalk(cfg.stderr, res.Stop)
		}
		return Success, nil
	}

	// Human: walk to completion, render the subroles projection (a child role
	// block per child; an empty set renders `No subroles.`).
	res := paging.All[glassfrog.RoleDetail](cfg.reqCtx, exec, req, subrolesWalkOptions(cfg)...)
	if res.Stop != nil && len(res.Records) == 0 {
		// A walk that stopped before gathering any record is a clean failure (e.g. a
		// first-page transport/auth/API error): no partial set to show.
		return reportFailure(cfg.stdout, cfg.stderr, format, res.Stop)
	}
	view := render.SubrolesView{Children: res.Records, Requested: includeSet(cfg.include)}
	text, rerr := renderFn(render.ResourceSubroles, humanFormat(format), view)
	if rerr != nil {
		fmt.Fprintln(cfg.stderr, rerr.Error())
		return RuntimeError, rerr
	}
	fmt.Fprint(cfg.stdout, text)
	if res.Stop != nil {
		return reportIncompleteSubrolesWalk(cfg.stderr, res.Stop)
	}
	return Success, nil
}

// runSubrolesFirstPage performs the --first-page opt-out: a single
// GET /roles/{id}/subroles page (no walk) in EVERY format, with one stderr note
// when the API reports more pages exist (still exit 0 — the operator chose the
// boundary). The structured path emits the same {data:[…]} envelope the default
// walk does; the human path renders the projection. --per-page (if set) sizes the
// single request; the walker is not involved.
func runSubrolesFirstPage(cfg subrolesConfig, exec executor, format output.OutputFormat, req apiclient.Request) (Outcome, error) {
	if cfg.perPageSet {
		// Pass the value through as-is — no client-side clamp (paging's contract): an
		// out-of-range value surfaces the API's rejection rather than being ignored.
		q := cloneQuery(req.Query)
		q.Set("per_page", strconv.Itoa(cfg.perPage))
		req.Query = q
	}

	if machineFmt, ok := format.MachineFormat(); ok {
		var page glassfrog.Page[json.RawMessage]
		if _, err := exec.Execute(cfg.reqCtx, req, &page); err != nil {
			return reportFailure(cfg.stdout, cfg.stderr, format, err)
		}
		doc, rerr := aggregateRawData(machineFmt, page.Data)
		if rerr != nil {
			fmt.Fprintln(cfg.stderr, rerr.Error())
			return RuntimeError, rerr
		}
		_, _ = cfg.stdout.Write(doc)
		if page.Meta.Pagination.HasNextPage {
			fmt.Fprintln(cfg.stderr, moreSubrolesNote)
		}
		return Success, nil
	}

	var page glassfrog.Page[glassfrog.RoleDetail]
	if _, err := exec.Execute(cfg.reqCtx, req, &page); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, format, err)
	}
	view := render.SubrolesView{Children: page.Data, Requested: includeSet(cfg.include)}
	text, rerr := renderFn(render.ResourceSubroles, humanFormat(format), view)
	if rerr != nil {
		fmt.Fprintln(cfg.stderr, rerr.Error())
		return RuntimeError, rerr
	}
	fmt.Fprint(cfg.stdout, text)
	if page.Meta.Pagination.HasNextPage {
		fmt.Fprintln(cfg.stderr, moreSubrolesNote)
	}
	return Success, nil
}

// reportIncompleteSubrolesWalk writes the mid-walk incomplete note (the partial
// set is already on stdout) and returns the classified non-zero outcome. It
// mirrors roles.go's reportIncompleteWalk but with the subroles wording: the Stop
// error is refined once so a non-2xx splits into permission/rate-limit (015), and
// the note text, the classified outcome, and the returned error all derive from
// that same refined value (the reportClientError invariant).
func reportIncompleteSubrolesWalk(stderr io.Writer, stop error) (Outcome, error) {
	refined := refineClientError(stop)
	fmt.Fprintf(stderr, incompleteSubrolesNote+"\n", refined.Error())
	return classifyClientError(refined), refined
}

// subrolesQuery builds the GET /roles/{id}/subroles query: include is comma-joined
// (style:form explode:false) when supplied. A nil return leaves the request
// unparameterised.
func subrolesQuery(cfg subrolesConfig) url.Values {
	if len(cfg.include) == 0 {
		return nil
	}
	return url.Values{"include": {strings.Join(cfg.include, ",")}}
}

// subrolesWalkOptions builds the paging options for the default walk. --per-page
// (016's WithPageSize) sizes the walk when provided (by presence, not value); the
// value is passed through as-is — no client-side clamp (paging's contract).
func subrolesWalkOptions(cfg subrolesConfig) []paging.Option {
	if cfg.perPageSet {
		return []paging.Option{paging.WithPageSize(cfg.perPage)}
	}
	return nil
}

// validateSubrolesFlags rejects the subroles flag misuses fail-fast, before any
// request (the 011/013 validate-before-call shape, pinned by a transport
// tripwire): --depth bounds a recursive tree, but subroles returns only one level,
// so passing --depth is a usage error. The message names the misuse and the fix.
func validateSubrolesFlags(cfg subrolesConfig) error {
	if cfg.depthSet {
		return fmt.Errorf(
			"--depth applies to a tree read; subroles returns one level only — remove --depth (use `glassfrog tree <id> --depth N` for a bounded tree)",
		)
	}
	return nil
}

// newSubrolesCommand builds the runnable `subroles` leaf (ADR-1): a guard-ready
// cobra command with a REQUIRED positional id (Args: cobra.ExactArgs(1)), a
// non-empty Short, and SilenceErrors/SilenceUsage so runSubroles owns its
// messages. It declares the subroles flags (--include, --first-page, --per-page)
// and — to reject it with a friendly message rather than a bare cobra "unknown
// flag" — the --depth flag it forbids. It reads the inherited persistent
// --base-url/--output flags, then delegates to the pure runSubroles. The seam is
// injected so tests drive a fake one; production passes productionSeam{}.
func newSubrolesCommand(seam subrolesSeam) *cobra.Command {
	var (
		include   []string
		firstPage bool
		perPage   int
		depth     int
	)
	cmd := &cobra.Command{
		Use:   "subroles <id>",
		Short: "List a role's immediate child roles, walking pages to completion",
		Long: "subroles lists the immediate child roles directly inside a role (one level " +
			"only), walking every page to completion by default so the list is complete or " +
			"plainly flagged incomplete. Each child can embed related resources with --include. " +
			"For the full recursive hierarchy use `tree`; for the flat org-wide list use `roles`.",
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
			outcome, oerr := runSubroles(subrolesConfig{
				seam:       seam,
				baseURL:    baseURL,
				outputFlag: outputFlag,
				id:         args[0],
				include:    include,
				firstPage:  firstPage,
				perPage:    perPage,
				// Presence, not value: --per-page=0 must reach the API rather than be
				// silently ignored (paging's no-clamp contract).
				perPageSet: cmd.Flags().Changed("per-page"),
				depthSet:   cmd.Flags().Changed("depth"),
				reqCtx:     cmd.Context(),
				stdout:     cmd.OutOrStdout(),
				stderr:     cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	cmd.Flags().StringSliceVar(&include, "include", nil, "Per-child related resources to embed (assignments,subroles,parent_role,policies,notes,skills)")
	cmd.Flags().BoolVar(&firstPage, "first-page", false, "Fetch only the first page and signal if more subroles exist")
	cmd.Flags().IntVar(&perPage, "per-page", 0, "Page size for the walk (the API owns the valid range)")
	cmd.Flags().IntVar(&depth, "depth", 0, "Not valid on subroles (one level only) — use `tree <id> --depth N` for a bounded tree")
	return cmd
}
