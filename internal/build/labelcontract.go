package build

import (
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

// The three config files whose label contract this guard pins (spec 030, ADR-6).
// All paths are relative to the repository root.
const (
	ReleaseDrafterFileName = ".github/release-drafter.yml"
	LabelerFileName        = ".github/labeler.yml"
	SettingsFileName       = ".github/settings.yml"
)

// CategoryLabels is the closed set of the seven 028 category/semver labels that
// release-drafter's `categories` map to note sections and that 028's
// labeler.yml/settings.yml must define. It is the cross-feature contract 028
// produces and 030 consumes (ADR-3). The guard treats a missing label as loudly
// as an extra one (change-detector rigor): renaming, dropping, or adding a
// category label in one file without the others fails.
var CategoryLabels = []string{
	"breaking", "features", "fixes", "docs", "infrastructure", "dependencies", "internal",
}

// SemverBuckets is the exact version-resolver mapping (ADR-2): the major bucket
// is exactly [breaking], minor exactly [features], patch exactly [fixes]. A
// bucket drifting from its single 028 semver-bearing label fails the guard.
var SemverBuckets = map[string][]string{
	"major": {"breaking"},
	"minor": {"features"},
	"patch": {"fixes"},
}

// NoReleaseNoteLabel is the eighth managed label (ADR-4): the exclusion label
// that must appear in settings.yml, labeler.yml, and release-drafter's
// exclude-labels. ManagedLabelCount is the exact size of the managed set across
// labeler.yml/settings.yml — seven categories plus the exclusion label.
const (
	NoReleaseNoteLabel = "no-release-note"
	ManagedLabelCount  = 8
)

// ReleaseDrafterConfig is the subset of .github/release-drafter.yml the guard
// inspects: the categories (label→section mapping), the version-resolver
// buckets, and the exclude-labels list. The hyphenated YAML keys become
// hyphenated JSON keys under sigs.k8s.io/yaml's YAML→JSON path, so the json tags
// carry the hyphens.
type ReleaseDrafterConfig struct {
	Categories      []DrafterCategory `json:"categories"`
	VersionResolver VersionResolver   `json:"version-resolver"`
	ExcludeLabels   []string          `json:"exclude-labels"`
}

// DrafterCategory is one release-drafter `categories` entry: a display title and
// the labels routed into that section. The guard reads only the labels (the
// title is tunable display text).
type DrafterCategory struct {
	Title  string   `json:"title"`
	Labels []string `json:"labels"`
}

// VersionResolver mirrors release-drafter's version-resolver: the three semver
// buckets, each carrying the labels that trigger that bump. Default is captured
// for completeness/debugging but is not asserted by this guard (the interface
// guard contract pins the buckets, not the default).
type VersionResolver struct {
	Major   ResolverBucket `json:"major"`
	Minor   ResolverBucket `json:"minor"`
	Patch   ResolverBucket `json:"patch"`
	Default string         `json:"default"`
}

// ResolverBucket is one version-resolver bucket's label list.
type ResolverBucket struct {
	Labels []string `json:"labels"`
}

// LabelerConfig is the subset of .github/labeler.yml the guard inspects: the
// list of managed-label entries. Only the label name matters here — the
// matchers (title/branch/files/negate) are 028's concern, not the contract.
type LabelerConfig struct {
	Labels []LabelerEntry `json:"labels"`
}

// LabelerEntry is one labeler.yml entry; Label is the managed label name.
type LabelerEntry struct {
	Label string `json:"label"`
}

// SettingsConfig is the subset of .github/settings.yml the guard inspects: the
// label catalog. Only the names matter for the contract (colors/descriptions
// are tunable).
type SettingsConfig struct {
	Labels []SettingsLabel `json:"labels"`
}

// SettingsLabel is one settings.yml catalog entry; Name is the managed label.
type SettingsLabel struct {
	Name string `json:"name"`
}

// LoadLabelContract reads and parses the three config files from the repository
// root, returning the parsed configs. A missing or unparseable file is an error
// (the RED state before the files exist).
func LoadLabelContract() (ReleaseDrafterConfig, LabelerConfig, SettingsConfig, error) {
	root, err := RepoRoot()
	if err != nil {
		return ReleaseDrafterConfig{}, LabelerConfig{}, SettingsConfig{}, err
	}

	var rd ReleaseDrafterConfig
	if err := loadYAML(root, ReleaseDrafterFileName, &rd); err != nil {
		return ReleaseDrafterConfig{}, LabelerConfig{}, SettingsConfig{}, err
	}
	var labeler LabelerConfig
	if err := loadYAML(root, LabelerFileName, &labeler); err != nil {
		return ReleaseDrafterConfig{}, LabelerConfig{}, SettingsConfig{}, err
	}
	var settings SettingsConfig
	if err := loadYAML(root, SettingsFileName, &settings); err != nil {
		return ReleaseDrafterConfig{}, LabelerConfig{}, SettingsConfig{}, err
	}
	return rd, labeler, settings, nil
}

// loadYAML reads name (relative to root) and unmarshals it into dst.
func loadYAML(root, name string, dst interface{}) error {
	path := filepath.Join(root, filepath.FromSlash(name))
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", name, err)
	}
	if err := yaml.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("parsing %s: %w", name, err)
	}
	return nil
}

