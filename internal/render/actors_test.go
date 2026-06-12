package render

import (
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// --- actors render (048): flat homogeneous directory list -------------------
//
// One render key over a flat homogeneous list — every row is the same Actor
// shape (unlike search's heterogeneous type-badged rows). The full golden pins
// the per-row per_/agt_ id, the kind badge, and the indented Name line; the
// compact golden pins the one-line-per-actor id + kind + name. Both pin the
// `no actors` empty line. A mixed human/agent page shows each row with its own
// kind badge.

// mixedActorsView is a human followed by an agent — exercising both id prefixes
// and both kind badges in one view.
func mixedActorsView() ActorsView {
	return ActorsView{Data: []glassfrog.Actor{
		{ID: "per_0123", Name: "Alice Smith", Kind: "human"},
		{ID: "agt_0456", Name: "Claude", Kind: "agent"},
	}}
}

func TestRender_ActorsFull_MixedKinds_Golden(t *testing.T) {
	want := "per_0123  [human]\n" +
		"  Name:  Alice Smith\n" +
		"agt_0456  [agent]\n" +
		"  Name:  Claude\n"
	assertRender(t, ResourceActors, FormatFull, mixedActorsView(), want)
}

func TestRender_ActorsCompact_MixedKinds_Golden(t *testing.T) {
	want := "per_0123  [human]  Alice Smith\n" +
		"agt_0456  [agent]  Claude\n"
	assertRender(t, ResourceActors, FormatCompact, mixedActorsView(), want)
}

// An empty directory (no actor matched the filters) renders the explicit
// `no actors` line under both human formats — a valid empty answer, not an error.
func TestRender_ActorsFull_Empty_Golden(t *testing.T) {
	assertRender(t, ResourceActors, FormatFull, ActorsView{}, "no actors\n")
}

func TestRender_ActorsCompact_Empty_Golden(t *testing.T) {
	assertRender(t, ResourceActors, FormatCompact, ActorsView{Data: nil}, "no actors\n")
}

// A blank (present-but-whitespace) or absent name renders the `—` absence marker
// rather than a fabricated value or an empty gap — the repo's trim-emptiness
// convention (CONSTITUTION VIII). The name is otherwise rendered verbatim, never
// truncated or reflowed (CONSTITUTION VI).
func TestRender_ActorsFull_BlankNameShowsMarker(t *testing.T) {
	view := ActorsView{Data: []glassfrog.Actor{
		{ID: "per_blank", Name: "   ", Kind: "human"},
		{ID: "agt_none", Name: "", Kind: "agent"},
	}}
	got, err := Render(ResourceActors, FormatFull, view)
	if err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	if strings.Count(got, "Name:  —") != 2 {
		t.Errorf("a blank and an absent name should both render the — marker:\n%s", got)
	}
}

// A long name is rendered verbatim — never truncated or reflowed (CONSTITUTION VI).
func TestRender_ActorsCompact_NameRenderedVerbatim(t *testing.T) {
	long := "A Very Long Actor Name That The Directory Must Not Truncate Or Reflow Across Lines"
	view := ActorsView{Data: []glassfrog.Actor{{ID: "per_long", Name: long, Kind: "human"}}}
	got, err := Render(ResourceActors, FormatCompact, view)
	if err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	if !strings.Contains(got, long) {
		t.Errorf("the name must be rendered verbatim, never truncated:\n%s", got)
	}
}
