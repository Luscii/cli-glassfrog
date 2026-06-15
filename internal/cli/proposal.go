package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/render"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// proposalSeam supplies everything the proposal write needs from the outside, so
// runProposalCreate is pure over injected values and every branch runs offline. It is
// the 042 tensionSeam shape (assemble + newClient + sleep + resolveSelection +
// readTemplateSource) PLUS readChangesSource — the one new member, which reads the
// --changes source (stdin/file/inline) behind the seam so the source read is offline-
// testable. productionSeam satisfies it (it already satisfies tensionSeam and gains
// readChangesSource here); tests bind a fake. It never reads ctx.Cred.Token — the
// token rides 007's AuthTransport inside the client.
type proposalSeam interface {
	tensionSeam
	readChangesSource(value string) ([]byte, error)
}

// readChangesSource is the single reader of the real filesystem / terminal / stdin for
// the --changes source: it resolves the flag value (reserved stdin keyword, existing
// file, or inline JSON) over real os.Stat/os.ReadFile, the real os.Stdin behind the
// 006 bounded reader, and the real terminal check. Tests bind a fake seam over
// injected bytes, so every source branch is exercised without a real pipe or
// filesystem dependency.
func (productionSeam) readChangesSource(value string) ([]byte, error) {
	return resolveChangesSource(value, os.Stat, os.ReadFile, term.IsTerminal(int(os.Stdin.Fd())), os.Stdin)
}

// proposalCreateConfig carries everything runProposalCreate needs, gathered by the
// create leaf's RunE. Keeping the write a function of injected values makes the whole
// flow — resolve, require, read, validate, assemble, build, send, render/classify —
// testable over a fake transport with no real network, ~/.glassfrogrc, pipe, or
// filesystem. changesValue is the unresolved --changes flag string; the seam
// classifies and reads it.
type proposalCreateConfig struct {
	seam           proposalSeam
	baseURL        string // inherited persistent --base-url (may be empty)
	baseURLPresent bool   // whether --base-url was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	outputFlag     string // inherited persistent --output (may be empty), resolved before any request
	outputPresent  bool   // whether --output was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	tensionID      string // the required positional anchor tension id (ExactArgs(1)), sent as proposal.tension_id
	changesValue   string // the raw --changes flag value; the seam resolves it (stdin/file/inline)

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runProposalCreate is the pure orchestration the `create` leaf delegates to — the
// CLI's second write and the anchor of the governance write path. It resolves the
// output format (020) FIRST, then runs the pure input checks fail-fast — all BEFORE
// any assembly, so a transport tripwire confirms NO request on every rejection:
// (1) require --changes presence (else a usage error naming it); (2) read the change
// source via the seam (stdin/file/inline); (3) apply the type floor (validateChanges:
// a valid, non-empty JSON array of typed objects). Then it assembles the connection,
// builds the retrying executor, marshals {proposal:{tension_id, changes}} with the
// VERBATIM change slice, and sends ONE Execute to POST /proposals with
// Content-Type: application/json (042's landed ContentType field) and NO If-Match (a
// create has no prior ETag). A POST is never auto-retried on 429 (017's isSafeMethod
// gate), so a rate-limited create surfaces once and cannot double-submit. A structured
// --output emits the raw {data: Proposal} verbatim (018); the human path decodes
// Document[Proposal] and renders the singular `proposal` template (or a selected user
// template, 035) over a ProposalView. The anchor tension id is passed through
// unvalidated (an unknown/malformed id surfaces as the API's 404/422 via the shared
// classifier). It issues the request with NO client-side Premium check (the 403 is
// server-surfaced), adds no new Outcome/ExitCode, and never reads the token.
func runProposalCreate(cfg proposalCreateConfig) (Outcome, error) {
	// 1. Resolve the render target FIRST (020 widened by 035): a present-but-invalid
	//    selector — or, for a user template, a missing/unparseable source or empty
	//    stdin — fails fast as a usage error before any assembly or request. Resolving
	//    --output ahead of the input checks keeps error precedence consistent with the
	//    reads — an invalid --output is reported even when --changes is bad.
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.outputPresent, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	// 2. Require --changes (a changeless proposal is meaningless; reject before any
	//    request). The flag has no default, so an absent flag is empty; an explicitly
	//    blank value is rejected the same way.
	if len(bytes.TrimSpace([]byte(cfg.changesValue))) == 0 {
		err := errors.New("--changes is required (a JSON array of changes — inline, a file path, or the reserved keyword stdin)")
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}

	// 3. Read the change source (stdin/file/inline) via the seam, then apply the type
	//    floor — both BEFORE any assembly, so the no-request-on-rejection tripwire holds.
	//    A bad source (unreadable file, terminal/empty stdin) or a change set that is not
	//    a non-empty JSON array of typed objects is a fail-fast usage error.
	raw, err := cfg.seam.readChangesSource(cfg.changesValue)
	if err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}
	changes, err := validateChanges(raw)
	if err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}

	// 4. Resolve the connection and build the client + retrying executor. A base-URL
	//    error surfaces here (no doomed send); classify + report it.
	ctx := cfg.seam.assemble(cfg.baseURL, cfg.baseURLPresent)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	exec := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, cfg.seam.sleep(), cfg.stderr)

	// 5. Marshal {proposal:{tension_id, changes}} with the validated change slice
	//    carried VERBATIM ([]json.RawMessage) — never reshaped, no command key dropped.
	//    No status/proposer field (server-owned).
	bodyBytes, merr := json.Marshal(glassfrog.NewCreateProposalRequest(cfg.tensionID, changes))
	if merr != nil {
		// Marshalling the input cannot fail in practice (the changes are already valid
		// JSON); surface it as a runtime defect rather than sending a malformed body.
		fmt.Fprintln(cfg.stderr, merr.Error())
		return RuntimeError, merr
	}

	// Send ONE POST /proposals with the JSON body. ContentType rides the landed 042
	// field (no transport change); NO If-Match (a create has no prior ETag). The anchor
	// tension id rides inside the body, not the path, so no path escaping is needed.
	req := apiclient.Request{
		Method:      http.MethodPost,
		Path:        "/proposals",
		Body:        bytes.NewReader(bodyBytes),
		ContentType: "application/json",
	}

	// 6. Dispatch on the resolved render target. On any failure route through the shared
	//    reportFailure (032). A user template (035) is a human render, so the structured
	//    branch is taken only for a built-in json/yaml selection (rt.tmpl is nil then, by
	//    construction in resolveRenderTarget).
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
	// `proposal` template (019), buffer-then-write.
	return writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceProposal, rt.format, view)
}

