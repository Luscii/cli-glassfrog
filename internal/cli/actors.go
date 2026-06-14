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
	"time"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/paging"
	"github.com/Luscii/cli-glassfrog/internal/render"
	"github.com/spf13/cobra"
)

// incompleteActorsWalkNote is the stderr line the default walk writes when it
// stops on an error after gathering at least one page: the partial set is already
// on stdout, so this names the cause and marks the list incomplete — a partial
// list is never silently presented as complete (CONSTITUTION VI; interface-cli).
// The %s is the cause; the command exits non-zero (classifyClientError(Stop)). It
// is the actors-worded sibling of search.go's incompleteSearchNote.
const incompleteActorsWalkNote = "note: result is incomplete — %s; the actors shown are a partial set"

// moreActorsNote is the stderr line the --first-page opt-out writes when the first
// page reports more pages exist: the operator chose the boundary, so this is not
// an error (exit 0) — it just keeps a partial list from being read as complete
// (interface-cli; CONSTITUTION VI).
const moreActorsNote = "note: more actors exist than shown; re-run without --first-page to fetch all"

// supportedActorKinds is the closed 2-value enum the --kind flag accepts — the
// actor types GET /actors filters on (the `kind` parameter). A value outside it is
// rejected fail-fast (the API would otherwise silently ignore it and return results
// indistinguishable from "no matches" — the closed-enum hazard 025 ADR-4 guards
// against). This guards INPUT only; the API owns the response vocabulary.
var supportedActorKinds = map[string]bool{
	"human": true,
	"agent": true,
}

// actorsSeam supplies everything the `actors` read needs from the outside, so
// runActorsList is pure over injected values and every branch runs offline. Same
// shape as searchSeam/projectsSeam (assemble + newClient + sleep + resolveSelection
// + readTemplateSource — 035 widened resolveFormat into the discriminated
// selection), so productionSeam satisfies it unchanged and the existing test fakes
// drive it. It never reads ctx.Cred.Token — the token rides 007's AuthTransport in
// the client.
type actorsSeam interface {
	assemble(baseURL string, baseURLPresent bool) apiclient.ConnectionContext
	newClient(ctx apiclient.ConnectionContext) (*apiclient.Client, error)
	sleep() func(time.Duration)
	resolveSelection(flagValue string, flagPresent bool) (output.Selection, error)
	readTemplateSource(ref output.TemplateRef) (string, error)
}

