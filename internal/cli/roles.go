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

// incompleteWalkNote is the stderr line the default walk writes when it stops on
// an error after gathering at least one page: the partial set is already on
// stdout, so this names the cause and marks the list incomplete — a partial list
// is never silently presented as complete (CONSTITUTION VI; interface-cli). The
// %s is the cause; the command exits non-zero (classifyClientError(Stop)).
const incompleteWalkNote = "note: result is incomplete — %s; the roles shown are a partial set"

// moreRolesNote is the stderr line the --first-page opt-out writes when the first
// page reports more pages exist: the operator chose the boundary, so this is not
// an error (exit 0) — it just keeps a partial list from being read as complete
// (interface-cli; CONSTITUTION VI).
const moreRolesNote = "note: more roles exist than shown; re-run without --first-page to fetch all"

// rolesSeam supplies everything the `roles` reads need from the outside, so
// runRoles is pure over injected values and every branch runs offline. It is the
// same shape Identity Read's meSeam exposes (assemble + newClient + sleep +
// resolveFormat), so productionSeam satisfies it unchanged and the existing test
// fakes drive it; `roles` builds the retrying executor once from these and hands
// it to both a direct Execute (single read) and paging.All (the list walk). It
// never reads ctx.Cred.Token — the token rides 007's AuthTransport inside the
// client.
type rolesSeam interface {
	assemble(baseURL string) apiclient.ConnectionContext
	newClient(ctx apiclient.ConnectionContext) (*apiclient.Client, error)
	sleep() func(time.Duration)
	resolveSelection(flagValue string) (output.Selection, error)
	readTemplateSource(ref output.TemplateRef) (string, error)
}

// rolesConfig carries everything runRoles needs, gathered by the command's RunE.
// Keeping runRoles a function of injected values makes the whole read — validate,
// assemble, build, walk/send, render/classify — testable over a fake transport
// with no real network or ~/.glassfrogrc.
type rolesConfig struct {
	seam       rolesSeam
	baseURL    string   // inherited persistent --base-url (may be empty)
	outputFlag string   // inherited persistent --output (may be empty), resolved before any request
	args       []string // 0 → list, 1 → single read by id

	// list flags (the single-read branch forbids them; validateRolesFlags guards)
	parent      string
	person      string
	tag         string
	hasSubroles *bool // tri-state: nil = omitted, else the requested value
	firstPage   bool
	perPage     int
	perPageSet  bool // whether --per-page was provided (cmd.Flags().Changed); presence, not value

	// single-read flag (the list branch forbids it; validateRolesFlags guards)
	include []string

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runRoles is the pure orchestration the `roles` leaf delegates to: resolve the
// output format, validate the flag combinations (and, on the single branch, the
// --include values) fail-fast before any assembly or request, assemble the
// connection and build the retrying executor once, then dispatch on whether a
// positional id was given — 0 args → the org-wide list walk, 1 arg → the single
// role read. It returns the code-free Outcome the command maps onto dispatch's
// error channel; it adds no new Outcome/ExitCode and never reads the token.
func runRoles(cfg rolesConfig) (Outcome, error) {
	// 1. Resolve the render target FIRST (020 widened by 035, ADR-1/ADR-4): a
	//    present-but-invalid selector — or, for a user template, a missing/unparseable
	//    source or empty stdin — fails fast as a usage error before any assembly or
	//    request.
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	hasID := len(cfg.args) == 1

	// 2. Validate the flag combinations BEFORE any assembly or request (fail-fast
	//    usage error, no wasted call — pinned by a tripwire transport).
	if err := validateRolesFlags(hasID, rolesFlagState{
		parentSet:      cfg.parent != "",
		personSet:      cfg.person != "",
		tagSet:         cfg.tag != "",
		hasSubrolesSet: cfg.hasSubroles != nil,
		firstPage:      cfg.firstPage,
		perPageSet:     cfg.perPageSet,
		includeSet:     len(cfg.include) > 0,
	}); err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}

	// 2b. On the single branch, validate --include against the closed enum BEFORE
	//     any request (a bad value would be silently ignored by the API, returning
	//     the role without the embed — 013's validateStatus rationale). The id is
	//     NOT validated locally (ADR-4): the API 404s an unknown/malformed id.
	if hasID {
		if err := validateRolesInclude(cfg.include); err != nil {
			fmt.Fprintln(cfg.stderr, err.Error())
			return UsageError, err
		}
	}

	// 3. Resolve the connection and build the client + retrying executor once. A
	//    base-URL error surfaces here (no doomed send); classify + report it.
	ctx := cfg.seam.assemble(cfg.baseURL)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	exec := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, cfg.seam.sleep(), cfg.stderr)

	if hasID {
		return runRoleGet(cfg, exec, rt, cfg.args[0])
	}
	return runRolesList(cfg, exec, rt)
}

