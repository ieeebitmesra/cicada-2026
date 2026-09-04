# Multi-stage Dockerfile for web-global-megaphone
FROM golang:1.22-alpine AS builder

WORKDIR /build
COPY go.mod ./
COPY webmed.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server webmed.go

# Minimal runner stage
FROM alpine:3.20

RUN adduser -D -u 1000 ctf

WORKDIR /home/ctf
COPY --from=builder /build/server .
RUN chown -R root:ctf /home/ctf && \
    chmod 750 /home/ctf/server

USER ctf
EXPOSE 8080

CMD ["./server"]
