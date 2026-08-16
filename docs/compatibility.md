# Compatibility and dependency pins

Watchtower deliberately supports Docker API 1.25 so it can run against older Docker Engine versions supplied with
Synology DSM. Compatibility takes precedence over following the newest Docker client library.

## Docker daemon requirement

The daemon must expose API 1.25 or newer. On a Docker host, check it with:

```bash
docker version --format 'server={{.Server.Version}} api={{.Server.APIVersion}} min={{.Server.MinAPIVersion}} arch={{.Server.Arch}}'
```

On Synology, run the command over an administrator SSH session. Keep `DOCKER_API_VERSION=1.25` in the Container Manager
project unless you have verified that the daemon supports a newer API. The `--api-version` flag and
`DOCKER_API_VERSION` environment variable remain supported; malformed values and versions outside the supported
1.25–1.51 range are rejected before connecting to Docker.

Before stopping containers, Watchtower checks known configuration fields that Docker would reject or silently discard
through the selected API. Unsupported update sets are skipped with an error instead of leaving containers stopped.
Legacy container-wide custom MAC addresses remain supported by API 1.25. Per-network custom MAC addresses require
`DOCKER_API_VERSION=1.44` or newer; older APIs cannot distinguish them from daemon-generated endpoint addresses.

## Intentional pins

| Pin | Location | Why it is pinned | Update policy |
| --- | --- | --- | --- |
| Docker SDK and CLI major v28 | `go.mod` | The replacement `github.com/moby/moby/client` currently requires API 1.40. | Dependabot may update v28 minor/patch releases. Major changes require a Synology compatibility review and live daemon test. |
| Docker API 1.25 | `internal/dockercompat/compat.go` | Preserves `AutoRemove`, `StopTimeout`, mounts, and compatibility with older DSM daemons. | Change only with recreation tests and a documented minimum-version migration. |
| Go toolchain patch | `go.mod`, workflows, `dockerfiles/Dockerfile` | Go security fixes must be applied consistently to builds and scans. | Update all locations together after tests and `govulncheck`. |
| Docker build images and frontend | `dockerfiles/Dockerfile` | Digests make builds reproducible. | Dependabot updates the tag and digest together; retain the digest. |
| Alpine package versions | `dockerfiles/Dockerfile` | The final certificate/timezone filesystem must be reproducible. | Update with the Alpine base digest and verify all target platforms build. |
| GitHub Actions | `.github/workflows/` | Full commit SHAs prevent mutable action tags from changing CI code. | Dependabot updates SHAs; never replace them with floating labels. |
| Demo and test containers | Compose files and test scripts | `tag@sha256` keeps build and monitoring fixtures reproducible while exposing a tag to Dependabot. | Dependabot updates Compose values. Update script digests deliberately. Do not remove digests. |

## Required compatibility checks

Changes to Docker dependencies, API handling, or recreation code must pass:

```bash
go test -race ./internal/dockercompat ./internal/flags ./internal/actions ./pkg/container
go test -race ./...
scripts/vulncheck.sh
```

The release image must still publish manifests for `linux/amd64`, `linux/arm64`, `linux/arm/v7`, and `linux/386`.