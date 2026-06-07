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

// ExitCode maps the categories that have producers today: the original three
// (Success→0, UsageError→2, RuntimeError→1), the two Identity Read (011) added as
// the first consuming command (NetworkUnavailable→6, APIError→3), and the two
// API Error Extraction (015) added by splitting APIError on the status
// (PermissionError→4, RateLimited→5 — codes 004 reserved, now live).
//
// outcomeCodes mirrors the producer-backed category→code arms by name; the
// length check plus the comma-ok lookup catch an arm being dropped or added
// (without them, dropping Success→0 would pass silently because a missing key
// reads back as the zero value 0, which equals its expected code — PR #10
// LEARNINGS, the zero-valued-expectation trap).
var outcomeCodes = map[Outcome]int{
	Success:            0,
	UsageError:         2,
	RuntimeError:       1,
	APIError:           3,
	PermissionError:    4,
	RateLimited:        5,
	NetworkUnavailable: 6,
}

func TestExitCode_ProducerBackedCategories(t *testing.T) {
	want := map[Outcome]int{
		Success:            0,
		UsageError:         2,
		RuntimeError:       1,
		APIError:           3,
		PermissionError:    4,
		RateLimited:        5,
		NetworkUnavailable: 6,
	}
	if len(outcomeCodes) != len(want) {
		t.Errorf("outcomeCodes has %d entries, want %d — a producer-backed category was added or removed without updating this test", len(outcomeCodes), len(want))
	}
	for o, w := range want {
		got, ok := outcomeCodes[o]
		if !ok {
			t.Errorf("category %v is missing from outcomeCodes", o)
			continue
		}
		if got != w {
			t.Errorf("outcomeCodes[%v] = %d, want %d", o, got, w)
		}
		if mapped := ExitCode(o); mapped != w {
			t.Errorf("ExitCode(%v) = %d, want %d", o, mapped, w)
		}
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
// here so it can never happen silently (interface-cli.md "Extension"). The
// length check plus the comma-ok lookup also catch a code being removed or
// renamed: without them, dropping "success" would pass silently because a
// missing key reads back as the zero value 0, which equals its expected code.
func TestExitCodeConstants_ExactValues(t *testing.T) {
	want := map[string]int{
		"success": 0, "internal": 1, "usage": 2, "api": 3,
		"permission": 4, "rate-limited": 5, "network-unavailable": 6,
	}
	if len(publishedCodes) != len(want) {
		t.Errorf("publishedCodes has %d entries, want %d — a code was added or removed without updating this test", len(publishedCodes), len(want))
	}
	for name, w := range want {
		got, ok := publishedCodes[name]
		if !ok {
			t.Errorf("code %q is missing from publishedCodes", name)
			continue
		}
		if got != w {
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
