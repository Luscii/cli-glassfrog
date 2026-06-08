package build

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// artifact is the subset of a dist/artifacts.json entry the verification reads.
// Consumers locate binaries via this manifest rather than parsing directory
// names, so GoReleaser's microarch suffix (e.g. _v1, _v8.0) never has to be
// hardcoded.
type artifact struct {
	Name   string `json:"name"`
	Path   string `json:"path"` // relative to the repository root
	Goos   string `json:"goos"`
	Goarch string `json:"goarch"`
	Type   string `json:"type"`
}

// HostBinary returns the path to a host-target glassfrog binary plus a label
// describing where it came from. It prefers a dist/artifacts.json-listed
// artifact for the host GOOS/GOARCH (the real shipped artifact); when no
// manifest entry exists it builds the host target on the fly with
// CGO_ENABLED=0, so `go test ./...` runs without goreleaser installed. The
// freshly-built binary lands under destDir (use t.TempDir()).
func HostBinary(destDir string) (path string, source string, err error) {
	if p, ok, derr := discoverDistBinary(); derr != nil {
		return "", "", derr
	} else if ok {
		return p, "dist/artifacts.json", nil
	}
	p, berr := buildHostBinary(destDir)
	if berr != nil {
		return "", "", berr
	}
	return p, "go build (CGO_ENABLED=0)", nil
}

// discoverDistBinary returns the path to the host-target binary recorded in
// the repository's dist/artifacts.json, if the manifest exists and lists one. A
// missing manifest is not an error — it is the signal to fall back to a host
// build.
func discoverDistBinary() (string, bool, error) {
	root, err := RepoRoot()
	if err != nil {
		return "", false, err
	}
	return discoverDistBinaryIn(root)
}

// discoverDistBinaryIn is the root-parameterized core of discoverDistBinary,
// split out so the manifest parsing and host-target selection can be unit-tested
// against a temp root without depending on a real dist/ produced by goreleaser.
func discoverDistBinaryIn(root string) (string, bool, error) {
	manifestPath := filepath.Join(root, "dist", "artifacts.json")
	raw, err := os.ReadFile(manifestPath)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("reading %s: %w", manifestPath, err)
	}
	var artifacts []artifact
	if err := json.Unmarshal(raw, &artifacts); err != nil {
		return "", false, fmt.Errorf("parsing %s: %w", manifestPath, err)
	}
	distDir := filepath.Join(root, "dist")
	for _, a := range artifacts {
		if a.Type != "Binary" || a.Goos != runtime.GOOS || a.Goarch != runtime.GOARCH {
			continue
		}
		abs := filepath.Join(root, filepath.FromSlash(a.Path))
		// The manifest is goreleaser output, but guard against a malformed or
		// hand-edited path escaping dist/ (e.g. via ".."): the resolved path is
		// executed downstream, so it must stay under <root>/dist/.
		if abs != distDir && !strings.HasPrefix(abs, distDir+string(os.PathSeparator)) {
			continue
		}
		if _, statErr := os.Stat(abs); statErr != nil {
			// Stale/missing entry — keep scanning for another valid host-target
			// Binary rather than abandoning the search on the first miss.
			continue
		}
		return abs, true, nil
	}
	return "", false, nil
}

// buildHostBinary compiles the host-target glassfrog with CGO_ENABLED=0 and
// -trimpath, mirroring the .goreleaser build entry so the fallback proves the
// same self-containment property. It returns the path to the produced binary.
func buildHostBinary(destDir string) (string, error) {
	root, err := RepoRoot()
	if err != nil {
		return "", err
	}
	out := filepath.Join(destDir, "glassfrog")
	cmd := exec.Command("go", "build", "-trimpath", "-o", out, ".")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if combined, runErr := cmd.CombinedOutput(); runErr != nil {
		return "", fmt.Errorf("host build failed: %w\n%s", runErr, combined)
	}
	return out, nil
}
