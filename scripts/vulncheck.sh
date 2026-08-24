#!/usr/bin/env bash
#
# Runs govulncheck and fails on any reachable vulnerability.
#
# Requires: go.

set -euo pipefail

go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...
