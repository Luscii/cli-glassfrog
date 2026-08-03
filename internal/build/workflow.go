package build

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"
)

// WorkflowFileName is the release workflow the guard reads, relative to the
// repository root.
const WorkflowFileName = ".github/workflows/release.yml"

// ReleaseTriggerType is the only GitHub release event type that may trigger the
// pipeline: a *published* release. RoutineTriggers are the event keys that would
// let the pipeline fire on routine activity (a merge, a tag push, a manual or
// scheduled run); the guard rejects any of them so "routine activity triggers no
// build" holds structurally.
const ReleaseTriggerType = "published"

var RoutineTriggers = []string{"push", "pull_request", "workflow_dispatch", "schedule"}

// GoReleaserVersion is the goreleaser-action `version:` the build and tap jobs
// must pin (spec 036). GoReleaser deprecated the `brews` formula publisher in
// favour of `homebrew_casks`; 036/ADR-1 needs the formula (casks are macOS-only,
// the spec needs Linux), so the pipeline stays on this brews-supporting v2 line.
// The guard pins it so an edit that bumps or unpins GoReleaser can't silently
// reintroduce the risk of `brews` being removed upstream and the tap job
// breaking. Bumping is a deliberate act: update this constant AND re-confirm the
// target version still ships `brews` (see .score/memory/DEPRECATION.md).
const GoReleaserVersion = "~> v2.16"

// VerifyRunners is the required mapping from each build target to the GitHub
// native-arch runner that must verify it (interface accord, ADR-3). The mapping
// is load-bearing: TestSelfContainment_HostBinary selects its target binary by
// the runner's *native* GOOS/GOARCH, so a leg pinned to the wrong runner would
// silently verify the host binary instead — defeating the cross-target gate. The
// guard pins each leg to its exact runner so an all-ubuntu matrix fails.
var VerifyRunners = map[string]string{
	"linux/amd64":  "ubuntu-latest",
	"linux/arm64":  "ubuntu-24.04-arm",
	"darwin/amd64": "macos-15-intel",
	"darwin/arm64": "macos-14",
}

// Workflow is the subset of .github/workflows/release.yml the release-workflow
// guard inspects.
type Workflow struct {
	// On is the workflow trigger. GitHub Actions spells this key `on:`, which
	// YAML 1.1 coerces to the boolean true — so once the YAML is converted to
	// JSON (sigs.k8s.io/yaml's path) the trigger lives under the key "true", NOT
	// "on". The json tag matches that coerced key on purpose: tagging it
	// json:"on" would decode nothing into On (the data is under "true"), leaving
	// Triggers zero-valued. That wouldn't pass silently — checkTrigger fails on an
	// empty release.types — but it would be a confusing false failure on a
	// perfectly valid workflow. A round-trip probe confirmed the coercion.
	On          Triggers          `json:"true"`
	Permissions map[string]string `json:"permissions"`
	Jobs        map[string]Job    `json:"jobs"`
}

// Triggers captures the release trigger plus the routine-activity event keys the
// guard must find absent.
type Triggers struct {
	Release          ReleaseTrigger `json:"release"`
	Push             interface{}    `json:"push"`
	PullRequest      interface{}    `json:"pull_request"`
	WorkflowDispatch interface{}    `json:"workflow_dispatch"`
	Schedule         interface{}    `json:"schedule"`
}

// ReleaseTrigger is the `release:` block; Types is the list of release event
// types (the guard requires exactly [published]).
type ReleaseTrigger struct {
	Types []string `json:"types"`
}

// Job is the subset of a workflow job the guard inspects. If is captured so the
// BDD suite can confirm no job gates on the release's *source* (draft-flow vs.
// hand-created) — the pipeline keys on the published event, identically either
// way.
type Job struct {
	RunsOn   string            `json:"runs-on"`
	Needs    StringOrSlice     `json:"needs"`
	If       string            `json:"if"`
	Env      map[string]string `json:"env"`
	Strategy Strategy          `json:"strategy"`
	Steps    []Step            `json:"steps"`
}

// Strategy.Matrix.Include is the explicit per-target matrix the verify job fans
// out over (one entry per build target → native-arch runner).
type Strategy struct {
	Matrix Matrix `json:"matrix"`
}

type Matrix struct {
	Include []map[string]string `json:"include"`
}

