package build

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestOperatingSurfacePackagingFeatures runs the executable acceptance for
// Operating-Surface Packaging (070). Like the sibling build-side suites
// (062–069) its Paths name ONLY this spec's feature file and it runs with the
// ~@wip filter, so only the scenarios implemented so far execute.
//
// The deliverable is distribution, not operation: a repo-root marketplace
// manifest a Claude plugin host reads when the repo is added as a marketplace,
// a `glassfrog-setup` provisioning skill inside the plugin, and the
// consistency guard that keeps the marketplace entry truthful to the plugin it
// ships. The artifacts carry no runtime Go path of their own, so the
// executable scenarios assert against their content and against the guard
// helpers' behaviour on both the real artifacts and synthetically drifted
// copies (drift red, version-pin red, sibling-append green).
func TestOperatingSurfacePackagingFeatures(t *testing.T) {
	w := &operatingSurfacePackagingWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.register(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unequipped-agent-operators/operating-surface-packaging.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: operating-surface-packaging feature scenarios failed")
	}
}

// operatingSurfacePackagingWorld is the per-scenario state: the loaded
// marketplace manifest (parsed + raw), the located glassfrog entry, the plugin
// manifest its source resolves to, the consistency findings of a guard run,
// and the synthetic-drift / sibling-append results.
type operatingSurfacePackagingWorld struct {
	manifest    MarketplaceManifest
	manifestRaw []byte

	entry      MarketplacePluginEntry
	entryFound bool

	sourcePlugin OrientationManifest
	sourceErr    error

	// findings holds a consistency-guard run over the REAL committed artifacts.
	findings []string
	// driftFindings / driftSourceErr hold guard results over synthetically
	// drifted inputs (identity divergence, version pin, unresolvable source).
	driftFindings  []string
	driftSourceErr error
	pinnedEntry    MarketplacePluginEntry

	// appended* hold the manifest after a synthetic sibling entry is appended
	// at the JSON level and re-parsed through the same parser.
	appended    MarketplaceManifest
	appendedErr error
	siblingName string

	pluginsIsArray bool
}

func (w *operatingSurfacePackagingWorld) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = operatingSurfacePackagingWorld{}
		return ctx, nil
	})

	// Rule: Install the operating surface from the repo ----------------------
	sc.Step(`^an agent environment had a Claude plugin host$`, w.givenLoaded)
	sc.Step(`^the environment adds "([^"]*)" as a plugin marketplace$`, w.whenAddsMarketplace)
	sc.Step(`^the host will find the marketplace manifest at "([^"]*)"$`, w.thenManifestAtPath)
	sc.Step(`^the marketplace will list the "([^"]*)" plugin as an installable entry$`, w.thenListsInstallableEntry)

	sc.Step(`^the repository had been added as a plugin marketplace$`, w.givenLoaded)
	sc.Step(`^the environment installs the "([^"]*)" plugin from it$`, w.whenInstallsPlugin)
	sc.Step(`^the plugin's skills, agents, and write-safety hook will become available in that environment$`, w.thenSurfaceAvailable)

	sc.Step(`^the marketplace listed the "([^"]*)" plugin$`, w.givenEntryResolved)
	sc.Step(`^the entry no longer resolves to a matching plugin definition in the repo$`, w.whenEntryNoLongerResolves)
	sc.Step(`^the mismatch will be treated as a defect to fix$`, w.thenTreatedAsDefect)
	sc.Step(`^it will not be accepted as a tolerable difference$`, w.thenNotTolerable)

	sc.Step(`^the marketplace entry for the "([^"]*)" plugin$`, w.givenEntryResolved)
	sc.Step(`^the named plugin and its "([^"]*)" source are resolved against the repo$`, w.whenResolvedAgainstRepo)
	sc.Step(`^the source will point at the real plugin definition$`, w.thenSourcePointsAtRealPlugin)
	sc.Step(`^the entry's identity will match the plugin manifest's$`, w.thenIdentityMatches)

	sc.Step(`^the "([^"]*)" marketplace entry carried a "([^"]*)" key$`, w.givenEntryCarriedKey)
	sc.Step(`^the internal/build consistency guard runs$`, w.whenConsistencyGuardRuns)
	sc.Step(`^the guard will fail$`, w.thenGuardWillFail)
	sc.Step(`^it will report that the plugin version is single-sourced in the plugin manifest$`, w.thenReportsSingleSourced)

	// Rule: Carry a future sibling plugin ------------------------------------
	sc.Step(`^the marketplace shipped with the "([^"]*)" plugin as its only entry$`, w.givenOnlyEntry)
	sc.Step(`^a sibling operating-surface plugin is later added$`, w.whenSiblingAppended)
	sc.Step(`^it will be listed as an additional entry in the plugins list$`, w.thenListedAsAdditionalEntry)
	sc.Step(`^the marketplace will require no restructuring to carry it$`, w.thenNoRestructuring)

	sc.Step(`^the marketplace manifest$`, w.givenLoaded)
	sc.Step(`^it is inspected for whether it can carry more than one plugin$`, w.whenInspectedForMultiEntry)
	sc.Step(`^its plugins list will admit additional entries without restructuring$`, w.thenAdmitsAdditionalEntries)
}