// newProposalCommand assembles the `proposal` command group and its leaves, registered
// through the guard (group-has-children, non-empty Short, no action). The group is
// built with its children attached BEFORE being returned for registration under root, so
// the guard's ">=1 child" rule holds at attach time (the tension/auth shape, plan
// ADR-1). The `proposal` namespace reserves room for the rest of the write-flow and the
// reads (withdraw/respond, and 056's list/get); 055 attached `create`, and 057 attaches
// the `propose` transition. The group is SHARED across the proposal family under
// first-to-land-creates: 055 created it; siblings (056 reads, 057 propose) attach their
// leaves to the existing group rather than redefining it. The seam is injected so tests
// drive a fake one; production passes productionSeam{} from Assemble.
func newProposalCommand(seam proposalSeam) *cobra.Command {
	group := &cobra.Command{
		Use:   "proposal",
		Short: "Work with proposals — create a draft against a tension and advance it into circulation",
	}
	MustRegister(group, newProposalCreateCommand(seam))
	MustRegister(group, newProposalProposeCommand(seam))
	return group
}

// newProposalCreateCommand builds the runnable `proposal create <tension-id>` leaf
// (ADR-1): a guard-ready cobra command with a REQUIRED positional anchor tension id
// (Args: cobra.ExactArgs(1) — zero or more than one positional is a usage error before
// RunE, no request), a non-empty Short, and SilenceErrors/SilenceUsage so
// runProposalCreate owns its messages. It declares the --changes flag — which lives
// only on `create`, so passing it to a future `proposal get`/`list` is a cobra
// unknown-flag usage error for free (the structural guard). It reads the inherited
// persistent --base-url/--output flags, then delegates to the pure runProposalCreate.
// The seam is injected so tests drive a fake one; production passes productionSeam{}.
func newProposalCreateCommand(seam proposalSeam) *cobra.Command {
	var changes string
	cmd := &cobra.Command{
		Use:   "create <tension-id>",
		Short: "Create a draft proposal anchored to a tension",
		Long: "create raises a draft governance proposal against an existing tension — the " +
			"CLI's second write and the anchor of the governance write path. The positional " +
			"tension id is the anchor the proposal is raised against; --changes carries the " +
			"governance change set as a JSON array, sourced inline, from a file, or from piped " +
			"stdin (the reserved keyword `stdin`). The change set is passed through verbatim " +
			"above a minimal floor — every element must be an object with a non-empty `type`. " +
			"The proposer is derived from the token and the status is set to draft by the " +
			"server; neither is a flag. The created proposal is printed with its prp_ id so a " +
			"later step can advance it to circulation. A missing or malformed change set is " +
			"refused before any request. The whole write surface is Premium-gated; the command " +
			"issues the request and surfaces the server's response.",
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
			outcome, oerr := runProposalCreate(proposalCreateConfig{
				seam:           seam,
				baseURL:        baseURL,
				baseURLPresent: cmd.Flags().Changed(apiclient.FlagBaseURL),
				outputFlag:     outputFlag,
				outputPresent:  cmd.Flags().Changed(output.FlagOutput),
				tensionID:      args[0],
				changesValue:   changes,
				reqCtx:         cmd.Context(),
				stdout:         cmd.OutOrStdout(),
				stderr:         cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	cmd.Flags().StringVar(&changes, "changes", "", "The governance change set: a JSON array, given inline, as a file path, or the reserved keyword stdin (required)")
	return cmd
}
