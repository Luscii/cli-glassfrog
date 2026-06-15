package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/render"
	"github.com/spf13/cobra"
)

// supportedProposalResponses is the spec's proposal-response value set — the consent
// vocabulary on the createProposalResponse request (spec/glassfrog-api-v5.yaml:
// no_objection, bring_to_meeting). It is the single source of truth for --response
// validation. Adding a value is a one-line change tracking the vendored spec enum. It
// is a NEW set, deliberately distinct from the action/project validateStatus set
// (status.go), the tension validateTensionStatus set (tension_reads.go), the proposal
// validateProposalStatus set (proposal_reads.go), AND validateMeetingType (tension.go)
// — reusing any would accept wrong values and reject valid ones (a correctness bug,
// plan ADR-1). It lives WITH the proposal command code (the validateProposalStatus /
// validateMeetingType precedent), NOT in the shared status.go. Unlike those optional
// filter/hint validators, the value here is REQUIRED, so the set deliberately omits
// null/empty: an absent --response is a usage error, not a "no constraint" case.
var supportedProposalResponses = map[string]bool{
	"no_objection":     true,
	"bring_to_meeting": true,
}

// validateProposalResponse rejects an omitted or unsupported --response value, returning
// a usage error before any context assembly or request (the validateProposalStatus /
// validateMeetingType fail-fast discipline, plan ADR-1). UNLIKE the optional filter/hint
// validators, --response is REQUIRED: an empty value (the flag absent or explicitly
// blank) is a usage error naming --response as required and listing the supported set —
// there is no default consent answer. An unsupported non-empty value is a usage error
// naming the value and the supported set. Pure — no network, no filesystem — so it runs
// ahead of any I/O and a tripwire transport can assert nothing was sent on rejection. A
// sibling validator of validateProposalStatus / validateMeetingType, not a second copy
// of any set.
func validateProposalResponse(value string) error {
	if value == "" {
		return fmt.Errorf(
			"--response is required — supported: %s",
			strings.Join(supportedProposalResponseNames(), ", "),
		)
	}
	if supportedProposalResponses[value] {
		return nil
	}
	return fmt.Errorf(
		"unsupported --response value %q — supported: %s",
		value, strings.Join(supportedProposalResponseNames(), ", "),
	)
}

