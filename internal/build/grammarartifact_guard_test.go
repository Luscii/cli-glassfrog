package build

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/grammar"
)

// The drift guard for the Agent-Facing Grammar Reference (077 T003). It runs under
// `go test ./...`, which the merge gate already executes, so no new CI wiring is
// needed (the 024/029 pattern, same as 072/075/076).

// grammarArtifactSides loads the three sides the guard compares: the committed
// artifact, a fresh derivation, and the record's live-facts manifest. A failure on
// any side is a test failure rather than a guard finding — the guard checks
// AGREEMENT between sides, and a side that will not load is a different problem
// with its own message.
func grammarArtifactSides(t *testing.T) (committed, regenerated []byte, manifestIDs []string) {
	t.Helper()
	committed, err := ReadGrammarArtifact()
	if err != nil {
		t.Fatalf("reading the committed artifact %s: %v — run `%s`", GrammarArtifactPath, err, GrammarRegenerationStep)
	}
	regenerated, err = RenderGrammarArtifact()
	if err != nil {
		t.Fatalf("re-deriving the artifact from its sources failed: %v", err)
	}
	raw, err := ReadGrammarFactsRecord()
	if err != nil {
		t.Fatalf("reading the grammar record %s: %v", GrammarFactsPath, err)
	}
	return committed, regenerated, ParseGrammarFactsRecord(raw).ManifestIDs
}

// TestGrammarArtifactDriftGuard is the green case: the committed artifact is
// exactly what the vendored contract and the grammar record produce. It is the
// tripwire a record edit, a contract refresh, or a hand edit turns red.
func TestGrammarArtifactDriftGuard(t *testing.T) {
	committed, regenerated, manifest := grammarArtifactSides(t)
	if findings := CheckGrammarArtifact(committed, regenerated, manifest); len(findings) != 0 {
		t.Fatalf("the committed grammar artifact has drifted from its sources:\n  - %s", strings.Join(findings, "\n  - "))
	}
}

// TestGrammarArtifactGuardNamesTheRegenerationStepInEveryFinding pins the
// reachable-remedy discipline: a guard's failure message is a specification, so
// every one of them must name the single remedy that satisfies all the sibling
// invariants at once.
func TestGrammarArtifactGuardNamesTheRegenerationStepInEveryFinding(t *testing.T) {
	_, regenerated, manifest := grammarArtifactSides(t)
	// One deliberately broken artifact that trips several invariants at once, so
	// the assertion covers the messages rather than one of them.
	broken := `{"generated":"","grammar":{"change_types":[],"facts":[{"id":"CSG-9","title":"t","shape":"s","disposition":"accepted","symptom":"y","provenance":"hearsay"}]}}`
	findings := CheckGrammarArtifact([]byte(broken), regenerated, manifest)
	if len(findings) < 4 {
		t.Fatalf("expected the broken artifact to trip several invariants; got %d findings: %v", len(findings), findings)
	}
	for _, f := range findings {
		if !strings.Contains(f, GrammarRegenerationStep) {
			t.Fatalf("finding %q does not name the regeneration step %q as the remedy", f, GrammarRegenerationStep)
		}
		if !strings.Contains(f, GrammarArtifactPath) {
			t.Fatalf("finding %q does not name the artifact it is about", f)
		}
	}
}

// --- One red case per divergence class -------------------------------------
//
// Each case perturbs ONE side and asserts the guard names that side. The
// perturbations are in-memory: the committed artifact on disk stays truthful, so
// these tests cannot leave the repository red for the next run.

// TestGrammarArtifactGuardCatchesAHandEditedArtifact — the artifact's bytes were
// changed directly. Every decoded half still agrees, so the guard must name the
// ENCODING rather than blame a source.
func TestGrammarArtifactGuardCatchesAHandEditedArtifact(t *testing.T) {
	committed, regenerated, manifest := grammarArtifactSides(t)
	// A reformat that changes bytes without changing meaning: re-encode compactly.
	var artifact grammar.Artifact
	if err := json.Unmarshal(committed, &artifact); err != nil {
		t.Fatalf("decoding the committed artifact: %v", err)
	}
	compact, err := json.Marshal(artifact)
	if err != nil {
		t.Fatalf("re-encoding: %v", err)
	}
	findings := CheckGrammarArtifact(compact, regenerated, manifest)
	if len(findings) == 0 {
		t.Fatal("the guard accepted a hand-reformatted artifact")
	}
	if !containsFold(strings.Join(findings, " "), "the ENCODING diverged") {
		t.Fatalf("the guard did not name the encoding as the diverged half: %v", findings)
	}
}

