#!/usr/bin/env bash
#
# glassfrog-write-gate.sh — the Write-Safety Guardrail's PreToolUse gate (063).
#
# The Claude Code plugin host runs this before every `Bash` tool call. It reads
# the tool-call JSON on stdin, and when the shell command is a governance write on
# the proposal write path it returns permissionDecision:"ask" — routing the call
# to the host's human-confirmation prompt so the agent cannot self-authorize the
# write (plan ADR-1/ADR-2). Reads and operational `tension` edits pass through
# ungated.
#
# Design commitments (plan/interface):
#   - Deterministic. The decision keys ONLY on the parsed command path against the
#     single-sourced registry (gated-commands.txt) — never on the agent's stated
#     intent or the command's flags. This is a `type:"command"` hook, not a
#     `type:"prompt"` one: enforcement must not depend on the agent judgment it
#     backstops.
#   - Fail-closed within the `proposal` namespace. An UNRECOGNIZED `glassfrog
#     proposal` subcommand is asked, not waved through, so a future write leaf is
#     gated by default until the registry lists it.
#   - Fail-safe everywhere else. It never blocks unrelated shell work: a non-
#     glassfrog command, or stdin it cannot parse into a positively-identified
#     proposal write, passes through.
#   - Pure bash. No jq/sed/grep/tr dependency — the runtime is pinned to `bash`
#     (portability; the interpreter hooks.json invokes). JSON is parsed with bash
#     builtins only.
#   - No stale-write recovery. The hook adds NO special path for a 412/exit-7
#     stale write: a retry is itself a proposal-path write, so it is simply gated
#     again (plan ADR-5). The re-read guidance stays in Operator Orientation (062).
#
# Accepted residual (plan R1, stated not hidden): a write reaching the shell in an
# exotic form the parser does not resolve — inside a command substitution
# `$(glassfrog proposal …)`, an alias, or a wrapper script — can evade the gate.
# Over-gating (asking on a read) is mere friction and safe; under-gating a write
# is the integrity hole the fail-closed bias guards against.

set -u

# --- Output helpers ---------------------------------------------------------

# allow: pass the command through untouched. Emit nothing and exit 0 so the host
# applies its normal permission flow — the gate only ever ESCALATES to ask, it
# never weakens the host's own settings by force-allowing.
allow() { exit 0; }

# json_escape: escape a string for embedding in a JSON string literal, using only
# bash builtins. Escapes `"` and `\`, the shorthand control escapes, and — so the
# emitted JSON is always valid — EVERY remaining character below 0x20 as \u00XX.
# A raw control character in the command (a lenient host can pass one through)
# would otherwise produce invalid JSON that the host may ignore, silently dropping
# the gate decision. (NUL cannot occur in a bash string, so 0x00 is moot.)
json_escape() {
  local s=$1 out='' i c code
  local n=${#s}
  for (( i = 0; i < n; i++ )); do
    c=${s:i:1}
    case $c in
      '"') out+='\"' ;;
      '\') out+='\\' ;;
      $'\b') out+='\b' ;;
      $'\f') out+='\f' ;;
      $'\n') out+='\n' ;;
      $'\r') out+='\r' ;;
      $'\t') out+='\t' ;;
      *)
        printf -v code '%d' "'$c"
        if [ "$code" -ge 0 ] && [ "$code" -lt 32 ]; then
          printf -v c '\\u%04x' "$code"
        fi
        out+=$c
        ;;
    esac
  done
  printf '%s' "$out"
}

# ask: emit the gate decision naming the write, and exit 0. The reason is carried
# both as hookSpecificOutput.permissionDecisionReason and top-level systemMessage
# so it reaches the practitioner whichever field the host version renders (the
# interface flags this spelling as host-version-specific).
ask() {
  local msg escaped
  msg=$1
  escaped=$(json_escape "$msg")
  printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"ask","permissionDecisionReason":"%s"},"systemMessage":"%s"}\n' "$escaped" "$escaped"
  exit 0
}

# --- JSON extraction (bash builtins only) -----------------------------------

