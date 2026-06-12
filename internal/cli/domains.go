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

// incompleteDomainsNote is the stderr line the default walk writes when it stops
// on an error after gathering at least one page: the partial set is already on
// stdout, so this names the cause and marks the list incomplete — a partial list
// is never silently presented as complete (CONSTITUTION VI; interface-cli). The
// %s is the cause; the command exits non-zero (classifyClientError(Stop)). It is
// the domains-worded sibling of subroles.go's incompleteSubrolesNote.
const incompleteDomainsNote = "note: result is incomplete — %s; the domains shown are a partial set"

// moreDomainsNote is the stderr line the --first-page opt-out writes when the
// first page reports more pages exist: the operator chose the boundary, so this
// is not an error (exit 0) — it just keeps a partial list from being read as
// complete (interface-cli; CONSTITUTION VI).
const moreDomainsNote = "note: more domains exist than shown; re-run without --first-page to fetch all"

// domainsSeam supplies everything the `domains` read needs from the outside, so
// runDomains is pure over injected values and every branch runs offline. Same
// shape as subrolesSeam/rolesSeam (assemble + newClient + sleep + resolveSelection + readTemplateSource),
// so productionSeam satisfies it unchanged and the existing test fakes drive it.
// It never reads ctx.Cred.Token — the token rides 007's AuthTransport in the
// client.
type domainsSeam interface {
	assemble(baseURL string, baseURLPresent bool) apiclient.ConnectionContext
	newClient(ctx apiclient.ConnectionContext) (*apiclient.Client, error)
	sleep() func(time.Duration)
	resolveSelection(flagValue string) (output.Selection, error)
	readTemplateSource(ref output.TemplateRef) (string, error)
}

// domainsConfig carries everything runDomains needs, gathered by the command's
// RunE. Keeping runDomains a function of injected values makes the whole read —
// validate, assemble, build, walk, render/classify — testable over a fake
// transport with no real network or ~/.glassfrogrc.
type domainsConfig struct {
	seam           domainsSeam
	baseURL        string // inherited persistent --base-url (may be empty)
	baseURLPresent bool   // whether --base-url was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	outputFlag     string // inherited persistent --output (may be empty), resolved before any request
	id             string // the required positional role id (ExactArgs(1))

	query      string // --query/-q full-text search; sent only when the trimmed value is non-blank
	firstPage  bool
	perPage    int
	perPageSet bool // whether --per-page was provided (Changed); presence, not value

	// --include is declared on `domains` only to reject it: embedding related
	// resources is a single-read concern (it belongs on `domain`), so passing it
	// to the list is a usage error (interface "--include is rejected on the
	// role-scoped list"). Presence, not value.
	includeSet bool

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runDomains is the pure orchestration the `domains` leaf delegates to: resolve
// the output format, validate the flags (reject --include) fail-fast before any
// assembly or request, assemble the connection and build the retrying executor,
// then walk GET /roles/{id}/domains to completion (reusing 025's walk +
// --first-page opt-out verbatim — plan ADR-3), carrying the optional `q` search
// term on every page. It adds no new Outcome/ExitCode and never reads the token.
func runDomains(cfg domainsConfig) (Outcome, error) {
	// 1. Resolve the render target FIRST (020 widened by 035): a present-but-invalid
	//    selector — or, for a user template, a missing/unparseable source or empty
	//    stdin — fails fast as a usage error before any assembly or request.
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	// 2. Validate the flags BEFORE any assembly or request (fail-fast usage error,
	//    pinned by a tripwire transport): --include is a single-read concern.
	//    The id is NOT validated locally (ADR-4): the API 404s an unknown id.
	if err := validateDomainsFlags(cfg); err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}

	// 3. Resolve the connection and build the client + retrying executor.
	ctx := cfg.seam.assemble(cfg.baseURL, cfg.baseURLPresent)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	exec := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, cfg.seam.sleep(), cfg.stderr)

	return runDomainsList(cfg, exec, rt)
}

