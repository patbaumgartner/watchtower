# Changelog

All notable changes to this fork are documented here. This project follows
[Semantic Versioning](https://semver.org/spec/v2.0.0.html) and
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

This is the first release line of `patbaumgartner/watchtower`, a maintained fork of
[`containrrr/watchtower`](https://github.com/containrrr/watchtower) after the upstream project was retired.

### Breaking

- The Go module path is now `github.com/patbaumgartner/watchtower`. Anyone importing this repository as a library
  must update their import paths.
- Container images moved. Replace `containrrr/watchtower` with `patbaumgartner/watchtower` in `docker run`
  commands, compose files, and Kubernetes manifests. Configuration, flags,
  environment variables, and labels are unchanged, so no other migration is needed.
- Per-architecture image tags (`amd64-*`, `arm64v8-*`, `armhf-*`, `i386-*`) are gone. Every tag is now a
  multi-architecture manifest covering `linux/amd64`, `linux/arm64`, `linux/arm/v7`, and `linux/386`; pull the
  plain tag and the runtime picks the right build.
- Building from source now requires Go 1.26 or newer.
- `dockerfiles/Dockerfile.self-contained` and `dockerfiles/Dockerfile.dev-self-contained` were removed.
  `dockerfiles/Dockerfile` cross-compiles for any platform buildx is given.

### Security

- The registry digest client no longer disables TLS certificate verification. It previously sent the registry
  bearer token over an unverified connection, so an on-path attacker could capture credentials and forge digest
  responses.
- Registry token responses are now closed, size-bounded, and their read errors checked; a `WWW-Authenticate`
  realm that fails to parse is rejected instead of causing a nil-pointer panic.
- Registry HTTP clients have request and response-header timeouts, so an unresponsive registry can no longer stall
  an update run indefinitely.
- The HTTP API serves from a dedicated mux with read, header, and idle timeouts instead of
  `http.ListenAndServe` on `DefaultServeMux`, and compares the API token in constant time.

### Fixed

- A failed container inspection after an update no longer panics while running post-update lifecycle hooks.
- Containers can be recreated through Docker API 1.25 when newer daemons report generated endpoint MAC addresses
  or healthcheck start intervals.
- Lifecycle hooks are started once and their execution timeout now covers output capture.
- Registries that return only an OAuth2 `access_token` (rather than `token`) now authenticate.
- `scripts/build-tplprev.sh` finds `wasm_exec.js` under the Go 1.24+ layout.

### Changed

- Upgraded to Go 1.26 and the Docker SDK v28.
- CI runs format, vet, staticcheck, shellcheck, a strict docs build, a govulncheck gate, a race-enabled test
  matrix on Linux/macOS/Windows, CodeQL, and a multi-architecture image build.
- Release automation replaced with Buildx and GoReleaser v2; images carry SBOMs and build provenance attestations.
