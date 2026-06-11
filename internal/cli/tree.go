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
	"github.com/Luscii/cli-glassfrog/internal/render"
	"github.com/spf13/cobra"
)

// treeSeam supplies everything the `tree` read needs from the outside, so runTree
// is pure over injected values and every branch runs offline. It is the same
// shape Identity Read's meSeam and Role Reads' rolesSeam expose (assemble +
// newClient + sleep + resolveFormat), so productionSeam satisfies it unchanged
// and the existing test fakes drive it. The tree reads are unpaginated — one
// Execute, no walk — but they still build the retrying executor (017) so a 429 is
// retried like every other read. It never reads ctx.Cred.Token — the token rides
// 007's AuthTransport inside the client.
type treeSeam interface {
	assemble(baseURL string) apiclient.ConnectionContext
	newClient(ctx apiclient.ConnectionContext) (*apiclient.Client, error)
	sleep() func(time.Duration)
	resolveSelection(flagValue string) (output.Selection, error)
	readTemplateSource(ref output.TemplateRef) (string, error)
}

// treeConfig carries everything runTree needs, gathered by the command's RunE.
// Keeping runTree a function of injected values makes the whole read — validate,
// assemble, build, send, render/classify — testable over a fake transport with no
// real network or ~/.glassfrogrc.
type treeConfig struct {
	seam       treeSeam
	baseURL    string   // inherited persistent --base-url (may be empty)
	outputFlag string   // inherited persistent --output (may be empty), resolved before any request
	args       []string // 0 → whole-org tree, 1 → subtree rooted at args[0]

	depth    int  // --depth value (meaningful only when depthSet)
	depthSet bool // whether --depth was provided (Changed); 0 = root only ≠ omitted = full tree
	include  []string

	// Pagination flags are declared on `tree` only to reject them: the tree is a
	// single unpaginated document, so --first-page/--per-page are a usage error
	// (interface "two completeness models"). Presence, not value.
	firstPage  bool
	perPageSet bool

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// supportedTreeIncludes is the closed enum of --include values the tree reads
// accept (interface; getOrgTree/getRoleTree). A value outside it is rejected
// fail-fast (the API would otherwise silently ignore it — plan ADR-4).
var supportedTreeIncludes = map[string]bool{
	"accountabilities": true,
	"domains":          true,
	"members":          true,
}

// runTree is the pure orchestration the `tree` leaf delegates to: resolve the
// output format, validate the flag combinations and --include values fail-fast
// before any assembly or request, assemble the connection and build the retrying
// executor, then branch on whether a positional id was given — 0 args →
// GET /tree (whole org), 1 arg → GET /roles/{id}/tree (rooted subtree). It issues
// exactly ONE request (the tree is unpaginated — no walk), then renders. It adds
// no new Outcome/ExitCode and never reads the token.
func runTree(cfg treeConfig) (Outcome, error) {
	// 1. Resolve the render target FIRST (020 widened by 035, ADR-1/ADR-4): a
	//    present-but-invalid selector — or, for a user template, a missing/unparseable
	//    source or empty stdin — fails fast as a usage error before any assembly or
	//    request.
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	// 2. Validate the flag combinations BEFORE any assembly or request (fail-fast
	//    usage error, no wasted call — pinned by a tripwire transport): the
	//    pagination flags are rejected (tree is unpaginated) and a negative --depth
	//    is rejected locally (cheap, avoids a doomed call the API would 400).
	if err := validateTreeFlags(cfg); err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}

	// 2b. Validate --include against the tree's closed set BEFORE any request (a
	//     bad value would be silently ignored by the API, returning the tree
	//     without the embed). The id is NOT validated locally (ADR-4): the API
	//     404s an unknown/malformed id.
	if err := validateIncludeSet(cfg.include, supportedTreeIncludes); err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}

	// 3. Resolve the connection and build the client + retrying executor. A
	//    base-URL error surfaces here (no doomed send); classify + report it.
	ctx := cfg.seam.assemble(cfg.baseURL)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	exec := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, cfg.seam.sleep(), cfg.stderr)

	return runTreeRead(cfg, exec, rt, treeRequest(cfg))
}

// treeRequest builds the single GET request: GET /tree for the whole org (no
// positional), or GET /roles/{id}/tree for the rooted subtree. The id is escaped
// as one path segment (passed through unvalidated per ADR-4, but a raw `/`/`..`
// must not redirect or traverse — PathEscape is a no-op for a valid role_… id and
// keeps a malformed id one opaque segment the API 404s, mirroring runRoleGet).
func treeRequest(cfg treeConfig) apiclient.Request {
	path := "/tree"
	if len(cfg.args) == 1 {
		path = "/roles/" + url.PathEscape(cfg.args[0]) + "/tree"
	}
	return apiclient.Request{Method: http.MethodGet, Path: path, Query: treeQuery(cfg)}
}

