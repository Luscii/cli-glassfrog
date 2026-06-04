package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/spf13/cobra"
)

// TestFeatures runs the executable acceptance scenarios this package owns: the
// CLI-skeleton features under features/no-runnable-cli/ (command-registration,
// argument-dispatch, help-and-version, exit-code-convention) and the
// credential-storage feature (the `auth login` command, Credential Storage
// 006). @wip scenarios are skipped — the @validation scenarios stay @wip
// because they are held out for independent verification (the validate skill),
// not implemented by the Builder.
//
// The paths are scoped to this package's own features rather than the whole
// features/ directory: each package's godog suite owns the feature(s) whose
// steps it defines (the cli command tree + auth login here; Credential
// Discovery's unauthenticated-access/credential-discovery.feature is owned by
// internal/auth). Pointing at credential-storage.feature specifically — not the
// unauthenticated-access/ directory — keeps the resolver's feature out of this
// suite, which has no matching step definitions for it.
func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeScenario,
		Options: &godog.Options{
			Format: "pretty",
			Paths: []string{
				"../../features/no-runnable-cli",
				"../../features/unauthenticated-access/credential-storage.feature",
			},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: feature scenarios failed")
	}
}

// world is the per-scenario state shared across Given/When/Then steps.
type world struct {
	root         *cobra.Command
	groups       map[string]*cobra.Command
	orderedGroup []*cobra.Command
	lastGroup    *cobra.Command
	found        *cobra.Command
	lastErr      error
	results      []error
	panicked     bool
	dispatched   bool

	// Dispatch (002): outcome and observable output of a Run invocation, plus
	// which leaf actions fired and the command a routing assertion last named.
	outcome    Outcome
	outcomeSet bool
	ran        map[string]bool
	output     string
	routed     string

	// Exit-Code Convention (004): the process exit code an invocation maps to,
	// computed through the production recover+map core (recoverToCode) so the
	// panic→1 path is exercised end-to-end rather than re-implemented here.
	exitCode    int
	exitCodeSet bool

	// Help & Version (003): a per-invocation root factory (so a parity
	// scenario's two invocations each run against a fresh assembled+configured
	// root) and the output of every invocation in the scenario, in order.
	newRoot func() *cobra.Command
	outputs []string

	// Credential Storage (006): the per-scenario store fixture (temp dirs,
	// controlled token sources, isTTY, scripted interactor, captured output).
	cred *credState
}

func (w *world) reset() {
	w.root = NewRootCommand()
	w.groups = map[string]*cobra.Command{}
	w.orderedGroup = nil
	w.lastGroup = nil
	w.found = nil
	w.lastErr = nil
	w.results = nil
	w.panicked = false
	w.dispatched = false
	w.outcome = Success
	w.outcomeSet = false
	w.ran = map[string]bool{}
	w.output = ""
	w.routed = ""
	w.exitCode = 0
	w.exitCodeSet = false
	w.newRoot = nil
	w.outputs = nil
	w.cred = &credState{}
	// Restore the build-time version var to its default placeholder so a 003
	// "built with version X" scenario cannot leak its value into later
	// scenarios (the var is the single source of truth both --version and the
	// version command read).
	version = "0.0.0-dev"
}

// track records a group as the most-recently-created one, so a later step
// (e.g. registering a subgroup "under it") can find its parent deterministically.
func (w *world) track(name string, g *cobra.Command) *cobra.Command {
	w.groups[name] = g
	w.lastGroup = g
	return g
}

func noopRun(*cobra.Command, []string) error { return nil }

func validLeaf(name string) *cobra.Command {
	return &cobra.Command{Use: name, Short: "the " + name + " command", RunE: noopRun}
}

// groupWith builds a valid group assembled (children registered through the
// guard) before it is returned for registration under its parent.
func groupWith(name string, subs ...string) *cobra.Command {
	g := &cobra.Command{Use: name, Short: "the " + name + " group"}
	for _, s := range subs {
		MustRegister(g, validLeaf(s))
	}
	return g
}

func seedGroup(name string) *cobra.Command { return groupWith(name, "info") }

