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

// incompleteProposalsWalkNote is the stderr line the default walk writes when it stops
// on an error after gathering at least one page: the partial set is already on stdout,
// so this names the cause and marks the list incomplete — a partial list is never
// silently presented as complete (CONSTITUTION VI; interface-cli). The %s is the cause;
// the command exits non-zero (classifyClientError(Stop)). It is the proposal-worded
// sibling of tension_reads.go's incompleteTensionsWalkNote.
const incompleteProposalsWalkNote = "note: result is incomplete — %s; the proposals shown are a partial set"

// moreProposalsNote is the stderr line the --first-page opt-out writes when the first
// page reports more pages exist: the operator chose the boundary, so this is not an
// error (exit 0) — it just keeps a partial list from being read as complete
// (interface-cli; CONSTITUTION VI).
const moreProposalsNote = "note: more proposals exist than shown; re-run without --first-page to fetch all"

// supportedProposalStatuses is the spec's proposal status set — the closed status enum
// on the listProposals filter (spec/glassfrog-api-v5.yaml: draft,
// proposed_outside_meeting, escalated, accepted, draft_with_conflicts). It is the
// single source of truth for `proposal list --status` validation; adding a value is a
// one-line change tracking the spec enum. It is a NEW set, deliberately distinct from
// the action/project validateStatus set (status.go) AND the tension
// validateTensionStatus set (tension_reads.go) — reusing either would accept invalid
// proposal statuses and reject valid ones (a correctness bug, plan ADR-3). It lives
// WITH the proposal command code (the validateTensionStatus/validateMeetingType
// precedent), NOT in the shared status.go. It deliberately omits null/empty: an absent
// --status is the "no filter" case the validator accepts and the query builder omits.
// Note draft_with_conflicts — the value the FEATURE-MODEL prose omits.
var supportedProposalStatuses = map[string]bool{
	"draft":                    true,
	"proposed_outside_meeting": true,
	"escalated":                true,
	"accepted":                 true,
	"draft_with_conflicts":     true,
}

// validateProposalStatus rejects a non-empty --status value outside the proposal status
// set, returning a usage error NAMING the unsupported value and listing the supported
// set — before any context assembly or request (the validateTensionStatus fail-fast
// discipline, plan ADR-3). An empty value (the flag absent) is valid: no status filter,
// the parameter is omitted from the request. Pure — no network, no filesystem — so it
// runs ahead of any I/O and a tripwire transport can assert nothing was sent on
// rejection. A sibling validator of validateTensionStatus / validateMeetingType, not a
// second copy of any set.
func validateProposalStatus(value string) error {
	if value == "" {
		return nil
	}
	if supportedProposalStatuses[value] {
		return nil
	}
	return fmt.Errorf(
		"unsupported --status value %q — supported: %s",
		value, strings.Join(supportedProposalStatusNames(), ", "),
	)
}

