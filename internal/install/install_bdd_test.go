package install

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/cucumber/godog"
)

// TestInstallScriptFeatures runs the executable acceptance for the Install
// Script (027): it drives ../../install.sh end-to-end against an httptest
// server standing in for GitHub Releases, so every scenario runs offline (no
// real network, no real GitHub). Platform is forced via a `uname` shim on PATH
// so a scenario written for "Linux amd64" runs deterministically on any CI
// runner. Its Paths name ONLY this spec's feature file, so the suite reports
// its own independent scenario count. The @validation scenarios stay @wip (held
// for the validate skill) and are skipped by the ~@wip filter.
func TestInstallScriptFeatures(t *testing.T) {
	w := &installWorld{}
	srv := httptest.NewServer(http.HandlerFunc(w.handle))
	defer srv.Close()
	w.server = srv

	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.register(t, sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/runtime-dependent-distribution/install-script.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: install-script feature scenarios failed")
	}
}

// release is one fake published release: the archive bytes and the checksums
// file bytes, keyed by asset filename.
type release struct {
	files map[string][]byte
}

// installWorld is the per-scenario state: the fake releases the server serves,
// the chosen install dir and HOME, the forced platform, and the captured
// outcome of the script run.
type installWorld struct {
	t      *testing.T
	server *httptest.Server

	releases  map[string]*release // by published tag (carries `v`)
	latestTag string              // what /releases/latest redirects to

	installDir string
	home       string
	unameOS    string // uname -s the shim reports
	unameArch  string // uname -m the shim reports
	noTooling  bool   // PATH carries neither downloader nor sha256 utility
	version    string // GLASSFROG_VERSION (empty → unset)

	ran      bool
	exitCode int
	stdout   string
	stderr   string

	mu        sync.Mutex
	requested []string // paths the server received
}

// handle serves the fake GitHub Releases surface: a redirect for
// /releases/latest and the registered assets under /releases/download/<tag>/.
func (w *installWorld) handle(rw http.ResponseWriter, r *http.Request) {
	w.mu.Lock()
	w.requested = append(w.requested, r.URL.Path)
	w.mu.Unlock()

	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, "/releases/latest"):
		rw.Header().Set("Location", "/Luscii/cli-glassfrog/releases/tag/"+w.latestTag)
		rw.WriteHeader(http.StatusFound)
	case strings.Contains(path, "/releases/download/"):
		rest := path[strings.Index(path, "/releases/download/")+len("/releases/download/"):]
		segs := strings.SplitN(rest, "/", 2)
		if len(segs) != 2 {
			http.NotFound(rw, r)
			return
		}
		rel := w.releases[segs[0]]
		if rel == nil {
			http.NotFound(rw, r)
			return
		}
		data, ok := rel.files[segs[1]]
		if !ok {
			http.NotFound(rw, r)
			return
		}
		_, _ = rw.Write(data)
	default:
		http.NotFound(rw, r)
	}
}

// registerRelease builds a release for tag on linux/amd64 (the forced default
// platform) and stores it. When corrupt, the checksums entry carries a hash
// that cannot match the archive.
func (w *installWorld) registerRelease(tag string, corrupt bool) {
	ver := strings.TrimPrefix(tag, "v")
	archiveName := "glassfrog_" + ver + "_linux_amd64.tar.gz"
	checksumsName := "glassfrog_" + ver + "_checksums.txt"
	arc := makeTarGz(w.t, stubBinary(tag))
	hash := sha256Hex(arc)
	if corrupt {
		hash = strings.Repeat("d", 64)
	}
	w.releases[tag] = &release{files: map[string][]byte{
		archiveName:   arc,
		checksumsName: []byte(hash + "  " + archiveName + "\n"),
	}}
}

