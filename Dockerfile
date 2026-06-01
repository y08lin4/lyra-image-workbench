# syntax=docker/dockerfile:1.7

ARG NODE_VERSION=20
ARG GO_VERSION=1.22
ARG ALPINE_VERSION=3.20

FROM --platform=$BUILDPLATFORM node:${NODE_VERSION}-alpine AS web-builder
WORKDIR /src/web

COPY web/package*.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS api-builder
WORKDIR /src

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN apk add --no-cache ca-certificates

COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
COPY --from=web-builder /src/web/dist ./web/dist

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/lyra-image-workbench ./cmd/local-server

FROM alpine:${ALPINE_VERSION} AS runtime
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 app \
    && adduser -S -D -H -u 10001 -G app app \
    && mkdir -p /app/data /app/outputs /app/web/dist \
    && chown -R app:app /app

COPY --from=api-builder /out/lyra-image-workbench /app/lyra-image-workbench
COPY --from=web-builder /src/web/dist /app/web/dist

ENV LOCAL_IMAGE_HOST=0.0.0.0 \
    LOCAL_IMAGE_PORT=8787 \
    LOCAL_IMAGE_DATA_DIR=/app/data \
    LOCAL_IMAGE_WEB_DIR=/app/web/dist

EXPOSE 8787
VOLUME ["/app/data", "/app/outputs"]

USER app
ENTRYPOINT ["/app/lyra-image-workbench"]
