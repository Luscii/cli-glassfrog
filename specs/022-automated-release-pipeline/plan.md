# Plan: Automated Release Pipeline

**Feature**: 022-automated-release-pipeline
**Role**: Shaper
**Inputs**: spec.md, PROJECT.md, .score/memory/DECISIONS.md (021's GoReleaser/build-only precedent), reference pipelines (an analogous Go-CLI `release.yml` + GoReleaser config, and a Luscii repo-template draft/promote-release flow)

---

## System Architecture

This feature ships **no application code** — it is CI/CD infrastructure composed of two declarative artifacts at the repo root, plus a reuse of 021's self-containment check:

1. **`.goreleaser` release sections** — 021 established a single `.goreleaser` file carrying *only* a `builds` block (four targets: darwin/linux × amd64/arm64, `CGO_ENABLED=0`, no Windows; `builds.ldflags` left as 023's version seam). 022 **extends that same file** with the `archives`, `checksum`, and `release` sections. It does not touch `builds` or `ldflags` (DECISIONS: 021 owns those; 022 must not re-decide the build).
2. **`.github/workflows/release.yml`** — a GitHub Actions workflow triggered by `release: published`, which drives GoReleaser and the cross-target verification gate.
3. **Cross-target self-containment gate** — 021's run-plus-OS-linkage check (the `internal/cli/smoke_test.go` subprocess pattern), executed against the produced `dist/` artifacts on a per-target runner matrix. 021 explicitly deferred cross-target *execution* of this check to "022's CI concern."

**Data / control flow:**

```
maintainer publishes a GitHub Release (drafted by #30 Release Drafting, or by hand)
        │  event: release { types: [published] }, ref = the release tag
        ▼
[build job]  checkout @ tag (fetch-depth 0)  →  goreleaser release --clean --skip=publish
        │      produces dist/: 4 tar.gz archives + checksums.txt (version derived from the tag)
        │      upload dist/ as a CI artifact
        ▼
[verify matrix]  4 native-arch runners (linux amd64/arm64, darwin amd64/arm64)
        │      each downloads dist/, runs 021's self-containment check on its target binary
        │      (execute → assert exit 0 → inspect linkage against the OS-only allowlist)
        │      ANY failure ⇒ stop, nothing published
        ▼
[publish job]  (only if build + every verify pass)  attach dist/ archives + checksums.txt
               to the SAME published release — honoring its notes and pre-release/latest
               status (does not author notes, does not change status)
```

Each architectural element maps to a spec accord: the workflow trigger ⇒ **Trigger**; GoReleaser `archives`+`checksum` ⇒ **Build and package**; the publish job's attach-to-existing-release ⇒ **Attach**; the build/verify gating ⇒ **Atomicity** (extended per ADR-3 to "build-or-verification failure aborts"); the publish job's honor-existing behavior ⇒ the spec's "honors pre-release/latest status … adds artifacts, not prose."

The build host needs the Go toolchain and `goreleaser`; **neither is a dependency of the produced artifact** (CONSTITUTION XII governs the binary's runtime, not the build host) — 021's load-bearing distinction, carried forward.

---

## Architecture Decisions

### ADR-1: Complete the single `.goreleaser` with release sections rather than introduce a parallel mechanism

**Context**: 021 adopted GoReleaser in build-only mode precisely so the release pipeline would *extend* one config instead of reconciling two build paths (DECISIONS, from 021). The spec leaves "release tooling" a plan decision but fixes *what* is published (one archive per target + one checksums file) and *when* (on publish). The `.goreleaser` file is intentionally partial after 021 and "022 owns completing it."

**Options considered**:
1. **Extend the existing `.goreleaser`** — add `archives`/`checksum`/`release` to 021's file; `goreleaser release` spans build→package→publish. One mechanism, no drift; the build that ships is the build 021 defined. Cost: couples 022 to GoReleaser's release model.
2. **Hand-rolled packaging** (`tar`/`sha256sum` in shell + `gh release upload`) — full control, but reintroduces a second build/packaging path GoReleaser already provides and that 021 explicitly avoided.

**Decision**: Option 1. Extend 021's `.goreleaser` with: an `archives` entry producing one `tar.gz` per target named to carry tool/version/OS/arch; a `checksum` entry producing a single `sha256` `checksums.txt` over all archives; and a `release` entry configured to attach to the existing published release (ADR-2). **`builds` and `builds.ldflags` are untouched** — 021 owns the matrix, 023 owns version injection through the ldflags seam. GoReleaser derives the version from the git tag, so no version logic is added here.

**Consequences**: One tool, one config, build→release with no drift; Homebrew Tap (#36) later extends the same file with `brew`. The exact archive/checksum field schema and name templates are interface-level (`/score:interface`). 022 must resist re-deciding `builds` — the config-guard test 021 introduced already pins the matrix and `CGO_ENABLED=0`.

### ADR-2: Trigger on `release: published`; attach to the existing release, honoring its notes and status

**Context**: The spec's trigger is *publishing* a GitHub Release (drafted by #30 or hand-created), not a raw tag push. 022 owns binaries + checksums; #30 owns notes, draft, and pre-release/latest status, which 022 must *honor, not decide*.

**Options considered**:
1. **`on: push: tags` + GoReleaser creates the release** — GoReleaser's canonical mode, but it would author the release and its notes, colliding with #30's ownership and contradicting the spec's publish-driven trigger.
2. **`on: release: published` + GoReleaser attaches to the existing release** — fires when a maintainer publishes; GoReleaser uploads artifacts to the release that already exists for the tag, leaving body and status as published. Matches the spec and the reference pipeline.

**Decision**: Option 2. The workflow declares `on: release: { types: [published] }` with `permissions: contents: write`. GoReleaser's `release` section is configured to upload into the **existing** release for the tag (append/keep-existing semantics) and to **not** author or replace the release body and **not** set `prerelease`/`make_latest` — so the status #30 (or the publisher) chose is preserved. A release published by hand (no #30) is handled identically.

**Consequences**: Clean 022↔#30 seam — #30 owns prose and status, 022 owns artifacts. The precise GoReleaser fields that express "don't touch the body / don't flip status" (e.g. release mode, `disable_changelog`-style options) are interface-level and must be pinned there and tested against a draft. Re-running for an already-published release converges on one artifact set via `--clean` plus replace-on-upload semantics (spec edge).

### ADR-3: Cross-target self-containment is a blocking release gate, via build-once + per-target verify matrix

**Context**: 021 proved self-containment on the host target and deferred cross-target *execution* (multi-arch runners / emulation) to "022's CI concern." The spec scoped 022 to build-package-attach and excluded re-running the behavioral test suite (#24/#29). Resolved with the developer: include a cross-target self-containment gate, because verifying that each shipped binary *runs cleanly on its target* is artifact verification — distinct from the behavioral test suite — and is the natural owner of 021's deferral.

**Options considered**:
1. **Blocking gate, build-once then verify-matrix** — build/package once (no publish), fan out the self-containment check across the four native-arch runners against those exact artifacts, publish only if all pass. The verified bytes are the published bytes.
2. **Host-arch-only verification** — cheapest, but leaves arm64/darwin artifacts unverified — the platforms most likely to carry a cross-compilation surprise.
3. **No gate** — keep 022 minimal; but 021's deferral would then be unowned.

**Decision**: Option 1. The build job runs GoReleaser **without publishing** to produce `dist/`, uploaded as a CI artifact. A verification matrix of four native-arch runners (linux amd64/arm64, darwin amd64/arm64) each downloads `dist/`, selects its target binary, and runs 021's check (execute → assert exit 0 → inspect dynamic-library linkage against the per-platform OS-only allowlist). The publish job runs **only** if the build and every matrix leg pass; any failure aborts with nothing uploaded. This **extends the spec's atomicity** from "any target build failure aborts" to "any build-or-verification failure aborts."

**Consequences**: The published artifacts are exactly the ones verified (no rebuild divergence). CI cost rises (a four-runner matrix per release, including arm64/darwin runners). Where a native runner is unavailable, emulation (QEMU) is the documented fallback (interface concern). 021's self-containment check is reused as-is, pointed at `dist/` — the dist-preferred path 021 built for exactly this.

---

## Integration Design

- **Self-Contained Executable Build (021 — upstream, hard dependency)**: provides the `.goreleaser` `builds` block 022 extends and the self-containment check 022 executes cross-target. 022's implementation is **gated on 021 landing** (021 is currently only `Analyzed`). Boundary: 021 owns `builds`/`ldflags`; 022 adds `archives`/`checksum`/`release`.
- **Version Embedding (023 — upstream seam)**: 022 does not inject the version; GoReleaser reads it from the git tag, and 023 owns the `builds.ldflags` injection that makes `--version` report it. 022 leaves `ldflags` untouched.
- **Release Drafting (#30 — trigger source)**: provides the published release (notes + status) whose publish event triggers 022. Soft dependency — a hand-published release works identically.
- **GitHub Releases (trigger + destination)**: the publish event triggers the workflow; the same release receives the artifacts. `GITHUB_TOKEN` with `contents: write` is the only credential; everything runs within this repository (no external secrets, matching the spec's "entirely within this repository").
- **Install Script (#27) / Homebrew Tap (#36) / NPM Wrapper (#37) — downstream consumers**: consume the published archives and verify against `checksums.txt`. Homebrew later extends the same `.goreleaser` with a `brew` section.

---

## Cross-cutting Concerns

- **Atomicity / failure** — gating is structural: publish is a separate job depending on build + the full verify matrix, so a partial failure simply never reaches publish (no partial release). Re-runs converge via GoReleaser `--clean` and replace-on-upload.
- **Build-host vs artifact dependency** — the build host needs Go + `goreleaser`; the artifact needs only its OS + network. The self-containment gate asserts the latter; the build-host tooling is not a XII regression (021's note, carried forward so reviewers don't conflate them).
- **Secrets & permissions** — only the workflow-provided `GITHUB_TOKEN` (`contents: write`). No signing or notarization (spec non-requirement); integrity is the checksums file.
- **Configuration** — the trigger event, the runner matrix, and the archive/checksum/release shape are the only knobs; all declarative. No application configuration.
- **Testing strategy** — a workflow is hard to unit-test; confidence comes from (a) 021's config-guard test extended to assert the new `archives`/`checksum`/`release` sections exist and the matrix is still exactly four targets, (b) the cross-target self-containment gate itself, and (c) a dry-run/snapshot (`goreleaser release --snapshot --skip=publish`) producing the expected `dist/` layout before wiring publish.

---

## Implementation Strategy

**Dependency gate**: 022 implementation begins after **021 lands** (the `.goreleaser` `builds` block and the self-containment check must exist to extend/reuse). Planning, interface, scenarios, and tasks can proceed now; the build work waits on 021.

**Phase 1 — Release configuration.** Extend `.goreleaser` with `archives` (one `tar.gz` per target, name template carrying tool/version/OS/arch), `checksum` (single `sha256` `checksums.txt`), and `release` (attach to existing published release; do not author notes or change status). Verify `goreleaser release --snapshot --clean --skip=publish` emits the four archives + `checksums.txt` under `dist/`. Extend 021's config-guard test to assert the new sections are present and the build matrix is unchanged.

**Phase 2 — Release workflow.** Add `.github/workflows/release.yml`: `on: release: { types: [published] }`, `permissions: contents: write`; a build job (checkout @ tag with `fetch-depth: 0`, setup Go from `go.mod`, `goreleaser release --clean --skip=publish`, upload `dist/`), and a publish job that uploads `dist/` to the release. Confirm publish honors existing notes/status (test against a draft release).

**Phase 3 — Cross-target verification gate.** Insert the verify matrix (four native-arch runners) between build and publish: each downloads `dist/`, runs 021's self-containment check against its target binary, and must pass before publish runs. Document the emulation fallback where a native runner is unavailable.

Phases are ordered: 1 → 2 → 3, all after 021.

---

## Risks

- **021 not yet implemented** — 022's build/verify work cannot run until 021's `.goreleaser` `builds` block and self-containment check exist. *High likelihood (021 is `Analyzed`), blocking impact.* Mitigation: sequence 021 first (already surfaced in the backlog); 022's plan/interface/scenarios/tasks proceed in parallel, implementation waits.
- **GoReleaser overwriting the release body or status** — a misconfigured `release` section could replace #30's notes or flip pre-release/latest, breaking the spec's honor-not-decide contract. *Medium.* Mitigation: pin append/keep-existing + no `prerelease`/`make_latest` override in interface; test against a draft release before first real publish.
- **arm64 / darwin runner availability or cost** — the verify matrix needs native arm64 and macOS runners. *Medium.* Mitigation: documented QEMU emulation fallback; the matrix is the only multi-arch cost and runs once per release.
- **Re-publish duplication** — re-running for an existing release could double-attach assets. *Low–medium.* Mitigation: `--clean` + replace-on-upload semantics; pin in interface and cover with the spec's re-publish edge scenario.
- **Version/tag mismatch** — GoReleaser derives the version from the tag; a checkout not at the tag yields a wrong version. *Low.* Mitigation: the workflow checks out the release tag ref with `fetch-depth: 0`.

---

## What This Plan Does Not Cover

- **Protocol/structural contracts** — the exact `.goreleaser` field schema (archive format/name template, checksum algorithm/filename, `release` mode fields), the workflow YAML specifics (action versions, runner labels, job names), and the per-platform OS-only linkage allowlist are the interface skill's concern (`/score:interface`).
- **Version embedding** — `builds.ldflags` injection and the `go install` fallback are 023; 022 leaves the seam untouched.
- **The build matrix and self-containment check internals** — owned by 021; 022 extends/reuses, never re-decides.
- **Release notes, draft maintenance, and status decisions** — owned by Release Drafting (#30); 022 honors what the published release carries.
- **Acquisition** — Install Script (#27), Homebrew Tap (#36), NPM Wrapper (#37) consume the published artifacts; not in scope here.
- **Signing / notarization** — explicitly out of scope (spec); integrity is the checksums file.
