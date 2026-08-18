package build

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/grammar"
)

// Agent-Facing Grammar Reference (077) — the derivation half.
//
// `go:embed` cannot reach outside a package's directory tree, and the two sources
// of the change-set grammar live elsewhere in the repository: the vendored v5
// contract (VendoredSpecPath) and the grammar record owned by the
// proposal-drafting skill (GrammarFactsPath, which must stay at its 072 home —
// pointers flow repository → surface only, so no symlink back into internal/ may
// exist). The seam between the two worlds is therefore a generated, committed
// artifact: this file derives it, the generator writes it, and the guard below
// pins the committed file byte-exact to a fresh derivation (plan ADR-1, ADR-5).
//
// NOTHING here is hand-maintained (plan ADR-2). Both halves come from the two
// derivation functions 072 already ships — LoadSpecChangeTypes for the contract
// side, ReadGrammarFactsRecord + ParseGrammarFactsRecord for the record side — so
// the generator and the guard cannot read the sources differently from each
// other, nor from 072's own guard.
//
// This is the one place in internal/build that imports a shipped package
// (internal/grammar). The convention it observes is "build never imports
// internal/cli" (see VersionInjectionTarget); internal/grammar is a stdlib-only
// leaf, and sharing its types is what keeps ONE definition of the artifact's
// shape across generator, guard, and command (interface-cli § "One schema, three
// consumers") instead of a second declaration here to drift from.

const (
	// GrammarArtifactPath is the committed generated artifact, relative to the
	// repository root — the seam between the repository's sources and the shipped
	// binary. Machine-produced and header-marked; a human edits the record, never
	// this file.
	GrammarArtifactPath = "internal/grammar/grammar.json"

	// GrammarRegenerationStep is the command that re-derives the artifact from
	// both sources. It is the remedy every guard failure below names, and the
	// step the artifact's own generated marker carries — single-sourced here so
	// the marker, the failure messages, and the go:generate directive cannot
	// disagree about how to regenerate.
	GrammarRegenerationStep = "go generate ./internal/grammar"

	// grammarArtifactSource names the derivation entry point, so a guard failure
	// points a reader at the code that produced the bytes it is complaining about.
	grammarArtifactSource = "internal/build/grammarartifact.go (BuildGrammarArtifact)"
)

// grammarGeneratedMarker is the do-not-edit marker the envelope carries
// (interface-cli § "The embedded artifact"). It names both sources and the
// regeneration step, so the file explains its own maintenance to whoever opens
// it. Repo-facing only — the accessor never hands it out, so it cannot leak a
// repository path into operator output.
var grammarGeneratedMarker = fmt.Sprintf(
	"DO NOT EDIT. Generated from %s and %s. Regenerate with `%s`; hand edits are rejected by the drift guard in %s.",
	VendoredSpecPath, GrammarFactsPath, GrammarRegenerationStep, grammarArtifactSource,
)

// specWrapperRE captures the clause naming the parts a nested-only type must
// ride inside, from the ProposalChange schema description ("… must appear as
// children of `CreateRole` or `UpdateRole`, not as top-level proposal changes.").
//
// The wrapper pair is DERIVED from the contract's prose rather than checked in as
// a Go table (interface-cli § Consistency Notes left the choice to this task;
// plan ADR-2 forbids a hand-maintained half). The prose parse is viable because
// the clause carries a stable label ("children of") and the derivation is
// self-checking: grammarWrapperTypes rejects an empty result and rejects any
// derived name absent from the contract's own enum, so a reword that breaks the
// parse fails generation loudly in the repository — the designed signal (plan
// R2), never a silently-empty `wrappers` field. That self-check is why the
// sanctioned fallback (a checked-in fact set-compare-guarded against the spec)
// is not needed: there is no second edit site to keep in step.
//
// `[^.]+` bounds the match at the sentence, and the nested-only list that
// precedes it is already consumed by specNestedOnlyRE's own parenthesized group,
// so the two derivations cannot borrow each other's names.
var specWrapperRE = regexp.MustCompile(`children of\s+([^.]+)`)

