package build

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// Workflow is the subset of .github/workflows/release.yml the release-workflow
// guard inspects.
type Workflow struct {
	// On is the workflow trigger. GitHub Actions spells this key `on:`, which
	// YAML 1.1 coerces to the boolean true — so once the YAML is converted to
	// JSON (sigs.k8s.io/yaml's path) the trigger lives under the key "true", NOT
	// "on". The json tag matches that coerced key on purpose; "fixing" it to
	// json:"on" would silently parse the trigger as empty and the guard would
	// pass a triggerless workflow. A round-trip probe confirmed the coercion.
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
	if wf.Permissions["contents"] != "write" {
		violations = append(violations, fmt.Sprintf(
			"permissions must grant contents: write (the only privilege), got %q", wf.Permissions["contents"]))
	}

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
		if !strings.Contains(args, "release") {
			violations = append(violations, fmt.Sprintf("build job must run `goreleaser release` (build+archive+checksum), got args %q", args))
		}
		if !strings.Contains(args, "--skip=publish") {
			violations = append(violations, fmt.Sprintf("build job must pass --skip=publish (GoReleaser must not publish; gh does), got args %q", args))
		}
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
	return violations
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

// uploadsDistArtifact reports whether a step uploads the dist/ directory as a CI
// artifact.
func uploadsDistArtifact(j Job) bool {
	for _, s := range j.Steps {
		if strings.Contains(s.Uses, "actions/upload-artifact") {
			if p, ok := s.With["path"].(string); ok && strings.Contains(p, "dist") {
				return true
			}
		}
	}
	return false
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
