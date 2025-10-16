package utils

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/mermonia/peridot/internal/logger"
)

func ExecHook(hook string) error {
	if hook == "" {
		return nil
	}

	parts := strings.Fields(hook)
	cmd := exec.Command(parts[0], parts[1:]...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Warn("The executed hook returned an error", "hook", hook, "error", err)
		return fmt.Errorf("the executed hook returned an error: %w", err)
	}

	fmt.Print(string(output))
	return nil
}
