FROM golang:1.26.2-alpine AS build

WORKDIR /usr/src/app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    set -eux; \
    go mod download; \
    go mod verify

COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

ARG APP

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    set -eux; \
    test -n "$APP"; \
    CGO_ENABLED=0 GOOS=linux go build \
      -ldflags="-w -s" \
      -o /out/application ./cmd/${APP}

FROM alpine:3.20

ARG APP_USER=10001

RUN set -eux; \
    apk add --no-cache ca-certificates; \
    addgroup -S -g ${APP_USER} app; \
    adduser -S -u ${APP_USER} -G app app

COPY --from=build /out/application /usr/local/bin/application

USER app

ENTRYPOINT ["/usr/local/bin/application"]
