# Stage 1: Build the incus-agent binary
FROM golang:1.23-alpine AS incus-builder

RUN apk add --no-cache git make

WORKDIR /src
RUN git clone --depth 1 --branch v6.22.0 https://github.com/lxc/incus.git .

RUN CGO_ENABLED=0 go build -tags "agent,netgo" -ldflags "-s -w" -o /incus-agent ./cmd/incus-agent

# Stage 2: Build the wrapper binary
FROM golang:1.23-alpine AS wrapper-builder

WORKDIR /src
COPY cmd/wrapper/ .

RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /incus-agent-wrapper .

# Stage 3: Package as Talos system extension
FROM scratch

COPY manifest.yaml /
COPY incus-agent.yaml /rootfs/etc/cri/conf.d/incus-agent.yaml

COPY --from=incus-builder /incus-agent /rootfs/usr/local/lib/containers/incus-agent/incus-agent
COPY --from=wrapper-builder /incus-agent-wrapper /rootfs/usr/local/lib/containers/incus-agent/incus-agent-wrapper
