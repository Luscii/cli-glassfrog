package cli

import (
	"fmt"
	"strings"

	"github.com/cucumber/godog"
	"github.com/spf13/cobra"
)

// registerHelpVersionSteps wires the Help & Version (003) Given/When/Then steps
// onto the shared world. They build a configured root (configureHelpAndVersion
// over a guard-registered command set) and assert what cobra's standard
// rendering produces for the --help flag, the version flag, and the version
// command. The When steps ("the caller invokes …") are shared with 002.
func (w *world) registerHelpVersionSteps(sc *godog.ScenarioContext) {
	// --- Givens (past tense "was/were registered", distinct from 002's present tense) ---
	sc.Step(`^a "([^"]*)" command and a "([^"]*)" group were registered, each with a summary$`, w.givenLeafAndGroup)
	sc.Step(`^a "([^"]*)" command and a "([^"]*)" group were registered$`, w.givenLeafAndGroup)
	sc.Step(`^a "([^"]*)" group containing "([^"]*)" and "([^"]*)" was registered, each with a summary$`, w.givenGroupContainingTwo)
	sc.Step(`^no commands were registered$`, w.givenNoCommands)
	sc.Step(`^a "([^"]*)" group whose "([^"]*)" subgroup contains "([^"]*)" was registered$`, w.givenNestedGroup)
	sc.Step(`^the command set was assembled$`, w.givenAssembled)
	sc.Step(`^a "([^"]*)" command with summary "([^"]*)" was registered$`, w.givenPathLeafWithSummary)
	sc.Step(`^no command named "([^"]*)" was registered$`, w.givenNoCommandNamedConfigured)
	sc.Step(`^the CLI was built with a version string$`, w.givenBuiltWithVersionString)
	sc.Step(`^the CLI was built with version "([^"]*)"$`, w.givenBuiltWithVersion)
	sc.Step(`^a "([^"]*)" group without its own version flag was registered$`, w.givenGroupNoVersionFlag)

	// --- Thens ---
	sc.Step(`^the CLI will list "([^"]*)" and "([^"]*)"$`, w.thenListsBoth)
	sc.Step(`^each will appear with its one-line summary$`, w.thenEachListedWithSummary)
	sc.Step(`^the CLI will show the "([^"]*)" summary$`, w.thenShowsSummaryOf)
	sc.Step(`^it will list "([^"]*)" and "([^"]*)", each with its summary$`, w.thenListsBothWithSummary)
	sc.Step(`^the CLI will produce a listing naming no commands$`, w.thenListsNoCommands)
	sc.Step(`^it will not fail$`, w.thenOutcomeSuccess)
	sc.Step(`^the CLI will list "([^"]*)" with its summary$`, w.thenListsOneWithSummary)
	sc.Step(`^it will not list the nested "([^"]*)" command$`, w.thenDoesNotListNested)
	sc.Step(`^"([^"]*)" will be listed before "([^"]*)"$`, w.thenListedBefore)
	sc.Step(`^(?:the listing will not include|it will not include) a "([^"]*)" command$`, w.thenListingExcludes)
	sc.Step(`^the CLI will show the "([^"]*)" usage line$`, w.thenShowsUsageLine)
	sc.Step(`^it will show the summary "([^"]*)"$`, w.thenShowsSummaryText)
	sc.Step(`^the CLI will not render usage for "([^"]*)"$`, w.thenNoUsageFor)
	sc.Step(`^the unknown-command outcome will be left to dispatch$`, w.thenLeftToDispatchUnknown)
	sc.Step(`^the CLI will produce help output$`, w.thenProducesHelp)
	sc.Step(`^it will not produce version output$`, w.thenNoVersionOutput)
	sc.Step(`^both invocations will print the same output$`, w.thenInvocationsMatch)
	sc.Step(`^the output will contain "([^"]*)"$`, w.thenOutputContains)
	sc.Step(`^the CLI will not print version output$`, w.thenNoVersionPrinted)
	sc.Step(`^the unrecognized flag will be left to dispatch as a usage error$`, w.thenUsageError)
}

// configuredFactory builds a per-invocation factory that assembles a fresh root,
// applies the scenario's registration, and configures help/version on it. It
// also primes w.root so Then steps can inspect the resulting command tree.
func (w *world) configuredFactory(register func(root *cobra.Command)) {
	w.newRoot = func() *cobra.Command {
		root := NewRootCommand()
		register(root)
		configureHelpAndVersion(root)
		return root
	}
	w.root = w.newRoot()
}

// --- Given implementations ---

