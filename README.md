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

## Deployment

There are two deployment methods. The **DaemonSet** method is recommended because Talos's containerd seccomp profile blocks the AF_VSOCK socket family needed by incus-agent.

### Method 1: DaemonSet (Recommended)

Deploys incus-agent as a privileged Kubernetes DaemonSet, bypassing the containerd seccomp restriction.

**Prerequisites:**
- Talos VM with `agent:config` disk device attached
- A running Kubernetes cluster on the VM

```bash
# 1. Attach config drive to VM (if not already done)
incus config device add <vm-name> agent disk source=agent:config

# 2. Deploy the DaemonSet
kubectl apply -f deploy/daemonset.yaml

# 3. Verify
kubectl -n incus-agent logs daemonset/incus-agent
incus ls
```

Or build and push your own image:

```bash
make build-daemonset
make push-daemonset
```

### Method 2: Talos System Extension

> **Note:** This method currently does not work due to Talos's containerd seccomp profile blocking AF_VSOCK sockets. It is kept for reference and future Talos versions that may allow seccomp overrides for extensions.

Add this extension when building your Talos image via [Image Factory](https://factory.talos.dev/) or `imager`:

```
ghcr.io/aredan/talos-incus-agent:6.22.0
```

Attach the config drive:

```bash
incus config device add <vm-name> agent disk source=agent:config
```

### Verify

```bash
# Check VM IP shows in Incus
incus ls

# Check agent logs (DaemonSet)
kubectl -n incus-agent logs daemonset/incus-agent

# Check agent logs (Extension)
talosctl logs ext-incus-agent
```

## Requirements

- Talos Linux >= v1.5.0
- Incus host with vsock support
- Kernel config: `vsock=yes`, `virtio_console=yes`
