# Risk: NPM Wrapper Package

**Feature**: 037-npm-wrapper-package
**Round**: 1
**Date**: 2026-06-12
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: Default 3×3 traffic light

> ⚠ Using default risk acceptability matrix — no project-level matrix found in PROJECT.md.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Risk Level | Controls | Residual Risk |
|---|---|---|---|---|---|---|---|
| H-1 | A tampered/corrupted release archive on the fallback path becomes a runnable binary | spec.md § Non-Behaviors | High | Low | Yellow | RC-1 | Yellow (no-signing accepted) |
| H-2 | 022 name-template drift 404s the fallback download / breaks generator asset lookup | plan.md § Risks (R1) | Medium | Medium | Yellow | RC-2, RC-3 | Green |
| H-3 | npm trusted-publisher not registered for a package name → publish fails or mis-publishes | plan.md § Risks (R2) | Medium | Medium | Yellow | RC-4, RC-5 | Yellow (first-release bootstrap) |
| H-4 | Partial publish window — umbrella before platform packages → unresolvable optional deps | plan.md § Architecture Decisions (ADR-7) | Medium | Low | Green | RC-6 | Green |
| H-5 | `--ignore-scripts` leaves no working command, silently | plan.md § Risks (R3) | Medium | Medium | Yellow | RC-7 | Green |
| H-6 | Unsupported-platform install (Windows/other arch) confuses the operator | spec.md § Driving Scenarios — Error | Low | Medium | Green | RC-8 | Green |
| H-7 | Launcher mangles exit code/args → agents act on wrong signals | spec.md § Non-Behaviors | Medium | Low | Green | RC-9, RC-10 | Green |
| H-8 | Missing `tar` extractor on the fallback host | plan.md § Risks (R4) | Low | Low | Green | RC-11 | Green |
| H-9 | Umbrella/platform version skew → wrapper runs a different binary version than reported | plan.md § Architecture Decisions (ADR-6) | Medium | Low | Green | RC-12, RC-13 | Green |

---

## Hazard Details

### H-1: Unverified fallback download becomes runnable

**Source**: spec.md § Non-Behaviors — "must not place a binary whose checksum does not match the release's checksums file on the fallback path"

**Description**: On the postinstall fallback path the channel downloads an archive from GitHub Releases and runs the extracted binary. A corrupted or tampered archive that reached the user as a runnable `glassfrog` would execute attacker-influenced code in the operator's (often an agent's) environment.

**Severity**: High — executing a tampered binary on the host is the worst outcome a packaging channel can produce; blast radius is the whole machine the agent runs on.

**Probability**: Low — the fallback is the *secondary* path (the bundled platform package is primary and ships verified-at-release bytes), and verification is designed into the fallback (ADR-3). It only triggers when optional deps are absent.

**Risk Level**: Yellow (High × Low)

**Controls**:
- **RC-1**: The fallback verifies the archive's sha256 against its checksums-file entry before placing anything, and places a verified binary or nothing (atomic temp→verify→move). Mirrors the install-script (027) integrity gate.

**Residual Risk**: Yellow — RC-1 ensures *integrity* against corruption/transport tampering. It does not provide *authenticity*: the checksums file is fetched over the same channel as the archive, so an attacker controlling the release host controls both (no signing/notarization — the deliberate 022/027 decision). Accepted with justification: integrity-via-checksums is the project-wide distribution stance; cryptographic signing is an additive concern tracked at the cluster level, not this channel's to introduce alone.

### H-2: 022 name-template drift breaks the fallback

**Source**: plan.md § Risks (R1); interface-spec.md § Consistency Notes

**Description**: The fallback constructs asset URLs from 022's `name_template`; the generator relies on the same shape. A change to 022's archive/checksum naming would 404 fallback downloads.

**Severity**: Medium — breaks the fallback path (and could break a release publish) for the npm channel; bundled-path installs are unaffected, and sibling channels are independent.

**Probability**: Medium — cross-spec coupling to a template owned by another spec; it can change without this channel noticing.

**Risk Level**: Yellow (Medium × Medium)

**Controls**:
- **RC-2**: The exact 022 asset names are encoded as test fixtures, so a template change breaks a `node --test` check before release.
- **RC-3**: The publish path bundles `dist/` binaries directly (not URL-constructed), so only the install-time fallback depends on the URL shape — the publish itself is drift-immune.

