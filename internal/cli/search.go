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

// searchPageSize is the per_page the search walk requests by default: the
// GET /search per_page MAXIMUM (100). It MUST be set explicitly because paging's
// generic default (500) exceeds the /search cap and the API rejects it with a 400
// (plan Cross-cutting / interface). --per-page overrides it; the API owns the
// 1–100 range (no client-side clamp — paging's contract).
const searchPageSize = 100

// incompleteSearchNote is the stderr line the default walk writes when it stops on
// an error after gathering at least one page: the partial set is already on
// stdout, so this names the cause and marks the list incomplete — a partial list
// is never silently presented as complete (CONSTITUTION VI; interface-cli). The
// %s is the cause; the command exits non-zero (classifyClientError(Stop)). It is
// the search-worded sibling of roles.go's incompleteWalkNote.
const incompleteSearchNote = "note: result is incomplete — %s; the results shown are a partial set"

// moreSearchNote is the stderr line the --first-page opt-out writes when the first
// page reports more pages exist: the operator chose the boundary, so this is not
// an error (exit 0) — it just keeps a partial list from being read as complete
// (interface-cli; CONSTITUTION VI).
const moreSearchNote = "note: more results exist than shown; re-run without --first-page to fetch all"

// supportedSearchTypes is the closed 8-value enum the --types flag accepts — the
// resource kinds GET /search ranks across. A value outside it is rejected fail-fast
// (the API would otherwise 400 or silently narrow — plan ADR-3). A response `type`
// value is NOT validated against this set: the API owns the response vocabulary
// (this guards INPUT only).
var supportedSearchTypes = map[string]bool{
	"role":    true,
	"note":    true,
	"project": true,
	"action":  true,
	"skill":   true,
	"actor":   true,
	"policy":  true,
	"domain":  true,
}

// searchSeam supplies everything the `search` read needs from the outside, so
// runSearch is pure over injected values and every branch runs offline. Same shape
// as rolesSeam/subrolesSeam/domainsSeam (assemble + newClient + sleep +
// resolveSelection + readTemplateSource — 035 widened resolveFormat into the
// discriminated selection), so productionSeam satisfies it unchanged and the
// existing test fakes drive it. It never reads ctx.Cred.Token — the token rides
// 007's AuthTransport in the client.
type searchSeam interface {
	assemble(baseURL string, baseURLPresent bool) apiclient.ConnectionContext
	newClient(ctx apiclient.ConnectionContext) (*apiclient.Client, error)
	sleep() func(time.Duration)
	resolveSelection(flagValue string, flagPresent bool) (output.Selection, error)
	readTemplateSource(ref output.TemplateRef) (string, error)
}

// searchConfig carries everything runSearch needs, gathered by the command's
// RunE. Keeping runSearch a function of injected values makes the whole read —
// validate, assemble, build, walk, render/classify — testable over a fake
// transport with no real network or ~/.glassfrogrc.
type searchConfig struct {
	seam           searchSeam
	baseURL        string // inherited persistent --base-url (may be empty)
	baseURLPresent bool   // whether --base-url was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	outputFlag     string // inherited persistent --output (may be empty), resolved before any request
	outputPresent  bool   // whether --output was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	query          string // the required positional full-text query (ExactArgs(1)), forwarded verbatim

	types      []string
	firstPage  bool
	perPage    int
	perPageSet bool // whether --per-page was provided (Changed); presence, not value

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runSearch is the pure orchestration the `search` leaf delegates to: resolve the
// output format FIRST, validate --types (reject-unknown) before any assembly or
// request, assemble the connection and build the retrying executor, then walk
// GET /search to completion (reusing 025/026's walk + --first-page opt-out
// verbatim, parameterized on SearchResult — ADR-4). It adds no new Outcome/ExitCode
// and never reads the token.
func runSearch(cfg searchConfig) (Outcome, error) {
	// 1. Resolve the render target FIRST (020 widened by 035, ADR-1/ADR-4): a
	//    present-but-invalid selector — or, for a user template, a missing/unparseable
	//    source or empty stdin — fails fast as a usage error before any assembly or
	//    request.
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.outputPresent, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	// 2. Validate --types against the closed 8-value set BEFORE any request (a bad
	//    value would 400 or silently narrow at the API — plan ADR-3). The query is
	//    NOT validated locally: the API owns websearch interpretation and rejects a
	//    malformed query with 400 (ADR-1).
	if err := validateTypes(cfg.types); err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}

	// 3. Resolve the connection and build the client + retrying executor. A base-URL
	//    error surfaces here (no doomed send); classify + report it.
	ctx := cfg.seam.assemble(cfg.baseURL, cfg.baseURLPresent)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	exec := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, cfg.seam.sleep(), cfg.stderr)

	return runSearchList(cfg, exec, rt)
}