// Step is the subset of a job step the guard inspects. With holds the action
// inputs (e.g. goreleaser's `args`, upload/download `path`); Run holds a shell
// step's script.
type Step struct {
	Name string                 `json:"name"`
	Uses string                 `json:"uses"`
	Run  string                 `json:"run"`
	With map[string]interface{} `json:"with"`
}

// StringOrSlice accepts a YAML/JSON value that may be either a scalar string or
// a list of strings — GitHub's `needs:` is written either way (`needs: build`
// or `needs: [build, verify]`). It always normalizes to a slice.
type StringOrSlice []string

func (s *StringOrSlice) UnmarshalJSON(b []byte) error {
	trimmed := strings.TrimSpace(string(b))
	if strings.HasPrefix(trimmed, "[") {
		return json.Unmarshal(b, (*[]string)(s))
	}
	var one string
	if err := json.Unmarshal(b, &one); err != nil {
		return err
	}
	*s = []string{one}
	return nil
}

// LoadWorkflow reads and parses .github/workflows/release.yml from the
// repository root, returning the parsed workflow and its absolute path. A
// missing or unparseable workflow is an error.
func LoadWorkflow() (Workflow, string, error) {
	root, err := RepoRoot()
	if err != nil {
		return Workflow{}, "", err
	}
	path := filepath.Join(root, filepath.FromSlash(WorkflowFileName))
	raw, err := os.ReadFile(path)
	if err != nil {
		return Workflow{}, path, fmt.Errorf("reading %s: %w", path, err)
	}
	wf, err := ParseWorkflow(raw)
	if err != nil {
		return Workflow{}, path, err
	}
	return wf, path, nil
}

// ParseWorkflow decodes release-workflow YAML into Workflow. Split from
// LoadWorkflow so the guard and the BDD suite can exercise mutated workflows
// in-memory without touching the filesystem.
func ParseWorkflow(raw []byte) (Workflow, error) {
	var wf Workflow
	if err := yaml.Unmarshal(raw, &wf); err != nil {
		return Workflow{}, fmt.Errorf("parsing %s: %w", WorkflowFileName, err)
	}
	return wf, nil
}

// CheckReleaseWorkflow returns the list of guard violations for the parsed
// release workflow. An empty result means the workflow triggers, gates, and
// publishes exactly as 022 specifies. Each message names the offending element.
//
// The guard enforces, with change-detector rigor:
//
// Trigger & permissions:
//   - the trigger is exactly `release: { types: [published] }`,
//   - no routine-activity trigger (push/pull_request/workflow_dispatch/schedule)
//     is present, so routine activity never builds,
//   - permissions grant contents: write (the only privilege).
//
// build job:
//   - runs goreleaser with `release ... --skip=publish` (build+package, no upload),
//   - uploads dist/ as a CI artifact.
//
// publish job:
//   - depends on build (so a build failure aborts the whole release),
//   - sets GH_TOKEN in its env (gh authenticates from the env, not permissions),
//   - uploads only the archives + checksums file (never a bare dist/*) to the
//     triggering release with --clobber (idempotent re-publish, assets-only so
//     the release body and pre-release/latest status are preserved).
//
// T003 extends this guard (via CheckVerifyGate) with the cross-target verify
// matrix and publish's dependence on it.
func CheckReleaseWorkflow(wf Workflow) []string {
	var violations []string

	violations = append(violations, checkTrigger(wf.On)...)
	violations = append(violations, checkPermissions(wf.Permissions)...)

	build, ok := wf.Jobs["build"]
	if !ok {
		violations = append(violations, "missing job: build")
	} else {
		violations = append(violations, checkBuildJob(build)...)
	}

	publish, ok := wf.Jobs["publish"]
	if !ok {
		violations = append(violations, "missing job: publish")
	} else {
		violations = append(violations, checkPublishJob(publish)...)
	}

	violations = append(violations, CheckVerifyGate(wf)...)
	violations = append(violations, CheckTapJob(wf)...)

	return violations
}

