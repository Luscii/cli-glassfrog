#!/usr/bin/env bash
# Make the PR Validation gate (spec 024) merge-blocking on main by requiring the
# stable `ci-success` status check.
#
# Usage:   ./scripts/setup-branch-protection.sh [owner/repo]
#          (owner/repo defaults to the current repo via `gh repo view`)
#
# This is the ONE part of PR Validation that cannot live in a committed file:
# GitHub branch protection / rulesets are repository settings, not workflow YAML
# that GitHub auto-applies. A maintainer with admin rights runs this once. Until
# it is applied, ci.yml still RUNS and REPORTS ci-success (the "report" half is
# intact and visible) — but the merge is not BLOCKED. Running this closes the
# "enforce" half.
#
# It adds `ci-success` ADDITIVELY via a repository RULESET, which is additive by
# construction: it layers a required_status_checks rule on top of whatever
# already exists, works whether or not classic branch protection is configured
# (no precondition, no 404 on a fresh repo), and never owns the rest of the
# protection config.
#
# Why a ruleset and NOT the classic endpoints (these are replace-on-write):
#   - NOT `PUT .../branches/main/protection` — replaces the ENTIRE protection
#     config; any omitted/null field (required_pull_request_reviews,
#     restrictions, ...) gets disabled, relaxing unrelated protections.
#   - NOT `PATCH .../required_status_checks` with only ci-success — REPLACES the
#     contexts list, silently dropping other already-required checks (e.g. code
#     or secret scanning).
#   - The additive `POST .../required_status_checks/contexts` works only when a
#     protection rule with required status checks ALREADY exists (else 404), so
#     it is not safe for first-time setup. The ruleset below has no such
#     precondition.
#
# Only `ci-success` is required — never the drifting matrix-cell contexts
# (test (ubuntu-latest), ...), whose names change with the matrix (ADR-4).
#
# `strict` (require branches up to date before merge) and enforce-admins are
# separate maintainer-policy decisions, intentionally left out here: this script
# does exactly one thing — make `ci-success` a required, merge-blocking check.
set -euo pipefail

REPO="${1:-$(gh repo view --json nameWithOwner -q .nameWithOwner)}"
RULESET_NAME="require-ci-success"
CHECK="ci-success"

echo "Target repository : ${REPO}"
echo "Ruleset           : ${RULESET_NAME}"
echo "Required check    : ${CHECK} (on the default branch / main)"
echo

# Idempotent: if a ruleset of this name already exists, report it and stop
# rather than creating a duplicate. Re-point or delete it by hand if its
# definition needs to change.
existing_id="$(gh api "repos/${REPO}/rulesets" --jq \
  ".[] | select(.name == \"${RULESET_NAME}\") | .id" 2>/dev/null || true)"
if [ -n "${existing_id}" ]; then
  echo "A ruleset named '${RULESET_NAME}' already exists (id ${existing_id}); nothing to do."
  echo "Inspect it with: gh api repos/${REPO}/rulesets/${existing_id}"
  exit 0
fi

# An active branch ruleset targeting the default branch, enforcing a single
# required status check: ci-success. `strict_required_status_checks_policy:
# false` keeps it from also requiring branches be up to date (a separate policy
# choice). This rule is additive — it does not touch any existing classic
# protection or other rulesets.
gh api --method POST "repos/${REPO}/rulesets" \
  --input - <<JSON
{
  "name": "${RULESET_NAME}",
  "target": "branch",
  "enforcement": "active",
  "conditions": {
    "ref_name": { "include": ["~DEFAULT_BRANCH"], "exclude": [] }
  },
  "rules": [
    {
      "type": "required_status_checks",
      "parameters": {
        "strict_required_status_checks_policy": false,
        "required_status_checks": [
          { "context": "${CHECK}" }
        ]
      }
    }
  ]
}
JSON

echo
echo "Done. '${CHECK}' is now a required, merge-blocking check on the default branch of ${REPO}."
echo "A pull request whose ${CHECK} is failing or not yet reported cannot be merged (fail-closed)."
