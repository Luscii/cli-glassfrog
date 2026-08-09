package build

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"regexp"
)

// OperatingSurfaceRoot is the repo-relative directory that ships as the agent
// operating surface. The self-containment guard walks every regular file under
// it — the checked set is derived by walking, never enumerated, so a new
// surface file is covered with no registration step.
const OperatingSurfaceRoot = "plugin"

// SurfaceLexiconEntry is one pattern of the two-family deny lexicon. The
// lexicon is the checked-in contract-fact defining "development-repository
// reference" (the scanned set is derived; the definition of forbidden is not
// derivable from repo state). Each entry carries the property it protects and
// a concrete example, so the next editor re-derives the pattern from the
// property instead of copying a value — extending or relaxing an entry is a
// deliberate contract edit, reviewed with its property comment and pinned by a
// guard fixture.
type SurfaceLexiconEntry struct {
	// Family labels the failure mode: "A — resolvable reference" (a pointer an
	// operator without the repository cannot follow) or "B — repo-machinery
	// phrase" (the strict ban: the development repository is not acknowledged
	// even pathlessly).
	Family string
	// Property states what the pattern protects — the reason this reference
	// class is forbidden.
	Property string
	// Example is a concrete violating sample; the guard's fixtures pin that it
	// matches, so a lexicon edit cannot silently un-match a violation class.
	Example string
	// Remedy is the reachable fix stated in every report carrying this entry.
	Remedy  string
	Pattern *regexp.Regexp
}

const (
	surfaceFamilyResolvable = "A — resolvable reference"
	surfaceFamilyMachinery  = "B — repo-machinery phrase"

	surfaceRemedyResolvable = "replace with the in-plugin component name, or remove the reference"
	surfaceRemedyMachinery  = "reword to state the rule through the surface's own consequences, or remove the mention"
)

// SurfaceDenyLexicon is the two-family deny lexicon every line of every walked
// surface file is matched against. Word-boundary and path-shape anchoring keeps
// the known-safe tokens safe (`prp_0123`, version strings, exit codes,
// `--per-page 500`, "the v5 spec", "CLI") — each class is pinned as a guard
// fixture so an edit here cannot silently widen a pattern.
var SurfaceDenyLexicon = []SurfaceLexiconEntry{
	// --- Family A: resolvable references ---------------------------------
	{
		Family:   surfaceFamilyResolvable,
		Property: "a spec-number id — no development-spec catalog exists where the operator stands, so a bare 0NN token is a repo id, never a surface name",
		Example:  "the gated write path (063)",
		Remedy:   surfaceRemedyResolvable,
		// No word boundary sits between `_` and a digit, so example ids like
		// prp_0123 stay safe; version strings (0.34.1) never form 0\d{2} at a
		// boundary.
		Pattern: regexp.MustCompile(`\b0\d{2}\b`),
	},
	{
		Family:   surfaceFamilyResolvable,
		Property: "a repo spec-directory or vendored-spec path — only the slash form is a path; \"the v5 spec\" and \"specification\" stay legal",
		Example:  "spec/glassfrog-api-v5.yaml",
		Remedy:   surfaceRemedyResolvable,
		Pattern:  regexp.MustCompile(`(?i)\bspecs?/`),
	},
	{
		Family:   surfaceFamilyResolvable,
		Property: "a repo source, test, docs, or script tree path",
		Example:  "internal/build/write_safety_guardrail_guard_test.go",
		Remedy:   surfaceRemedyResolvable,
		Pattern:  regexp.MustCompile(`(?i)\b(?:internal|features|docs|scripts)/`),
	},
	{
		Family:   surfaceFamilyResolvable,
		Property: "a pipeline-memory path",
		Example:  ".score/memory/DECISIONS.md",
		Remedy:   surfaceRemedyResolvable,
		// No \b prefix: "." is a non-word character, so \b would require a word
		// character immediately before it and miss ".score/" at the start of a
		// line or after a space.
		Pattern: regexp.MustCompile(`(?i)\.score/`),
	},
	{
		Family:   surfaceFamilyResolvable,
		Property: "a Go source file, including _test.go — repo implementation, invisible to the operator",
		Example:  "write_safety_guardrail_guard_test.go",
		Remedy:   surfaceRemedyResolvable,
		Pattern:  regexp.MustCompile(`(?i)\b[\w-]+\.go\b`),
	},
	{
		Family:   surfaceFamilyResolvable,
		Property: "a pipeline artifact",
		Example:  "the plan.md phasing",
		Remedy:   surfaceRemedyResolvable,
		Pattern:  regexp.MustCompile(`(?i)\b(?:plan|tasks|spec)\.md\b`),
	},
	{
		Family:   surfaceFamilyResolvable,
		Property: "a pipeline interface artifact",
		Example:  "interface-spec.md",
		Remedy:   surfaceRemedyResolvable,
		Pattern:  regexp.MustCompile(`(?i)\binterface-[\w-]+\.md\b`),
	},
	{
		Family:   surfaceFamilyResolvable,
		Property: "a design-history citation — design history lives in the repo, not the surface (case-sensitive)",
		Example:  "(plan ADR-5)",
		Remedy:   surfaceRemedyResolvable,
		Pattern:  regexp.MustCompile(`\bADR-\d+\b`),
	},
	{
		Family:   surfaceFamilyResolvable,
		Property: "a portfolio document — case-SENSITIVE on purpose, so ordinary prose words (decisions, learnings, roadmap) stay legal in the surface",
		Example:  "supersedes LEARNINGS 2026-08-05, F5",
		Remedy:   surfaceRemedyResolvable,
		Pattern:  regexp.MustCompile(`\b(?:FEATURE-MODEL|ROADMAP|BACKLOG|ISSUE-TREE|LEARNINGS|DECISIONS|DEPRECATION|STATUS\.md|PROJECT\.md|VISION\.md|CONSTITUTION\.md)\b`),
	},
	// --- Family B: repo-machinery phrases ---------------------------------
	{
		Family:   surfaceFamilyMachinery,
		Property: "enforcement machinery is repo-side; the surface never names its own watchers",
		Example:  "the drift tripwire turns the build red",
		Remedy:   surfaceRemedyMachinery,
		Pattern:  regexp.MustCompile(`(?i)drift\s+(?:guard|tripwire)`),
	},
	{
		Family:   surfaceFamilyMachinery,
		Property: "the repo, named without a path — the strict ban does not acknowledge the development repository in any form",
		Example:  "a build-time guard in the source repository",
		Remedy:   surfaceRemedyMachinery,
		Pattern:  regexp.MustCompile(`(?i)(?:source|development|parent)\s+repositor(?:y|ies)`),
	},
	{
		Family:   surfaceFamilyMachinery,
		Property: "merge gating is repo machinery — case-sensitive and boundary-anchored so \"CLI\" and lowercase prose stay safe",
		Example:  "turns CI red",
		Remedy:   surfaceRemedyMachinery,
		Pattern:  regexp.MustCompile(`\bCI\b`),
	},
}

