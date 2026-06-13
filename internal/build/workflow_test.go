package build

import (
	"strings"
	"testing"
)

// verifyJobBlock is the verify matrix job, factored out so a drift case can
// excise it exactly to test the fully-missing-verify path. It begins at
// `  verify:` and ends with a trailing newline so it splices cleanly between
// the build job's last line and `  publish:`.
const verifyJobBlock = `  verify:
    needs: build
    runs-on: ${{ matrix.runner }}
    strategy:
      fail-fast: true
      matrix:
        include:
          - { goos: linux, goarch: amd64, runner: ubuntu-latest }
          - { goos: linux, goarch: arm64, runner: ubuntu-24.04-arm }
          - { goos: darwin, goarch: amd64, runner: macos-13 }
          - { goos: darwin, goarch: arm64, runner: macos-14 }
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: actions/download-artifact@v4
        with:
          name: dist
          path: dist/
      - name: Verify ${{ matrix.goos }}/${{ matrix.goarch }} self-containment
        run: go test -run '^TestSelfContainment_HostBinary$' -v ./internal/build
`

// tapJobBlock is the 036 Homebrew tap-publish job, factored out so a drift case
// can excise it to test the fully-missing-tap path. It begins at `  tap:` and
// ends with a trailing newline so it appends cleanly after the publish job.
const tapJobBlock = `  tap:
    needs: [publish]
    if: ${{ !github.event.release.prerelease }}
    runs-on: ubuntu-latest
    env:
      HOMEBREW_TAP_TOKEN: ${{ secrets.CI_GITHUB_TOKEN }}
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
          ref: ${{ github.event.release.tag_name }}
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2.16"
          args: release --clean
`

// validWorkflowYAML mirrors the shipped .github/workflows/release.yml
// (build → verify matrix → publish, publish needs [build, verify]). The drift
// cases mutate copies of this baseline so each test changes exactly one thing.
// TestReleaseWorkflow_RealWorkflow pins the real file separately.
const validWorkflowYAML = `
name: release
on:
  release:
    types: [published]
permissions:
  contents: write
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean --skip=publish
      - uses: actions/upload-artifact@v4
        with:
          name: dist
          path: dist/
` + verifyJobBlock + `  publish:
    needs: [build, verify]
    runs-on: ubuntu-latest
    env:
      GH_TOKEN: ${{ github.token }}
    steps:
      - uses: actions/download-artifact@v4
        with:
          name: dist
          path: dist/
      - name: Attach archives and checksums to the release
        run: |
          gh release upload "${{ github.event.release.tag_name }}" \
            dist/*.tar.gz dist/*checksums.txt \
            --clobber
` + tapJobBlock

// TestReleaseWorkflow_RealWorkflow is the change-detector against the shipped
// .github/workflows/release.yml: the guard must pass on the real file, so a
// future edit that breaks the trigger, the gating, or the filtered upload fails
// here.
func TestReleaseWorkflow_RealWorkflow(t *testing.T) {
	wf, path, err := LoadWorkflow()
	if err != nil {
		t.Fatalf("loading the release workflow: %v", err)
	}
	if violations := CheckReleaseWorkflow(wf); len(violations) != 0 {
		t.Fatalf("the shipped %s must pass the release-workflow guard, got violations:\n  %s",
			path, strings.Join(violations, "\n  "))
	}
}

