package build

import (
	"strings"
	"testing"
)

// TestVersionInjection_RealConfig is the change-detector against the shipped
// .goreleaser.yaml: builds.ldflags must inject internal/cli.version (spec 023).
// The matrix config-guard ignores ldflags, so without this a future edit that
// blanks the seam or drifts the -X symbol path would pass CI silently and ship
// a release whose --version reports the build-info/placeholder value instead of
// the tag.
func TestVersionInjection_RealConfig(t *testing.T) {
	cfg, path, err := LoadConfig()
	if err != nil {
		t.Fatalf("loading the build config: %v", err)
	}
	if violations := CheckVersionInjection(cfg); len(violations) != 0 {
		t.Fatalf("the shipped %s must inject the version, got violations:\n  %s",
			path, strings.Join(violations, "\n  "))
	}
}

// TestVersionInjection_Drift exercises the guard against in-memory configs: a
// blanked seam, a missing ldflags, a drifted symbol path, and a malformed `-X`
// form must each fail and NAME the target symbol; a correctly-targeted -X must
// pass in either valid form (`-X sym=val` or `-X=sym=val`), regardless of the
// value template or other flags sharing the entry.
func TestVersionInjection_Drift(t *testing.T) {
	cases := []struct {
		name          string
		ldflags       []string
		wantViolation bool
	}{
		{"blanked seam", []string{""}, true},
		{"no ldflags at all", nil, true},
		{"drifted symbol path (main.version)", []string{"-X github.com/Luscii/cli-glassfrog/main.version=v1.0.0"}, true},
		// `-Xsym=val` (no separator) is not a form the Go linker parses as -X, so
		// the guard must reject it rather than let a broken seam through.
		{"malformed -X without separator", []string{"-Xgithub.com/Luscii/cli-glassfrog/internal/cli.version=1.0.0"}, true},
		{"target named outside a -X flag", []string{"-s -w -extldflags github.com/Luscii/cli-glassfrog/internal/cli.version=x"}, true},
		{"correct injection with template", []string{"-X github.com/Luscii/cli-glassfrog/internal/cli.version=v{{ .Version }}"}, false},
		{"correct injection, -X=sym=val form", []string{"-X=github.com/Luscii/cli-glassfrog/internal/cli.version=1.0.0"}, false},
		{"correct injection among other flags", []string{"-s -w -X github.com/Luscii/cli-glassfrog/internal/cli.version=v2.0.0"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{Builds: []Build{{Ldflags: tc.ldflags}}}
			violations := CheckVersionInjection(cfg)
			if tc.wantViolation && len(violations) == 0 {
				t.Fatalf("expected a violation for %q, got none", tc.name)
			}
			if !tc.wantViolation && len(violations) != 0 {
				t.Fatalf("expected no violation for %q, got: %v", tc.name, violations)
			}
			if tc.wantViolation && !strings.Contains(strings.Join(violations, " "), VersionInjectionTarget) {
				t.Fatalf("violation should name the target symbol %q, got: %v", VersionInjectionTarget, violations)
			}
		})
	}
}
