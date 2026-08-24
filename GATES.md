# Gates: dependency and CI update

OWNS: go.mod, go.sum, .github/workflows/ci.yml, pkg/container/client.go, pkg/container/client_test.go

Scope: apply every policy-allowed pending dependency update, repair Go 1.27 CI compatibility, and verify the complete build surface

- [ ] G0: this ledger states executable outcomes that can fail
  CHECK: node /home/patbaumgartner/.agents/skills/unlazy/scripts/gate-lint.mjs GATES.md
  EXPECT: LINT OK
  EVIDENCE: pending

- [ ] G1: Dependabot's pending direct Go updates are applied and the module graph is tidy
  CHECK: test "$(go list -m -f '{{.Version}}' github.com/docker/cli)" = "v29.7.2+incompatible" && test "$(go list -m -f '{{.Version}}' github.com/stretchr/testify)" = "v1.12.1" && go mod tidy -diff && echo "Go dependency versions and module graph verified"
  EXPECT: Go dependency versions and module graph verified
  EVIDENCE: pending

- [ ] G2: no policy-allowed direct Go module update remains
  CHECK: updates=$(go list -m -u -f '{{if and (not .Indirect) .Update}}{{.Path}} {{.Version}} -> {{.Update.Version}}{{end}}' all | sed '/^$/d'); assert_empty() { test -z "$1"; }; if assert_empty "known-positive-update"; then exit 1; fi; assert_empty "$updates" && echo "No direct Go dependency updates remain"
  EXPECT: No direct Go dependency updates remain
  EVIDENCE: pending

- [ ] G3: all local CI lint and test gates pass
  CHECK: test -z "$(gofmt -l .)" && go vet ./... && go install honnef.co/go/tools/cmd/staticcheck@v0.8.1 && staticcheck ./... && shellcheck scripts/*.sh build.sh && go test -race ./... && echo "CI lint and tests passed"
  EXPECT: CI lint and tests passed
  EVIDENCE: pending

- [ ] G4: the resolved dependency graph has no known reachable vulnerabilities
  CHECK: scripts/vulncheck.sh && echo "Vulnerability gate passed"
  EXPECT: Vulnerability gate passed
  EVIDENCE: pending

- [ ] G5: documentation dependencies are current and the strict documentation build passes
  CHECK: test "$(python3 -m pip index versions mkdocs 2>/dev/null | sed -n '1s/^mkdocs (\([^)]*\))$/\1/p')" = "1.6.1" && test "$(python3 -m pip index versions mkdocs-material 2>/dev/null | sed -n '1s/^mkdocs-material (\([^)]*\))$/\1/p')" = "9.7.7" && rm -rf /tmp/watchtower-docs-gate && python3 -m venv /tmp/watchtower-docs-gate && /tmp/watchtower-docs-gate/bin/pip install -r docs-requirements.txt && scripts/build-tplprev.sh && /tmp/watchtower-docs-gate/bin/mkdocs build --strict && echo "Documentation dependencies and build verified"
  EXPECT: Documentation dependencies and build verified
  EVIDENCE: pending

- [ ] G6: the release container builds for every published platform
  CHECK: docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7,linux/386 --file dockerfiles/Dockerfile --tag watchtower:dependency-check . && echo "Multi-platform container build passed"
  EXPECT: Multi-platform container build passed
  EVIDENCE: pending
