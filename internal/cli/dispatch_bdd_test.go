package cli

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/cucumber/godog"
	"github.com/spf13/cobra"
)

// registerDispatchSteps wires the Argument Dispatch (002) Given/When/Then steps
// onto the shared world. These exercise cli.Run against an assembled tree and
// assert the (outcome category, output) the dispatch accords promise.
func (w *world) registerDispatchSteps(sc *godog.ScenarioContext) {
	// --- Givens (present-tense "is registered", distinct from 001's past tense) ---
	sc.Step(`^a "([^"]*)" group with a "([^"]*)" subcommand is registered$`, w.givenGroupWithOneSub)
	sc.Step(`^a "([^"]*)" command is registered at the top level$`, w.givenTopLevelCommand)
	sc.Step(`^"([^"]*)" is the only registered command beginning with "([^"]*)"$`, w.givenOnlyCommandWithPrefix)
	sc.Step(`^no command named "([^"]*)" is registered$`, w.givenNoCommandNamed)
	sc.Step(`^a "([^"]*) ([^"]*)" command is registered$`, w.givenNestedLeafRegistered)
	sc.Step(`^a "([^"]*)" group with "([^"]*)" and "([^"]*)" subcommands is registered$`, w.givenGroupWithTwoSubs)
	sc.Step(`^any registered command set$`, w.givenAnyCommandSet)

	// --- Whens (the caller invokes a command line) ---
	sc.Step(`^the caller invokes "([^"]*)"$`, w.whenCallerInvokes)
	sc.Step(`^the caller invokes "([^"]*)" with no further token$`, w.whenCallerInvokes)
	sc.Step(`^the caller invokes "([^"]*)" with no tokens$`, w.whenCallerInvokes)

	// --- Thens ---
	sc.Step(`^dispatch will route to the "([^"]*)" command$`, w.thenRouteTo)
	sc.Step(`^dispatch will route to "([^"]*)"$`, w.thenRouteTo)
	sc.Step(`^the "([^"]*)" command's action will run$`, w.thenActionRan)
	sc.Step(`^its action will run$`, w.thenRoutedActionRan)
	sc.Step(`^dispatch will not route to "([^"]*)"$`, w.thenNotRouteTo)
	sc.Step(`^(?:dispatch|it) will report an unknown-command error naming "([^"]*)"$`, w.thenUnknownCommandNaming)
	sc.Step(`^it will point the caller to help$`, w.thenPointsToHelp)
	sc.Step(`^it will classify the outcome as a usage error$`, w.thenUsageError)
	sc.Step(`^dispatch will report a usage error naming the unexpected "([^"]*)"$`, w.thenUsageErrorNaming)
	sc.Step(`^the "([^"]*)" command will not run$`, w.thenCommandDidNotRun)
	sc.Step(`^dispatch will resolve to the "([^"]*)" group$`, w.thenResolveToGroup)
	sc.Step(`^route to a help outcome listing "([^"]*)" and "([^"]*)"$`, w.thenHelpListing)
	sc.Step(`^the outcome will be a success$`, w.thenOutcomeSuccess)
	sc.Step(`^dispatch will resolve to the root$`, w.thenResolveToRoot)
	sc.Step(`^route to a help outcome$`, w.thenHelpOutcome)
}

// recordingLeaf is a valid leaf whose action records that it ran, so a routing
// assertion can tell a routed command apart from one that was never reached.
func (w *world) recordingLeaf(name string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: "the " + name + " command",
		// Mirror the production leaves and the unit-test tree: leaves accept no
		// positional arguments, so the BDD harness rejects unexpected tokens the
		// same way real commands do (002 Invalid-input accord).
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			w.ran[name] = true
			return nil
		},
	}
}

func (w *world) recordingGroup(name string, subs ...string) *cobra.Command {
	g := &cobra.Command{Use: name, Short: "the " + name + " group"}
	for _, s := range subs {
		MustRegister(g, w.recordingLeaf(s))
	}
	return g
}

// --- Given implementations ---

func (w *world) givenGroupWithOneSub(group, sub string) error {
	return Register(w.root, w.track(group, w.recordingGroup(group, sub)))
}

func (w *world) givenTopLevelCommand(name string) error {
	return Register(w.root, w.recordingLeaf(name))
}

func (w *world) givenOnlyCommandWithPrefix(name, _ string) error {
	// Register the longer command (e.g. "roles") that begins with the typed
	// prefix (e.g. "ro"); the scenario then asserts the prefix does NOT resolve
	// to it — exact matching, no abbreviation. (The prefix argument is context
	// for the reader; resolution is exact, so it is not needed at setup.)
	return Register(w.root, w.track(name, w.recordingGroup(name, "list")))
}

func (w *world) givenNoCommandNamed(_ string) error {
	// The named command is absent. Seed an unrelated command so the root has
	// subcommands — cobra only emits an unknown-command error (and the help
	// pointer) when the root is a non-runnable parent.
	return Register(w.root, w.recordingLeaf("version"))
}

