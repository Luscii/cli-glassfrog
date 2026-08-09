package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOperatingSurfaceSelfContainment is the live tripwire: it walks the real
// plugin/ from the repo root (the same repo-root helper the sibling guards
// use) and fails on any deny-lexicon match, any dangling in-surface path, or a
// missing/empty surface. It runs inside `go test ./...`, so both merge gates
// already execute it with zero new wiring; anyone adding a plugin file learns
// the rule at the first test run.
func TestOperatingSurfaceSelfContainment(t *testing.T) {
	root, err := RepoRoot()
	if err != nil {
		t.Fatalf("locating the repo root: %v", err)
	}
	scan, err := ScanOperatingSurface(root)
	if err != nil {
		t.Fatalf("scanning the operating surface: %v", err)
	}
	if len(scan.Violations) > 0 {
		t.Fatalf("the operating surface references the development repository (%d violations):\n  - %s",
			len(scan.Violations), strings.Join(scan.Violations, "\n  - "))
	}
}

// --- Lexicon self-checks -----------------------------------------------------

// TestSurfaceLexiconExamplesMatch pins the contract the lexicon source states:
// every entry's concrete example matches its own pattern. A pattern edit that
// un-matches its documented violation class is a red fixture, not a merge.
func TestSurfaceLexiconExamplesMatch(t *testing.T) {
	for _, entry := range SurfaceDenyLexicon {
		if entry.Property == "" || entry.Example == "" {
			t.Errorf("lexicon entry %q (family %s) is missing its property comment or example", entry.Pattern, entry.Family)
			continue
		}
		if !entry.Pattern.MatchString(entry.Example) {
			t.Errorf("lexicon entry (family %s: %s): example %q does not match its own pattern %q",
				entry.Family, entry.Property, entry.Example, entry.Pattern)
		}
	}
}

// knownSafeTokens is the corpus of operating-world tokens the lexicon must
// never match — each class pinned here so a lexicon edit cannot silently
// widen a pattern. These mirror the interface accord's known-safe list.
var knownSafeTokens = []string{
	"the example id prp_0123 reads the proposal again",
	"hand the ten_0456 id onward",
	"the role_0789 anchor",
	"version 0.34.1 of the plugin",
	"exit code 7 is the stale-write refusal",
	"exit codes 0–7 cover every outcome",
	"walk with --per-page 500",
	"the v5 spec carries the enum",
	"the published specification speaks here",
	"ask the CLI for the exact flags",
}

// TestSurfaceLexiconKnownSafeTokens proves each known-safe token class matches
// no lexicon entry, keeping operating-world references legal by construction.
func TestSurfaceLexiconKnownSafeTokens(t *testing.T) {
	for _, tok := range knownSafeTokens {
		for _, entry := range SurfaceDenyLexicon {
			if m := entry.Pattern.FindString(tok); m != "" {
				t.Errorf("known-safe token %q trips lexicon entry (family %s: %s) on %q",
					tok, entry.Family, entry.Property, m)
			}
		}
	}
}

// --- Fixture surfaces --------------------------------------------------------

// writeFixtureSurface lays a fixture operating surface under a fresh temp
// root: files maps plugin/-relative paths to contents. It returns the fixture
// root (the directory playing the repo root), so scans never touch the real
// plugin/.
func writeFixtureSurface(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		abs := filepath.Join(root, OperatingSurfaceRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("laying fixture surface: %v", err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatalf("laying fixture surface: %v", err)
		}
	}
	return root
}

func TestScanOperatingSurfaceCleanFixture(t *testing.T) {
	root := writeFixtureSurface(t, map[string]string{
		"skills/example/SKILL.md": "Run `glassfrog proposal list` to situate; the write-safety gate\n(plugin/hooks/gate.sh) asks before a governance write.\n",
		"hooks/gate.sh":           "#!/usr/bin/env bash\n# asks before a governance write\n",
	})
	scan, err := ScanOperatingSurface(root)
	if err != nil {
		t.Fatalf("clean fixture surface errored: %v", err)
	}
	if len(scan.Violations) != 0 {
		t.Fatalf("clean fixture surface reported violations:\n  - %s", strings.Join(scan.Violations, "\n  - "))
	}
	if len(scan.Files) != 2 {
		t.Fatalf("walk found %d files, want 2: %v", len(scan.Files), scan.Files)
	}
}

