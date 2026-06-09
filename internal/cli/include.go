package cli

import (
	"fmt"
	"sort"
	"strings"
)

// validateIncludeSet rejects any --include value outside the given closed set,
// before any request (the 011 validateInclude shape, pinned by a transport
// tripwire): the API silently ignores an unknown ?include value and returns the
// resource WITHOUT the embed (the silent-wrong-results hazard 013/025 guard
// against), so this fails loud — naming the offending value(s) and the supported
// set. Each unsupported value is quoted individually, the noun agrees in number,
// and values are reported in stable (sorted) order.
//
// It is shared by the tree and subroles reads, each passing its OWN closed set
// (the two reads expose different include vocabularies — accountabilities/
// domains/members for tree, the getRole set for subroles). Never one shared set:
// a shared set would accept a cross-read value the API drops for the other read
// (plan ADR-4 / interface Consistency Notes). The org `roles` read keeps its own
// validateRolesInclude (025) — this generic helper is introduced by 026 and does
// not disturb landed code.
func validateIncludeSet(targets []string, supported map[string]bool) error {
	var unsupported []string
	for _, t := range targets {
		if !supported[t] {
			unsupported = append(unsupported, t)
		}
	}
	if len(unsupported) == 0 {
		return nil
	}
	sort.Strings(unsupported)
	quoted := make([]string, len(unsupported))
	for i, t := range unsupported {
		quoted[i] = fmt.Sprintf("%q", t)
	}
	noun := "value"
	if len(unsupported) > 1 {
		noun = "values"
	}
	return fmt.Errorf(
		"unsupported --include %s %s — supported: %s",
		noun, strings.Join(quoted, ", "), strings.Join(sortedIncludeNames(supported), ", "),
	)
}

// sortedIncludeNames lists a supported-include set in stable (sorted) order for
// the usage message.
func sortedIncludeNames(supported map[string]bool) []string {
	names := make([]string, 0, len(supported))
	for name := range supported {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
