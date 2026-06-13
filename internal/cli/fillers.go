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

// incompleteFillersWalkNote is the stderr line the default walk writes when it
// stops on an error after gathering at least one page: the partial set is already
// on stdout, so this names the cause and marks the list incomplete — a partial list
// is never silently presented as complete (CONSTITUTION VI; interface-cli). The %s
// is the cause; the command exits non-zero (classifyClientError(Stop)). It is the
// fillers-worded sibling of actors.go's incompleteActorsWalkNote.
const incompleteFillersWalkNote = "note: result is incomplete — %s; the fillers shown are a partial set"

// moreFillersNote is the stderr line the --first-page opt-out writes when the first
// page reports more pages exist: the operator chose the boundary, so this is not an
// error (exit 0) — it just keeps a partial list from being read as complete
// (interface-cli; CONSTITUTION VI).
const moreFillersNote = "note: more fillers exist than shown; re-run without --first-page to fetch all"

// fillersSeam supplies everything the `fillers` read needs from the outside, so
// runFillersList is pure over injected values and every branch runs offline. Same
// shape as projectsSeam/actorsSeam (assemble + newClient + sleep + resolveSelection
// + readTemplateSource), so productionSeam satisfies it unchanged and the existing
// test fakes drive it. It never reads ctx.Cred.Token — the token rides 007's
// AuthTransport in the client.
type fillersSeam interface {
	assemble(baseURL string, baseURLPresent bool) apiclient.ConnectionContext
	newClient(ctx apiclient.ConnectionContext) (*apiclient.Client, error)
	sleep() func(time.Duration)
	resolveSelection(flagValue string, flagPresent bool) (output.Selection, error)
	readTemplateSource(ref output.TemplateRef) (string, error)
}

// fillersConfig carries everything runFillersList needs, gathered by the command's
// RunE. Keeping the read a function of injected values makes the whole flow —
// resolve, assemble, build, walk, render/classify — testable over a fake transport
// with no real network or ~/.glassfrogrc. `fillers` takes a REQUIRED positional
// role id (cobra.ExactArgs(1)) and NO filter flags and NO --include (plan ADR-3) —
// the endpoint offers none beyond the default include + pagination.
type fillersConfig struct {
	seam           fillersSeam
	baseURL        string // inherited persistent --base-url (may be empty)
	baseURLPresent bool   // whether --base-url was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	outputFlag     string // inherited persistent --output (may be empty), resolved before any request
	outputPresent  bool   // whether --output was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	id             string // the required positional role id (ExactArgs(1))

	firstPage  bool
	perPage    int
	perPageSet bool // whether --per-page was provided (Changed); presence, not value

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runFillersList is the pure orchestration the `fillers` leaf delegates to: resolve
// the output format (020) FIRST — the ONLY pre-assembly check, since there is no
// closed-enum input to validate (plan ADR-3, unlike 038's --status / 048's --kind).
// A present-but-invalid selector is a usage error with NO request issued. Then
// assemble the connection, build the retrying executor, and walk
// GET /roles/{role_id}/assignments to completion. The role id is a free identifier
// passed through to a clean 404. It adds no new Outcome/ExitCode and never reads the
// token.
func runFillersList(cfg fillersConfig) (Outcome, error) {
	// 1. Resolve the render target FIRST (020 widened by 035): a present-but-invalid
	//    selector — or, for a user template, a missing/unparseable source or empty
	//    stdin — fails fast as a usage error before any assembly or request. There is
	//    no second (closed-enum) input to validate after it — the command exposes no
	//    filter flag (plan ADR-3) — so this is the only pre-assembly check.
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.outputPresent, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	// 2. Resolve the connection and build the client + retrying executor. A base-URL
	//    error surfaces here (no doomed send); classify + report it.
	ctx := cfg.seam.assemble(cfg.baseURL, cfg.baseURLPresent)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	exec := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, cfg.seam.sleep(), cfg.stderr)

	return runFillersListWalk(cfg, exec, rt)
}

// runFillersListWalk walks GET /roles/{role_id}/assignments. The output format
// changes ONLY how the gathered set is rendered — never how much is fetched: every
// format walks to completion by default and signals incompleteness the same way (a
// stderr note, never a silently short list). --first-page opts out to a single page.
// This is the 038/048 projects/actors walked-list shape with Assignment items. The
// id is escaped as one path segment (passed through unvalidated per ADR-3, but a raw
// `/`/`..` must not redirect/traverse). The request carries the endpoint's default
// include=actor implicitly, so each filler row has the actor's name/kind without a
// flag.
func runFillersListWalk(cfg fillersConfig, exec executor, rt renderTarget) (Outcome, error) {
	path := "/roles/" + url.PathEscape(cfg.id) + "/assignments"
	req := apiclient.Request{Method: http.MethodGet, Path: path}

	if cfg.firstPage {
		return runFillersFirstPage(cfg, exec, rt, req)
	}

	// Structured: walk to completion preserving each assignment's raw bytes, then emit
	// the aggregated {data:[…]} document (reusing the resource-neutral aggregator). A
	// user template resolves to a non-structured format (rt.tmpl != nil → Full), so
	// this branch is never taken under -o stdin/file (035).
	if machineFmt, ok := rt.format.MachineFormat(); ok {
		res := paging.All[json.RawMessage](cfg.reqCtx, exec, req, fillersWalkOptions(cfg)...)
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
			return reportIncompleteFillersWalk(cfg.stderr, res.Stop)
		}
		return Success, nil
	}

	// Human / user template: walk to completion, render the fillers projection (the new
	// 047 `fillers` key; an empty set renders `no fillers`).
	res := paging.All[glassfrog.Assignment](cfg.reqCtx, exec, req, fillersWalkOptions(cfg)...)
	if res.Stop != nil && len(res.Records) == 0 {
		// A walk that stopped before gathering any record is a clean failure (e.g. a
		// first-page transport/auth/API error): no partial set to show.
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, res.Stop)
	}
	view := render.FillersView{Data: res.Records}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceFillers, rt.format, view); outcome != Success {
		return outcome, rerr
	}
	if res.Stop != nil {
		return reportIncompleteFillersWalk(cfg.stderr, res.Stop)
	}
	return Success, nil
}