func initializeScenario(sc *godog.ScenarioContext) {
	w := &world{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		w.reset()
		return ctx, nil
	})

	// Argument Dispatch (002) steps live in dispatch_bdd_test.go.
	w.registerDispatchSteps(sc)
	// Help & Version (003) steps live in helpversion_bdd_test.go.
	w.registerHelpVersionSteps(sc)
	// Exit-Code Convention (004) steps live in exitcode_bdd_test.go.
	w.registerExitCodeSteps(sc)
	// Credential Storage (006) steps live in credstorage_bdd_test.go.
	w.registerCredStorageSteps(sc)

	// --- Givens ---
	sc.Step(`^the command set was empty$`, func() error { return nil })
	sc.Step(`^an otherwise valid registration state$`, func() error { return nil })
	sc.Step(`^the command set already contained a "([^"]*)" group$`, w.givenGroupPresent)
	sc.Step(`^a "([^"]*)" group with "([^"]*)" and "([^"]*)" subcommands was registered$`, w.givenGroupWithSubs)
	sc.Step(`^a "([^"]*)" group and a "([^"]*)" group were registered$`, w.givenTwoGroups)
	sc.Step(`^a "([^"]*)" group was registered$`, w.givenGroupSeed)
	sc.Step(`^the name "([^"]*)" was already registered at the top level$`, w.givenNameAtTopLevel)
	sc.Step(`^several commands were registered successfully$`, w.givenSeveralRegistered)
	sc.Step(`^one further command fails registration$`, w.givenOneFurtherFails)

	// --- Whens ---
	sc.Step(`^a Maintainer registers a "([^"]*)" leaf with summary "([^"]*)" and an action$`, w.whenRegisterLeaf)
	sc.Step(`^a Maintainer registers a "([^"]*)" group$`, w.whenRegisterGroup)
	sc.Step(`^a Maintainer registers a "([^"]*)" group containing "([^"]*)" and "([^"]*)" subcommands$`, w.whenRegisterGroupContaining)
	sc.Step(`^the command set is queried for "([^"]*)" with no further path$`, w.whenQueryBareGroup)
	sc.Step(`^a Maintainer registers a "([^"]*)" subcommand under each group$`, w.whenRegisterUnderEach)
	sc.Step(`^a Maintainer registers a "([^"]*)" subgroup containing an "([^"]*)" subcommand under it$`, w.whenRegisterSubgroup)
	sc.Step(`^a Maintainer registers another command named "([^"]*)" at the top level$`, w.whenRegisterDuplicate)
	sc.Step(`^a Maintainer registers a command whose name is empty or only whitespace$`, w.whenRegisterEmptyName)
	sc.Step(`^a Maintainer registers a command whose summary is empty or only whitespace$`, w.whenRegisterEmptySummary)
	sc.Step(`^a Maintainer registers a leaf command that has no action$`, w.whenRegisterLeafNoAction)
	sc.Step(`^a Maintainer registers a group that has no children$`, w.whenRegisterEmptyGroup)
	sc.Step(`^the CLI starts$`, w.whenCLIStarts)

	// --- Thens ---
	sc.Step(`^querying the command set for "([^"]*)" will return that command$`, w.thenQueryReturns)
	sc.Step(`^enumerating the command set will list "([^"]*)" with its summary$`, w.thenEnumerateListsWithSummary)
	sc.Step(`^both "([^"]*)" and "([^"]*)" will be present in the command set$`, w.thenBothPresent)
	sc.Step(`^the "([^"]*)" group will be unchanged$`, w.thenGroupUnchanged)
	sc.Step(`^querying the path "([^"]*)" will return the (\S+) command$`, w.thenQueryPathReturns)
	sc.Step(`^enumerating the command set will list the "([^"]*)" group and both subcommands$`, w.thenEnumerateListsGroup)
	sc.Step(`^the "([^"]*)" group node will be returned$`, w.thenGroupNodeReturned)
	sc.Step(`^its "([^"]*)" and "([^"]*)" subcommands will be reachable through it$`, w.thenSubcommandsReachable)
	sc.Step(`^both registrations will succeed$`, w.thenBothSucceed)
	sc.Step(`^"([^"]*)" and "([^"]*)" will resolve independently$`, w.thenResolveIndependently)
	sc.Step(`^registration will fail with an error naming "([^"]*)"$`, w.thenFailNaming)
	sc.Step(`^the failure will occur before any user command runs$`, w.thenFailBeforeDispatch)
	sc.Step(`^registration will fail with an error identifying the command$`, w.thenFailIdentifying)
	sc.Step(`^registration will fail with an error identifying the group$`, w.thenFailIdentifying)
	sc.Step(`^no command will be dispatched$`, w.thenNoDispatch)
	sc.Step(`^the CLI will not expose a partial command tree$`, w.thenNoPartialTree)
}

