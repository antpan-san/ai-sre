package cli

import (
	"errors"
	"fmt"
	"os"
)

func requireOpsMutationConfirm(yes bool, action string) error {
	if yes || isStdinTTY() {
		return nil
	}
	return fmt.Errorf("%s 在非 TTY 环境需显式确认，请追加 --yes", action)
}

func requireOpsRoot(action string) error {
	if os.Geteuid() == 0 {
		return nil
	}
	return errors.New(action + " 需 root 权限，请使用: sudo " + progName + " ops …")
}