// supportedProposalStatusNames lists the supported proposal statuses in stable
// (sorted) order for the usage message, so the same input always yields the same
// deterministic text (the supportedTensionStatusNames shape).
func supportedProposalStatusNames() []string {
	names := make([]string, 0, len(supportedProposalStatuses))
	for name := range supportedProposalStatuses {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// proposalsConfig carries everything runProposalList needs, gathered by the `list`
// command's RunE. Keeping the read a function of injected values makes the whole flow —
// resolve, validate, assemble, build, walk, render/classify — testable over a fake
// transport with no real network or ~/.glassfrogrc. It reuses 055's proposalSeam (the
// reads touch only the embedded tensionSeam half — assemble/newClient/sleep/
// resolveSelection/readTemplateSource; readChangesSource is the write-only member).
type proposalsConfig struct {
	seam           proposalSeam
	baseURL        string // inherited persistent --base-url (may be empty)
	baseURLPresent bool   // whether --base-url was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	outputFlag     string // inherited persistent --output (may be empty), resolved before any request
	outputPresent  bool   // whether --output was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)

	// The five optional server-side filters. --status is validated locally against the
	// proposal status set (plan ADR-3); the other four are free values passed through.
	// Each is sent as its query parameter only when its value is non-empty; an
	// explicitly-empty flag (e.g. --status "", where Changed() is true but the value is
	// "") is treated as no filter and omitted — the omit decision keys on the value, not
	// on Changed().
	status        string
	roleID        string
	proposerID    string
	proposedAfter string
	acceptedAfter string

	firstPage  bool
	perPage    int
	perPageSet bool // whether --per-page was provided (Changed); presence, not value

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runProposalList is the pure orchestration the `proposal list` leaf delegates to:
// resolve the output format (020, widened by 035) FIRST, then validate the one
// closed-enum input (--status) fail-fast via validateProposalStatus (an unsupported
// value is a usage error with NO request issued) — both pure checks run before any
// assembly, in the same output-first order as the sibling reads so error precedence is
// consistent. Then assemble the connection, build the retrying executor, and walk the
// global proposals endpoint (GET /proposals) to completion. Unlike `tension list
// <role-id>`, the list is GLOBAL — no path id; the circle is the optional --role-id
// filter (plan ADR-1). The four non-status filters and the id are free values passed
// through (plan ADR-3). It adds no new Outcome/ExitCode and never reads the token.
func runProposalList(cfg proposalsConfig) (Outcome, error) {
	// 1. Resolve the render target FIRST (020 widened by 035): a present-but-invalid
	//    selector — or, for a user template, a missing/unparseable source or empty
	//    stdin — fails fast as a usage error before any assembly or request.
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.outputPresent, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	// 2. Validate --status BEFORE any assembly or request (fail-fast usage error, no
	//    wasted call — pinned by a tripwire transport in the tests). A NEW validator
	//    over the proposal status set (plan ADR-3); not validateStatus/
	//    validateTensionStatus. Both checks are pure and pre-assembly, so the
	//    no-request-on-rejection tripwire holds regardless of their relative order.
	if err := validateProposalStatus(cfg.status); err != nil {
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

	return runProposalListWalk(cfg, exec, rt)
}

// runProposalListWalk walks the global proposals endpoint (GET /proposals). The output
// format changes ONLY how the gathered set is rendered — never how much is fetched:
// every format walks to completion by default and signals incompleteness the same way
// (a stderr note, never a silently short list). --first-page opts out to a single page.
// This is the 025/038/043 walked-list shape with Proposal items.
func runProposalListWalk(cfg proposalsConfig, exec executor, rt renderTarget) (Outcome, error) {
	req := apiclient.Request{Method: http.MethodGet, Path: "/proposals", Query: proposalsQuery(cfg)}

	if cfg.firstPage {
		return runProposalListFirstPage(cfg, exec, rt, req)
	}

	// Structured: walk to completion preserving each proposal's raw bytes, then emit the
	// aggregated {data:[…]} document (reusing the resource-neutral aggregator).
	if machineFmt, ok := rt.format.MachineFormat(); ok {
		res := paging.All[json.RawMessage](cfg.reqCtx, exec, req, proposalsWalkOptions(cfg)...)
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
			return reportIncompleteProposalsWalk(cfg.stderr, res.Stop)
		}
		return Success, nil
	}

	// Human / user template: walk to completion, render the proposals projection (the
	// new 056 `proposals` key; an empty set renders `no proposals`).
	res := paging.All[glassfrog.Proposal](cfg.reqCtx, exec, req, proposalsWalkOptions(cfg)...)
	if res.Stop != nil && len(res.Records) == 0 {
		// A walk that stopped before gathering any record is a clean failure (e.g. a
		// first-page transport/auth/API error): no partial set to show.
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, res.Stop)
	}
	view := render.ProposalsView{Data: res.Records}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceProposals, rt.format, view); outcome != Success {
		return outcome, rerr
	}
	if res.Stop != nil {
		return reportIncompleteProposalsWalk(cfg.stderr, res.Stop)
	}
	return Success, nil
}

// runProposalListFirstPage performs the --first-page opt-out: a single GET /proposals
// page (no walk) in EVERY format, with one stderr note when the API reports more pages
// exist (still exit 0 — the operator chose the boundary). The structured path emits the
// same {data:[…]} envelope the default walk does; the human path renders the
// projection. --per-page (if set) sizes the single request; the walker is not involved.
func runProposalListFirstPage(cfg proposalsConfig, exec executor, rt renderTarget, req apiclient.Request) (Outcome, error) {
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
			fmt.Fprintln(cfg.stderr, moreProposalsNote)
		}
		return Success, nil
	}

	var page glassfrog.Page[glassfrog.Proposal]
	if _, err := exec.Execute(cfg.reqCtx, req, &page); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	view := render.ProposalsView{Data: page.Data}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceProposals, rt.format, view); outcome != Success {
		return outcome, rerr
	}
	if page.Meta.Pagination.HasNextPage {
		fmt.Fprintln(cfg.stderr, moreProposalsNote)
	}
	return Success, nil
}

