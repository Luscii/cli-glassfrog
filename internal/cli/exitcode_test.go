package cli

import "testing"

// publishedCodes names the seven frozen exit codes for the change-detector,
// uniqueness, and shell-reserved checks below. The constants themselves live in
// exitcode.go and are the single source of truth; this map mirrors them by name
// so a renumber is caught against the exact-value expectations.
var publishedCodes = map[string]int{
	"success":             codeSuccess,
	"internal":            codeInternalError,
	"usage":               codeUsageError,
	"api":                 codeAPIError,
	"permission":          codePermissionError,
	"rate-limited":        codeRateLimited,
	"network-unavailable": codeNetworkUnavailable,
}

// ExitCode maps the categories that have producers today: Success→0,
// UsageError→2, and RuntimeError→1. The operational categories (codes 3–6) have
// no Outcome value yet (ADR-2), so they are pinned at the constant level only.
// Asserting RuntimeError→1 directly here keeps this suite the change-detector
// for the producer-backed category→code arms — independent of the indirect
// coverage via runToExitCode and the BDD scenarios.
func TestExitCode_ProducerBackedCategories(t *testing.T) {
	if got := ExitCode(Success); got != 0 {
		t.Errorf("ExitCode(Success) = %d, want 0", got)
	}
	if got := ExitCode(UsageError); got != 2 {
		t.Errorf("ExitCode(UsageError) = %d, want 2", got)
	}
	if got := ExitCode(RuntimeError); got != 1 {
		t.Errorf("ExitCode(RuntimeError) = %d, want 1", got)
	}
}

// The Fail-Safe default: any category without an explicit case maps to 1, never
// 0 (CONSTITUTION III). Outcome(99) stands in for an unmapped/future category.
func TestExitCode_DefaultArmIsInternalError(t *testing.T) {
	if got := ExitCode(Outcome(99)); got != 1 {
		t.Errorf("ExitCode(unmapped) = %d, want 1 (Fail-Safe default)", got)
	}
}

// Change-detector: the exact frozen values. A future renumber breaks loudly
// here so it can never happen silently (interface-cli.md "Extension").
func TestExitCodeConstants_ExactValues(t *testing.T) {
	want := map[string]int{
		"success": 0, "internal": 1, "usage": 2, "api": 3,
		"permission": 4, "rate-limited": 5, "network-unavailable": 6,
	}
	for name, w := range want {
		if got := publishedCodes[name]; got != w {
			t.Errorf("code %q = %d, want %d (frozen convention — a renumber must be deliberate)", name, got, w)
		}
	}
}

// No two categories share a code. This is the constant-level half of the
// "Codes and categories are one-to-one" accord; the full category↔code
// one-to-one waits for the operational producers (ADR-2).
func TestExitCodeConstants_Distinct(t *testing.T) {
	seen := map[int]string{}
	for name, code := range publishedCodes {
		if prior, dup := seen[code]; dup {
			t.Errorf("code %d assigned to both %q and %q — codes must be distinct", code, prior, name)
		}
		seen[code] = name
	}
}

// No assigned code falls in the shell-reserved range: 126 (not executable),
// 127 (command not found), and 128+N (terminated by signal N). Staying below
// 126 keeps $? unambiguous against shell/signal semantics (interface-cli.md
// Error Communication).
func TestExitCodeConstants_NoShellReserved(t *testing.T) {
	for name, code := range publishedCodes {
		if code >= 126 {
			t.Errorf("code %q = %d falls in the shell-reserved range (126, 127, 128+N)", name, code)
		}
	}
}