// runSearchList walks GET /search. The output format changes ONLY how the gathered
// set is rendered — never how much is fetched: every format walks to completion by
// default and signals incompleteness the same way (a stderr note, never a silently
// short list). --first-page opts out to a single page. The query (and --types when
// set) ride EVERY page of the walk — paging.All clones and preserves the base
// request's query across pages. Result order is the API's relevance order,
// preserved exactly (no client re-sort/de-dup/filter — ADR-2).
func runSearchList(cfg searchConfig, exec executor, rt renderTarget) (Outcome, error) {
	req := apiclient.Request{Method: http.MethodGet, Path: "/search", Query: searchQuery(cfg)}

	if cfg.firstPage {
		return runSearchFirstPage(cfg, exec, rt, req)
	}

	// Structured: walk to completion preserving each row's raw bytes, then emit the
	// aggregated {data:[…]} document (reusing the resource-neutral aggregator).
	if machineFmt, ok := rt.format.MachineFormat(); ok {
		res := paging.All[json.RawMessage](cfg.reqCtx, exec, req, searchWalkOptions(cfg)...)
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
			return reportIncompleteSearchWalk(cfg.stderr, res.Stop)
		}
		return Success, nil
	}

	// Human / user template: walk to completion, render the search projection (a
	// `type`-badged block per hit in relevance order; an empty set renders
	// `No results.`).
	res := paging.All[glassfrog.SearchResult](cfg.reqCtx, exec, req, searchWalkOptions(cfg)...)
	if res.Stop != nil && len(res.Records) == 0 {
		// A walk that stopped before gathering any record is a clean failure (e.g. a
		// first-page transport/auth/API error): no partial set to show.
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, res.Stop)
	}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceSearch, rt.format, render.NewSearchView(res.Records)); outcome != Success {
		return outcome, rerr
	}
	if res.Stop != nil {
		return reportIncompleteSearchWalk(cfg.stderr, res.Stop)
	}
	return Success, nil
}

// runSearchFirstPage performs the --first-page opt-out: a single GET /search page
// (no walk) in EVERY format, with one stderr note when the API reports more pages
// exist (still exit 0 — the operator chose the boundary). The structured path emits
// the same {data:[…]} envelope the default walk does; the human path renders the
// projection. The single request carries per_page at the resolved size (default
// 100, the /search max; --per-page overrides) so the first page is a full page; the
// walker is not involved.
func runSearchFirstPage(cfg searchConfig, exec executor, rt renderTarget, req apiclient.Request) (Outcome, error) {
	q := cloneQuery(req.Query)
	q.Set("per_page", strconv.Itoa(searchPageSizeFor(cfg)))
	req.Query = q

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
			fmt.Fprintln(cfg.stderr, moreSearchNote)
		}
		return Success, nil
	}

	var page glassfrog.Page[glassfrog.SearchResult]
	if _, err := exec.Execute(cfg.reqCtx, req, &page); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceSearch, rt.format, render.NewSearchView(page.Data)); outcome != Success {
		return outcome, rerr
	}
	if page.Meta.Pagination.HasNextPage {
		fmt.Fprintln(cfg.stderr, moreSearchNote)
	}
	return Success, nil
}

// reportIncompleteSearchWalk writes the mid-walk incomplete note (the partial set
// is already on stdout) and returns the classified non-zero outcome. It mirrors
// roles.go's reportIncompleteWalk with the search wording: the Stop error is
// refined once so a non-2xx splits into permission/rate-limit (015), and the note
// text, the classified outcome, and the returned error all derive from that same
// refined value (the reportClientError invariant).
func reportIncompleteSearchWalk(stderr io.Writer, stop error) (Outcome, error) {
	refined := refineClientError(stop)
	fmt.Fprintf(stderr, incompleteSearchNote+"\n", refined.Error())
	return classifyClientError(refined), refined
}