// actorsConfig carries everything runActors needs, gathered by the command's
// RunE. Keeping the read a function of injected values makes the whole flow —
// resolve, validate, assemble, build, walk/send, render/classify — testable over a
// fake transport with no real network or ~/.glassfrogrc. `actors` takes an OPTIONAL
// positional id (cobra.MaximumNArgs(1), 049 ADR-1): 0 args → the directory list
// (its subject is the whole organization, narrowed only by the three optional,
// combinable filters), 1 arg → the single-actor read.
type actorsConfig struct {
	seam           actorsSeam
	baseURL        string   // inherited persistent --base-url (may be empty)
	baseURLPresent bool     // whether --base-url was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	outputFlag     string   // inherited persistent --output (may be empty), resolved before any request
	outputPresent  bool     // whether --output was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	args           []string // 0 → directory list, 1 → single read by id (049 ADR-1)

	// list filters (the single-read branch forbids them; validateActorsFlags guards)
	kind     string // --kind filter, validated against {human, agent} before any request
	kindSet  bool   // whether --kind was provided (Changed); kind is sent only when set AND non-empty
	roleID   string // --role-id filter, sent verbatim as role_id (free identifier, passed through)
	roleSet  bool   // whether --role-id was provided (Changed); role_id is sent only when set AND non-empty
	query    string // --query/-q free-text search, sent verbatim as q
	querySet bool   // whether --query was provided (Changed); q is sent only when set AND non-empty

	firstPage  bool
	perPage    int
	perPageSet bool // whether --per-page was provided (Changed); presence, not value

	// single-read flag (the list branch forbids it; validateActorsFlags guards)
	include []string // --include, validated against {roles, assignments} before any request (049 ADR-3)

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// actorsWalkConfig is the minimal context the actors list-walk needs once the
// request (path + query) is built by the caller. It single-sources the walk over
// the `actors` directory (runActors, path /actors) and the subrole roll-up
// (runSubroleActors, path /roles/{role_id}/subroles/actors): the page loop, the
// structured/human render branch, the completeness note, and the errors.As
// classification chain are identical (plan ADR-3 — reuse, do not copy). Only the
// request differs, so it is the parameter, not these fields.
type actorsWalkConfig struct {
	firstPage  bool
	perPage    int
	perPageSet bool // whether --per-page was provided (Changed); presence, not value

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// walk projects the directory config down to the shared walk context, so both the
// `actors` directory and the subrole roll-up drive the same list-walk helpers.
func (cfg actorsConfig) walk() actorsWalkConfig {
	return actorsWalkConfig{
		firstPage:  cfg.firstPage,
		perPage:    cfg.perPage,
		perPageSet: cfg.perPageSet,
		reqCtx:     cfg.reqCtx,
		stdout:     cfg.stdout,
		stderr:     cfg.stderr,
	}
}

// runActors is the pure orchestration the `actors` leaf delegates to: resolve the
// output format (020) FIRST, then run the mode-separation guards and validate the
// branch's closed-enum input fail-fast — all pure, pre-assembly checks, output-first
// so error precedence is consistent with the sibling reads (an invalid --output is
// reported even when a misplaced flag or a bad --include is also present; interface
// § Interactions). Then assemble the connection, build the retrying executor once,
// and dispatch on whether a positional id was given (049 ADR-1): 0 args → the
// org-wide directory walk (048, preserved), 1 arg → the single-actor read. It adds
// no new Outcome/ExitCode and never reads the token.
func runActors(cfg actorsConfig) (Outcome, error) {
	// 1. Resolve the render target FIRST (020 widened by 035): a present-but-invalid
	//    selector — or, for a user template, a missing/unparseable source or empty
	//    stdin — fails fast as a usage error before any assembly or request. Resolving
	//    --output ahead of the mode-separation / --include / --kind checks keeps error
	//    precedence consistent with the sibling reads (interface § Interactions).
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.outputPresent, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	hasID := len(cfg.args) == 1

	// 2. Mode-separation guards BEFORE any assembly or request (049 ADR-1): the list
	//    filters (--kind/--role-id/--query/--first-page/--per-page) apply only to the
	//    directory, so passing any of them with an id is a usage error; --include
	//    applies only to a single actor, so passing it without an id is a usage error.
	//    cobra's MaximumNArgs(1) already rejects a second positional.
	if err := validateActorsFlags(hasID, actorsFlagState{
		kindSet:    cfg.kindSet,
		roleIDSet:  cfg.roleSet,
		querySet:   cfg.querySet,
		firstPage:  cfg.firstPage,
		perPageSet: cfg.perPageSet,
		includeSet: len(cfg.include) > 0,
	}); err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}

	// 3. Validate the branch's one closed-enum input BEFORE any request (a bad value
	//    would be silently ignored at the API, returning a result indistinguishable
	//    from a real empty answer — plan ADR-3). On the single branch that is
	//    --include over {roles, assignments}; on the list branch it is --kind over
	//    {human, agent}. --role-id/--query and the actor id itself are free values
	//    passed through (the API reports a bad one cleanly). All checks are pure and
	//    pre-assembly, so the no-request-on-rejection tripwire holds.
	if hasID {
		if err := validateActorInclude(cfg.include); err != nil {
			fmt.Fprintln(cfg.stderr, err.Error())
			return UsageError, err
		}
	} else if err := validateKind(cfg.kind); err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}

	// 4. Resolve the connection and build the client + retrying executor once. A
	//    base-URL error surfaces here (no doomed send); classify + report it.
	ctx := cfg.seam.assemble(cfg.baseURL, cfg.baseURLPresent)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	exec := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, cfg.seam.sleep(), cfg.stderr)

	if hasID {
		return runActorRead(cfg, exec, rt, cfg.args[0])
	}
	req := apiclient.Request{Method: http.MethodGet, Path: "/actors", Query: actorsQuery(cfg)}
	return runActorsListWalk(cfg.walk(), exec, rt, req)
}

