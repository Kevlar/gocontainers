package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	// If not running in containerized context, re-execute self with new namespaces.
	if os.Getenv("CONTAINER") != "1" {
		cmd := exec.Command("/proc/self/exe")
		cmd.Env = append(os.Environ(), "CONTAINER=1")
		// Create new PID, mount, network, UTS and IPC namespaces.
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWPID |
				syscall.CLONE_NEWNS |   // New mount namespace
				syscall.CLONE_NEWNET |
				syscall.CLONE_NEWUTS |
				syscall.CLONE_NEWIPC,
		}
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			log.Fatalf("Error starting container process: %v", err)
		}
		return
	}

	// We are inside the containerized process.
	// Because we've unshared the PID namespace, this process will be PID 1.
	fmt.Printf("Inside container, PID: %d\n", os.Getpid())

	// IMPORTANT: Ensure the mount propagation is private in our new mount namespace,
	// so that mounting a new proc filesystem doesn't affect the host.
	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		log.Fatalf("Failed to remount root as private: %v", err)
	}

	// Now mount a new proc filesystem over /proc.
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		log.Fatalf("Failed to mount /proc: %v", err)
	}

	// Change the hostname (possible because we are in a new UTS namespace).
	if err := syscall.Sethostname([]byte("mycontainer")); err != nil {
		log.Fatalf("Failed setting hostname: %v", err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Unable to get hostname: %v", err)
	}
	fmt.Printf("Hostname set to: %s\n", hostname)

	// Launch an interactive shell.
	cmd := exec.Command("sh")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Shell command failed: %v", err)
	}
}