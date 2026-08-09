package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
		// Lift the created id by decoding a COPY of the retained bytes (074
		// ADR-6): the raw bytes stay untouched for the verbatim emission — the
		// machine path never re-marshals a server document (018). An undecodable
		// create body was already classified above (Execute's decode validates
		// the JSON), so a failed unmarshal here means valid JSON that is not a
		// {data: Proposal} document: no id can be lifted, the helper
		// short-circuits with the id-undeterminable reason, and the create's own
		// document is emitted unchanged.
		var created glassfrog.Document[glassfrog.Proposal]
		id := ""
		if err := json.Unmarshal(raw, &created); err == nil {
			id = created.Data.ID
		}
		// Post-create read-back (074 ADR-1/ADR-2/ADR-5): when it answered, the
		// EMITTED document is the read-back's — the later and richer of the two
		// server documents, and the only one verified to carry the verdict fields
		// inline. When it did not, the create's document is emitted exactly as
		// today, and the structured advisory below states the verdict was
		// unobtainable and why. Either way stdout carries ONE verbatim server
		// document, no composed envelope, no CLI-added keys.
		_, readBackRaw, reason := readBackProposalVerdict(cfg.reqCtx, exec, id)
		emit := raw
		if reason == "" && readBackRaw != nil {
			emit = readBackRaw
		}
		doc, rerr := output.RenderSuccess(machineFmt, emit)
		if rerr != nil {
			// Buffer-then-write: a render failure leaves stdout empty and maps to
			// RuntimeError(1). The error is token-free (018 contract).
			fmt.Fprintln(cfg.stderr, rerr.Error())
			return RuntimeError, rerr
		}
		_, _ = cfg.stdout.Write(doc)
		writeMachineVerdictAdvisory(cfg.stderr, machineFmt, newVerdictSource(id, reason))
		return Success, nil
	}

	var doc glassfrog.Document[glassfrog.Proposal]
	if _, err := exec.Execute(cfg.reqCtx, req, &doc); err != nil {
		return reportFailure(cfg.stdout, cfg.stderr, rt.format, err)
	}

	// 7. Post-create read-back (074 ADR-1/ADR-2): one GET /proposals/{id} for the id
	//    the create returned, through the SAME retrying executor (a GET is
	//    retry-eligible; the POST above never was). Its failure is never the
	//    command's failure — every path out of the helper leads to reporting the
	//    created proposal with Success, so the prp_ id is never withheld. When the
	//    read-back answered, ITS document renders the body: available_transitions
	//    and status are verdict dimensions read at the same instant as the flag,
	//    and the detail read is the surface verified to carry the verdict fields.
	readBack, _, reason := readBackProposalVerdict(cfg.reqCtx, exec, doc.Data.ID)
	verdict := render.NewProposalVerdict(readBack.Valid, readBack.ValidationAlerts, reason, doc.Data.ID)
	proposal := doc.Data
	if reason == "" {
		proposal = readBack
	}
	view := render.ProposalCreatedView{
		ProposalView: render.ProposalView{Proposal: proposal},
		Verdict:      verdict,
	}
	// writeHuman renders through the selected user template (035) or the built-in
	// `proposal-created` template (074 ADR-4 — the shared proposal body plus the
	// verdict block), buffer-then-write. Every field path a pre-074 user template
	// referenced still resolves: ProposalCreatedView embeds ProposalView.
	outcome, oerr = writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceProposalCreated, rt.format, view)
	if outcome != Success {
		return outcome, oerr
	}
	// The stderr advisory: the verdict's provenance, or the reason it is
	// unavailable plus the remedy (045's disambiguating-advisory precedent). In a
	// human format it is the prose line derived from the same value the machine
	// formats serialize structurally (T006), so the two renderings cannot disagree.
	fmt.Fprintln(cfg.stderr, newVerdictSource(doc.Data.ID, reason).proseLine())
	return Success, nil
}