// runActorRead reads a single actor by id (GET /actors/{id}). It sends one Execute
// (no walk, no Page[T] — a single resource) with ?include= built from the
// already-validated --include values (comma-joined), passing the id through
// unvalidated (049 ADR-3) so an unknown/malformed id surfaces as the API's clean
// 404/4xx via the shared classifier. The id is PathEscaped to one opaque segment so
// a raw `/` cannot redirect the request to a different endpoint (the 025 runRoleGet
// discipline). A structured --output emits the raw {data: ActorDetail} document
// verbatim via output.RenderSuccess (the single-resource raw-bytes path, 018 ADR-2 —
// NOT aggregateRawData, and never a decode-and-re-encode); the human path decodes
// the ActorDetail and renders the `actor` template over an ActorDetailView carrying
// the requested-include set, so each footprint section is omitted when unrequested
// and shows an explicit-absence marker when requested-but-empty (049 ADR-4).
func runActorRead(cfg actorsConfig, exec executor, rt renderTarget, id string) (Outcome, error) {
	var q url.Values
	if len(cfg.include) > 0 {
		q = url.Values{"include": {strings.Join(cfg.include, ",")}}
	}
	req := apiclient.Request{Method: http.MethodGet, Path: "/actors/" + url.PathEscape(id), Query: q}

	if machineFmt, ok := rt.format.MachineFormat(); ok {
		var raw json.RawMessage
		if _, err := exec.Execute(cfg.reqCtx, req, &raw); err != nil {
			return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
		}
		doc, rerr := output.RenderSuccess(machineFmt, raw)
		if rerr != nil {
			fmt.Fprintln(cfg.stderr, rerr.Error())
			return RuntimeError, rerr
		}
		_, _ = cfg.stdout.Write(doc)
		return Success, nil
	}

	var doc glassfrog.ActorDocument
	if _, err := exec.Execute(cfg.reqCtx, req, &doc); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	view := render.ActorDetailView{Detail: doc.Data, Requested: includeSet(cfg.include)}
	return writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceActor, rt.format, view)
}

// runActorsListWalk walks the supplied actors-list request to completion. The
// output format changes ONLY how the gathered set is rendered — never how much is
// fetched: every format walks to completion by default and signals incompleteness
// the same way (a stderr note, never a silently short list). --first-page opts out
// to a single page. This is the 038/041 roles/projects/search walked-list shape with
// Actor items, shared by the `actors` directory (path /actors) and the subrole
// roll-up (path /roles/{role_id}/subroles/actors) — only the request differs (plan
// ADR-3). The request's query filter rides EVERY page of the walk — paging.All
// clones and preserves the base request's query across pages (plan Risk).
func runActorsListWalk(cfg actorsWalkConfig, exec executor, rt renderTarget, req apiclient.Request) (Outcome, error) {
	if cfg.firstPage {
		return runActorsFirstPage(cfg, exec, rt, req)
	}

	// Structured: walk to completion preserving each actor's raw bytes, then emit the
	// aggregated {data:[…]} document (reusing the resource-neutral aggregator). A user
	// template resolves to a non-structured format (rt.tmpl != nil → Full), so this
	// branch is never taken under -o stdin/file (035).
	if machineFmt, ok := rt.format.MachineFormat(); ok {
		res := paging.All[json.RawMessage](cfg.reqCtx, exec, req, actorsWalkOptions(cfg)...)
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
			return reportIncompleteActorsWalk(cfg.stderr, res.Stop)
		}
		return Success, nil
	}

	// Human / user template: walk to completion, render the actors projection (the new
	// 048 `actors` key, or a selected user template; an empty set renders `no actors`).
	res := paging.All[glassfrog.Actor](cfg.reqCtx, exec, req, actorsWalkOptions(cfg)...)
	if res.Stop != nil && len(res.Records) == 0 {
		// A walk that stopped before gathering any record is a clean failure (e.g. a
		// first-page transport/auth/API error): no partial set to show.
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, res.Stop)
	}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceActors, rt.format, render.ActorsView{Data: res.Records}); outcome != Success {
		return outcome, rerr
	}
	if res.Stop != nil {
		return reportIncompleteActorsWalk(cfg.stderr, res.Stop)
	}
	return Success, nil
}

