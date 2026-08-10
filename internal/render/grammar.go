package render

import (
	"sort"

	"github.com/Luscii/cli-glassfrog/internal/grammar"
)

// GrammarView is the data the `grammar` templates render (077): the change-set
// grammar the binary carries, flattened into the shapes a text/template can range
// over.
//
// It is the first render view over data that is not a server response — the
// embedded artifact IS the source data (plan ADR-4). Everything derived is derived
// HERE rather than in the template, following the TreeView precedent: templates
// stay flat ranges, and the derivation is unit-testable.
//
// The two provenance tokens are carried as fields rather than written into the
// templates, so the strings agents compare literally have exactly one spelling
// site (internal/grammar). They are present even when an array is empty, which is
// what lets the empty-residue case still name whose knowledge is missing.
type GrammarView struct {
	// ChangeTypes is every contract-enumerated type with its placement, in the
	// artifact's order (alphabetical).
	ChangeTypes []GrammarTypeRow
	// Groups is the same vocabulary condensed by placement class — top-level
	// first, then nested-only — for the compact format's one-line-per-class form.
	// A class with no members is omitted rather than rendered empty.
	Groups []GrammarPlacementGroup
	// NestedOnly and Wrappers carry the nesting rule's two halves, so `full` can
	// state the rule ONCE instead of repeating the wrapper pair on every entry.
	// Both are empty when the contract enumerates no nested-only type, and the
	// rule's section is then omitted — there is no rule to state.
	NestedOnly []string
	Wrappers   []string
	// Facts is the empirical residue in the record's manifest order. Empty is a
	// real state (the residue can retire to nothing), rendered as an explicit
	// statement rather than a vanished section.
	Facts []GrammarFactRow

	ContractProvenance string
	ObservedProvenance string
}

// GrammarTypeRow is one change type with its placement class.
type GrammarTypeRow struct {
	Type      string
	Placement string
}

// GrammarPlacementGroup is one placement class with the types that carry it.
// Wrappers is set only for a class that has wrappers (nested-only), so the
// compact line can name what the types must ride inside without a second lookup.
type GrammarPlacementGroup struct {
	Placement string
	Types     []string
	Wrappers  []string
}

// GrammarFactRow is one empirical fact. It carries the four fields the accord
// renders and deliberately not the record's Evidence or lineage, which the
// artifact already excludes.
type GrammarFactRow struct {
	ID          string
	Title       string
	Shape       string
	Disposition string
	Symptom     string
}

// NewGrammarView projects the embedded grammar onto the render view. It preserves
// the artifact's ordering — that ordering is the interface's determinism contract,
// so re-sorting here would make the human formats disagree with the structured
// ones — and derives only what the templates cannot: the placement grouping and
// the nesting rule's two halves.
//
// The wrapper set is the UNION across nested-only entries, sorted: the generator
// gives every nested-only entry the same pair, and taking the union means a future
// per-type wrapper set still yields a rule that names every part rather than
// whichever entry happened to come first.
func NewGrammarView(g grammar.Grammar) GrammarView {
	view := GrammarView{
		ChangeTypes:        make([]GrammarTypeRow, 0, len(g.ChangeTypes)),
		Facts:              make([]GrammarFactRow, 0, len(g.Facts)),
		ContractProvenance: grammar.ProvenancePublishedContract,
		ObservedProvenance: grammar.ProvenanceEmpiricalObservation,
	}

	byPlacement := map[string][]string{}
	wrapperSet := map[string]bool{}
	for _, ct := range g.ChangeTypes {
		view.ChangeTypes = append(view.ChangeTypes, GrammarTypeRow{Type: ct.Type, Placement: ct.Placement})
		byPlacement[ct.Placement] = append(byPlacement[ct.Placement], ct.Type)
		if ct.Placement == grammar.PlacementNestedOnly {
			view.NestedOnly = append(view.NestedOnly, ct.Type)
			for _, w := range ct.Wrappers {
				wrapperSet[w] = true
			}
		}
	}
	for w := range wrapperSet {
		view.Wrappers = append(view.Wrappers, w)
	}
	sort.Strings(view.Wrappers)

	// Top-level before nested-only: the placement a reader needs most often first,
	// and a fixed order so the compact output is deterministic. A class the
	// contract does not use is omitted; an unrecognized class (which the drift
	// guard makes unshippable) would otherwise vanish silently, so it is appended
	// after the known two rather than dropped.
	for _, placement := range []string{grammar.PlacementTopLevel, grammar.PlacementNestedOnly} {
		types, ok := byPlacement[placement]
		if !ok {
			continue
		}
		delete(byPlacement, placement)
		group := GrammarPlacementGroup{Placement: placement, Types: types}
		if placement == grammar.PlacementNestedOnly {
			group.Wrappers = view.Wrappers
		}
		view.Groups = append(view.Groups, group)
	}
	for _, placement := range sortedKeys(byPlacement) {
		view.Groups = append(view.Groups, GrammarPlacementGroup{Placement: placement, Types: byPlacement[placement]})
	}

	for _, f := range g.Facts {
		view.Facts = append(view.Facts, GrammarFactRow{
			ID:          f.ID,
			Title:       f.Title,
			Shape:       f.Shape,
			Disposition: f.Disposition,
			Symptom:     f.Symptom,
		})
	}
	return view
}

// sortedKeys returns a map's keys in sorted order, so an unexpected placement
// class renders deterministically instead of in map-iteration order.
func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
