package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// Offline render-and-inspect for the 036 Homebrew formula (spec 036 T003).
//
// A real `brew install` from a live tap cannot run in a unit test (it needs a
// published release + a network download), so — exactly as 021 renders the
// build matrix and 022 proves the workflow against parsed artifacts — this test
// drives GoReleaser OFFLINE: `goreleaser release --snapshot --clean
// --skip=publish` renders Formula/glassfrog.rb into a dist/ without tagging,
// uploading, or pushing. The test then asserts the formula's shape and — the
// hard contract — that each sha256 the formula records equals the matching line
// in the snapshot's checksums.txt. That is the in-process proof of the feature
// scenario "The published formula's checksums match the release's checksums
// file": the formula references the exact bytes the release publishes.
//
// The render is gated on the `goreleaser` binary being available; PR validation
// runs `go test ./...` without it, so the test skips cleanly there (the render
// also runs in CI's release path and is a documented local invocation). The
// render is isolated into a temp dist/ so it never clobbers the shared dist/
// the self-containment tests read.

var (
	formulaOnce   sync.Once
	formulaResult renderedFormula
	formulaErr    error
)

// renderedFormula is the parsed outcome of one offline snapshot render.
type renderedFormula struct {
	body      string            // the rendered Formula/glassfrog.rb contents
	version   string            // the formula's `version "..."` field
	shas      map[string]string // archive filename -> sha256 recorded in the formula
	checksums map[string]string // archive filename -> sha256 from the snapshot checksums.txt
}

// goreleaserBin reports the goreleaser executable path and whether it is
// available. Render-dependent tests skip when it is not.
func goreleaserBin() (string, bool) {
	if p, err := exec.LookPath("goreleaser"); err == nil {
		return p, true
	}
	return "", false
}

// getRenderedFormula renders the formula at most once per test binary (the
// render is expensive — it cross-compiles the four targets). It returns
// available=false when goreleaser is absent so callers can skip; a render or
// parse failure is returned as err.
func getRenderedFormula() (rf renderedFormula, err error, available bool) {
	bin, ok := goreleaserBin()
	if !ok {
		return renderedFormula{}, nil, false
	}
	formulaOnce.Do(func() { formulaResult, formulaErr = doRenderFormula(bin) })
	return formulaResult, formulaErr, true
}

// doRenderFormula invokes goreleaser against the REAL .goreleaser.yaml (a copy
// with only an isolating `dist:` override appended), parses the rendered formula
// and the snapshot checksums file, and returns them paired by archive filename.
func doRenderFormula(bin string) (renderedFormula, error) {
	root, err := RepoRoot()
	if err != nil {
		return renderedFormula{}, err
	}

	// Render into an isolated temp dist so the shared dist/ the self-containment
	// tests read is left untouched. GoReleaser takes the dist directory from the
	// config's top-level `dist:` key, so the render config is the real config
	// verbatim plus that one absolute override — the formula/brews/release
	// content under test is exactly what ships.
	workDir, err := os.MkdirTemp("", "glassfrog-formula-render-*")
	if err != nil {
		return renderedFormula{}, err
	}
	defer os.RemoveAll(workDir)
	distDir := filepath.Join(workDir, "dist")

	realCfg, err := os.ReadFile(filepath.Join(root, ConfigFileName))
	if err != nil {
		return renderedFormula{}, fmt.Errorf("reading the real %s: %w", ConfigFileName, err)
	}
	renderCfg := filepath.Join(workDir, "goreleaser-render.yaml")
	if err := os.WriteFile(renderCfg, []byte(string(realCfg)+"\ndist: "+distDir+"\n"), 0o644); err != nil {
		return renderedFormula{}, err
	}

	cmd := exec.Command(bin, "release", "--snapshot", "--clean", "--skip=publish", "-f", renderCfg)
	cmd.Dir = root
	// The brews repository.token templates {{ .Env.HOMEBREW_TAP_TOKEN }}; supply a
	// dummy so the offline render (which never pushes) can evaluate the template.
	cmd.Env = append(os.Environ(), "HOMEBREW_TAP_TOKEN=offline-render-dummy")
	if out, runErr := cmd.CombinedOutput(); runErr != nil {
		return renderedFormula{}, fmt.Errorf("goreleaser snapshot render failed: %v\n%s", runErr, out)
	}

	body, err := os.ReadFile(filepath.Join(distDir, "homebrew", "Formula", "glassfrog.rb"))
	if err != nil {
		return renderedFormula{}, fmt.Errorf("reading the rendered formula: %w", err)
	}
	checksums, err := parseSnapshotChecksums(distDir)
	if err != nil {
		return renderedFormula{}, err
	}

	rf := renderedFormula{body: string(body), shas: parseFormulaSHAs(string(body)), checksums: checksums}
	if m := regexp.MustCompile(`version "([^"]+)"`).FindStringSubmatch(rf.body); m != nil {
		rf.version = m[1]
	}
	return rf, nil
}

