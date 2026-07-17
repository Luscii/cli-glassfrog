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
# bash builtins. Handles the characters that would otherwise break the JSON or
# truncate the value ("\ and the control chars a command line can carry).
json_escape() {
  local s=$1 out='' i c
  local n=${#s}
  for (( i = 0; i < n; i++ )); do
    c=${s:i:1}
    case $c in
      '"') out+='\"' ;;
      '\') out+='\\' ;;
      $'\n') out+='\n' ;;
      $'\t') out+='\t' ;;
      $'\r') out+='\r' ;;
      *) out+=$c ;;
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

# effect_for LEAF TARGET — the human-facing effect clause for a gated write. States
# WHAT changes (command, target, effect); never whether the change is sound.
effect_for() {
  local leaf=$1 target=$2
  case $leaf in
    create) printf 'create a draft proposal from tension %s' "$target" ;;
    propose) printf 'advance proposal %s into circulation' "$target" ;;
    respond) printf 'record a response on proposal %s' "$target" ;;
    withdraw) printf 'withdraw proposal %s' "$target" ;;
    *) printf 'run an unrecognized proposal write (%s), gated by default until the guardrail registry lists it' "$leaf" ;;
  esac
}

# classify_segment SEGMENT — inspect one command segment. Echoes the gated leaf and
# target ("leaf<TAB>target") and returns 0 when the segment is a proposal write to
# gate (a registered write leaf, OR — fail-closed — an unrecognized proposal
# subcommand). Returns 1 for anything that passes ungated (non-glassfrog, a read,
# a tension edit, a recognized proposal read).
classify_segment() {
  local segment=$1
  local -a toks
  read -ra toks <<<"$segment"
  local n=${#toks[@]} i=0

  # Skip a leading VAR=val environment prefix.
  while [ $i -lt "$n" ]; do
    case ${toks[i]} in
      [A-Za-z_]*=*) (( i++ )) ;;
      *) break ;;
    esac
  done
  [ $i -lt "$n" ] || return 1

  # The invocation token must resolve to `glassfrog` (bare, or the basename of an
  # absolute/relative path).
  local inv=${toks[i]}
  [ "${inv##*/}" = "glassfrog" ] || return 1

  # Find the `proposal` group token among the remaining tokens. Absent → this is a
  # read, a tension edit, or some other glassfrog command → ungated.
  local p=-1 j
  for (( j = i + 1; j < n; j++ )); do
    if [ "${toks[j]}" = "proposal" ]; then
      p=$j
      break
    fi
  done
  [ $p -ge 0 ] || return 1

  # The leaf is the first non-flag token after `proposal`; the target the next
  # non-flag token after the leaf.
  local leaf='' target='' k
  for (( k = p + 1; k < n; k++ )); do
    case ${toks[k]} in -*) continue ;; esac
    leaf=${toks[k]}
    break
  done
  for (( k = k + 1; k < n; k++ )); do
    case ${toks[k]} in -*) continue ;; esac
    target=${toks[k]}
    break
  done
  [ -n "$target" ] || target="the target"

  # A recognized proposal read passes ungated.
  if [ -n "$leaf" ] && is_proposal_read "$leaf"; then
    return 1
  fi
  # A registered write leaf, or a missing/unrecognized proposal subcommand
  # (fail-closed), is gated.
  printf '%s\t%s' "$leaf" "$target"
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
    leaf=${hit%%$'\t'*}
    target=${hit#*$'\t'}
    ask "Governance write: \`$command\` will $(effect_for "$leaf" "$target"). Confirm to proceed; the write is sent only if you approve."
  fi
done <<<"$segments"

# No segment was a proposal write.
allow
