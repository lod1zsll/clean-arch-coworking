#!/bin/sh
# build.sh
set -eu

APP="${1:?usage: ./build.sh <app-name> [run] [-t <tag>]}"
CMD="${2:-}"
TAG="latest"

shift 2 2>/dev/null || shift $# 2>/dev/null || true

while [ $# -gt 0 ]; do
	case "$1" in
	-t)
		TAG="${2:?-t requires a value}"
		shift 2
		;;
	*)
		echo "unknown option: $1"
		exit 1
		;;
	esac
done

echo ">>> Running tests..."
go test ./...

echo ">>> Building image outbox-${APP}:${TAG}..."
docker build \
	--build-arg APP="$APP" \
	-t "outbox-${APP}:${TAG}" \
	.

if [ "${CMD}" = "run" ]; then
	echo ">>> Starting outbox-${APP} via docker compose..."
	docker compose -f ./docker/docker-compose.yml up -d
fi
