package build

import (
	"encoding/json"
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

// ResolverDefault is the fallback bump the guard pins (030 ADR-2): spec 030
// requires a patch bump when no included PR carries a semver-bearing label.
// Since 071's schema migration it pins the `semver-increment` of the
// condition-less `version-resolver` category — the declared fallback — rather
// than a `version-resolver.default` key. The name encodes the property, not
// the position. A drift away from this fallback would silently change the bump
// for label-less releases, so the guard fails as loudly as a drifted bucket.
const ResolverDefault = "patch"

// NoReleaseNoteLabel is the eighth managed label (ADR-4): the exclusion label
// that must appear in settings.yml, labeler.yml, and release-drafter's
// exclude-labels. ManagedLabelCount is the exact size of the managed set across
// labeler.yml/settings.yml — seven categories plus the exclusion label.
const (
	NoReleaseNoteLabel = "no-release-note"
	ManagedLabelCount  = 8
)

// The category-type vocabulary the parse branches on (071's schema): every
// `categories` entry is a changelog note section (the action's default when
// `type` is omitted), a pre-merge exclusion, or a semver-resolution rule.
const (
	CategoryTypeChangelog       = "changelog"
	CategoryTypePreExclude      = "pre-exclude"
	CategoryTypeVersionResolver = "version-resolver"
)

// ReleaseDrafterConfig is the subset of .github/release-drafter.yml the guard
// inspects. Since 071 the categories list is the single contract source: the
// note-section labels, the exclusion, and the semver buckets plus fallback all
// live there. The hyphenated YAML keys become hyphenated JSON keys under
// sigs.k8s.io/yaml's YAML→JSON path, so the json tags carry the hyphens.
type ReleaseDrafterConfig struct {
	Categories []DrafterCategory `json:"categories"`

	// Rejection detectors only (071 ADR-4): the superseded top-level keys are
	// parsed solely so their PRESENCE can be reported as "this config is on the
	// superseded schema". Nothing is ever read from them for contract purposes,
	// and they exist to be empty — do not "clean them up", or the guard loses
	// its by-name rejection and reports schema drift as missing labels.
	// json.RawMessage because presence is the property: a decoded value type
	// (slice, map) cannot distinguish an absent key from a present-but-empty
	// one (`exclude-labels: []`) or a present-but-null one (`version-resolver:`),
	// and all of those are the superseded schema.
	LegacyVersionResolver json.RawMessage `json:"version-resolver"`
	LegacyExcludeLabels   json.RawMessage `json:"exclude-labels"`
}

// DrafterCategory is one release-drafter `categories` entry. The guard reads
// the type, the condition's labels, and (for version-resolver entries) the
// semver increment; the title is tunable display text.
type DrafterCategory struct {
	Title           string       `json:"title"`
	Type            string       `json:"type"`             // empty means changelog — the action's own default
	SemverIncrement string       `json:"semver-increment"` // read only for version-resolver entries
	When            *DrafterWhen `json:"when"`             // pointer: nil ("no condition" — the fallback) differs from empty

	// Rejection detectors only (071 ADR-4), like the top-level legacy keys:
	// the superseded category-level label shorthands, parsed to be refused.
	// json.RawMessage so presence fires even for `labels: []` or `label: ""`.
	LegacyLabels json.RawMessage `json:"labels"`
	LegacyLabel  json.RawMessage `json:"label"`
}

// DrafterWhen is a category's condition in its mapping form — the only form
// this repository writes (071 ADR-7). A list-form `when` fails the parse
// loudly rather than shipping a permanently untested arm.
type DrafterWhen struct {
	Labels []string `json:"labels"`
	Label  string   `json:"label"` // singular shorthand; folded in with Labels
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
//   - release-drafter's changelog categories' `when` labels == the seven
//     CategoryLabels, and the category labels in labeler.yml and settings.yml
//     (each file's managed set minus the exclusion label) == the same seven;
//   - the version-resolver categories' major/minor/patch buckets == exactly
//     breaking/features/fixes (SemverBuckets), and the condition-less
//     version-resolver category declares the fallback bump ==
//     ResolverDefault (patch);
//   - NoReleaseNoteLabel is present in settings.yml, labeler.yml, AND
//     release-drafter's pre-exclude category;
//   - the managed set in labeler.yml and settings.yml is exactly
//     ManagedLabelCount (eight) entries;
//   - the config is on the current schema (071 ADR-4): the superseded
//     `version-resolver`/`exclude-labels` keys and category-level
//     `labels`/`label` shorthands are rejected by name, so schema drift is
//     never reported as merely-missing labels.
func CheckLabelContract(rd ReleaseDrafterConfig, labeler LabelerConfig, settings SettingsConfig) []string {
	var violations []string

	// (0) The superseded schema is rejected by name first (071 ADR-4), so a
	// whole-file schema problem is never read as a pile of missing labels.
	violations = append(violations, drafterLegacyShape(rd)...)

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

	// (b) version-resolver buckets are exactly the 028 semver-bearing labels,
	// and the fallback default bump is patch (spec 030). Both are derived from
	// the version-resolver categories: conditioned entries form the buckets,
	// the condition-less entry declares the fallback. The action's built-in
	// fallback is also patch, so deleting the declaration changes no observable
	// output — only the absent-declaration violation below can catch it.
	buckets := drafterSemverBuckets(rd)
	violations = append(violations,
		diffLabelSet("version-resolver major", buckets["major"], SemverBuckets["major"])...)
	violations = append(violations,
		diffLabelSet("version-resolver minor", buckets["minor"], SemverBuckets["minor"])...)
	violations = append(violations,
		diffLabelSet("version-resolver patch", buckets["patch"], SemverBuckets["patch"])...)
	switch fallback := drafterFallbackIncrement(rd); fallback {
	case ResolverDefault:
		// declared and correct
	case "":
		violations = append(violations, fmt.Sprintf(
			"version-resolver default must be declared as %q (spec 030: patch bump when no semver-bearing label is present), but no condition-less version-resolver category declares it",
			ResolverDefault))
	default:
		violations = append(violations, fmt.Sprintf(
			"version-resolver default must be %q (spec 030: patch bump when no semver-bearing label is present), got %q",
			ResolverDefault, fallback))
	}

	// (c) the exclusion label is present in all three places.
	if !containsString(settingsNames(settings), NoReleaseNoteLabel) {
		violations = append(violations, fmt.Sprintf(
			"settings.yml must define the %q exclusion label", NoReleaseNoteLabel))
	}
	if !containsString(labelerNames(labeler), NoReleaseNoteLabel) {
		violations = append(violations, fmt.Sprintf(
			"labeler.yml must define the %q exclusion label", NoReleaseNoteLabel))
	}
	if !containsString(drafterExcludedLabels(rd), NoReleaseNoteLabel) {
		violations = append(violations, fmt.Sprintf(
			"release-drafter.yml pre-exclude category must exclude %q", NoReleaseNoteLabel))
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

// categoryType resolves a category entry's effective type: an omitted `type`
// means changelog, the action's own default.
func categoryType(c DrafterCategory) string {
	if c.Type == "" {
		return CategoryTypeChangelog
	}
	return c.Type
}

// whenLabels flattens a condition's labels, folding the singular `label`
// shorthand in with `labels`. A nil condition yields nothing.
func whenLabels(w *DrafterWhen) []string {
	if w == nil {
		return nil
	}
	labels := append([]string(nil), w.Labels...)
	if w.Label != "" {
		labels = append(labels, w.Label)
	}
	return labels
}

// drafterCategoryLabels flattens the `when` labels across the CHANGELOG
// categories only. Including version-resolver entries would triple-count the
// semver labels, and including the pre-exclude entry would read
// no-release-note as an unexpected category label.
func drafterCategoryLabels(rd ReleaseDrafterConfig) []string {
	var labels []string
	for _, c := range rd.Categories {
		if categoryType(c) != CategoryTypeChangelog {
			continue
		}
		labels = append(labels, whenLabels(c.When)...)
	}
	return labels
}

// drafterSemverBuckets groups the conditioned version-resolver categories'
// `when` labels by semver increment. The condition-less fallback entry is
// deliberately excluded — it is drafterFallbackIncrement's subject.
func drafterSemverBuckets(rd ReleaseDrafterConfig) map[string][]string {
	buckets := make(map[string][]string)
	for _, c := range rd.Categories {
		if categoryType(c) != CategoryTypeVersionResolver || c.When == nil {
			continue
		}
		buckets[c.SemverIncrement] = append(buckets[c.SemverIncrement], whenLabels(c.When)...)
	}
	return buckets
}

// drafterFallbackIncrement returns the `semver-increment` of the
// condition-less version-resolver category — the declared fallback — or the
// empty string when no such category exists.
func drafterFallbackIncrement(rd ReleaseDrafterConfig) string {
	for _, c := range rd.Categories {
		if categoryType(c) == CategoryTypeVersionResolver && c.When == nil {
			return c.SemverIncrement
		}
	}
	return ""
}

// drafterExcludedLabels flattens the `when` labels across the pre-exclude
// categories: the labels whose PRs are dropped before drafting.
func drafterExcludedLabels(rd ReleaseDrafterConfig) []string {
	var labels []string
	for _, c := range rd.Categories {
		if categoryType(c) == CategoryTypePreExclude {
			labels = append(labels, whenLabels(c.When)...)
		}
	}
	return labels
}

// drafterLegacyShape is the 071 ADR-4 rejection: a config still expressing the
// contract at the superseded positions fails by NAME, so the failure reads as
// "wrong schema — migrate", never as a pile of missing labels whose obvious
// (and wrong) fix is re-adding the superseded keys.
func drafterLegacyShape(rd ReleaseDrafterConfig) []string {
	var violations []string
	// A RawMessage is non-nil iff the key was present in the file — value
	// irrelevant, so `exclude-labels: []` and a bare `version-resolver:` are
	// rejected the same as populated forms.
	if rd.LegacyVersionResolver != nil {
		violations = append(violations,
			"release-drafter.yml is on the superseded schema: top-level \"version-resolver\" is no longer read — migrate its buckets and default to version-resolver categories (071)")
	}
	if rd.LegacyExcludeLabels != nil {
		violations = append(violations,
			"release-drafter.yml is on the superseded schema: top-level \"exclude-labels\" is no longer read — migrate it to a pre-exclude category (071)")
	}
	for _, c := range rd.Categories {
		if c.LegacyLabels != nil || c.LegacyLabel != nil {
			name := c.Title
			if name == "" {
				name = categoryType(c)
			}
			violations = append(violations, fmt.Sprintf(
				"release-drafter.yml is on the superseded schema: category %q declares labels at the category level — migrate them under the category's \"when\" (071)",
				name))
		}
	}
	return violations
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