// CheckVerifyGate enforces the cross-target self-containment gate (022 ADR-3):
//
//   - a verify job exists, fans out over a matrix (runs-on references the matrix
//     runner), and its matrix covers exactly the four supported targets,
//   - verify runs the self-containment check (TestSelfContainment_HostBinary)
//     against the downloaded dist/,
//   - verify depends on build (it checks the built bytes),
//   - publish depends on BOTH build and verify, so any build-or-verification
//     failure aborts before anything is attached.
//
// Split from the rest of the guard so the verify-gate contract is one cohesive,
// separately-testable unit; CheckReleaseWorkflow calls it so the shipped
// workflow is checked as a whole.
func CheckVerifyGate(wf Workflow) []string {
	var violations []string

	verify, ok := wf.Jobs["verify"]
	if !ok {
		violations = append(violations, "missing job: verify — the cross-target self-containment gate")
		// Without a verify job the publish-needs-verify check below is moot; still
		// report the publish gap so both halves of the contract are visible.
	} else {
		if !strings.Contains(verify.RunsOn, "matrix.") {
			violations = append(violations, fmt.Sprintf(
				"verify job must fan out over the matrix (runs-on: ${{ matrix.runner }}), got %q", verify.RunsOn))
		}
		violations = append(violations, checkVerifyMatrix(verify.Strategy.Matrix)...)
		if !runsSelfContainmentCheck(verify) {
			violations = append(violations,
				"verify job must run the self-containment check (TestSelfContainment_HostBinary) against the dist/ artifact")
		}
		if !downloadsDistArtifact(verify) {
			violations = append(violations,
				"verify job must download the dist/ artifact — without it TestSelfContainment_HostBinary falls back to a local `go build` and verifies a rebuild, not the released bytes")
		}
		if !needsContains(verify.Needs, "build") {
			violations = append(violations, "verify job must `needs: build` to check the built dist/ bytes")
		}
	}

	if publish, ok := wf.Jobs["publish"]; ok {
		if !needsContains(publish.Needs, "verify") {
			violations = append(violations,
				"publish job must `needs: [build, verify]` so a self-containment failure aborts the release (nothing published)")
		}
	}

	return violations
}

// TapTokenEnv is the env var the tap job must expose the cross-repo token under
// (036). The .goreleaser.yaml brews.repository.token templates
// `{{ index .Env "HOMEBREW_TAP_TOKEN" }}`, so the job has to inject the secret
// under exactly this name for the brew push to authenticate.
const TapTokenEnv = "HOMEBREW_TAP_TOKEN"

// CheckTapJob enforces the 036 Homebrew formula-publish job contract — the
// CI-testable proxy for the tap scenarios that need a live tap to exercise
// end-to-end:
//
//   - a tap job exists and `needs: [publish]`, so the formula is published only
//     after the assets are attached and the cross-target verify gate passed;
//   - it is gated on a non-prerelease (if: ${{ !github.event.release.prerelease }})
//     — the authoritative stable-only gate (NOT skip_upload: auto, which keys
//     off a semver suffix that can diverge from the GitHub pre-release flag);
//   - it injects the cross-repo token under HOMEBREW_TAP_TOKEN from a secret;
//   - it runs GoReleaser's brew publisher (`goreleaser release`, WITHOUT
//     --skip=publish, which would skip the formula push — the job's purpose);
//   - it does NOT create or modify the GitHub release (no `gh release` step;
//     the release publisher is disabled in config via release.disable: true,
//     which the config-guard pins separately).
//
// Split out so the tap contract is one cohesive, separately-testable unit;
// CheckReleaseWorkflow calls it so the shipped workflow is checked as a whole.
func CheckTapJob(wf Workflow) []string {
	tap, ok := wf.Jobs["tap"]
	if !ok {
		return []string{"missing job: tap — the Homebrew formula publisher (036)"}
	}
	var violations []string

	if !needsContains(tap.Needs, "publish") {
		violations = append(violations,
			"tap job must `needs: [publish]` so the formula references the just-attached, verified assets")
	}
	// The gate must be EXACTLY the non-prerelease flag — not merely contain it.
	// A substring check would accept an augmented condition like
	// `${{ !github.event.release.prerelease || true }}` (always runs) or
	// `&& github.event.release.draft`, which changes behavior while still
	// "mentioning" the flag. Normalize away whitespace and the optional `${{ }}`
	// wrapper, then require exact equality.
	if normalizeIf(tap.If) != "!github.event.release.prerelease" {
		violations = append(violations, fmt.Sprintf(
			"tap job must gate on exactly the non-prerelease flag (if: ${{ !github.event.release.prerelease }}) — the authoritative stable-only gate, got %q", tap.If))
	}
	if !strings.Contains(tap.Env[TapTokenEnv], "secrets.") {
		violations = append(violations, fmt.Sprintf(
			"tap job must inject the cross-repo token as env %s from a secret (${{ secrets.* }}), got %q", TapTokenEnv, tap.Env[TapTokenEnv]))
	}
	args := goreleaserArgs(tap)
	switch {
	case args == "":
		violations = append(violations, "tap job must run the goreleaser-action (the brew publisher)")
	case !runsGoreleaserRelease(args):
		violations = append(violations, fmt.Sprintf(
			"tap job must run `goreleaser release` (the brew publisher), got args %q", args))
	case strings.Contains(args, "--skip=publish"):
		violations = append(violations, fmt.Sprintf(
			"tap job must NOT pass --skip=publish — that skips the brew formula push, which is the job's whole purpose (got args %q)", args))
	}
	if args != "" {
		violations = append(violations, checkGoreleaserVersionPin(tap, "tap")...)
	}
	// Brew-publisher-only: the tap job must never create or modify the GitHub
	// release (that boundary stays with the publish job's `gh release upload`).
	for _, s := range tap.Steps {
		if strings.Contains(s.Run, "gh release") {
			violations = append(violations,
				"tap job must not touch the GitHub release (found a `gh release` step) — asset attachment and release status stay with the publish job")
			break
		}
	}
	return violations
}

