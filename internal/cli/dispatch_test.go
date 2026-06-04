package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// dispatchTree builds a small assembled tree for dispatch tests: a `version`
// leaf, a `boom` leaf whose action returns a runtime error, and a `roles` group
// with a `list` leaf. ran records which leaf actions fired, so a test can assert
// a command did or did not run.
func dispatchTree(ran map[string]bool) *cobra.Command {
	root := NewRootCommand()
	mark := func(name string, err error) *cobra.Command {
		return &cobra.Command{
			Use:   name,
			Short: "the " + name + " command",
			// Mirror the real leaves: they accept no positional arguments, so an
			// unexpected one is a usage error rather than silently ignored.
			Args: cobra.NoArgs,
			RunE: func(*cobra.Command, []string) error {
				ran[name] = true
				return err
			},
		}
	}
	MustRegister(root, mark("version", nil))
	MustRegister(root, mark("boom", errors.New("kaboom")))
	roles := &cobra.Command{Use: "roles", Short: "Read roles"}
	MustRegister(roles, mark("list", nil))
	MustRegister(root, roles)
	return root
}

// runQuiet runs Run with output captured, so test logs stay clean and so a
// cobra help/usage dump does not leak to the test runner.
func runQuiet(root *cobra.Command, args ...string) (Outcome, error) {
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	return Run(root, args)
}

// runCapture runs Run and returns the combined stdout+stderr alongside the
// outcome, for assertions on what the operator actually sees.
func runCapture(root *cobra.Command, args ...string) (Outcome, error, string) {
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	outcome, err := Run(root, args)
	return outcome, err, buf.String()
}

func TestRun_TopLevelLeafRoutes_Success(t *testing.T) {
	ran := map[string]bool{}
	outcome, err := runQuiet(dispatchTree(ran), "version")
	if err != nil {
		t.Fatalf("version returned error: %v", err)
	}
	if outcome != Success {
		t.Fatalf("version: outcome = %v, want Success", outcome)
	}
	if !ran["version"] {
		t.Fatal("version action did not run")
	}
}

func TestRun_NestedLeafRoutes_Success(t *testing.T) {
	ran := map[string]bool{}
	outcome, err := runQuiet(dispatchTree(ran), "roles", "list")
	if err != nil {
		t.Fatalf("roles list returned error: %v", err)
	}
	if outcome != Success {
		t.Fatalf("roles list: outcome = %v, want Success", outcome)
	}
	if !ran["list"] {
		t.Fatal("roles list action did not run")
	}
}

func TestRun_BareGroupResolvesToHelp_Success(t *testing.T) {
	outcome, err := runQuiet(dispatchTree(map[string]bool{}), "roles")
	if err != nil || outcome != Success {
		t.Fatalf("bare group: outcome = %v, err = %v; want Success, nil", outcome, err)
	}
}

func TestRun_EmptyInvocationResolvesToRoot_Success(t *testing.T) {
	outcome, err := runQuiet(dispatchTree(map[string]bool{}))
	if err != nil || outcome != Success {
		t.Fatalf("empty invocation: outcome = %v, err = %v; want Success, nil", outcome, err)
	}
}

func TestRun_UnknownTopLevelCommand_UsageError(t *testing.T) {
	ran := map[string]bool{}
	outcome, err := runQuiet(dispatchTree(ran), "rolez")
	if outcome != UsageError {
		t.Fatalf("unknown command: outcome = %v, want UsageError", outcome)
	}
	if err == nil || !strings.Contains(err.Error(), "rolez") {
		t.Fatalf("unknown command error should name the token, got: %v", err)
	}
	if len(ran) != 0 {
		t.Fatalf("no command should have run, ran: %v", ran)
	}
}

func TestRun_UnknownSubcommandUnderGroup_UsageError(t *testing.T) {
	ran := map[string]bool{}
	outcome, err := runQuiet(dispatchTree(ran), "roles", "lst")
	if outcome != UsageError {
		t.Fatalf("unknown subcommand: outcome = %v, want UsageError", outcome)
	}
	if err == nil || !strings.Contains(err.Error(), "lst") {
		t.Fatalf("unknown subcommand error should name %q, got: %v", "lst", err)
	}
	if ran["list"] {
		t.Fatal("a near-miss subcommand must not run the real one")
	}
}

// Pins the exact-match non-behavior: a prefix never resolves to a longer
// command. Depends on cobra's EnablePrefixMatching staying false.
func TestRun_PrefixDoesNotResolve_UsageError(t *testing.T) {
	ran := map[string]bool{}
	outcome, err := runQuiet(dispatchTree(ran), "ro", "list")
	if outcome != UsageError {
		t.Fatalf("prefix: outcome = %v, want UsageError", outcome)
	}
	if err == nil || !strings.Contains(err.Error(), "ro") {
		t.Fatalf("prefix error should name the unresolved token %q, got: %v", "ro", err)
	}
	if ran["list"] {
		t.Fatal("a prefix must not route to roles list")
	}
}

