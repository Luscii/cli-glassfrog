package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/paging"
	"github.com/Luscii/cli-glassfrog/internal/render"
	"github.com/spf13/cobra"
)

// incompletePoliciesNote is the stderr line the default walk writes when it stops
// on an error after gathering at least one page: the partial set is already on
// stdout, so this names the cause and marks the list incomplete — a partial list
// is never silently presented as complete (CONSTITUTION VI; interface-cli). The
// %s is the cause; the command exits non-zero (classifyClientError(Stop)). It is
// the policies-worded sibling of roles.go's incompleteWalkNote.
const incompletePoliciesNote = "note: result is incomplete — %s; the policies shown are a partial set"

// morePoliciesNote is the stderr line the --first-page opt-out writes when the
// first page reports more pages exist: the operator chose the boundary, so this is
// not an error (exit 0) — it just keeps a partial list from being read as complete
// (interface-cli; CONSTITUTION VI).
const morePoliciesNote = "note: more policies exist than shown; re-run without --first-page to fetch all"

// policiesSeam supplies everything the policy reads need from the outside, so the
// run* functions are pure over injected values and every branch runs offline. It
// is the same shape as rolesSeam/subrolesSeam (assemble + newClient + sleep +
// resolveSelection + readTemplateSource), so productionSeam satisfies it unchanged and the existing test
// fakes drive it. The reads build the RetryExecutor-wrapped *Client once from these
// and hand it to both a direct Execute (the single policy read) and paging.All
// (the list walk). It never reads ctx.Cred.Token — the token rides 007's
// AuthTransport inside the client. Shared by both `policies` and `policy`.
type policiesSeam interface {
	assemble(baseURL string) apiclient.ConnectionContext
	newClient(ctx apiclient.ConnectionContext) (*apiclient.Client, error)
	sleep() func(time.Duration)
	resolveSelection(flagValue string) (output.Selection, error)
	readTemplateSource(ref output.TemplateRef) (string, error)
}

// policiesConfig carries everything runPoliciesList needs, gathered by the
// command's RunE. Keeping the read a function of injected values makes the whole
// flow — resolve, assemble, build, walk, render/classify — testable over a fake
// transport with no real network or ~/.glassfrogrc.
type policiesConfig struct {
	seam       policiesSeam
	baseURL    string // inherited persistent --base-url (may be empty)
	outputFlag string // inherited persistent --output (may be empty), resolved before any request
	id         string // the required positional role id (ExactArgs(1))

	query      string // --query/-q free-text search, sent verbatim as q
	querySet   bool   // whether --query was provided (Changed); q is sent only when set AND non-empty
	firstPage  bool
	perPage    int
	perPageSet bool // whether --per-page was provided (Changed); presence, not value

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runPoliciesList is the pure orchestration the `policies` leaf delegates to:
// resolve the output format (020) FIRST, then assemble the connection and build
// the retrying executor, then walk GET /roles/{id}/policies to completion. There
// is NO local input validation (plan ADR-3): the role id is a free identifier
// passed through to a clean 404, and --query is a free-text search string, not a
// closed enum. It adds no new Outcome/ExitCode and never reads the token.
func runPoliciesList(cfg policiesConfig) (Outcome, error) {
	// 1. Resolve the render target FIRST (020 widened by 035): a present-but-invalid
	//    selector — or, for a user template, a missing/unparseable source or empty
	//    stdin — fails fast as a usage error before any assembly or request.
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	// 2. Resolve the connection and build the client + retrying executor. A
	//    base-URL error surfaces here (no doomed send); classify + report it.
	ctx := cfg.seam.assemble(cfg.baseURL)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	exec := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, cfg.seam.sleep(), cfg.stderr)

	return runPoliciesListWalk(cfg, exec, rt)
}