// normalizeIf canonicalizes a job `if:` expression for exact comparison: it
// strips all whitespace and the optional `${{ }}` template wrapper, so
// `${{ !github.event.release.prerelease }}` and `!github.event.release.prerelease`
// both reduce to `!github.event.release.prerelease`, while an augmented
// condition (`… || true`, `… && …`) does not.
func normalizeIf(cond string) string {
	// strings.Fields splits on ANY whitespace run (spaces, tabs, newlines), so
	// joining with "" removes all of it — robust to a multi-line `if:`.
	c := strings.Join(strings.Fields(cond), "")
	c = strings.TrimPrefix(c, "${{")
	c = strings.TrimSuffix(c, "}}")
	return c
}

// checkVerifyMatrix asserts the matrix covers exactly the four supported targets
// (darwin/linux × amd64/arm64), each mapped to a runner. A missing target fails
// as loudly as an extra one — the gate must verify every shipped binary.
func checkVerifyMatrix(m Matrix) []string {
	if len(m.Include) == 0 {
		return []string{"verify matrix must enumerate the four target→runner legs (matrix.include is empty)"}
	}
	want := make(map[string]bool, len(SupportedGoos)*len(SupportedGoarch))
	for _, os := range SupportedGoos {
		for _, arch := range SupportedGoarch {
			want[os+"/"+arch] = false
		}
	}
	var violations []string
	for i, leg := range m.Include {
		key := leg["goos"] + "/" + leg["goarch"]
		if _, ok := want[key]; !ok {
			violations = append(violations, fmt.Sprintf(
				"verify matrix leg %d targets unsupported %q", i+1, key))
			continue
		}
		if want[key] {
			violations = append(violations, fmt.Sprintf("verify matrix duplicates target %q", key))
		}
		want[key] = true
		switch runner := strings.TrimSpace(leg["runner"]); runner {
		case "":
			violations = append(violations, fmt.Sprintf("verify matrix target %q has no runner", key))
		case VerifyRunners[key]:
			// correct native-arch runner
		default:
			violations = append(violations, fmt.Sprintf(
				"verify matrix target %q must run on %q (its native-arch runner), got %q — a non-native runner verifies the host binary, not this target",
				key, VerifyRunners[key], runner))
		}
	}
	for key, covered := range want {
		if !covered {
			violations = append(violations, fmt.Sprintf("verify matrix is missing target %q", key))
		}
	}
	return violations
}

// runsSelfContainmentCheck reports whether a verify step invokes 021's
// self-containment test by name.
func runsSelfContainmentCheck(j Job) bool {
	for _, s := range j.Steps {
		if strings.Contains(s.Run, "TestSelfContainment_HostBinary") {
			return true
		}
	}
	return false
}