func (w *installWorld) register(t *testing.T, sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		w.t = t
		w.releases = map[string]*release{}
		w.latestTag = "v1.4.0"
		w.registerRelease("v1.4.0", false)
		w.installDir = filepath.Join(t.TempDir(), "bin")
		w.home = t.TempDir()
		w.unameOS, w.unameArch = "Linux", "x86_64"
		w.noTooling = false
		w.version = ""
		w.ran = false
		w.exitCode = 0
		w.stdout, w.stderr = "", ""
		w.requested = nil
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a clean Linux amd64 host with no "glassfrog" binary installed$`, w.noop)
	sc.Step(`^the latest stable release has the four platform archives and a checksums file attached$`, w.noop)
	sc.Step(`^the chosen install directory is not present in the operator's PATH$`, w.noop)
	sc.Step(`^a host that has neither a usable downloader nor a sha256 utility$`, w.givenNoTooling)
	sc.Step(`^a published release "([^"]*)" exists alongside a newer "([^"]*)"$`, w.givenTwoReleases)
	sc.Step(`^the operator sets a writable custom install directory$`, w.givenCustomDir)
	sc.Step(`^the operator requests a version that has no published release$`, w.givenMissingVersion)
	sc.Step(`^the downloaded archive does not match its entry in the checksums file$`, w.givenCorruptChecksum)
	sc.Step(`^"([^"]*)" "([^"]*)" is already installed at the target location$`, w.givenAlreadyInstalled)
	sc.Step(`^"([^"]*)" is the latest stable release$`, w.givenLatestIs)
	sc.Step(`^the install script is run on a host whose platform is not a supported target$`, w.givenUnsupportedPlatform)

	// --- Whens (all trigger the run; the run is executed once per scenario) ---
	sc.Step(`^the operator runs the install script with no configuration$`, w.runOnce)
	sc.Step(`^platform detection completes$`, w.runOnce)
	sc.Step(`^the script finishes installing the binary$`, w.runOnce)
	sc.Step(`^the operator runs the install script$`, w.runOnce)
	sc.Step(`^the operator runs the script requesting version "([^"]*)"$`, w.runRequestingVersion)
	sc.Step(`^the script runs$`, w.runOnce)
	sc.Step(`^the operator re-runs the script with no version pinned$`, w.runOnce)
	sc.Step(`^the script attempts to download the matching archive$`, w.runOnce)

	// --- Thens ---
	sc.Step(`^it will download the "([^"]*)" archive and the checksums file$`, w.thenDownloadedArchiveAndChecksums)
	sc.Step(`^it will verify the archive against the checksums file$`, w.thenVerified)
	sc.Step(`^it will install the binary into a per-user directory on PATH without sudo$`, w.thenInstalledNoSudo)
	sc.Step(`^it will report the install location and the installed version$`, w.thenReportsLocationAndVersion)
	sc.Step(`^it will report the install location$`, w.thenReportsLocation)
	sc.Step(`^it will print the exact line to add the directory to PATH$`, w.thenPrintsExportLine)
	sc.Step(`^it will not modify any shell profile or environment file$`, w.thenNoProfileTouched)
	sc.Step(`^it will stop with a message naming the detected platform and the supported targets$`, w.thenNamesPlatformAndTargets)
	sc.Step(`^it will install nothing$`, w.thenInstalledNothing)
	sc.Step(`^it will exit with a non-zero status$`, w.thenExitNonZero)
	sc.Step(`^it will stop before downloading anything$`, w.thenNoDownload)
	sc.Step(`^it will report which tool category is missing and what satisfies it$`, w.thenNamesMissingTool)
	sc.Step(`^it will exit with a usage error status$`, w.thenExitUsage)
	sc.Step(`^it will resolve the "([^"]*)" release rather than the latest$`, w.thenResolvedPinned)
	sc.Step(`^it will download, verify, and install the "([^"]*)" binary$`, w.thenInstalledVersion)
	sc.Step(`^it will install the binary into that directory instead of the default$`, w.thenInstalledInConfiguredDir)
	sc.Step(`^it will report that location$`, w.thenReportsLocation)
	sc.Step(`^it will stop with a message naming the requested version$`, w.thenNamesRequestedVersion)
	sc.Step(`^it will stop before installing$`, w.thenInstalledNothing)
	sc.Step(`^no "glassfrog" binary will be written to the target location$`, w.thenNoBinaryWritten)
	sc.Step(`^it will exit with a non-zero status naming the integrity failure$`, w.thenExitIntegrity)
	sc.Step(`^it will install "([^"]*)" over the existing binary at the same location$`, w.thenInstalledVersion)
	sc.Step(`^the installed binary will report "([^"]*)"$`, w.thenInstalledBinaryReports)
}

// --- Given implementations --------------------------------------------------

func (w *installWorld) noop() error           { return nil }
func (w *installWorld) givenNoTooling() error { w.noTooling = true; return nil }
func (w *installWorld) givenCorruptChecksum() error {
	w.registerRelease("v1.4.0", true)
	return nil
}

func (w *installWorld) givenTwoReleases(older, newer string) error {
	w.registerRelease(older, false)
	w.registerRelease(newer, false)
	w.latestTag = newer
	return nil
}

func (w *installWorld) givenCustomDir() error {
	w.installDir = filepath.Join(w.t.TempDir(), "custom", "tools")
	return nil
}

func (w *installWorld) givenMissingVersion() error {
	w.version = "v9.9.9" // deliberately not registered → 404 on download
	return nil
}