// runDomainsList walks GET /roles/{id}/domains. The output format changes ONLY
// how the gathered set is rendered — never how much is fetched: every format
// walks to completion by default and signals incompleteness the same way (a
// stderr note, never a silently short list). --first-page opts out to a single
// page. This is the 025/026 list shape with Domain items and the domains path,
// carrying the optional `q` search term on every page request. The id is escaped
// as one path segment (passed through unvalidated per ADR-4, but a raw `/`/`..`
// must not redirect/traverse).
func runDomainsList(cfg domainsConfig, exec executor, rt renderTarget) (Outcome, error) {
	path := "/roles/" + url.PathEscape(cfg.id) + "/domains"
	req := apiclient.Request{Method: http.MethodGet, Path: path, Query: domainsQuery(cfg)}

	if cfg.firstPage {
		return runDomainsFirstPage(cfg, exec, rt, req)
	}

	// Structured: walk to completion preserving each domain's raw bytes, then emit
	// the aggregated {data:[…]} document (reusing the resource-neutral aggregator).
	if machineFmt, ok := rt.format.MachineFormat(); ok {
		res := paging.All[json.RawMessage](cfg.reqCtx, exec, req, domainsWalkOptions(cfg)...)
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
			return reportIncompleteDomainsWalk(cfg.stderr, res.Stop)
		}
		return Success, nil
	}

	// Human / user template: walk to completion, render the domains projection (a
	// block per domain; an empty set renders `No domains.`).
	res := paging.All[glassfrog.Domain](cfg.reqCtx, exec, req, domainsWalkOptions(cfg)...)
	if res.Stop != nil && len(res.Records) == 0 {
		// A walk that stopped before gathering any record is a clean failure (e.g. a
		// first-page transport/auth/API error): no partial set to show.
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, res.Stop)
	}
	view := render.DomainsView{Domains: res.Records}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceDomains, rt.format, view); outcome != Success {
		return outcome, rerr
	}
	if res.Stop != nil {
		return reportIncompleteDomainsWalk(cfg.stderr, res.Stop)
	}
	return Success, nil
}

// runDomainsFirstPage performs the --first-page opt-out: a single
// GET /roles/{id}/domains page (no walk) in EVERY format, with one stderr note
// when the API reports more pages exist (still exit 0 — the operator chose the
// boundary). The structured path emits the same {data:[…]} envelope the default
// walk does; the human path renders the projection. --per-page (if set) sizes the
// single request; the walker is not involved. The `q` search term (when present)
// already rides req.Query.
func runDomainsFirstPage(cfg domainsConfig, exec executor, rt renderTarget, req apiclient.Request) (Outcome, error) {
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
			fmt.Fprintln(cfg.stderr, moreDomainsNote)
		}
		return Success, nil
	}

	var page glassfrog.Page[glassfrog.Domain]
	if _, err := exec.Execute(cfg.reqCtx, req, &page); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	view := render.DomainsView{Domains: page.Data}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceDomains, rt.format, view); outcome != Success {
		return outcome, rerr
	}
	if page.Meta.Pagination.HasNextPage {
		fmt.Fprintln(cfg.stderr, moreDomainsNote)
	}
	return Success, nil
}

// reportIncompleteDomainsWalk writes the mid-walk incomplete note (the partial
// set is already on stdout) and returns the classified non-zero outcome. It
// mirrors subroles.go's reportIncompleteSubrolesWalk: the Stop error is refined
// once so a non-2xx splits into permission/rate-limit (015), and the note text,
// the classified outcome, and the returned error all derive from that same
// refined value (the reportClientError invariant).
func reportIncompleteDomainsWalk(stderr io.Writer, stop error) (Outcome, error) {
	refined := refineClientError(stop)
	fmt.Fprintf(stderr, incompleteDomainsNote+"\n", refined.Error())
	return classifyClientError(refined), refined
}

