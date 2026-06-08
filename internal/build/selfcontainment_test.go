package build

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// skipUnsupportedLinkagePlatform skips a test on platforms where linkage
// inspection is not supported. extractDeps shells out to otool (darwin) / ldd
// (linux) and returns an error on any other GOOS, so the self-containment
// checks — which build and inspect a real binary — only run on the two
// supported build targets. The pure-logic tests (parseOtool/parseLdd,
// osOnlyViolations, the config-guard) carry no such guard and run everywhere.
// Mirrors the Windows skips already in internal/auth/auth_test.go.
func skipUnsupportedLinkagePlatform(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("linkage inspection unsupported on %s — self-containment targets are darwin/linux", runtime.GOOS)
	}
}

// TestSelfContainment_HostBinary is the self-containment verification on the
// host target: it obtains a host-target glassfrog (dist-artifact-preferred,
// host-build fallback), executes `glassfrog version` asserting exit 0, then
// inspects the binary's dynamic-library linkage against the per-platform OS-only
// allowlist. This is CONSTITUTION XII made executable, proven on at least the
// host target (cross-target breadth is 022's concern).
//
// The execute probe runs `version` — a no-network, version-class command — so
// the check makes no API call and proves only that the loader and runtime start
// on a clean host.
func TestSelfContainment_HostBinary(t *testing.T) {
	skipUnsupportedLinkagePlatform(t)
	bin, source, err := HostBinary(t.TempDir())
	if err != nil {
		t.Fatalf("obtaining a host-target binary: %v", err)
	}
	t.Logf("verifying host binary from %s: %s", source, bin)

	// Execute check: the binary runs on this host and exits 0.
	if out, runErr := exec.Command(bin, "version").CombinedOutput(); runErr != nil {
		t.Fatalf("glassfrog version did not run cleanly: %v\n%s", runErr, out)
	}

	// Linkage check: every dynamic dependency is OS-provided.
	deps, err := extractDeps(runtime.GOOS, bin)
	if err != nil {
		t.Fatalf("inspecting linkage: %v", err)
	}
	if violations := osOnlyViolations(runtime.GOOS, deps); len(violations) != 0 {
		t.Fatalf("binary links libraries outside the OS allowlist (not self-contained):\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// TestSelfContainment_HostBuildFallback proves the host-build fallback engages
// without goreleaser: it builds the host binary directly (the path taken on a
// clean checkout with no dist/) and confirms it runs. This is the
// "runs under `go test ./...` without goreleaser installed" acceptance.
func TestSelfContainment_HostBuildFallback(t *testing.T) {
	skipUnsupportedLinkagePlatform(t)
	bin, err := buildHostBinary(t.TempDir())
	if err != nil {
		t.Fatalf("host build fallback failed: %v", err)
	}
	if out, runErr := exec.Command(bin, "version").CombinedOutput(); runErr != nil {
		t.Fatalf("fallback-built binary did not run: %v\n%s", runErr, out)
	}
	deps, err := extractDeps(runtime.GOOS, bin)
	if err != nil {
		t.Fatalf("inspecting fallback binary linkage: %v", err)
	}
	if violations := osOnlyViolations(runtime.GOOS, deps); len(violations) != 0 {
		t.Fatalf("fallback binary is not self-contained:\n  %s", strings.Join(violations, "\n  "))
	}
}

// TestSelfContainment_RejectsForeignDependency exercises the failure the check
// exists to catch: a binary that links a library outside the OS allowlist must
// be flagged, naming the offending dependency. Driven against synthetic
// dependency lists so it is host-independent (covers both platforms' allowlists).
func TestSelfContainment_RejectsForeignDependency(t *testing.T) {
	cases := []struct {
		name      string
		goos      string
		deps      []string
		wantNamed string
	}{
		{
			name:      "macOS Homebrew library is rejected",
			goos:      "darwin",
			deps:      []string{"/usr/lib/libSystem.B.dylib", "/opt/homebrew/lib/libpq.5.dylib"},
			wantNamed: "/opt/homebrew/lib/libpq.5.dylib",
		},
		{
			name:      "Linux shared library outside the loader is rejected",
			goos:      "linux",
			deps:      []string{"linux-vdso.so.1", "/lib/x86_64-linux-gnu/libpq.so.5", "/lib64/ld-linux-x86-64.so.2"},
			wantNamed: "/lib/x86_64-linux-gnu/libpq.so.5",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			violations := osOnlyViolations(tc.goos, tc.deps)
			if len(violations) == 0 {
				t.Fatalf("expected the check to reject a foreign dependency, but it passed")
			}
			if !strings.Contains(strings.Join(violations, "\n"), tc.wantNamed) {
				t.Fatalf("the violation must name %q, got: %v", tc.wantNamed, violations)
			}
		})
	}
}

// TestSelfContainment_PerPlatformAllowlist confirms the allowlist is per-target,
// not universal: the libraries an OS provides differ, so self-containment holds
// only on a clean host of the binary's own target. The same Linux library path
// that is a violation on macOS is judged by Linux rules on Linux, and macOS
// system frameworks are not part of the Linux allowlist.
func TestSelfContainment_PerPlatformAllowlist(t *testing.T) {
	// macOS: system libraries and frameworks are OS-provided.
	if v := osOnlyViolations("darwin", []string{
		"/usr/lib/libSystem.B.dylib",
		"/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation",
	}); len(v) != 0 {
		t.Fatalf("macOS system libraries must be allowed, got violations: %v", v)
	}
	// Linux: only the loader/vDSO are OS-provided; a macOS framework path is not
	// part of the Linux allowlist (per-target, not universal).
	if v := osOnlyViolations("linux", []string{"linux-vdso.so.1", "/lib64/ld-linux-x86-64.so.2"}); len(v) != 0 {
		t.Fatalf("Linux loader/vDSO must be allowed, got violations: %v", v)
	}
	if v := osOnlyViolations("linux", []string{"/System/Library/Frameworks/Security.framework/Versions/A/Security"}); len(v) == 0 {
		t.Fatalf("a macOS framework is not part of the Linux allowlist and must be flagged on Linux")
	}
}

// TestParseOtool covers the macOS dependency parser against canned output,
// independent of host: the inspected-file line and load-command suffix are
// dropped, leaving only dependency paths.
func TestParseOtool(t *testing.T) {
	out := "dist/glassfrog_darwin_arm64/glassfrog:\n" +
		"\t/usr/lib/libSystem.B.dylib (compatibility version 1.0.0, current version 1.0.0)\n" +
		"\t/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation (compatibility version 150.0.0, current version 2602.0.0)\n"
	deps := parseOtool(out, "dist/glassfrog_darwin_arm64/glassfrog")
	want := []string{
		"/usr/lib/libSystem.B.dylib",
		"/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation",
	}
	if len(deps) != len(want) {
		t.Fatalf("expected %d deps, got %d: %v", len(want), len(deps), deps)
	}
	for i := range want {
		if deps[i] != want[i] {
			t.Fatalf("dep %d = %q, want %q", i, deps[i], want[i])
		}
	}
}

// TestParseLdd covers the Linux dependency parser against canned output for both
// the static case (no deps) and the dynamic case (resolved paths and the
// loader), independent of host.
func TestParseLdd(t *testing.T) {
	if deps := parseLdd("\tstatically linked\n"); len(deps) != 0 {
		t.Fatalf("a statically-linked binary has no deps, got %v", deps)
	}
	if deps := parseLdd("\tnot a dynamic executable\n"); len(deps) != 0 {
		t.Fatalf("a non-dynamic executable has no deps, got %v", deps)
	}
	dyn := "\tlinux-vdso.so.1 (0x00007ffd...)\n" +
		"\tlibpq.so.5 => /lib/x86_64-linux-gnu/libpq.so.5 (0x00007f...)\n" +
		"\t/lib64/ld-linux-x86-64.so.2 (0x00007f...)\n"
	deps := parseLdd(dyn)
	want := []string{"linux-vdso.so.1", "/lib/x86_64-linux-gnu/libpq.so.5", "/lib64/ld-linux-x86-64.so.2"}
	if len(deps) != len(want) {
		t.Fatalf("expected %d deps, got %d: %v", len(want), len(deps), deps)
	}
	for i := range want {
		if deps[i] != want[i] {
			t.Fatalf("dep %d = %q, want %q", i, deps[i], want[i])
		}
	}

	// "name => not found": the loader cannot resolve the dependency. The parser
	// must report the library NAME (the missing dependency), not the literal
	// "not found" — and that name must be flagged as a self-containment violation.
	missing := parseLdd("\tlibcustom.so.1 => not found\n")
	if len(missing) != 1 || missing[0] != "libcustom.so.1" {
		t.Fatalf("a 'not found' dependency must be named by its library, got %v", missing)
	}
	if v := osOnlyViolations("linux", missing); len(v) != 1 || v[0] != "libcustom.so.1" {
		t.Fatalf("an unresolved dependency must be a named violation, got %v", v)
	}
}
