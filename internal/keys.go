package lab

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// EnsureKey generates an ed25519 key pair at keys/<stack>/id_ed25519 if it
// does not already exist.
func EnsureKey(stack string) error {
	keyDir := filepath.Join("keys", stack)
	keyPath := filepath.Join(keyDir, "id_ed25519")

	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return fmt.Errorf("create key dir: %w", err)
	}

	if _, err := os.Stat(keyPath); err == nil {
		Skip(fmt.Sprintf("SSH key already exists for stack %s", stack))
		return nil
	}

	Info(fmt.Sprintf("Generating SSH key for stack %s", stack))
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-q")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-keygen: %w", err)
	}

	Ok(fmt.Sprintf("Key generated at %s", keyPath))
	return nil
}
