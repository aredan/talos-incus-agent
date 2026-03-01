package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const (
	// configDrive is the SCSI CD-ROM device that Incus attaches when
	// the "agent:config" disk device is present.
	configDrive = "/dev/sr0"

	// mountPoint is where we temporarily mount the ISO to copy certs.
	mountPoint = "/run/incus_agent_iso"

	// agentDir is the working directory for incus-agent.  It stores
	// the TLS certificates and agent configuration.
	agentDir = "/run/incus_agent"

	// virtioPort is the virtio-serial port that incus-agent uses
	// for signalling readiness to the host.
	virtioPort = "/dev/virtio-ports/org.linuxcontainers.incus"
)

func main() {
	log.SetPrefix("incus-agent-wrapper: ")
	log.SetFlags(0)

	// Resolve the absolute path to our incus-agent binary BEFORE any chdir.
	// The binary is in the same directory as this wrapper.
	selfDir, err := filepath.Abs(".")
	if err != nil {
		log.Fatalf("failed to get working directory: %v", err)
	}
	agentBin := filepath.Join(selfDir, "incus-agent")
	log.Printf("resolved incus-agent binary at %s", agentBin)

	// Verify the binary exists.
	if _, err := os.Stat(agentBin); err != nil {
		log.Fatalf("incus-agent binary not found: %v", err)
	}

	// Wait for the virtio-serial port to appear (signals that the VM
	// was launched by Incus with agent support).
	log.Println("waiting for virtio-serial port...")
	waitForPath(virtioPort, 120*time.Second)
	log.Println("virtio-serial port found")

	// Wait for the config drive to appear.
	log.Println("waiting for config drive...")
	waitForPath(configDrive, 60*time.Second)
	log.Println("config drive found")

	// Prepare directories.
	must(os.MkdirAll(mountPoint, 0o700))
	must(os.MkdirAll(agentDir, 0o700))

	// Mount the ISO config drive.
	log.Println("mounting config drive...")
	err = syscall.Mount(configDrive, mountPoint, "iso9660", syscall.MS_RDONLY, "")
	if err != nil {
		log.Fatalf("failed to mount config drive: %v", err)
	}
	log.Println("config drive mounted")

	// Copy only certificate and config files from the ISO to agentDir.
	// Skip the incus-agent and lxd-agent binaries (we use our own).
	entries, err := os.ReadDir(mountPoint)
	if err != nil {
		log.Fatalf("failed to read config drive: %v", err)
	}
	skipFiles := map[string]bool{
		"incus-agent": true,
		"lxd-agent":   true,
		"install.sh":  true,
	}
	for _, entry := range entries {
		if entry.IsDir() || skipFiles[entry.Name()] {
			continue
		}
		src := filepath.Join(mountPoint, entry.Name())
		dst := filepath.Join(agentDir, entry.Name())
		if err := copyFile(src, dst); err != nil {
			log.Fatalf("failed to copy %s: %v", entry.Name(), err)
		}
		log.Printf("copied %s", entry.Name())
	}

	// Unmount the ISO.
	if err := syscall.Unmount(mountPoint, 0); err != nil {
		log.Printf("warning: failed to unmount config drive: %v", err)
	}

	// Verify required certificate files exist.
	for _, f := range []string{"agent.crt", "agent.key", "server.crt"} {
		p := filepath.Join(agentDir, f)
		if _, err := os.Stat(p); err != nil {
			log.Fatalf("required file missing: %s", p)
		}
	}
	log.Println("certificates verified")

	// Change working directory to agentDir so incus-agent finds its certs.
	must(os.Chdir(agentDir))

	// Exec the real incus-agent using the absolute path resolved earlier.
	args := []string{"incus-agent"}
	env := os.Environ()

	log.Printf("execing incus-agent from %s (cwd: %s)", agentBin, agentDir)

	err = syscall.Exec(agentBin, args, env)
	// If we get here, exec failed.
	log.Fatalf("failed to exec incus-agent: %v", err)
}

// waitForPath polls until the given path exists or the timeout expires.
func waitForPath(path string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			log.Fatalf("timed out waiting for %s", path)
		}
		time.Sleep(time.Second)
	}
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