// supportedProposalResponseNames lists the supported response values in stable (sorted)
// order for the usage message, so the same input always yields the same deterministic
// text (the supportedProposalStatusNames shape).
func supportedProposalResponseNames() []string {
	names := make([]string, 0, len(supportedProposalResponses))
	for name := range supportedProposalResponses {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// proposalRespondConfig carries everything runProposalRespond needs, gathered by the
// respond leaf's RunE. Keeping the write a function of injected values makes the whole
// flow — resolve, validate, assemble, send, render/classify — testable over a fake
// transport with no real network, ~/.glassfrogrc, pipe, or filesystem. response is the
// raw --response flag value; the validator owns the required + closed-enum check.
type proposalRespondConfig struct {
	seam           proposalSeam
	baseURL        string // inherited persistent --base-url (may be empty)
	baseURLPresent bool   // whether --base-url was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	outputFlag     string // inherited persistent --output (may be empty), resolved before any request
	outputPresent  bool   // whether --output was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	proposalID     string // the required positional prp_ id (ExactArgs(1)), sent as the /proposals/{id}/responses path id
	response       string // the required --response value, validated against the closed response set before any request

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runProposalRespond is the pure orchestration the `respond` leaf delegates to — the
// consume/respond write of the governance flow, the single-write structural twin of
// Tension Capture (042). It resolves the output format (020, widened by 035) FIRST, then
// validates the required closed-enum --response fail-fast via validateProposalResponse
// (an omitted OR unsupported value is a UsageError(2) with NO request issued) — both
// pure checks run BEFORE any assembly, so a transport tripwire confirms no request on
// rejection (the 011/042/056 fail-fast discipline). Then it assembles the connection,
// builds the retrying executor, marshals {response:{value}}, and sends ONE Execute to
// POST /proposals/{id}/responses with Content-Type: application/json (042's landed
// ContentType field) and NO If-Match (recording is an append-create, not a guarded edit;
// plan ADR-3). The body carries no person field (the server derives the responding
// person from the token). A POST is never auto-retried on 429 (017's isSafeMethod gate),
// so a rate-limited recording surfaces once and cannot double-record. On 201 a structured
// --output emits the raw {data: ProposalVote} verbatim (018); the human path decodes
// Document[ProposalVote] and renders the singular `proposal-response` view surfacing the
// parent proposal_status (the auto-acceptance signal). On ANY error it routes through the
// shared reportFailure (032) with NO status interception — a 403 (Premium gate) is a
// generic PermissionError(4), a 422 (already responded) and a 404 (unknown proposal) are
// real APIError(3) failures, never folded into success. The prp_ id is escaped as one
// path segment but passed through unvalidated (an unknown id → 404). It adds no new
// Outcome/ExitCode/model/render key and never reads the token.
func runProposalRespond(cfg proposalRespondConfig) (Outcome, error) {
	// 1. Resolve the render target FIRST (020 widened by 035): a present-but-invalid
	//    selector — or, for a user template, a missing/unparseable source or empty
	//    stdin — fails fast as a usage error before any assembly or request. Resolving
	//    --output ahead of the input check keeps error precedence consistent with the
	//    siblings — an invalid --output is reported even when --response is also bad.
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.outputPresent, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	// 2. Validate the required closed-enum --response (pure, no I/O) BEFORE any assembly,
	//    so the no-request-on-rejection tripwire holds. An omitted value is the
	//    required-error; an unsupported value names the value + set. Both are UsageError(2).
	if err := validateProposalResponse(cfg.response); err != nil {
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

	// 4. Marshal {response:{value}} with the validated consent value. No person field
	//    (server-derived from the token), no status field.
	bodyBytes, merr := json.Marshal(glassfrog.NewProposalResponseInput(cfg.response))
	if merr != nil {
		// Marshalling a single validated string cannot fail in practice; surface it as a
		// runtime defect rather than sending a malformed body.
		fmt.Fprintln(cfg.stderr, merr.Error())
		return RuntimeError, merr
	}

	// Send ONE POST /proposals/{id}/responses with the JSON body. ContentType rides the
	// landed 042 field; NO If-Match (an append-create has no prior ETag of the vote to
	// guard — plan ADR-3). Escape the id as a single path segment: passed through
	// unvalidated (plan ADR-1), but a raw `/` or `..` must not redirect the request.
	req := apiclient.Request{
		Method:      http.MethodPost,
		Path:        "/proposals/" + url.PathEscape(cfg.proposalID) + "/responses",
		Body:        bytes.NewReader(bodyBytes),
		ContentType: "application/json",
	}

	// 5. Dispatch on the resolved render target. On any failure route through the shared
	//    reportFailure (032) with NO status interception — 403/422/404 are real failures.
	//    A user template (035) is a human render, so the structured branch is taken only
	//    for a built-in json/yaml selection (rt.tmpl is nil then, by construction).
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

	var doc glassfrog.Document[glassfrog.ProposalVote]
	if _, err := exec.Execute(cfg.reqCtx, req, &doc); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}
	view := render.ProposalVoteView{ProposalVote: doc.Data}
	// writeHuman renders through the selected user template (035) or the built-in
	// `proposal-response` template (058), buffer-then-write.
	return writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceProposalResponse, rt.format, view)
}

// newProposalRespondCommand builds the runnable `proposal respond <prp-id>` leaf
// (ADR-1): a guard-ready cobra command with a REQUIRED positional prp_ id
// (Args: cobra.ExactArgs(1) — zero or more than one positional is a usage error before
// RunE, no request), a non-empty Short, and SilenceErrors/SilenceUsage so
// runProposalRespond owns its messages. It declares ONLY the --response flag — which
// lives only on `respond`, so passing it to another leaf is a cobra unknown-flag usage
// error for free (the structural guard). It reads the inherited persistent
// --base-url/--output flags, then delegates to the pure runProposalRespond. The seam is
// injected so tests drive a fake one; production passes productionSeam{}.
func newProposalRespondCommand(seam proposalSeam) *cobra.Command {
	var response string
	cmd := &cobra.Command{
		Use:   "respond <prp-id>",
		Short: "Record a consent-window response to a circulating proposal",
		Long: "respond records one member's consent-window response to a circulating governance " +
			"proposal — the consume/respond write of the write flow. The positional proposal id " +
			"names the proposal to respond to; --response carries the required consent value, one " +
			"of `no_objection` (willing to let the proposal pass) or `bring_to_meeting` (wants live " +
			"discussion — blocks auto-acceptance). The responding person is derived from the token " +
			"and is never sent. A missing or unsupported --response is refused before any request. " +
			"On success the recorded response is printed with its prr_ id and the parent proposal's " +
			"status — which reads `accepted` when this very response triggered auto-acceptance. The " +
			"write is Premium-gated; the command issues the request and surfaces the server's " +
			"response, never retrying a rejected or rate-limited recording.",
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
			outcome, oerr := runProposalRespond(proposalRespondConfig{
				seam:           seam,
				baseURL:        baseURL,
				baseURLPresent: cmd.Flags().Changed(apiclient.FlagBaseURL),
				outputFlag:     outputFlag,
				outputPresent:  cmd.Flags().Changed(output.FlagOutput),
				proposalID:     args[0],
				response:       response,
				reqCtx:         cmd.Context(),
				stdout:         cmd.OutOrStdout(),
				stderr:         cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	cmd.Flags().StringVar(&response, "response", "", "The consent value to record: one of "+strings.Join(supportedProposalResponseNames(), ", ")+" (required)")
	return cmd
}