func (w *installWorld) givenAlreadyInstalled(name, tag string) error {
	if err := os.MkdirAll(w.installDir, 0o755); err != nil {
		return err
	}
	// Pre-place an older stub so the upgrade can be observed by content.
	return os.WriteFile(filepath.Join(w.installDir, name), stubBinary(tag), 0o755)
}

func (w *installWorld) givenLatestIs(tag string) error {
	w.registerRelease(tag, false)
	w.latestTag = tag
	return nil
}

func (w *installWorld) givenUnsupportedPlatform() error {
	w.unameOS, w.unameArch = "MINGW64_NT-10.0", "x86_64" // a Windows host
	return nil
}

// --- When implementation ----------------------------------------------------

// runOnce execs install.sh exactly once per scenario, capturing its streams and
// exit code. A `uname` shim forces the platform; PATH either prepends the shim
// (normal) or contains only the shim (the missing-tooling case, so curl/wget/
// sha256 are absent).
func (w *installWorld) runOnce() error {
	if w.ran {
		return nil
	}
	w.ran = true

	shimDir := filepath.Join(w.t.TempDir(), "shim")
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		return err
	}
	uname := "#!/bin/sh\ncase \"$1\" in\n -s) echo " + w.unameOS + ";;\n -m) echo " + w.unameArch + ";;\n *) echo unknown;;\nesac\n"
	if err := os.WriteFile(filepath.Join(shimDir, "uname"), []byte(uname), 0o755); err != nil {
		return err
	}

	pathVal := shimDir
	if !w.noTooling {
		pathVal = shimDir + string(os.PathListSeparator) + os.Getenv("PATH")
	}
	env := []string{
		"PATH=" + pathVal,
		"HOME=" + w.home,
		"GLASSFROG_DOWNLOAD_BASE_URL=" + w.server.URL,
		"GLASSFROG_INSTALL_DIR=" + w.installDir,
	}
	if tmp := os.Getenv("TMPDIR"); tmp != "" {
		env = append(env, "TMPDIR="+tmp)
	}
	if w.version != "" {
		env = append(env, "GLASSFROG_VERSION="+w.version)
	}

	cmd := exec.Command("/bin/sh", scriptPath(w.t))
	cmd.Env = env
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	w.exitCode = exitCodeOf(w.t, cmd, err)
	w.stdout, w.stderr = out.String(), errb.String()
	return nil
}

func (w *installWorld) runRequestingVersion(version string) error {
	w.version = version
	return w.runOnce()
}

// --- Then implementations ---------------------------------------------------

func (w *installWorld) installedPath() string { return filepath.Join(w.installDir, "glassfrog") }

func (w *installWorld) requestedPaths() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.requested...)
}

func (w *installWorld) requestedAny(substr string) bool {
	for _, p := range w.requestedPaths() {
		if strings.Contains(p, substr) {
			return true
		}
	}
	return false
}

func (w *installWorld) thenDownloadedArchiveAndChecksums(name string) error {
	// The feature spells the archive with a `<version>` placeholder; resolve it
	// against the latest tag the scenario configured.
	ver := strings.TrimPrefix(w.latestTag, "v")
	wantArchive := strings.ReplaceAll(name, "<version>", ver)
	if !w.requestedAny(wantArchive) {
		return fmt.Errorf("expected a request for %q; saw %v", wantArchive, w.requestedPaths())
	}
	if !w.requestedAny("_checksums.txt") {
		return fmt.Errorf("expected a checksums request; saw %v", w.requestedPaths())
	}
	return nil
}

func (w *installWorld) thenVerified() error {
	if w.exitCode != 0 {
		return fmt.Errorf("install did not succeed (exit %d)\nstderr: %s", w.exitCode, w.stderr)
	}
	if !w.requestedAny("_checksums.txt") {
		return fmt.Errorf("the checksums file was never fetched; saw %v", w.requestedPaths())
	}
	return nil
}

func (w *installWorld) thenInstalledNoSudo() error {
	info, err := os.Stat(w.installedPath())
	if err != nil {
		return fmt.Errorf("binary not installed at %s: %v\nstderr: %s", w.installedPath(), err, w.stderr)
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("installed binary is not executable: mode %v", info.Mode())
	}
	if strings.Contains(w.stdout+w.stderr, "sudo") {
		return fmt.Errorf("the installer must never invoke sudo; output mentions it:\n%s\n%s", w.stdout, w.stderr)
	}
	return nil
}

func (w *installWorld) thenReportsLocationAndVersion() error {
	if err := w.thenReportsLocation(); err != nil {
		return err
	}
	if !strings.Contains(w.stdout, w.latestTag) {
		return fmt.Errorf("stdout should report the installed version %q:\n%s", w.latestTag, w.stdout)
	}
	return nil
}

