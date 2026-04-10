package hardware

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/warthog618/go-gpiocdev"
)

type GPIODOutput struct {
	line *gpiocdev.Line
}

func NewGPIODOutput(offset int) (*GPIODOutput, error) {
	chips := []string{"/dev/gpiochip4", "/dev/gpiochip0", "/dev/gpiochip1"}
	if configured := strings.TrimSpace(os.Getenv("PIRONMAN5_GPIOCHIP")); configured != "" {
		chips = append([]string{configured}, chips...)
	}

	var errs []error
	for _, chip := range chips {
		if _, err := os.Stat(chip); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", chip, err))
			continue
		}
		line, err := gpiocdev.RequestLine(chip, offset, gpiocdev.AsOutput(0), gpiocdev.WithConsumer("pironman5"))
		if err != nil {
			errs = append(errs, fmt.Errorf("%s offset %d: %w", chip, offset, err))
			continue
		}
		return &GPIODOutput{line: line}, nil
	}
	return nil, errors.Join(errs...)
}

func (g *GPIODOutput) Set(_ context.Context, on bool) error {
	value := 0
	if on {
		value = 1
	}
	if err := g.line.SetValue(value); err != nil {
		return fmt.Errorf("set gpio value: %w", err)
	}
	return nil
}

func (g *GPIODOutput) Close() error {
	return g.line.Close()
}