// TestGrammarArtifactGuardCatchesARecordEditWithoutRegeneration — the record's
// fact text changed and nobody re-ran the generator. The residue half diverges;
// the vocabulary half does not.
func TestGrammarArtifactGuardCatchesARecordEditWithoutRegeneration(t *testing.T) {
	committed, regenerated, manifest := grammarArtifactSides(t)
	// Model the fresh derivation a rewritten record would produce.
	edited := grammarArtifactWith(t, regenerated, func(a *grammar.Artifact) {
		if len(a.Grammar.Facts) == 0 {
			t.Skip("the record carries no live facts to edit")
		}
		a.Grammar.Facts[0].Symptom = "a rewritten symptom the committed artifact has never seen"
	})
	findings := CheckGrammarArtifact(committed, edited, manifest)
	if len(findings) == 0 {
		t.Fatal("the guard accepted a record edit that outran the artifact")
	}
	joined := strings.Join(findings, " ")
	if !strings.Contains(joined, "RECORD-DERIVED residue differs") || !strings.Contains(joined, GrammarFactsPath) {
		t.Fatalf("the guard did not name the record as the diverged half: %v", findings)
	}
	if strings.Contains(joined, "CONTRACT-DERIVED vocabulary differs") {
		t.Fatalf("the guard blamed the contract for a record edit: %v", findings)
	}
}

// TestGrammarArtifactGuardCatchesAContractRefreshWithoutRegeneration — the
// vendored contract's change-type enum grew and nobody re-ran the generator. This
// is the feature file's "A contract refresh that outruns the rendering fails the
// build" scenario, pinned at the unit level too.
func TestGrammarArtifactGuardCatchesAContractRefreshWithoutRegeneration(t *testing.T) {
	committed, regenerated, manifest := grammarArtifactSides(t)
	refreshed := grammarArtifactWith(t, regenerated, func(a *grammar.Artifact) {
		a.Grammar.ChangeTypes = append(a.Grammar.ChangeTypes, grammar.ChangeType{
			Type:       "ZzArchiveRole",
			Placement:  grammar.PlacementTopLevel,
			Provenance: grammar.ProvenancePublishedContract,
		})
	})
	findings := CheckGrammarArtifact(committed, refreshed, manifest)
	if len(findings) == 0 {
		t.Fatal("the guard accepted a contract refresh that outran the artifact")
	}
	joined := strings.Join(findings, " ")
	if !strings.Contains(joined, "CONTRACT-DERIVED vocabulary differs") || !strings.Contains(joined, VendoredSpecPath) {
		t.Fatalf("the guard did not name the contract as the diverged half: %v", findings)
	}
}

// TestGrammarArtifactGuardCatchesAMissingMarker — the envelope's do-not-edit
// marker was stripped. The invariant is independent of byte-equality, so the
// finding must name the envelope by itself.
func TestGrammarArtifactGuardCatchesAMissingMarker(t *testing.T) {
	_, regenerated, manifest := grammarArtifactSides(t)
	for name, marker := range map[string]string{
		"absent":                    "",
		"whitespace only":           "   \n",
		"no do-not-edit":            "Generated from the contract; regenerate with `" + GrammarRegenerationStep + "`.",
		"no regeneration step":      "DO NOT EDIT — this file is generated.",
		"prose without either half": "the change-set grammar",
	} {
		t.Run(name, func(t *testing.T) {
			stripped := grammarArtifactWith(t, regenerated, func(a *grammar.Artifact) { a.Generated = marker })
			findings := CheckGrammarArtifact(stripped, regenerated, manifest)
			joined := strings.Join(findings, " ")
			if !strings.Contains(joined, "no well-formed generated marker") {
				t.Fatalf("the guard did not reject the marker %q: %v", marker, findings)
			}
		})
	}
}

// TestGrammarArtifactGuardAcceptsTheCanonicalMarker is the marker invariant's
// green half — without it the check could reject everything and still pass the
// red cases above.
func TestGrammarArtifactGuardAcceptsTheCanonicalMarker(t *testing.T) {
	if !grammarMarkerIsWellFormed(grammarGeneratedMarker) {
		t.Fatalf("the canonical marker is rejected by its own invariant: %q", grammarGeneratedMarker)
	}
}