// domainsQuery builds the GET /roles/{id}/domains query. The `q` full-text search
// is set ONLY when the trimmed --query value is non-blank (plan ADR-3 — a
// blank/whitespace term sends no `q`, matching the API's own ignore semantics);
// because it lives on req.Query, the walker carries it on every page request so
// search composes with the walk. A nil return leaves the request unparameterised.
func domainsQuery(cfg domainsConfig) url.Values {
	term := strings.TrimSpace(cfg.query)
	if term == "" {
		return nil
	}
	return url.Values{"q": {term}}
}

// domainsWalkOptions builds the paging options for the default walk. --per-page
// (016's WithPageSize) sizes the walk when provided (by presence, not value); the
// value is passed through as-is — no client-side clamp (paging's contract).
func domainsWalkOptions(cfg domainsConfig) []paging.Option {
	if cfg.perPageSet {
		return []paging.Option{paging.WithPageSize(cfg.perPage)}
	}
	return nil
}

// validateDomainsFlags rejects the domains flag misuses fail-fast, before any
// request (the 011/013 validate-before-call shape, pinned by a transport
// tripwire): --include embeds related resources, which is a single-read concern,
// so passing it to the role-scoped list is a usage error. The message names the
// misuse and the fix (and points at --query for the list's own search).
func validateDomainsFlags(cfg domainsConfig) error {
	if cfg.includeSet {
		return fmt.Errorf(
			"--include applies to the single `domain` read; use `glassfrog domain <id> --include policies` (the domains list takes --query to search)",
		)
	}
	return nil
}

// newDomainsCommand builds the runnable `domains` leaf (plan ADR-1): a guard-ready
// cobra command with a REQUIRED positional role id (Args: cobra.ExactArgs(1)), a
// non-empty Short cross-referencing the singular `domain` read, and
// SilenceErrors/SilenceUsage so runDomains owns its messages. It declares the
// list flags (--query/-q, --first-page, --per-page) and — to reject it with a
// friendly message rather than a bare cobra "unknown flag" — the --include flag
// it forbids. It reads the inherited persistent --base-url/--output flags, then
// delegates to the pure runDomains. The seam is injected so tests drive a fake
// one; production passes productionSeam{}.
func newDomainsCommand(seam domainsSeam) *cobra.Command {
	var (
		query     string
		firstPage bool
		perPage   int
		include   []string
	)
	cmd := &cobra.Command{
		Use:   "domains <id>",
		Short: "List the domains a role controls, walking pages to completion",
		Long: "domains lists the areas of control (domains) held by a role, walking every " +
			"page to completion by default so the list is complete or plainly flagged " +
			"incomplete. Search within the list with --query. To read one domain by its own " +
			"id (with its policies), use the singular `domain <dom-id>`.",
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
			outcome, oerr := runDomains(domainsConfig{
				seam:           seam,
				baseURL:        baseURL,
				baseURLPresent: cmd.Flags().Changed(apiclient.FlagBaseURL),
				outputFlag:     outputFlag,
				id:             args[0],
				query:          query,
				firstPage:      firstPage,
				perPage:        perPage,
				// Presence, not value: --per-page=0 must reach the API rather than be
				// silently ignored (paging's no-clamp contract).
				perPageSet: cmd.Flags().Changed("per-page"),
				includeSet: cmd.Flags().Changed("include"),
				reqCtx:     cmd.Context(),
				stdout:     cmd.OutOrStdout(),
				stderr:     cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Full-text search over the role's domains (the API q param); sent only when non-blank")
	cmd.Flags().BoolVar(&firstPage, "first-page", false, "Fetch only the first page and signal if more domains exist")
	cmd.Flags().IntVar(&perPage, "per-page", 0, "Page size for the walk (the API owns the valid range)")
	cmd.Flags().StringSliceVar(&include, "include", nil, "Not valid on the domains list — use `domain <id> --include policies` for a single domain's policies")
	return cmd
}
