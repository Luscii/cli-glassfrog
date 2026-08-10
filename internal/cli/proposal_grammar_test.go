package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/build"
	"github.com/Luscii/cli-glassfrog/internal/grammar"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/rcfile"
	"sigs.k8s.io/yaml"
)

// Unit coverage for `proposal grammar` (077 T005) — the acceptance criteria the
// feature file does not carry as scenarios: the flag conduct, the exit-code
// envelope, the user-template path, and the help text's honesty.

// grammarOutputSeam implements ONLY the output slice of the seam. Driving
// runProposalGrammar over it is a compile-time proof of the client-less accord:
// this dependency exposes no way to assemble a connection, build a client, or
// resolve a token, so no future edit inside the command can start doing so without
// widening the seam and being noticed in review.
type grammarOutputSeam struct {
	selection    output.Selection
	selectionSet bool
	selErr       error
	tmplText     map[string]string

	selectionCalls int
}

func (s *grammarOutputSeam) resolveSelection(_ string, _ bool) (output.Selection, error) {
	s.selectionCalls++
	if s.selErr != nil {
		return output.Selection{Format: output.DefaultFormat}, s.selErr
	}
	if !s.selectionSet {
		return output.Selection{Format: output.DefaultFormat}, nil
	}
	return s.selection, nil
}

func (s *grammarOutputSeam) readTemplateSource(ref output.TemplateRef) (string, error) {
	return readTemplateSourceFrom(ref, func(path string) ([]byte, error) {
		text, ok := s.tmplText[path]
		if !ok {
			return nil, os.ErrNotExist
		}
		return []byte(text), nil
	}, true, nil)
}

// runGrammarOver drives the pure runProposalGrammar over a seam, returning the
// outcome and captured streams.
func runGrammarOver(t *testing.T, seam selectionSeam, outputFlag string, outputPresent bool) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	outcome, _ := runProposalGrammar(proposalGrammarConfig{
		seam:          seam,
		outputFlag:    outputFlag,
		outputPresent: outputPresent,
		stdout:        &out,
		stderr:        &errb,
	})
	return outcome, out.String(), errb.String()
}

// runGrammarCommand dispatches an invocation through a real root with the real
// `proposal` group attached, so cobra's argument and flag validation are the ones
// under test. The transport is a tripwire: a request would answer 500 and the
// assertion below would notice.
func runGrammarCommand(t *testing.T, args ...string) (Outcome, int, string, string, *fakeProposalSeam) {
	t.Helper()
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{
		ctx:       validMeContext(),
		transport: &cannedTransport{status: 500, body: `{"detail":"never"}`},
	}}
	root := NewRootCommand()
	MustRegister(root, newProposalCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, args)
	return outcome, ExitCode(outcome), out.String(), errb.String(), seam
}

func TestProposalGrammar_SucceedsOverASeamThatCannotReachTheNetwork(t *testing.T) {
	seam := &grammarOutputSeam{}
	outcome, stdout, stderr := runGrammarOver(t, seam, "", false)
	if outcome != Success {
		t.Fatalf("outcome %v, want Success; stderr: %s", outcome, stderr)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Fatal("the reference rendered nothing")
	}
	if seam.selectionCalls != 1 {
		t.Errorf("the output selection must be resolved exactly once; got %d", seam.selectionCalls)
	}
}

// TestProposalGrammar_AMalformedTokenValueDoesNotBlockTheRead is the accord's
// credential-free conduct, exercised over the REAL settings-file walk: a
// .glassfrogrc whose token value is garbage resolves cleanly for output purposes,
// because the command never asks about the token.
func TestProposalGrammar_AMalformedTokenValueDoesNotBlockTheRead(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, rcfile.FileName), []byte("glassfrog_token = %%% not a token %%%\n"), 0o600); err != nil {
		t.Fatalf("seeding the settings file: %v", err)
	}
	sel, err := output.ResolveSelectionFromOS("", false, dir, dir)
	if err != nil {
		t.Fatalf("a garbage token value must not break output resolution: %v", err)
	}
	outcome, stdout, stderr := runGrammarOver(t, &grammarOutputSeam{selection: sel, selectionSet: true}, "", false)
	if outcome != Success {
		t.Fatalf("outcome %v, want Success; stderr: %s", outcome, stderr)
	}
	if !strings.Contains(stdout, "Change-set grammar") {
		t.Errorf("the reference did not render:\n%s", stdout)
	}
}

