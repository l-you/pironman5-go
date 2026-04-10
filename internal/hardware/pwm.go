package hardware

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type PWMFan struct {
	statePath string
	speedGlob string
}

func NewPWMFan() *PWMFan {
	return &PWMFan{
		statePath: "/sys/class/thermal/cooling_device0/cur_state",
		speedGlob: "/sys/devices/platform/cooling_fan/hwmon/*/fan1_input",
	}
}

func (p *PWMFan) Ready() bool {
	if _, err := os.Stat(p.statePath); err != nil {
		return false
	}
	matches, err := filepath.Glob(p.speedGlob)
	return err == nil && len(matches) > 0
}

func (p *PWMFan) Supported() bool {
	return p.Ready()
}

func (p *PWMFan) State(context.Context) (int, error) {
	data, err := os.ReadFile(p.statePath)
	if err != nil {
		return 0, fmt.Errorf("read pwm fan state: %w", err)
	}
	state, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse pwm fan state: %w", err)
	}
	return state, nil
}

func (p *PWMFan) SetState(_ context.Context, state int) error {
	if state < 0 {
		state = 0
	}
	if state > 3 {
		state = 3
	}
	if err := os.WriteFile(p.statePath, []byte(strconv.Itoa(state)), 0o644); err != nil {
		return fmt.Errorf("write pwm fan state: %w", err)
	}
	return nil
}

func (p *PWMFan) SpeedRPM(context.Context) (int, error) {
	matches, err := filepath.Glob(p.speedGlob)
	if err != nil {
		return 0, fmt.Errorf("find pwm fan speed: %w", err)
	}
	if len(matches) == 0 {
		return 0, fmt.Errorf("pwm fan speed file not found")
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		return 0, fmt.Errorf("read pwm fan speed: %w", err)
	}
	speed, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse pwm fan speed: %w", err)
	}
	return speed, nil
}

func (p *PWMFan) Off(ctx context.Context) error {
	if !p.Ready() {
		return nil
	}
	return p.SetState(ctx, 0)
}

func (p *PWMFan) Close() error { return nil }