// runRolesList performs the org-wide list read. The output format changes ONLY
// how the gathered set is rendered — never how much is fetched: json/yaml and
// full/compact return the same roles, walked to completion by default, and
// signal incompleteness the same way (a stderr note, never a silently short
// list). A different fetch depth per format would be unpredictable, so both
// formats share the walk.
//
// The default path walks GET /roles to completion via paging.All; --first-page
// opts out to a single page (both formats). A first-page failure (no records
// gathered) reports like any read error — nothing on stdout; a mid-walk failure
// renders the partial set and exits non-zero via classifyClientError(Stop). The
// structured walk preserves each role's raw bytes (paging.All over
// json.RawMessage), so no field is dropped or number coerced (018 fidelity);
// only the {data:[…]} envelope is synthesized, because an aggregate of N pages
// has no single page's meta — completeness travels on stderr, in-band meta is
// not needed.
func runRolesList(cfg rolesConfig, exec executor, rt renderTarget) (Outcome, error) {
	req := apiclient.Request{Method: http.MethodGet, Path: "/roles", Query: rolesListQuery(cfg)}

	// The --first-page opt-out: a single page in EVERY format, signalling if more
	// exist (exit 0). The operator chose the boundary, so it is not an error.
	if cfg.firstPage {
		return runRolesFirstPage(cfg, exec, rt, req)
	}

	// Structured: walk to completion preserving each role's raw bytes, then emit
	// the aggregated {data:[…]} document. A user template selects DefaultFormat (a
	// human format), so it never lands here.
	if machineFmt, ok := rt.format.MachineFormat(); ok {
		res := paging.All[json.RawMessage](cfg.reqCtx, exec, req, rolesWalkOptions(cfg)...)
		if res.Stop != nil && len(res.Records) == 0 {
			return reportFailure(cfg.stdout, cfg.stderr, rt.format, res.Stop)
		}
		doc, rerr := aggregateRawData(machineFmt, res.Records)
		if rerr != nil {
			// Buffer-then-write: a render failure leaves stdout empty and maps to
			// RuntimeError(1). The error is token-free (018 contract).
			fmt.Fprintln(cfg.stderr, rerr.Error())
			return RuntimeError, rerr
		}
		_, _ = cfg.stdout.Write(doc)
		if res.Stop != nil {
			return reportIncompleteWalk(cfg.stderr, res.Stop)
		}
		return Success, nil
	}

	// Human / user template: walk to completion, render the org-roles projection
	// through the built-in template or the caller's template (writeHuman).
	res := paging.All[glassfrog.Role](cfg.reqCtx, exec, req, rolesWalkOptions(cfg)...)
	if res.Stop != nil && len(res.Records) == 0 {
		// A walk that stopped before gathering any record is a clean failure (e.g. a
		// first-page transport/auth/API error): no partial set to show, so report it
		// like any read error — nothing on stdout.
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, res.Stop)
	}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceOrgRoles, rt.format, res.Records); outcome != Success {
		return outcome, rerr
	}
	if res.Stop != nil {
		return reportIncompleteWalk(cfg.stderr, res.Stop)
	}
	return Success, nil
}

// reportIncompleteWalk writes the mid-walk incomplete note (the partial set is
// already on stdout) and returns the classified non-zero outcome. The Stop error
// is refined once so a non-2xx splits into permission/rate-limit as 015 landed.
// Shared by the human and structured walks so both flag incompleteness the same
// way.
func reportIncompleteWalk(stderr io.Writer, stop error) (Outcome, error) {
	refined := refineClientError(stop)
	fmt.Fprintf(stderr, incompleteWalkNote+"\n", refined.Error())
	// Return the REFINED error, not the original stop: the note text, the
	// classified outcome, and the returned error must all derive from the same
	// value (the reportClientError invariant), so a downstream errors.As sees the
	// extracted *ProblemError on a mid-walk 401/403/429 rather than the generic
	// *ResponseError.
	return classifyClientError(refined), refined
}

