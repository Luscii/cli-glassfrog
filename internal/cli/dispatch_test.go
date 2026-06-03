package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// dispatchTree builds a small assembled tree for dispatch tests: a `version`
// leaf, a `roles` group with a `list` leaf, and (optionally) a `boom` leaf
// whose action returns a runtime error. ran records which leaf actions fired,
// so a test can assert a command did or did not run.
func dispatchTree(ran map[string]bool) *cobra.Command {
	root := NewRootCommand()
	mark := func(name string, err error) *cobra.Command {
		return &cobra.Command{
			Use:   name,
			Short: "the " + name + " command",
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

// A resolved command whose own action errors is reported via the returned
// error but classified Success — a distinct RuntimeError category is deferred
// to Exit-Code Convention (004).
func TestRun_RuntimeActionError_IsSuccessCategory(t *testing.T) {
	ran := map[string]bool{}
	outcome, err := runQuiet(dispatchTree(ran), "boom")
	if !ran["boom"] {
		t.Fatal("boom action did not run")
	}
	if outcome != Success {
		t.Fatalf("runtime action error: outcome = %v, want Success (RuntimeError deferred to 004)", outcome)
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
