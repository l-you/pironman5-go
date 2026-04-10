package oled

import (
	"testing"

	"github.com/l-you/pironman5-go/internal/config"
	"github.com/l-you/pironman5-go/internal/status"
)

func TestRenderPagesDrawPixels(t *testing.T) {
	cfg := config.System{TemperatureUnit: "C"}
	snap := status.Snapshot{
		CPUPercent:    42,
		CPUTempC:      51,
		MemoryTotal:   1024 * 1024 * 1024,
		MemoryUsed:    512 * 1024 * 1024,
		MemoryPercent: 50,
		FanRPM:        1234,
		IPs:           []status.IP{{Interface: "eth0", Address: "192.0.2.10"}},
		Disks:         []status.Disk{{Name: "nvme0n1", Total: 1000, Used: 500, Percent: 50}},
	}
	for page := 0; page < 3; page++ {
		img := Render(page, snap, cfg)
		if img.Bounds().Dx() != Width || img.Bounds().Dy() != Height {
			t.Fatalf("page %d bounds = %v", page, img.Bounds())
		}
		if countWhite(img.Pix) == 0 {
			t.Fatalf("page %d rendered no white pixels", page)
		}
	}
}

func countWhite(pixels []uint8) int {
	count := 0
	for _, pixel := range pixels {
		if pixel == 255 {
			count++
		}
	}
	return count
}