func (w *world) givenLeafAndGroup(leaf, group string) error {
	w.configuredFactory(func(root *cobra.Command) {
		MustRegister(root, validLeaf(leaf))
		MustRegister(root, groupWith(group, "list"))
	})
	return nil
}

func (w *world) givenGroupContainingTwo(group, a, b string) error {
	w.configuredFactory(func(root *cobra.Command) {
		MustRegister(root, groupWith(group, a, b))
	})
	return nil
}

func (w *world) givenNoCommands() error {
	w.configuredFactory(func(*cobra.Command) {})
	return nil
}

func (w *world) givenNestedGroup(group, subgroup, leaf string) error {
	w.configuredFactory(func(root *cobra.Command) {
		sub := groupWith(subgroup, leaf)
		outer := &cobra.Command{Use: group, Short: "the " + group + " group"}
		MustRegister(outer, sub)
		MustRegister(root, outer)
	})
	return nil
}

func (w *world) givenAssembled() error {
	w.newRoot = Assemble
	w.root = Assemble()
	return nil
}

func (w *world) givenPathLeafWithSummary(path, summary string) error {
	parts := strings.Fields(path)
	w.configuredFactory(func(root *cobra.Command) {
		if len(parts) == 1 {
			MustRegister(root, &cobra.Command{Use: parts[0], Short: summary, RunE: noopRun})
			return
		}
		// Nested path: build the group(s), put the summarized leaf at the tip.
		leaf := &cobra.Command{Use: parts[len(parts)-1], Short: summary, RunE: noopRun}
		group := &cobra.Command{Use: parts[len(parts)-2], Short: "the " + parts[len(parts)-2] + " group"}
		MustRegister(group, leaf)
		MustRegister(root, group)
	})
	return nil
}

func (w *world) givenNoCommandNamedConfigured(_ string) error {
	// The named command is absent; seed an unrelated command so the root is a
	// non-runnable parent (cobra only emits unknown-command for a parent).
	w.configuredFactory(func(root *cobra.Command) {
		MustRegister(root, validLeaf("version"))
	})
	return nil
}

func (w *world) givenBuiltWithVersionString() error {
	w.newRoot = Assemble
	w.root = Assemble()
	return nil
}

func (w *world) givenBuiltWithVersion(v string) error {
	// Pin the single source of truth both --version and the version command
	// read; reset() restores the default after the scenario.
	version = v
	w.newRoot = Assemble
	w.root = Assemble()
	return nil
}

func (w *world) givenGroupNoVersionFlag(group string) error {
	// configureHelpAndVersion puts the --version flag on the root only, so a
	// group has no version flag of its own.
	w.configuredFactory(func(root *cobra.Command) {
		MustRegister(root, groupWith(group, "list"))
	})
	return nil
}

// --- Then implementations ---

func (w *world) thenListsBoth(a, b string) error {
	listed := availableCommands(w.output)
	for _, name := range []string{a, b} {
		if !containsString(listed, name) {
			return fmt.Errorf("listing %v did not include %q; output: %q", listed, name, w.output)
		}
	}
	return nil
}

func (w *world) thenEachListedWithSummary() error {
	for _, ln := range listingLines(w.output) {
		fields := strings.Fields(ln)
		if len(fields) < 2 {
			return fmt.Errorf("listed command %q has no summary; output: %q", strings.TrimSpace(ln), w.output)
		}
	}
	return nil
}

func (w *world) thenShowsSummaryOf(name string) error {
	cmd, _, err := w.root.Find(strings.Fields(name))
	if err != nil || cmd == nil {
		return fmt.Errorf("could not resolve %q to read its summary: %v", name, err)
	}
	if !strings.Contains(w.output, cmd.Short) {
		return fmt.Errorf("output does not show the %q summary %q; output: %q", name, cmd.Short, w.output)
	}
	return nil
}

func (w *world) thenListsBothWithSummary(a, b string) error {
	if err := w.thenListsBoth(a, b); err != nil {
		return err
	}
	return w.thenEachListedWithSummary()
}

func (w *world) thenListsNoCommands() error {
	// With an empty command set cobra produces help output (here, the root's
	// description) but no "Available Commands:" section — a listing that names
	// no commands, which is exactly the contract.
	if strings.TrimSpace(w.output) == "" {
		return fmt.Errorf("expected help output for the empty command set, got nothing")
	}
	if listed := availableCommands(w.output); len(listed) > 0 {
		return fmt.Errorf("expected no commands listed, got %v", listed)
	}
	return nil
}

func (w *world) thenListsOneWithSummary(name string) error {
	listed := availableCommands(w.output)
	if !containsString(listed, name) {
		return fmt.Errorf("listing %v did not include %q; output: %q", listed, name, w.output)
	}
	return w.thenEachListedWithSummary()
}

