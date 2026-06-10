package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/render"
	"github.com/spf13/cobra"
)

// domainSeam supplies everything the single `domain` read needs from the outside,
// so runDomain is pure over injected values and every branch runs offline. It is
// the single-read shape (assemble + newClient + resolveFormat) — no walker and no
// sleep, because the single read issues exactly one Execute (unpaginated; 016/017
// are not on this path). productionSeam satisfies it unchanged (it carries the
// extra list methods harmlessly), and the existing test fakes drive it. It never
// reads ctx.Cred.Token — the token rides 007's AuthTransport in the client.
type domainSeam interface {
	assemble(baseURL string) apiclient.ConnectionContext
	newClient(ctx apiclient.ConnectionContext) (*apiclient.Client, error)
	resolveFormat(flagValue string) (output.OutputFormat, error)
}

// domainConfig carries everything runDomain needs, gathered by the command's
// RunE. Keeping runDomain a function of injected values makes the whole read —
// validate, assemble, build, send, render/classify — testable over a fake
// transport with no real network or ~/.glassfrogrc.
type domainConfig struct {
	seam       domainSeam
	baseURL    string   // inherited persistent --base-url (may be empty)
	outputFlag string   // inherited persistent --output (may be empty), resolved before any request
	id         string   // the required positional domain id (ExactArgs(1))
	include    []string // --include, validated against {policies}

	// The list-only walk/search flags are declared on `domain` only to reject
	// them: the single read is unpaginated and unsearchable, so passing any of
	// them is a usage error (interface). Presence, not value.
	querySet     bool
	firstPageSet bool
	perPageSet   bool

	reqCtx context.Context
	stdout io.Writer
	stderr io.Writer
}

// supportedDomainIncludes is the closed enum of --include values getDomain
// accepts — exactly {policies} (interface; spec). A value outside it is rejected
// fail-fast (the API would otherwise silently ignore it and return the domain
// WITHOUT the embed — plan ADR-4). It is the single read's OWN set, never shared
// with the role/subroles include sets the API would drop here.
var supportedDomainIncludes = map[string]bool{
	"policies": true,
}

// runDomain is the pure orchestration the `domain` leaf delegates to: resolve the
// output format, reject the list-only flags and validate --include against the
// closed {policies} set fail-fast before any assembly or request, assemble the
// connection and build the client, then issue ONE GET /domains/{id} into a
// DomainDocument (no walk, no If-None-Match — the single read is unpaginated and
// uncached). It adds no new Outcome/ExitCode and never reads the token.
func runDomain(cfg domainConfig) (Outcome, error) {
	// 1. Resolve the output format FIRST (020): a present-but-invalid selector
	//    fails fast as a usage error before any assembly or request.
	format, ferr := cfg.seam.resolveFormat(cfg.outputFlag)
	if ferr != nil {
		return reportFormatResolutionError(cfg.stderr, ferr)
	}

	// 2. Reject the list-only walk/search flags BEFORE any request (the single
	//    read is unpaginated/unsearchable — fail-fast usage error, pinned by a
	//    transport tripwire).
	if err := validateDomainFlags(cfg); err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}

	// 2b. Validate --include against the closed {policies} set BEFORE any request.
	//     The id is NOT validated locally (ADR-4): the API 404s an unknown id.
	if err := validateIncludeSet(cfg.include, supportedDomainIncludes); err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return UsageError, err
	}

	// 3. Resolve the connection and build the client. A base-URL error surfaces
	//    here (no doomed send); classify + report it.
	ctx := cfg.seam.assemble(cfg.baseURL)
	client, err := cfg.seam.newClient(ctx)
	if err != nil {
		return reportClientError(cfg.stderr, err)
	}

	return runDomainGet(cfg, client, format, cfg.id)
}

