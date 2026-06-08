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
		out, err := exec.Command("otool", "-L", binPath).Output()
		if err != nil {
			return nil, fmt.Errorf("otool -L %s: %w", binPath, err)
		}
		return parseOtool(string(out), binPath), nil
	case "linux":
		// ldd exits non-zero for a statically-linked binary ("not a dynamic
		// executable"), which is the self-contained case — so the exit status
		// is informational, not an error. Parse the combined output regardless.
		out, _ := exec.Command("ldd", binPath).CombinedOutput()
		return parseLdd(string(out)), nil
	default:
		return nil, fmt.Errorf("linkage inspection unsupported on %s", goos)
	}
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

// parseLdd extracts the dependency paths from `ldd` output. A statically-linked
// binary reports "not a dynamic executable" / "statically linked" and yields no
// dependencies. Otherwise each line is either "name => /path (0x...)" (take the
// resolved path) or a bare "/loader (0x...)" / "vdso (0x...)" entry.
func parseLdd(out string) []string {
	if strings.Contains(out, "not a dynamic executable") || strings.Contains(out, "statically linked") {
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
			resolved := strings.TrimSpace(line[i+2:])
			if resolved == "" {
				continue // "name => not found" with no path; skip
			}
			deps = append(deps, resolved)
			continue
		}
		deps = append(deps, line)
	}
	return deps
}
