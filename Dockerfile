# Stage 1: Build the test wrapper
FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY cmd/wrapper/ .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o /incus-agent-wrapper .

# Stage 2: Package as Talos system extension
FROM scratch

COPY manifest.yaml /
COPY incus-agent.yaml /rootfs/usr/local/etc/containers/incus-agent.yaml
COPY --from=builder /incus-agent-wrapper /rootfs/usr/local/lib/containers/incus-agent/incus-agent-wrapper