// TestScanOperatingSurfaceSeededViolations seeds one file with a Family A id,
// a Family B phrase, and a dangling in-surface path, and requires all three in
// one run — never first-only — each naming file, line, matched text, and the
// reachable remedy.
func TestScanOperatingSurfaceSeededViolations(t *testing.T) {
	root := writeFixtureSurface(t, map[string]string{
		"skills/leaky/SKILL.md": "the gated write path (067) does the create\n" +
			"a drift guard in the source repository watches this\n" +
			"see plugin/hooks/does-not-exist.txt for the registry\n",
	})
	scan, err := ScanOperatingSurface(root)
	if err != nil {
		t.Fatalf("seeded fixture surface errored: %v", err)
	}
	joined := strings.Join(scan.Violations, "\n")
	wants := []struct{ name, fragment string }{
		{"family A spec-number id", `plugin/skills/leaky/SKILL.md:1: forbidden reference "067" (family A — resolvable reference:`},
		{"family A remedy", "Remedy: replace with the in-plugin component name, or remove the reference."},
		{"family B drift-guard phrase", `forbidden reference "drift guard" (family B — repo-machinery phrase:`},
		{"family B source-repository phrase", `forbidden reference "source repository" (family B — repo-machinery phrase:`},
		{"family B remedy", "Remedy: reword to state the rule through the surface's own consequences, or remove the mention."},
		{"dangling path", `plugin/skills/leaky/SKILL.md:3: dangling in-surface path "plugin/hooks/does-not-exist.txt"`},
		{"dangling remedy", "Remedy: correct the path to the existing in-surface file, or remove the reference."},
	}
	for _, want := range wants {
		if !strings.Contains(joined, want.fragment) {
			t.Errorf("the run does not report the %s (want fragment %q); got:\n%s", want.name, want.fragment, joined)
		}
	}
	if len(scan.Violations) < 4 {
		t.Errorf("want every violation in one run (>= 4), got %d:\n%s", len(scan.Violations), joined)
	}
}

// TestScanOperatingSurfaceDerivedWalk adds a file to an already-clean fixture
// surface and requires the next scan to cover it with no registration step —
// the checked set is derived by walking, never enumerated.
func TestScanOperatingSurfaceDerivedWalk(t *testing.T) {
	root := writeFixtureSurface(t, map[string]string{
		"skills/example/SKILL.md": "ask the CLI for the exact flags\n",
	})
	added := filepath.Join(root, OperatingSurfaceRoot, "skills", "future", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(added), 0o755); err != nil {
		t.Fatalf("adding future surface file: %v", err)
	}
	if err := os.WriteFile(added, []byte("the future path leaks (067)\n"), 0o644); err != nil {
		t.Fatalf("adding future surface file: %v", err)
	}
	scan, err := ScanOperatingSurface(root)
	if err != nil {
		t.Fatalf("scan errored: %v", err)
	}
	found := false
	for _, f := range scan.Files {
		if f == "plugin/skills/future/SKILL.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the added file is not among the files checked: %v", scan.Files)
	}
	joined := strings.Join(scan.Violations, "\n")
	if !strings.Contains(joined, "plugin/skills/future/SKILL.md:1") {
		t.Fatalf("the added file's violation was not caught without registration; got:\n%s", joined)
	}
}

// TestScanOperatingSurfaceEmptyOrMissing pins the loud failure: zero files or
// a missing surface directory is an error naming the condition — no skip path,
// no warning tier, no vacuous pass.
func TestScanOperatingSurfaceEmptyOrMissing(t *testing.T) {
	empty := t.TempDir()
	if err := os.MkdirAll(filepath.Join(empty, OperatingSurfaceRoot), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ScanOperatingSurface(empty); err == nil || !strings.Contains(err.Error(), "missing or empty") {
		t.Errorf("an empty surface must fail loudly (want %q), got err=%v", "missing or empty", err)
	}
	missing := t.TempDir()
	if _, err := ScanOperatingSurface(missing); err == nil || !strings.Contains(err.Error(), "missing or empty") {
		t.Errorf("a missing surface must fail loudly (want %q), got err=%v", "missing or empty", err)
	}
}

// TestScanOperatingSurfaceKnownSafeFile scans a fixture file carrying every
// known-safe token class and requires zero violations with the content left
// intact — the check reads, never rewrites.
func TestScanOperatingSurfaceKnownSafeFile(t *testing.T) {
	content := strings.Join(knownSafeTokens, "\n") + "\n"
	root := writeFixtureSurface(t, map[string]string{"skills/safe/SKILL.md": content})
	scan, err := ScanOperatingSurface(root)
	if err != nil {
		t.Fatalf("known-safe fixture errored: %v", err)
	}
	if len(scan.Violations) != 0 {
		t.Fatalf("known-safe tokens tripped the check:\n  - %s", strings.Join(scan.Violations, "\n  - "))
	}
	after, err := os.ReadFile(filepath.Join(root, OperatingSurfaceRoot, "skills", "safe", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != content {
		t.Fatal("the scan rewrote the surface file it read")
	}
}