// grammarWrapperTypes derives the wrapper parts a nested-only change type must
// appear inside, from the ProposalChange description, and validates the result
// against the contract's own enum. Returned sorted, so the artifact's bytes do
// not move if the prose ever names the pair in the other order.
//
// An empty derivation, or a derived name the enum does not carry, is an error:
// the placement rule would otherwise render with no wrappers to name, which
// reads as "there is no rule" — the opposite of what the contract says.
func grammarWrapperTypes(description string, enum []string) ([]string, error) {
	m := specWrapperRE.FindStringSubmatch(description)
	if m == nil {
		return nil, fmt.Errorf("%s: could not locate the nested-only wrapper clause (\"children of …\") in the ProposalChange description — the contract's wording changed; re-derive it in %s", VendoredSpecPath, grammarArtifactSource)
	}
	wrappers := backtickedCamelTokens(m[1])
	if len(wrappers) == 0 {
		return nil, fmt.Errorf("%s: the nested-only wrapper clause names no change type — the contract's wording changed; re-derive it in %s", VendoredSpecPath, grammarArtifactSource)
	}
	enumSet := toStringSet(enum)
	for _, w := range wrappers {
		if !enumSet[w] {
			return nil, fmt.Errorf("%s: the nested-only wrapper clause names %q, absent from ProposalChange.properties.type.enum — the contract's wording changed; re-derive it in %s", VendoredSpecPath, w, grammarArtifactSource)
		}
	}
	return sortedStrings(wrappers), nil
}

// BuildGrammarArtifact derives the whole artifact from the two repository
// sources. It is the single derivation both the generator and the drift guard
// call, so the committed bytes and the guard's expectation can never come from
// two different readings of the same files.
//
// Determinism is a contract, not an accident (interface-cli § "Ordering is
// deterministic"): change types are sorted alphabetically, facts follow the
// record's live-facts manifest order, and the wrapper list is sorted. The same
// two source files always yield the same bytes.
func BuildGrammarArtifact() (grammar.Artifact, error) {
	specRaw, err := readRepoFile(VendoredSpecPath)
	if err != nil {
		return grammar.Artifact{}, err
	}
	recordRaw, err := ReadGrammarFactsRecord()
	if err != nil {
		return grammar.Artifact{}, err
	}
	return BuildGrammarFromSources(specRaw, recordRaw)
}

// BuildGrammarFromSources is BuildGrammarArtifact over injected source bytes — the
// whole derivation, pure over its inputs. Splitting the file reads off lets a test
// (or an acceptance scenario) derive from a perturbed contract or a rewritten
// record — modelling a refresh or a fact retirement — without writing a file, and
// it keeps the derivation itself free of I/O.
func BuildGrammarFromSources(specRaw []byte, recordRaw string) (grammar.Artifact, error) {
	enum, nestedOnly, description, err := ParseSpecChangeTypes(specRaw)
	if err != nil {
		return grammar.Artifact{}, err
	}
	if strings.TrimSpace(description) == "" {
		return grammar.Artifact{}, fmt.Errorf("%s: the ProposalChange schema carries no description — the nested-only rule's source does not resolve", VendoredSpecPath)
	}
	wrappers, err := grammarWrapperTypes(description, enum)
	if err != nil {
		return grammar.Artifact{}, err
	}
	changeTypes, err := grammarChangeTypes(enum, nestedOnly, wrappers)
	if err != nil {
		return grammar.Artifact{}, err
	}
	facts, err := grammarFactEntries(ParseGrammarFactsRecord(recordRaw))
	if err != nil {
		return grammar.Artifact{}, err
	}
	return grammar.Artifact{
		Generated: grammarGeneratedMarker,
		Grammar: grammar.Grammar{
			ChangeTypes: changeTypes,
			Facts:       facts,
		},
	}, nil
}