// reportIncompleteProposalsWalk writes the mid-walk incomplete note (the partial set is
// already on stdout) and returns the classified non-zero outcome. It mirrors
// tension_reads.go's reportIncompleteTensionsWalk with the proposal wording: the Stop
// error is refined once so a non-2xx splits into permission/rate-limit (015), and the
// note text, the classified outcome, and the returned error all derive from that same
// refined value (the reportClientError invariant).
func reportIncompleteProposalsWalk(stderr io.Writer, stop error) (Outcome, error) {
	refined := refineClientError(stop)
	fmt.Fprintf(stderr, incompleteProposalsWalkNote+"\n", refined.Error())
	return classifyClientError(refined), refined
}

// proposalsQuery builds the GET /proposals query from the five optional filters. Each
// is sent only when its value is non-empty; an explicitly-empty flag (e.g. --status "")
// is treated as no filter and omitted — the omit decision keys on the value, not on
// Changed(). --status was already validated by runProposalList against the proposal
// status set; the four others are free values passed through (plan ADR-3) for the server
// to validate. A nil return leaves the request unparameterised.
func proposalsQuery(cfg proposalsConfig) url.Values {
	q := url.Values{}
	if cfg.status != "" {
		q.Set("status", cfg.status)
	}
	if cfg.roleID != "" {
		q.Set("role_id", cfg.roleID)
	}
	if cfg.proposerID != "" {
		q.Set("proposer_id", cfg.proposerID)
	}
	if cfg.proposedAfter != "" {
		q.Set("proposed_after", cfg.proposedAfter)
	}
	if cfg.acceptedAfter != "" {
		q.Set("accepted_after", cfg.acceptedAfter)
	}
	if len(q) == 0 {
		return nil
	}
	return q
}

// proposalsWalkOptions builds the paging options for the default walk. --per-page
// (016's WithPageSize) sizes the walk when provided (by presence, not value); the value
// is passed through as-is — no client-side clamp (paging's contract).
func proposalsWalkOptions(cfg proposalsConfig) []paging.Option {
	if cfg.perPageSet {
		return []paging.Option{paging.WithPageSize(cfg.perPage)}
	}
	return nil
}