// verdictSource is the CLI-owned advisory the create writes to stderr (074). It is
// rendered as prose in a human format and structurally in a machine format,
// following the format-aware diagnostic convention (032). It is a CLI shape, not a
// server shape — which is what keeps 018's verbatim contract untouched: the server's
// document on stdout is never reshaped, and this never rides stdout.
//
// ReadBack answers "did the CLI manage to ask?" — the one question the emitted
// document cannot answer, and therefore the field that makes all four verdict states
// machine-distinguishable. ProposalID is omitted when no id could be determined,
// Reason is omitted when the read-back answered, and Remedy is omitted when there is
// none to name: an absent key means "not applicable", never an empty string.
type verdictSource struct {
	ReadBack   bool   `json:"read_back"`
	ProposalID string `json:"proposal_id,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Remedy     string `json:"remedy,omitempty"`
}

// newVerdictSource builds the advisory from the read-back's outcome. It is the single
// source of the advisory's content in BOTH renderings — the prose line is derived
// from the same value the structured form encodes, so the two can never disagree
// about whether the CLI asked. An empty id with a reason is the id-undeterminable
// case: no ProposalID and no Remedy, because without an id there is nothing to
// re-read and naming a command the caller cannot run would fabricate a next step.
func newVerdictSource(id string, unavailableReason string) verdictSource {
	if unavailableReason == "" {
		return verdictSource{ReadBack: true, ProposalID: id}
	}
	v := verdictSource{Reason: unavailableReason}
	if id != "" {
		v.ProposalID = id
		v.Remedy = "glassfrog proposal get " + id
	}
	return v
}

// proseLine renders the advisory for the human formats (interface-cli § stderr):
// one line naming the read that produced the verdict, or the cause and the remedy
// when it could not be obtained. The no-id line names no remedy — without an id
// there is nothing to re-read, and that is the honest thing to say. Derived from
// the same fields the structured form serializes, so the two cannot disagree.
func (v verdictSource) proseLine() string {
	if v.ReadBack {
		return fmt.Sprintf("the validity verdict was read back from proposal %s after the create", v.ProposalID)
	}
	if v.ProposalID == "" {
		return "could not determine the created proposal's id from the create response, so no validity verdict was obtained; the create response is reported above"
	}
	return fmt.Sprintf("could not read proposal %s back to obtain its validity verdict: %s; the proposal was created — run %q to read its verdict", v.ProposalID, v.Reason, v.Remedy)
}

// writeMachineVerdictAdvisory writes the verdict advisory to stderr in the
// SELECTED machine format (074 ADR-5, amended): {"verdict_source": {…}} rendered
// through the same structured-render helper the success documents use, so an
// agent parses the advisory the same way it parses stdout (032's format-aware
// diagnostic principle). It rides stderr because stdout is occupied by the
// server's own verbatim document — a CLI-owned diagnostic may have a CLI-owned
// shape, and that shape never touches the verbatim stream (018). Best-effort by
// design: the advisory must never become the command's failure (ADR-2), and its
// payload is a locally-marshalled struct of plain strings, so the error paths
// are unreachable in practice.
func writeMachineVerdictAdvisory(stderr io.Writer, f output.Format, v verdictSource) {
	payload, err := json.Marshal(struct {
		VerdictSource verdictSource `json:"verdict_source"`
	}{v})
	if err != nil {
		return
	}
	doc, err := output.RenderSuccess(f, payload)
	if err != nil {
		return
	}
	_, _ = stderr.Write(doc)
}

// readBackProposalVerdict performs the post-create read-back (074): ONE
// GET /proposals/{id} through the supplied executor, decoding the server's
// verdict. Its failures are NEVER the command's failures — it returns a
// human-readable reason instead of an error, so a failed read-back can never
// withhold the created proposal's id or produce a non-zero exit (ADR-2). The raw
// bytes are returned so the machine path can emit the read-back's own document
// verbatim; they are nil when no read-back produced a body the CLI could read —
// which includes a 2xx body that decodes cleanly but carries no proposal, or a
// different one (see the id guard below).
// An empty id short-circuits with the id-undeterminable reason and issues NO
// request — a path fabricated from an empty id would 404 for nothing (ADR-6).
// The path is built exactly as `proposal get` builds it: url.PathEscape keeps a
// malformed id one opaque segment.
//
// Returns (proposal, raw, reason): reason is empty when the read-back answered.
func readBackProposalVerdict(ctx context.Context, exec executor, id string) (glassfrog.Proposal, json.RawMessage, string) {
	if id == "" {
		return glassfrog.Proposal{}, nil, "the created proposal's id could not be determined"
	}
	req := apiclient.Request{Method: http.MethodGet, Path: "/proposals/" + url.PathEscape(id)}
	var raw json.RawMessage
	if _, err := exec.Execute(ctx, req, &raw); err != nil {
		return glassfrog.Proposal{}, nil, readBackVerdictReason(err)
	}
	var doc glassfrog.Document[glassfrog.Proposal]
	if err := json.Unmarshal(raw, &doc); err != nil {
		return glassfrog.Proposal{}, nil, "the read-back response could not be read"
	}
	// A clean unmarshal is not yet an answer. A 200 carrying `{}`, `{"data":{}}`,
	// or any document without the proposal decodes without error into a ZERO
	// Proposal — and both call sites treat an empty reason as "the read-back
	// answered", so that zero value would replace the created proposal: the
	// human body would render an empty id, and the machine path would emit the
	// empty document in place of the create's. That withholds the created prp_
	// id, which the spec forbids without qualification. Requiring the returned
	// id to be the one asked for also rejects a document for a DIFFERENT
	// proposal, whose verdict would otherwise be reported as this create's.
	// Rejecting is the safe direction: a false reject still reports the created
	// proposal with a reason, while accepting a mismatch loses the handle.
	if doc.Data.ID != id {
		return glassfrog.Proposal{}, nil, "the read-back response could not be read"
	}
	return doc.Data, raw, ""
}

// readBackVerdictReason maps a read-back exchange error onto the verdict-
// unavailable reason vocabulary (interface-cli § "The read-back's failures never
// reach an exit code"). Every failure family gets a distinct reason; none routes
// through the shared failure envelope, because the read-back's failure is not
// the command's failure (ADR-2). The 429 arm names the exhausted request budget
// — the executor has already retried a safe GET, so a surviving 429 means the
// per-organization budget is spent, and the operator's move is to re-read later,
// never to re-create. Text is response-side only, never the token.
func readBackVerdictReason(err error) string {
	var responseErr *apiclient.ResponseError
	if errors.As(err, &responseErr) {
		if responseErr.StatusCode == http.StatusTooManyRequests {
			return "the read-back was rate limited (the request budget was exhausted)"
		}
		var problemErr *apiclient.ProblemError
		if errors.As(refineClientError(err), &problemErr) {
			return "the read-back was refused (" + problemCause(problemErr) + ")"
		}
		return fmt.Sprintf("the read-back was refused (status %d)", responseErr.StatusCode)
	}
	var decodeErr *apiclient.DecodeError
	if errors.As(err, &decodeErr) {
		return "the read-back response could not be read"
	}
	// A wire-level failure (*TransportError), or anything unrecognized: the
	// proposal could not be read back at all, with the underlying cause named.
	return "the proposal could not be read back (" + err.Error() + ")"
}

// newProposalCommand assembles the `proposal` command group and its leaves, registered
// through the guard (group-has-children, non-empty Short, no action). The group is
// built with its children attached BEFORE being returned for registration under root,
// so the guard's ">=1 child" rule holds at attach time (the tension/auth shape, plan
// ADR-1). The `proposal` namespace parents the write `create` (055), the `propose`
// transition (057), the `respond` consume/respond write (058), and the reads `list` /
// `get` (056), and the `withdraw` transition (059). The group, the
// glassfrog.Proposal model, and the
// singular `proposal` render key are SHARED across the proposal family under
// first-to-land-creates: 055 created the group here; siblings (056 reads, 057 propose)
// attach their leaves to the existing group and grow the shared model/render rather than
// duplicating them. The seam is injected so tests drive a fake one; production passes
// productionSeam{} from Assemble. All leaves share the one proposalSeam — the reads touch
// only the embedded tensionSeam half.
func newProposalCommand(seam proposalSeam) *cobra.Command {
	group := &cobra.Command{
		Use:   "proposal",
		Short: "Work with proposals — create, advance, withdraw, list, and read governance proposals",
	}
	MustRegister(group, newProposalCreateCommand(seam))
	MustRegister(group, newProposalProposeCommand(seam))
	MustRegister(group, newProposalWithdrawCommand(seam))
	MustRegister(group, newProposalRespondCommand(seam))
	MustRegister(group, newProposalListCommand(seam))
	MustRegister(group, newProposalGetCommand(seam))
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