// grammarChangeTypes projects the contract's enum onto the rendered vocabulary:
// one entry per enumerated type, its placement decided by membership in the
// nested-only set, the wrapper pair carried by the nested-only entries and ONLY
// by them. Sorted alphabetically by type.
//
// A nested-only name the enum does not carry is an error rather than an extra
// entry: it would render a type no proposal can name, and it means the two halves
// of the contract's own ProposalChange schema disagree.
func grammarChangeTypes(enum, nestedOnly, wrappers []string) ([]grammar.ChangeType, error) {
	enumSet := toStringSet(enum)
	for _, n := range nestedOnly {
		if !enumSet[n] {
			return nil, fmt.Errorf("%s: nested-only type %q is absent from ProposalChange.properties.type.enum — the contract's description and enum disagree", VendoredSpecPath, n)
		}
	}
	nestedSet := toStringSet(nestedOnly)

	entries := make([]grammar.ChangeType, 0, len(enum))
	seen := map[string]bool{}
	for _, t := range enum {
		if seen[t] {
			continue // a duplicated enum member renders once
		}
		seen[t] = true
		entry := grammar.ChangeType{
			Type:       t,
			Placement:  grammar.PlacementTopLevel,
			Provenance: grammar.ProvenancePublishedContract,
		}
		if nestedSet[t] {
			entry.Placement = grammar.PlacementNestedOnly
			// A fresh copy per entry: a shared backing array would let a later
			// mutation of one entry's wrappers reach every other entry's.
			entry.Wrappers = append([]string(nil), wrappers...)
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Type < entries[j].Type })
	return entries, nil
}

// grammarFactEntries projects the record's live facts onto the rendered residue,
// in the record's live-facts manifest order — the manifest is the record's own
// statement of what is live, so it is the ordering source rather than section
// order.
//
// The result is always non-nil, so an empty manifest marshals as `facts: []` with
// the key present (interface-cli: the key is never omitted) rather than as
// `null`. A manifest id with no matching section is an error: 072's own guard
// rejects that state, and generating a silently-shorter residue from a
// half-finished retirement would hide it.
func grammarFactEntries(rec GrammarFactsRecord) ([]grammar.Fact, error) {
	sections := make(map[string]GrammarFact, len(rec.Facts))
	for _, f := range rec.Facts {
		sections[f.ID] = f
	}
	entries := make([]grammar.Fact, 0, len(rec.ManifestIDs))
	for _, id := range rec.ManifestIDs {
		f, ok := sections[id]
		if !ok {
			return nil, fmt.Errorf("%s: the Live-facts manifest declares %q but no ## %s section exists — complete the retirement or restore the section, then run `%s`", GrammarFactsPath, id, id, GrammarRegenerationStep)
		}
		entries = append(entries, grammar.Fact{
			ID:          f.ID,
			Title:       f.Title,
			Shape:       f.Fields["Shape"],
			Disposition: f.Disposition(),
			Symptom:     f.Fields["Symptom"],
			Provenance:  grammar.ProvenanceEmpiricalObservation,
		})
	}
	return entries, nil
}

// RenderGrammarArtifact derives the artifact and marshals it to the exact bytes
// the committed file carries: two-space indented JSON with a trailing newline, so
// the file reads as a normal text file in a diff and the guard's byte-comparison
// has one canonical form to compare against.
func RenderGrammarArtifact() ([]byte, error) {
	artifact, err := BuildGrammarArtifact()
	if err != nil {
		return nil, err
	}
	return MarshalGrammarArtifact(artifact)
}

// MarshalGrammarArtifact is the canonical encoding — the single place the on-disk
// form is decided, so the generator and the guard cannot format the same artifact
// differently. Exported so a test can pin a hand-built artifact's bytes without
// reaching for the real sources.
func MarshalGrammarArtifact(artifact grammar.Artifact) ([]byte, error) {
	b, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling the grammar artifact: %w", err)
	}
	return append(b, '\n'), nil
}

// ReadGrammarArtifact reads the committed artifact's raw bytes from the
// repository root.
func ReadGrammarArtifact() ([]byte, error) {
	return readRepoFile(GrammarArtifactPath)
}

// --- The drift guard (T003) -------------------------------------------------
//
// CheckGrammarArtifact is regenerate-and-compare PLUS five named structural
// invariants (plan ADR-5). Byte-equality alone catches every divergence but
// reports it as "bytes differ", which tells a reader nothing about which of the
// three files moved; the invariants make each failure nameable.
//
// The remedy is the same for every finding — run the regeneration step — and it
// is REACHABLE for all of them at once, which is the property that makes the
// message a usable specification rather than a dead end. A fresh derivation is
// canonically encoded (so byte-equality holds), decodable, carries the contract's
// non-empty enum (LoadSpecChangeTypes rejects an empty one), derives its facts
// FROM the manifest (so the id sets agree by construction), stamps both
// provenance tokens from constants, and writes the canonical marker. No finding's
// remedy can therefore break a sibling invariant.
//
// The one divergence a regeneration does NOT fix is a source that has itself gone
// wrong — a record whose manifest names a fact that no longer has a section, or a
// contract whose description no longer states the nested-only rule. Those fail
// the derivation, loudly, with their own remedy (finish the retirement; re-derive
// the citation); they never surface here as a drift finding, because there are no
// regenerated bytes to compare against.

