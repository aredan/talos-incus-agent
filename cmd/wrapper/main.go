// Package main implements a wrapper that mounts the Incus config drive,
// copies TLS certificates, and execs the real incus-agent binary.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

const (
	agentDir   = "/run/incus_agent"
	isoDevice  = "/dev/disk/by-label/incus_agent"
	isoMount   = "/mnt/incus_agent_iso"
	agentBin   = "./incus-agent"

	requiredFiles = "agent.crt,agent.key,server.crt"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "incus-agent-wrapper: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Ensure the agent runtime directory exists.
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return fmt.Errorf("creating agent dir: %w", err)
	}

	// Check if certs already exist (restart resilience).
	if !certsExist() {
		// Mount the ISO config drive.
		if err := os.MkdirAll(isoMount, 0o755); err != nil {
			return fmt.Errorf("creating iso mount dir: %w", err)
		}

		if err := syscall.Mount(isoDevice, isoMount, "iso9660", syscall.MS_RDONLY, ""); err != nil {
			return fmt.Errorf("mounting config drive %s: %w", isoDevice, err)
		}

		// Copy all files from ISO to the agent directory.
		if err := copyDir(isoMount, agentDir); err != nil {
			// Try to unmount even on error.
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
	fmt.Printf("incus-agent-wrapper: execing %s\n", agentBin)

	return syscall.Exec(agentBin, argv, os.Environ())
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