func (w *installWorld) thenReportsLocation() error {
	if !strings.Contains(w.stdout, w.installDir) {
		return fmt.Errorf("stdout should report the install location %q:\n%s", w.installDir, w.stdout)
	}
	return nil
}

func (w *installWorld) thenPrintsExportLine() error {
	if !strings.Contains(w.stdout, "export PATH=") || !strings.Contains(w.stdout, w.installDir) {
		return fmt.Errorf("stdout should print the exact export line for %q:\n%s", w.installDir, w.stdout)
	}
	return nil
}

func (w *installWorld) thenNoProfileTouched() error {
	for _, f := range []string{".bashrc", ".zshrc", ".profile", ".bash_profile"} {
		if _, err := os.Stat(filepath.Join(w.home, f)); err == nil {
			return fmt.Errorf("the installer must not write %s", f)
		}
	}
	return nil
}

func (w *installWorld) thenNamesPlatformAndTargets() error {
	if !strings.Contains(w.stderr, w.unameOS) {
		return fmt.Errorf("stderr should name the detected platform %q:\n%s", w.unameOS, w.stderr)
	}
	if !strings.Contains(w.stderr, "darwin/linux") || !strings.Contains(w.stderr, "amd64/arm64") {
		return fmt.Errorf("stderr should name the supported targets:\n%s", w.stderr)
	}
	return nil
}

func (w *installWorld) thenInstalledNothing() error {
	if _, err := os.Stat(w.installedPath()); err == nil {
		return fmt.Errorf("nothing should be installed, but %s exists", w.installedPath())
	}
	return nil
}

func (w *installWorld) thenNoBinaryWritten() error { return w.thenInstalledNothing() }

func (w *installWorld) thenExitNonZero() error {
	if w.exitCode == 0 {
		return fmt.Errorf("exit code should be non-zero\nstdout: %s\nstderr: %s", w.stdout, w.stderr)
	}
	return nil
}

func (w *installWorld) thenExitUsage() error {
	if w.exitCode != 2 {
		return fmt.Errorf("exit code = %d, want 2 (usage/environment)\nstderr: %s", w.exitCode, w.stderr)
	}
	return nil
}

func (w *installWorld) thenExitIntegrity() error {
	if w.exitCode != 3 {
		return fmt.Errorf("exit code = %d, want 3 (integrity)\nstderr: %s", w.exitCode, w.stderr)
	}
	if !strings.Contains(strings.ToLower(w.stderr), "integrity") {
		return fmt.Errorf("stderr should name the integrity failure:\n%s", w.stderr)
	}
	return nil
}

func (w *installWorld) thenNoDownload() error {
	for _, p := range w.requestedPaths() {
		if strings.Contains(p, "/releases/") {
			return fmt.Errorf("nothing should have been fetched, but the server saw %q", p)
		}
	}
	return nil
}

func (w *installWorld) thenNamesMissingTool() error {
	low := strings.ToLower(w.stderr)
	if !strings.Contains(low, "downloader") && !strings.Contains(low, "curl") &&
		!strings.Contains(low, "wget") && !strings.Contains(low, "sha256") {
		return fmt.Errorf("stderr should name the missing tool category:\n%s", w.stderr)
	}
	return nil
}

func (w *installWorld) thenResolvedPinned(tag string) error {
	if w.requestedAny("/releases/latest") {
		return fmt.Errorf("a pinned version must not hit the latest redirect; saw %v", w.requestedPaths())
	}
	if !w.requestedAny("/releases/download/" + tag + "/") {
		return fmt.Errorf("expected a download under tag %q; saw %v", tag, w.requestedPaths())
	}
	return nil
}

func (w *installWorld) thenInstalledVersion(tag string) error {
	body, err := os.ReadFile(w.installedPath())
	if err != nil {
		return fmt.Errorf("binary not installed at %s: %v\nstderr: %s", w.installedPath(), err, w.stderr)
	}
	if !strings.Contains(string(body), tag) {
		return fmt.Errorf("installed binary should be the %q build; content: %q", tag, string(body))
	}
	return nil
}

func (w *installWorld) thenInstalledBinaryReports(tag string) error {
	return w.thenInstalledVersion(tag)
}

func (w *installWorld) thenInstalledInConfiguredDir() error {
	if _, err := os.Stat(w.installedPath()); err != nil {
		return fmt.Errorf("binary should be installed into the configured dir %s: %v", w.installDir, err)
	}
	return nil
}

func (w *installWorld) thenNamesRequestedVersion() error {
	if err := w.thenExitNonZero(); err != nil {
		return err
	}
	if !strings.Contains(w.stderr, w.version) {
		return fmt.Errorf("stderr should name the requested version %q:\n%s", w.version, w.stderr)
	}
	return nil
}
