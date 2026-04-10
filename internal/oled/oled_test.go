package oled

import (
	"image"
	"image/color"
	"path/filepath"
	"strings"
	"testing"

	"github.com/l-you/pironman5-go/internal/config"
	"github.com/l-you/pironman5-go/internal/pbm"
	"github.com/l-you/pironman5-go/internal/status"
)

func TestRenderPagesDrawPixels(t *testing.T) {
	cfg := config.System{
		TemperatureUnit:      "C",
		OLEDPageMode:         config.OLEDPageModeAuto,
		OLEDPage:             config.OLEDPagePerformance,
		OLEDDisk:             "total",
		OLEDNetworkInterface: "all",
	}
	snap := status.Snapshot{
		CPUPercent:    42,
		CPUTempC:      51,
		MemoryTotal:   1024 * 1024 * 1024,
		MemoryUsed:    512 * 1024 * 1024,
		MemoryPercent: 50,
		FanRPM:        1234,
		IPs: []status.IP{
			{Interface: "eth0", Address: "192.0.2.10"},
			{Interface: "wlan0", Address: "198.51.100.24"},
		},
		Disks: []status.Disk{
			{Name: "nvme0n1", Mount: "/", Total: 1000, Used: 500, Percent: 50},
			{Name: "sda1", Mount: "/mnt/data", Total: 2000, Used: 500, Percent: 25},
		},
	}
	for _, page := range Pages(cfg) {
		img := Render(page, snap, cfg)
		if img.Bounds().Dx() != Width || img.Bounds().Dy() != Height {
			t.Fatalf("page %s bounds = %v", page, img.Bounds())
		}
		if countWhite(img.Pix) == 0 {
			t.Fatalf("page %s rendered no white pixels", page)
		}
	}
}

func TestRenderCustomPBMPage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logo.pbm")
	src := image.NewGray(image.Rect(0, 0, Width, Height))
	for y := 12; y < 52; y++ {
		for x := 24; x < 104; x++ {
			src.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	if err := pbm.EncodeFile(path, src); err != nil {
		t.Fatal(err)
	}

	cfg := config.System{OLEDImagePath: path}
	img := Render(config.OLEDPageImage, status.Snapshot{}, cfg)
	if img.Bounds().Dx() != Width || img.Bounds().Dy() != Height {
		t.Fatalf("bounds = %v", img.Bounds())
	}
	if countWhite(img.Pix) == 0 {
		t.Fatal("custom image rendered no white pixels")
	}
}

func TestRenderCustomImageRejectsWrongPBMSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "small.pbm")
	src := image.NewGray(image.Rect(0, 0, 8, 8))
	src.SetGray(0, 0, color.Gray{Y: 255})
	if err := pbm.EncodeFile(path, src); err != nil {
		t.Fatal(err)
	}
	cfg := config.System{OLEDImagePath: path}
	img := Render(config.OLEDPageImage, status.Snapshot{}, cfg)
	if countWhite(img.Pix) == 0 {
		t.Fatal("expected error text for invalid PBM size")
	}
}

func TestFixedPage(t *testing.T) {
	cfg := config.System{OLEDPageMode: config.OLEDPageModeFixed, OLEDPage: config.OLEDPageHeart}
	pages := Pages(cfg)
	if len(pages) != 1 || pages[0] != config.OLEDPageHeart {
		t.Fatalf("pages = %#v, want heart only", pages)
	}
}

func TestAutoPagesIncludeImageOnlyWhenConfigured(t *testing.T) {
	withoutImage := Pages(config.System{OLEDPageMode: config.OLEDPageModeAuto})
	for _, page := range withoutImage {
		if page == config.OLEDPageImage {
			t.Fatalf("unexpected image page without image path: %#v", withoutImage)
		}
	}
	withImage := Pages(config.System{OLEDPageMode: config.OLEDPageModeAuto, OLEDImagePath: "/tmp/oled.pbm"})
	if withImage[len(withImage)-1] != config.OLEDPageImage {
		t.Fatalf("pages = %#v, want image page last", withImage)
	}
}

func TestTemperatureLabelUsesASCII(t *testing.T) {
	label := temperatureLabel(51.2, "C")
	if strings.Contains(label, "?") || strings.Contains(label, "\u00b0") {
		t.Fatalf("temperature label = %q, want ASCII without degree glyph", label)
	}
	if label != " 51 degC" {
		t.Fatalf("temperature label = %q, want %q", label, " 51 degC")
	}
}

func TestSelectsConfiguredIPAndDisk(t *testing.T) {
	snap := status.Snapshot{
		IPs: []status.IP{
			{Interface: "eth0", Address: "192.0.2.10"},
			{Interface: "wlan0", Address: "198.51.100.24"},
		},
		Disks: []status.Disk{
			{Name: "root", Mount: "/", Total: 1000, Used: 250, Percent: 25},
			{Name: "data", Mount: "/mnt/data", Total: 3000, Used: 1500, Percent: 50},
		},
	}
	cfg := config.System{OLEDNetworkInterface: "wlan0", OLEDDisk: "/mnt/data"}
	ips := selectedIPs(snap, cfg)
	if len(ips) != 1 || ips[0].Interface != "wlan0" {
		t.Fatalf("ips = %#v, want wlan0", ips)
	}
	disks := selectedDisks(snap, cfg)
	if len(disks) != 1 || disks[0].Name != "data" {
		t.Fatalf("disks = %#v, want data", disks)
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
