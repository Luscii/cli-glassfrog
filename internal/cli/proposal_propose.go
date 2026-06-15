package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/render"
	"github.com/spf13/cobra"
)

// proposalProposeConfig carries everything runProposalPropose needs, gathered by the
// propose leaf's RunE. Keeping the transition a function of injected values makes the
// whole flow — resolve, assemble, send, render/classify — testable over a fake transport
// with no real network, ~/.glassfrogrc, pipe, or filesystem. The propose transition has
// no editable inputs (it advances a proposal whole), so there is no flag value to carry;
// the only positional is the prp_ id sent in the request path.
type proposalProposeConfig struct {
	seam           proposalSeam
	baseURL        string // inherited persistent --base-url (may be empty)
	baseURLPresent bool   // whether --base-url was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	outputFlag     string // inherited persistent --output (may be empty), resolved before any request
	outputPresent  bool   // whether --output was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	proposalID     string // the required positional prp_ id (ExactArgs(1)), sent as the /proposals/{id}/propose path id

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runProposalPropose is the pure orchestration the `propose` leaf delegates to — the
// `propose` transition of the governance write flow and the first command to issue a
// bodyless POST that ALSO decodes a response body. It resolves the output format (020)
// FIRST (a present-but-invalid --output is a fail-fast UsageError(2) before any assembly
// or request — there are no input-field checks, the leaf has no flags), then assembles
// the connection, builds the retrying executor, and sends ONE Execute to
// POST /proposals/{id}/propose with NO body, NO Content-Type (the bodyless discard
// DELETE shape, not the create body shape), and out == &Document[Proposal] (the 200
// carries the advanced proposal to decode). It sends NO prior GET — the transition is
// server-authorized, so the command does not pre-read available_transitions (plan ADR-3)
// — and NO If-Match (a transition has no prior ETag). A POST is never auto-retried on 429
// (017's isSafeMethod gate), so a rate-limited advance surfaces once and cannot
// double-fire. On success (a 200) it renders the advanced proposal exactly as
// proposal get does — machine format → output.RenderSuccess over the raw {data: Proposal}
// bytes; human/-o template → writeHuman over the decoded ProposalView. On ANY error it
// routes to reportFailure with NO status interception — a 404 (unknown/invisible
// proposal) and a 422 (transition not allowed) are REAL failures, the exact inverse of
// discard's 404-as-success (plan ADR-3); the Premium 403 is a generic PermissionError(4)
// with no plan-gate message. The prp_ id is escaped as one path segment but passed
// through unvalidated. It adds no new Outcome/ExitCode/model/render key and never reads
// the token.
func runProposalPropose(cfg proposalProposeConfig) (Outcome, error) {
	// 1. Resolve the render target FIRST (020 widened by 035): a present-but-invalid
	//    selector — or, for a user template, a missing/unparseable source or empty
	//    stdin — fails fast as a usage error before any assembly or request. There are
	//    no input-field checks (propose has no editable flags); resolving --output first
	//    keeps error precedence consistent with the siblings.
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

	// 3. Build ONE bodyless POST /proposals/{id}/propose: no Body, ContentType left empty
	//    (no Content-Type header — the discard DELETE shape, not the create body shape).
	//    Escape the id as a single path segment: passed through unvalidated (plan ADR-1),
	//    but a raw `/` or `..` must not redirect the request. NO If-Match, NO prior GET
	//    (the transition is server-authorized — plan ADR-3).
	req := apiclient.Request{
		Method: http.MethodPost,
		Path:   "/proposals/" + url.PathEscape(cfg.proposalID) + "/propose",
	}

	// 4. Dispatch on the resolved render target. The 200 carries the advanced proposal,
	//    so out decodes the {data: Proposal} document. On ANY error route through the
	//    shared reportFailure (032) with NO status interception — 404 and 422 are real
	//    failures (the inverse of discard's 404-as-success). A user template (035) is a
	//    human render, so the structured branch is taken only for a built-in json/yaml
	//    selection (rt.tmpl is nil then, by construction in resolveRenderTarget).
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
	// writeHuman renders through the selected user template (035) or the built-in
	// `proposal` template (019/056), buffer-then-write.
	return writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceProposal, rt.format, view)
}

// newProposalProposeCommand builds the runnable `proposal propose <prp-id>` leaf
// (ADR-1): a guard-ready cobra command with a REQUIRED positional prp_ id
// (Args: cobra.ExactArgs(1) — zero or more than one positional is a usage error before
// RunE, no request), a non-empty Short, and SilenceErrors/SilenceUsage so
// runProposalPropose owns its messages. It declares NO flags of its own — a transition
// advances a proposal whole, so there is nothing to edit and the endpoint takes no
// request body; a stray --status/--changes/--body is therefore a cobra unknown-flag
// usage error for free (the structural guard 045/056 rely on). It reads the inherited
// persistent --base-url/--output flags, then delegates to the pure runProposalPropose.
// The seam is injected so tests drive a fake one; production passes productionSeam{}.
func newProposalProposeCommand(seam proposalSeam) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose <prp-id>",
		Short: "Advance a draft proposal into circulation",
		Long: "propose advances a draft governance proposal from `draft` into circulation — the " +
			"`propose` transition of the write flow. The positional proposal id names the proposal " +
			"to advance; the transition advances it whole, so there is nothing to edit and the " +
			"command takes no flags and sends no request body. The advance is server-authorized and " +
			"Premium-gated: the command issues the request and lets the server decide — it does not " +
			"pre-read the proposal to check whether `propose` is allowed. On success the advanced " +
			"proposal is printed — now `proposed_outside_meeting`, carrying the server-set response " +
			"deadline, the proposer's implicit no-objection, and the updated available transitions — " +
			"so a later step can read the new state and respond or withdraw. A transition the server " +
			"refuses (422) and an unknown proposal (404) are surfaced as real failures, never as " +
			"success.",
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
			outcome, oerr := runProposalPropose(proposalProposeConfig{
				seam:           seam,
				baseURL:        baseURL,
				baseURLPresent: cmd.Flags().Changed(apiclient.FlagBaseURL),
				outputFlag:     outputFlag,
				outputPresent:  cmd.Flags().Changed(output.FlagOutput),
				proposalID:     args[0],
				reqCtx:         cmd.Context(),
				stdout:         cmd.OutOrStdout(),
				stderr:         cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	return cmd
}
