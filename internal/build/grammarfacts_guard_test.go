package build

import (
	"strings"
	"testing"
)

// TestGrammarFactsGuard is the best-effort citation-integrity + internal-
// consistency guard for the change-set grammar record (072, plan ADR-3). It
// holds the record to its own manifest and to the contract it cites, deriving
// every side from its file at test time — the record's manifest, fields,
// disposition vocabulary, cited types, and nested-only citation from the record;
// the change-type enum and nested-only set from the vendored spec. It hard-codes
// no fact ids, enum values, or type names, which is what lets a deliberate
// retirement (section + manifest entry dropped together) pass while a partial
// one fails.
//
// COVERAGE (explicitly partial — stated, not silent): the guard checks that the
// record agrees with ITSELF and that its citations still resolve in the
// contract. It CANNOT detect the spec SEMANTICALLY absorbing a fact — CSG-2's
// accepted-but-invalid behavior is prose, not a schema key, so a refresh that
// documents it in words the guard does not parse leaves the fact overstaying.
// That residue belongs to the vendored-spec refresh-diff review (LEARNINGS
// 2026-08-05 S7 action), not to this guard.
func TestGrammarFactsGuard(t *testing.T) {
	raw, err := ReadGrammarFactsRecord()
	if err != nil {
		t.Fatalf("could not read the grammar-facts record (%s): %v", GrammarFactsPath, err)
	}
	rec := ParseGrammarFactsRecord(raw)

	enum, nestedOnly, err := LoadSpecChangeTypes()
	if err != nil {
		t.Fatalf("could not derive the spec-side sets from %s: %v", VendoredSpecPath, err)
	}

	// Sanity-check the extraction so a regression in the parsers fails loudly
	// rather than passing vacuously.
	if len(rec.Facts) == 0 {
		t.Fatal("no fact sections parsed from the record — the parser or the record regressed")
	}
	if len(rec.ManifestIDs) == 0 {
		t.Fatal("the Live-facts manifest parsed empty — the parser or the record regressed")
	}
	if len(enum) == 0 || len(nestedOnly) == 0 {
		t.Fatalf("spec-side extraction empty (enum=%d, nested-only=%d) — the spec parser regressed", len(enum), len(nestedOnly))
	}

	// The committed record must agree with itself and with the contract.
	if v := CheckGrammarFacts(rec, enum, nestedOnly); len(v) != 0 {
		t.Fatalf("the committed grammar-facts record drifted:\n  - %s", strings.Join(v, "\n  - "))
	}
}