// CheckGrammarArtifact returns one finding per way the committed artifact and the
// repository's sources have diverged. Empty means the artifact is exactly what the
// two sources produce.
//
// committed is the artifact's bytes as checked in; regenerated is the canonical
// encoding of a fresh derivation (RenderGrammarArtifact); manifestIDs is the
// record's live-facts manifest, the independent source the residue's id set is
// checked against.
//
// It never stops at the first failure — one broken invariant must not mask the
// rest — except when the committed artifact does not decode at all, where nothing
// further is evaluable.
func CheckGrammarArtifact(committed, regenerated []byte, manifestIDs []string) []string {
	var findings []string
	remedy := fmt.Sprintf("run `%s`", GrammarRegenerationStep)

	// Invariant 1: the committed artifact decodes.
	var artifact grammar.Artifact
	if err := json.Unmarshal(committed, &artifact); err != nil {
		return []string{fmt.Sprintf("%s does not decode as the grammar artifact envelope (%v) — the ARTIFACT half is corrupt; it is generated, never hand-edited, so %s", GrammarArtifactPath, err, remedy)}
	}

	// Invariant 2: the contract-derived vocabulary is non-empty. An artifact that
	// decodes to nothing renders an empty reference, which reads as an answer.
	if len(artifact.Grammar.ChangeTypes) == 0 {
		findings = append(findings, fmt.Sprintf("%s carries no change types — the CONTRACT-DERIVED half is empty; %s", GrammarArtifactPath, remedy))
	}

	// Invariant 3: the residue's fact ids equal the record's live-facts manifest,
	// in the manifest's order. The manifest is the record's own statement of what
	// is live, so it is the independent side — checking the artifact against the
	// record's sections instead would compare the derivation with itself.
	artifactIDs := make([]string, 0, len(artifact.Grammar.Facts))
	for _, f := range artifact.Grammar.Facts {
		artifactIDs = append(artifactIDs, f.ID)
	}
	if !stringSlicesEqual(artifactIDs, manifestIDs) {
		findings = append(findings, fmt.Sprintf("%s renders facts %v but the record's Live-facts manifest declares %v (%s) — the RECORD-DERIVED half is stale; %s", GrammarArtifactPath, artifactIDs, manifestIDs, GrammarFactsPath, remedy))
	}

	// Invariant 4: every entry in both arrays carries its provenance token. A
	// consumer compares these literally to tell a contract-published shape from a
	// verified observation, so a missing or wrong token silently blurs the two.
	for _, ct := range artifact.Grammar.ChangeTypes {
		if ct.Provenance != grammar.ProvenancePublishedContract {
			findings = append(findings, fmt.Sprintf("%s: change type %q carries provenance %q, want %q — the CONTRACT-DERIVED half lost its provenance marking; %s", GrammarArtifactPath, ct.Type, ct.Provenance, grammar.ProvenancePublishedContract, remedy))
		}
	}
	for _, f := range artifact.Grammar.Facts {
		if f.Provenance != grammar.ProvenanceEmpiricalObservation {
			findings = append(findings, fmt.Sprintf("%s: fact %q carries provenance %q, want %q — the RECORD-DERIVED half lost its provenance marking; %s", GrammarArtifactPath, f.ID, f.Provenance, grammar.ProvenanceEmpiricalObservation, remedy))
		}
	}

	// Invariant 5: the generated marker is present and still says what it is for.
	// It is what tells a reader who opens the file that hand-editing it is futile.
	if !grammarMarkerIsWellFormed(artifact.Generated) {
		findings = append(findings, fmt.Sprintf("%s carries no well-formed generated marker (%q) — the ENVELOPE half is degraded; the marker must say DO NOT EDIT and name the regeneration step, so %s", GrammarArtifactPath, artifact.Generated, remedy))
	}

	// Byte-equality against the fresh derivation, with the diverged half named.
	if !bytesEqualString(committed, regenerated) {
		findings = append(findings, fmt.Sprintf("%s is not byte-identical to a fresh derivation — %s; %s", GrammarArtifactPath, grammarDivergedHalf(artifact, regenerated), remedy))
	}

	return findings
}