# extract_scalar_string KEY JSON — return the string value of a simple top-level
# scalar key (e.g. tool_name). Assumes the value has no escaped quotes, which is
# true for the values we read this way. Prints nothing and returns 1 if absent.
extract_scalar_string() {
  local key=$1 json=$2 after
  case $json in
    *"\"$key\""*) : ;;
    *) return 1 ;;
  esac
  after=${json#*\"$key\"}
  after=${after#*:}
  # ltrim whitespace
  after=${after#"${after%%[![:space:]]*}"}
  [ "${after:0:1}" = '"' ] || return 1
  after=${after:1}
  printf '%s' "${after%%\"*}"
}

# extract_command JSON — return the value of tool_input.command, unescaping JSON
# string escapes so an embedded \" does not truncate it early. Returns 1 (and
# prints nothing) when the key is absent or the string is unterminated (malformed).
extract_command() {
  local json=$1 after out='' i c nx n
  case $json in
    *'"command"'*) : ;;
    *) return 1 ;;
  esac
  after=${json#*\"command\"}
  after=${after#"${after%%[![:space:]]*}"}
  [ "${after:0:1}" = ':' ] || return 1
  after=${after:1}
  after=${after#"${after%%[![:space:]]*}"}
  [ "${after:0:1}" = '"' ] || return 1
  after=${after:1}
  n=${#after}
  for (( i = 0; i < n; i++ )); do
    c=${after:i:1}
    if [ "$c" = '\' ]; then
      nx=${after:i+1:1}
      case $nx in
        '"') out+='"' ;;
        '\') out+='\' ;;
        '/') out+='/' ;;
        n) out+=$'\n' ;;
        t) out+=$'\t' ;;
        r) out+=$'\r' ;;
        *) out+=$nx ;;
      esac
      (( i++ ))
      continue
    fi
    if [ "$c" = '"' ]; then
      printf '%s' "$out"
      return 0
    fi
    out+=$c
  done
  return 1 # unterminated string — malformed
}

# --- Registry ---------------------------------------------------------------

# Load the gated proposal-write leaves from the single-sourced registry next to
# this script. GATED is a space-delimited set of leaf names (e.g. " create
# propose "). If the registry is unreadable, GATED stays empty and every
# `glassfrog proposal` write falls to the fail-closed `ask` branch — conservative
# by construction, never silently allowing.
load_registry() {
  local dir line grp leaf
  dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" 2>/dev/null && pwd) || return 0
  local registry="$dir/gated-commands.txt"
  [ -r "$registry" ] || return 0
  while IFS= read -r line || [ -n "$line" ]; do
    line=${line%$'\r'}
    line=${line#"${line%%[![:space:]]*}"}
    line=${line%"${line##*[![:space:]]}"}
    [ -z "$line" ] && continue
    case $line in \#*) continue ;; esac
    read -r grp leaf _ <<<"$line"
    if [ "$grp" = "proposal" ] && [ -n "$leaf" ]; then
      GATED+="$leaf "
    fi
  done <"$registry"
}

is_gated() { case " $GATED " in *" $1 "*) return 0 ;; *) return 1 ;; esac; }

# Recognized proposal READS pass ungated. These are script-internal knowledge, not
# part of the gated registry (which lists writes only). Over-gating a not-yet-known
# read is safe friction; the drift tripwire guards the overall proposal surface, so
# a newly-added read is caught and reclassified rather than silently mis-gated.
PROPOSAL_READS=" list get "
is_proposal_read() { case "$PROPOSAL_READS" in *" $1 "*) return 0 ;; *) return 1 ;; esac; }

# --- Command classification -------------------------------------------------

# effect_for LEAF TARGET — the human-facing effect clause for a registry-listed
# write. States WHAT changes (command, target, effect); never whether the change is
# sound. The default arm covers a registry leaf without a bespoke phrasing (e.g. a
# future write added to the registry) — still naming command/target/effect.
effect_for() {
  local leaf=$1 target=$2
  case $leaf in
    create) printf 'create a draft proposal from tension %s' "$target" ;;
    propose) printf 'advance proposal %s into circulation' "$target" ;;
    respond) printf 'record a response on proposal %s' "$target" ;;
    withdraw) printf 'withdraw proposal %s' "$target" ;;
    *) printf 'perform the %s proposal write on %s' "$leaf" "$target" ;;
  esac
}