// TestGrammarArtifactGuardCatchesAManifestFactMismatch — the manifest and the
// rendered residue disagree: a retirement that dropped the manifest entry but left
// the artifact, or the reverse. Also covers a reorder, since the residue's order is
// part of the interface contract.
func TestGrammarArtifactGuardCatchesAManifestFactMismatch(t *testing.T) {
	committed, regenerated, manifest := grammarArtifactSides(t)
	if len(manifest) < 2 {
		t.Skipf("this case needs at least two live facts; the manifest declares %v", manifest)
	}
	cases := map[string][]string{
		"a fact retired from the manifest but left in the artifact": manifest[1:],
		"a fact added to the manifest but not yet rendered":         append(append([]string(nil), manifest...), "CSG-99"),
		"the manifest reordered":                                    {manifest[1], manifest[0]},
	}
	for name, perturbed := range cases {
		t.Run(name, func(t *testing.T) {
			findings := CheckGrammarArtifact(committed, regenerated, perturbed)
			joined := strings.Join(findings, " ")
			if !strings.Contains(joined, "Live-facts manifest declares") {
				t.Fatalf("the guard did not name the manifest/residue mismatch: %v", findings)
			}
			if !strings.Contains(joined, GrammarFactsPath) {
				t.Fatalf("the mismatch finding does not name the record: %v", findings)
			}
		})
	}
}

// TestGrammarArtifactGuardCatchesAnUndecodableArtifact — the corrupt-artifact
// invariant. Nothing else is evaluable, so the guard reports exactly one finding
// rather than a cascade that buries it.
func TestGrammarArtifactGuardCatchesAnUndecodableArtifact(t *testing.T) {
	_, regenerated, manifest := grammarArtifactSides(t)
	findings := CheckGrammarArtifact([]byte(`{"grammar": {"change_types": [`), regenerated, manifest)
	if len(findings) != 1 {
		t.Fatalf("expected exactly one finding for an undecodable artifact; got %v", findings)
	}
	if !strings.Contains(findings[0], "does not decode") || !strings.Contains(findings[0], GrammarRegenerationStep) {
		t.Fatalf("finding %q does not name the decode failure and the remedy", findings[0])
	}
}

// TestGrammarArtifactGuardCatchesALostProvenanceToken — an entry stripped of its
// provenance marking. Both arrays are covered, because a consumer tells a
// contract-published shape from a verified observation by the token alone.
func TestGrammarArtifactGuardCatchesALostProvenanceToken(t *testing.T) {
	_, regenerated, manifest := grammarArtifactSides(t)

	t.Run("a change type", func(t *testing.T) {
		perturbed := grammarArtifactWith(t, regenerated, func(a *grammar.Artifact) {
			a.Grammar.ChangeTypes[0].Provenance = ""
		})
		findings := CheckGrammarArtifact(perturbed, regenerated, manifest)
		joined := strings.Join(findings, " ")
		if !strings.Contains(joined, "lost its provenance marking") || !strings.Contains(joined, "CONTRACT-DERIVED") {
			t.Fatalf("the guard did not name the lost contract provenance: %v", findings)
		}
	})

	t.Run("a fact", func(t *testing.T) {
		perturbed := grammarArtifactWith(t, regenerated, func(a *grammar.Artifact) {
			if len(a.Grammar.Facts) == 0 {
				t.Skip("the record carries no live facts")
			}
			a.Grammar.Facts[0].Provenance = grammar.ProvenancePublishedContract
		})
		findings := CheckGrammarArtifact(perturbed, regenerated, manifest)
		joined := strings.Join(findings, " ")
		if !strings.Contains(joined, "lost its provenance marking") || !strings.Contains(joined, "RECORD-DERIVED") {
			t.Fatalf("the guard did not name the mismarked fact: %v", findings)
		}
	})
}

// TestGrammarArtifactGuardCatchesAnEmptyVocabulary — an artifact that decodes but
// carries nothing. It would render an empty reference, which reads as an answer.
func TestGrammarArtifactGuardCatchesAnEmptyVocabulary(t *testing.T) {
	_, regenerated, manifest := grammarArtifactSides(t)
	emptied := grammarArtifactWith(t, regenerated, func(a *grammar.Artifact) {
		a.Grammar.ChangeTypes = []grammar.ChangeType{}
	})
	findings := CheckGrammarArtifact(emptied, regenerated, manifest)
	joined := strings.Join(findings, " ")
	if !strings.Contains(joined, "carries no change types") {
		t.Fatalf("the guard accepted an empty vocabulary: %v", findings)
	}
}

// grammarArtifactWith decodes the canonical bytes, applies a perturbation, and
// re-encodes them canonically — so a test can model "what a changed source would
// produce" (or a tampered artifact) without touching a real file.
func grammarArtifactWith(t *testing.T, canonical []byte, mutate func(*grammar.Artifact)) []byte {
	t.Helper()
	var artifact grammar.Artifact
	if err := json.Unmarshal(canonical, &artifact); err != nil {
		t.Fatalf("decoding the canonical artifact: %v", err)
	}
	mutate(&artifact)
	doc, err := MarshalGrammarArtifact(artifact)
	if err != nil {
		t.Fatalf("re-encoding the perturbed artifact: %v", err)
	}
	return doc
}
