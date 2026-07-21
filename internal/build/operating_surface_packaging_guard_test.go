package build

import "testing"

// TestMarketplaceConsistencyGuard is the best-effort consistency guard for
// Operating-Surface Packaging (070, plan ADR-5). It keeps the repo-root
// marketplace manifest truthful to the plugin it ships: a marketplace entry
// that drifts from the committed plugin definition is a defect, not a
// difference (spec accord), and this guard turns that drift into a build
// failure before an operator ever sees a listing that lies.
//
// Both sides of every comparison are read from disk at test time — the entry
// from .claude-plugin/marketplace.json, the plugin identity from
// plugin/.claude-plugin/plugin.json — so the guard hard-codes none of the
// guarded VALUES. The one thing it must name is the install identity
// (MarketplacePluginName, "glassfrog"): the documented install command
// (/plugin install glassfrog@glassfrog, plan ADR-1) depends on that exact
// name, so it is pinned as a checked-in contract fact and cross-checked
// against the plugin manifest's own name below — a rename on either side
// surfaces here.
//
// The entry lookup is existence-based (a plugins entry named glassfrog
// EXISTS), never "is the only entry" — the marketplace is general in shape
// (plan ADR-2) and a future sibling plugin must be one appended entry that
// leaves this guard untouched.
//
// COVERAGE (explicitly partial — stated, not silent): this guard verifies
// IN-REPO consistency only —
//   - the manifest parses and carries the glassfrog entry;
//   - the entry's `source` resolves to a directory containing a valid plugin
//     manifest;
//   - entry `name` and `description` equal the plugin manifest's (identity
//     and verbatim-equal description);
//   - the entry carries NO `version` key (the version stays single-sourced in
//     plugin.json; the in-repo source installs the checkout's version).
//
// NOT COVERED (no in-repo source to anchor against; left to review and the
// host): the host's marketplace schema and its evolution, the install flow
// itself, and the README/guide install prose (not enumerable).
func TestMarketplaceConsistencyGuard(t *testing.T) {
	manifest, _, err := ReadMarketplaceManifest()
	if err != nil {
		t.Fatalf("could not read the repo-root marketplace manifest (%s): %v", MarketplaceManifestPath, err)
	}

	entry, found := FindMarketplacePlugin(manifest, MarketplacePluginName)
	if !found {
		t.Fatalf("the marketplace manifest (%s) carries no plugins entry named %q — the documented install command (/plugin install %s@%s) would find nothing to install", MarketplaceManifestPath, MarketplacePluginName, MarketplacePluginName, manifest.Name)
	}

	// The entry's source must resolve, within the repo, to a directory that
	// contains a valid plugin manifest — the "entry no longer resolves to a
	// matching plugin definition" defect.
	plugin, err := MarketplaceSourcePluginManifest(entry.Source)
	if err != nil {
		t.Fatalf("the %q entry's source does not resolve to the committed plugin definition: %v", entry.Name, err)
	}

	// The integrated identity check: entry name and description must equal the
	// plugin manifest's, and no version pin may appear on the entry. Each
	// finding names its offending field.
	if findings := CheckMarketplaceEntryConsistency(entry, plugin); len(findings) != 0 {
		t.Fatalf("the marketplace entry drifted from the plugin it ships:\n  - %s", joinDrift(findings))
	}

	// The contract-fact cross-check: the pinned install identity must be the
	// plugin manifest's own name (both files just read from disk), so
	// MarketplacePluginName can never silently diverge from the plugin it
	// names.
	if plugin.Name != MarketplacePluginName {
		t.Errorf("the plugin manifest's name %q no longer matches the pinned install identity %q — the documented install command drifted; reconcile MarketplacePluginName with %s", plugin.Name, MarketplacePluginName, OrientationManifestPath)
	}
}