// runPoliciesListWalk walks GET /roles/{id}/policies. The output format changes
// ONLY how the gathered set is rendered — never how much is fetched: every format
// walks to completion by default and signals incompleteness the same way (a stderr
// note, never a silently short list). --first-page opts out to a single page. This
// is the 025 roles-list shape with Policy items and the per-role policies path. The
// id is escaped as one path segment (passed through unvalidated per ADR-3, but a
// raw `/`/`..` must not redirect/traverse).
func runPoliciesListWalk(cfg policiesConfig, exec executor, rt renderTarget) (Outcome, error) {
	path := "/roles/" + url.PathEscape(cfg.id) + "/policies"
	req := apiclient.Request{Method: http.MethodGet, Path: path, Query: policiesQuery(cfg)}

	if cfg.firstPage {
		return runPoliciesFirstPage(cfg, exec, rt, req)
	}

	// Structured: walk to completion preserving each policy's raw bytes, then emit
	// the aggregated {data:[…]} document (reusing the resource-neutral aggregator).
	if machineFmt, ok := rt.format.MachineFormat(); ok {
		res := paging.All[json.RawMessage](cfg.reqCtx, exec, req, policiesWalkOptions(cfg)...)
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
			return reportIncompletePoliciesWalk(cfg.stderr, res.Stop)
		}
		return Success, nil
	}

	// Human / user template: walk to completion, render the policies projection (a
	// block per policy; an empty set renders `No policies.`).
	res := paging.All[glassfrog.Policy](cfg.reqCtx, exec, req, policiesWalkOptions(cfg)...)
	if res.Stop != nil && len(res.Records) == 0 {
		// A walk that stopped before gathering any record is a clean failure (e.g. a
		// first-page transport/auth/API error): no partial set to show.
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, res.Stop)
	}
	view := render.PoliciesView{Policies: res.Records}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourcePolicies, rt.format, view); outcome != Success {
		return outcome, rerr
	}
	if res.Stop != nil {
		return reportIncompletePoliciesWalk(cfg.stderr, res.Stop)
	}
	return Success, nil
}

// runPoliciesFirstPage performs the --first-page opt-out: a single
// GET /roles/{id}/policies page (no walk) in EVERY format, with one stderr note
// when the API reports more pages exist (still exit 0 — the operator chose the
// boundary). The structured path emits the same {data:[…]} envelope the default
// walk does; the human path renders the projection. --per-page (if set) sizes the
// single request; the walker is not involved.
func runPoliciesFirstPage(cfg policiesConfig, exec executor, rt renderTarget, req apiclient.Request) (Outcome, error) {
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
			fmt.Fprintln(cfg.stderr, morePoliciesNote)
		}
		return Success, nil
	}

	var page glassfrog.Page[glassfrog.Policy]
	if _, err := exec.Execute(cfg.reqCtx, req, &page); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	view := render.PoliciesView{Policies: page.Data}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourcePolicies, rt.format, view); outcome != Success {
		return outcome, rerr
	}
	if page.Meta.Pagination.HasNextPage {
		fmt.Fprintln(cfg.stderr, morePoliciesNote)
	}
	return Success, nil
}

// reportIncompletePoliciesWalk writes the mid-walk incomplete note (the partial
// set is already on stdout) and returns the classified non-zero outcome. It
// mirrors roles.go's reportIncompleteWalk with the policies wording: the Stop error
// is refined once so a non-2xx splits into permission/rate-limit (015), and the
// note text, the classified outcome, and the returned error all derive from that
// same refined value (the reportClientError invariant).
func reportIncompletePoliciesWalk(stderr io.Writer, stop error) (Outcome, error) {
	refined := refineClientError(stop)
	fmt.Fprintf(stderr, incompletePoliciesNote+"\n", refined.Error())
	return classifyClientError(refined), refined
}

// policiesQuery builds the GET /roles/{id}/policies query. --query is sent as the
// endpoint's `q` parameter ONLY when the flag was provided (Changed) AND non-empty
// (the 026 --depth optional-flag discipline; plan ADR-3): `--query ""` and an
// omitted flag both behave as no filter. The value is free text passed through
// verbatim — no enum validation. A nil return leaves the request unparameterised.
func policiesQuery(cfg policiesConfig) url.Values {
	if cfg.querySet && cfg.query != "" {
		return url.Values{"q": {cfg.query}}
	}
	return nil
}

// policiesWalkOptions builds the paging options for the default walk. --per-page
// (016's WithPageSize) sizes the walk when provided (by presence, not value); the
// value is passed through as-is — no client-side clamp (paging's contract).
func policiesWalkOptions(cfg policiesConfig) []paging.Option {
	if cfg.perPageSet {
		return []paging.Option{paging.WithPageSize(cfg.perPage)}
	}
	return nil
}

