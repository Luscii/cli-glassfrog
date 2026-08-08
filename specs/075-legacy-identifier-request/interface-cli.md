# Interface Accord: Legacy Identifier Request — CLI

**Feature**: 075-legacy-identifier-request
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (command-local `--legacy-id` on the four supported leaves; cobra unknown-flag as the refusal), ADR-2 (structured fidelity by construction — param is the only request-side change), ADR-3 (`LegacyID *int64` models + requested-bit render threading), Render Design section.

---

This accord pins the operator-facing surface of the legacy-identifier opt-in: the flag, its help text, the outbound request contract, the structured and human output shapes per read, and the refusal on unsupported reads. The serialization/format machinery it rides on is pinned by 018/019/020/035; exit codes by 004 (+ 015/054 extensions). Nothing here consumes the number — that is the drafting path's future concern.

---

## Surface

### The flag

`--legacy-id` — boolean, default `false`, **command-local** (`Flags()`, never `PersistentFlags()`), registered on exactly four command leaves covering the six supported reads:

| Command | Branches covered | API operation(s) |
|---|---|---|
| `roles [id]` | list + single | `listRoles`, `getRole` |
| `tree [id]` | whole-org + subtree | `getRoleTree` |
| `actors [id]` | directory + single | `listActors`, `getActor` |
| `me` | identity read | `getMe` |

Both branches of `roles` and `actors` are contract-supported, so `validateRolesFlags` (and the actors sibling) gain **no** cross-branch entry for this flag.

**Help text** — one shared constant, used verbatim by all four registrations (the guard checks for it; see interface-spec.md):

> `Also request each resource's transition-only numeric legacy_id (a temporary v3→v5 migration bridge that retires with the v3 API — not a stable identifier; agent-backed actors have none)`

The single constant is the accord: the retirement clock lives in the option's own help and nowhere else — no stderr note, no output label (spec clarification).

### The request

| Flag state | Query contract |
|---|---|
| set | `include_legacy_id=true` appended to the read's `url.Values`; on walked lists (`roles`, `actors`) it rides `paging.All`'s per-page query clone, so every page carries it |
| unset | no `include_legacy_id` parameter at all — the outbound request is byte-identical to today's |

### The field

`legacy_id` — integer or null, appearing on resources exactly as the API returns it. Typed models gaining `LegacyID` (nullable integer): `Role`, `Actor`, `Organization`, `Membership`, `TreeNode`. Decode-only; no CLI input anywhere accepts a legacy id (spec non-behavior).

**Decode tolerance**: the field accepts a JSON **integer** or a JSON **string bearing an integer** (`"14062695"`) and yields the same value either way (plan ADR-3). Every observed value is an integer (LEARNINGS W5); the tolerance guards against the looseness recorded in the sibling `databaseId` spelling, so an operator never loses a whole read over an optional extra. Scoped to this field alone. A **non-numeric** string, or any other JSON type, still fails the decode — the tolerance widens the accepted *spelling*, not the accepted *values*, so a genuine contract break stays loud instead of yielding a silent absence.

**`TreeNode` note**: this model carries the field on observed evidence, not contract evidence — the vendored `TreeNode` schema omits `legacy_id` while the live tree read returns it on every node (LEARNINGS W1). Do not remove the field on the grounds that the schema lacks it.

---

## Interactions

### Structured output (`-o json` / `-o yaml`)

Raw-byte pass-through (018) — the document mirrors the response shape exactly:

- Requested + carried: `"legacy_id": 14062695`
- Requested + null (agent-backed actor): `"legacy_id": null` — no reason field is added
- Requested + resource is an embed (a role's `fillers`/`subroles`, an actor's `roles`, `me`'s `?include=roles`): **no key**, exactly as the response
- Not requested: **no key anywhere**

`me -o json --legacy-id` therefore carries all three numbers the response carries — `data.actor.legacy_id`, `data.organization.legacy_id`, `data.membership.legacy_id` — with no filtering.

**Example** (`roles role_4d8f01d9… --legacy-id -o json`, abbreviated):

```json
{ "data": { "id": "role_4d8f01d9…", "type": "role", "legacy_id": 14062695, "name": "Cloud Visionary", "fillers": [ { "id": "per_0123…", "name": "Alice Smith" } ] } }
```

(`fillers[]` members carry no `legacy_id` key — embeds are excluded by the contract.)

### Human output — shared idioms

- **Compact** formats append a `legacy_id=<n>` segment when requested; absence renders the settled compact dash: `legacy_id=—`.
- **Full** formats add a `Legacy id:` line (or inline parenthetical where the template is line-per-resource); absence renders `(none)`, and an agent-backed actor renders `(none — agent-backed actor)` — the `(none — anchor role)` idiom from `role.full`.
- **Embed note**: when the flag is set, each *plural embedded group whose members render stable identifiers **and whose read does not carry the number for them*** gets the suffix `(this read carries no legacy id for embedded roles)` on its group heading — once per group, never per member. Groups that render no identifiers at all (Domains, Accountabilities, Policies, Notes, Skills — description/title-only) get no note: nothing in them implies a number could sit there. The single-line `Parent role:` embed is likewise exempt (no group heading to carry the note). This is an interface-level refinement of the spec's "once, alongside that embedded group"; the spec's own edge scenario exercises only id-rendering groups (sub-roles, fillers).
  **Which reads' embeds carry the number is a per-read observed fact, not a rule** (LEARNINGS W2/W3). Verified: `getRole`'s five embed families (`subroles`, `assignments`, `fillers`, `accountabilities`, `domains`) carry **no** number → they get the note. `getMe`'s `?include=roles` **does** carry it on every embedded role → it gets **no** note and renders the number instead. Both use the same `Role` schema; the difference is per endpoint. The note's wording says what *this read* carries, never what exists in the system of record, so it stays true even where behavior was not separately probed.
- **Not requested**: every human render is byte-identical to today's.

### Human output — per template

| Template | Requested rendering |
|---|---|
| `roles.compact` | `legacy_id=<n|—>` segment after the name segment |
| `roles.full` | `Legacy id: <n|(none)>` line after the `Name (role_…)` line |
| `role.compact` | `legacy_id=<n|—>` segment after the name segment |
| `role.full` | `Legacy id:` line after the name line; `Fillers`/`Assignments`/`Subroles` headings get the embed-note suffix |
| `tree.compact` | `legacy_id=<n|—>` segment per row |
| `tree.full` | `Legacy id:` line per row (beside `Purpose:`); `Members` heading gets the embed-note suffix |
| `actors.compact` | `legacy_id=<n|—>` segment after the name segment (`[agent]` in the same line names the backing for a `—`) |
| `actors.full` | `Legacy id:` line per actor; agent-backed → `(none — agent-backed actor)` |
| `actor.compact` | `legacy_id=<n|—>` segment (kind `[agent]` adjacent) |
| `actor.full` | `Legacy id:` line after `Name:`; `Roles`/`Assignments` headings get the embed-note suffix |
| `me.compact` | `legacy_id=<n|—>` after the actor cluster and `org_legacy_id=<n|—>` after the `org=` segment |
| `me.full` | `(legacy id <n>)` appended to the `actor:` and `organization:` lines; absence `(no legacy id)`, agent-backed actor `(no legacy id — agent-backed actor)`. The `roles:` group (when `?include=roles`) renders `(legacy id <n>)` on **each embedded role** and carries **no** embed note — verified: this read's embedded roles do carry the number (LEARNINGS W2). The membership number is **not rendered** (spec clarification — structured carries it) |

Built-in templates render through view-level dereferenced display values (the `TreeRow` precedent), so no pointer artifacts reach output.

### User templates (`-o <file>` / stdin, 035)

A user template addresses the decoded typed structs and sees `LegacyID` wherever the model carries it — including `Membership` — as a nullable integer pointer. The built-in curation (membership omitted, embed notes) is built-in-template behavior only; the CLI imposes none of it on operator templates.

---

## Error Communication

No new exit codes; no existing code changes meaning.

| Condition | Behavior | Exit |
|---|---|---|
| `--legacy-id` on any unsupported command (`me roles`, `me actions`, `me projects`, `subroles`, `fillers`, `search`, `domains`, `policies`, `assignments`, `tension *`, `proposal *`, …) | cobra rejects pre-request: `unknown flag: --legacy-id` (usage error path; names the option; no request sent) | 2 |
| Read fails (404 / 401 / 403 / 429 / plan gate) with the flag set | Unchanged diagnostic and code — the flag alters no failure path | as today |
| Requested but no resource carried a number (incl. post-retirement) | Success; absence idioms above; **no diagnostic, no retirement claim** | 0 |
| Response `legacy_id` is a JSON string (e.g. `"14062695"`) | accepted and rendered identically to an integer — the decode tolerance absorbs it | 0 |
| Response `legacy_id` is neither integer nor string (real contract break) | typed-decode failure on the human path → existing decode-error classification | 3 |

---

## Consistency Notes

- **Refusal by non-registration** (plan ADR-1): the supported set is expressed by where the flag exists; there is no allowlist validator to keep in sync. The mechanical "nowhere else" check lives in the guard (interface-spec.md).
- **Faithful echo / curated human split**: mirrors 041's structured-vs-human division and 018's raw-bytes fidelity; the spec's System Overview names the principle.
- **Compact `—` and full `(none — reason)`**: reuses the settled absence idioms (`actors.compact` name dash; `role.full` `(none — anchor role)`), not new markers.
- **`me roles` exclusion**: the `org-roles.*` templates (012) are untouched — that read is contract-excluded, and its command rejects the flag by non-registration.
- **No `internal/resolve` registration**: deliberate (spec non-behavior — no env/rcfile persistence); diverges from the base-url/output pattern of flag+env+rcfile on purpose.
- **Sibling file**: interface-spec.md pins the drift guard's contract-fact artifact and its invariants, including the help-text constant check referenced above.