// aggregateRawData concatenates the verbatim per-record bytes gathered across a
// walk into a single {"data":[…]} document and renders it in the structured
// format. It is resource-neutral — the org `roles` list (025), `subroles` (026),
// and `domains` (033) all aggregate their walked pages through it. Each record's
// bytes are preserved exactly (no struct round-trip → no field dropped, no number
// coerced — 018 fidelity); only the envelope is synthesized, because an aggregate
// of N pages has no single page's meta. A nil record set renders {"data":[]}, not
// {"data":null}, so an empty result is a valid empty list rather than a null.
func aggregateRawData(f output.Format, records []json.RawMessage) ([]byte, error) {
	if records == nil {
		records = []json.RawMessage{}
	}
	payload, err := json.Marshal(struct {
		Data []json.RawMessage `json:"data"`
	}{Data: records})
	if err != nil {
		return nil, err
	}
	return output.RenderSuccess(f, payload)
}

// runRolesFirstPage performs the --first-page opt-out: a single GET /roles page
// (no walk) in EVERY format, with one stderr note when the API reports more
// pages exist (still exit 0 — the operator chose the boundary). The structured
// path emits the same {data:[…]} envelope the default walk does (so the json
// shape does not change with --first-page); the human path renders the
// projection. --per-page (if set) sizes the single request; the walker is not
// involved.
func runRolesFirstPage(cfg rolesConfig, exec executor, rt renderTarget, req apiclient.Request) (Outcome, error) {
	if cfg.perPageSet {
		// Pass the provided value through as-is — no client-side clamp (paging's
		// contract): an out-of-range value (0, negative, > API max) surfaces the
		// API's rejection rather than being silently ignored.
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
			fmt.Fprintln(cfg.stderr, moreRolesNote)
		}
		return Success, nil
	}

	var page glassfrog.Page[glassfrog.Role]
	if _, err := exec.Execute(cfg.reqCtx, req, &page); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	if outcome, rerr := writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceOrgRoles, rt.format, page.Data); outcome != Success {
		return outcome, rerr
	}
	if page.Meta.Pagination.HasNextPage {
		fmt.Fprintln(cfg.stderr, moreRolesNote)
	}
	return Success, nil
}

// rolesListQuery builds the GET /roles query from the validated list filters.
// Each filter is sent only when supplied; --has-subroles is tri-state, sent only
// when the operator set it (cfg.hasSubroles != nil) so omitted ≠ false. A nil
// return (no filters) leaves the request unparameterised.
func rolesListQuery(cfg rolesConfig) url.Values {
	q := url.Values{}
	if cfg.parent != "" {
		q.Set("parent_role_id", cfg.parent)
	}
	if cfg.person != "" {
		q.Set("person_id", cfg.person)
	}
	if cfg.tag != "" {
		q.Set("tag", cfg.tag)
	}
	if cfg.hasSubroles != nil {
		q.Set("has_subroles", strconv.FormatBool(*cfg.hasSubroles))
	}
	if len(q) == 0 {
		return nil
	}
	return q
}

// rolesWalkOptions builds the paging options for the default walk. --per-page
// (016's WithPageSize) sizes the walk when the flag was provided (by presence,
// not value): the value is passed through as-is — no client-side clamp (paging's
// contract) — so an out-of-range value (0, negative, > API max) surfaces the
// API's rejection as the walk's Stop rather than being silently ignored.
func rolesWalkOptions(cfg rolesConfig) []paging.Option {
	if cfg.perPageSet {
		return []paging.Option{paging.WithPageSize(cfg.perPage)}
	}
	return nil
}

// cloneQuery returns a shallow-safe copy of a url.Values so a first-page request
// can set per_page without mutating a caller-shared map. A nil input yields a
// fresh map ready for Set. Resource-neutral — shared by the `roles` (025),
// `subroles` (026), and `domains` (033) first-page reads.
func cloneQuery(src url.Values) url.Values {
	dst := make(url.Values, len(src)+1)
	for k, v := range src {
		cp := make([]string, len(v))
		copy(cp, v)
		dst[k] = cp
	}
	return dst
}

