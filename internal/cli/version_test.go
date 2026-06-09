package cli

import (
	"runtime/debug"
	"testing"
)

// buildInfo builds a *debug.BuildInfo whose Main.Version is v — the field
// runtime/debug records as the module version, which resolveVersion reads as
// its build-info fallback.
func buildInfo(v string) *debug.BuildInfo {
	return &debug.BuildInfo{Main: debug.Module{Version: v}}
}

// resolveVersion implements the spec-023 three-tier precedence. These tests pin
// every branch with crafted inputs — no binary build needed — and the
// never-empty invariant that version output must always satisfy.
func TestResolveVersion(t *testing.T) {
	cases := []struct {
		name     string
		injected string
		info     *debug.BuildInfo
		ok       bool
		want     string
	}{
		{
			name:     "injected wins outright",
			injected: "v1.4.0",
			info:     nil,
			ok:       false,
			want:     "v1.4.0",
		},
		{
			name:     "injected wins over recorded build info",
			injected: "v1.4.0",
			info:     buildInfo("v9.9.9"),
			ok:       true,
			want:     "v1.4.0",
		},
		{
			name:     "tagged source install reports the recorded module version",
			injected: "",
			info:     buildInfo("v1.3.2"),
			ok:       true,
			want:     "v1.3.2",
		},
		{
			name:     "untagged source install reports the pseudo-version verbatim",
			injected: "",
			info:     buildInfo("v0.0.0-20260101120000-abc123def456"),
			ok:       true,
			want:     "v0.0.0-20260101120000-abc123def456",
		},
		{
			name:     "plain local build reports Go's development marker verbatim",
			injected: "",
			info:     buildInfo("(devel)"),
			ok:       true,
			want:     "(devel)",
		},
		{
			name:     "no injection and no build info reports the placeholder",
			injected: "",
			info:     nil,
			ok:       false,
			want:     placeholderVersion,
		},
		{
			name:     "build info present but empty version falls to the placeholder",
			injected: "",
			info:     buildInfo(""),
			ok:       true,
			want:     placeholderVersion,
		},
		{
			name:     "pre-release suffix is preserved exactly",
			injected: "v1.4.0-rc.1",
			info:     nil,
			ok:       false,
			want:     "v1.4.0-rc.1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveVersion(tc.injected, tc.info, tc.ok)
			if got != tc.want {
				t.Fatalf("resolveVersion(%q, %v, %v) = %q, want %q",
					tc.injected, tc.info, tc.ok, got, tc.want)
			}
			if got == "" {
				t.Fatal("resolveVersion must never return an empty string")
			}
		})
	}
}

// The never-empty invariant holds across the full combination space, including
// the degenerate ok-without-info case a future ReadBuildInfo edge could yield.
func TestResolveVersionNeverEmpty(t *testing.T) {
	infos := []struct {
		info *debug.BuildInfo
		ok   bool
	}{
		{nil, false},
		{nil, true},
		{buildInfo(""), true},
		{buildInfo("v1.0.0"), true},
		{buildInfo("(devel)"), true},
	}
	for _, injected := range []string{"", "v2.0.0"} {
		for _, in := range infos {
			if got := resolveVersion(injected, in.info, in.ok); got == "" {
				t.Fatalf("resolveVersion(%q, %v, %v) returned empty", injected, in.info, in.ok)
			}
		}
	}
}