// runDomainGet reads a single domain by id (GET /domains/{id}). It sends one
// Execute (no walk) with ?include= built from the already-validated --include
// values (comma-joined, style:form explode:false), passing the id through
// unvalidated (ADR-4) so an unknown/malformed id surfaces as the API's 404/4xx
// via the shared classifier. A structured --output emits the raw {data: Domain}
// payload verbatim (018); the human path decodes the DomainDocument and renders
// the `domain` template over a DomainView that also carries the requested-include
// set, so the policies section is omitted when unrequested and shows an
// explicit-absence marker when requested-but-empty (ADR-2). The id is escaped as
// one path segment (the runRoleGet safeguard: a raw `/`/`..` must not
// redirect/traverse).
func runDomainGet(cfg domainConfig, exec executor, format output.OutputFormat, id string) (Outcome, error) {
	var q url.Values
	if len(cfg.include) > 0 {
		q = url.Values{"include": {strings.Join(cfg.include, ",")}}
	}
	req := apiclient.Request{Method: http.MethodGet, Path: "/domains/" + url.PathEscape(id), Query: q}

	if machineFmt, ok := format.MachineFormat(); ok {
		var raw json.RawMessage
		if _, err := exec.Execute(cfg.reqCtx, req, &raw); err != nil {
			return reportClientError(cfg.stderr, err)
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

	var doc glassfrog.DomainDocument
	if _, err := exec.Execute(cfg.reqCtx, req, &doc); err != nil {
		return reportClientError(cfg.stderr, err)
	}
	view := render.DomainView{Domain: doc.Data, Requested: includeSet(cfg.include)}
	text, rerr := renderFn(render.ResourceDomain, humanFormat(format), view)
	if rerr != nil {
		fmt.Fprintln(cfg.stderr, rerr.Error())
		return RuntimeError, rerr
	}
	fmt.Fprint(cfg.stdout, text)
	return Success, nil
}

// validateDomainFlags rejects the list-only flags fail-fast, before any request
// (the 011/013 validate-before-call shape, pinned by a transport tripwire): the
// single domain read is unpaginated and unsearchable, so --query/--first-page/
// --per-page are usage errors. The message names the misuse and the fix (the
// plural `domains` list owns paging and search).
func validateDomainFlags(cfg domainConfig) error {
	var offending []string
	if cfg.querySet {
		offending = append(offending, "--query")
	}
	if cfg.firstPageSet {
		offending = append(offending, "--first-page")
	}
	if cfg.perPageSet {
		offending = append(offending, "--per-page")
	}
	if len(offending) > 0 {
		return fmt.Errorf(
			"%s %s the domains list, not a single domain — remove %s (the single `domain` read is unpaginated and unsearchable; use `glassfrog domains <role-id>` to list and search)",
			joinFlags(offending), pluralVerb(len(offending)), pluralThem(len(offending)),
		)
	}
	return nil
}

// newDomainCommand builds the runnable `domain` leaf (plan ADR-1): a guard-ready
// cobra command with a REQUIRED positional domain id (Args: cobra.ExactArgs(1)),
// a non-empty Short cross-referencing the plural `domains` list, and
// SilenceErrors/SilenceUsage so runDomain owns its messages. It declares
// --include and — to reject them with a friendly message rather than a bare cobra
// "unknown flag" — the list-only --query/--first-page/--per-page flags it
// forbids. It reads the inherited persistent --base-url/--output flags, then
// delegates to the pure runDomain. The seam is injected so tests drive a fake
// one; production passes productionSeam{}.
func newDomainCommand(seam domainSeam) *cobra.Command {
	var (
		include   []string
		query     string
		firstPage bool
		perPage   int
	)
	cmd := &cobra.Command{
		Use:   "domain <id>",
		Short: "Read one domain by its id, optionally embedding its policies",
		Long: "domain reads a single area of control (domain) by its own id, optionally " +
			"embedding the policies scoped to it with --include policies. The single read is " +
			"unpaginated and unsearchable. To list the domains a role controls (and search " +
			"them), use the plural `domains <role-id>`.",
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
			outcome, oerr := runDomain(domainConfig{
				seam:       seam,
				baseURL:    baseURL,
				outputFlag: outputFlag,
				id:         args[0],
				include:    include,
				// Presence, not value: a list-only flag passed to the single read is a
				// usage error regardless of its value (Changed).
				querySet:     cmd.Flags().Changed("query"),
				firstPageSet: cmd.Flags().Changed("first-page"),
				perPageSet:   cmd.Flags().Changed("per-page"),
				reqCtx:       cmd.Context(),
				stdout:       cmd.OutOrStdout(),
				stderr:       cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	cmd.Flags().StringSliceVar(&include, "include", nil, "Related resources to embed (policies)")
	cmd.Flags().StringVarP(&query, "query", "q", "", "Not valid on the single domain read — use `domains <role-id> --query` to search a role's domains")
	cmd.Flags().BoolVar(&firstPage, "first-page", false, "Not valid on the single domain read (unpaginated) — use `domains <role-id>`")
	cmd.Flags().IntVar(&perPage, "per-page", 0, "Not valid on the single domain read (unpaginated) — use `domains <role-id>`")
	return cmd
}