func (w *world) givenNestedLeafRegistered(group, leaf string) error {
	return Register(w.root, w.track(group, w.recordingGroup(group, leaf)))
}

func (w *world) givenGroupWithTwoSubs(group, a, b string) error {
	return Register(w.root, w.track(group, w.recordingGroup(group, a, b)))
}

func (w *world) givenAnyCommandSet() error {
	if err := Register(w.root, w.recordingLeaf("version")); err != nil {
		return err
	}
	return Register(w.root, w.recordingGroup("roles", "list"))
}

// --- When implementation ---

// whenCallerInvokes parses a "glassfrog <tokens>" command line, drops the
// binary name, and dispatches the remaining tokens through Run with output
// captured for help/error assertions.
func (w *world) whenCallerInvokes(invocation string) error {
	fields := strings.Fields(invocation)
	var args []string
	if len(fields) > 1 {
		args = fields[1:]
	}
	buf := &bytes.Buffer{}
	w.root.SetOut(buf)
	w.root.SetErr(buf)
	w.outcome, w.lastErr = Run(w.root, args)
	w.outcomeSet = true
	w.output = buf.String()
	return nil
}

// --- Then implementations ---

func (w *world) thenRouteTo(name string) error {
	w.routed = name
	if w.outcome != Success {
		return fmt.Errorf("routing to %q: outcome = %v, want Success", name, w.outcome)
	}
	if !w.ran[name] {
		return fmt.Errorf("dispatch did not route to %q (its action never ran)", name)
	}
	return nil
}

func (w *world) thenActionRan(name string) error {
	if !w.ran[name] {
		return fmt.Errorf("the %q command's action did not run", name)
	}
	return nil
}

func (w *world) thenRoutedActionRan() error {
	if w.routed == "" {
		return fmt.Errorf("no command was named by a prior routing step")
	}
	return w.thenActionRan(w.routed)
}

func (w *world) thenNotRouteTo(name string) error {
	if w.ran[name] {
		return fmt.Errorf("dispatch routed to %q, but it should not have", name)
	}
	return nil
}

func (w *world) thenUnknownCommandNaming(token string) error {
	if w.lastErr == nil {
		return fmt.Errorf("expected an unknown-command error naming %q, got nil", token)
	}
	if !strings.Contains(w.lastErr.Error(), token) {
		return fmt.Errorf("error should name the unrecognized token %q, got: %v", token, w.lastErr)
	}
	return nil
}

func (w *world) thenPointsToHelp() error {
	if !strings.Contains(w.output, "--help") {
		return fmt.Errorf("expected output to point the caller to help, got: %q", w.output)
	}
	return nil
}

func (w *world) thenUsageError() error {
	if !w.outcomeSet {
		return fmt.Errorf("no invocation was dispatched")
	}
	if w.outcome != UsageError {
		return fmt.Errorf("outcome = %v, want UsageError", w.outcome)
	}
	return nil
}

func (w *world) thenUsageErrorNaming(token string) error {
	if err := w.thenUsageError(); err != nil {
		return err
	}
	bare := strings.TrimLeft(token, "-")
	if w.lastErr == nil || !strings.Contains(w.lastErr.Error(), bare) {
		return fmt.Errorf("usage error should name the unexpected %q, got: %v", token, w.lastErr)
	}
	return nil
}

func (w *world) thenCommandDidNotRun(name string) error {
	if w.ran[name] {
		return fmt.Errorf("the %q command ran, but the invocation should have been refused", name)
	}
	return nil
}

func (w *world) thenResolveToGroup(name string) error {
	if w.outcome != Success || w.lastErr != nil {
		return fmt.Errorf("resolving to the %q group: outcome = %v, err = %v; want Success, nil", name, w.outcome, w.lastErr)
	}
	return nil
}

func (w *world) thenHelpListing(a, b string) error {
	for _, name := range []string{a, b} {
		if !strings.Contains(w.output, name) {
			return fmt.Errorf("help outcome did not list %q; output: %q", name, w.output)
		}
	}
	return nil
}

func (w *world) thenOutcomeSuccess() error {
	if !w.outcomeSet {
		return fmt.Errorf("no invocation was dispatched")
	}
	if w.outcome != Success {
		return fmt.Errorf("outcome = %v, want Success", w.outcome)
	}
	return nil
}

func (w *world) thenResolveToRoot() error {
	if w.outcome != Success || w.lastErr != nil {
		return fmt.Errorf("resolving to root: outcome = %v, err = %v; want Success, nil", w.outcome, w.lastErr)
	}
	return nil
}

func (w *world) thenHelpOutcome() error {
	if !strings.Contains(w.output, "Usage:") {
		return fmt.Errorf("expected a help outcome (usage text), got: %q", w.output)
	}
	return nil
}
