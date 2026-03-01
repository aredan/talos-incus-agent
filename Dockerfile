# Stage 1: Build incus-agent from Incus source
FROM golang:1.25-alpine AS incus-builder

ARG INCUS_VERSION=v6.22.0

RUN apk add --no-cache git gcc musl-dev linux-headers acl-dev acl-static

WORKDIR /src
RUN git clone --depth 1 --branch ${INCUS_VERSION} https://github.com/lxc/incus.git .

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -tags agent,netgo \
    -ldflags "-s -w -linkmode external -extldflags '-static'" \
    -o /incus-agent \
    ./cmd/incus-agent

# Stage 2: Build the wrapper
FROM golang:1.23-alpine AS wrapper-builder

WORKDIR /src
COPY cmd/wrapper/ .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o /incus-agent-wrapper .

# Stage 3: Package as Talos system extension
FROM scratch

COPY manifest.yaml /
COPY incus-agent.yaml /rootfs/usr/local/etc/containers/incus-agent.yaml
COPY --from=incus-builder /incus-agent /rootfs/usr/local/lib/containers/incus-agent/incus-agent
COPY --from=wrapper-builder /incus-agent-wrapper /rootfs/usr/local/lib/containers/incus-agent/incus-agent-wrapper