// TestGrammarFactsGuardConditions proves each of the eight violation conditions
// fires loudly, naming the offending element and its resolution path — a guard
// that never fails is no guard. Each case mutates a valid fixture (or the passed
// spec sides) to trip exactly one condition, then asserts the message names what
// the interface-spec § Error Communication table requires.
func TestGrammarFactsGuardConditions(t *testing.T) {
	// A valid record + spec sides the fixture is built against. The nested-only
	// list mirrors the real contract's six accountability/domain types, so the
	// baseline passes.
	enum := []string{"CreateRole", "UpdateRole", "CreatePolicy", "CreateAccountability", "UpdateAccountability", "RemoveAccountability", "CreateDomain", "UpdateDomain", "RemoveDomain"}
	nestedOnly := []string{"CreateAccountability", "UpdateAccountability", "RemoveAccountability", "CreateDomain", "UpdateDomain", "RemoveDomain"}

	// Baseline must pass — otherwise the mutations prove nothing.
	if v := CheckGrammarFacts(ParseGrammarFactsRecord(validGrammarRecordFixture()), enum, nestedOnly); len(v) != 0 {
		t.Fatalf("baseline fixture is not clean:\n  - %s", strings.Join(v, "\n  - "))
	}

	cases := []struct {
		name       string
		record     string   // fixture (default valid)
		specEnum   []string // default enum
		specNested []string // default nestedOnly
		wantNames  []string // substrings the failure must name
	}{
		{
			name:      "1: manifest declares an id with no section",
			record:    strings.Replace(validGrammarRecordFixture(), "**Live facts**: CSG-1, CSG-2", "**Live facts**: CSG-1, CSG-2, CSG-9", 1),
			wantNames: []string{"CSG-9", "complete the retirement", "restore the section"},
		},
		{
			name:      "2: a section's id is absent from the manifest",
			record:    strings.Replace(validGrammarRecordFixture(), "**Live facts**: CSG-1, CSG-2", "**Live facts**: CSG-1", 1),
			wantNames: []string{"CSG-2", "add the id to the manifest", "finish the deletion"},
		},
		{
			name:      "3: the record has zero fact sections",
			record:    emptyShellGrammarRecordFixture(),
			wantNames: []string{"no fact sections", "empty record", "delete the record"},
		},
		{
			name:      "4: a required field is missing or empty",
			record:    strings.Replace(validGrammarRecordFixture(), "- **Evidence**: prp_ebe2815f", "- **Evidence**:", 1),
			wantNames: []string{"CSG-1", "Evidence", "supply the field"},
		},
		{
			name:      "5: a Disposition outside the closed vocabulary",
			record:    strings.Replace(validGrammarRecordFixture(), "- **Disposition**: accepted\n", "- **Disposition**: probably-fine\n", 1),
			wantNames: []string{"CSG-1", "probably-fine", "accepted-but-invalid"},
		},
		{
			// The record's Shape names a type the (unchanged) spec enum never
			// carries — a stale citation, not a spec move.
			name:      "6: a cited change type absent from the spec enum",
			record:    strings.Replace(validGrammarRecordFixture(), "- **Shape**: an `UpdateRole` self-targeting the circle", "- **Shape**: a `BogusChange` self-targeting the circle", 1),
			wantNames: []string{"CSG-2", "BogusChange", "re-derive the citation", "retire the fact"},
		},
		{
			name:       "7: the nested-only citation is not set-equal to the spec's",
			specNested: []string{"CreateAccountability", "UpdateAccountability", "RemoveAccountability", "CreateDomain", "UpdateDomain"}, // RemoveDomain dropped by a refresh
			wantNames:  []string{"nested-only", "RemoveDomain", "re-derive the citation", "retire the fact"},
		},
		{
			name:      "8: the empirical marker is absent or degraded",
			record:    strings.Replace(validGrammarRecordFixture(), "> Empirical record. Every fact is observed server behavior and is not part of the published contract.\n\n", "", 1),
			wantNames: []string{"empirical marker", "restore the", "Empirical record"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := tc.record
			if record == "" {
				record = validGrammarRecordFixture()
			}
			e := tc.specEnum
			if e == nil {
				e = enum
			}
			n := tc.specNested
			if n == nil {
				n = nestedOnly
			}
			v := CheckGrammarFacts(ParseGrammarFactsRecord(record), e, n)
			if len(v) == 0 {
				t.Fatalf("condition did not fire; expected a violation naming %v", tc.wantNames)
			}
			joined := strings.Join(v, "\n")
			for _, want := range tc.wantNames {
				if !strings.Contains(joined, want) {
					t.Errorf("failure did not name %q; got:\n%s", want, joined)
				}
			}
		})
	}
}

// validGrammarRecordFixture is a minimal well-formed record — leading marker,
// header with a Live-facts manifest, a contract-citations section carrying the
// six nested-only types under the explicit label, and two complete facts. Shared
// by the guard condition table and the guard BDD suite.
func validGrammarRecordFixture() string {
	return "> Empirical record. Every fact is observed server behavior and is not part of the published contract.\n\n" +
		"# Change-Set Grammar Facts\n\n" +
		"- **Live facts**: CSG-1, CSG-2\n\n" +
		"## Contract citations\n\n" +
		"- Nested-only types: `CreateAccountability`, `UpdateAccountability`, `RemoveAccountability`, `CreateDomain`, `UpdateDomain`, `RemoveDomain`.\n\n" +
		"## CSG-1 — own-circle policy is a top-level CreatePolicy\n\n" +
		"- **Shape**: a top-level `CreatePolicy` part, not wrapped in an `UpdateRole`\n" +
		"- **Disposition**: accepted\n" +
		"- **Symptom**: a wrapped shape is refused; the web UI emits the top-level form\n" +
		"- **Evidence**: prp_ebe2815f\n" +
		"- **Provenance**: LEARNINGS 2026-08-05 F5\n\n" +
		"## CSG-2 — self-targeting UpdateRole\n\n" +
		"- **Shape**: an `UpdateRole` self-targeting the circle\n" +
		"- **Disposition**: accepted-but-invalid\n" +
		"- **Symptom**: valid: false, blocking alert, available_transitions: []\n" +
		"- **Evidence**: prp_c76cd6bf\n" +
		"- **Provenance**: LEARNINGS 2026-08-05 F6\n"
}

// emptyShellGrammarRecordFixture is the terminal case: every fact retired, the
// manifest left empty, no fact sections — the empty shell the guard rejects
// (condition 3).
func emptyShellGrammarRecordFixture() string {
	return "> Empirical record. Every fact is observed server behavior and is not part of the published contract.\n\n" +
		"# Change-Set Grammar Facts\n\n" +
		"- **Live facts**:\n\n" +
		"## Contract citations\n\n" +
		"- Nested-only types: `CreateAccountability`, `UpdateAccountability`, `RemoveAccountability`, `CreateDomain`, `UpdateDomain`, `RemoveDomain`.\n"
}