// runFillersFirstPage performs the --first-page opt-out: a single
// GET /roles/{role_id}/assignments page (no walk) in EVERY format, with one stderr
// note when the API reports more pages exist (still exit 0 — the operator chose the
// boundary). The structured path emits the same {data:[…]} envelope the default walk
// does; the human path renders the projection. --per-page (if set) sizes the single
// request; the walker is not involved.
func runFillersFirstPage(cfg fillersConfig, exec executor, rt renderTarget, req apiclient.Request) (Outcome, error) {
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
			fmt.Fprintln(cfg.stderr, moreFillersNote)
		}
		return Success, nil
	}

	var page glassfrog.Page[glassfrog.Assignment]
	if _, err := exec.Execute(cfg.reqCtx, req, &page); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	view := render.FillersView{Data: page.Data}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceFillers, rt.format, view); outcome != Success {
		return outcome, rerr
	}
	if page.Meta.Pagination.HasNextPage {
		fmt.Fprintln(cfg.stderr, moreFillersNote)
	}
	return Success, nil
}

// reportIncompleteFillersWalk writes the mid-walk incomplete note (the partial set
// is already on stdout) and returns the classified non-zero outcome. It mirrors
// actors.go's reportIncompleteActorsWalk with the fillers wording: the Stop error is
// refined once so a non-2xx splits into permission/rate-limit (015), and the note
// text, the classified outcome, and the returned error all derive from that same
// refined value (the reportClientError invariant).
func reportIncompleteFillersWalk(stderr io.Writer, stop error) (Outcome, error) {
	refined := refineClientError(stop)
	fmt.Fprintf(stderr, incompleteFillersWalkNote+"\n", refined.Error())
	return classifyClientError(refined), refined
}

// fillersWalkOptions builds the paging options for the default walk. --per-page
// (016's WithPageSize) sizes the walk when provided (by presence, not value); the
// value is passed through as-is — no client-side clamp (paging's contract).
func fillersWalkOptions(cfg fillersConfig) []paging.Option {
	if cfg.perPageSet {
		return []paging.Option{paging.WithPageSize(cfg.perPage)}
	}
	return nil
}

// newFillersCommand builds the runnable `fillers` leaf (ADR-1): a guard-ready cobra
// command with a REQUIRED positional role id (Args: cobra.ExactArgs(1) — an omitted
// or extra positional is a fail-fast usage error via cobra's own arg validator, no
// hand-rolled guard), a non-empty Short, and SilenceErrors/SilenceUsage so
// runFillersList owns its messages. It declares NO filter flags and NO --include
// (plan ADR-3) — the endpoint offers none beyond the default include + pagination —
// only the shared --first-page/--per-page walk flags. There is no singular sibling
// because the API exposes no GET /assignments/{id} (ADR-1). It reads the inherited
// persistent --base-url/--output flags, then delegates to the pure runFillersList.
// The seam is injected so tests drive a fake one; production passes productionSeam{}.
func newFillersCommand(seam fillersSeam) *cobra.Command {
	var (
		firstPage bool
		perPage   int
	)
	cmd := &cobra.Command{
		Use:   "fillers <role-id>",
		Short: "List the actors who fill a role, walking pages to completion",
		Long: "fillers lists the actors who fill a role by its id — the answer to \"whom do I " +
			"contact about this role?\" — walking every page to completion by default so the " +
			"list is complete or plainly flagged incomplete. Each filler is shown with its " +
			"focus and, for elected seats, its election expiry. The per_/agt_ id and kind " +
			"badge in each row distinguish a person from an agent and bridge into the " +
			"per-actor reads. It takes no filters and no --include — the role id is its only " +
			"input. Stop at one page with --first-page.",
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
			outcome, oerr := runFillersList(fillersConfig{
				seam:           seam,
				baseURL:        baseURL,
				baseURLPresent: cmd.Flags().Changed(apiclient.FlagBaseURL),
				outputFlag:     outputFlag,
				outputPresent:  cmd.Flags().Changed(output.FlagOutput),
				id:             args[0],
				firstPage:      firstPage,
				perPage:        perPage,
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
	cmd.Flags().BoolVar(&firstPage, "first-page", false, "Fetch only the first page and signal if more fillers exist")
	cmd.Flags().IntVar(&perPage, "per-page", 0, "Page size for the walk (the API owns the valid range)")
	return cmd
}