// treeQuery builds the tree query: depth is sent ONLY when the flag was set
// (Changed), so --depth 0 (root only) is distinct from omitting it (full tree);
// include is comma-joined (style:form explode:false). A nil return (neither set)
// leaves the request unparameterised.
func treeQuery(cfg treeConfig) url.Values {
	q := url.Values{}
	if cfg.depthSet {
		q.Set("depth", strconv.Itoa(cfg.depth))
	}
	if len(cfg.include) > 0 {
		q.Set("include", strings.Join(cfg.include, ","))
	}
	if len(q) == 0 {
		return nil
	}
	return q
}

// runTreeRead issues the single tree request and renders the result. It mirrors
// runRoleGet (025): a structured --output emits the raw {data: TreeNode} payload
// verbatim (018 — the recursion needs no special machine-path handling); the
// human path decodes the TreeDocument and renders the `tree` template over a
// TreeView that flattens the recursion into depth-carrying rows and carries the
// requested-include set, so each per-node section is omitted when unrequested and
// shows an explicit-absence marker when requested-but-empty (ADR-2). There is no
// walk and no incompleteness note — the response IS the complete (depth-bounded)
// tree.
func runTreeRead(cfg treeConfig, exec executor, rt renderTarget, req apiclient.Request) (Outcome, error) {
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

	var doc glassfrog.TreeDocument
	if _, err := exec.Execute(cfg.reqCtx, req, &doc); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	view := render.NewTreeView(doc.Data, includeSet(cfg.include))
	return writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceTree, rt.format, view)
}

// validateTreeFlags rejects the tree's flag misuses fail-fast, before any request
// (the 011/013 validate-before-call shape, pinned by a transport tripwire): the
// pagination flags (--first-page/--per-page) apply only to a paginated read, so
// passing either on the unpaginated tree is a usage error; a negative --depth is
// rejected locally (the API would 400 it — cheaper to catch here). --depth 0 is
// valid (root only). The message names the misuse and the fix.
func validateTreeFlags(cfg treeConfig) error {
	var offending []string
	if cfg.firstPage {
		offending = append(offending, "--first-page")
	}
	if cfg.perPageSet {
		offending = append(offending, "--per-page")
	}
	if len(offending) > 0 {
		return fmt.Errorf(
			"%s %s a paginated read; the tree is a single unpaginated document — remove %s (use `glassfrog subroles` for a paginated child list)",
			joinFlags(offending), pluralVerb(len(offending)), pluralThem(len(offending)),
		)
	}
	if cfg.depthSet && cfg.depth < 0 {
		return fmt.Errorf("--depth must be 0 or greater (0 = the root node alone; omit --depth for the full subtree)")
	}
	return nil
}

// newTreeCommand builds the runnable `tree` leaf (ADR-1): a guard-ready cobra
// command with an optional positional id (Args: cobra.MaximumNArgs(1)), a
// non-empty Short, and SilenceErrors/SilenceUsage so runTree owns its messages.
// It declares the tree flags (--depth, --include) and — to reject them with a
// friendly message rather than a bare cobra "unknown flag" — the pagination flags
// it forbids (--first-page, --per-page). It reads the inherited persistent
// --base-url/--output flags, then delegates to the pure runTree. The seam is
// injected so tests drive a fake one; production passes productionSeam{}.
func newTreeCommand(seam treeSeam) *cobra.Command {
	var (
		depth     int
		include   []string
		firstPage bool
		perPage   int
	)
	cmd := &cobra.Command{
		Use:   "tree [id]",
		Short: "Read the organization's role tree, or the subtree rooted at a role",
		Long: "tree reads the circle hierarchy — the nesting between roles — as a single " +
			"nested document. With no positional it reads the whole organization (the anchor " +
			"role as root); with a role id it reads the subtree rooted at that role. The tree " +
			"is returned in one unpaginated response, bounded by an optional --depth. Each node " +
			"can embed related resources with --include. This is the nesting view; `roles` is the " +
			"flat list and `subroles` lists one role's immediate children.",
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
			outcome, oerr := runTree(treeConfig{
				seam:       seam,
				baseURL:    baseURL,
				outputFlag: outputFlag,
				args:       args,
				depth:      depth,
				// --depth is optional: send it only when set (Changed), so --depth 0
				// (root only) is distinct from omitting it (full tree).
				depthSet:   cmd.Flags().Changed("depth"),
				include:    include,
				firstPage:  firstPage,
				perPageSet: cmd.Flags().Changed("per-page"),
				reqCtx:     cmd.Context(),
				stdout:     cmd.OutOrStdout(),
				stderr:     cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	cmd.Flags().IntVar(&depth, "depth", 0, "Maximum descendant depth (0 = root only; omit for the full subtree)")
	cmd.Flags().StringSliceVar(&include, "include", nil, "Per-node related resources to embed (accountabilities,domains,members)")
	cmd.Flags().BoolVar(&firstPage, "first-page", false, "Not valid on tree (the tree is unpaginated) — use `subroles` for a paginated list")
	cmd.Flags().IntVar(&perPage, "per-page", 0, "Not valid on tree (the tree is unpaginated) — use `subroles` for a paginated list")
	return cmd
}
