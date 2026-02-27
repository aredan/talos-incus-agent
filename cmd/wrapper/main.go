// Package main implements a wrapper that mounts the Incus config drive,
// copies TLS certificates, and execs the real incus-agent binary.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const (
	agentDir     = "/var/lib/incus-agent"
	isoDevice    = "/dev/disk/by-label/incus_agent"
	isoMount     = "/mnt/incus_agent_iso"
	agentBin     = "./incus-agent"
	virtioPort   = "/dev/virtio-ports/org.linuxcontainers.incus"
	waitTimeout  = 120 // seconds to wait for devices
	waitInterval = 2   // seconds between checks
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "incus-agent-wrapper: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("incus-agent-wrapper: starting")

	// Wait for the virtio-serial port to appear.
	fmt.Printf("incus-agent-wrapper: waiting for %s\n", virtioPort)
	if err := waitForPath(virtioPort); err != nil {
		return err
	}
	fmt.Printf("incus-agent-wrapper: found %s\n", virtioPort)

	// Ensure the agent runtime directory exists.
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return fmt.Errorf("creating agent dir: %w", err)
	}

	// Check if certs already exist (restart resilience).
	if !certsExist() {
		// Wait for the config drive to appear.
		fmt.Printf("incus-agent-wrapper: waiting for %s\n", isoDevice)
		if err := waitForPath(isoDevice); err != nil {
			return err
		}

		// Mount the ISO config drive.
		if err := os.MkdirAll(isoMount, 0o755); err != nil {
			return fmt.Errorf("creating iso mount dir: %w", err)
		}

		fmt.Printf("incus-agent-wrapper: mounting %s\n", isoDevice)
		if err := syscall.Mount(isoDevice, isoMount, "iso9660", syscall.MS_RDONLY, ""); err != nil {
			return fmt.Errorf("mounting config drive %s: %w", isoDevice, err)
		}

		// List what's on the ISO for debugging.
		entries, _ := os.ReadDir(isoMount)
		for _, e := range entries {
			fmt.Printf("incus-agent-wrapper: ISO contains: %s\n", e.Name())
		}

		// Copy all files from ISO to the agent directory.
		if err := copyDir(isoMount, agentDir); err != nil {
			_ = syscall.Unmount(isoMount, 0)
			return fmt.Errorf("copying config files: %w", err)
		}

		// Unmount the ISO.
		if err := syscall.Unmount(isoMount, 0); err != nil {
			return fmt.Errorf("unmounting config drive: %w", err)
		}

		// Verify required files are present.
		for _, name := range []string{"agent.crt", "agent.key", "server.crt"} {
			path := filepath.Join(agentDir, name)
			if _, err := os.Stat(path); err != nil {
				return fmt.Errorf("required file missing after ISO copy: %s", path)
			}
		}

		fmt.Println("incus-agent-wrapper: certificates copied from config drive")
	} else {
		fmt.Println("incus-agent-wrapper: certificates already present, skipping ISO mount")
	}

	// Exec the real incus-agent.
	argv := []string{agentBin, "--secrets-location", agentDir}
	fmt.Printf("incus-agent-wrapper: execing %s %v\n", agentBin, argv[1:])

	return syscall.Exec(agentBin, argv, os.Environ())
}

// waitForPath waits until the given path exists or times out.
func waitForPath(path string) error {
	deadline := time.Now().Add(time.Duration(waitTimeout) * time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for %s after %ds", path, waitTimeout)
		}
		time.Sleep(time.Duration(waitInterval) * time.Second)
	}
}

// certsExist checks whether the required TLS cert files already exist.
func certsExist() bool {
	for _, name := range []string{"agent.crt", "agent.key", "server.crt"} {
		if _, err := os.Stat(filepath.Join(agentDir, name)); err != nil {
			return false
		}
	}
	return true
}

// copyDir copies all files from src directory to dst directory (non-recursive).
func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("reading dir %s: %w", src, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copying %s -> %s: %w", src, dst, err)
	}

	return out.Close()
}
