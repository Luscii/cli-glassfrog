package cli

import (
	"fmt"
	"sort"
	"strings"
)

// supportedActionStatuses is the spec's action status set — the `status` enum on
// GET /me/actions (spec/glassfrog-api-v5.yaml: archived, cancelled, completed,
// current, scheduled, someday, waiting). It is the single source of truth for
// --status validation, shared by My Actions (013, introduced here) and My
// Projects (014), so the set is never inlined a second time. Adding a status is a
// one-line change tracking the spec enum (the planning-adjustable risk).
var supportedActionStatuses = map[string]bool{
	"archived":  true,
	"cancelled": true,
	"completed": true,
	"current":   true,
	"scheduled": true,
	"someday":   true,
	"waiting":   true,
}

// validateStatus rejects a non-empty --status value outside the spec's status
// set, returning a usage error NAMING the unsupported value and listing the
// supported set — before any context assembly or request (the 011 validateInclude
// fail-fast discipline, ADR-2). An empty value (the flag absent) is valid: no
// status constraint. The validator is pure — no network, no filesystem — so it
// runs ahead of any I/O and a tripwire transport can assert nothing was sent on
// rejection. Shared verbatim by My Projects (014).
func validateStatus(status string) error {
	if status == "" {
		return nil
	}
	if supportedActionStatuses[status] {
		return nil
	}
	return fmt.Errorf(
		"unsupported --status value %q — supported: %s",
		status, strings.Join(supportedStatusNames(), ", "),
	)
}

// supportedStatusNames lists the supported statuses in stable (sorted) order for
// the usage message, so the same input always yields the same deterministic text.
func supportedStatusNames() []string {
	names := make([]string, 0, len(supportedActionStatuses))
	for name := range supportedActionStatuses {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