func (w *world) thenDoesNotListNested(name string) error {
	if containsString(availableCommands(w.output), name) {
		return fmt.Errorf("listing included the nested command %q, but only immediate children should appear; output: %q", name, w.output)
	}
	return nil
}

func (w *world) thenListedBefore(first, second string) error {
	listed := availableCommands(w.output)
	fi, si := indexOf(listed, first), indexOf(listed, second)
	if fi < 0 || si < 0 {
		return fmt.Errorf("both %q and %q must be listed; got %v", first, second, listed)
	}
	if fi >= si {
		return fmt.Errorf("%q should be listed before %q; got order %v", first, second, listed)
	}
	return nil
}

func (w *world) thenListingExcludes(name string) error {
	if containsString(availableCommands(w.output), name) {
		return fmt.Errorf("listing must not include the built-in %q command; output: %q", name, w.output)
	}
	return nil
}

func (w *world) thenShowsUsageLine(path string) error {
	// cobra's usage line is "  glassfrog <path> [flags]". Assert the invocation
	// path is present in the rendered usage.
	want := "glassfrog " + path
	if !strings.Contains(w.output, "Usage:") || !strings.Contains(w.output, want) {
		return fmt.Errorf("expected a usage line for %q (containing %q), got: %q", path, want, w.output)
	}
	return nil
}

func (w *world) thenShowsSummaryText(summary string) error {
	if !strings.Contains(w.output, summary) {
		return fmt.Errorf("expected the summary %q in the usage output, got: %q", summary, w.output)
	}
	return nil
}

func (w *world) thenNoUsageFor(name string) error {
	if strings.Contains(w.output, "glassfrog "+name) {
		return fmt.Errorf("CLI rendered usage for the unregistered command %q; output: %q", name, w.output)
	}
	return nil
}

func (w *world) thenLeftToDispatchUnknown() error {
	if err := w.thenUsageError(); err != nil {
		return err
	}
	if w.lastErr == nil || !strings.Contains(w.lastErr.Error(), "unknown command") {
		return fmt.Errorf("expected dispatch's unknown-command error, got: %v", w.lastErr)
	}
	return nil
}

func (w *world) thenProducesHelp() error {
	if !strings.Contains(w.output, "Usage:") {
		return fmt.Errorf("expected help output (usage text), got: %q", w.output)
	}
	return nil
}

func (w *world) thenNoVersionOutput() error {
	// Help wins: the output must be help, not the bare version line. Compare
	// against the RESOLVED version (the value --version would print), not the raw
	// injected var — which is empty by default (spec 023) and would make this
	// check vacuous.
	if strings.TrimSpace(w.output) == resolvedVersion() {
		return fmt.Errorf("output was the version string, expected help: %q", w.output)
	}
	return nil
}

func (w *world) thenInvocationsMatch() error {
	if len(w.outputs) < 2 {
		return fmt.Errorf("expected two invocations to compare, recorded %d", len(w.outputs))
	}
	a, b := w.outputs[len(w.outputs)-2], w.outputs[len(w.outputs)-1]
	if a != b {
		return fmt.Errorf("invocations produced different output: %q vs %q", a, b)
	}
	return nil
}

func (w *world) thenOutputContains(want string) error {
	for _, out := range w.outputs {
		if !strings.Contains(out, want) {
			return fmt.Errorf("an invocation's output %q does not contain %q", out, want)
		}
	}
	return nil
}

func (w *world) thenNoVersionPrinted() error {
	// Compare against the RESOLVED version (non-empty), not the raw injected var:
	// the default var is empty (spec 023), and strings.Contains(out, "") is always
	// true, which would make this assertion always fail.
	if strings.Contains(w.output, resolvedVersion()) {
		return fmt.Errorf("CLI printed version output for a non-version request: %q", w.output)
	}
	return nil
}

// --- listing helpers ---

// listingLines returns the raw lines of cobra's "Available Commands:" section.
func listingLines(helpOutput string) []string {
	var lines []string
	inSection := false
	for _, ln := range strings.Split(helpOutput, "\n") {
		if strings.HasPrefix(ln, "Available Commands:") {
			inSection = true
			continue
		}
		if inSection {
			if strings.TrimSpace(ln) == "" {
				break
			}
			lines = append(lines, ln)
		}
	}
	return lines
}

func containsString(ss []string, want string) bool { return indexOf(ss, want) >= 0 }

func indexOf(ss []string, want string) int {
	for i, s := range ss {
		if s == want {
			return i
		}
	}
	return -1
}
