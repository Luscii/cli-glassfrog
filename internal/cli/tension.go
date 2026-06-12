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
	"sort"
	"strings"
	"time"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/render"
	"github.com/spf13/cobra"
)

// supportedMeetingTypes is the spec's tension meeting_type set — the meeting_type
// enum on the Tension schema / TensionInput (spec/glassfrog-api-v5.yaml: tactical,
// governance). It is the single source of truth for --meeting-type validation.
// Adding a value is a one-line change tracking the spec enum. It deliberately does
// not include null/empty: an absent --meeting-type is the "no constraint" case the
// validator accepts and the marshaller omits.
var supportedMeetingTypes = map[string]bool{
	"tactical":   true,
	"governance": true,
}

// validateMeetingType rejects a non-empty --meeting-type value outside the spec's
// set, returning a usage error NAMING the unsupported value and listing the
// supported set — before any context assembly or request (the validateStatus
// fail-fast discipline, plan ADR-3). An empty value (the flag absent) is valid: no
// meeting-type constraint, the field is omitted from the body. Pure — no network,
// no filesystem — so it runs ahead of any I/O and a tripwire transport can assert
// nothing was sent on rejection. A sibling validator of validateStatus, not a
// second copy of any set.
func validateMeetingType(value string) error {
	if value == "" {
		return nil
	}
	if supportedMeetingTypes[value] {
		return nil
	}
	return fmt.Errorf(
		"unsupported --meeting-type value %q — supported: %s",
		value, strings.Join(supportedMeetingTypeNames(), ", "),
	)
}