// searchQuery builds the GET /search query: the positional `query` is attached
// VERBATIM (byte-for-byte — no parse/escape/normalize/split; the API owns websearch
// interpretation — ADR-1), always present (it is required). `types` is attached as
// the comma-separated value ONLY when --types is set (omitting it requests all
// types, the API default — never spelled out as the full list — ADR-3).
func searchQuery(cfg searchConfig) url.Values {
	q := url.Values{}
	q.Set("query", cfg.query)
	if len(cfg.types) > 0 {
		q.Set("types", strings.Join(cfg.types, ","))
	}
	return q
}

// searchWalkOptions builds the paging options for the default walk: per_page at the
// resolved size (default 100, the /search max — overriding paging's generic 500
// which /search rejects; --per-page overrides, passed through as-is with no
// client-side clamp per paging's contract).
func searchWalkOptions(cfg searchConfig) []paging.Option {
	return []paging.Option{paging.WithPageSize(searchPageSizeFor(cfg))}
}

// searchPageSizeFor resolves the per_page size: --per-page when provided (by
// presence, not value — a provided 0/negative reaches the API rather than being
// silently ignored), otherwise the /search default maximum (100).
func searchPageSizeFor(cfg searchConfig) int {
	if cfg.perPageSet {
		return cfg.perPage
	}
	return searchPageSize
}

// validateTypes rejects any --types value outside the closed 8-value search type
// set, before any request (the reject-unknown fail-fast shape, pinned by a
// transport tripwire — plan ADR-3). It delegates to the flag-agnostic
// validateClosedFlagSet so the message names --types (not --include) while sharing
// the landed sorted/quoted/named-set formatting. An empty list (the flag absent) is
// valid — it requests all types.
func validateTypes(types []string) error {
	return validateClosedFlagSet("--types", types, supportedSearchTypes)
}

// newSearchCommand builds the runnable `search` leaf (ADR-1): a guard-ready cobra
// command with a REQUIRED positional query (Args: cobra.ExactArgs(1) — both a
// missing query and a >1 positional are fail-fast usage errors before assembly), a
// non-empty Short, and SilenceErrors/SilenceUsage so runSearch owns its messages.
// It is an org-wide, cross-type sibling command — child of no resource group. It
// declares the local --types/--first-page/--per-page flags, reads the inherited
// persistent --base-url/--output flags, then delegates to the pure runSearch. The
// seam is injected so tests drive a fake one; production passes productionSeam{}.
func newSearchCommand(seam searchSeam) *cobra.Command {
	var (
		types     []string
		firstPage bool
		perPage   int
	)
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search across every resource type by topic, ranked by relevance",
		Long: "search runs one relevance-ranked full-text query across every governance " +
			"resource type (roles, notes, projects, actions, skills, actors, policies, " +
			"domains) and prints a uniform ranked list — each result's type and id are the " +
			"bridge into the matching read command. The query is forwarded to the API " +
			"verbatim, so a multi-word query MUST be quoted as one argument " +
			"(`search \"strategy review -archived\"`). Scope it with --types; the list walks " +
			"every page to completion by default, or stops at one page with --first-page.",
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
			outcome, oerr := runSearch(searchConfig{
				seam:           seam,
				baseURL:        baseURL,
				baseURLPresent: cmd.Flags().Changed(apiclient.FlagBaseURL),
				outputFlag:     outputFlag,
				outputPresent:  cmd.Flags().Changed(output.FlagOutput),
				query:          args[0],
				types:          types,
				firstPage:      firstPage,
				perPage:        perPage,
				// Presence, not value: --per-page=0 must reach the API rather than be
				// silently ignored (paging's no-clamp contract).
				perPageSet: cmd.Flags().Changed("per-page"),
				reqCtx:     cmd.Context(),
				stdout:     cmd.OutOrStdout(),
				stderr:     cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	cmd.Flags().StringSliceVar(&types, "types", nil, "Scope to resource types (role,note,project,action,skill,actor,policy,domain)")
	cmd.Flags().BoolVar(&firstPage, "first-page", false, "Fetch only the first page and signal if more results exist")
	cmd.Flags().IntVar(&perPage, "per-page", 0, "Page size for the walk (the API owns the 1–100 range)")
	return cmd
}
