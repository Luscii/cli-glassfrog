# Interface Accord: Argument Dispatch — Specification

**Feature**: 002-argument-dispatch
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture (dispatch entry + outcome category); ADR-2 (code-free outcome classification).

---

## Surface

The dispatch entry is what the entrypoint calls and what Exit-Code Convention (004) will consume. It has two parts: the **entry point** and the **outcome category**.

### Entry point

| Entry point | Signature (contract) | Description |
|---|---|---|
| Run | `Run(root, args) -> (Outcome, error)` | Resolves `args` against the assembled `root` command tree, executes the resolved command (or routes a group/root to help), and returns the categorized `Outcome` plus any error. The single dispatch boundary; the entrypoint calls it instead of executing the tree directly. |

### Outcome category

A code-free classification with two values:

| Value | Meaning |
|---|---|
| `Success` | A command was routed and dispatched (it ran, or a group/root resolved to a help/listing outcome). |
| `UsageError` | The invocation did not name a valid command, or carried an unknown flag / unexpected argument — nothing ran (or running was refused). |

A resolved command whose own action returns an error is reported via the returned `error`; giving runtime failures a distinct category (`RuntimeError`) is **deferred to Exit-Code Convention (004)**, the consumer that needs to tell them apart. The category names *what kind* of outcome occurred. It carries **no** exit code and **no** rendered message — those belong to Exit-Code Convention and Help & Version respectively.

---

## Interactions

- **Assembly → dispatch → exit**: the entrypoint builds the tree (`Assemble`, 001), passes it to `Run` with the invocation arguments, and acts on the returned `Outcome`.
- **Category derivation** (how `Run` decides — see plan's Outcome Classification table): unresolved token or unknown flag/arg → `UsageError`; otherwise → `Success` (a resolved command that runs — even if its own action returns an error, which travels via the returned `error`; categorizing that runtime failure is deferred to 004).
- **Consumer**: until Exit-Code Convention (004) exists, the entrypoint maps the category minimally (success → 0, any error → non-zero) as a documented placeholder; 004 replaces that mapping.

---

## Error Communication

- `Run` returns the `Outcome` category alongside any error — it does not swallow errors and does not itself terminate the process.
- Constraint violations are not applicable here (dispatch does not register anything); the failure mode is the `UsageError` category above, accompanied by the underlying error for the caller to surface. A resolved command's own error is also returned (uncategorized for now — see the deferral note above).

---

## Consistency Notes

- **Sibling (`interface-cli.md`, this spec)**: the caller-facing resolution/usage behavior is the observable face of this `Run`/`Outcome` contract.
- **Exit-Code Convention (future, 004)**: the `Outcome` category is the input it maps to process codes — this is the contract that lets 004 avoid re-deriving the category from an untyped error.
- **001 (`interface-spec.md`)**: parallels 001's `Register`/`MustRegister` extension contract — both are internal Go-level surfaces this project treats as accords because a sibling capability builds on them.