// runRoleGet reads a single role by id (GET /roles/{id}). It sends one Execute
// (no walk) with ?include= built from the already-validated --include values
// (comma-joined, style:form explode:false), passing the id through unvalidated
// (ADR-4) so an unknown/malformed id surfaces as the API's 404/4xx via the shared
// classifier. A structured --output emits the raw {data: RoleDetail} payload
// verbatim (018); the human path decodes the RoleDetail and renders the `role`
// template over a RoleView that also carries the requested-include set, so each
// section is omitted when unrequested and shows an explicit-absence marker when
// requested-but-empty (ADR-2).
func runRoleGet(cfg rolesConfig, exec executor, rt renderTarget, id string) (Outcome, error) {
	var q url.Values
	if len(cfg.include) > 0 {
		q = url.Values{"include": {strings.Join(cfg.include, ",")}}
	}
	// Escape the id as a single path segment: the id is passed through unvalidated
	// (ADR-4), but a raw `/` or `..` would otherwise redirect the request to a
	// different endpoint or traverse the path. PathEscape is a no-op for a valid
	// role_… id and keeps a malformed/adversarial id one opaque segment the API
	// reports as a 404 — upholding ADR-4 (don't validate locally) while preventing
	// endpoint injection.
	req := apiclient.Request{Method: http.MethodGet, Path: "/roles/" + url.PathEscape(id), Query: q}

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

	var doc glassfrog.RoleDocument
	if _, err := exec.Execute(cfg.reqCtx, req, &doc); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	view := render.RoleView{Detail: doc.Data, Requested: includeSet(cfg.include)}
	return writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceRole, rt.format, view)
}

// supportedRoleIncludes is the closed enum of --include values getRole accepts
// (spec). A value outside it is rejected fail-fast (validateRolesInclude).
var supportedRoleIncludes = map[string]bool{
	"assignments": true,
	"subroles":    true,
	"parent_role": true,
	"policies":    true,
	"notes":       true,
	"skills":      true,
}

// validateRolesInclude rejects any unsupported --include value against the closed
// enum, before any request (the 011 validateInclude shape, pinned by a transport
// tripwire): the API would silently ignore a bad value and return the role
// WITHOUT the requested embed, so this fails loud naming the offending value(s)
// and the supported set. Each unsupported value is quoted individually and the
// noun agrees in number; values are reported in stable (sorted) order.
func validateRolesInclude(targets []string) error {
	var unsupported []string
	for _, t := range targets {
		if !supportedRoleIncludes[t] {
			unsupported = append(unsupported, t)
		}
	}
	if len(unsupported) == 0 {
		return nil
	}
	sort.Strings(unsupported)
	quoted := make([]string, len(unsupported))
	for i, t := range unsupported {
		quoted[i] = fmt.Sprintf("%q", t)
	}
	noun := "value"
	if len(unsupported) > 1 {
		noun = "values"
	}
	return fmt.Errorf(
		"unsupported --include %s %s — supported: %s",
		noun, strings.Join(quoted, ", "), strings.Join(supportedRoleIncludeNames(), ", "),
	)
}

