# Talos Incus Agent Extension

A [Talos Linux](https://www.talos.dev/) system extension that packages the [Incus](https://linuxcontainers.org/incus/) VM agent. This allows Talos VMs running in Incus to report their IP address, CPU, and memory usage back to the Incus host.

## Problem

Talos Linux VMs in Incus don't show their IP in `incus ls` because Talos doesn't ship the `incus-agent`. The agent is responsible for communicating VM state back to the host via vsock.

## How It Works

1. Incus attaches a config drive ISO (with TLS certificates) to the VM via the `agent:config` disk device
2. Talos boots and starts the `incus-agent` extension service
3. A Go wrapper mounts the ISO, copies TLS certs to `/run/incus_agent/`, then execs the real `incus-agent`
4. The agent communicates with the Incus host via vsock and signals readiness via virtio-serial
5. `incus ls` now shows the VM's IP address

## Building

```bash
# Build the extension image
make build

# Build and push to registry
make push

# Customize registry/tag
make push REGISTRY=ghcr.io IMAGE_NAME=myorg/talos-incus-agent TAG=v6.22.0
```

## Usage

### 1. Include in Talos Image

Add this extension when building your Talos image via [Image Factory](https://factory.talos.dev/) or `imager`:

```
ghcr.io/aredan/talos-incus-agent:v6.22.0
```

### 2. Attach Config Drive

Ensure VMs have the Incus agent config drive attached. If using `omni-incus-infra-provider`, this is done automatically. Otherwise, add the device manually:

```bash
incus config device add <vm-name> agent disk source=agent:config
```

### 3. Verify

After booting:

```bash
# Check agent logs
talosctl logs ext-incus-agent

# Verify IP shows in Incus
incus ls
```

## Requirements

- Talos Linux >= v1.5.0
- Incus host with vsock support
- Kernel config: `vsock=yes`, `virtio_console=yes`
