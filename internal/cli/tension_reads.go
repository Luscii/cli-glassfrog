package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/paging"
	"github.com/Luscii/cli-glassfrog/internal/render"
	"github.com/spf13/cobra"
)

// incompleteTensionsWalkNote is the stderr line the default walk writes when it
// stops on an error after gathering at least one page: the partial set is already
// on stdout, so this names the cause and marks the list incomplete — a partial list
// is never silently presented as complete (CONSTITUTION VI; interface-cli). The %s
// is the cause; the command exits non-zero (classifyClientError(Stop)). It is the
// tension-worded sibling of projects.go's incompleteRoleProjectsWalkNote.
const incompleteTensionsWalkNote = "note: result is incomplete — %s; the tensions shown are a partial set"

// moreTensionsNote is the stderr line the --first-page opt-out writes when the
// first page reports more pages exist: the operator chose the boundary, so this is
// not an error (exit 0) — it just keeps a partial list from being read as complete
// (interface-cli; CONSTITUTION VI).
const moreTensionsNote = "note: more tensions exist than shown; re-run without --first-page to fetch all"

// tensionsConfig carries everything runTensionList needs, gathered by the `list`
// command's RunE. Keeping the read a function of injected values makes the whole
// flow — resolve, validate, assemble, build, walk, render/classify — testable over
// a fake transport with no real network or ~/.glassfrogrc. It reuses 042's
// tensionSeam (identical to projectsSeam; paging is a paging.All call in the body,
// not a seam method).
type tensionsConfig struct {
	seam           tensionSeam
	baseURL        string // inherited persistent --base-url (may be empty)
	baseURLPresent bool   // whether --base-url was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	outputFlag     string // inherited persistent --output (may be empty), resolved before any request
	outputPresent  bool   // whether --output was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	id             string // the required positional role id (ExactArgs(1))

	status string // --status filter, validated against the tension status set before any request

	firstPage  bool
	perPage    int
	perPageSet bool // whether --per-page was provided (Changed); presence, not value

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runTensionList is the pure orchestration the `tension list` leaf delegates to:
// resolve the output format (020, widened by 035) FIRST, then validate the one
// closed-enum input (--status) fail-fast via validateTensionStatus (an unsupported
// value is a usage error with NO request issued) — both pure checks run before any
// assembly, in the same output-first order as the sibling reads so error precedence
// is consistent. Then assemble the connection, build the retrying executor, and
// walk GET /roles/{role_id}/tensions to completion. The role id is a free
// identifier passed through to a clean 404 (plan ADR-3). It adds no new
// Outcome/ExitCode and never reads the token.
func runTensionList(cfg tensionsConfig) (Outcome, error) {
	// 1. Resolve the render target FIRST (020 widened by 035): a present-but-invalid
	//    selector — or, for a user template, a missing/unparseable source or empty
	//    stdin — fails fast as a usage error before any assembly or request.
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.outputPresent, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	// 2. Validate --status BEFORE any assembly or request (fail-fast usage error,
	//    no wasted call — pinned by a tripwire transport in the tests). A NEW
	//    validator over the tension status set (plan ADR-3); not validateStatus.
	//    Both checks are pure and pre-assembly, so the no-request-on-rejection
	//    tripwire holds regardless of their relative order.
	if err := validateTensionStatus(cfg.status); err != nil {
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

	return runTensionListWalk(cfg, exec, rt)
}

// runTensionListWalk walks GET /roles/{role_id}/tensions. The output format changes
// ONLY how the gathered set is rendered — never how much is fetched: every format
// walks to completion by default and signals incompleteness the same way (a stderr
// note, never a silently short list). --first-page opts out to a single page. This
// is the 025/038 roles/projects walked-list shape with Tension items. The id is
// escaped as one path segment (passed through unvalidated per ADR-3, but a raw
// `/`/`..` must not redirect/traverse).
func runTensionListWalk(cfg tensionsConfig, exec executor, rt renderTarget) (Outcome, error) {
	path := "/roles/" + url.PathEscape(cfg.id) + "/tensions"
	req := apiclient.Request{Method: http.MethodGet, Path: path, Query: tensionsQuery(cfg)}

	if cfg.firstPage {
		return runTensionListFirstPage(cfg, exec, rt, req)
	}

	// Structured: walk to completion preserving each tension's raw bytes, then emit
	// the aggregated {data:[…]} document (reusing the resource-neutral aggregator).
	if machineFmt, ok := rt.format.MachineFormat(); ok {
		res := paging.All[json.RawMessage](cfg.reqCtx, exec, req, tensionsWalkOptions(cfg)...)
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
			return reportIncompleteTensionsWalk(cfg.stderr, res.Stop)
		}
		return Success, nil
	}

	// Human / user template: walk to completion, render the tensions projection (the
	// new 043 `tensions` key; an empty set renders `no tensions`).
	res := paging.All[glassfrog.Tension](cfg.reqCtx, exec, req, tensionsWalkOptions(cfg)...)
	if res.Stop != nil && len(res.Records) == 0 {
		// A walk that stopped before gathering any record is a clean failure (e.g. a
		// first-page transport/auth/API error): no partial set to show.
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, res.Stop)
	}
	view := render.TensionsView{Data: res.Records}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceTensions, rt.format, view); outcome != Success {
		return outcome, rerr
	}
	if res.Stop != nil {
		return reportIncompleteTensionsWalk(cfg.stderr, res.Stop)
	}
	return Success, nil
}