// checkPermissions enforces that the workflow grants exactly contents: write —
// the least-privilege contract. It fails if contents is not write AND if any
// other permission key is present (e.g. an added actions: write), so the "only
// privilege" claim holds against drift, not just the happy path.
func checkPermissions(perms map[string]string) []string {
	var violations []string
	if perms["contents"] != "write" {
		violations = append(violations, fmt.Sprintf(
			"permissions must grant contents: write, got %q", perms["contents"]))
	}
	var extra []string
	for k := range perms {
		if k != "contents" {
			extra = append(extra, k)
		}
	}
	if len(extra) > 0 {
		sort.Strings(extra)
		violations = append(violations, fmt.Sprintf(
			"permissions must be contents: write only (least-privilege); found extra grant(s): %v", extra))
	}
	return violations
}

// checkTrigger enforces that the only trigger is a published release. A missing
// or extra trigger type fails as loudly as a routine-activity trigger.
func checkTrigger(t Triggers) []string {
	var violations []string
	if len(t.Release.Types) != 1 || t.Release.Types[0] != ReleaseTriggerType {
		violations = append(violations, fmt.Sprintf(
			"trigger must be exactly release types [%s], got %v", ReleaseTriggerType, t.Release.Types))
	}
	routine := map[string]interface{}{
		"push": t.Push, "pull_request": t.PullRequest,
		"workflow_dispatch": t.WorkflowDispatch, "schedule": t.Schedule,
	}
	for _, name := range RoutineTriggers {
		if routine[name] != nil {
			violations = append(violations, fmt.Sprintf(
				"routine-activity trigger %q must be absent — only a published release may trigger the pipeline", name))
		}
	}
	return violations
}

// checkBuildJob enforces the build-once-without-publish + upload-dist contract.
func checkBuildJob(j Job) []string {
	var violations []string
	if j.RunsOn != "ubuntu-latest" {
		violations = append(violations, fmt.Sprintf("build job must run on ubuntu-latest, got %q", j.RunsOn))
	}
	args := goreleaserArgs(j)
	if args == "" {
		violations = append(violations, "build job must run the goreleaser-action (no goreleaser step found)")
	} else {
		if !runsGoreleaserRelease(args) {
			violations = append(violations, fmt.Sprintf("build job must run `goreleaser release` (build+archive+checksum), got args %q", args))
		}
		if !strings.Contains(args, "--skip=publish") {
			violations = append(violations, fmt.Sprintf("build job must pass --skip=publish (GoReleaser must not publish; gh does), got args %q", args))
		}
		violations = append(violations, checkGoreleaserVersionPin(j, "build")...)
	}
	if !uploadsDistArtifact(j) {
		violations = append(violations, "build job must upload dist/ as a CI artifact (so verify/publish read the verified bytes)")
	}
	return violations
}

// checkPublishJob enforces dependence on build, the GH_TOKEN env, and the
// filtered, idempotent asset upload.
func checkPublishJob(j Job) []string {
	var violations []string
	if !needsContains(j.Needs, "build") {
		violations = append(violations, "publish job must `needs: build` so a build failure aborts the release (nothing published)")
	}
	if !strings.Contains(j.Env["GH_TOKEN"], "github.token") {
		violations = append(violations, "publish job must set env GH_TOKEN: ${{ github.token }} — gh authenticates from the env, not permissions alone")
	}
	upload := uploadStepRun(j)
	if upload == "" {
		violations = append(violations, "publish job must upload assets with `gh release upload`")
		return violations
	}
	if !strings.Contains(upload, "github.event.release.tag_name") {
		violations = append(violations, "publish job must upload to the triggering release tag (github.event.release.tag_name)")
	}
	if !strings.Contains(upload, "--clobber") {
		violations = append(violations, "publish job upload must use --clobber so a re-run / partial-failure recovery converges on one asset set")
	}
	// Filtered globs only — never a bare dist/* (which would attach GoReleaser
	// metadata/build dirs). Require both the archive glob and the checksums glob.
	if !strings.Contains(upload, "dist/*.tar.gz") {
		violations = append(violations, "publish job must upload dist/*.tar.gz (the four archives), not a bare dist/*")
	}
	if !strings.Contains(upload, "checksums.txt") {
		violations = append(violations, "publish job must upload the checksums file (dist/*checksums.txt)")
	}
	// A bare dist/* as a SEPARATE argument re-introduces the metadata/build-dir
	// hazard even alongside the filtered globs. Tokenize the upload command and
	// reject an exact dist/* arg (substring matching would false-positive on
	// dist/*.tar.gz / dist/*checksums.txt).
	for _, tok := range strings.Fields(upload) {
		if tok == "dist/*" || tok == "dist/" || tok == "dist" {
			violations = append(violations, fmt.Sprintf(
				"publish job must not upload a bare %q — it attaches GoReleaser metadata/build dirs the spec forbids", tok))
			break
		}
	}
	return violations
}

