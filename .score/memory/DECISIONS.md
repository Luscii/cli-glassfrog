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
  Run returns Success / UsageError (two values, matching what the spec names). A distinct RuntimeError for a resolved command's own failure is deferred to Exit-Code Convention (004), the consumer that needs the distinction — until then a resolved command's error is returned uncategorized. Dispatch must NOT emit exit codes (that's 004); the entrypoint maps the category minimally (0 / non-zero) as a documented placeholder.

- Help & Version uses cobra's standard help rendering, not a custom template (from 003-help-and-version, 2026-06-03)
  Listing, per-command usage, flags section, and alphabetical sorting are cobra defaults; future command specs get standard help for free by setting a non-empty Short summary. Alphabetical order depends on the cobra package-global EnableCommandSorting staying true — pin it with a regression test (mirrors 002's EnablePrefixMatching pin). The spec's original "no flags section / no long description" non-behavior was narrowed (clarify 2026-06-03) to forbid only new *required* documentation data.

- Hide cobra's built-in `help` and `completion` commands; keep the `--help` flag (from 003-help-and-version, 2026-06-03)
  cobra injects `help` and `completion` outside the registration guard (LEARNINGS finding). Both *commands* are removed from resolution: replace the default help command with a hidden command under a **non-`help` name** (SetHelpCommand(&cobra.Command{Use: "__help_disabled", Hidden: true})) so the `help` token no longer resolves — `Hidden:true` alone only hides from listings, it does not disable `glassfrog help` — and set CompletionOptions.DisableDefaultCmd=true for `completion`. The `--help` *flag* is retained. Future specs that want a framework built-in surfaced must opt in explicitly.