// CheckLabelContract returns the list of guard violations for the parsed label
// contract. An empty result means all eight labels agree across the three files
// (ADR-6). Each message names the offending file and label so a reviewer can fix
// the drift without re-reading the configs.
//
// The guard enforces, with change-detector rigor (a missing entry fails as
// loudly as an extra one):
//
//   - release-drafter `categories` labels == the seven CategoryLabels, and the
//     category labels in labeler.yml and settings.yml (each file's managed set
//     minus the exclusion label) == the same seven;
//   - the version-resolver major/minor/patch buckets == exactly
//     breaking/features/fixes (SemverBuckets);
//   - NoReleaseNoteLabel is present in settings.yml, labeler.yml, AND
//     release-drafter's exclude-labels;
//   - the managed set in labeler.yml and settings.yml is exactly
//     ManagedLabelCount (eight) entries.
func CheckLabelContract(rd ReleaseDrafterConfig, labeler LabelerConfig, settings SettingsConfig) []string {
	var violations []string

	// (a) Category labels agree across all three files with the canonical seven.
	// Comparing each file to CategoryLabels also catches a *coordinated* rename
	// (all three drift together) — ADR-3 wants that to fail loudly, forcing a
	// deliberate guard update.
	violations = append(violations,
		diffLabelSet("release-drafter.yml category", drafterCategoryLabels(rd), CategoryLabels)...)
	violations = append(violations,
		diffLabelSet("labeler.yml category", categoryLabelsOf(labelerNames(labeler)), CategoryLabels)...)
	violations = append(violations,
		diffLabelSet("settings.yml category", categoryLabelsOf(settingsNames(settings)), CategoryLabels)...)

	// (b) version-resolver buckets are exactly the 028 semver-bearing labels.
	violations = append(violations,
		diffLabelSet("version-resolver major", rd.VersionResolver.Major.Labels, SemverBuckets["major"])...)
	violations = append(violations,
		diffLabelSet("version-resolver minor", rd.VersionResolver.Minor.Labels, SemverBuckets["minor"])...)
	violations = append(violations,
		diffLabelSet("version-resolver patch", rd.VersionResolver.Patch.Labels, SemverBuckets["patch"])...)

	// (c) the exclusion label is present in all three places.
	if !containsString(settingsNames(settings), NoReleaseNoteLabel) {
		violations = append(violations, fmt.Sprintf(
			"settings.yml must define the %q exclusion label", NoReleaseNoteLabel))
	}
	if !containsString(labelerNames(labeler), NoReleaseNoteLabel) {
		violations = append(violations, fmt.Sprintf(
			"labeler.yml must define the %q exclusion label", NoReleaseNoteLabel))
	}
	if !containsString(rd.ExcludeLabels, NoReleaseNoteLabel) {
		violations = append(violations, fmt.Sprintf(
			"release-drafter.yml exclude-labels must contain %q", NoReleaseNoteLabel))
	}

	// (d) the managed set is exactly eight (seven categories + exclusion) in the
	// two files that own the catalog/matcher. release-drafter's categories are
	// covered by (a); this pins the catalog/matcher files so a ninth label can't
	// slip in unnoticed.
	if n := len(uniqueStrings(labelerNames(labeler))); n != ManagedLabelCount {
		violations = append(violations, fmt.Sprintf(
			"labeler.yml must manage exactly %d labels (seven categories + %q), found %d",
			ManagedLabelCount, NoReleaseNoteLabel, n))
	}
	if n := len(uniqueStrings(settingsNames(settings))); n != ManagedLabelCount {
		violations = append(violations, fmt.Sprintf(
			"settings.yml must catalog exactly %d labels (seven categories + %q), found %d",
			ManagedLabelCount, NoReleaseNoteLabel, n))
	}

	return violations
}

// drafterCategoryLabels flattens the labels across all release-drafter
// categories into one slice (each category entry carries its own labels list).
func drafterCategoryLabels(rd ReleaseDrafterConfig) []string {
	var labels []string
	for _, c := range rd.Categories {
		labels = append(labels, c.Labels...)
	}
	return labels
}

// labelerNames returns every managed label name declared in labeler.yml.
func labelerNames(l LabelerConfig) []string {
	names := make([]string, 0, len(l.Labels))
	for _, e := range l.Labels {
		names = append(names, e.Label)
	}
	return names
}

// settingsNames returns every managed label name catalogued in settings.yml.
func settingsNames(s SettingsConfig) []string {
	names := make([]string, 0, len(s.Labels))
	for _, e := range s.Labels {
		names = append(names, e.Name)
	}
	return names
}

// categoryLabelsOf returns the managed names with the exclusion label removed —
// i.e. the category labels a file should declare. An extra non-category,
// non-exclusion label survives the filter and so is caught by the category diff
// (as an "unexpected ... declared" violation) rather than hiding.
func categoryLabelsOf(names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		if n != NoReleaseNoteLabel {
			out = append(out, n)
		}
	}
	return out
}

// diffLabelSet compares a declared label set against the expected closed set,
// emitting a violation for every unexpected (extra) label and every missing
// required one. context names the file/section so the message is actionable.
// Mirrors config.go's diffTargetSet, specialized for label wording.
func diffLabelSet(context string, declared, expected []string) []string {
	var violations []string

	expectedSet := make(map[string]bool, len(expected))
	for _, e := range expected {
		expectedSet[e] = true
	}
	declaredSet := make(map[string]bool, len(declared))
	for _, d := range declared {
		declaredSet[d] = true
		if !expectedSet[d] {
			violations = append(violations, fmt.Sprintf("unexpected %s label declared: %q", context, d))
		}
	}
	for _, e := range expected {
		if !declaredSet[e] {
			violations = append(violations, fmt.Sprintf("required %s label missing: %q", context, e))
		}
	}
	return violations
}

// containsString reports whether want is in xs.
func containsString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

// uniqueStrings returns the distinct values in xs, preserving first-seen order.
// Used so the managed-count check counts distinct labels (a duplicated entry
// can't pad the set to eight).
func uniqueStrings(xs []string) []string {
	seen := make(map[string]bool, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}