**Residual Risk**: Green — RC-2 surfaces drift in CI before a release ships; RC-3 limits the blast radius to the fallback path.

### H-3: npm trusted-publisher misconfiguration / bootstrap gap

**Source**: plan.md § Risks (R2); plan.md § Architecture Decisions (ADR-7)

**Description**: OIDC trusted publishing requires each of the five package names to be registered as a trusted publisher on npmjs.com before the first publish. If unregistered (or misconfigured for access level), the publish job fails — or, mis-set, could publish with unintended visibility.

**Severity**: Medium — a failed publish blocks the npm channel for that release; other acquisition channels are unaffected, and no governance/user data is at risk.

**Probability**: Medium on the first release (bootstrap), Low thereafter (names are fixed).

**Risk Level**: Yellow (Medium × Medium)

**Controls**:
- **RC-4**: The one-time trusted-publisher registration is documented as a release prerequisite (T006), in the spirit of 024's branch-protection bootstrap.
- **RC-5**: The publish job is gated on `build` + `verify`, so it only runs against a verified release — a publish failure aborts cleanly with nothing half-published to the registry beyond per-package atomicity.

**Residual Risk**: Yellow — the first-release bootstrap step is manual and external (npmjs.com); accepted with documented justification. After the first successful publish the residual drops to Green.

### H-4: Partial publish window — umbrella before platform packages

**Source**: plan.md § Architecture Decisions (ADR-7)

**Description**: If the umbrella publishes before its pinned platform packages exist, consumers installing during that window get unresolvable `optionalDependencies`.

