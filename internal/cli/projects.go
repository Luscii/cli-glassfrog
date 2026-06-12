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

// incompleteRoleProjectsWalkNote is the stderr line the default walk writes when
// it stops on an error after gathering at least one page: the partial set is
// already on stdout, so this names the cause and marks the list incomplete — a
// partial list is never silently presented as complete (CONSTITUTION VI;
// interface-cli). The %s is the cause; the command exits non-zero
// (classifyClientError(Stop)). It is the role-projects-worded sibling of
// policies.go's incompletePoliciesNote. (Distinct from me_projects.go's
// incompleteProjectsNote — that is /me/projects' first-page-only signal.)
const incompleteRoleProjectsWalkNote = "note: result is incomplete — %s; the projects shown are a partial set"

// moreRoleProjectsNote is the stderr line the --first-page opt-out writes when the
// first page reports more pages exist: the operator chose the boundary, so this is
// not an error (exit 0) — it just keeps a partial list from being read as complete
// (interface-cli; CONSTITUTION VI).
const moreRoleProjectsNote = "note: more projects exist than shown; re-run without --first-page to fetch all"

// projectsSeam supplies everything the project reads need from the outside, so the
// run* functions are pure over injected values and every branch runs offline. It
// is the same shape as policiesSeam/rolesSeam (assemble + newClient + sleep +
// resolveSelection + readTemplateSource), so productionSeam satisfies it unchanged and the existing test
// fakes drive it. The reads build the RetryExecutor-wrapped *Client once from these
// and hand it to both a direct Execute (the single project read) and paging.All
// (the list walk). It never reads ctx.Cred.Token — the token rides 007's
// AuthTransport inside the client. Shared by both `projects` and `project`.
type projectsSeam interface {
	assemble(baseURL string, baseURLPresent bool) apiclient.ConnectionContext
	newClient(ctx apiclient.ConnectionContext) (*apiclient.Client, error)
	sleep() func(time.Duration)
	resolveSelection(flagValue string, flagPresent bool) (output.Selection, error)
	readTemplateSource(ref output.TemplateRef) (string, error)
}

// projectsConfig carries everything runProjectsList needs, gathered by the
// command's RunE. Keeping the read a function of injected values makes the whole
// flow — validate, resolve, assemble, build, walk, render/classify — testable over
// a fake transport with no real network or ~/.glassfrogrc.
type projectsConfig struct {
	seam           projectsSeam
	baseURL        string // inherited persistent --base-url (may be empty)
	baseURLPresent bool   // whether --base-url was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	outputFlag     string // inherited persistent --output (may be empty), resolved before any request
	outputPresent  bool   // whether --output was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	id             string // the required positional role id (ExactArgs(1))

	query    string // --query/-q free-text search, sent verbatim as q
	querySet bool   // whether --query was provided (Changed); q is sent only when set AND non-empty
	status   string // --status filter, validated against the shared status set before any request
	tag      string // --tag free-text filter, sent verbatim as tag
	tagSet   bool   // whether --tag was provided (Changed); tag is sent only when set AND non-empty

	firstPage  bool
	perPage    int
	perPageSet bool // whether --per-page was provided (Changed); presence, not value

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runProjectsList is the pure orchestration the `projects` leaf delegates to:
// resolve the output format (020) FIRST, then validate the one closed-enum input
// (--status) fail-fast (an unsupported value is a usage error with NO request
// issued) — both pure checks run before any assembly, in the same output-first
// order as the sibling reads (me_projects.go, policies.go) so error precedence is
// consistent. Then assemble the connection, build the retrying executor, and walk
// GET /roles/{id}/projects to completion. --query and --tag are free text passed
// through (plan ADR-3); the role id is a free identifier passed through to a clean
// 404. It adds no new Outcome/ExitCode and never reads the token.
func runProjectsList(cfg projectsConfig) (Outcome, error) {
	// 1. Resolve the render target FIRST (020 widened by 035): a present-but-invalid
	//    selector — or, for a user template, a missing/unparseable source or empty
	//    stdin — fails fast as a usage error before any assembly or request.
	//    Resolving --output ahead of --status keeps error precedence consistent with
	//    the sibling reads (me_projects.go, policies.go).
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.outputPresent, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	// 2. Validate --status BEFORE any assembly or request (fail-fast usage error,
	//    no wasted call — pinned by a tripwire transport in the tests). Reuses the
	//    shared validateStatus + status set (013/014); no second validator. Both
	//    checks are pure and pre-assembly, so the no-request-on-rejection tripwire
	//    holds regardless of their relative order.
	if err := validateStatus(cfg.status); err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}

	// 3. Resolve the connection and build the client + retrying executor. A
	//    base-URL error surfaces here (no doomed send); classify + report it.
	ctx := cfg.seam.assemble(cfg.baseURL, cfg.baseURLPresent)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	exec := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, cfg.seam.sleep(), cfg.stderr)

	return runProjectsListWalk(cfg, exec, rt)
}

