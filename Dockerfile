FROM golang:1.26.2-trixie AS build

WORKDIR /usr/src/app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    set -eux; \
    go mod download; \
    go mod verify

COPY cmd ./cmd
COPY internal ./internal

ARG APP

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    set -eux; \
    test -n "$APP"; \
    go build -o /out/application ./cmd/${APP}

FROM debian:13-slim

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends ca-certificates; \
    rm -rf /var/lib/apt/lists/*; \
    groupadd -r app; \
    useradd -r -g app -s /usr/sbin/nologin app

COPY --from=build /out/application /usr/local/bin/application

USER app

ENTRYPOINT ["/usr/local/bin/application"]