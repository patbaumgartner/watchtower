# Contributing

## Prerequisites

To contribute code changes to this project you will need:

- [Go](https://golang.org/doc/install) 1.26.6 or newer
- `jq`, ShellCheck, and Staticcheck for the same checks CI runs
- [Docker](https://docs.docker.com/engine/installation/) with Buildx, if you want to build images

Check your current Go version as follows:

```bash
go version
```

## Checking out the code

```bash
git clone git@github.com:<yourfork>/watchtower.git
cd watchtower
```

## Building and testing

watchtower is a go application and is built with go commands. The following commands assume that you are at the root level of your repo.

```bash
go build ./...                         # compiles everything
./build.sh                             # builds the watchtower binary with version metadata
go test -race ./...                    # runs the test suite the way CI does
test -z "$(gofmt -l .)" && go vet ./... # the formatting and vet gates CI enforces
staticcheck ./...                       # static analysis
shellcheck scripts/*.sh build.sh        # shell script linting
scripts/vulncheck.sh                   # the govulncheck gate CI enforces
./watchtower --help                    # runs the application (outside of a container)
```

Build a container image the same way CI does — `dockerfiles/Dockerfile` cross-compiles for whichever platforms
you ask buildx for:

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -f dockerfiles/Dockerfile \
  -t watchtower:dev .
```

Drop `--platform` to build only for your own machine, and add `--load` to import the result into your local
Docker daemon.

## Documentation

The site under `docs/` is built with MkDocs. CI runs `mkdocs build --strict`, so broken links and nav entries fail
the build:

```bash
pip install -r docs-requirements.txt
scripts/build-tplprev.sh               # builds the notification template preview WASM bundle
mkdocs serve
```