// --- Given implementations ---

func (w *world) givenGroupPresent(name string) error {
	return Register(w.root, w.track(name, seedGroup(name)))
}

func (w *world) givenGroupWithSubs(name, a, b string) error {
	return Register(w.root, w.track(name, groupWith(name, a, b)))
}

func (w *world) givenTwoGroups(a, b string) error {
	for _, name := range []string{a, b} {
		g := w.track(name, seedGroup(name))
		w.orderedGroup = append(w.orderedGroup, g)
		if err := Register(w.root, g); err != nil {
			return err
		}
	}
	return nil
}

func (w *world) givenGroupSeed(name string) error {
	return Register(w.root, w.track(name, seedGroup(name)))
}

func (w *world) givenNameAtTopLevel(name string) error {
	return Register(w.root, validLeaf(name))
}

func (w *world) givenSeveralRegistered() error {
	if err := Register(w.root, validLeaf("version")); err != nil {
		return err
	}
	return Register(w.root, groupWith("roles", "list"))
}

func (w *world) givenOneFurtherFails() error {
	defer func() {
		if r := recover(); r != nil {
			w.panicked = true
		}
	}()
	// A malformed registration via MustRegister aborts startup (panics).
	MustRegister(w.root, &cobra.Command{Use: "broken", Short: ""})
	return nil
}

// --- When implementations ---

func (w *world) whenRegisterLeaf(name, summary string) error {
	w.lastErr = Register(w.root, &cobra.Command{Use: name, Short: summary, RunE: noopRun})
	return w.lastErr
}

func (w *world) whenRegisterGroup(name string) error {
	w.lastErr = Register(w.root, w.track(name, seedGroup(name)))
	return w.lastErr
}

func (w *world) whenRegisterGroupContaining(name, a, b string) error {
	w.lastErr = Register(w.root, w.track(name, groupWith(name, a, b)))
	return w.lastErr
}

func (w *world) whenQueryBareGroup(name string) error {
	found, _, err := w.root.Find([]string{name})
	w.found = found
	return err
}

func (w *world) whenRegisterUnderEach(sub string) error {
	for _, g := range w.orderedGroup {
		w.results = append(w.results, Register(g, validLeaf(sub)))
	}
	return nil
}

func (w *world) whenRegisterSubgroup(subgroup, leaf string) error {
	parent := w.lastRegisteredGroup()
	if parent == nil {
		return fmt.Errorf("no parent group available for subgroup %q", subgroup)
	}
	w.lastErr = Register(parent, groupWith(subgroup, leaf))
	return w.lastErr
}

func (w *world) whenRegisterDuplicate(name string) error {
	w.lastErr = Register(w.root, validLeaf(name))
	return nil // the error is the assertion target, not a step failure
}

func (w *world) whenRegisterEmptyName() error {
	w.lastErr = Register(w.root, &cobra.Command{Use: "   ", Short: "a summary", RunE: noopRun})
	return nil
}

func (w *world) whenRegisterEmptySummary() error {
	w.lastErr = Register(w.root, &cobra.Command{Use: "thing", Short: "   ", RunE: noopRun})
	return nil
}

func (w *world) whenRegisterLeafNoAction() error {
	w.lastErr = Register(w.root, &cobra.Command{Use: "thing", Short: "a summary"})
	return nil
}

func (w *world) whenRegisterEmptyGroup() error {
	w.lastErr = Register(w.root, &cobra.Command{Use: "empty", Short: "a summary"})
	return nil
}

func (w *world) whenCLIStarts() error {
	// Registration aborted at startup (panic), so Execute is never reached and
	// nothing is dispatched.
	if w.panicked {
		w.dispatched = false
		return nil
	}
	w.dispatched = true
	return nil
}

// --- Then implementations ---

func (w *world) thenQueryReturns(name string) error {
	got, _, err := w.root.Find([]string{name})
	if err != nil || got == nil || got.Name() != name {
		return fmt.Errorf("expected %q to resolve, got %v (err %v)", name, got, err)
	}
	return nil
}

func (w *world) thenEnumerateListsWithSummary(name string) error {
	for _, c := range w.root.Commands() {
		if c.Name() == name {
			if strings.TrimSpace(c.Short) == "" {
				return fmt.Errorf("%q is listed without a summary", name)
			}
			return nil
		}
	}
	return fmt.Errorf("%q not found when enumerating the command set", name)
}

