#!/usr/bin/env bash
#
# Runs govulncheck and fails on any reachable vulnerability that is not explicitly
# accepted below. The gate is never weakened: an accepted advisory must have no
# upstream fix and a written justification.
#
# Requires: go, jq.

set -euo pipefail

# Advisories accepted with justification.
#
# GO-2026-4887  Moby AuthZ plugin bypass on oversized request bodies.
# GO-2026-4883  Moby off-by-one in plugin privilege validation.
#
# Both are defects in the Moby *daemon*. Watchtower uses github.com/docker/docker
# only as an API client: it neither serves the Docker API, runs an authorization
# plugin, nor installs plugins. govulncheck matches them because that module path
# is shared between daemon and client, and no fixed version exists for it.
ACCEPTED=(
  GO-2026-4887
  GO-2026-4883
)

report=$(go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 -format json ./...)
accepted=$(printf '%s\n' "${ACCEPTED[@]}" | jq -R . | jq -sc .)

# A finding whose first trace frame names a function is one govulncheck considers
# reachable from this module's code; module-only matches are informational.
called=$(printf '%s' "$report" | jq -sc '
  [ .[] | select(has("finding")) | .finding
    | select((.trace[0].function // null) != null) | .osv ] | unique')

mapfile -t unaccepted < <(printf '%s' "$called" |
  jq -r --argjson accepted "$accepted" 'map(select(IN($accepted[]) | not))[]')

mapfile -t stale < <(printf '%s' "$accepted" |
  jq -r --argjson called "$called" 'map(select(IN($called[]) | not))[]')

if [ ${#stale[@]} -gt 0 ]; then
  echo "::warning::No longer reported; remove from the accept list in $0:"
  printf '  %s\n' "${stale[@]}"
fi

if [ ${#unaccepted[@]} -gt 0 ]; then
  echo "::error::govulncheck reported unaccepted vulnerabilities:"
  printf '%s' "$report" | jq -sr --args '
    .[] | select(has("osv")) | .osv
    | select(.id | IN($ARGS.positional[]))
    | "  \(.id): \(.summary)\n    https://pkg.go.dev/vuln/\(.id)"' "${unaccepted[@]}"
  exit 1
fi

echo "govulncheck: no unaccepted vulnerabilities."
