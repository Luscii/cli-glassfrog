package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/Luscii/cli-glassfrog/internal/grammar"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/render"
	"github.com/spf13/cobra"
)

// Agent-Facing Grammar Reference (077) — `glassfrog proposal grammar`.
//
// This is the CLI's first CLIENT-LESS command. Every other read assembles a
// connection, resolves a token and a base URL, builds a client, and sends a
// request; this one does none of it. The knowledge it serves is compiled into the
// binary (internal/grammar), so the command's whole job is: resolve the output
// selection, load the embedded structure, render it.
//
// That is not an optimization — it is the capability. The reference has to be
// consultable on any machine that has the CLI: before `auth login` has ever run,
// with a malformed credential VALUE present, and with no network. Those accords hold
// by construction here rather than by a check, because there is no code path that
// could read a credential or open a socket.
//
// The precise boundary, because it is easy to over-promise here: the credential
// lives in `.glassfrogrc`, which is also where the `output` setting lives, and
// output resolution fails loud on a file that does not parse at all (020/040 — no
// fall-through). So a garbage token cannot block this command, while an unparseable
// settings file fails it exactly as it fails every other command. Faking immunity
// by softening the shared format resolution here would make this the only command
// that tolerated a broken settings file, which is the worse surprise. The seam it takes is the OUTPUT
// selection slice only (selectionSeam — resolve the selection, read a user
// template's text), never the client-building one.
//
// It also never judges. No argument or flag of this command takes a change set:
// cobra rejects any positional, so one offered for checking fails as a usage error
// before any code of ours runs, and the failure carries no validity language. The
// server stays the only judge of a change set's validity.
//
// The inherited output-template source (`-o <file>` / `-o stdin`) is the one path
// that consumes caller bytes, and it is deliberately not an exception: it renders
// the caller's template and evaluates nothing, exactly as it does on every read.
// Refusing it here would make this the only command whose --output behaves
// differently, which the Conduct accord ("renders the way other reads do")
// forbids.

// proposalGrammarConfig carries everything runProposalGrammar needs, gathered by
// the leaf's RunE. There is no baseURL field and no request context: the command
// resolves neither, and a field for a value that is never resolved would invite a
// future reader to wire it up.
type proposalGrammarConfig struct {
	seam          selectionSeam
	outputFlag    string // inherited persistent --output (may be empty)
	outputPresent bool   // whether --output was supplied (cobra Changed()); the flag rung's presence (040 ADR-2)

	stdout io.Writer
	stderr io.Writer
}

// runProposalGrammar is the pure orchestration the `grammar` leaf delegates to.
// Three steps, no request:
//
//  1. Resolve the render target (020 widened by 035) — a present-but-invalid
//     selector, or a missing/unparseable user template, is a fail-fast usage error.
//     This is the ONLY configuration the command touches; token and base-URL
//     resolution are never invoked, not invoked-and-ignored, so a malformed
//     credential value cannot block the reference. An unparseable `.glassfrogrc`
//     still fails here, because that file carries the `output` setting too — see
//     the boundary note in the package comment above.
//  2. Load the embedded grammar. A decode failure is a corrupt build — the drift
//     guard makes a bad artifact unshippable — so it classifies as the CLI-internal
//     runtime fault (exit 1), never as an API or network category: no request
//     exists to have failed.
//  3. Dispatch on the resolved target. A machine format serializes the embedded
//     structure DIRECTLY (interface-cli § Consistency Notes: there is no server
//     envelope to mirror, so `json` is the structure itself — deliberately not
//     wrapped in `{data: …}`); a human format renders the `grammar` templates, or
//     the operator's own template, over the same structure.
//
// Every exit code that requires an exchange is unproducible here by construction:
// this command sends no request and creates nothing, so there is no response, 403,
// 429, transport error, ETag conflict, or accepted-but-invalid create to classify.
// That is codes 3–8 today (API, permission, rate-limit, network, stale-write,
// invalid-create), but the rule is the absent request path, not the list — an
// exchange-derived code added later is unproducible here for the same reason.
func runProposalGrammar(cfg proposalGrammarConfig) (Outcome, error) {
	rt, outcome, oerr, ok := resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.outputPresent, cfg.stderr)
	if !ok {
		return outcome, oerr
	}

	g, err := grammar.Load()
	if err != nil {
		fmt.Fprintln(cfg.stderr, err.Error())
		return RuntimeError, err
	}

	if machineFmt, ok := rt.format.MachineFormat(); ok {
		payload, merr := json.Marshal(g)
		if merr != nil {
			// Unreachable in practice: the value came from decoding JSON into
			// plain strings and slices. Surfaced rather than ignored so a future
			// field that cannot marshal fails loud instead of emitting nothing.
			fmt.Fprintln(cfg.stderr, merr.Error())
			return RuntimeError, merr
		}
		doc, rerr := output.RenderSuccess(machineFmt, payload)
		if rerr != nil {
			// Buffer-then-write: a render failure leaves stdout empty and maps to
			// RuntimeError(1), as it does on every read.
			fmt.Fprintln(cfg.stderr, rerr.Error())
			return RuntimeError, rerr
		}
		_, _ = cfg.stdout.Write(doc)
		return Success, nil
	}

	return writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceGrammar, rt.format, render.NewGrammarView(g))
}