// runProjectsListWalk walks GET /roles/{id}/projects. The output format changes
// ONLY how the gathered set is rendered — never how much is fetched: every format
// walks to completion by default and signals incompleteness the same way (a stderr
// note, never a silently short list). --first-page opts out to a single page. This
// is the 025/034 roles/policies walked-list shape with Project items. The id is
// escaped as one path segment (passed through unvalidated per ADR-3, but a raw
// `/`/`..` must not redirect/traverse).
func runProjectsListWalk(cfg projectsConfig, exec executor, rt renderTarget) (Outcome, error) {
	path := "/roles/" + url.PathEscape(cfg.id) + "/projects"
	req := apiclient.Request{Method: http.MethodGet, Path: path, Query: projectsQuery(cfg)}

	if cfg.firstPage {
		return runProjectsFirstPage(cfg, exec, rt, req)
	}

	// Structured: walk to completion preserving each project's raw bytes, then emit
	// the aggregated {data:[…]} document (reusing the resource-neutral aggregator).
	if machineFmt, ok := rt.format.MachineFormat(); ok {
		res := paging.All[json.RawMessage](cfg.reqCtx, exec, req, projectsWalkOptions(cfg)...)
		if res.Stop != nil && len(res.Records) == 0 {
			return reportFailure(cfg.stdout, cfg.stderr, rt.format, res.Stop)
		}
		doc, rerr := aggregateRawData(machineFmt, res.Records)
		if rerr != nil {
			fmt.Fprintln(cfg.stderr, rerr.Error())
			return RuntimeError, rerr
		}
		_, _ = cfg.stdout.Write(doc)
		if res.Stop != nil {
			return reportIncompleteProjectsWalk(cfg.stderr, res.Stop)
		}
		return Success, nil
	}

	// Human / user template: walk to completion, render the projects projection (the
	// landed 014 `projects` key; an empty set renders `no projects`).
	res := paging.All[glassfrog.Project](cfg.reqCtx, exec, req, projectsWalkOptions(cfg)...)
	if res.Stop != nil && len(res.Records) == 0 {
		// A walk that stopped before gathering any record is a clean failure (e.g. a
		// first-page transport/auth/API error): no partial set to show.
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, res.Stop)
	}
	view := render.ProjectsView{Data: res.Records}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceProjects, rt.format, view); outcome != Success {
		return outcome, rerr
	}
	if res.Stop != nil {
		return reportIncompleteProjectsWalk(cfg.stderr, res.Stop)
	}
	return Success, nil
}

// runProjectsFirstPage performs the --first-page opt-out: a single
// GET /roles/{id}/projects page (no walk) in EVERY format, with one stderr note
// when the API reports more pages exist (still exit 0 — the operator chose the
// boundary). The structured path emits the same {data:[…]} envelope the default
// walk does; the human path renders the projection. --per-page (if set) sizes the
// single request; the walker is not involved.
func runProjectsFirstPage(cfg projectsConfig, exec executor, rt renderTarget, req apiclient.Request) (Outcome, error) {
	if cfg.perPageSet {
		// Pass the value through as-is — no client-side clamp (paging's contract): an
		// out-of-range value surfaces the API's rejection rather than being ignored.
		q := cloneQuery(req.Query)
		q.Set("per_page", strconv.Itoa(cfg.perPage))
		req.Query = q
	}

	if machineFmt, ok := rt.format.MachineFormat(); ok {
		var page glassfrog.Page[json.RawMessage]
		if _, err := exec.Execute(cfg.reqCtx, req, &page); err != nil {
			return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
		}
		doc, rerr := aggregateRawData(machineFmt, page.Data)
		if rerr != nil {
			fmt.Fprintln(cfg.stderr, rerr.Error())
			return RuntimeError, rerr
		}
		_, _ = cfg.stdout.Write(doc)
		if page.Meta.Pagination.HasNextPage {
			fmt.Fprintln(cfg.stderr, moreRoleProjectsNote)
		}
		return Success, nil
	}

	var page glassfrog.Page[glassfrog.Project]
	if _, err := exec.Execute(cfg.reqCtx, req, &page); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	view := render.ProjectsView{Data: page.Data}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceProjects, rt.format, view); outcome != Success {
		return outcome, rerr
	}
	if page.Meta.Pagination.HasNextPage {
		fmt.Fprintln(cfg.stderr, moreRoleProjectsNote)
	}
	return Success, nil
}