// surfacePathPattern finds `plugin/…`-shaped tokens for the positive
// resolution check: any in-surface path a surface file mentions must resolve,
// so in-surface references stay real where the operator stands.
var surfacePathPattern = regexp.MustCompile(`\bplugin/[A-Za-z0-9._/-]+`)

// SurfaceScan is one pass over the operating surface: the derived file set
// (repo-relative, sorted) and every violation found in the run — all of them,
// never first-only.
type SurfaceScan struct {
	Files      []string
	Violations []string
}

// ScanOperatingSurface walks every regular file under root's plugin/ directory
// (all extensions, comment lines included) and matches each line against both
// lexicon families, plus the in-surface path resolution check. A missing
// directory or a walk that finds zero files is a loud error, never a vacuous
// pass; any walk error is a failure, never a skip. Every violation in the run
// is reported — never first-only.
func ScanOperatingSurface(root string) (*SurfaceScan, error) {
	surface := filepath.Join(root, OperatingSurfaceRoot)
	scan := &SurfaceScan{}
	err := filepath.WalkDir(surface, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		scan.Files = append(scan.Files, rel)
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scan.Violations = append(scan.Violations, scanSurfaceContent(root, rel, string(raw))...)
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("surface missing or empty — nothing to verify (%s is not there)", surface)
		}
		return nil, fmt.Errorf("walking the operating surface: %w", err)
	}
	if len(scan.Files) == 0 {
		return nil, fmt.Errorf("surface missing or empty — nothing to verify (the walk under %s found zero files)", surface)
	}
	sort.Strings(scan.Files)
	return scan, nil
}

// scanSurfaceContent matches one file's content, line by line, against both
// lexicon families and the in-surface path resolution check. rel is the
// file's root-relative path (plugin/…); root anchors the resolution check.
func scanSurfaceContent(root, rel, content string) []string {
	var violations []string
	for i, line := range strings.Split(content, "\n") {
		for _, entry := range SurfaceDenyLexicon {
			for _, match := range entry.Pattern.FindAllString(line, -1) {
				violations = append(violations, fmt.Sprintf(
					"%s:%d: forbidden reference %q (family %s: %s). Remedy: %s.",
					rel, i+1, match, entry.Family, entry.Property, entry.Remedy))
			}
		}
		for _, token := range surfacePathPattern.FindAllString(line, -1) {
			// A sentence-ending period is prose punctuation, not part of the path.
			token = strings.TrimRight(token, ".")
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(token))); err != nil {
				violations = append(violations, fmt.Sprintf(
					"%s:%d: dangling in-surface path %q. Remedy: correct the path to the existing in-surface file, or remove the reference.",
					rel, i+1, token))
			}
		}
	}
	return violations
}
