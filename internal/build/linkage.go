package build

import (
	"fmt"
	"os/exec"
	"strings"
)

// osOnlyViolations returns the dynamic dependencies that fall outside the
// per-platform OS-only allowlist. An empty result means the binary depends only
// on OS-provided libraries — self-contained per CONSTITUTION XII. A non-empty
// result is the set of offending dependencies the verification names.
//
// This is "self-contained, not fully-static": a macOS Go binary always links
// the OS-provided libSystem (permitted); a Linux CGO_ENABLED=0 binary links
// nothing beyond the loader. A dependency outside the allowlist is the failure
// the check exists to catch.
func osOnlyViolations(goos string, deps []string) []string {
	var bad []string
	for _, d := range deps {
		if !isOSProvided(goos, d) {
			bad = append(bad, d)
		}
	}
	return bad
}

// isOSProvided reports whether a single dynamic dependency is part of the
// platform's OS allowlist.
func isOSProvided(goos, dep string) bool {
	dep = strings.TrimSpace(dep)
	switch goos {
	case "darwin":
		// Only system libraries under /usr/lib and /System/Library (e.g.
		// /usr/lib/libSystem.B.dylib, the CoreFoundation/Security frameworks).
		return strings.HasPrefix(dep, "/usr/lib/") || strings.HasPrefix(dep, "/System/Library/")
	case "linux":
		// A CGO_ENABLED=0 binary is statically linked; only the kernel vDSO and
		// the dynamic loader may legitimately appear. Anything else is a real
		// library dependency and a violation.
		return isLinuxLoader(dep)
	default:
		// An unknown platform has no allowlist — treat every dependency as a
		// violation rather than silently passing.
		return false
	}
}

// isLinuxLoader reports whether a Linux dynamic-dependency entry is the kernel
// vDSO or the dynamic loader, both OS-provided and not a real library link.
func isLinuxLoader(dep string) bool {
	base := dep
	if i := strings.LastIndexByte(base, '/'); i >= 0 {
		base = base[i+1:]
	}
	return strings.HasPrefix(base, "linux-vdso") ||
		strings.HasPrefix(base, "ld-linux") ||
		strings.HasPrefix(base, "ld-musl")
}

// extractDeps lists a binary's dynamic-library dependencies on the host
// platform, shelling out to the platform's standard inspector (otool on macOS,
// ldd on Linux) and parsing its output. The parsers are split out so they can
// be unit-tested against canned tool output on any host.
func extractDeps(goos, binPath string) ([]string, error) {
	switch goos {
	case "darwin":
		// CombinedOutput (not Output) so otool's stderr — where it reports a
		// non-Mach-O binary, a missing file, etc. — is surfaced in the error;
		// otool -L writes only stdout on success, and parseOtool keeps only
		// tab-indented lines, so any stray stderr cannot pollute the parse.
		out, err := exec.Command("otool", "-L", binPath).CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("otool -L %s: %w\n%s", binPath, err, out)
		}
		return parseOtool(string(out), binPath), nil
	case "linux":
		// ldd exits non-zero for a statically-linked binary ("not a dynamic
		// executable"), which IS the self-contained case — so that exit status
		// is informational, not a failure. But any OTHER ldd error (ldd missing,
		// not executable, permission denied) must surface: silently parsing
		// empty output would falsely report zero dependencies and pass the
		// self-containment check without inspecting linkage at all.
		out, err := exec.Command("ldd", binPath).CombinedOutput()
		text := string(out)
		if err != nil && !lddReportsStatic(text) {
			return nil, fmt.Errorf("ldd %s: %w\n%s", binPath, err, text)
		}
		return parseLdd(text), nil
	default:
		return nil, fmt.Errorf("linkage inspection unsupported on %s", goos)
	}
}

// lddReportsStatic reports whether ldd output indicates a statically-linked /
// non-dynamic binary — the self-contained case where ldd's non-zero exit is
// expected and carries no real error. Shared by extractDeps (to decide whether
// a non-zero exit is benign) and parseLdd (to short-circuit to zero deps).
func lddReportsStatic(out string) bool {
	return strings.Contains(out, "not a dynamic executable") ||
		strings.Contains(out, "statically linked")
}

// parseOtool extracts the dependency paths from `otool -L` output. The first
// line names the inspected file (and a dylib may list its own install name);
// both are dropped so only genuine dependencies remain.
func parseOtool(out, binPath string) []string {
	var deps []string
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "\t") {
			continue // only tab-indented lines are dependency entries
		}
		path := strings.TrimSpace(line)
		if i := strings.Index(path, " ("); i >= 0 {
			path = path[:i] // drop the " (compatibility version ...)" suffix
		}
		if path == "" || path == binPath {
			continue
		}
		deps = append(deps, path)
	}
	return deps
}

// parseLdd extracts the dependency identifiers from `ldd` output. A
// statically-linked binary reports "not a dynamic executable" / "statically
// linked" and yields no dependencies. Otherwise each line is either
// "name => /path (0x...)" (take the resolved path), "name => not found" (the
// loader could not resolve it — take the unresolved NAME), or a bare
// "/loader (0x...)" / "vdso (0x...)" entry.
func parseLdd(out string) []string {
	if lddReportsStatic(out) {
		return nil
	}
	var deps []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Drop the trailing load address "(0x...)".
		if i := strings.Index(line, " ("); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if i := strings.Index(line, "=>"); i >= 0 {
			name := strings.TrimSpace(line[:i])
			resolved := strings.TrimSpace(line[i+2:])
			// "name => not found" (or an empty right side): the loader cannot
			// resolve this dependency. That is a missing-library violation — the
			// most important case to surface — so report the library NAME, not
			// the literal "not found" (which would lose which library is missing).
			if resolved == "" || resolved == "not found" {
				if name != "" {
					deps = append(deps, name)
				}
				continue
			}
			deps = append(deps, resolved)
			continue
		}
		deps = append(deps, line)
	}
	return deps
}
