package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestOperatorOrientationFeatures runs the executable acceptance for Operator
// Orientation (062). Like the other build-side suites (021/022/036) its Paths
// name ONLY this spec's feature file and it runs with the ~@wip filter, so only
// the scenarios implemented so far execute.
//
// The deliverable is a committed Claude plugin (a manifest + one orientation
// skill), so the executable scenarios assert against that artifact rather than a
// runtime: the plugin is well-formed and discoverable, its content carries the
// required cross-cutting knowledge, and a best-effort drift guard keeps the
// skill's enumerable facts truthful to the shipped CLI. The four @validation
// scenarios (no invented surface, no Holacracy coaching, no distribution
// machinery, describes-but-never-enforces gating) stay @wip, held for
// /score:validate.
func TestOperatorOrientationFeatures(t *testing.T) {
	w := &orientationWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.register(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unequipped-agent-operators/operator-orientation.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: operator-orientation feature scenarios failed")
	}
}

// orientationWorld is the per-scenario state: the loaded manifest + skill, a
// parse error (for the malformed-manifest path), and the drift findings.
type orientationWorld struct {
	manifestRaw []byte
	manifest    OrientationManifest
	manifestErr error
	skill       string
}

func (w *orientationWorld) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = orientationWorld{}
		return ctx, nil
	})

	// --- Scenario: Orientation is consultable once the plugin is present ---
	sc.Step(`^the plugin was present in an agent's environment$`, w.givenPluginPresent)
	sc.Step(`^the agent looks for operating knowledge$`, w.whenLooksForKnowledge)
	sc.Step(`^the orientation knowledge will be available to consult$`, w.thenKnowledgeConsultable)
	sc.Step(`^no configuration beyond the CLI's existing credential setup will be required$`, w.thenNoExtraConfig)

	// --- Scenario: Malformed manifest leaves the plugin unloadable ---
	sc.Step(`^the plugin manifest at "([^"]*)" was malformed$`, w.givenManifestMalformed)
	sc.Step(`^the plugin host attempts to load the plugin$`, w.whenHostLoads)
	sc.Step(`^the host will not register the orientation skill$`, w.thenHostDoesNotRegister)
	sc.Step(`^the agent will fall back to rediscovery with no command in the CLI broken$`, w.thenCLIUnbroken)
}

// --- Scenario: Orientation is consultable once the plugin is present -------

func (w *orientationWorld) givenPluginPresent() error {
	m, raw, err := ReadOrientationManifest()
	if err != nil {
		return fmt.Errorf("plugin manifest did not load: %w", err)
	}
	skill, err := ReadOrientationSkill()
	if err != nil {
		return fmt.Errorf("orientation skill did not load: %w", err)
	}
	w.manifest, w.manifestRaw, w.skill = m, raw, skill
	return nil
}

func (w *orientationWorld) whenLooksForKnowledge() error {
	// Looking for operating knowledge is consulting the skill the host discovered;
	// it was loaded in the Given. Nothing to drive.
	return nil
}

func (w *orientationWorld) thenKnowledgeConsultable() error {
	if w.manifest.Name != "glassfrog-operator" {
		return fmt.Errorf("manifest name is %q, want glassfrog-operator — the host could not identify the plugin", w.manifest.Name)
	}
	// The skill is consultable iff it is discoverable: a frontmatter block that
	// carries the name and the description trigger surface.
	if !hasFrontmatterField(w.skill, "name") || !hasFrontmatterField(w.skill, "description") {
		return fmt.Errorf("SKILL.md lacks the frontmatter name/description that makes it discoverable")
	}
	return nil
}

func (w *orientationWorld) thenNoExtraConfig() error {
	if !ManifestDemandsNoSetup(w.manifestRaw) {
		return fmt.Errorf("manifest declares a key (mcpServers/hooks/commands/agents) that would force setup beyond the CLI's credential flow")
	}
	return nil
}

// --- Scenario: Malformed manifest leaves the plugin unloadable -------------

func (w *orientationWorld) givenManifestMalformed(path string) error {
	if path != OrientationManifestPath {
		return fmt.Errorf("scenario names manifest path %q, guard reads %q", path, OrientationManifestPath)
	}
	// A truncated object — valid JSON start, no close — models a manifest the host
	// cannot parse.
	w.manifestRaw = []byte(`{"name": "glassfrog-operator", "version": "0.1.0"`)
	return nil
}

func (w *orientationWorld) whenHostLoads() error {
	w.manifest, w.manifestErr = ParseOrientationManifest(w.manifestRaw)
	return nil
}

func (w *orientationWorld) thenHostDoesNotRegister() error {
	if w.manifestErr == nil {
		return fmt.Errorf("malformed manifest parsed without error — the host would wrongly register the skill")
	}
	return nil
}

func (w *orientationWorld) thenCLIUnbroken() error {
	// The plugin tree is pure data: nothing under plugin/ is compiled into the
	// CLI, so a manifest the host rejects cannot break a glassfrog command. The
	// agent just falls back to rediscovery.
	clean, err := OrientationPluginHasNoGoCode()
	if err != nil {
		return fmt.Errorf("could not inspect the plugin tree: %w", err)
	}
	if !clean {
		return fmt.Errorf("plugin tree carries Go code — a manifest failure could no longer be isolated from the CLI")
	}
	return nil
}

// hasFrontmatterField reports whether the SKILL.md leading YAML frontmatter block
// (delimited by the first two `---` lines) declares the given field.
func hasFrontmatterField(skill, field string) bool {
	trimmed := strings.TrimSpace(skill)
	if !strings.HasPrefix(trimmed, "---") {
		return false
	}
	rest := trimmed[len("---"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return false
	}
	front := rest[:end]
	for _, line := range strings.Split(front, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), field+":") {
			return true
		}
	}
	return false
}