// TestReleaseWorkflow_Drift exercises the guard against in-memory mutated
// workflows. Change-detector rigor: a routine-activity trigger, a lost gate, a
// missing GH_TOKEN, or a bare dist/* upload must each fail and name the offence.
func TestReleaseWorkflow_Drift(t *testing.T) {
	cases := []struct {
		name      string
		yaml      string
		wantPass  bool
		wantNamed []string
	}{
		{
			name:     "the canonical build+publish workflow passes",
			yaml:     validWorkflowYAML,
			wantPass: true,
		},
		{
			name: "a push trigger (routine activity) is rejected and named",
			yaml: strings.Replace(validWorkflowYAML,
				"on:\n  release:\n    types: [published]\n",
				"on:\n  push:\n    branches: [main]\n  release:\n    types: [published]\n", 1),
			wantPass:  false,
			wantNamed: []string{"push", "routine-activity"},
		},
		{
			name: "a release type other than published is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				"types: [published]", "types: [created]", 1),
			wantPass:  false,
			wantNamed: []string{"published"},
		},
		{
			name: "missing contents: write is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				"permissions:\n  contents: write\n", "permissions:\n  contents: read\n", 1),
			wantPass:  false,
			wantNamed: []string{"contents: write"},
		},
		{
			name: "a build job that omits --skip=publish is rejected (GoReleaser must not publish)",
			yaml: strings.Replace(validWorkflowYAML,
				"args: release --clean --skip=publish", "args: release --clean", 1),
			wantPass:  false,
			wantNamed: []string{"--skip=publish"},
		},
		{
			name: "publish not depending on build (no abort gate) is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				"  publish:\n    needs: [build, verify]\n", "  publish:\n", 1),
			wantPass:  false,
			wantNamed: []string{"needs: build"},
		},
		{
			name: "publish not depending on verify (verification not gating) is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				"    needs: [build, verify]\n", "    needs: [build]\n", 1),
			wantPass:  false,
			wantNamed: []string{"needs: [build, verify]"},
		},
		{
			name: "a verify job without needs: build is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				"  verify:\n    needs: build\n    runs-on: ${{ matrix.runner }}\n",
				"  verify:\n    runs-on: ${{ matrix.runner }}\n", 1),
			wantPass:  false,
			wantNamed: []string{"verify job must `needs: build`"},
		},
		{
			name: "a fully missing verify job (no self-containment gate at all) is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				verifyJobBlock, "", 1),
			wantPass:  false,
			wantNamed: []string{"missing job: verify"},
		},
		{
			name: "a verify matrix missing a target (darwin/arm64 dropped) is rejected and named",
			yaml: strings.Replace(validWorkflowYAML,
				"          - { goos: darwin, goarch: arm64, runner: macos-14 }\n", "", 1),
			wantPass:  false,
			wantNamed: []string{"missing target", "darwin/arm64"},
		},
		{
			name: "a verify job that does not run the self-containment check is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				"run: go test -run '^TestSelfContainment_HostBinary$' -v ./internal/build",
				"run: echo skip", 1),
			wantPass:  false,
			wantNamed: []string{"TestSelfContainment_HostBinary"},
		},
		{
			name: "publish without GH_TOKEN in env is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				"    env:\n      GH_TOKEN: ${{ github.token }}\n", "", 1),
			wantPass:  false,
			wantNamed: []string{"GH_TOKEN"},
		},
		{
			name: "a bare dist/* upload (no filtered globs) is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				"dist/*.tar.gz dist/*checksums.txt \\", "dist/* \\", 1),
			wantPass:  false,
			wantNamed: []string{"dist/*.tar.gz"},
		},
		{
			name:      "an upload without --clobber (non-idempotent) is rejected",
			yaml:      strings.Replace(validWorkflowYAML, "            --clobber\n", "", 1),
			wantPass:  false,
			wantNamed: []string{"--clobber"},
		},
		{
			// #3 — an extra permission breaks the least-privilege contract.
			name: "an extra permission grant (actions: write) is rejected and named",
			yaml: strings.Replace(validWorkflowYAML,
				"permissions:\n  contents: write\n", "permissions:\n  contents: write\n  actions: write\n", 1),
			wantPass:  false,
			wantNamed: []string{"least-privilege", "actions"},
		},
		{
			// #4 — a bare dist/* arg alongside the filtered globs still attaches junk.
			name: "an additional bare dist/* arg alongside the filtered globs is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				"dist/*.tar.gz dist/*checksums.txt \\", "dist/*.tar.gz dist/*checksums.txt dist/* \\", 1),
			wantPass:  false,
			wantNamed: []string{"bare", "dist/*"},
		},
		{
			// #5 — a verify leg on a non-native runner silently verifies the host binary.
			name: "a verify leg on the wrong (non-native) runner is rejected and named",
			yaml: strings.Replace(validWorkflowYAML,
				"          - { goos: darwin, goarch: arm64, runner: macos-14 }\n",
				"          - { goos: darwin, goarch: arm64, runner: ubuntu-latest }\n", 1),
			wantPass:  false,
			wantNamed: []string{"native-arch runner", "darwin/arm64", "macos-14"},
		},
		{
			// #6 — without the dist/ download, verify checks a local rebuild, not the
			// released bytes. Removes verify's download step (the first occurrence;
			// publish's identical step, which comes later, is left intact).
			name: "a verify job that does not download dist/ is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				"      - uses: actions/download-artifact@v4\n        with:\n          name: dist\n          path: dist/\n",
				"", 1),
			wantPass:  false,
			wantNamed: []string{"verify job must download"},
		},
		{
			// round 2 #1 — a substring-y upload path (dist2/) must NOT satisfy the
			// guard: HostBinary discovery needs a real dist/ directory.
			name: "a build upload-artifact path of dist2/ (substring false-positive) is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				"      - uses: actions/upload-artifact@v4\n        with:\n          name: dist\n          path: dist/\n",
				"      - uses: actions/upload-artifact@v4\n        with:\n          name: dist\n          path: dist2/\n", 1),
			wantPass:  false,
			wantNamed: []string{"build job must upload dist"},
		},
		{
			// round 2 #1 — the artifact name is pinned too; a renamed artifact breaks
			// the download-by-name handshake.
			name: "a build upload-artifact named something other than dist is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				"      - uses: actions/upload-artifact@v4\n        with:\n          name: dist\n          path: dist/\n",
				"      - uses: actions/upload-artifact@v4\n        with:\n          name: dist-bin\n          path: dist/\n", 1),
			wantPass:  false,
			wantNamed: []string{"build job must upload dist"},
		},
		{
			// round 2 #2 — same exact-match rule for verify's download (first
			// download-artifact occurrence; publish's is left intact).
			name: "a verify download-artifact path of dist2/ is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				"      - uses: actions/download-artifact@v4\n        with:\n          name: dist\n          path: dist/\n",
				"      - uses: actions/download-artifact@v4\n        with:\n          name: dist\n          path: dist2/\n", 1),
			wantPass:  false,
			wantNamed: []string{"verify job must download"},
		},
		{
			// 036 — a fully missing tap job (no Homebrew publisher at all) is rejected.
			name:      "a fully missing tap job is rejected and named",
			yaml:      strings.Replace(validWorkflowYAML, tapJobBlock, "", 1),
			wantPass:  false,
			wantNamed: []string{"missing job: tap"},
		},
		{
			// 036 — the tap job must run after publish (so it references attached, verified assets).
			name: "a tap job not depending on publish is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				"  tap:\n    needs: [publish]\n", "  tap:\n", 1),
			wantPass:  false,
			wantNamed: []string{"tap job must `needs: [publish]`"},
		},
		{
			// 036 — without the pre-release gate, a pre-release would move the tap.
			name: "a tap job missing the non-prerelease gate is rejected and named",
			yaml: strings.Replace(validWorkflowYAML,
				"    if: ${{ !github.event.release.prerelease }}\n", "", 1),
			wantPass:  false,
			wantNamed: []string{"non-prerelease flag"},
		},
		{
			// 036 — a gate keyed off something other than the pre-release flag is rejected.
			name: "a tap gate that is not the pre-release flag is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				"    if: ${{ !github.event.release.prerelease }}\n", "    if: ${{ github.event.release.draft }}\n", 1),
			wantPass:  false,
			wantNamed: []string{"non-prerelease flag"},
		},
		{
			// 036 — a gate that MENTIONS the flag but augments it (|| true → always
			// runs) must be rejected: the check is exact, not substring.
			name: "a tap gate augmented with || true is rejected (exact-match, not substring)",
			yaml: strings.Replace(validWorkflowYAML,
				"    if: ${{ !github.event.release.prerelease }}\n", "    if: ${{ !github.event.release.prerelease || true }}\n", 1),
			wantPass:  false,
			wantNamed: []string{"non-prerelease flag"},
		},
		{
			// 036 — the cross-repo token must be injected from a secret.
			name: "a tap job without the HOMEBREW_TAP_TOKEN secret env is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				"    env:\n      HOMEBREW_TAP_TOKEN: ${{ secrets.CI_GITHUB_TOKEN }}\n", "", 1),
			wantPass:  false,
			wantNamed: []string{"HOMEBREW_TAP_TOKEN"},
		},
		{
			// 036 — args that merely MENTION "release" (not the subcommand) must be
			// rejected: the check is on the first token, not a substring.
			name: "a tap job whose args only mention release (build --release-notes) is rejected",
			yaml: strings.Replace(validWorkflowYAML,
				"          args: release --clean\n", "          args: build --release-notes\n", 1),
			wantPass:  false,
			wantNamed: []string{"goreleaser release"},
		},
		{
			// 036 — --skip=publish in the tap job would skip the formula push (the job's purpose).
			name: "a tap job that passes --skip=publish is rejected and named",
			yaml: strings.Replace(validWorkflowYAML,
				"          args: release --clean\n", "          args: release --clean --skip=publish\n", 1),
			wantPass:  false,
			wantNamed: []string{"--skip=publish"},
		},
		{
			// 036 — the tap job must not touch the GitHub release (that stays with publish).
			name: "a tap job that runs gh release is rejected and named",
			yaml: strings.Replace(validWorkflowYAML,
				"          args: release --clean\n",
				"          args: release --clean\n      - name: tamper\n        run: gh release edit foo\n", 1),
			wantPass:  false,
			wantNamed: []string{"must not touch the GitHub release"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wf, err := ParseWorkflow([]byte(tc.yaml))
			if err != nil {
				t.Fatalf("parsing fixture workflow: %v", err)
			}
			violations := CheckReleaseWorkflow(wf)
			if tc.wantPass {
				if len(violations) != 0 {
					t.Fatalf("expected the workflow to pass, got violations:\n  %s",
						strings.Join(violations, "\n  "))
				}
				return
			}
			if len(violations) == 0 {
				t.Fatalf("expected the workflow to fail the guard, but it passed")
			}
			joined := strings.Join(violations, "\n")
			for _, want := range tc.wantNamed {
				if !strings.Contains(joined, want) {
					t.Fatalf("guard violation must name %q, got:\n%s", want, joined)
				}
			}
		})
	}
}
