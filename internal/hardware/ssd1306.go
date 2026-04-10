package hardware

import (
	"context"
	"fmt"
	"image"
	"io"

	"github.com/l-you/pironman5-go/internal/oled"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/host/v3"
)

type NoopDisplay struct{}

func (NoopDisplay) Display(context.Context, *image.Gray, int) error { return nil }
func (NoopDisplay) Clear(context.Context) error                     { return nil }
func (NoopDisplay) Off(context.Context) error                       { return nil }
func (NoopDisplay) Close() error                                    { return nil }

type SSD1306Display struct {
	bus io.Closer
	dev *i2c.Dev
}

func NewSSD1306Display() (*SSD1306Display, error) {
	if _, err := host.Init(); err != nil {
		return nil, fmt.Errorf("init periph host: %w", err)
	}
	bus, err := i2creg.Open("/dev/i2c-1")
	if err != nil {
		return nil, fmt.Errorf("open i2c bus: %w", err)
	}
	for _, addr := range []uint16{0x3c, 0x3d} {
		display := &SSD1306Display{bus: bus, dev: &i2c.Dev{Bus: bus, Addr: addr}}
		if err := display.init(); err == nil {
			return display, nil
		}
	}
	_ = bus.Close()
	return nil, fmt.Errorf("ssd1306 not detected at 0x3c or 0x3d")
}

func (d *SSD1306Display) Display(ctx context.Context, img *image.Gray, rotation int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := d.command(0x21, 0, oled.Width-1); err != nil {
		return err
	}
	if err := d.command(0x22, 0, oled.Height/8-1); err != nil {
		return err
	}
	buf := imageToSSD1306(img, rotation)
	for i := 0; i < len(buf); i += 16 {
		end := i + 16
		if end > len(buf) {
			end = len(buf)
		}
		chunk := append([]byte{0x40}, buf[i:end]...)
		if err := d.dev.Tx(chunk, nil); err != nil {
			return fmt.Errorf("write oled data: %w", err)
		}
	}
	return nil
}

func (d *SSD1306Display) Clear(ctx context.Context) error {
	return d.Display(ctx, image.NewGray(image.Rect(0, 0, oled.Width, oled.Height)), 0)
}

func (d *SSD1306Display) Off(context.Context) error {
	return d.command(0xae)
}

func (d *SSD1306Display) Close() error {
	return d.bus.Close()
}

func (d *SSD1306Display) init() error {
	return d.command(
		0xae, 0xd5, 0x80, 0xa8, 0x3f, 0xd3, 0x00, 0x40,
		0x8d, 0x14, 0x20, 0x00, 0xa1, 0xc8, 0xda, 0x12,
		0x81, 0xcf, 0xd9, 0xf1, 0xdb, 0x40, 0xa4, 0xa6, 0xaf,
	)
}

func (d *SSD1306Display) command(commands ...byte) error {
	for _, cmd := range commands {
		if err := d.dev.Tx([]byte{0x00, cmd}, nil); err != nil {
			return fmt.Errorf("write oled command 0x%02x: %w", cmd, err)
		}
	}
	return nil
}

func imageToSSD1306(img *image.Gray, rotation int) []byte {
	buf := make([]byte, oled.Width*oled.Height/8)
	for page := 0; page < oled.Height/8; page++ {
		for x := 0; x < oled.Width; x++ {
			var b byte
			for bit := 0; bit < 8; bit++ {
				y := page*8 + bit
				sx, sy := x, y
				if rotation == 180 {
					sx = oled.Width - 1 - x
					sy = oled.Height - 1 - y
				}
				if img.GrayAt(sx, sy).Y > 127 {
					b |= 1 << bit
				}
			}
			buf[page*oled.Width+x] = b
		}
	}
	return buf
}