// runTensionListFirstPage performs the --first-page opt-out: a single
// GET /roles/{role_id}/tensions page (no walk) in EVERY format, with one stderr
// note when the API reports more pages exist (still exit 0 — the operator chose the
// boundary). The structured path emits the same {data:[…]} envelope the default
// walk does; the human path renders the projection. --per-page (if set) sizes the
// single request; the walker is not involved.
func runTensionListFirstPage(cfg tensionsConfig, exec executor, rt renderTarget, req apiclient.Request) (Outcome, error) {
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
			fmt.Fprintln(cfg.stderr, moreTensionsNote)
		}
		return Success, nil
	}

	var page glassfrog.Page[glassfrog.Tension]
	if _, err := exec.Execute(cfg.reqCtx, req, &page); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	view := render.TensionsView{Data: page.Data}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceTensions, rt.format, view); outcome != Success {
		return outcome, rerr
	}
	if page.Meta.Pagination.HasNextPage {
		fmt.Fprintln(cfg.stderr, moreTensionsNote)
	}
	return Success, nil
}

// reportIncompleteTensionsWalk writes the mid-walk incomplete note (the partial set
// is already on stdout) and returns the classified non-zero outcome. It mirrors
// projects.go's reportIncompleteProjectsWalk with the tension wording: the Stop
// error is refined once so a non-2xx splits into permission/rate-limit (015), and
// the note text, the classified outcome, and the returned error all derive from
// that same refined value (the reportClientError invariant).
func reportIncompleteTensionsWalk(stderr io.Writer, stop error) (Outcome, error) {
	refined := refineClientError(stop)
	fmt.Fprintf(stderr, incompleteTensionsWalkNote+"\n", refined.Error())
	return classifyClientError(refined), refined
}

// tensionsQuery builds the GET /roles/{role_id}/tensions query from the single
// optional filter. --status (sent as `status`) is sent only when non-empty — it was
// already validated by runTensionList against the tension status set, so an empty
// value is the "no filter" case (`--status ""` and an omitted flag both behave as no
// filter). A nil return leaves the request unparameterised.
func tensionsQuery(cfg tensionsConfig) url.Values {
	q := url.Values{}
	if cfg.status != "" {
		q.Set("status", cfg.status)
	}
	if len(q) == 0 {
		return nil
	}
	return q
}

// tensionsWalkOptions builds the paging options for the default walk. --per-page
// (016's WithPageSize) sizes the walk when provided (by presence, not value); the
// value is passed through as-is — no client-side clamp (paging's contract).
func tensionsWalkOptions(cfg tensionsConfig) []paging.Option {
	if cfg.perPageSet {
		return []paging.Option{paging.WithPageSize(cfg.perPage)}
	}
	return nil
}