// newProposalListCommand builds the runnable `proposal list` leaf (ADR-1): a guard-ready
// cobra command with NO positional (Args: cobra.NoArgs — the list is GLOBAL; any
// positional is a usage error before RunE, no request), a non-empty Short, and
// SilenceErrors/SilenceUsage so runProposalList owns its messages. It declares the
// list-only flags (--status, --role-id, --proposer-id, --proposed-after,
// --accepted-after, --first-page, --per-page) — these exist ONLY on `list`, so passing
// one to `get` is a cobra unknown-flag usage error (the structural list-only guard,
// ADR-1). It reads the inherited persistent --base-url/--output flags, then delegates
// to the pure runProposalList. The seam is injected so tests drive a fake one;
// production passes productionSeam{}.
func newProposalListCommand(seam proposalSeam) *cobra.Command {
	var (
		status        string
		roleID        string
		proposerID    string
		proposedAfter string
		acceptedAfter string
		firstPage     bool
		perPage       int
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the proposals visible to the caller, walking pages to completion",
		Long: "list lists the proposals visible to the caller (GET /proposals), walking every " +
			"page to completion by default so the list is complete or plainly flagged incomplete. " +
			"The list is global — it takes no positional; narrow it with the optional --role-id " +
			"(the circle), --proposer-id, --proposed-after, --accepted-after, and --status filters " +
			"(--status is validated locally against one of: " + strings.Join(supportedProposalStatusNames(), ", ") + "). " +
			"This is the read counterpart to `proposal create`. To read one proposal with full " +
			"detail use `proposal get <prp-id>`.",
		Args:          cobra.NoArgs,
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
			outcome, oerr := runProposalList(proposalsConfig{
				seam:           seam,
				baseURL:        baseURL,
				baseURLPresent: cmd.Flags().Changed(apiclient.FlagBaseURL),
				outputFlag:     outputFlag,
				outputPresent:  cmd.Flags().Changed(output.FlagOutput),
				status:         status,
				roleID:         roleID,
				proposerID:     proposerID,
				proposedAfter:  proposedAfter,
				acceptedAfter:  acceptedAfter,
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
	cmd.Flags().StringVar(&status, "status", "", "Filter by proposal status (one of: "+strings.Join(supportedProposalStatusNames(), ", ")+")")
	cmd.Flags().StringVar(&roleID, "role-id", "", "Filter to proposals in the circle supported by this role (sent as role_id)")
	cmd.Flags().StringVar(&proposerID, "proposer-id", "", "Filter to proposals created by this person (sent as proposer_id)")
	cmd.Flags().StringVar(&proposedAfter, "proposed-after", "", "Only proposals proposed on or after this timestamp (sent as proposed_after)")
	cmd.Flags().StringVar(&acceptedAfter, "accepted-after", "", "Only proposals accepted on or after this timestamp (sent as accepted_after)")
	cmd.Flags().BoolVar(&firstPage, "first-page", false, "Fetch only the first page and signal if more proposals exist")
	cmd.Flags().IntVar(&perPage, "per-page", 0, "Page size for the walk (the API owns the valid range)")
	return cmd
}

// proposalGetConfig carries everything runProposalGet needs, gathered by the `get`
// command's RunE. It declares no list flags — the single read has none (ADR-1).
type proposalGetConfig struct {
	seam           proposalSeam
	baseURL        string // inherited persistent --base-url (may be empty)
	baseURLPresent bool   // whether --base-url was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	outputFlag     string // inherited persistent --output (may be empty), resolved before any request
	outputPresent  bool   // whether --output was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runProposalGet reads a single proposal by id (GET /proposals/{id}). It resolves the
// output format FIRST (020 widened by 035), assembles the connection and builds the
// retrying executor, then sends one Execute (no walk). The id is escaped as a single
// path segment but passed through unvalidated (ADR-3) so an unknown/invisible id
// surfaces as the API's 404/4xx via the shared classifier — no local regex gate. A
// structured --output emits the raw {data: Proposal} payload verbatim (018); the human
// path decodes Document[Proposal] and renders the singular `proposal` template (055,
// grown by 056) over a ProposalView — surfacing the changes by type, the aggregate
// response summary, and the available transitions (the latter printed, never invoked —
// spec non-behavior). It adds no new Outcome/ExitCode and never reads the token.
func runProposalGet(cfg proposalGetConfig, id string) (Outcome, error) {
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

	// Escape the id as a single path segment: passed through unvalidated (ADR-3), but a
	// raw `/` or `..` must not redirect the request or traverse the path. PathEscape is
	// a no-op for a valid prp_… id and keeps a malformed/adversarial id one opaque
	// segment the API reports as a 404.
	req := apiclient.Request{Method: http.MethodGet, Path: "/proposals/" + url.PathEscape(id)}

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

	var doc glassfrog.Document[glassfrog.Proposal]
	if _, err := exec.Execute(cfg.reqCtx, req, &doc); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	view := render.ProposalView{Proposal: doc.Data}
	return writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceProposal, rt.format, view)
}

// newProposalGetCommand builds the runnable `proposal get <prp-id>` leaf (ADR-1): a
// guard-ready cobra command with a REQUIRED positional proposal id (Args:
// cobra.ExactArgs(1)), a non-empty Short, and SilenceErrors/SilenceUsage so
// runProposalGet owns its messages. It declares NO list flags — so passing --status,
// --role-id, --proposer-id, --proposed-after, --accepted-after, --first-page, or
// --per-page is a cobra unknown-flag usage error before any request (this is how the
// spec's "filters apply only to the list" is enforced — no hand-rolled cross-combo
// guard, ADR-1). It reads the inherited persistent --base-url/--output flags, then
// delegates to the pure runProposalGet. The seam is injected so tests drive a fake one;
// production passes productionSeam{}.
func newProposalGetCommand(seam proposalSeam) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <prp-id>",
		Short: "Read a single proposal by its id, with its full detail",
		Long: "get reads one proposal by its prp_ id and prints its status, changes (by type), " +
			"aggregate response summary, and available transitions. The available transitions are " +
			"printed but never invoked — advancing a proposal is a separate write command. To list " +
			"the proposals visible to you use `proposal list`.",
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
			outcome, oerr := runProposalGet(proposalGetConfig{
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
