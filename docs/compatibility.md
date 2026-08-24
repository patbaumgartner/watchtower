# Compatibility and dependency pins

Watchtower uses the maintained split Moby API and client modules. They avoid the vulnerabilities reported against the
legacy monolithic Docker SDK while retaining compatibility with Docker Engine versions that expose API 1.42 or newer.

## Docker daemon requirement

The daemon must expose API 1.42 or newer. On a Docker host, check it with:

```bash
docker version --format 'server={{.Server.Version}} api={{.Server.APIVersion}} min={{.Server.MinAPIVersion}} arch={{.Server.Arch}}'
```

On Synology, run the command over an administrator SSH session. Docker Engine 24.0.2 exposes API 1.43, so use
`DOCKER_API_VERSION=1.42` or `1.43` in Container Manager. The `--api-version` flag and `DOCKER_API_VERSION` environment
variable remain supported; malformed values and versions outside the supported 1.42–1.55 range are rejected before
connecting to Docker.

Before stopping containers, Watchtower checks known configuration fields that Docker would reject or silently discard
through the selected API. Unsupported update sets are skipped with an error instead of leaving containers stopped.
Legacy container-wide custom MAC addresses cannot be recreated by the split client through API 1.42 or 1.43, so
Watchtower skips those updates before stopping the container. Per-network custom MAC addresses require
`DOCKER_API_VERSION=1.44` or newer. Containers with legacy kernel-memory limits are also skipped because those fields
are absent from the current API model.

## Intentional pins

| Pin | Location | Why it is pinned | Update policy |
| --- | --- | --- | --- |
| Split Moby API and client modules | `go.mod` | The legacy monolithic Docker SDK has unresolved security advisories. | Update the API and client together, then run recreation tests and a live daemon compatibility probe. |
| Docker API 1.42 | `internal/dockercompat/compat.go` | Matches the split client model and remains compatible with Docker Engine 24 on Synology. | Change only with recreation tests and a documented minimum-version migration. |
| Go toolchain patch | `go.mod`, workflows, `dockerfiles/Dockerfile` | Go security fixes must be applied consistently to builds and scans. | Update all locations together after tests and `govulncheck`. |
| Docker build images and frontend | `dockerfiles/Dockerfile` | Digests make builds reproducible. | Dependabot updates the tag and digest together; retain the digest. |
| Alpine package versions | `dockerfiles/Dockerfile` | The final certificate/timezone filesystem must be reproducible. | Update with the Alpine base digest and verify all target platforms build. |
| GitHub Actions | `.github/workflows/` | Full commit SHAs prevent mutable action tags from changing CI code. | Dependabot updates SHAs; never replace them with floating labels. |
| Test containers | Test scripts | `tag@sha256` keeps test fixtures reproducible. | Review tags and digests manually. Do not remove digests. |

## Required compatibility checks

Changes to Docker dependencies, API handling, or recreation code must pass:

```bash
go test -race ./internal/dockercompat ./internal/flags ./internal/actions ./pkg/container
go test -race ./...
scripts/vulncheck.sh
```

The release image must still publish manifests for `linux/amd64`, `linux/arm64`, `linux/arm/v7`, and `linux/386`.