// newProposalGrammarCommand builds the runnable `proposal grammar` leaf: a
// guard-ready cobra command with NO positional arguments (Args: cobra.NoArgs — any
// positional is a usage error before RunE, and that is the accord's refusal of a
// change set offered for judgment), no command-local flags, a non-empty Short, and
// SilenceErrors/SilenceUsage so the shared dispatch owns the message.
//
// It takes the OUTPUT slice of the seam (selectionSeam), not the proposal write
// seam — the narrowest type that expresses "this command cannot reach the network."
// The `proposal` group's full seam satisfies it structurally, so registration needs
// no second seam. The inherited `--base-url` is deliberately not even READ here: it
// parses and is inert, because reading it would be the first step toward resolving
// a connection this command must never have.
func newProposalGrammarCommand(seam selectionSeam) *cobra.Command {
	cmd := &cobra.Command{
		Use: "grammar",
		// The Short carries all four elements the accord names for it (interface-cli
		// § Surface > The command): the change-set grammar, consulted BEFORE
		// assembling, the contract-published types with their placement, and the
		// verified empirical observations. The two provenance standings belong here
		// rather than only in Long, because `glassfrog proposal --help` shows the
		// Short alone — an agent scanning the subcommand list would otherwise get no
		// signal that the reference mixes two standings, which is this feature's
		// second user scenario.
		Short: "Show the change-set grammar before assembling: contract-published types and placement, plus verified empirical observations",
		Long: "grammar prints the change-set grammar for a proposal's changes[] array — consult it " +
			"BEFORE assembling a change set. It renders every change type the published Glassfrog " +
			"API v5 contract enumerates with its placement (top-level, or nested-only inside a " +
			"CreateRole/UpdateRole part), the nesting rule itself, and the verified empirical " +
			"observations the contract does not carry, each with the symptom that getting the shape " +
			"wrong produces. Every rendered shape carries its provenance, so a contract-published " +
			"shape and a verified observation are never confused.\n\n" +
			"The command INFORMS and never validates. It takes no arguments and accepts no change " +
			"set: there is nothing to submit to it, and it renders no verdict — the server remains " +
			"the only judge of whether a change set is valid. It makes no API request, resolves no " +
			"credential, and needs no network, so it works on any machine that has the CLI, before " +
			"`glassfrog auth login` has ever run.",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			outputFlag, err := cmd.Flags().GetString(output.FlagOutput)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "could not read the --output flag: %v\n", err)
				return err
			}
			outcome, oerr := runProposalGrammar(proposalGrammarConfig{
				seam:          seam,
				outputFlag:    outputFlag,
				outputPresent: cmd.Flags().Changed(output.FlagOutput),
				stdout:        cmd.OutOrStdout(),
				stderr:        cmd.ErrOrStderr(),
			})
			return outcomeToDispatchError(outcome, oerr)
		},
	}
	return cmd
}