# classify_segment SEGMENT — inspect one command segment. On a proposal write to
# gate it echoes "<kind>\t<leaf>\t<target>" and returns 0, where <kind> is:
#   - "gated"      — a registry-listed write leaf (the registry is the source of
#                    truth for what is a governance write); or
#   - "failclosed" — an unrecognized proposal subcommand token, gated by default
#                    (a future write leaf until the registry lists it).
# It returns 1 for anything that passes ungated: non-glassfrog, no proposal group,
# NO subcommand leaf at all (bare `proposal` or a help/usage path like
# `proposal --help` — cobra prints usage and writes nothing), a recognized proposal
# read (get/list), a tension edit, or a plain read.
classify_segment() {
  local segment=$1
  local -a toks
  read -ra toks <<<"$segment"
  local n=${#toks[@]} i=0

  # Resolve the real invocation token, stepping past a leading `VAR=val` env prefix
  # and any transparent command-runner wrappers (`command`, `env`, `nohup`, …). A
  # write must not slip through as `command glassfrog proposal propose` or
  # `env VAR=val glassfrog proposal propose`, where the first token is the wrapper.
  # Only these NAMED wrappers are stepped over (so `echo glassfrog proposal …` is
  # still not treated as running glassfrog); for each we also skip its leading
  # `-flags` and `VAR=val` assignments. Wrappers with positional/option-arg
  # grammars (`timeout <dur>`, `sudo -u NAME`, `xargs`) are NOT fully handled and
  # remain accepted residual (plan R1) — partial handling only ever gates more,
  # never regresses.
  while [ $i -lt "$n" ]; do
    case ${toks[i]} in
      [A-Za-z_]*=*) (( i++ )) ;; # VAR=val env assignment
      *)
        case ${toks[i]##*/} in
          command|env|nohup|setsid|stdbuf|nice|ionice|time)
            (( i++ ))
            # Step past the wrapper's own leading options and env assignments.
            while [ $i -lt "$n" ]; do
              case ${toks[i]} in
                -* | [A-Za-z_]*=*) (( i++ )) ;;
                *) break ;;
              esac
            done
            ;;
          *) break ;;
        esac
        ;;
    esac
  done
  [ $i -lt "$n" ] || return 1

  # The invocation token must resolve to `glassfrog` (bare, or the basename of an
  # absolute/relative path).
  local inv=${toks[i]}
  [ "${inv##*/}" = "glassfrog" ] || return 1

  # Collect the POSITIONAL arguments after `glassfrog`, in order. The subcommand
  # path is positional (`glassfrog <group> <leaf> <id>`), so the group is the first
  # positional, the leaf the second, the target the third. Matching a `proposal`
  # token positionally — not "anywhere" — is what keeps a flag VALUE equal to
  # `proposal` (e.g. `--output proposal`, `--body proposal`) from mis-triggering the
  # gate on an unrelated read or tension edit.
  #
  # Flags are skipped. The two persistent value-flags (`--base-url`, `-o`/`--output`)
  # in space form consume the following token as their value, so that token is NOT a
  # positional — skipping it prevents the reverse error: `glassfrog --output json
  # proposal propose` must still resolve group=`proposal` (not `json`), so a real
  # write is never let through. These are the only inheritable value-flags that can
  # precede the group; command-specific value-flags (`--changes`/`--response`/
  # `--body`) only appear after their subcommand.
  local -a pos=()
  local j tok skipval=0
  for (( j = i + 1; j < n; j++ )); do
    tok=${toks[j]}
    if [ "$skipval" = 1 ]; then
      skipval=0
      continue
    fi
    case $tok in
      --base-url|--output|-o) skipval=1 ;; # space-form value flag: its value is next
      -*) : ;;                             # any other flag (booleans, --flag=value, bundled shorts)
      *) pos+=("$tok") ;;                  # a positional argument
    esac
  done

  local group=${pos[0]:-} leaf=${pos[1]:-} target=${pos[2]:-}

  # Only the proposal group is in scope; a read, a tension edit, or any other group
  # passes ungated.
  [ "$group" = "proposal" ] || return 1

  # No subcommand leaf (bare `proposal`, or only flags such as --help): cobra prints
  # usage and writes nothing, so this is not a governance write — pass ungated. The
  # fail-closed default below applies to an unrecognized subcommand TOKEN, never to
  # the absence of one.
  [ -n "$leaf" ] || return 1

  [ -n "$target" ] || target="the target"

  # A recognized proposal read passes ungated.
  if is_proposal_read "$leaf"; then
    return 1
  fi

  # A registry-listed write leaf is gated — the registry is the source of truth for
  # what counts as a governance write. Any other proposal subcommand is gated
  # fail-closed: a future write leaf until the registry lists it.
  if is_gated "$leaf"; then
    printf 'gated\t%s\t%s' "$leaf" "$target"
  else
    printf 'failclosed\t%s\t%s' "$leaf" "$target"
  fi
  return 0
}

# --- Main -------------------------------------------------------------------

GATED=''
load_registry

input=$(cat)

# Only Bash tool calls reach the gate; anything else passes.
tool_name=$(extract_scalar_string tool_name "$input") || allow
[ "$tool_name" = "Bash" ] || allow

# A Bash call we cannot read a command out of is not something we can positively
# identify as a proposal write — fail safe and let it through.
command=$(extract_command "$input") || allow

# Split the command line into segments on the common shell separators so a chained
# `x && glassfrog proposal propose p` is still seen. Splitting inside quotes only
# risks an extra ask (safe), never a missed write.
segments=$command
segments=${segments//&&/$'\n'}
segments=${segments//||/$'\n'}
segments=${segments//;/$'\n'}
segments=${segments//|/$'\n'}

while IFS= read -r segment; do
  [ -n "$segment" ] || continue
  if hit=$(classify_segment "$segment"); then
    kind=${hit%%$'\t'*}
    rest=${hit#*$'\t'}
    leaf=${rest%%$'\t'*}
    target=${rest#*$'\t'}
    if [ "$kind" = "gated" ]; then
      effect=$(effect_for "$leaf" "$target")
    else
      effect="run an unrecognized proposal subcommand ($leaf), gated by default until the guardrail registry lists it"
    fi
    ask "Governance write: \`$command\` will $effect. Confirm to proceed; the write is sent only if you approve."
  fi
done <<<"$segments"

# No segment was a proposal write.
allow
