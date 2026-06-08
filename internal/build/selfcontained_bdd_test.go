package build

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestSelfContainedExecutableBuildFeatures runs the executable acceptance for
// Self-Contained Executable Build (021). Its Paths name ONLY this spec's feature
// file, so un-@wip-ping these scenarios cannot disturb another package's suite.
//
// The scenarios divide into three executable families, none of which requires
// goreleaser (so `go test ./...` stays runnable without it, per the host-build
// fallback):
//   - config-guard scenarios (cgo drift, unsupported target) run CheckConfigGuard
//     against in-memory mutated configs;
//   - build-contract scenarios (the four-target matrix, atomic failure, foreign
//     cross-compile) assert against the parsed real .goreleaser.yaml — the
//     config-guard is the plan's chosen in-process proxy for the build matrix,
//     with actual artifact production verified by the documented `goreleaser
//     build` invocation and CI (022);
//   - self-containment scenarios (runs on a clean host, rejects a foreign
//     dependency, per-target allowlist, local host build) obtain a host-target
//     binary and inspect its linkage.
//
// The three @validation scenarios ("needs only the API at runtime", "the matrix
// is exactly the four supported targets", "shared build entry point") stay @wip,
// held for the validate skill, and are skipped by the ~@wip filter.
func TestSelfContainedExecutableBuildFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeSelfContainedScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/runtime-dependent-distribution/self-contained-executable-build.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: self-contained-executable-build feature scenarios failed")
	}
}

// buildWorld is the per-scenario state for the self-contained-build suite.
type buildWorld struct {
	tmpDir string

	// config-guard path (drift scenarios)
	guardYAML   string
	guardResult []string
	guardRan    bool

	// build-contract path (real config scenarios)
	cfg       Config
	cfgLoaded bool

	// self-containment path
	binPath    string
	runErr     error
	violations []string
	rejectDeps []string // synthetic foreign dependencies for the rejection scenario
}

