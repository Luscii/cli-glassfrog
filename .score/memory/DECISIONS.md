# Decisions

Architectural precedent from the specification pipeline. Each entry records a decision that future specs should follow or explicitly diverge from. Managed by the Score plan skill.

---

- Build the CLI in Go as a self-contained executable (from 001-command-registration, 2026-06-03)
  Chosen over Rust and compiled TS to satisfy CONSTITUTION XII natively (single self-contained binary, no runtime). Foundational language decision inherited by every later spec; built with GOOS/GOARCH cross-compilation and CGO_ENABLED=0 to avoid cgo where supported. ("Fully static" linking is platform-specific and is not the criterion — self-containment is.)

- Adopt cobra as the command framework; its command tree is the command registry (from 001-command-registration, 2026-06-03)
  Sibling capabilities (Argument Dispatch, Help & Version) become thin layers over cobra rather than separate subsystems. Later command specs register into the cobra tree.

- Enforce fail-loud registration through a guard wrapper, not cobra defaults (from 001-command-registration, 2026-06-03)
  cobra tolerates duplicate names and missing summaries; a Register/MustRegister guard validates (non-empty name+summary, leaf-has-action, group-has-children, no sibling collision) before AddCommand. All command registration must go through the guard — direct AddCommand is the bypass to watch.

- Wire commands explicitly in main, not via package init() side effects (from 001-command-registration, 2026-06-03)
  Deterministic ordering and a single legible source of truth for the command set (aligns with CONSTITUTION I). Adding a command = one wiring line + its own package; no existing command is edited.

- Route via cobra's built-in resolution; keep matching exact (from 002-argument-dispatch, 2026-06-03)
  Dispatch relies on cobra's Execute over the assembled tree rather than a hand-rolled matcher: exact match, unknown-command error + best-effort suggestion, and unknown-flag rejection are cobra defaults. cobra's EnablePrefixMatching (a package-global) MUST stay false — prefix/abbreviation matching is a non-behavior; pin it with a regression test.

- Dispatch classifies each outcome into a code-free category for Exit-Code Convention (from 002-argument-dispatch, 2026-06-03)
  Run returns Success / UsageError / RuntimeError; dispatch is the only layer that knows whether a failure was resolution/usage vs the command's own RunE error. It must NOT emit exit codes (that's Exit-Code Convention / 004). Until 004 lands, the entrypoint maps the category minimally (0 / non-zero) as a documented placeholder.
