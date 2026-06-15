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

// proposalWithdrawConfig carries everything runProposalWithdraw needs, gathered by the
// withdraw leaf's RunE. Keeping the transition a function of injected values makes the
// whole flow — resolve, assemble, send, render/classify — testable over a fake transport
// with no real network, ~/.glassfrogrc, pipe, or filesystem. The withdraw transition has
// no editable inputs (it returns a proposal whole to draft), so there is no flag value to
// carry; the only positional is the prp_ id sent in the request path.
type proposalWithdrawConfig struct {
	seam           proposalSeam
	baseURL        string // inherited persistent --base-url (may be empty)
	baseURLPresent bool   // whether --base-url was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	outputFlag     string // inherited persistent --output (may be empty), resolved before any request
	outputPresent  bool   // whether --output was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	proposalID     string // the required positional prp_ id (ExactArgs(1)), sent as the /proposals/{id}/withdraw path id

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runProposalWithdraw is the pure orchestration the `withdraw` leaf delegates to — the
// `withdraw` transition of the governance write flow and the structural twin of the
// landed `propose` leaf (plan ADR-1). It resolves the output format (020) FIRST (a
// present-but-invalid --output is a fail-fast UsageError(2) before any assembly or
// request — there are no input-field checks, the leaf has no flags), then assembles the
// connection, builds the retrying executor, and sends ONE Execute to
// POST /proposals/{id}/withdraw with NO body, NO Content-Type, and
// out == &Document[Proposal] (the 200 carries the withdrawn proposal — now back in
// `draft`, with proposed_at/response_deadline cleared and prior responses deleted
// server-side — to decode). It sends NO prior GET — the transition is server-authorized,
// so the command does not pre-read available_transitions (plan ADR-1) — and NO If-Match
// (a transition has no prior ETag). A POST is never auto-retried on 429 (017's
// isSafeMethod gate), so a rate-limited withdraw surfaces once and cannot double-fire. On
// success (a 200) it renders the withdrawn proposal exactly as proposal propose/get do —
// machine format → output.RenderSuccess over the raw {data: Proposal} bytes;
// human/-o template → writeHuman over the decoded ProposalView. On ANY error it routes to
// reportFailure with NO status interception — a 404 (unknown/invisible proposal) and a
// 422 (transition not allowed — already `draft`, or `withdraw` not offered) are REAL
// failures, the exact inverse of discard's 404-as-success (plan ADR-1); the Premium 403
// is a generic PermissionError(4) with no plan-gate message. The withdraw is destructive
// (the server deletes existing responses and clears the timestamps), but the command
// narrates none of it and adds no confirmation/--force guard — the deletion is legible in
// the returned data (plan ADR-2). The prp_ id is escaped as one path segment but passed
// through unvalidated. It adds no new Outcome/ExitCode/model/render key and never reads
// the token.
func runProposalWithdraw(cfg proposalWithdrawConfig) (Outcome, error) {
	// 1. Resolve the render target FIRST (020 widened by 035): a present-but-invalid
	//    selector — or, for a user template, a missing/unparseable source or empty
	//    stdin — fails fast as a usage error before any assembly or request. There are
	//    no input-field checks (withdraw has no editable flags); resolving --output first
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

	// 3. Build ONE bodyless POST /proposals/{id}/withdraw: no Body, ContentType left empty
	//    (no Content-Type header — the bodyless transition shape, not the create body
	//    shape). Escape the id as a single path segment: passed through unvalidated (plan
	//    ADR-1), but a raw `/` or `..` must not redirect the request. NO If-Match, NO prior
	//    GET (the transition is server-authorized — plan ADR-1).
	req := apiclient.Request{
		Method: http.MethodPost,
		Path:   "/proposals/" + url.PathEscape(cfg.proposalID) + "/withdraw",
	}

	// 4. Dispatch on the resolved render target. The 200 carries the withdrawn proposal
	//    (now `draft`), so out decodes the {data: Proposal} document. On ANY error route
	//    through the shared reportFailure (032) with NO status interception — 404 and 422
	//    are real failures (the inverse of discard's 404-as-success). A user template (035)
	//    is a human render, so the structured branch is taken only for a built-in json/yaml
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

// newProposalWithdrawCommand builds the runnable `proposal withdraw <prp-id>` leaf
// (ADR-1): a guard-ready cobra command with a REQUIRED positional prp_ id
// (Args: cobra.ExactArgs(1) — zero or more than one positional is a usage error before
// RunE, no request), a non-empty Short, and SilenceErrors/SilenceUsage so
// runProposalWithdraw owns its messages. It declares NO flags of its own — a transition
// returns a proposal whole to draft, so there is nothing to edit and the endpoint takes
// no request body; a stray --status/--changes/--force/--yes is therefore a cobra
// unknown-flag usage error for free (the structural guard 045/056 rely on, and the
// no-confirmation/no---force stance of plan ADR-2). It reads the inherited persistent
// --base-url/--output flags, then delegates to the pure runProposalWithdraw. The seam is
// injected so tests drive a fake one; production passes productionSeam{}.
func newProposalWithdrawCommand(seam proposalSeam) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw <prp-id>",
		Short: "Withdraw a circulating proposal back to draft",
		Long: "withdraw pulls a circulating governance proposal back off its consent window and " +
			"returns it to `draft` for re-editing — the `withdraw` transition of the write flow and " +
			"the mirror of `propose`. The positional proposal id names the proposal to withdraw; the " +
			"transition returns it whole, so there is nothing to edit and the command takes no flags " +
			"and sends no request body. The withdraw is server-authorized and Premium-gated: the " +
			"command issues the request and lets the server decide — it does not pre-read the proposal " +
			"to check whether `withdraw` is allowed. On success the proposal is printed — now `draft`, " +
			"with its proposed timestamps cleared, its prior responses deleted server-side, and the " +
			"updated available transitions (offering `propose` again) — so a later step can amend the " +
			"draft and re-circulate it. The withdraw is destructive (it deletes existing responses) " +
			"but the command prompts for no confirmation and requires no --force: it is " +
			"non-interactive and agent-driven, and surfaces the consequence in the returned data. A " +
			"transition the server refuses (422) and an unknown proposal (404) are surfaced as real " +
			"failures, never as success.",
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
			outcome, oerr := runProposalWithdraw(proposalWithdrawConfig{
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
