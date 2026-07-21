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

	// setupSkill holds the glassfrog-setup content with whitespace collapsed
	// to single spaces, so phrase assertions are resilient to markdown
	// line-wrapping; setupSkillRaw keeps the verbatim content for structural
	// checks (frontmatter, fenced command blocks).
	setupSkill    string
	setupSkillRaw string
	setupAnchors  SetupSkillAnchors

	// no-operating-surface inspection results (the packaging-wide validation
	// scenario).
	manifestExtraKeys   []string
	unresolvedLeaves    []string
	restatedOrientation []string
	mentionedPathSkills []string
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

	// Rule: Go from installed plugin to ready-to-drive through a guided setup
	sc.Step(`^the plugin was installed$`, w.givenSetupSkillLoaded)
	sc.Step(`^the "([^"]*)" CLI was present in the environment$`, w.givenSetupSkillLoadedNamed)
	sc.Step(`^a working credential was configured$`, w.givenSetupSkillLoaded)
	sc.Step(`^the operator invokes the setup skill$`, w.whenSetupSkillInvoked)
	sc.Step(`^the setup skill will confirm the CLI presence and the authenticated identity$`, w.thenConfirmsPresenceAndIdentity)
	sc.Step(`^it will report the environment ready to drive the CLI$`, w.thenReportsReady)

	sc.Step(`^the "([^"]*)" CLI was not present in the environment$`, w.givenSetupSkillLoadedNamed)
	sc.Step(`^the setup skill will report the CLI as missing$`, w.thenReportsCLIMissing)
	sc.Step(`^it will direct the operator to the install script, Homebrew tap, and npm wrapper channels$`, w.thenDirectsToChannels)
	sc.Step(`^it will not attempt to install or bundle the binary itself$`, w.thenNoSelfInstall)

	sc.Step(`^no working credential was configured$`, w.givenSetupSkillLoaded)
	sc.Step(`^the auth check will fail$`, w.thenAuthCheckFailureHandled)
	sc.Step(`^the setup skill will guide the operator through the CLI's existing X-Auth-Token setup$`, w.thenGuidesXAuthTokenSetup)
	sc.Step(`^it will introduce no credential mechanism of its own$`, w.thenNoOwnCredentialMechanism)

	sc.Step(`^the produced setup skill content$`, w.givenSetupSkillLoaded)
	sc.Step(`^it is inspected for how it handles a missing CLI or a missing credential$`, w.whenSetupSkillInvoked)
	sc.Step(`^it will only point at the CLI's existing install channels and credential setup$`, w.thenOnlyPointsAtExistingFixes)
	sc.Step(`^it will install no binary and store no credential of its own$`, w.thenInstallsNothingStoresNothing)

	sc.Step(`^the setup skill had directed the operator to an install channel for a missing CLI$`, w.givenSetupSkillLoaded)
	sc.Step(`^the operator completes the fix$`, w.whenSetupSkillInvoked)
	sc.Step(`^the setup skill will run the presence check again before moving to the auth check$`, w.thenRechecksPresenceAfterFix)
	sc.Step(`^a failing re-check will route back to the fix, never to a ready report$`, w.thenFailingRecheckRoutesBack)

	// Cross-rule validation: distribution only, no operating surface of its own
	sc.Step(`^the packaging artifacts produced by this feature$`, w.givenPackagingArtifacts)
	sc.Step(`^they are inspected for orientation content, operator paths, commands, or API capability$`, w.whenInspectedForOperatingSurface)
	sc.Step(`^none will be present$`, w.thenNoOperatingSurfacePresent)
	sc.Step(`^every operating fact will still live in the plugin the marketplace distributes$`, w.thenOperatingFactsLiveInPlugin)
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

// --- Setup-skill steps ----------------------------------------------------
//
// The setup skill is a declarative artifact — the scenarios' environment
// states (CLI present/absent, credential working/failing) frame WHICH part of
// the instructed journey the Then steps assert; the assertions themselves run
// against the committed content, whitespace-collapsed for phrase checks and
// raw for structural ones, per the family convention.

