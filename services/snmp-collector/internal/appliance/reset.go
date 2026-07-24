package appliance

import (
	"fmt"
	"os"
	"os/exec"
)

// FactoryReset removes mutable customer state but deliberately preserves the
// operating system, manager binaries, and every immutable installed release.
func (l Layout) FactoryReset() error {
	for _, target := range []string{l.Etc, l.Data, l.Runtime} {
		if target == "" || target == "/" || target == l.Releases {
			return fmt.Errorf("unsafe factory-reset target %q", target)
		}
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("remove %s: %w", target, err)
		}
	}
	return l.Ensure()
}

// FactoryResetAndReboot is the privileged appliance-manager action. It stops
// Compose before removing mutable state, preserves immutable software, and
// reboots into the first-boot wizard. FactoryReset remains separate for safe
// unit testing and image-build validation.
func (l Layout) FactoryResetAndReboot() error {
	if output, err := exec.Command("systemctl", "stop", "equate-stack.service").CombinedOutput(); err != nil {
		return fmt.Errorf("stop Equate stack: %w: %s", err, string(output))
	}
	if err := l.FactoryReset(); err != nil {
		return err
	}
	if err := exec.Command("systemctl", "reboot").Start(); err != nil {
		return fmt.Errorf("reboot after factory reset: %w", err)
	}
	return nil
}