// reportIncompleteProjectsWalk writes the mid-walk incomplete note (the partial
// set is already on stdout) and returns the classified non-zero outcome. It
// mirrors policies.go's reportIncompletePoliciesWalk with the projects wording: the
// Stop error is refined once so a non-2xx splits into permission/rate-limit (015),
// and the note text, the classified outcome, and the returned error all derive from
// that same refined value (the reportClientError invariant).
func reportIncompleteProjectsWalk(stderr io.Writer, stop error) (Outcome, error) {
	refined := refineClientError(stop)
	fmt.Fprintf(stderr, incompleteRoleProjectsWalkNote+"\n", refined.Error())
	return classifyClientError(refined), refined
}

// projectsQuery builds the GET /roles/{id}/projects query from the three combinable
// filters. --query (sent as `q`) and --tag (sent as `tag`) are free text sent ONLY
// when the flag was provided (Changed) AND non-empty (the 026 --depth optional-flag
// discipline; plan ADR-3): `--query ""`/`--tag ""` and an omitted flag both behave
// as no filter. --status (sent as `status`) is sent when non-empty — it was already
// validated by runProjectsList against the shared set, so an empty value is the
// "no constraint" case. Each present filter is its own query parameter; the API
// applies them together. A nil return leaves the request unparameterised.
func projectsQuery(cfg projectsConfig) url.Values {
	q := url.Values{}
	if cfg.querySet && cfg.query != "" {
		q.Set("q", cfg.query)
	}
	if cfg.status != "" {
		q.Set("status", cfg.status)
	}
	if cfg.tagSet && cfg.tag != "" {
		q.Set("tag", cfg.tag)
	}
	if len(q) == 0 {
		return nil
	}
	return q
}

// projectsWalkOptions builds the paging options for the default walk. --per-page
// (016's WithPageSize) sizes the walk when provided (by presence, not value); the
// value is passed through as-is — no client-side clamp (paging's contract).
func projectsWalkOptions(cfg projectsConfig) []paging.Option {
	if cfg.perPageSet {
		return []paging.Option{paging.WithPageSize(cfg.perPage)}
	}
	return nil
}

