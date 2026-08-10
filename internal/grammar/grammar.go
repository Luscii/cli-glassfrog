package grammar

import (
	"embed"
	"encoding/json"
	"fmt"
	"sync"
)

// artifactFile is the committed generated artifact's name inside this package.
// The embed and the generator both key on it through this constant, so the two
// cannot disagree about which file is the seam.
const artifactFile = "grammar.json"

// artifactFS carries the committed artifact at compile time, so the reference is
// served by the binary itself: no repository access, no filesystem read, no
// network, and nothing an install layout can leave behind (plan ADR-1). The
// variable is unexported — the only way out of this package is Load, which hands
// out the payload and never the maintenance envelope.
//
//go:embed grammar.json
var artifactFS embed.FS

// loaded caches the one decode. The artifact is immutable for the process's
// lifetime, so decoding it per invocation would be pure waste; caching the ERROR
// alongside the value keeps a corrupt build deterministic rather than
// intermittently succeeding.
var loaded struct {
	once    sync.Once
	grammar Grammar
	err     error
}

// Load returns the change-set grammar the binary carries: the contract-enumerated
// change types with their placement rules, and the empirical residue, each entry
// marked by provenance.
//
// It returns the `grammar` payload and NEVER the envelope — the do-not-edit marker
// is repo-facing maintenance metadata, so there is no path by which it can reach
// operator output. The returned value is a fresh copy each call: the cached decode
// stays immutable, so a caller that reorders or appends to the slices it got
// cannot change what the next caller sees.
//
// A decode failure is returned as an error, never a panic and never a silent zero
// value. The drift guard makes it impossible for a committed artifact to be
// undecodable, so reaching this error means a corrupt binary — the caller
// classifies it as a CLI-internal fault (interface-cli § Error Communication),
// which is why an empty vocabulary is an error too: an artifact that decodes to
// nothing is as corrupt as one that does not decode, and rendering an empty
// reference would look like an answer.
func Load() (Grammar, error) {
	loaded.once.Do(func() {
		loaded.grammar, loaded.err = decodeArtifact()
	})
	if loaded.err != nil {
		return Grammar{}, loaded.err
	}
	return loaded.grammar.clone(), nil
}

// decodeArtifact reads and decodes the embedded artifact. Split from Load so the
// decode is testable directly and the caching stays a one-liner.
func decodeArtifact() (Grammar, error) {
	raw, err := artifactFS.ReadFile(artifactFile)
	if err != nil {
		return Grammar{}, fmt.Errorf("the embedded change-set grammar could not be read: %w", err)
	}
	return decodeArtifactBytes(raw)
}

// decodeArtifactBytes decodes artifact bytes into the payload. Exercised directly
// by the corrupt-artifact tests, which is the only way to reach the failure paths:
// the committed artifact is build-guaranteed decodable.
func decodeArtifactBytes(raw []byte) (Grammar, error) {
	var artifact Artifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		return Grammar{}, fmt.Errorf("the embedded change-set grammar could not be decoded: %w", err)
	}
	if len(artifact.Grammar.ChangeTypes) == 0 {
		return Grammar{}, fmt.Errorf("the embedded change-set grammar carries no change types — the artifact is corrupt or was never generated")
	}
	if artifact.Grammar.Facts == nil {
		// A decoded `null` (or an absent key) becomes a nil slice, which would
		// serialize back out as `null` and break the interface's promise that the
		// key is always an array. Normalize here so every consumer — the templates
		// and the structured output alike — sees the empty-residue case as [].
		artifact.Grammar.Facts = []Fact{}
	}
	return artifact.Grammar, nil
}

// clone returns a deep copy, so the cached decode cannot be mutated through a
// value Load handed out.
func (g Grammar) clone() Grammar {
	out := Grammar{
		ChangeTypes: make([]ChangeType, len(g.ChangeTypes)),
		Facts:       make([]Fact, len(g.Facts)),
	}
	copy(out.Facts, g.Facts)
	for i, ct := range g.ChangeTypes {
		out.ChangeTypes[i] = ct
		if ct.Wrappers != nil {
			out.ChangeTypes[i].Wrappers = append([]string(nil), ct.Wrappers...)
		}
	}
	return out
}