// runActorsFirstPage performs the --first-page opt-out: a single GET /actors page
// (no walk) in EVERY format, with one stderr note when the API reports more pages
// exist (still exit 0 — the operator chose the boundary). The structured path emits
// the same {data:[…]} envelope the default walk does; the human path renders the
// projection. --per-page (if set) sizes the single request; the walker is not
// involved.
func runActorsFirstPage(cfg actorsWalkConfig, exec executor, rt renderTarget, req apiclient.Request) (Outcome, error) {
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
			fmt.Fprintln(cfg.stderr, moreActorsNote)
		}
		return Success, nil
	}

	var page glassfrog.Page[glassfrog.Actor]
	if _, err := exec.Execute(cfg.reqCtx, req, &page); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceActors, rt.format, render.ActorsView{Data: page.Data}); outcome != Success {
		return outcome, rerr
	}
	if page.Meta.Pagination.HasNextPage {
		fmt.Fprintln(cfg.stderr, moreActorsNote)
	}
	return Success, nil
}

// reportIncompleteActorsWalk writes the mid-walk incomplete note (the partial set
// is already on stdout) and returns the classified non-zero outcome. It mirrors
// search.go's reportIncompleteSearchWalk with the actors wording: the Stop error is
// refined once so a non-2xx splits into permission/rate-limit (015), and the note
// text, the classified outcome, and the returned error all derive from that same
// refined value (the reportClientError invariant).
func reportIncompleteActorsWalk(stderr io.Writer, stop error) (Outcome, error) {
	refined := refineClientError(stop)
	fmt.Fprintf(stderr, incompleteActorsWalkNote+"\n", refined.Error())
	return classifyClientError(refined), refined
}

// actorsQuery builds the GET /actors query from the three combinable filters. Each
// is sent ONLY when the flag was provided (Changed) AND non-empty (the 033/038
// optional-flag discipline; plan ADR-3): an omitted flag and `--flag ""` both behave
// as no filter. --kind (sent as `kind`) was already validated by runActorsList
// against the closed set; --role-id (sent as `role_id`) and --query (sent as `q`)
// are free values forwarded verbatim. Each present filter is its own query
// parameter; the API applies them together (narrowing the directory further). A nil
// return leaves the request unparameterised (every actor requested).
func actorsQuery(cfg actorsConfig) url.Values {
	q := url.Values{}
	if cfg.kindSet && cfg.kind != "" {
		q.Set("kind", cfg.kind)
	}
	if cfg.roleSet && cfg.roleID != "" {
		q.Set("role_id", cfg.roleID)
	}
	if cfg.querySet && cfg.query != "" {
		q.Set("q", cfg.query)
	}
	if len(q) == 0 {
		return nil
	}
	return q
}

// actorsWalkOptions builds the paging options for the default walk. --per-page
// (016's WithPageSize) sizes the walk when provided (by presence, not value); the
// value is passed through as-is — no client-side clamp (paging's contract). Unlike
// /search, /actors has no endpoint-specific maximum to override, so an absent
// --per-page uses paging.All's generic default.
func actorsWalkOptions(cfg actorsWalkConfig) []paging.Option {
	if cfg.perPageSet {
		return []paging.Option{paging.WithPageSize(cfg.perPage)}
	}
	return nil
}

