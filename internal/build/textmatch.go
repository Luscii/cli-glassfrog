package build

import "strings"

// Spec-neutral text-matching helpers shared by the build-side plugin-artifact
// suites (operator orientation 062, governance navigation 064, …) and by the
// navigation drift guard. They live in their own file rather than in any single
// spec's implementation so removing or renaming one spec cannot break another
// spec's tests.

// mentionsToken reports a case-insensitive word-boundary match for a bare token
// (e.g. a command leaf or a format name) so "roles" matches but "controls" does
// not. A hyphen is a boundary, so "subrole-actors" is matched as the whole leaf.
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

func isWordByte(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}