func (w *operatingSurfacePackagingWorld) givenSetupSkillLoaded() error {
	raw, err := ReadSetupSkill()
	if err != nil {
		return fmt.Errorf("could not read the setup skill: %w", err)
	}
	w.setupSkillRaw = raw
	w.setupSkill = normalizeWS(raw)
	w.setupAnchors, err = LiveSetupSkillAnchors()
	if err != nil {
		return fmt.Errorf("could not extract the setup skill's in-repo anchors: %w", err)
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) givenSetupSkillLoadedNamed(cli string) error {
	if cli != "glassfrog" {
		return fmt.Errorf("the setup skill provisions the %q CLI, not %q", "glassfrog", cli)
	}
	if w.setupSkillRaw == "" {
		return w.givenSetupSkillLoaded()
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) whenSetupSkillInvoked() error {
	if w.setupSkillRaw == "" {
		return fmt.Errorf("no setup skill content is in play — the Given did not run")
	}
	if _, ok := SkillFrontmatterField(w.setupSkillRaw, "description"); !ok {
		return fmt.Errorf("the setup skill carries no frontmatter description — it could never be invoked")
	}
	return nil
}

// thenConfirmsPresenceAndIdentity: the skill instructs both checks — the
// innocuous presence command and the pinned authenticated identity read — with
// presence ordered before auth.
func (w *operatingSurfacePackagingWorld) thenConfirmsPresenceAndIdentity() error {
	presence := strings.Index(w.setupSkillRaw, "glassfrog --version")
	if presence < 0 {
		return fmt.Errorf("the skill does not instruct the presence check (glassfrog --version)")
	}
	auth := strings.Index(w.setupSkillRaw, SetupAuthCheckCommand)
	if auth < 0 {
		return fmt.Errorf("the skill does not instruct the auth check (%s)", SetupAuthCheckCommand)
	}
	if auth < presence {
		return fmt.Errorf("the auth check is instructed before the presence check — a missing binary would surface as a credential failure")
	}
	if !strings.Contains(w.setupSkill, "identity") {
		return fmt.Errorf("the auth check does not read as an authenticated identity confirmation")
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) thenReportsReady() error {
	if !strings.Contains(w.setupSkill, "ready to drive the CLI") {
		return fmt.Errorf("the skill never reports the environment ready to drive the CLI")
	}
	if !strings.Contains(w.setupSkill, "both passed") {
		return fmt.Errorf("the ready report is not conditioned on both checks passing")
	}
	return nil
}

// thenReportsCLIMissing: command-not-found maps to the missing-binary class,
// not to anything credential-shaped.
func (w *operatingSurfacePackagingWorld) thenReportsCLIMissing() error {
	if !strings.Contains(w.setupSkill, "not found") || !strings.Contains(w.setupSkill, "the binary is missing") {
		return fmt.Errorf("the skill does not map a command-not-found to a missing binary")
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) thenDirectsToChannels() error {
	lower := strings.ToLower(w.setupSkill)
	for _, channel := range []string{"install script", "homebrew tap", "npm wrapper"} {
		if !strings.Contains(lower, channel) {
			return fmt.Errorf("the skill does not direct the operator to the %s channel", channel)
		}
	}
	// The npm coordinate is quoted from its in-repo source, never invented.
	if w.setupAnchors.NPMPackage == "" || !strings.Contains(w.setupSkillRaw, w.setupAnchors.NPMPackage) {
		return fmt.Errorf("the skill does not quote the npm wrapper's install coordinate %q", w.setupAnchors.NPMPackage)
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) thenNoSelfInstall() error {
	if !strings.Contains(w.setupSkill, "never installs, bundles, or places the binary itself") {
		return fmt.Errorf("the skill does not state that it installs no binary itself")
	}
	return nil
}

// thenAuthCheckFailureHandled: a non-zero auth-check exit maps to the
// failing-credential class, with exit-code semantics deferred to orientation
// rather than restated.
func (w *operatingSurfacePackagingWorld) thenAuthCheckFailureHandled() error {
	if !strings.Contains(w.setupSkill, "non-zero exit") {
		return fmt.Errorf("the skill does not interpret a non-zero exit of the auth check")
	}
	if !strings.Contains(w.setupSkill, "orientation skill's exit-code reference") {
		return fmt.Errorf("the skill does not defer exit-code semantics to orientation")
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) thenGuidesXAuthTokenSetup() error {
	if !strings.Contains(w.setupSkillRaw, "X-Auth-Token") {
		return fmt.Errorf("the skill never names the CLI's X-Auth-Token mechanism")
	}
	if !strings.Contains(w.setupSkillRaw, "glassfrog auth login") {
		return fmt.Errorf("the skill does not walk the operator to the CLI's own credential command")
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) thenNoOwnCredentialMechanism() error {
	if !strings.Contains(w.setupSkill, "no credential mechanism of its own") {
		return fmt.Errorf("the skill does not state that it introduces no credential mechanism of its own")
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) thenOnlyPointsAtExistingFixes() error {
	if err := w.thenDirectsToChannels(); err != nil {
		return err
	}
	return w.thenGuidesXAuthTokenSetup()
}

func (w *operatingSurfacePackagingWorld) thenInstallsNothingStoresNothing() error {
	if err := w.thenNoSelfInstall(); err != nil {
		return err
	}
	if !strings.Contains(w.setupSkill, "stores nothing") {
		return fmt.Errorf("the skill does not state that it stores no credential of its own")
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) thenRechecksPresenceAfterFix() error {
	if !strings.Contains(w.setupSkill, "re-run the presence check") {
		return fmt.Errorf("the skill does not instruct re-running the presence check after a fix")
	}
	if !strings.Contains(w.setupSkill, "only a passing re-check moves you forward") {
		return fmt.Errorf("the skill lets a fix pass without a verifying re-check")
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) thenFailingRecheckRoutesBack() error {
	if !strings.Contains(w.setupSkill, "a failing re-check routes back") {
		return fmt.Errorf("the skill does not route a failing re-check back to the fix")
	}
	if !strings.Contains(w.setupSkill, "never to a ready report") {
		return fmt.Errorf("the skill does not rule out a ready report on a failing re-check")
	}
	return nil
}

// --- Distribution-only validation steps -----------------------------------

func (w *operatingSurfacePackagingWorld) givenPackagingArtifacts() error {
	if err := w.givenLoaded(); err != nil {
		return err
	}
	w.entry, w.entryFound = FindMarketplacePlugin(w.manifest, MarketplacePluginName)
	if !w.entryFound {
		return fmt.Errorf("the marketplace lists no %q plugin entry", MarketplacePluginName)
	}
	w.sourcePlugin, w.sourceErr = MarketplaceSourcePluginManifest(w.entry.Source)
	return w.givenSetupSkillLoaded()
}

// whenInspectedForOperatingSurface inspects both packaging artifacts for
// operating surface of their own: manifest keys beyond the distribution
// shape, command leaves the CLI does not expose, restated orientation
// enumerations, and operator-path content.
func (w *operatingSurfacePackagingWorld) whenInspectedForOperatingSurface() error {
	// (a) The manifest may carry only distribution keys.
	var keyed map[string]json.RawMessage
	if err := json.Unmarshal(w.manifestRaw, &keyed); err != nil {
		return err
	}
	allowed := map[string]bool{"$schema": true, "name": true, "owner": true, "metadata": true, "plugins": true}
	w.manifestExtraKeys = nil
	for key := range keyed {
		if !allowed[key] {
			w.manifestExtraKeys = append(w.manifestExtraKeys, key)
		}
	}

	// (b) Every `glassfrog <leaf>` the skill instructs (in code spans/blocks)
	// must resolve in the shipped CLI — packaging invents no command.
	liveTop, err := LiveTopLevelCommands()
	if err != nil {
		return err
	}
	facts, err := LiveOrientationFacts()
	if err != nil {
		return err
	}
	resolvable := map[string]bool{}
	for _, cmd := range liveTop {
		resolvable[cmd] = true
	}
	if w.setupAnchors.MeResolves {
		resolvable["me"] = true
	}
	if facts.AuthLogin {
		resolvable["auth"] = true
	}
	w.unresolvedLeaves = nil
	for _, leaf := range setupSkillCommandLeaves(w.setupSkillRaw) {
		if !resolvable[leaf] {
			w.unresolvedLeaves = append(w.unresolvedLeaves, leaf)
		}
	}

	// (c) Orientation's reference enumerations must not be restated: the
	// format-token enumeration and the exit-code reaction table are
	// orientation's anchors.
	w.restatedOrientation = nil
	for _, anchor := range []string{"tokens are exactly", "| Code | Meaning |"} {
		if strings.Contains(w.setupSkill, normalizeWS(anchor)) {
			w.restatedOrientation = append(w.restatedOrientation, anchor)
		}
	}

	// (d) No operator-path content: the sibling path skills' names (derived
	// from the plugin's skills directory, minus orientation and setup itself)
	// must not appear in either packaging artifact.
	root, err := RepoRoot()
	if err != nil {
		return err
	}
	skillDirs, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(path.Clean(w.entry.Source)), "skills"))
	if err != nil {
		return err
	}
	w.mentionedPathSkills = nil
	for _, d := range skillDirs {
		if !d.IsDir() || d.Name() == "orientation" || d.Name() == SetupSkillName() {
			continue
		}
		if strings.Contains(w.setupSkillRaw, d.Name()) || strings.Contains(string(w.manifestRaw), d.Name()) {
			w.mentionedPathSkills = append(w.mentionedPathSkills, d.Name())
		}
	}
	return nil
}

func (w *operatingSurfacePackagingWorld) thenNoOperatingSurfacePresent() error {
	if len(w.manifestExtraKeys) > 0 {
		return fmt.Errorf("the marketplace manifest carries keys beyond the distribution shape: %v", w.manifestExtraKeys)
	}
	if len(w.unresolvedLeaves) > 0 {
		return fmt.Errorf("the setup skill instructs command leaves the CLI does not expose: %v", w.unresolvedLeaves)
	}
	if len(w.restatedOrientation) > 0 {
		return fmt.Errorf("the setup skill restates orientation's reference enumerations: %v", w.restatedOrientation)
	}
	if len(w.mentionedPathSkills) > 0 {
		return fmt.Errorf("the packaging artifacts carry operator-path content: %v", w.mentionedPathSkills)
	}
	return nil
}

// thenOperatingFactsLiveInPlugin: the operating knowledge stays inside the
// plugin the marketplace distributes — the entry's source resolves to the
// plugin that carries the skills, and setup defers reference knowledge there.
func (w *operatingSurfacePackagingWorld) thenOperatingFactsLiveInPlugin() error {
	if w.sourceErr != nil {
		return fmt.Errorf("the marketplace entry does not resolve to the plugin: %w", w.sourceErr)
	}
	root, err := RepoRoot()
	if err != nil {
		return err
	}
	skillsDir := filepath.Join(root, filepath.FromSlash(path.Clean(w.entry.Source)), "skills")
	if _, err := os.Stat(filepath.Join(skillsDir, "orientation", "SKILL.md")); err != nil {
		return fmt.Errorf("the distributed plugin no longer carries the orientation skill: %w", err)
	}
	if _, err := os.Stat(filepath.Join(skillsDir, SetupSkillName(), "SKILL.md")); err != nil {
		return fmt.Errorf("the distributed plugin no longer carries the setup skill: %w", err)
	}
	if !strings.Contains(w.setupSkill, "orientation skill") {
		return fmt.Errorf("the setup skill does not defer reference knowledge to the orientation skill")
	}
	return nil
}

// setupSkillCommandLeaves extracts the first word after `glassfrog ` from the
// skill's code contexts — fenced blocks and inline backtick spans — where an
// instruction is a command, not prose (prose like "the glassfrog plugin" must
// not read as a command leaf).
func setupSkillCommandLeaves(raw string) []string {
	var code []string
	inFence := false
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			code = append(code, trimmed)
		}
	}
	// Inline backtick spans: after splitting on the backtick, the odd-indexed
	// segments are the ones inside a span — prose between spans stays out.
	for i, span := range strings.Split(raw, "`") {
		if i%2 == 1 {
			code = append(code, span)
		}
	}
	seen := map[string]bool{}
	var leaves []string
	for _, c := range code {
		for _, fields := range [][]string{strings.Fields(c)} {
			for i := 0; i+1 < len(fields); i++ {
				if fields[i] != "glassfrog" {
					continue
				}
				next := fields[i+1]
				if next == "" || next[0] < 'a' || next[0] > 'z' {
					continue
				}
				if !seen[next] {
					seen[next] = true
					leaves = append(leaves, next)
				}
			}
		}
	}
	return leaves
}