// validateKind rejects a non-empty --kind value outside the closed 2-value set
// {human, agent}, before any request (the reject-unknown fail-fast shape, pinned by
// a transport tripwire — plan ADR-3). It delegates to the flag-agnostic
// validateClosedFlagSet so the message names --kind (not --include) while sharing
// the landed sorted/quoted/named-set formatting — keeping validateIncludeSet's own
// --include message intact for its current callers. An empty value (the flag
// absent, or `--kind ""`) is valid: no kind constraint.
func validateKind(kind string) error {
	if kind == "" {
		return nil
	}
	return validateClosedFlagSet("--kind", []string{kind}, supportedActorKinds)
}

// supportedActorIncludes is the closed 2-value set the single read's --include
// accepts — the related resources getActor embeds (`roles`, `assignments`). It is a
// PER-READ set, distinct from the role --include set (026's two-validator
// precedent): a shared set would accept role-only values (subroles/policies/notes/…)
// the actor endpoint silently drops, returning the actor with no embed —
// indistinguishable from "this actor has none" (the silent-wrong-results hazard 049
// ADR-3 guards against). This guards INPUT only; the API owns the response.
var supportedActorIncludes = map[string]bool{
	"roles":       true,
	"assignments": true,
}

// validateActorInclude rejects any unsupported --include value outside the closed
// {roles, assignments} set, before any request (the reject-unknown fail-fast shape,
// pinned by a transport tripwire — 049 ADR-3). It delegates to the shared
// validateIncludeSet so the message names --include and lists the supported set in
// stable (sorted) order — "assignments, roles" — reusing the landed include.go
// formatting without disturbing the role read's own validateRolesInclude.
func validateActorInclude(targets []string) error {
	return validateIncludeSet(targets, supportedActorIncludes)
}

// actorsFlagState carries which list/single flags were supplied, so
// validateActorsFlags can reject a cross-branch misuse without re-reading cobra.
type actorsFlagState struct {
	kindSet    bool
	roleIDSet  bool
	querySet   bool
	firstPage  bool
	perPageSet bool
	includeSet bool
}

// validateActorsFlags rejects the cross-branch flag misuses fail-fast, before any
// request (049 ADR-1's mode-separation, pinned by a transport tripwire): the list
// filters (--kind/--role-id/--query) and the walk controls (--first-page/--per-page)
// apply only to the directory list, so passing any of them with an actor id is a
// usage error; --include applies only to a single actor read, so passing it without
// an id is a usage error. cobra's MaximumNArgs(1) already rejects more than one
// positional. The message names the misuse and the fix.
func validateActorsFlags(hasID bool, fs actorsFlagState) error {
	if hasID {
		var offending []string
		if fs.kindSet {
			offending = append(offending, "--kind")
		}
		if fs.roleIDSet {
			offending = append(offending, "--role-id")
		}
		if fs.querySet {
			offending = append(offending, "--query")
		}
		if fs.firstPage {
			offending = append(offending, "--first-page")
		}
		if fs.perPageSet {
			offending = append(offending, "--per-page")
		}
		if len(offending) > 0 {
			return fmt.Errorf(
				"%s %s the actor directory list, not a single actor read — remove %s or omit the actor id",
				joinFlags(offending), pluralVerb(len(offending)), pluralThem(len(offending)),
			)
		}
		return nil
	}
	if fs.includeSet {
		return fmt.Errorf("--include applies to a single actor read; pass an actor id (e.g. `glassfrog actors per_...`)")
	}
	return nil
}

