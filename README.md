# Talos Incus Agent

Runs the [Incus](https://linuxcontainers.org/incus/) VM agent inside [Talos Linux](https://www.talos.dev/) VMs as a Kubernetes DaemonSet. This allows `incus ls` to show VM IP addresses, CPU, and memory usage.

## Problem

Talos Linux VMs in Incus don't show their IP in `incus ls` because Talos doesn't ship the `incus-agent`. The agent communicates VM state back to the host via vsock.

## How It Works

1. Incus attaches a config drive ISO (with TLS certificates) to the VM via the `agent:config` disk device
2. A DaemonSet pod runs on each node with privileged access
3. A Go wrapper mounts the ISO, copies TLS certs to `/run/incus_agent/`, then execs the real `incus-agent`
4. The agent communicates with the Incus host via vsock and signals readiness via virtio-serial
5. `incus ls` now shows the VM's IP address

## Deployment

### 1. Attach config drive to each VM

```bash
incus config device add <vm-name> agent disk source=agent:config
```

### 2. Deploy the DaemonSet

```bash
kubectl apply -f deploy/daemonset.yaml
```

### 3. Verify

```bash
kubectl -n incus-agent logs daemonset/incus-agent
incus ls
```

## Building

```bash
make build
make push
```

## Requirements

- Talos Linux >= v1.5.0
- Incus host with vsock support
- `agent:config` disk device attached to VMs