// runsGoreleaserRelease reports whether the goreleaser-action args invoke the
// `release` subcommand: the FIRST whitespace-separated token must be `release`,
// not merely a string that mentions it (e.g. `build --release-notes` or
// `--skip=release` would pass a substring check while doing something else).
func runsGoreleaserRelease(args string) bool {
	fields := strings.Fields(args)
	return len(fields) > 0 && fields[0] == "release"
}

// goreleaserVersion returns the `version` input of the goreleaser-action step,
// or "" if no such step exists (or it sets no version).
func goreleaserVersion(j Job) string {
	for _, s := range j.Steps {
		if strings.Contains(s.Uses, "goreleaser/goreleaser-action") {
			if v, ok := s.With["version"].(string); ok {
				return v
			}
			return ""
		}
	}
	return ""
}

// checkGoreleaserVersionPin asserts a goreleaser-running job pins the action to
// GoReleaserVersion exactly — so a bump/unpin that could drop the `brews`
// publisher fails the guard rather than shipping silently. jobName names the job
// in the violation message.
func checkGoreleaserVersionPin(j Job, jobName string) []string {
	if v := goreleaserVersion(j); v != GoReleaserVersion {
		return []string{fmt.Sprintf(
			"%s job's goreleaser-action must pin version: %q (the brews-supporting line — a bump/unpin risks losing the deprecated brews publisher), got %q",
			jobName, GoReleaserVersion, v)}
	}
	return nil
}

// goreleaserArgs returns the `args` input of the goreleaser-action step, or ""
// if no such step exists.
func goreleaserArgs(j Job) string {
	for _, s := range j.Steps {
		if strings.Contains(s.Uses, "goreleaser/goreleaser-action") {
			if a, ok := s.With["args"].(string); ok {
				return a
			}
			return "" // action present but no args — treated as "not running release"
		}
	}
	return ""
}

// uploadsDistArtifact reports whether a step uploads the dist/ directory as the
// `dist` CI artifact.
func uploadsDistArtifact(j Job) bool {
	for _, s := range j.Steps {
		if distArtifactStep(s, "upload-artifact") {
			return true
		}
	}
	return false
}

// downloadsDistArtifact reports whether a step downloads the `dist` CI artifact
// into dist/. The verify job must do this so the self-containment check inspects
// the built dist/ binary rather than falling back to a local `go build`.
func downloadsDistArtifact(j Job) bool {
	for _, s := range j.Steps {
		if distArtifactStep(s, "download-artifact") {
			return true
		}
	}
	return false
}

// distArtifactStep reports whether a step uses actions/<action> for the dist
// artifact: the artifact `name` must be exactly "dist" and the `path` exactly
// "dist" or "dist/". The match is exact, NOT substring: a path like "dist2/"
// would satisfy a Contains(path, "dist") check but break HostBinary's discovery,
// which expects <repo>/dist/artifacts.json under a dist/ directory — so a drift
// to a wrong directory or artifact name must fail the guard, not silently let
// the verify step fall back to a local rebuild.
func distArtifactStep(s Step, action string) bool {
	if !strings.Contains(s.Uses, "actions/"+action) {
		return false
	}
	name, _ := s.With["name"].(string)
	path, _ := s.With["path"].(string)
	return name == "dist" && (path == "dist" || path == "dist/")
}

// uploadStepRun returns the shell of the step that runs `gh release upload`, or
// "" if none.
func uploadStepRun(j Job) string {
	for _, s := range j.Steps {
		if strings.Contains(s.Run, "gh release upload") {
			return s.Run
		}
	}
	return ""
}

func needsContains(needs StringOrSlice, want string) bool {
	for _, n := range needs {
		if n == want {
			return true
		}
	}
	return false
}