// formulaURLSHA pairs each download url (directly followed by its sha256 line in
// the rendered formula) so the recorded checksum can be keyed by archive name.
var formulaURLSHA = regexp.MustCompile(`url "([^"]+)"\s+sha256 "([0-9a-f]{64})"`)

// parseFormulaSHAs maps each archive filename referenced in the formula to the
// sha256 the formula records for it.
func parseFormulaSHAs(body string) map[string]string {
	out := map[string]string{}
	for _, m := range formulaURLSHA.FindAllStringSubmatch(body, -1) {
		out[filepath.Base(m[1])] = m[2]
	}
	return out
}

// parseSnapshotChecksums reads dist/*checksums.txt and maps each archive
// filename to its sha256 (lines are `<sha256>  <filename>`).
func parseSnapshotChecksums(distDir string) (map[string]string, error) {
	matches, err := filepath.Glob(filepath.Join(distDir, "*checksums.txt"))
	if err != nil {
		return nil, err
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("expected exactly one checksums file in %s, found %d", distDir, len(matches))
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 {
			out[fields[1]] = fields[0]
		}
	}
	return out, nil
}

// TestFormulaRender_OfflineShapeAndChecksums renders the formula offline and
// asserts: the class/desc/version, all four per-platform url+sha256 blocks, the
// install and test stanzas, and — the hard contract — that every sha256 the
// formula records equals the matching line in the snapshot checksums file.
func TestFormulaRender_OfflineShapeAndChecksums(t *testing.T) {
	rf, err, available := getRenderedFormula()
	if !available {
		t.Skip("goreleaser not on PATH — skipping the offline formula render-and-inspect")
	}
	if err != nil {
		t.Fatalf("rendering the formula offline: %v", err)
	}

	// Structural shape (the real GoReleaser output: on_macos/on_linux blocks with
	// Hardware::CPU branches, a top-level desc/version, and the install/test
	// stanzas). The illustrative on_arm/on_intel in the interface is GoReleaser's
	// to render; the binding shape is the class, both OS blocks, install + test.
	for _, want := range []string{
		"class Glassfrog < Formula",
		"on_macos do",
		"on_linux do",
		`bin.install "glassfrog"`,
		`system "#{bin}/glassfrog", "version"`,
	} {
		if !strings.Contains(rf.body, want) {
			t.Errorf("rendered formula must contain %q; formula:\n%s", want, rf.body)
		}
	}
	if rf.version == "" {
		t.Errorf("rendered formula must carry a populated version field")
	}

	// All four platform url+sha256 blocks are present.
	if len(rf.shas) != 4 {
		t.Fatalf("expected 4 url+sha256 blocks, got %d: %v", len(rf.shas), rf.shas)
	}
	for _, target := range []string{"darwin_amd64", "darwin_arm64", "linux_amd64", "linux_arm64"} {
		found := false
		for filename := range rf.shas {
			if strings.Contains(filename, target) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("rendered formula missing a url+sha256 block for %s (archives: %v)", target, keysOf(rf.shas))
		}
	}

	// Hard contract: each recorded sha256 equals the checksums-file entry for the
	// same archive — the formula references the exact published bytes.
	for filename, formulaSHA := range rf.shas {
		wantSHA, ok := rf.checksums[filename]
		if !ok {
			t.Errorf("formula references %s but it is absent from the checksums file", filename)
			continue
		}
		if formulaSHA != wantSHA {
			t.Errorf("sha256 mismatch for %s: formula records %s, checksums file has %s", filename, formulaSHA, wantSHA)
		}
	}
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
