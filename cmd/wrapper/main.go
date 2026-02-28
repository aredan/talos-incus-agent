package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Println("incus-agent-wrapper: STARTED")
	fmt.Fprintf(os.Stderr, "incus-agent-wrapper: STARTED (stderr)\n")

	// List /dev/virtio-ports if it exists
	entries, err := os.ReadDir("/dev/virtio-ports")
	if err != nil {
		fmt.Printf("incus-agent-wrapper: /dev/virtio-ports not found: %v\n", err)
	} else {
		for _, e := range entries {
			fmt.Printf("incus-agent-wrapper: found virtio-port: %s\n", e.Name())
		}
	}

	// List /dev/disk/by-label if it exists
	entries, err = os.ReadDir("/dev/disk/by-label")
	if err != nil {
		fmt.Printf("incus-agent-wrapper: /dev/disk/by-label not found: %v\n", err)
	} else {
		for _, e := range entries {
			fmt.Printf("incus-agent-wrapper: found disk label: %s\n", e.Name())
		}
	}

	// Keep running
	for i := 0; ; i++ {
		fmt.Printf("incus-agent-wrapper: heartbeat %d\n", i)
		time.Sleep(10 * time.Second)
	}
}