// supportedRoleIncludeNames lists the supported include values in stable order
// for the usage message.
func supportedRoleIncludeNames() []string {
	names := make([]string, 0, len(supportedRoleIncludes))
	for name := range supportedRoleIncludes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// includeSet turns the validated --include slice into a presence set the `role`
// RoleView carries, so the template can tell a requested-but-empty section from
// an unrequested one.
func includeSet(includes []string) map[string]bool {
	m := make(map[string]bool, len(includes))
	for _, v := range includes {
		m[v] = true
	}
	return m
}

// rolesFlagState carries which list/single flags were supplied, so
// validateRolesFlags can reject a cross-branch misuse without re-reading cobra.
type rolesFlagState struct {
	parentSet      bool
	personSet      bool
	tagSet         bool
	hasSubrolesSet bool
	firstPage      bool
	perPageSet     bool
	includeSet     bool
}

// validateRolesFlags rejects the cross-branch flag misuses fail-fast, before any
// request (the 011/013 validate-before-call shape, pinned by a transport
// tripwire): the list filters (--parent/--person/--tag/--has-subroles) and the
// walk controls (--first-page/--per-page) apply only to the list, so passing any
// of them with a role id is a usage error; --include applies only to a single
// role, so passing it without an id is a usage error. cobra's MaximumNArgs(1)
// already rejects more than one positional. The message names the misuse and the
// fix.
func validateRolesFlags(hasID bool, fs rolesFlagState) error {
	if hasID {
		var offending []string
		if fs.parentSet {
			offending = append(offending, "--parent")
		}
		if fs.personSet {
			offending = append(offending, "--person")
		}
		if fs.tagSet {
			offending = append(offending, "--tag")
		}
		if fs.hasSubrolesSet {
			offending = append(offending, "--has-subroles")
		}
		if fs.firstPage {
			offending = append(offending, "--first-page")
		}
		if fs.perPageSet {
			offending = append(offending, "--per-page")
		}
		if len(offending) > 0 {
			return fmt.Errorf(
				"%s %s the role list, not a single role — remove %s or omit the role id",
				joinFlags(offending), pluralVerb(len(offending)), pluralThem(len(offending)),
			)
		}
		return nil
	}
	if fs.includeSet {
		return fmt.Errorf("--include applies to a single role; pass a role id (e.g. `glassfrog roles role_...`)")
	}
	return nil
}

// joinFlags renders a flag list for a usage message: "--a", "--a and --b", or
// "--a, --b and --c".
func joinFlags(flags []string) string {
	switch len(flags) {
	case 1:
		return flags[0]
	case 2:
		return flags[0] + " and " + flags[1]
	default:
		return strings.Join(flags[:len(flags)-1], ", ") + " and " + flags[len(flags)-1]
	}
}

func pluralVerb(n int) string {
	if n == 1 {
		return "applies to"
	}
	return "apply to"
}

func pluralThem(n int) string {
	if n == 1 {
		return "it"
	}
	return "them"
}

// newRolesCommand builds the runnable `roles` leaf (ADR-1): a guard-ready cobra
// command with an optional positional id (Args: cobra.MaximumNArgs(1)), a
// non-empty Short, and SilenceErrors/SilenceUsage so runRoles owns its messages.
// It declares the list and single-read flags and reads the inherited persistent
// --base-url/--output flags, then delegates to the pure runRoles. It REPLACES the
// earlier stub `roles` group (its `list`/`get` "not yet implemented" children are
// removed); Assemble wires this seam-taking constructor. The seam is injected so
// tests drive a fake one; production passes productionSeam{}.
func newRolesCommand(seam rolesSeam) *cobra.Command {
	var (
		parent      string
		person      string
		tag         string
		hasSubroles bool
		firstPage   bool
		perPage     int
		include     []string
	)
	cmd := &cobra.Command{
		Use:   "roles [id]",
		Short: "Read the organization's roles, or one role by id",
		Long: "roles lists every role in the organization (walking pages to completion " +
			"by default), or reads one role by id when a positional id is given. Unlike " +
			"the token-scoped `me roles`, this is the whole-organization surface. Each " +
			"role is shown as a reshaped projection — never the raw API envelope. The " +
			"single read can embed related resources with --include.",
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
			// --has-subroles is tri-state: send it only when the operator set it
			// (Changed), so omitted ≠ false (interface-cli; risk note).
			var hasSubrolesPtr *bool
			if cmd.Flags().Changed("has-subroles") {
				hasSubrolesPtr = &hasSubroles
			}
			outcome, oerr := runRoles(rolesConfig{
				seam:        seam,
				baseURL:     baseURL,
				outputFlag:  outputFlag,
				args:        args,
				parent:      parent,
				person:      person,
				tag:         tag,
				hasSubroles: hasSubrolesPtr,
				firstPage:   firstPage,
				perPage:     perPage,
				// Presence, not value: --per-page=0 alongside an id must still be
				// rejected as a list-only flag (Changed), and a provided 0/negative
				// must reach the API rather than be silently ignored.
				perPageSet: cmd.Flags().Changed("per-page"),
				include:    include,
				reqCtx:     cmd.Context(),
				stdout:     cmd.OutOrStdout(),
				stderr:     cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	cmd.Flags().StringVar(&parent, "parent", "", "Filter to roles within a parent role (role_…)")
	cmd.Flags().StringVar(&person, "person", "", "Filter to roles assigned to a person or agent (per_/agt_…)")
	cmd.Flags().StringVar(&tag, "tag", "", "Filter roles by tag name (case-insensitive)")
	cmd.Flags().BoolVar(&hasSubroles, "has-subroles", false, "Filter by whether a role has sub-roles (omit for all)")
	cmd.Flags().BoolVar(&firstPage, "first-page", false, "Fetch only the first page and signal if more roles exist")
	cmd.Flags().IntVar(&perPage, "per-page", 0, "Page size for the walk (the API owns the valid range)")
	cmd.Flags().StringSliceVar(&include, "include", nil, "Related resources to embed on a single role (assignments,subroles,parent_role,policies,notes,skills)")
	return cmd
}