// --- Given ------------------------------------------------------------------

func (w *operatingSurfacePackagingWorld) givenLoaded() error {
	m, raw, err := ReadMarketplaceManifest()
	if err != nil {
		return fmt.Errorf("could not read the marketplace manifest: %w", err)
	}
	w.manifest, w.manifestRaw = m, raw
	return nil
}

// givenEntryResolved loads the manifest, locates the named entry, and resolves
// its source to the plugin manifest — the fully-consistent starting state the
// drift and identity scenarios diverge from.
func (w *operatingSurfacePackagingWorld) givenEntryResolved(name string) error {
	if err := w.givenLoaded(); err != nil {
		return err
	}
	w.entry, w.entryFound = FindMarketplacePlugin(w.manifest, name)
	if !w.entryFound {
		return fmt.Errorf("the marketplace lists no %q plugin entry", name)
	}
	w.sourcePlugin, w.sourceErr = MarketplaceSourcePluginManifest(w.entry.Source)
	if w.sourceErr != nil {
		return fmt.Errorf("the %q entry's source did not resolve: %w", name, w.sourceErr)
	}
	return nil
}

// givenEntryCarriedKey synthesizes the version-pinned entry: the REAL entry's
// fields re-marshalled with the named key added, decoded through the same
// parse path the guard uses — so the scenario exercises key-presence
// detection, not a hand-built struct.
func (w *operatingSurfacePackagingWorld) givenEntryCarriedKey(name, key string) error {
	if key != "version" {
		return fmt.Errorf("this suite only synthesizes a %q pin; got %q", "version", key)
	}
	if err := w.givenEntryResolved(name); err != nil {
		return err
	}
	pinned, err := json.Marshal(map[string]string{
		"name":        w.entry.Name,
		"source":      w.entry.Source,
		"description": w.entry.Description,
		key:           "9.9.9",
	})
	if err != nil {
		return err
	}
	if err := json.Unmarshal(pinned, &w.pinnedEntry); err != nil {
		return err
	}
	if !w.pinnedEntry.HasVersion {
		return fmt.Errorf("the synthesized entry did not register its %q key — the parse path lost the pin", key)
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) givenOnlyEntry(name string) error {
	if err := w.givenLoaded(); err != nil {
		return err
	}
	if len(w.manifest.Plugins) != 1 || w.manifest.Plugins[0].Name != name {
		return fmt.Errorf("the scenario premises a single %q entry; the manifest carries %d entries", name, len(w.manifest.Plugins))
	}
	w.entry, w.entryFound = w.manifest.Plugins[0], true
	return nil
}

// --- When ---------------------------------------------------------------

func (w *operatingSurfacePackagingWorld) whenAddsMarketplace(repo string) error {
	// The add is the host reading the manifest from the repo checkout; the
	// repo slug names where the checkout comes from. Content-wise the add is
	// the load the Given performed — re-verify it parsed.
	if strings.TrimSpace(repo) == "" {
		return fmt.Errorf("the marketplace add names no repository")
	}
	if w.manifestRaw == nil {
		return w.givenLoaded()
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) whenInstallsPlugin(name string) error {
	w.entry, w.entryFound = FindMarketplacePlugin(w.manifest, name)
	if !w.entryFound {
		return fmt.Errorf("the marketplace lists no %q plugin entry to install", name)
	}
	w.sourcePlugin, w.sourceErr = MarketplaceSourcePluginManifest(w.entry.Source)
	return nil
}

// whenEntryNoLongerResolves synthesizes both drift classes the defect accord
// covers: a source that resolves to no plugin definition, and an entry whose
// identity diverges from the plugin manifest it ships.
func (w *operatingSurfacePackagingWorld) whenEntryNoLongerResolves() error {
	if !w.entryFound {
		return fmt.Errorf("no marketplace entry is in play — the Given did not run")
	}
	// The real artifacts' consistency, for contrast in the Then steps.
	w.findings = CheckMarketplaceEntryConsistency(w.entry, w.sourcePlugin)

	// Drift class 1: the source resolves to a directory with no plugin
	// manifest (a real directory, so only the resolution is synthetic).
	_, w.driftSourceErr = MarketplaceSourcePluginManifest("./docs")

	// Drift class 2: the entry's identity diverges from the plugin manifest.
	diverged := w.entry
	diverged.Name = w.entry.Name + "-renamed"
	diverged.Description = w.entry.Description + " (drifted)"
	w.driftFindings = CheckMarketplaceEntryConsistency(diverged, w.sourcePlugin)
	return nil
}

func (w *operatingSurfacePackagingWorld) whenResolvedAgainstRepo(source string) error {
	if !w.entryFound {
		return fmt.Errorf("no marketplace entry is in play — the Given did not run")
	}
	if w.entry.Source != source {
		return fmt.Errorf("the entry's source is %q, not the %q the scenario names", w.entry.Source, source)
	}
	w.sourcePlugin, w.sourceErr = MarketplaceSourcePluginManifest(w.entry.Source)
	if w.sourceErr == nil {
		w.findings = CheckMarketplaceEntryConsistency(w.entry, w.sourcePlugin)
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) whenConsistencyGuardRuns() error {
	if !w.pinnedEntry.HasVersion {
		return fmt.Errorf("no version-pinned entry is in play — the Given did not run")
	}
	w.driftFindings = CheckMarketplaceEntryConsistency(w.pinnedEntry, w.sourcePlugin)
	return nil
}

// whenSiblingAppended appends a synthetic sibling entry at the JSON level and
// re-parses the result through the same parser the guard uses — the concrete
// form of "a sibling is one appended entry, never a restructure".
func (w *operatingSurfacePackagingWorld) whenSiblingAppended() error {
	w.siblingName = "holacracy-practice"
	appendedRaw, err := w.appendSibling(w.siblingName)
	if err != nil {
		return err
	}
	w.appended, w.appendedErr = ParseMarketplaceManifest(appendedRaw)
	return nil
}

func (w *operatingSurfacePackagingWorld) whenInspectedForMultiEntry() error {
	var keyed map[string]json.RawMessage
	if err := json.Unmarshal(w.manifestRaw, &keyed); err != nil {
		return err
	}
	plugins, ok := keyed["plugins"]
	if !ok {
		return fmt.Errorf("the manifest carries no %q key", "plugins")
	}
	w.pluginsIsArray = len(plugins) > 0 && plugins[0] == '['

	// The checkable form of "can carry more than one": an appended copy still
	// parses and lists both entries.
	w.siblingName = "holacracy-practice"
	appendedRaw, err := w.appendSibling(w.siblingName)
	if err != nil {
		return err
	}
	w.appended, w.appendedErr = ParseMarketplaceManifest(appendedRaw)
	return nil
}

// appendSibling returns the raw manifest with one synthetic sibling entry
// appended to the plugins list — every other top-level key untouched.
func (w *operatingSurfacePackagingWorld) appendSibling(name string) ([]byte, error) {
	if w.manifestRaw == nil {
		return nil, fmt.Errorf("no manifest is in play — the Given did not run")
	}
	var keyed map[string]json.RawMessage
	if err := json.Unmarshal(w.manifestRaw, &keyed); err != nil {
		return nil, err
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(keyed["plugins"], &entries); err != nil {
		return nil, fmt.Errorf("the plugins key does not decode as a list: %w", err)
	}
	sibling, err := json.Marshal(map[string]string{
		"name":        name,
		"source":      "github:Luscii/" + name,
		"description": "A sibling operating-surface plugin, appended as one more entry.",
	})
	if err != nil {
		return nil, err
	}
	entries = append(entries, sibling)
	rawEntries, err := json.Marshal(entries)
	if err != nil {
		return nil, err
	}
	keyed["plugins"] = rawEntries
	return json.Marshal(keyed)
}

// --- Then ---------------------------------------------------------------

func (w *operatingSurfacePackagingWorld) thenManifestAtPath(p string) error {
	if p != MarketplaceManifestPath {
		return fmt.Errorf("the scenario names %q but the manifest constant is %q", p, MarketplaceManifestPath)
	}
	if w.manifestRaw == nil {
		return fmt.Errorf("the marketplace manifest was not read from %s", MarketplaceManifestPath)
	}
	if strings.TrimSpace(w.manifest.Name) == "" {
		return fmt.Errorf("the manifest parsed without a marketplace name")
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) thenListsInstallableEntry(name string) error {
	entry, found := FindMarketplacePlugin(w.manifest, name)
	if !found {
		return fmt.Errorf("the marketplace lists no %q plugin entry", name)
	}
	if strings.TrimSpace(entry.Source) == "" {
		return fmt.Errorf("the %q entry has no source to install from", name)
	}
	if strings.TrimSpace(entry.Description) == "" {
		return fmt.Errorf("the %q entry carries no description", name)
	}
	return nil
}

// thenSurfaceAvailable asserts the installed source directory carries the
// three surface kinds the plugin ships — skills, agents, and the write-safety
// hook — all located through the entry's own source, never a hard-coded path.
func (w *operatingSurfacePackagingWorld) thenSurfaceAvailable() error {
	if w.sourceErr != nil {
		return fmt.Errorf("the install could not resolve the plugin definition: %w", w.sourceErr)
	}
	root, err := RepoRoot()
	if err != nil {
		return err
	}
	sourceDir := filepath.Join(root, filepath.FromSlash(path.Clean(w.entry.Source)))

	skillDirs, err := os.ReadDir(filepath.Join(sourceDir, "skills"))
	if err != nil || len(skillDirs) == 0 {
		return fmt.Errorf("the installed plugin carries no skills directory with entries: %v", err)
	}
	sawSkill := false
	for _, d := range skillDirs {
		if d.IsDir() {
			if _, statErr := os.Stat(filepath.Join(sourceDir, "skills", d.Name(), "SKILL.md")); statErr == nil {
				sawSkill = true
			}
		}
	}
	if !sawSkill {
		return fmt.Errorf("no skills/<name>/SKILL.md found under the installed plugin")
	}

	agentEntries, err := os.ReadDir(filepath.Join(sourceDir, "agents"))
	if err != nil {
		return fmt.Errorf("the installed plugin carries no agents directory: %w", err)
	}
	sawAgent := false
	for _, d := range agentEntries {
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			sawAgent = true
		}
	}
	if !sawAgent {
		return fmt.Errorf("no agents/*.md found under the installed plugin")
	}

	hooks, err := os.ReadFile(filepath.Join(sourceDir, "hooks", "hooks.json"))
	if err != nil {
		return fmt.Errorf("the installed plugin carries no hooks/hooks.json: %w", err)
	}
	if !strings.Contains(string(hooks), "glassfrog-write-gate") {
		return fmt.Errorf("hooks/hooks.json does not wire the write-safety gate")
	}
	return nil
}

// thenTreatedAsDefect: both drift classes make the guard fail — an
// unresolvable source is an error, an identity divergence produces findings —
// and a guard failure is a red build, the "defect to fix" the accord demands.
func (w *operatingSurfacePackagingWorld) thenTreatedAsDefect() error {
	if w.driftSourceErr == nil {
		return fmt.Errorf("a source resolving to no plugin definition was accepted — the guard would stay green on a dead source")
	}
	if len(w.driftFindings) == 0 {
		return fmt.Errorf("an identity divergence produced no findings — the guard would stay green on a lying entry")
	}
	return nil
}

// thenNotTolerable: the guard's pass condition over the real artifacts is
// ZERO findings — there is no tolerated band between "consistent" and
// "defect", so any finding is a failure, and each finding directs the fix.
func (w *operatingSurfacePackagingWorld) thenNotTolerable() error {
	if len(w.findings) != 0 {
		return fmt.Errorf("the committed artifacts themselves carry findings — the green baseline is not zero: %v", w.findings)
	}
	for _, f := range w.driftFindings {
		if !strings.Contains(f, MarketplaceManifestPath) && !strings.Contains(f, OrientationManifestPath) {
			return fmt.Errorf("drift finding does not point at the manifest to fix: %q", f)
		}
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) thenSourcePointsAtRealPlugin() error {
	if w.sourceErr != nil {
		return fmt.Errorf("the source did not resolve to a plugin definition: %w", w.sourceErr)
	}
	if strings.TrimSpace(w.sourcePlugin.Name) == "" {
		return fmt.Errorf("the resolved plugin manifest carries no name")
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) thenIdentityMatches() error {
	if w.sourceErr != nil {
		return fmt.Errorf("no resolved plugin manifest to compare against: %w", w.sourceErr)
	}
	if len(w.findings) != 0 {
		return fmt.Errorf("the entry's identity drifted from the plugin manifest's: %v", w.findings)
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) thenGuardWillFail() error {
	if len(w.driftFindings) == 0 {
		return fmt.Errorf("the consistency guard produced no findings for the version-pinned entry")
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) thenReportsSingleSourced() error {
	for _, f := range w.driftFindings {
		if strings.Contains(f, "single-sourced") && strings.Contains(f, OrientationManifestPath) {
			return nil
		}
	}
	return fmt.Errorf("no finding reports the version as single-sourced in the plugin manifest: %v", w.driftFindings)
}

func (w *operatingSurfacePackagingWorld) thenListedAsAdditionalEntry() error {
	if w.appendedErr != nil {
		return fmt.Errorf("the appended manifest no longer parses: %w", w.appendedErr)
	}
	if got, want := len(w.appended.Plugins), len(w.manifest.Plugins)+1; got != want {
		return fmt.Errorf("the appended manifest lists %d entries, want %d", got, want)
	}
	if _, found := FindMarketplacePlugin(w.appended, w.siblingName); !found {
		return fmt.Errorf("the sibling %q is not listed in the appended manifest", w.siblingName)
	}
	return nil
}

// thenNoRestructuring: after the append, the original entry is untouched and
// every top-level identity field reads the same — the sibling changed one
// list, nothing else.
func (w *operatingSurfacePackagingWorld) thenNoRestructuring() error {
	if w.appendedErr != nil {
		return fmt.Errorf("the appended manifest no longer parses: %w", w.appendedErr)
	}
	kept, found := FindMarketplacePlugin(w.appended, w.entry.Name)
	if !found {
		return fmt.Errorf("the original %q entry is gone after the append", w.entry.Name)
	}
	if kept != w.entry {
		return fmt.Errorf("the original %q entry changed shape after the append: %+v vs %+v", w.entry.Name, kept, w.entry)
	}
	if w.appended.Name != w.manifest.Name || w.appended.Owner != w.manifest.Owner || w.appended.Schema != w.manifest.Schema {
		return fmt.Errorf("appending the sibling changed the marketplace's own identity fields")
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) thenAdmitsAdditionalEntries() error {
	if !w.pluginsIsArray {
		return fmt.Errorf("the manifest's plugins key is not a JSON list — additional entries would restructure it")
	}
	if w.appendedErr != nil {
		return fmt.Errorf("an appended copy no longer parses: %w", w.appendedErr)
	}
	if _, found := FindMarketplacePlugin(w.appended, w.siblingName); !found {
		return fmt.Errorf("an appended sibling entry is not listed by the same parser")
	}
	return nil
}
