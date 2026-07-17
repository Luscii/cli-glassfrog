package build

import "strings"

// Spec-neutral text-matching helpers shared by the build-side plugin-artifact
// suites (operator orientation 062, governance navigation 064, …) and by the
// navigation drift guard. They live in their own file rather than in any single
// spec's implementation so removing or renaming one spec cannot break another
// spec's tests.

// mentionsToken reports a case-insensitive word-boundary match for a bare token
// (e.g. a command leaf or a format name) so "roles" matches "the roles" but not
// "controls". A hyphen counts as a word character (see isWordByte), so "roles"
// does NOT match inside hyphenated prose like "sub-roles" — a leaf must be named
// explicitly. A hyphenated leaf like "subrole-actors" still matches on its
// surrounding boundaries (e.g. the backticks around `subrole-actors`).
func mentionsToken(text, token string) bool {
	low := strings.ToLower(text)
	t := strings.ToLower(token)
	for {
		i := strings.Index(low, t)
		if i < 0 {
			return false
		}
		before := i == 0 || !isWordByte(low[i-1])
		after := i+len(t) >= len(low) || !isWordByte(low[i+len(t)])
		if before && after {
			return true
		}
		low = low[i+len(t):]
	}
}

// isWordByte defines the character class that forms a "word" for mentionsToken's
// boundary check. A hyphen is included so a leaf like "roles" is not matched
// inside "sub-roles" (which would let the drift guard pass while the leaf was no
// longer named explicitly), while hyphenated leaves like "subrole-actors" still
// match against their non-word surroundings.
func isWordByte(b byte) bool {
	return b == '_' || b == '-' || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}