**Severity**: Medium — transient install degradation (the binary won't resolve from the bundled path; the postinstall fallback or refusal would engage).

**Probability**: Low — the window is narrow and ordering is controlled.

**Risk Level**: Green (Medium × Low)

**Controls**:
- **RC-6**: The publish job publishes the four platform packages **first**, then the umbrella, so the pinned optional deps always exist when the umbrella appears.

**Residual Risk**: Green.

### H-5: `--ignore-scripts` leaves no working command silently

**Source**: plan.md § Risks (R3)

**Description**: With no matching optional dep AND postinstall scripts disabled, the umbrella installs with no binary placed and (without a backstop) no error — a silent non-working command.

**Severity**: Medium — an agent gets a CLI that doesn't work, against Fail-Safe-not-silent.

**Probability**: Medium — security policies and some CI setups disable install scripts by default.

**Risk Level**: Yellow (Medium × Medium)

**Controls**:
- **RC-7**: The launcher is a runtime backstop — when no binary resolves it emits a clear refusal (detected platform + supported set + reinstall advice) and exits non-zero, converting a silent gap into a loud, recoverable failure (ADR-4).

**Residual Risk**: Green — RC-7 satisfies Fail-Safe: the failure is obvious and the next step is named.

### H-6: Unsupported-platform install confusion

**Source**: spec.md § Driving Scenarios — Error ("unsupported platform is refused at install")

**Description**: Node/npm run on Windows and other arches with no released binary; an install there must not produce a broken or confusing command.

**Severity**: Low — a clear refusal, no data or governance harm.

**Probability**: Medium — Windows Node environments will attempt installs.

**Risk Level**: Green (Low × Medium)

**Controls**:
- **RC-8**: The postinstall refuses on an unsupported target (and the launcher backstops), naming the detected platform and the four supported targets, placing nothing, exiting non-zero (ADR-5).

**Residual Risk**: Green.

### H-7: Passthrough fidelity loss (exit code / args)

**Source**: spec.md § Non-Behaviors — "must not modify, re-parse, or reinterpret the binary's arguments, output, or exit code"

**Description**: If the launcher altered exit codes or arguments, an agent scripting against the 004 exit-code convention would act on wrong signals — a silent-wrong-behavior hazard.

**Severity**: Medium — agents make downstream decisions from exit codes; a wrong code misleads them (Action-Transparency / Fail-Safe adjacent).

**Probability**: Low — passthrough is the launcher's sole job and is directly tested.

**Risk Level**: Green (Medium × Low)

**Controls**:
- **RC-9**: The launcher exits with the child's exact exit code and re-raises terminating signals; argv and stdio are inherited unchanged (ADR-4).
- **RC-10**: A `node --test` passthrough test asserts argument forwarding and exit-code propagation against a stub binary (T004/T001).

**Residual Risk**: Green.

### H-8: Missing `tar` extractor on the fallback host

**Source**: plan.md § Risks (R4)

**Description**: The fallback shells out to system `tar`; a host lacking it cannot extract the archive.

**Severity**: Low — a clear failure, nothing placed, no harm.

**Probability**: Low — `tar` ships on macOS/Linux, the only supported targets.

**Risk Level**: Green (Low × Low)

**Controls**:
- **RC-11**: The fallback probes for `tar` and fails clearly before any placement, naming the missing tool (mirrors 027's tooling probe).

**Residual Risk**: Green.

### H-9: Umbrella/platform version skew

**Source**: plan.md § Architecture Decisions (ADR-6)

**Description**: If version coupling were loose, a wrapper version could resolve a different-version platform binary, so `--version` and behavior would not match the installed package.

**Severity**: Medium — an operator/agent believes they run version X but run Y (Spec-Fidelity / No-Fabricated-Data adjacent).

**Probability**: Low — coupling is enforced by exact-version pinning.

**Risk Level**: Green (Medium × Low)

**Controls**:
- **RC-12**: The umbrella pins each `optionalDependency` to `=X.Y.Z`, so a wrapper version resolves only its matching-version platform package (ADR-6).
- **RC-13**: A `@validation` scenario asserts the placed binary's `--version` equals the installed package version and the release tag.

**Residual Risk**: Green.

---

## Residual Risk Summary

| Level | Count | Hazards |
|---|---|---|
| Red (unacceptable) | 0 | — |
| Yellow (justified) | 2 | H-1, H-3 |
| Green (accepted) | 7 | H-2, H-4, H-5, H-6, H-7, H-8, H-9 |

**Unacceptable risks**: None. All residual risks are Yellow or Green after controls. The two Yellow risks are accepted with documented justification: H-1 (integrity-via-checksums without authenticity is the project-wide no-signing stance, 022/027) and H-3 (first-release trusted-publisher bootstrap is a one-time documented manual step, dropping to Green thereafter).

---

## Traceability Index

### Hazards

| ID | Source |
|---|---|
| H-1 | spec.md § Non-Behaviors |
| H-2 | plan.md § Risks (R1) |
| H-3 | plan.md § Risks (R2) / § Architecture Decisions (ADR-7) |
| H-4 | plan.md § Architecture Decisions (ADR-7) |
| H-5 | plan.md § Risks (R3) |
| H-6 | spec.md § Driving Scenarios — Error |
| H-7 | spec.md § Non-Behaviors |
| H-8 | plan.md § Risks (R4) |
| H-9 | plan.md § Architecture Decisions (ADR-6) |

### Controls

| ID | Mitigates | Grounding |
|---|---|---|
| RC-1 | H-1 | plan.md ADR-3 — 027-conformant sha256 verification + atomic placement |
| RC-2 | H-2 | plan.md ADR-3 / interface-spec.md § Consistency Notes — template pinned in test fixtures |
| RC-3 | H-2 | plan.md ADR-7 — publish bundles verified `dist/` bytes, not URL-constructed |
| RC-4 | H-3 | plan.md ADR-7 / tasks.md T006 — documented one-time trusted-publisher setup |
| RC-5 | H-3 | plan.md ADR-7 — publish gated on `build` + `verify` |
| RC-6 | H-4 | plan.md ADR-7 — platform-packages-first publish ordering |
| RC-7 | H-5 | plan.md ADR-4 — launcher runtime backstop |
| RC-8 | H-6 | plan.md ADR-5 — postinstall + launcher unsupported-platform refusal |
| RC-9 | H-7 | plan.md ADR-4 — exit-code/signal/argv passthrough |
| RC-10 | H-7 | tasks.md T001/T004 — `node --test` passthrough assertions |
| RC-11 | H-8 | plan.md ADR-3 — `tar` probe + clear failure before placement |
| RC-12 | H-9 | plan.md ADR-6 — `=X.Y.Z` exact optionalDependency pinning |
| RC-13 | H-9 | npm-wrapper-package.feature — `@validation` version-match scenario |