// supportedMeetingTypeNames lists the supported meeting types in stable (sorted)
// order for the usage message, so the same input always yields the same
// deterministic text (the supportedStatusNames shape).
func supportedMeetingTypeNames() []string {
	names := make([]string, 0, len(supportedMeetingTypes))
	for name := range supportedMeetingTypes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// tensionSeam supplies everything the tension write needs from the outside, so
// runTensionCreate is pure over injected values and every branch runs offline. It
// is the projectsSeam shape (assemble + newClient + sleep + resolveSelection +
// readTemplateSource — 020's format resolution widened by User-Defined Template
// Output (035); the write needs no paging), so productionSeam satisfies it unchanged
// and the existing test fakes drive it. It never reads ctx.Cred.Token — the token
// rides 007's AuthTransport inside the client.
type tensionSeam interface {
	assemble(baseURL string, baseURLPresent bool) apiclient.ConnectionContext
	newClient(ctx apiclient.ConnectionContext) (*apiclient.Client, error)
	sleep() func(time.Duration)
	resolveSelection(flagValue string) (output.Selection, error)
	readTemplateSource(ref output.TemplateRef) (string, error)
}

// tensionCreateConfig carries everything runTensionCreate needs, gathered by the
// command's RunE. Keeping the write a function of injected values makes the whole
// flow — resolve, validate, assemble, build, send, render/classify — testable over
// a fake transport with no real network or ~/.glassfrogrc. labelSet/meetingTypeSet
// are cmd.Flags().Changed(...) — a field is sent only when set AND non-empty.
type tensionCreateConfig struct {
	seam           tensionSeam
	baseURL        string // inherited persistent --base-url (may be empty)
	baseURLPresent bool   // whether --base-url was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)
	outputFlag     string // inherited persistent --output (may be empty), resolved before any request
	id             string // the required positional sensing role id (ExactArgs(1))

	body           string // --body, required non-empty (validated before any request)
	label          string // --label, optional free text
	labelSet       bool   // whether --label was provided (Changed); sent only when set AND non-empty
	meetingType    string // --meeting-type, validated against the closed set before any request
	meetingTypeSet bool   // whether --meeting-type was provided (Changed)

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// runTensionCreate is the pure orchestration the `create` leaf delegates to — the
// CLI's first write. It resolves the output format (020) FIRST, then validates the
// two pure inputs fail-fast (a missing/whitespace-only --body and an unsupported
// --meeting-type are each a usage error with NO request issued), in the same
// output-first order as the sibling reads so error precedence is consistent. Then
// it assembles the connection, builds the retrying executor, marshals the
// {tension:{…}} body, and sends ONE Execute to POST /roles/{id}/tensions with
// Content-Type: application/json (T001/ADR-1). A structured --output emits the raw
// {data: Tension} payload verbatim (018); the human path decodes Document[Tension]
// and renders the `tension` template (or a selected user template, 035) over a
// TensionView (mirroring runProjectGet). The role id is escaped as one path segment
// but passed through unvalidated (ADR-3) so an unknown/malformed id surfaces as the
// API's 404/422 via the shared classifier. It adds no new Outcome/ExitCode and never
// reads the token.
func runTensionCreate(cfg tensionCreateConfig) (Outcome, error) {
	// 1. Resolve the render target FIRST (020 widened by 035): a present-but-invalid
	//    selector — or, for a user template, a missing/unparseable source or empty
	//    stdin — fails fast as a usage error before any assembly or request. Resolving
	//    --output ahead of the input checks keeps error precedence consistent with the
	//    reads — an invalid --output is reported even when --body/--meeting-type is bad.
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	// 2. Validate --body non-empty BEFORE any assembly or request (fail-fast usage
	//    error, no wasted call — pinned by a tripwire transport). A bodyless tension
	//    is meaningless, so reject it client-side rather than spending a request on
	//    the API's 422 (plan ADR-3). Empty means empty after TrimSpace.
	if strings.TrimSpace(cfg.body) == "" {
		err := errors.New("--body is required and must not be empty")
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}

	// 3. Validate --meeting-type against the closed set BEFORE any request (the
	//    validateStatus fail-fast shape; ADR-3). Both checks are pure and
	//    pre-assembly, so the no-request-on-rejection tripwire holds regardless of
	//    their relative order.
	if err := validateMeetingType(cfg.meetingType); err != nil {
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

	// 5. Marshal {tension:{body[,label][,meeting_type]}}: label/meeting-type ride
	//    only when the flag was provided (Changed) AND non-empty — NewTensionInput's
	//    omitempty drops a still-empty value, so `--label ""` and an omitted flag both
	//    send no field. status and sensed_by are never sent (server owns them).
	label := ""
	if cfg.labelSet {
		label = cfg.label
	}
	meetingType := ""
	if cfg.meetingTypeSet {
		meetingType = cfg.meetingType
	}
	bodyBytes, merr := json.Marshal(glassfrog.NewTensionInput(cfg.body, label, meetingType))
	if merr != nil {
		// Marshalling plain strings cannot fail in practice; surface it as a runtime
		// defect rather than sending a malformed body.
		fmt.Fprintln(cfg.stderr, merr.Error())
		return RuntimeError, merr
	}

	// Escape the id as a single path segment: passed through unvalidated (ADR-3),
	// but a raw `/` or `..` must not redirect the request or traverse the path.
	req := apiclient.Request{
		Method:      http.MethodPost,
		Path:        "/roles/" + url.PathEscape(cfg.id) + "/tensions",
		Body:        bytes.NewReader(bodyBytes),
		ContentType: "application/json",
	}

	// 6. Dispatch on the resolved render target. A POST is never auto-retried on 429
	//    (017's isSafeMethod gate), so a rate-limited capture surfaces once and cannot
	//    double-create (silent conformance to §133). On any failure route through the
	//    shared reportFailure (032). A user template (035) is a human render, so the
	//    structured branch is taken only for a built-in json/yaml selection (rt.tmpl
	//    is nil then, by construction in resolveRenderTarget).
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
	// writeHuman renders through the selected user template (035) or the built-in
	// `tension` template (019), buffer-then-write.
	return writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceTension, rt.format, view)
}

// newTensionCommand assembles the `tension` command group and its leaves,
// registered through the guard (group-has-children, non-empty Short, no action).
// The group is built with its children attached BEFORE being returned for
// registration under root, so the guard's ">=1 child" rule holds at attach time
// (the auth/auth login shape, plan ADR-2). The `tension` namespace parents the
// write verb `create` (042) and the read verbs `list`/`get` (043), so the group
// Short names the whole surface rather than just capture (a write-only Short would
// make `glassfrog tension --help` misleading). The seam is injected so tests drive
// a fake one; production passes productionSeam{} from Assemble.
func newTensionCommand(seam tensionSeam) *cobra.Command {
	group := &cobra.Command{
		Use:   "tension",
		Short: "Work with tensions — capture one, or list and read a role's tensions",
	}
	MustRegister(group, newTensionCreateCommand(seam))
	MustRegister(group, newTensionListCommand(seam))
	MustRegister(group, newTensionGetCommand(seam))
	return group
}

// newTensionCreateCommand builds the runnable `tension create <role-id>` leaf
// (ADR-2): a guard-ready cobra command with a REQUIRED positional sensing role id
// (Args: cobra.ExactArgs(1)), a non-empty Short, and SilenceErrors/SilenceUsage so
// runTensionCreate owns its messages. It declares the write flags (--body,
// --label, --meeting-type) — these live only on `create`, so passing one to a
// future `tension get` would be a cobra unknown-flag usage error (the structural
// guard). It reads the inherited persistent --base-url/--output flags, then
// delegates to the pure runTensionCreate. The seam is injected so tests drive a
// fake one; production passes productionSeam{}.
func newTensionCreateCommand(seam tensionSeam) *cobra.Command {
	var (
		body        string
		label       string
		meetingType string
	)
	cmd := &cobra.Command{
		Use:   "create <role-id>",
		Short: "Capture a tension against a sensing role",
		Long: "create records a tension sensed by a role — the CLI's first write and the " +
			"entry point to a governance proposal. The positional role id is the sensing " +
			"role (the token's person is the sensing person, derived by the API); --body " +
			"is the required tension text. Attach an optional --label and a validated " +
			"--meeting-type (tactical|governance). The created tension is printed with its " +
			"ten_ id so a later proposal can reference it. A missing or blank body, or an " +
			"unsupported meeting-type, is refused before any request.",
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
			outcome, oerr := runTensionCreate(tensionCreateConfig{
				seam:           seam,
				baseURL:        baseURL,
				baseURLPresent: cmd.Flags().Changed(apiclient.FlagBaseURL),
				outputFlag:     outputFlag,
				id:             args[0],
				body:           body,
				label:          label,
				labelSet:       cmd.Flags().Changed("label"),
				meetingType:    meetingType,
				meetingTypeSet: cmd.Flags().Changed("meeting-type"),
				reqCtx:         cmd.Context(),
				stdout:         cmd.OutOrStdout(),
				stderr:         cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	cmd.Flags().StringVar(&body, "body", "", "The tension text (required; rejected if empty or whitespace-only)")
	cmd.Flags().StringVar(&label, "label", "", "Optional short label for the tension (omitted when empty)")
	cmd.Flags().StringVar(&meetingType, "meeting-type", "", "Optional routing hint (one of: "+strings.Join(supportedMeetingTypeNames(), ", ")+")")
	return cmd
}
