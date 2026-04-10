package daemon

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func StopPID(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read pid file %s: %w", path, err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return fmt.Errorf("parse pid file %s: %w", path, err)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("stop process %d: %w", pid, err)
	}
	return nil
}
