// Command gen regenerates the committed change-set grammar artifact
// (internal/grammar/grammar.json) from the two repository sources: the vendored
// Glassfrog API v5 contract and the change-set grammar record owned by the
// proposal-drafting skill.
//
// It is a dev-time tool, invoked through the `go:generate` directive in the
// grammar package it feeds:
//
//	go generate ./internal/grammar
//
// It is deliberately a thin shell over internal/build.WriteGrammarArtifact — the
// derivation lives there because the drift guard must call exactly the same code
// to regenerate in-memory and byte-compare. A generator with its own derivation
// would be a second source of truth for the artifact's content, which is the one
// thing this seam exists to prevent (plan ADR-2, ADR-5).
//
// It is never linked into the shipped binary: the CLI depends on the embedded
// artifact, not on the code that produced it, so the contract and the record stay
// repository-only by construction.
package main

import (
	"fmt"
	"os"

	"github.com/Luscii/cli-glassfrog/internal/build"
)

func main() {
	if err := build.WriteGrammarArtifact(); err != nil {
		fmt.Fprintf(os.Stderr, "regenerating %s failed: %v\n", build.GrammarArtifactPath, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "regenerated %s\n", build.GrammarArtifactPath)
}