// policyConfig carries everything runPolicyGet needs, gathered by the `policy`
// command's RunE. It declares no list flags — the single read has none (ADR-1).
type policyConfig struct {
	seam       policiesSeam
	baseURL    string // inherited persistent --base-url (may be empty)
	outputFlag string // inherited persistent --output (may be empty), resolved before any request

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runPolicyGet reads a single policy by id (GET /policies/{id}). It resolves the
// output format FIRST (020), assembles the connection and builds the retrying
// executor, then sends one Execute (no walk). The id is escaped as a single path
// segment but passed through unvalidated (ADR-3) so an unknown/malformed id
// surfaces as the API's 404/4xx via the shared classifier — no local regex gate.
// A structured --output emits the raw {data: Policy} payload verbatim (018); the
// human path decodes Document[Policy] and renders the `policy` template over a
// PolicyView carrying the full body. It adds no new Outcome/ExitCode and never
// reads the token.
func runPolicyGet(cfg policyConfig, id string) (Outcome, error) {
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	ctx := cfg.seam.assemble(cfg.baseURL)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	exec := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, cfg.seam.sleep(), cfg.stderr)

	// Escape the id as a single path segment: passed through unvalidated (ADR-3),
	// but a raw `/` or `..` must not redirect the request or traverse the path.
	// PathEscape is a no-op for a valid pol_… id and keeps a malformed/adversarial
	// id one opaque segment the API reports as a 404.
	req := apiclient.Request{Method: http.MethodGet, Path: "/policies/" + url.PathEscape(id)}

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

	var doc glassfrog.Document[glassfrog.Policy]
	if _, err := exec.Execute(cfg.reqCtx, req, &doc); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	view := render.PolicyView{Policy: doc.Data}
	return writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourcePolicy, rt.format, view)
}

// newPolicyCommand builds the runnable `policy` leaf (ADR-1): a guard-ready cobra
// command with a REQUIRED positional policy id (Args: cobra.ExactArgs(1)), a
// non-empty Short, and SilenceErrors/SilenceUsage so runPolicyGet owns its
// messages. It declares NO list flags — so passing --query/-q, --first-page, or
// --per-page is a cobra unknown-flag usage error before any request (this is how
// the spec's "the search filter applies only to the list" is enforced — no
// hand-rolled cross-combo guard, ADR-1). It reads the inherited persistent
// --base-url/--output flags, then delegates to the pure runPolicyGet. The seam is
// injected so tests drive a fake one; production passes productionSeam{}.
func newPolicyCommand(seam policiesSeam) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy <pol-id>",
		Short: "Read a single policy by its id, including its full body",
		Long: "policy reads one policy by its id and prints its title and full body " +
			"verbatim, with the role/domain it governs and its timestamps. To list the " +
			"policies on a role use `policies <role-id>`.",
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
			outcome, oerr := runPolicyGet(policyConfig{
				seam:       seam,
				baseURL:    baseURL,
				outputFlag: outputFlag,
				reqCtx:     cmd.Context(),
				stdout:     cmd.OutOrStdout(),
				stderr:     cmd.ErrOrStderr(),
			}, args[0])
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	return cmd
}

// newPoliciesCommand builds the runnable `policies` leaf (ADR-1): a guard-ready
// cobra command with a REQUIRED positional role id (Args: cobra.ExactArgs(1)), a
// non-empty Short, and SilenceErrors/SilenceUsage so runPoliciesList owns its
// messages. It declares the list-only flags (--query/-q, --first-page, --per-page)
// — these exist ONLY on `policies`, so passing one to `policy` is a cobra
// unknown-flag usage error (the structural list-only guard, ADR-1). It reads the
// inherited persistent --base-url/--output flags, then delegates to the pure
// runPoliciesList. The seam is injected so tests drive a fake one; production
// passes productionSeam{}.
func newPoliciesCommand(seam policiesSeam) *cobra.Command {
	var (
		query     string
		firstPage bool
		perPage   int
	)
	cmd := &cobra.Command{
		Use:   "policies <role-id>",
		Short: "List the policies governing a role, walking pages to completion",
		Long: "policies lists the policies on a role's interior by its id, walking every " +
			"page to completion by default so the list is complete or plainly flagged " +
			"incomplete. Narrow it with a free-text --query. This is the addressable policy " +
			"surface; `roles <id> --include=policies` embeds policies inline on a role. To " +
			"read one policy with its full body use `policy <pol-id>`.",
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
			outcome, oerr := runPoliciesList(policiesConfig{
				seam:       seam,
				baseURL:    baseURL,
				outputFlag: outputFlag,
				id:         args[0],
				query:      query,
				// Presence, not value: q is sent only when --query is Changed AND
				// non-empty, so `--query ""` behaves as no filter (ADR-3).
				querySet:  cmd.Flags().Changed("query"),
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
	cmd.Flags().BoolVar(&firstPage, "first-page", false, "Fetch only the first page and signal if more policies exist")
	cmd.Flags().IntVar(&perPage, "per-page", 0, "Page size for the walk (the API owns the valid range)")
	return cmd
}