// TestProposalGrammar_AnUnparseableSettingsFileFailsAsEveryCommandDoes records a
// real boundary of the accord's "works with a malformed credential file" claim.
// The credential lives in .glassfrogrc — the SAME file the --output setting lives
// in — so a file that does not parse at all fails output resolution, exactly as it
// does for every other command (020/040 fail loud rather than fall through). What
// the accord guarantees is that no CREDENTIAL is ever consulted; it cannot
// guarantee immunity to an unreadable settings file, and this command must not
// deviate from the shared conduct to fake it. Pinned here so the boundary is a
// recorded decision rather than a surprise.
func TestProposalGrammar_AnUnparseableSettingsFileFailsAsEveryCommandDoes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, rcfile.FileName), []byte("this line carries no equals sign\n"), 0o600); err != nil {
		t.Fatalf("seeding the settings file: %v", err)
	}
	_, err := output.ResolveSelectionFromOS("", false, dir, dir)
	if err == nil {
		t.Fatal("an unparseable settings file must fail output resolution loudly")
	}
	outcome, stdout, stderr := runGrammarOver(t, &grammarOutputSeam{selErr: err}, "", false)
	if outcome != UsageError {
		t.Fatalf("outcome %v, want UsageError — the shared format-resolution conduct", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("nothing must reach stdout on a resolution failure; got %q", stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("the resolution failure said nothing on stderr")
	}
}

// TestProposalGrammar_RejectsEveryPositionalAsUsage: there is no input path, so a
// change set offered for checking is refused before any code of ours runs — and the
// refusal carries no verdict on the change set.
func TestProposalGrammar_RejectsEveryPositionalAsUsage(t *testing.T) {
	for _, args := range [][]string{
		{"proposal", "grammar", "changes.json"},
		{"proposal", "grammar", `[{"type":"CreatePolicy"}]`},
		{"proposal", "grammar", "one", "two"},
		{"proposal", "grammar", "--output", "json", "changes.json"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			outcome, code, stdout, stderr, seam := runGrammarCommand(t, args...)
			if outcome != UsageError || code != 2 {
				t.Fatalf("outcome %v (exit %d), want UsageError (exit 2); stderr: %s", outcome, code, stderr)
			}
			if strings.TrimSpace(stdout) != "" {
				t.Errorf("a refused invocation must print no reference; got %q", stdout)
			}
			if strings.TrimSpace(stderr) == "" {
				t.Error("the usage failure said nothing on stderr")
			}
			for _, verdict := range []string{"invalid", "valid", "malformed", "rejected", "schema"} {
				if strings.Contains(strings.ToLower(stdout+stderr), verdict) {
					t.Errorf("the refusal uses the validity vocabulary %q, which reads as a verdict: %s", verdict, stderr)
				}
			}
			if seam.assembleCalled || seam.newClientCalled {
				t.Error("a refused invocation must not assemble a connection or build a client")
			}
		})
	}
}

// TestProposalGrammar_RejectsFlagsItDoesNotOwn: the command declares no local
// flag, so a sibling's flag is a cobra unknown-flag usage error for free.
func TestProposalGrammar_RejectsFlagsItDoesNotOwn(t *testing.T) {
	for _, args := range [][]string{
		{"proposal", "grammar", "--changes", "[]"},
		{"proposal", "grammar", "--status", "draft"},
		{"proposal", "grammar", "--per-page", "10"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			outcome, code, _, stderr, _ := runGrammarCommand(t, args...)
			if outcome != UsageError || code != 2 {
				t.Fatalf("outcome %v (exit %d), want UsageError (exit 2); stderr: %s", outcome, code, stderr)
			}
		})
	}
}

// TestProposalGrammar_BaseURLParsesAndIsInert: the flag is inherited and accepted,
// and it changes nothing — a base URL only matters to a request, and there is none.
func TestProposalGrammar_BaseURLParsesAndIsInert(t *testing.T) {
	_, code, plain, _, _ := runGrammarCommand(t, "proposal", "grammar", "--output", "json")
	if code != 0 {
		t.Fatalf("the plain run exited %d, want 0", code)
	}
	outcome, code, withFlag, stderr, seam := runGrammarCommand(t, "proposal", "grammar", "--base-url", "https://example.invalid", "--output", "json")
	if outcome != Success || code != 0 {
		t.Fatalf("outcome %v (exit %d), want Success; stderr: %s", outcome, code, stderr)
	}
	if withFlag != plain {
		t.Errorf("--base-url changed the output:\nwithout: %s\nwith:    %s", plain, withFlag)
	}
	if seam.assembleCalled {
		t.Error("--base-url must never be resolved — the command assembles no connection")
	}
}