// newTensionListCommand builds the runnable `tension list <role-id>` leaf (ADR-1):
// a guard-ready cobra command with a REQUIRED positional role id (Args:
// cobra.ExactArgs(1)), a non-empty Short, and SilenceErrors/SilenceUsage so
// runTensionList owns its messages. It declares the list-only flags (--status,
// --first-page, --per-page) — these exist ONLY on `list`, so passing one to `get`
// is a cobra unknown-flag usage error (the structural list-only guard, ADR-1). It
// reads the inherited persistent --base-url/--output flags, then delegates to the
// pure runTensionList. The seam is injected so tests drive a fake one; production
// passes productionSeam{}.
func newTensionListCommand(seam tensionSeam) *cobra.Command {
	var (
		status    string
		firstPage bool
		perPage   int
	)
	cmd := &cobra.Command{
		Use:   "list <role-id>",
		Short: "List the tensions a role carries, walking pages to completion",
		Long: "list lists the tensions sensed on a role by its id, walking every page to " +
			"completion by default so the list is complete or plainly flagged incomplete. " +
			"Narrow it with a --status filter (one of: " + strings.Join(supportedTensionStatusNames(), ", ") + "). " +
			"This is the read counterpart to `tension create`. To read one tension with full " +
			"detail use `tension get <ten-id>`.",
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
			outcome, oerr := runTensionList(tensionsConfig{
				seam:           seam,
				baseURL:        baseURL,
				baseURLPresent: cmd.Flags().Changed(apiclient.FlagBaseURL),
				outputFlag:     outputFlag,
				outputPresent:  cmd.Flags().Changed(output.FlagOutput),
				id:             args[0],
				status:         status,
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
	cmd.Flags().StringVar(&status, "status", "", "Filter by tension status (one of: "+strings.Join(supportedTensionStatusNames(), ", ")+")")
	cmd.Flags().BoolVar(&firstPage, "first-page", false, "Fetch only the first page and signal if more tensions exist")
	cmd.Flags().IntVar(&perPage, "per-page", 0, "Page size for the walk (the API owns the valid range)")
	return cmd
}

// tensionGetConfig carries everything runTensionGet needs, gathered by the `get`
// command's RunE. It declares no list flags — the single read has none (ADR-1).
type tensionGetConfig struct {
	seam           tensionSeam
	baseURL        string // inherited persistent --base-url (may be empty)
	baseURLPresent bool   // whether --base-url was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	outputFlag     string // inherited persistent --output (may be empty), resolved before any request
	outputPresent  bool   // whether --output was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runTensionGet reads a single tension by id (GET /tensions/{id}). It resolves the
// output format FIRST (020 widened by 035), assembles the connection and builds the
// retrying executor, then sends one Execute (no walk). The id is escaped as a single
// path segment but passed through unvalidated (ADR-3) so an unknown/malformed id
// surfaces as the API's 404/4xx via the shared classifier — no local regex gate. A
// structured --output emits the raw {data: Tension} payload verbatim (018); the
// human path decodes Document[Tension] and renders the landed singular `tension`
// template (042) over a TensionView (mirroring runProjectGet/runTensionCreate). It
// adds no new Outcome/ExitCode and never reads the token.
func runTensionGet(cfg tensionGetConfig, id string) (Outcome, error) {
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
	// PathEscape is a no-op for a valid ten_… id and keeps a malformed/adversarial
	// id one opaque segment the API reports as a 404.
	req := apiclient.Request{Method: http.MethodGet, Path: "/tensions/" + url.PathEscape(id)}

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

	var doc glassfrog.Document[glassfrog.Tension]
	if _, err := exec.Execute(cfg.reqCtx, req, &doc); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	view := render.TensionView{Tension: doc.Data}
	return writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceTension, rt.format, view)
}

// newTensionGetCommand builds the runnable `tension get <ten-id>` leaf (ADR-1): a
// guard-ready cobra command with a REQUIRED positional tension id (Args:
// cobra.ExactArgs(1)), a non-empty Short, and SilenceErrors/SilenceUsage so
// runTensionGet owns its messages. It declares NO list flags — so passing --status,
// --first-page, or --per-page is a cobra unknown-flag usage error before any request
// (this is how the spec's "filter applies only to the list" is enforced — no
// hand-rolled cross-combo guard, ADR-1). It reads the inherited persistent
// --base-url/--output flags, then delegates to the pure runTensionGet. The seam is
// injected so tests drive a fake one; production passes productionSeam{}.
func newTensionGetCommand(seam tensionSeam) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <ten-id>",
		Short: "Read a single tension by its id, with its full detail",
		Long: "get reads one tension by its ten_ id and prints its status, body, sensing " +
			"role, sensed-by person, meeting type, parent role, and timestamps. To list the " +
			"tensions a role carries use `tension list <role-id>`.",
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
			outcome, oerr := runTensionGet(tensionGetConfig{
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

// supportedTensionStatuses is the spec's tension status set — the status enum on
// the Tension schema (spec/glassfrog-api-v5.yaml: unprocessed, processed,
// archived). It is the single source of truth for `tension list --status`
// validation. Adding a value is a one-line change tracking the spec enum. It is a
// NEW set, deliberately distinct from the action/project validateStatus set
// (current/completed/…) in status.go — reusing that set would accept invalid
// tension statuses and reject valid ones (a correctness bug, plan ADR-3). It
// deliberately does not include null/empty: an absent --status is the "no filter"
// case the validator accepts and the query builder omits.
var supportedTensionStatuses = map[string]bool{
	"unprocessed": true,
	"processed":   true,
	"archived":    true,
}

// validateTensionStatus rejects a non-empty --status value outside the tension
// status set, returning a usage error NAMING the unsupported value and listing the
// supported set — before any context assembly or request (the validateMeetingType
// fail-fast discipline, plan ADR-3). An empty value (the flag absent) is valid: no
// status filter, the parameter is omitted from the request. Pure — no network, no
// filesystem — so it runs ahead of any I/O and a tripwire transport can assert
// nothing was sent on rejection. A sibling validator of validateMeetingType /
// validateStatus, not a second copy of any set.
func validateTensionStatus(value string) error {
	if value == "" {
		return nil
	}
	if supportedTensionStatuses[value] {
		return nil
	}
	return fmt.Errorf(
		"unsupported --status value %q — supported: %s",
		value, strings.Join(supportedTensionStatusNames(), ", "),
	)
}

// supportedTensionStatusNames lists the supported tension statuses in stable
// (sorted) order for the usage message, so the same input always yields the same
// deterministic text (the supportedMeetingTypeNames shape).
func supportedTensionStatusNames() []string {
	names := make([]string, 0, len(supportedTensionStatuses))
	for name := range supportedTensionStatuses {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
