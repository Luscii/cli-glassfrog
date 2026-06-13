package render

import (
	"strings"
	"testing"
)

// TestRender_TensionDiscardFull_Golden pins the synthesized full projection: the
// single confirmation line naming the discarded ten_ id with the [discarded]
// marker (mirroring the `tension` key's `<ten_…>  [<status>]` shape).
func TestRender_TensionDiscardFull_Golden(t *testing.T) {
	want := "ten_0123  [discarded]\n"
	assertRender(t, ResourceTensionDiscard, FormatFull, TensionDiscardView{ID: "ten_0123"}, want)
}

// TestRender_TensionDiscardCompact_Golden pins that compact renders the SAME
// single confirmation line as full — there is nothing more to show for a bodyless
// soft-delete (plan ADR-3).
func TestRender_TensionDiscardCompact_Golden(t *testing.T) {
	want := "ten_0123  [discarded]\n"
	assertRender(t, ResourceTensionDiscard, FormatCompact, TensionDiscardView{ID: "ten_0123"}, want)
}

// TestRender_TensionDiscard_FullAndCompactIdentical confirms the two formats are
// byte-identical (the view carries a single field, so both projections collapse to
// the same line), for a realistically long ten_ id.
func TestRender_TensionDiscard_FullAndCompactIdentical(t *testing.T) {
	view := TensionDiscardView{ID: "ten_0123456789abcdef0123456789abcdef"}
	full, err := Render(ResourceTensionDiscard, FormatFull, view)
	if err != nil {
		t.Fatalf("full render: %v", err)
	}
	compact, err := Render(ResourceTensionDiscard, FormatCompact, view)
	if err != nil {
		t.Fatalf("compact render: %v", err)
	}
	if full != compact {
		t.Errorf("full and compact must be identical:\nfull:    %q\ncompact: %q", full, compact)
	}
	if !strings.Contains(full, view.ID) {
		t.Errorf("the confirmation line must name the discarded id %q:\n%s", view.ID, full)
	}
}

// TestRender_TensionDiscard_ClaimsOnlyTheID is the structural mirror of the spec's
// validation scenario: the view exposes only the id, so no server-owned field (a
// discarded-at timestamp, a status, a body) can leak into the rendered result. Any
// such substring would be a fabrication the bodyless response never returned.
func TestRender_TensionDiscard_ClaimsOnlyTheID(t *testing.T) {
	for _, format := range []Format{FormatFull, FormatCompact} {
		got, err := Render(ResourceTensionDiscard, format, TensionDiscardView{ID: "ten_0123"})
		if err != nil {
			t.Fatalf("render %s: %v", format, err)
		}
		for _, fabricated := range []string{"discarded_at", "discarded-at", "2026-", "Body:", "Sensing role:"} {
			if strings.Contains(got, fabricated) {
				t.Errorf("%s must claim nothing the server did not return, but found %q:\n%s", format, fabricated, got)
			}
		}
	}
}
