// Package install holds the exec-tests for the host-side installer
// (../../install.sh, spec 027). The installer is a POSIX shell script, not Go,
// so these tests drive it as a black box: pure functions are exercised by
// sourcing the script and calling them directly; the end-to-end install/verify
// pipeline is exercised by an httptest server standing in for GitHub Releases
// (the GLASSFROG_DOWNLOAD_BASE_URL seam, ADR-4). Nothing here touches the real
// network. Encoding the exact spec-022 asset names as fixtures means a change
// to GoReleaser's name_template breaks these tests rather than silently 404ing
// the live installer.
package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// scriptPath resolves the absolute path to install.sh at the repo root (two
// levels up from internal/install).
func scriptPath(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "install.sh"))
	if err != nil {
		t.Fatalf("resolving install.sh path: %v", err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("install.sh not found at %s: %v", abs, err)
	}
	return abs
}

// sourceAndCall sources install.sh (with the library guard set so main() does
// not run) and evaluates expr, returning stdout, stderr, and the exit code. It
// is the seam for unit-testing the script's pure functions.
func sourceAndCall(t *testing.T, expr string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command("/bin/sh", "-c", ". '"+scriptPath(t)+"'; "+expr)
	cmd.Env = append(os.Environ(), "GLASSFROG_INSTALL_LIB=1")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	code = exitCodeOf(t, cmd, err)
	return out.String(), errb.String(), code
}

// exitCodeOf extracts the process exit code from a finished exec.Cmd. A nil err
// means 0; an *exec.ExitError carries the code; anything else (failure to even
// start) fails the test.
func exitCodeOf(t *testing.T, cmd *exec.Cmd, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	if _, ok := err.(*exec.ExitError); ok {
		return cmd.ProcessState.ExitCode()
	}
	t.Fatalf("running shell: %v", err)
	return -1
}

// sha256Hex returns the lowercase hex sha256 of b — the form GoReleaser writes
// into the checksums file and the form install.sh compares against.
func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// stubBinary builds the bytes of a throwaway `glassfrog` that echoes the tag it
// was built for, so a test can tell which version landed on disk without
// executing a real cross-compiled binary.
func stubBinary(tag string) []byte {
	return []byte("#!/bin/sh\necho \"glassfrog version " + tag + "\"\n")
}

// makeTarGz wraps a single `glassfrog` entry (bin) into a gzip-compressed tar,
// matching the shape GoReleaser's tar.gz archive presents: the binary at the
// archive root under its build name.
func makeTarGz(t *testing.T, bin []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: "glassfrog", Mode: 0o755, Size: int64(len(bin))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("tar header: %v", err)
	}
	if _, err := tw.Write(bin); err != nil {
		t.Fatalf("tar body: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

// TestDetectPlatform pins the platform-mapping table directly (a pure function),
// covering every supported (uname -s, uname -m) pair, the synonym arches, and
// the reject paths — without any network or filesystem effect.
func TestDetectPlatform(t *testing.T) {
	supported := []struct {
		unameS, unameM, want string
	}{
		{"Darwin", "x86_64", "darwin amd64"},
		{"Darwin", "amd64", "darwin amd64"},
		{"Darwin", "arm64", "darwin arm64"},
		{"Darwin", "aarch64", "darwin arm64"},
		{"Linux", "x86_64", "linux amd64"},
		{"Linux", "amd64", "linux amd64"},
		{"Linux", "arm64", "linux arm64"},
		{"Linux", "aarch64", "linux arm64"},
	}
	for _, tc := range supported {
		stdout, _, code := sourceAndCall(t, "detect_platform "+tc.unameS+" "+tc.unameM)
		if code != 0 {
			t.Errorf("detect_platform %s %s: exit %d, want 0", tc.unameS, tc.unameM, code)
		}
		if got := strings.TrimSpace(stdout); got != tc.want {
			t.Errorf("detect_platform %s %s = %q, want %q", tc.unameS, tc.unameM, got, tc.want)
		}
	}

	rejected := []struct{ unameS, unameM string }{
		{"MINGW64_NT-10.0", "x86_64"}, // Windows
		{"Linux", "riscv64"},          // unsupported arch
		{"FreeBSD", "amd64"},          // unsupported OS
	}
	for _, tc := range rejected {
		stdout, stderr, code := sourceAndCall(t, "detect_platform "+tc.unameS+" "+tc.unameM)
		if code != 2 {
			t.Errorf("detect_platform %s %s: exit %d, want 2", tc.unameS, tc.unameM, code)
		}
		if strings.TrimSpace(stdout) != "" {
			t.Errorf("detect_platform %s %s printed to stdout on reject: %q", tc.unameS, tc.unameM, stdout)
		}
		// The message must name the detected platform and the supported set.
		if !strings.Contains(stderr, tc.unameS) || !strings.Contains(stderr, "darwin/linux") {
			t.Errorf("detect_platform %s %s stderr should name detected + supported: %q", tc.unameS, tc.unameM, stderr)
		}
	}
}

// TestAssetNames pins the spec-022 name template (a pure function). If
// GoReleaser's name_template ever drifts, this and the live installer break
// together.
func TestAssetNames(t *testing.T) {
	cases := []struct {
		ver, os, arch, want string
	}{
		{"1.4.0", "linux", "amd64", "glassfrog_1.4.0_linux_amd64.tar.gz glassfrog_1.4.0_checksums.txt"},
		{"1.3.0", "darwin", "arm64", "glassfrog_1.3.0_darwin_arm64.tar.gz glassfrog_1.3.0_checksums.txt"},
	}
	for _, tc := range cases {
		stdout, _, code := sourceAndCall(t, "asset_names "+tc.ver+" "+tc.os+" "+tc.arch)
		if code != 0 {
			t.Errorf("asset_names %s %s %s: exit %d", tc.ver, tc.os, tc.arch, code)
		}
		if got := strings.TrimSpace(stdout); got != tc.want {
			t.Errorf("asset_names %s %s %s = %q, want %q", tc.ver, tc.os, tc.arch, got, tc.want)
		}
	}
}

// TestVersionNormalisation pins the v-prefix handling: the download path keeps
// the published `v`, the asset name drops it.
func TestVersionNormalisation(t *testing.T) {
	cases := []struct {
		fn, in, want string
	}{
		{"normalize_tag", "1.3.0", "v1.3.0"},
		{"normalize_tag", "v1.3.0", "v1.3.0"},
		{"strip_v", "v1.3.0", "1.3.0"},
		{"strip_v", "1.3.0", "1.3.0"},
	}
	for _, tc := range cases {
		stdout, _, code := sourceAndCall(t, tc.fn+" "+tc.in)
		if code != 0 {
			t.Errorf("%s %s: exit %d", tc.fn, tc.in, code)
		}
		if got := strings.TrimSpace(stdout); got != tc.want {
			t.Errorf("%s %s = %q, want %q", tc.fn, tc.in, got, tc.want)
		}
	}
}