// newProjectsCommand builds the runnable `projects` leaf (ADR-1): a guard-ready
// cobra command with a REQUIRED positional role id (Args: cobra.ExactArgs(1)), a
// non-empty Short, and SilenceErrors/SilenceUsage so runProjectsList owns its
// messages. It declares the list-only flags (--query/-q, --status, --tag,
// --first-page, --per-page) — these exist ONLY on `projects`, so passing one to
// `project` is a cobra unknown-flag usage error (the structural list-only guard,
// ADR-1). It reads the inherited persistent --base-url/--output flags, then
// delegates to the pure runProjectsList. The seam is injected so tests drive a fake
// one; production passes productionSeam{}.
func newProjectsCommand(seam projectsSeam) *cobra.Command {
	var (
		query     string
		status    string
		tag       string
		firstPage bool
		perPage   int
	)
	cmd := &cobra.Command{
		Use:   "projects <role-id>",
		Short: "List the projects a role owns, walking pages to completion",
		Long: "projects lists the projects owned by a role by its id, walking every page " +
			"to completion by default so the list is complete or plainly flagged incomplete. " +
			"Narrow it with a free-text --query, a --status filter, and/or a --tag. This is " +
			"the role-addressable project surface; `me projects` reads the authenticated " +
			"practitioner's own projects. To read one project with full detail use " +
			"`project <proj-id>`.",
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
			outcome, oerr := runProjectsList(projectsConfig{
				seam:           seam,
				baseURL:        baseURL,
				baseURLPresent: cmd.Flags().Changed(apiclient.FlagBaseURL),
				outputFlag:     outputFlag,
				outputPresent:  cmd.Flags().Changed(output.FlagOutput),
				id:             args[0],
				query:          query,
				// Presence, not value: q is sent only when --query is Changed AND
				// non-empty, so `--query ""` behaves as no filter (ADR-3).
				querySet:  cmd.Flags().Changed("query"),
				status:    status,
				tag:       tag,
				tagSet:    cmd.Flags().Changed("tag"),
				firstPage: firstPage,
				perPage:   perPage,
				// Presence, not value: a provided 0/negative --per-page must reach the
				// API rather than be silently ignored (paging's no-clamp contract).
				perPageSet: cmd.Flags().Changed("per-page"),
				reqCtx:     cmd.Context(),
				stdout:     cmd.OutOrStdout(),
				stderr:     cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Free-text search, sent as the endpoint's q parameter (omitted when empty)")
	cmd.Flags().StringVar(&status, "status", "", "Filter by project status (one of: "+strings.Join(supportedStatusNames(), ", ")+")")
	cmd.Flags().StringVar(&tag, "tag", "", "Filter by tag name, sent as the endpoint's tag parameter (omitted when empty)")
	cmd.Flags().BoolVar(&firstPage, "first-page", false, "Fetch only the first page and signal if more projects exist")
	cmd.Flags().IntVar(&perPage, "per-page", 0, "Page size for the walk (the API owns the valid range)")
	return cmd
}

// projectConfig carries everything runProjectGet needs, gathered by the `project`
// command's RunE. It declares no list flags — the single read has none (ADR-1).
type projectConfig struct {
	seam           projectsSeam
	baseURL        string // inherited persistent --base-url (may be empty)
	baseURLPresent bool   // whether --base-url was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	outputFlag     string // inherited persistent --output (may be empty), resolved before any request
	outputPresent  bool   // whether --output was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runProjectGet reads a single project by id (GET /projects/{id}). It resolves the
// output format FIRST (020), assembles the connection and builds the retrying
// executor, then sends one Execute (no walk). The id is escaped as a single path
// segment but passed through unvalidated (ADR-3) so an unknown/malformed id
// surfaces as the API's 404/4xx via the shared classifier — no local regex gate.
// A structured --output emits the raw {data: Project} payload verbatim (018); the
// human path decodes Document[Project] and renders the `project` template over a
// ProjectView carrying the full detail (mirroring runPolicyGet). It adds no new
// Outcome/ExitCode and never reads the token.
func runProjectGet(cfg projectConfig, id string) (Outcome, error) {
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.outputPresent, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	ctx := cfg.seam.assemble(cfg.baseURL, cfg.baseURLPresent)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	exec := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, cfg.seam.sleep(), cfg.stderr)

	// Escape the id as a single path segment: passed through unvalidated (ADR-3),
	// but a raw `/` or `..` must not redirect the request or traverse the path.
	// PathEscape is a no-op for a valid proj_… id and keeps a malformed/adversarial
	// id one opaque segment the API reports as a 404.
	req := apiclient.Request{Method: http.MethodGet, Path: "/projects/" + url.PathEscape(id)}

	if machineFmt, ok := rt.format.MachineFormat(); ok {
		var raw json.RawMessage
		if _, err := exec.Execute(cfg.reqCtx, req, &raw); err != nil {
			return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
		}
		doc, rerr := output.RenderSuccess(machineFmt, raw)
		if rerr != nil {
			// Buffer-then-write: a render failure leaves stdout empty and maps to
			// RuntimeError(1). The error is token-free (018 contract).
			fmt.Fprintln(cfg.stderr, rerr.Error())
			return RuntimeError, rerr
		}
		_, _ = cfg.stdout.Write(doc)
		return Success, nil
	}

	var doc glassfrog.Document[glassfrog.Project]
	if _, err := exec.Execute(cfg.reqCtx, req, &doc); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	view := render.ProjectView{Project: doc.Data}
	return writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceProject, rt.format, view)
}

// newProjectCommand builds the runnable `project` leaf (ADR-1): a guard-ready cobra
// command with a REQUIRED positional project id (Args: cobra.ExactArgs(1)), a
// non-empty Short, and SilenceErrors/SilenceUsage so runProjectGet owns its
// messages. It declares NO list flags — so passing --query/-q, --status, --tag,
// --first-page, or --per-page is a cobra unknown-flag usage error before any
// request (this is how the spec's "filters apply only to the list" is enforced —
// no hand-rolled cross-combo guard, ADR-1). It reads the inherited persistent
// --base-url/--output flags, then delegates to the pure runProjectGet. The seam is
// injected so tests drive a fake one; production passes productionSeam{}.
func newProjectCommand(seam projectsSeam) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project <proj-id>",
		Short: "Read a single project by its id, with its full detail",
		Long: "project reads one project by its id and prints its status, description, " +
			"owning role, parent, the sub-projects/actions presence signals, tags, " +
			"timestamps, link, and note. To list the projects a role owns use " +
			"`projects <role-id>`.",
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
			outcome, oerr := runProjectGet(projectConfig{
				seam:           seam,
				baseURL:        baseURL,
				baseURLPresent: cmd.Flags().Changed(apiclient.FlagBaseURL),
				outputFlag:     outputFlag,
				outputPresent:  cmd.Flags().Changed(output.FlagOutput),
				reqCtx:         cmd.Context(),
				stdout:         cmd.OutOrStdout(),
				stderr:         cmd.ErrOrStderr(),
			}, args[0])
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	return cmd
}