func TestRun_UnknownFlag_UsageError_CommandDoesNotRun(t *testing.T) {
	ran := map[string]bool{}
	outcome, err := runQuiet(dispatchTree(ran), "roles", "list", "--bogus")
	if outcome != UsageError {
		t.Fatalf("unknown flag: outcome = %v, want UsageError", outcome)
	}
	if err == nil || !strings.Contains(err.Error(), "bogus") {
		t.Fatalf("unknown flag error should name the flag, got: %v", err)
	}
	if ran["list"] {
		t.Fatal("the command must not run when an unknown flag is rejected")
	}
}

// A resolved command whose own action errors is classified RuntimeError, with
// the error still travelling via the return. 002 deferred this distinct
// category to Exit-Code Convention (004), which maps ExitCode(RuntimeError) == 1.
func TestRun_RuntimeActionError_IsRuntimeErrorCategory(t *testing.T) {
	ran := map[string]bool{}
	outcome, err := runQuiet(dispatchTree(ran), "boom")
	if !ran["boom"] {
		t.Fatal("boom action did not run")
	}
	if outcome != RuntimeError {
		t.Fatalf("runtime action error: outcome = %v, want RuntimeError", outcome)
	}
	if err == nil || !strings.Contains(err.Error(), "kaboom") {
		t.Fatalf("runtime error should travel via the returned error, got: %v", err)
	}
}

// Pins cobra's package-global to its exact-match default. If any code sets it
// true, the exact-match non-behavior breaks process-wide and silently.
func TestExactMatch_PrefixMatchingDisabled(t *testing.T) {
	if cobra.EnablePrefixMatching {
		t.Fatal("cobra.EnablePrefixMatching must stay false for exact matching")
	}
}

// The Invalid-input accord: an unexpected positional argument the resolved
// command does not accept is a usage error, and the command does not run —
// never silently ignored.
func TestRun_UnexpectedPositionalArg_UsageError_CommandDoesNotRun(t *testing.T) {
	ran := map[string]bool{}
	outcome, err := runQuiet(dispatchTree(ran), "version", "extra")
	if outcome != UsageError {
		t.Fatalf("unexpected positional arg: outcome = %v, want UsageError", outcome)
	}
	if err == nil || !strings.Contains(err.Error(), "extra") {
		t.Fatalf("usage error should name the unexpected argument, got: %v", err)
	}
	if ran["version"] {
		t.Fatal("the command must not run when an unexpected positional arg is rejected")
	}
}

// A runtime action error is classified RuntimeError, not UsageError, even
// though the command also declares an Args validator — the validator passed, so
// the error is the action's own and not an arg rejection.
func TestRun_RuntimeError_NotMisclassifiedAsArgError(t *testing.T) {
	ran := map[string]bool{}
	outcome, err := runQuiet(dispatchTree(ran), "boom")
	if !ran["boom"] {
		t.Fatal("boom action did not run")
	}
	if outcome != RuntimeError {
		t.Fatalf("runtime error: outcome = %v, want RuntimeError", outcome)
	}
	if err == nil || !strings.Contains(err.Error(), "kaboom") {
		t.Fatalf("runtime error should travel via the returned error, got: %v", err)
	}
}

// The nested-group unknown-subcommand error is synthesized by dispatch (cobra
// returned nil), so dispatch must write it to stderr — otherwise the operator
// sees only the group help and never which token was unrecognized.
func TestRun_NestedUnknownSubcommand_WritesErrorToStderr(t *testing.T) {
	outcome, err, output := runCapture(dispatchTree(map[string]bool{}), "roles", "lst")
	if outcome != UsageError || err == nil {
		t.Fatalf("nested unknown subcommand: outcome = %v, err = %v; want UsageError + error", outcome, err)
	}
	if !strings.Contains(output, "lst") {
		t.Fatalf("the synthesized error naming %q must reach the operator's output, got: %q", "lst", output)
	}
	if !strings.Contains(output, "--help") {
		t.Fatalf("the synthesized error should point the caller to help, got: %q", output)
	}
}

// A resolved command that returns a *commandUsageError is classified
// UsageError (the invocation, not the action's execution, is at fault), so
// Exit-Code Convention maps it to code 2 — distinct from a plain action error
// (RuntimeError → 1). This is the seam Credential Storage (006) uses for
// "no token to store" and "existing credential, no --overwrite".
func TestRun_CommandUsageError_IsUsageErrorCategory(t *testing.T) {
	root := NewRootCommand()
	MustRegister(root, &cobra.Command{
		Use:           "needsinput",
		Short:         "the needsinput command",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(*cobra.Command, []string) error {
			return &commandUsageError{errors.New("nothing supplied")}
		},
	})
	outcome, err := runQuiet(root, "needsinput")
	if outcome != UsageError {
		t.Fatalf("command usage error: outcome = %v, want UsageError", outcome)
	}
	if err == nil || !strings.Contains(err.Error(), "nothing supplied") {
		t.Fatalf("the underlying message should travel via the error, got: %v", err)
	}
}

func TestOutcome_String_RuntimeError(t *testing.T) {
	if got := RuntimeError.String(); got != "RuntimeError" {
		t.Fatalf("RuntimeError.String() = %q, want %q", got, "RuntimeError")
	}
}

func TestOutcome_String_UnknownPreservesValue(t *testing.T) {
	got := Outcome(99).String()
	if got != "Outcome(99)" {
		t.Fatalf("unknown Outcome should preserve its numeric value, got %q", got)
	}
}
