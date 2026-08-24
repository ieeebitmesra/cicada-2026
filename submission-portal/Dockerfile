# syntax=docker/dockerfile:1
# Submission Portal — hardened multi-stage build (no CGO → static binary).
#
# NOTE: dependency chain (modernc.org/sqlite, charm libs) requires Go 1.25 —
# keep this base image in sync with the `go` directive in go.mod.

FROM golang:1.25-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOFLAGS=-trimpath GOTOOLCHAIN=local

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations

RUN go build -o /out/server ./cmd/server \
 && go build -o /out/admin  ./cmd/admin

FROM alpine:3.20 AS runtime
RUN addgroup -S ctf && adduser -S ctf -G ctf \
 && apk add --no-cache ca-certificates tzdata \
 && mkdir -p /app/data /app/configs /app/migrations \
 && chown -R ctf:ctf /app/data

WORKDIR /app
COPY --from=build /out/server /out/admin ./
COPY configs/server.yaml configs/rounds.yaml ./configs/
COPY migrations ./migrations

USER ctf
EXPOSE 2222

ENV CTF_CONFIG=/app/configs/server.yaml \
    CTF_ROUNDS=/app/configs/rounds.yaml

ENTRYPOINT ["./server"]