// TestProposalGrammar_StructuredOutputIsTheEmbeddedStructureItself pins the
// deliberate deviation recorded in the accord's Consistency Notes: there is no
// server envelope to mirror, so `json` is the structure directly — never wrapped in
// `{data: …}`.
func TestProposalGrammar_StructuredOutputIsTheEmbeddedStructureItself(t *testing.T) {
	outcome, code, stdout, stderr, _ := runGrammarCommand(t, "proposal", "grammar", "--output", "json")
	if outcome != Success || code != 0 {
		t.Fatalf("outcome %v (exit %d); stderr: %s", outcome, code, stderr)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &top); err != nil {
		t.Fatalf("the structured output is not a JSON object: %v\n%s", err, stdout)
	}
	if len(top) != 2 {
		t.Fatalf("the document must carry exactly change_types and facts; got %d keys: %v", len(top), keysOf(top))
	}
	for _, key := range []string{"change_types", "facts"} {
		if _, ok := top[key]; !ok {
			t.Fatalf("the document omits %q: %v", key, keysOf(top))
		}
	}
	if _, wrapped := top["data"]; wrapped {
		t.Error("the document is wrapped in a data envelope; there is no server response to mirror")
	}

	// It is the embedded structure, value for value.
	want, err := grammar.Load()
	if err != nil {
		t.Fatalf("loading the embedded grammar: %v", err)
	}
	var got grammar.Grammar
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decoding the document: %v", err)
	}
	wantJSON, _ := json.Marshal(want)
	gotJSON, _ := json.Marshal(got)
	if string(wantJSON) != string(gotJSON) {
		t.Errorf("the document is not the embedded structure:\nwant %s\ngot  %s", wantJSON, gotJSON)
	}
}

// TestProposalGrammar_YAMLIsTheSameDocument: yaml is the same document in another
// serialization, not a different projection.
func TestProposalGrammar_YAMLIsTheSameDocument(t *testing.T) {
	_, _, jsonOut, _, _ := runGrammarCommand(t, "proposal", "grammar", "--output", "json")
	outcome, code, yamlOut, stderr, _ := runGrammarCommand(t, "proposal", "grammar", "--output", "yaml")
	if outcome != Success || code != 0 {
		t.Fatalf("the yaml run: outcome %v (exit %d); stderr: %s", outcome, code, stderr)
	}
	converted, err := yaml.YAMLToJSON([]byte(yamlOut))
	if err != nil {
		t.Fatalf("the yaml output does not parse: %v\n%s", err, yamlOut)
	}
	var fromYAML, fromJSON grammar.Grammar
	if err := json.Unmarshal(converted, &fromYAML); err != nil {
		t.Fatalf("decoding the yaml document: %v", err)
	}
	if err := json.Unmarshal([]byte(jsonOut), &fromJSON); err != nil {
		t.Fatalf("decoding the json document: %v", err)
	}
	a, _ := json.Marshal(fromYAML)
	b, _ := json.Marshal(fromJSON)
	if string(a) != string(b) {
		t.Errorf("json and yaml disagree:\njson: %s\nyaml: %s", b, a)
	}
}

// TestProposalGrammar_AUserTemplateAppliesOverTheStructure: a caller template
// applies over the grammar exactly as it applies over a read payload (035).
func TestProposalGrammar_AUserTemplateAppliesOverTheStructure(t *testing.T) {
	const path = "grammar.tmpl"
	seam := &grammarOutputSeam{
		selection:    output.Selection{Format: output.DefaultFormat, Template: &output.TemplateRef{Kind: output.TemplateFile, Path: path}},
		selectionSet: true,
		tmplText:     map[string]string{path: `{{range .ChangeTypes}}{{.Type}}={{.Placement}} {{end}}|{{range .Facts}}{{.ID}} {{end}}`},
	}
	outcome, stdout, stderr := runGrammarOver(t, seam, path, true)
	if outcome != Success {
		t.Fatalf("outcome %v, want Success; stderr: %s", outcome, stderr)
	}
	g, err := grammar.Load()
	if err != nil {
		t.Fatalf("loading the embedded grammar: %v", err)
	}
	for _, ct := range g.ChangeTypes {
		if !strings.Contains(stdout, ct.Type+"="+ct.Placement) {
			t.Errorf("the user template did not see change type %q: %s", ct.Type, stdout)
		}
	}
	for _, f := range g.Facts {
		if !strings.Contains(stdout, f.ID) {
			t.Errorf("the user template did not see fact %q: %s", f.ID, stdout)
		}
	}
}

