package hardware

import (
	"context"

	"github.com/l-you/pironman5-go/internal/rgb"
)

type NoopDigitalOutput struct{}

func (NoopDigitalOutput) Set(context.Context, bool) error { return nil }
func (NoopDigitalOutput) Close() error                    { return nil }

type NoopStrip struct{}

func (NoopStrip) Show(context.Context, []rgb.Color) error { return nil }
func (NoopStrip) Close() error                            { return nil }
