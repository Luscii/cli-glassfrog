package cli

import (
	"fmt"
	"sort"
	"strings"
)

// supportedTensionStatuses is the spec's tension status set — the status enum on
// the Tension schema (spec/glassfrog-api-v5.yaml: unprocessed, processed,
// archived). It is the single source of truth for `tension list --status`
// validation. Adding a value is a one-line change tracking the spec enum. It is a
// NEW set, deliberately distinct from the action/project validateStatus set
// (current/completed/…) in status.go — reusing that set would accept invalid
// tension statuses and reject valid ones (a correctness bug, plan ADR-3). It
// deliberately does not include null/empty: an absent --status is the "no filter"
// case the validator accepts and the query builder omits.
var supportedTensionStatuses = map[string]bool{
	"unprocessed": true,
	"processed":   true,
	"archived":    true,
}

// validateTensionStatus rejects a non-empty --status value outside the tension
// status set, returning a usage error NAMING the unsupported value and listing the
// supported set — before any context assembly or request (the validateMeetingType
// fail-fast discipline, plan ADR-3). An empty value (the flag absent) is valid: no
// status filter, the parameter is omitted from the request. Pure — no network, no
// filesystem — so it runs ahead of any I/O and a tripwire transport can assert
// nothing was sent on rejection. A sibling validator of validateMeetingType /
// validateStatus, not a second copy of any set.
func validateTensionStatus(value string) error {
	if value == "" {
		return nil
	}
	if supportedTensionStatuses[value] {
		return nil
	}
	return fmt.Errorf(
		"unsupported --status value %q — supported: %s",
		value, strings.Join(supportedTensionStatusNames(), ", "),
	)
}

// supportedTensionStatusNames lists the supported tension statuses in stable
// (sorted) order for the usage message, so the same input always yields the same
// deterministic text (the supportedMeetingTypeNames shape).
func supportedTensionStatusNames() []string {
	names := make([]string, 0, len(supportedTensionStatuses))
	for name := range supportedTensionStatuses {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