// newActorsCommand builds the runnable `actors` leaf (049 ADR-1): a guard-ready
// cobra command with an OPTIONAL positional id (Args: cobra.MaximumNArgs(1) — a
// second positional is a fail-fast usage error via cobra's own arg validator, no
// hand-rolled guard), a non-empty Short, and SilenceErrors/SilenceUsage so runActors
// owns its messages. With no positional it lists the whole-organization directory
// (048), narrowed by the optional, combinable filters (--kind, --role-id, --query/-q,
// --first-page/--per-page); with one positional it reads that single actor and, on
// --include, embeds their governance footprint. It reads the inherited persistent
// --base-url/--output flags, then delegates to the pure runActors. The seam is
// injected so tests drive a fake one; production passes productionSeam{}. No
// `people`/`agents` command is created — --kind selects either, and an agt_ id reads
// the one ungated /actors endpoint (never the ai_integration-gated /agents alias).
func newActorsCommand(seam actorsSeam) *cobra.Command {
	var (
		kind      string
		roleID    string
		query     string
		firstPage bool
		perPage   int
		include   []string
	)
	cmd := &cobra.Command{
		Use:   "actors [id]",
		Short: "List the actors in the organization, or read one actor by id",
		Long: "actors lists the people and agents in the organization (walking every page " +
			"to completion by default), or reads a single actor by id when a positional " +
			"per_/agt_ id is given. With no id its subject is the whole organization, " +
			"narrowed by the optional, combinable filters: --kind (human or agent), " +
			"--role-id (the actors filling a role), and a free-text --query/-q; stop at one " +
			"page with --first-page. With an id it reads that one actor and can embed their " +
			"governance footprint with --include (roles, assignments). The directory's " +
			"per_/agt_ id and kind badge are the bridge into the single read.",
		Args:          cobra.MaximumNArgs(1),
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
			outcome, oerr := runActors(actorsConfig{
				seam:           seam,
				baseURL:        baseURL,
				baseURLPresent: cmd.Flags().Changed(apiclient.FlagBaseURL),
				outputFlag:     outputFlag,
				outputPresent:  cmd.Flags().Changed(output.FlagOutput),
				args:           args,
				kind:           kind,
				// Presence, not value: each filter is sent only when its flag is Changed
				// AND non-empty, so `--flag ""` behaves as no filter (ADR-3); the
				// mode-separation guard also keys on presence (a list flag with an id is
				// rejected whatever its value).
				kindSet:   cmd.Flags().Changed("kind"),
				roleID:    roleID,
				roleSet:   cmd.Flags().Changed("role-id"),
				query:     query,
				querySet:  cmd.Flags().Changed("query"),
				firstPage: firstPage,
				perPage:   perPage,
				// Presence, not value: a provided 0/negative --per-page must reach the
				// API rather than be silently ignored (paging's no-clamp contract).
				perPageSet: cmd.Flags().Changed("per-page"),
				include:    include,
				reqCtx:     cmd.Context(),
				stdout:     cmd.OutOrStdout(),
				stderr:     cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "", "Filter by actor type (one of: "+strings.Join(supportedKindNames(), ", ")+"), sent as the kind parameter (directory list only)")
	cmd.Flags().StringVar(&roleID, "role-id", "", "Filter to the actors filling a role by its id, sent as the role_id parameter (directory list only; omitted when empty)")
	cmd.Flags().StringVarP(&query, "query", "q", "", "Free-text search over actor names, sent as the endpoint's q parameter (directory list only; omitted when empty)")
	cmd.Flags().BoolVar(&firstPage, "first-page", false, "Fetch only the first page and signal if more actors exist (directory list only)")
	cmd.Flags().IntVar(&perPage, "per-page", 0, "Page size for the walk (directory list only; the API owns the valid range)")
	cmd.Flags().StringSliceVar(&include, "include", nil, "Related resources to embed on a single actor read (roles,assignments)")
	return cmd
}

// supportedKindNames lists the supported actor kinds in stable (sorted) order for
// the --kind flag help, so the help text and the reject-unknown error name the same
// deterministic set.
func supportedKindNames() []string {
	names := make([]string, 0, len(supportedActorKinds))
	for name := range supportedActorKinds {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
