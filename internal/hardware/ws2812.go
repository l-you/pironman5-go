package hardware

import (
	"context"
	"fmt"
	"os"

	"github.com/l-you/pironman5-go/internal/rgb"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/host/v3"
)

type SPILEDStrip struct {
	port spi.PortCloser
	conn spi.Conn
}

func NewSPILEDStrip() (*SPILEDStrip, error) {
	if _, err := os.Stat("/dev/spidev0.0"); err != nil {
		return nil, fmt.Errorf("spi device not available: %w", err)
	}
	if _, err := host.Init(); err != nil {
		return nil, fmt.Errorf("init periph host: %w", err)
	}
	port, err := spireg.Open("/dev/spidev0.0")
	if err != nil {
		return nil, fmt.Errorf("open spi: %w", err)
	}
	conn, err := port.Connect(2400*physic.KiloHertz, spi.Mode0, 8)
	if err != nil {
		_ = port.Close()
		return nil, fmt.Errorf("connect spi: %w", err)
	}
	return &SPILEDStrip{port: port, conn: conn}, nil
}

func (s *SPILEDStrip) Show(ctx context.Context, colors []rgb.Color) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data := EncodeWS2812GRB(colors)
	if err := s.conn.Tx(data, nil); err != nil {
		return fmt.Errorf("write ws2812 spi frame: %w", err)
	}
	return nil
}

func (s *SPILEDStrip) Close() error {
	return s.port.Close()
}

func EncodeWS2812GRB(colors []rgb.Color) []byte {
	bitCount := len(colors) * 24 * 3
	data := make([]byte, 8+(bitCount+7)/8+8)
	bitPos := 8 * 8
	for _, color := range colors {
		for _, component := range []uint8{color.G, color.R, color.B} {
			for bit := 7; bit >= 0; bit-- {
				pattern := uint8(0b100)
				if component&(1<<bit) != 0 {
					pattern = 0b110
				}
				for p := 2; p >= 0; p-- {
					if pattern&(1<<p) != 0 {
						data[bitPos/8] |= 1 << (7 - bitPos%8)
					}
					bitPos++
				}
			}
		}
	}
	return data
}