// grammarMarkerIsWellFormed reports whether the envelope's marker still states
// both halves of its job: that the file must not be hand-edited, and how to
// regenerate it. Checked for substance rather than exact equality so this
// invariant stays independent of the byte-comparison — a reworded-but-still-honest
// marker is a byte divergence, not a degraded marker.
func grammarMarkerIsWellFormed(marker string) bool {
	if strings.TrimSpace(marker) == "" {
		return false
	}
	return strings.Contains(strings.ToUpper(marker), "DO NOT EDIT") &&
		strings.Contains(marker, GrammarRegenerationStep)
}

// grammarDivergedHalf names which half of the artifact moved, by comparing the
// committed artifact's decoded halves against the fresh derivation's. This is what
// turns "bytes differ" into a sentence a reader can act on: a vendored-contract
// refresh and a record edit produce the same byte difference but different halves.
//
// When every decoded half agrees, the difference is in the encoding itself — a
// hand edit to whitespace, key order, or indentation.
func grammarDivergedHalf(committed grammar.Artifact, regenerated []byte) string {
	var fresh grammar.Artifact
	if err := json.Unmarshal(regenerated, &fresh); err != nil {
		// Unreachable in practice: the regenerated bytes come from marshalling the
		// same type. Named rather than swallowed so a future change that breaks the
		// assumption does not produce a confidently-wrong "half" claim.
		return "the fresh derivation could not be decoded to name the diverged half"
	}
	var halves []string
	if !grammarJSONEqual(committed.Grammar.ChangeTypes, fresh.Grammar.ChangeTypes) {
		halves = append(halves, fmt.Sprintf("the CONTRACT-DERIVED vocabulary differs from %s (a contract refresh outran the artifact)", VendoredSpecPath))
	}
	if !grammarJSONEqual(committed.Grammar.Facts, fresh.Grammar.Facts) {
		halves = append(halves, fmt.Sprintf("the RECORD-DERIVED residue differs from %s (a record edit outran the artifact)", GrammarFactsPath))
	}
	if committed.Generated != fresh.Generated {
		halves = append(halves, "the ENVELOPE's generated marker differs from the canonical one")
	}
	if len(halves) == 0 {
		return "every decoded half agrees, so the ENCODING diverged (the committed file was hand-edited or reformatted)"
	}
	return strings.Join(halves, "; ")
}

// grammarJSONEqual compares two artifact halves by their canonical JSON encoding —
// a deep comparison that needs no per-type equality method and stays correct as
// the field tables grow. A marshal failure reports inequality rather than
// panicking; the caller is already reporting a divergence.
func grammarJSONEqual(a, b any) bool {
	ab, aerr := json.Marshal(a)
	bb, berr := json.Marshal(b)
	if aerr != nil || berr != nil {
		return false
	}
	return string(ab) == string(bb)
}

// bytesEqualString compares two byte slices as strings, so the comparison reads
// the same way the failure message does.
func bytesEqualString(a, b []byte) bool { return string(a) == string(b) }

// stringSlicesEqual reports whether two slices carry the same values in the same
// ORDER — unlike stringSetsEqual, which is order-insensitive. The residue's order
// is part of the interface contract, so a reordered manifest is a real divergence.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// WriteGrammarArtifact derives the artifact and writes it to its committed path.
// The generator's whole body; exported so the generator main stays a thin shell
// and the write path is covered by a test in this package.
func WriteGrammarArtifact() error {
	doc, err := RenderGrammarArtifact()
	if err != nil {
		return err
	}
	root, err := RepoRoot()
	if err != nil {
		return err
	}
	path := filepath.Join(root, filepath.FromSlash(GrammarArtifactPath))
	if err := os.WriteFile(path, doc, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", GrammarArtifactPath, err)
	}
	return nil
}