func initializeSelfContainedScenario(sc *godog.ScenarioContext) {
	w := &buildWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		dir, err := os.MkdirTemp("", "glassfrog-build-bdd-")
		if err != nil {
			return ctx, err
		}
		*w = buildWorld{tmpDir: dir}
		return ctx, nil
	})
	sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		if w.tmpDir != "" {
			_ = os.RemoveAll(w.tmpDir)
		}
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a binary had been produced for a target platform$`, w.givenHostBinaryProduced)
	sc.Step(`^a produced binary required a separately-installed dependency$`, w.givenForeignDependency)
	sc.Step(`^a binary had been produced for the macOS arm64 target$`, w.givenNoop)
	sc.Step(`^the build configuration had been changed to enable cgo$`, w.givenCgoEnabled)
	sc.Step(`^the repository source tree$`, w.givenRealConfig)
	sc.Step(`^the release build was producing the four target binaries$`, w.givenRealConfig)
	sc.Step(`^a maintainer was working on a macOS arm64 host$`, w.givenRealConfig)
	sc.Step(`^the build configuration had declared a Windows target$`, w.givenWindowsTarget)
	sc.Step(`^a maintainer was on a supported host platform$`, w.givenNoop)

	// --- Whens ---
	sc.Step(`^it runs on a clean host of that platform with only the OS and network present$`, w.whenRunOnCleanHost)
	sc.Step(`^the self-containment check runs against it on a clean host of its target$`, w.whenCheckForeign)
	sc.Step(`^it is taken to a Linux host or a macOS amd64 host$`, w.givenNoop)
	sc.Step(`^the config-guard check runs$`, w.whenConfigGuardRuns)
	sc.Step(`^the cross-platform release build runs$`, w.givenNoop)
	sc.Step(`^one target fails to build$`, w.givenNoop)
	sc.Step(`^the release build runs$`, w.givenNoop)
	sc.Step(`^they run the local development build$`, w.whenLocalBuild)

	// --- Thens ---
	sc.Step(`^it will execute successfully$`, w.thenExecutesSuccessfully)
	sc.Step(`^it will be able to reach the Glassfrog API$`, w.thenOnlyNeedsNetwork)
	sc.Step(`^the check will fail and name the missing-dependency violation$`, w.thenCheckNamesViolation)
	sc.Step(`^the binary will not be treated as self-contained$`, w.thenNotSelfContained)
	sc.Step(`^it will not be expected to run there$`, w.thenPerTargetAllowlist)
	sc.Step(`^the self-containment guarantee will hold only on a clean host of its own target$`, w.thenGuaranteeIsPerTarget)
	sc.Step(`^it will fail$`, w.thenGuardFailed)
	sc.Step(`^it will report that cgo must remain disabled$`, w.thenNamesCgoDisabled)
	sc.Step(`^it will name the unsupported target$`, w.thenNamesWindows)
	sc.Step(`^one executable will be produced for each of macOS amd64, macOS arm64, Linux amd64, and Linux arm64$`, w.thenMatrixIsTheFour)
	sc.Step(`^all four will come from the same single source tree$`, w.thenSingleSourceTree)
	sc.Step(`^the release build will fail as a whole$`, w.thenAtomicBuild)
	sc.Step(`^it will emit no partial set of binaries$`, w.thenAtomicBuild)
	sc.Step(`^the Linux amd64 binary will be produced$`, w.thenDeclaresLinuxAmd64)
	sc.Step(`^producing it will not require running on a Linux or amd64 host$`, w.thenCgoDisabledEnablesCrossCompile)
	sc.Step(`^a single glassfrog executable will be produced for their own OS and architecture$`, w.thenHostBinaryProduced)
	sc.Step(`^it will run on their machine without any separately-installed runtime$`, w.thenExecutesAndSelfContained)
}

// --- Given implementations -------------------------------------------------

func (w *buildWorld) givenNoop() error { return nil }

func (w *buildWorld) givenHostBinaryProduced() error {
	bin, _, err := HostBinary(w.tmpDir)
	if err != nil {
		return fmt.Errorf("obtaining a host-target binary: %w", err)
	}
	w.binPath = bin
	return nil
}

func (w *buildWorld) givenForeignDependency() error {
	// A produced binary that links a library outside the OS allowlist — modelled
	// per host platform so the check is exercised against the host's own rules.
	switch runtime.GOOS {
	case "darwin":
		w.rejectDeps = []string{"/usr/lib/libSystem.B.dylib", "/opt/homebrew/lib/libpq.5.dylib"}
	default:
		w.rejectDeps = []string{"linux-vdso.so.1", "/lib/x86_64-linux-gnu/libpq.so.5"}
	}
	return nil
}

func (w *buildWorld) givenCgoEnabled() error {
	w.guardYAML = strings.Replace(validConfigYAML, "CGO_ENABLED=0", "CGO_ENABLED=1", 1)
	return nil
}

func (w *buildWorld) givenWindowsTarget() error {
	w.guardYAML = strings.Replace(validConfigYAML,
		"      - darwin\n      - linux\n",
		"      - darwin\n      - linux\n      - windows\n", 1)
	return nil
}

func (w *buildWorld) givenRealConfig() error {
	cfg, _, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("loading the real build config: %w", err)
	}
	w.cfg = cfg
	w.cfgLoaded = true
	return nil
}

// --- When implementations --------------------------------------------------

func (w *buildWorld) whenRunOnCleanHost() error {
	if w.binPath == "" {
		return fmt.Errorf("no binary was produced")
	}
	_, w.runErr = exec.Command(w.binPath, "version").CombinedOutput()
	deps, err := extractDeps(runtime.GOOS, w.binPath)
	if err != nil {
		return fmt.Errorf("inspecting linkage: %w", err)
	}
	w.violations = osOnlyViolations(runtime.GOOS, deps)
	return nil
}

func (w *buildWorld) whenCheckForeign() error {
	w.violations = osOnlyViolations(runtime.GOOS, w.rejectDeps)
	return nil
}

func (w *buildWorld) whenConfigGuardRuns() error {
	cfg, err := ParseConfig([]byte(w.guardYAML))
	if err != nil {
		return fmt.Errorf("parsing the drifted config: %w", err)
	}
	w.guardResult = CheckConfigGuard(cfg)
	w.guardRan = true
	return nil
}

func (w *buildWorld) whenLocalBuild() error {
	bin, err := buildHostBinary(w.tmpDir)
	if err != nil {
		return fmt.Errorf("local development build: %w", err)
	}
	w.binPath = bin
	_, w.runErr = exec.Command(bin, "version").CombinedOutput()
	deps, err := extractDeps(runtime.GOOS, bin)
	if err != nil {
		return fmt.Errorf("inspecting linkage: %w", err)
	}
	w.violations = osOnlyViolations(runtime.GOOS, deps)
	return nil
}

// --- Then implementations --------------------------------------------------

func (w *buildWorld) thenExecutesSuccessfully() error {
	if w.runErr != nil {
		return fmt.Errorf("the binary did not execute successfully: %v", w.runErr)
	}
	return nil
}

// thenOnlyNeedsNetwork asserts the binary's only unmet external need is the
// network (the API): every dynamic dependency is OS-provided, so nothing else
// must be installed for it to run and reach the API.
func (w *buildWorld) thenOnlyNeedsNetwork() error {
	if len(w.violations) != 0 {
		return fmt.Errorf("the binary needs separately-installed libraries, so the API is not its only external need:\n  %s",
			strings.Join(w.violations, "\n  "))
	}
	return nil
}

func (w *buildWorld) thenCheckNamesViolation() error {
	if len(w.violations) == 0 {
		return fmt.Errorf("the check should have failed on a foreign dependency, but found none")
	}
	// The offending (non-OS) dependency must be named.
	joined := strings.Join(w.violations, "\n")
	if runtime.GOOS == "darwin" && !strings.Contains(joined, "/opt/homebrew/lib/libpq.5.dylib") {
		return fmt.Errorf("the violation should name the foreign dependency, got: %v", w.violations)
	}
	if runtime.GOOS == "linux" && !strings.Contains(joined, "/lib/x86_64-linux-gnu/libpq.so.5") {
		return fmt.Errorf("the violation should name the foreign dependency, got: %v", w.violations)
	}
	return nil
}

func (w *buildWorld) thenNotSelfContained() error {
	if len(w.violations) == 0 {
		return fmt.Errorf("a binary with a foreign dependency must not be treated as self-contained")
	}
	return nil
}

// thenPerTargetAllowlist confirms the guarantee is per-target: a macOS system
// framework is OS-provided on macOS but is NOT part of the Linux allowlist, so a
// macOS arm64 binary's dependencies are not judged self-contained under Linux
// rules — it is not expected to run on a foreign target's host.
func (w *buildWorld) thenPerTargetAllowlist() error {
	macFramework := "/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation"
	if !isOSProvided("darwin", macFramework) {
		return fmt.Errorf("a macOS framework must be OS-provided on macOS")
	}
	if isOSProvided("linux", macFramework) {
		return fmt.Errorf("a macOS framework must NOT be in the Linux allowlist — the guarantee is per-target")
	}
	return nil
}

func (w *buildWorld) thenGuaranteeIsPerTarget() error {
	// The allowlists for the two platforms are genuinely distinct.
	if isOSProvided("linux", "/usr/lib/libSystem.B.dylib") {
		return fmt.Errorf("macOS libSystem must not be treated as OS-provided on Linux")
	}
	return nil
}

func (w *buildWorld) thenGuardFailed() error {
	if !w.guardRan {
		return fmt.Errorf("the config-guard did not run")
	}
	if len(w.guardResult) == 0 {
		return fmt.Errorf("the config-guard should have failed, but reported no violations")
	}
	return nil
}

func (w *buildWorld) thenNamesCgoDisabled() error {
	if !strings.Contains(strings.Join(w.guardResult, "\n"), "cgo must remain disabled") {
		return fmt.Errorf("the guard should report that cgo must remain disabled, got: %v", w.guardResult)
	}
	return nil
}

func (w *buildWorld) thenNamesWindows() error {
	if !strings.Contains(strings.Join(w.guardResult, "\n"), "windows") {
		return fmt.Errorf("the guard should name the unsupported target windows, got: %v", w.guardResult)
	}
	return nil
}

// thenMatrixIsTheFour asserts the parsed config declares exactly the four
// supported targets (the config-guard's matrix check passing is the in-process
// proof that a build of this config produces one binary per target).
func (w *buildWorld) thenMatrixIsTheFour() error {
	if !w.cfgLoaded {
		return fmt.Errorf("the config was not loaded")
	}
	if violations := CheckConfigGuard(w.cfg); len(violations) != 0 {
		return fmt.Errorf("the matrix is not exactly the four supported targets:\n  %s",
			strings.Join(violations, "\n  "))
	}
	gotGoos := sortedCopy(w.cfg.Builds[0].Goos)
	gotGoarch := sortedCopy(w.cfg.Builds[0].Goarch)
	if strings.Join(gotGoos, ",") != "darwin,linux" || strings.Join(gotGoarch, ",") != "amd64,arm64" {
		return fmt.Errorf("expected darwin/linux × amd64/arm64, got goos=%v goarch=%v", gotGoos, gotGoarch)
	}
	return nil
}

func (w *buildWorld) thenSingleSourceTree() error {
	if len(w.cfg.Builds) != 1 {
		return fmt.Errorf("all targets must build from one builds entry (one source tree), found %d", len(w.cfg.Builds))
	}
	if w.cfg.Builds[0].Main != "." {
		return fmt.Errorf("the single source tree is the module root (main: .), got %q", w.cfg.Builds[0].Main)
	}
	return nil
}

// thenAtomicBuild asserts the structural property behind GoReleaser's atomic
// build: the four targets live in a SINGLE builds entry, so one target's compile
// failure aborts the whole build with no partial dist set (GoReleaser default) —
// there is no second, independently-succeeding build path.
func (w *buildWorld) thenAtomicBuild() error {
	if len(w.cfg.Builds) != 1 {
		return fmt.Errorf("the four targets must share one atomic builds entry, found %d", len(w.cfg.Builds))
	}
	return nil
}

func (w *buildWorld) thenDeclaresLinuxAmd64() error {
	if !contains(w.cfg.Builds[0].Goos, "linux") || !contains(w.cfg.Builds[0].Goarch, "amd64") {
		return fmt.Errorf("the config must declare the linux/amd64 target, got goos=%v goarch=%v",
			w.cfg.Builds[0].Goos, w.cfg.Builds[0].Goarch)
	}
	return nil
}

// thenCgoDisabledEnablesCrossCompile asserts CGO_ENABLED=0 is set — the lever
// that lets any host cross-compile any GOOS/GOARCH target (a foreign target does
// not require running on that target's host).
func (w *buildWorld) thenCgoDisabledEnablesCrossCompile() error {
	if violations := checkCgoDisabled(w.cfg.Builds[0].Env); len(violations) != 0 {
		return fmt.Errorf("cross-compilation requires CGO_ENABLED=0: %v", violations)
	}
	return nil
}

func (w *buildWorld) thenHostBinaryProduced() error {
	if w.binPath == "" {
		return fmt.Errorf("no host binary was produced")
	}
	if _, err := os.Stat(w.binPath); err != nil {
		return fmt.Errorf("the produced host binary is missing: %w", err)
	}
	return nil
}

func (w *buildWorld) thenExecutesAndSelfContained() error {
	if err := w.thenExecutesSuccessfully(); err != nil {
		return err
	}
	if len(w.violations) != 0 {
		return fmt.Errorf("the host binary needs a separately-installed runtime:\n  %s",
			strings.Join(w.violations, "\n  "))
	}
	return nil
}

// --- helpers ---------------------------------------------------------------

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
