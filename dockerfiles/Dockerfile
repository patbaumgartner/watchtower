# syntax=docker/dockerfile:1@sha256:ecfaec9ed6d810b56388c508f4121597bfbba70d41a6dfeee4d8cad5f295fc32
#
# Cross-compiles watchtower for the platform requested by buildx and packages it
# into a scratch image. Build with:
#
#   docker buildx build --platform linux/amd64,linux/arm64 -f dockerfiles/Dockerfile .
# Keep tags for Dependabot discovery and digests for reproducibility. Update both together.

FROM --platform=$BUILDPLATFORM golang:1.27.0-alpine@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc AS builder

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

FROM --platform=$BUILDPLATFORM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b AS certs
# Package versions are coupled to this Alpine release and must remain exact.
RUN apk add --no-cache ca-certificates=20260611-r0 tzdata=2026c-r0

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
