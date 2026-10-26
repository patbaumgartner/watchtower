# syntax=docker/dockerfile:1@sha256:ecfaec9ed6d810b56388c508f4121597bfbba70d41a6dfeee4d8cad5f295fc32
#
# Cross-compiles watchtower for the platform requested by buildx and packages it
# into a scratch image. Build with:
#
#   docker buildx build --platform linux/amd64,linux/arm64 -f dockerfiles/Dockerfile .
# Keep tags for Dependabot discovery and digests for reproducibility. Update both together.

FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine@sha256:b8c9b82affbbb82a020bbcf998dad2ba9516944ac7e883524642a481a9505e9c AS builder

ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /src

# Resolve modules first so dependency downloads are cached independently of source changes.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v} \
    go build -trimpath \
    -ldflags "-s -w -X github.com/patbaumgartner/watchtower/internal/meta.Version=${VERSION}" \
    -o /out/watchtower .

FROM --platform=$BUILDPLATFORM alpine:3.24@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a AS certs
# Exact package pins broke when Alpine 3.24 shipped newer package versions
# (ca-certificates 20261020-r0). Track the repo versions instead.
RUN apk add --no-cache ca-certificates tzdata

FROM scratch

LABEL "com.centurylinklabs.watchtower"="true"
LABEL org.opencontainers.image.source="https://github.com/patbaumgartner/watchtower"
LABEL org.opencontainers.image.description="Automates Docker container base image updates."
LABEL org.opencontainers.image.licenses="Apache-2.0"

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=certs /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /out/watchtower /watchtower

EXPOSE 8080

HEALTHCHECK CMD [ "/watchtower", "--health-check" ]

ENTRYPOINT ["/watchtower"]
