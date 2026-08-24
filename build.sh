#!/bin/bash
set -e

BINFILE=watchtower
if [ -n "$MSYSTEM" ]; then
    BINFILE=watchtower.exe
fi
VERSION=$(git describe --tags 2>/dev/null) || VERSION=unknown
echo "Building $VERSION..."
go build -o "$BINFILE" -ldflags "-X github.com/patbaumgartner/watchtower/internal/meta.Version=$VERSION"
