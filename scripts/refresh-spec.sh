#!/usr/bin/env bash
# Refresh the vendored copy of the GlassFrog API v5 spec from the canonical source.
#
# Usage:   ./scripts/refresh-spec.sh
# After:   git diff -- spec/glassfrog-api-v5.yaml   # review what changed
#
# The v5 API is in Beta, so the published spec can change. Re-running this and
# committing any diff keeps git history as the spec changelog.
set -euo pipefail

SPEC_URL="https://app.glassfrog.com/api/v5/docs/spec.yaml"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEST="${SCRIPT_DIR}/../spec/glassfrog-api-v5.yaml"

echo "Fetching ${SPEC_URL} ..."
curl -fsSL "${SPEC_URL}" -o "${DEST}"

echo "Wrote $(wc -l < "${DEST}" | tr -d ' ') lines to ${DEST}"
echo "Review changes with: git diff -- spec/glassfrog-api-v5.yaml"
