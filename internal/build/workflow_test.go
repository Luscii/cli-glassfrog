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
`

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