// TestProposalGrammar_ProducesOnlyTheDocumentedExitCodes: the accord states codes
// 3–7 as a contract fact rather than an aspiration — no request path exists to
// produce them. Sweeping every reachable invocation shape is how that stays true.
func TestProposalGrammar_ProducesOnlyTheDocumentedExitCodes(t *testing.T) {
	invocations := [][]string{
		{"proposal", "grammar"},
		{"proposal", "grammar", "--output", "compact"},
		{"proposal", "grammar", "--output", "json"},
		{"proposal", "grammar", "--output", "yaml"},
		{"proposal", "grammar", "--base-url", "https://example.invalid"},
		{"proposal", "grammar", "positional"},
		{"proposal", "grammar", "--changes", "[]"},
		{"proposal", "grammar", "--output", "/no/such/template"},
	}
	for _, args := range invocations {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, code, _, stderr, seam := runGrammarCommand(t, args...)
			switch code {
			case 0, 2:
				// Success, or a usage error: the only two the accord admits in practice.
			default:
				t.Fatalf("exit %d is outside the command's envelope (0 or 2; 1 only on a corrupt build); stderr: %s", code, stderr)
			}
			if seam.assembleCalled || seam.newClientCalled {
				t.Error("no invocation may assemble a connection or build a client")
			}
			tr := seam.transport.(*cannedTransport)
			if tr.calls != 0 {
				t.Errorf("no invocation may send a request; got %d", tr.calls)
			}
		})
	}
}

// TestProposalGrammar_HelpStatesItInformsAndNeverValidates: the accord requires the
// help text to say the command judges nothing. An agent reading `--help` must not
// come away thinking it can submit a change set for checking.
func TestProposalGrammar_HelpStatesItInformsAndNeverValidates(t *testing.T) {
	cmd := newProposalGrammarCommand(&grammarOutputSeam{})
	if strings.TrimSpace(cmd.Short) == "" {
		t.Fatal("the leaf has no Short help")
	}
	if cmd.Use != "grammar" {
		t.Fatalf("the leaf's Use token is %q, want %q", cmd.Use, "grammar")
	}
	if cmd.Args == nil {
		t.Fatal("the leaf declares no Args validator, so a positional would reach the action")
	}
	if err := cmd.Args(cmd, []string{"changes.json"}); err == nil {
		t.Error("the leaf accepts a positional argument")
	}
	if cmd.HasAvailableLocalFlags() {
		t.Errorf("the leaf declares local flags: %s", cmd.LocalFlags().FlagUsages())
	}
	help := strings.ToLower(cmd.Short + "\n" + cmd.Long)
	for _, claim := range []string{
		"never validates",   // the informs-never-validates boundary
		"no verdict",        // and what that means for the caller
		"no api request",    // the client-less conduct
		"provenance",        // the provenance marking
		"before assembling", // when to consult it
	} {
		if !strings.Contains(help, claim) {
			t.Errorf("the help text does not state %q:\n%s", claim, cmd.Short+"\n"+cmd.Long)
		}
	}
}

// TestProposalGrammarIsNotInTheGatedRegistry: the gated registry lists WRITES. The
// grammar leaf is a read, so adding it there would gate a read forever — the
// mirror of the PROPOSAL_READS edit, asserted so a later well-meaning addition is
// caught.
func TestProposalGrammarIsNotInTheGatedRegistry(t *testing.T) {
	leaves, err := build.ReadGatedRegistry()
	if err != nil {
		t.Fatalf("reading the gated registry: %v", err)
	}
	for _, leaf := range leaves {
		if strings.Fields(leaf)[len(strings.Fields(leaf))-1] == "grammar" {
			t.Fatalf("the gated registry lists %q — it lists writes only, and grammar is a read", leaf)
		}
	}
}

func keysOf(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
