#!/bin/sh
# build.sh
set -eu

APP="${1:?usage: ./build.sh <app-name> [tag]}"
TAG="${2:-latest}"

echo ">>> Running tests..."
go test ./...

echo ">>> Building image outbox-${APP}:${TAG}..."
docker build \
    --build-arg APP="$APP" \
    -t "outbox-${APP}:${TAG}" \
    .