func (w *world) thenBothPresent(a, b string) error {
	for _, name := range []string{a, b} {
		if got, _, err := w.root.Find([]string{name}); err != nil || got.Name() != name {
			return fmt.Errorf("%q not present: %v", name, err)
		}
	}
	return nil
}

func (w *world) thenGroupUnchanged(name string) error {
	g, ok := w.groups[name]
	if !ok {
		return fmt.Errorf("group %q was not tracked", name)
	}
	if len(g.Commands()) == 0 {
		return fmt.Errorf("group %q lost its children", name)
	}
	return nil
}

func (w *world) thenQueryPathReturns(path, leaf string) error {
	parts := strings.Fields(path)
	got, _, err := w.root.Find(parts)
	if err != nil || got == nil || got.Name() != leaf {
		return fmt.Errorf("path %q did not resolve to %q: got %v (err %v)", path, leaf, got, err)
	}
	return nil
}

func (w *world) thenEnumerateListsGroup(name string) error {
	for _, c := range w.root.Commands() {
		if c.Name() == name {
			if len(c.Commands()) < 2 {
				return fmt.Errorf("group %q does not list both subcommands", name)
			}
			return nil
		}
	}
	return fmt.Errorf("group %q not found in enumeration", name)
}

func (w *world) thenGroupNodeReturned(name string) error {
	if w.found == nil || w.found.Name() != name {
		return fmt.Errorf("expected group node %q, got %v", name, w.found)
	}
	if len(w.found.Commands()) == 0 {
		return fmt.Errorf("%q resolved but is not a group (no children)", name)
	}
	return nil
}

func (w *world) thenSubcommandsReachable(a, b string) error {
	if w.found == nil {
		return errors.New("no group node was resolved")
	}
	for _, sub := range []string{a, b} {
		if got, _, err := w.found.Find([]string{sub}); err != nil || got.Name() != sub {
			return fmt.Errorf("subcommand %q not reachable through the group: %v", sub, err)
		}
	}
	return nil
}

func (w *world) thenBothSucceed() error {
	if len(w.results) != 2 {
		return fmt.Errorf("expected 2 registrations, got %d", len(w.results))
	}
	for i, err := range w.results {
		if err != nil {
			return fmt.Errorf("registration %d failed: %v", i, err)
		}
	}
	return nil
}

func (w *world) thenResolveIndependently(pathA, pathB string) error {
	a, _, errA := w.root.Find(strings.Fields(pathA))
	b, _, errB := w.root.Find(strings.Fields(pathB))
	if errA != nil || errB != nil {
		return fmt.Errorf("paths did not resolve: %v / %v", errA, errB)
	}
	if a == b {
		return fmt.Errorf("%q and %q resolved to the same command", pathA, pathB)
	}
	return nil
}

func (w *world) thenFailNaming(name string) error {
	var re *RegistrationError
	if !errors.As(w.lastErr, &re) {
		return fmt.Errorf("expected a RegistrationError, got %v", w.lastErr)
	}
	if re.Command != name {
		return fmt.Errorf("error should name %q, named %q", name, re.Command)
	}
	if !strings.Contains(w.lastErr.Error(), "already registered") {
		return fmt.Errorf("duplicate error should mention the collision: %v", w.lastErr)
	}
	return nil
}

func (w *world) thenFailBeforeDispatch() error {
	if w.dispatched {
		return errors.New("a command was dispatched despite a failed registration")
	}
	return nil
}

func (w *world) thenFailIdentifying() error {
	var re *RegistrationError
	if !errors.As(w.lastErr, &re) {
		return fmt.Errorf("expected a RegistrationError identifying the command, got %v", w.lastErr)
	}
	return nil
}

func (w *world) thenNoDispatch() error {
	if w.dispatched {
		return errors.New("a command was dispatched, but startup should have aborted")
	}
	return nil
}

func (w *world) thenNoPartialTree() error {
	if !w.panicked {
		return errors.New("startup did not abort on the failed registration")
	}
	return nil
}

// lastRegisteredGroup returns the most-recently-created group (tracked via
// w.track), so "register a subgroup under it" has a deterministic parent.
func (w *world) lastRegisteredGroup() *cobra.Command {
	return w.lastGroup
}
