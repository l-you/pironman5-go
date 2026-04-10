package oled

import (
	"context"
	"image"
	"image/color"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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

	cfg := config.System{OLEDImagePath: path, OLEDImagePaths: []string{path}, OLEDImageInterval: config.DefaultOLEDImageInterval}
	cfg.Normalize()
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
	cfg := config.System{OLEDImagePath: path, OLEDImagePaths: []string{path}, OLEDImageInterval: config.DefaultOLEDImageInterval}
	cfg.Normalize()
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
	cfg := config.System{OLEDPageMode: config.OLEDPageModeAuto, OLEDImagePaths: []string{"/tmp/a.pbm", "/tmp/b.pbm"}, OLEDImageInterval: config.DefaultOLEDImageInterval}
	cfg.Normalize()
	withImage := Pages(cfg)
	if withImage[len(withImage)-1] != config.OLEDPageImage {
		t.Fatalf("pages = %#v, want image page last", withImage)
	}
}

func TestNextImageIndex(t *testing.T) {
	start := time.Unix(10, 0)
	index, changed := nextImageIndex(3, 0, start, start.Add(7*time.Second), 3*time.Second)
	if index != 2 {
		t.Fatalf("index = %d, want 2", index)
	}
	if !changed.Equal(start.Add(6 * time.Second)) {
		t.Fatalf("last change = %s", changed)
	}
}

func TestFixedSingleImageHasNoRefreshTimer(t *testing.T) {
	now := time.Unix(10, 0)
	cfg := config.System{
		OLEDEnable:        true,
		OLEDPageMode:      config.OLEDPageModeFixed,
		OLEDPage:          config.OLEDPageImage,
		OLEDImagePath:     "/tmp/only.pbm",
		OLEDImagePaths:    []string{"/tmp/only.pbm"},
		OLEDImageInterval: config.DefaultOLEDImageInterval,
	}
	cfg.Normalize()
	current := (&loopState{}).current(cfg, now)
	if current.nextWait != 0 {
		t.Fatalf("next wait = %s, want no scheduled refresh", current.nextWait)
	}
}

func TestFixedImageRotationUsesConfiguredInterval(t *testing.T) {
	now := time.Unix(10, 0)
	cfg := config.System{
		OLEDEnable:        true,
		OLEDPageMode:      config.OLEDPageModeFixed,
		OLEDPage:          config.OLEDPageImage,
		OLEDImagePath:     "/tmp/a.pbm",
		OLEDImagePaths:    []string{"/tmp/a.pbm", "/tmp/b.pbm"},
		OLEDImageInterval: 3,
	}
	cfg.Normalize()
	current := (&loopState{}).current(cfg, now)
	if current.nextWait != 3*time.Second {
		t.Fatalf("next wait = %s, want 3s", current.nextWait)
	}
}

func TestFixedImageServiceDrawsOnceAndSleeps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logo.pbm")
	src := image.NewGray(image.Rect(0, 0, Width, Height))
	for y := 8; y < Height-8; y++ {
		for x := 8; x < Width-8; x++ {
			src.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	if err := pbm.EncodeFile(path, src); err != nil {
		t.Fatal(err)
	}

	cfg := config.System{
		OLEDEnable:        true,
		OLEDPageMode:      config.OLEDPageModeFixed,
		OLEDPage:          config.OLEDPageImage,
		OLEDImagePath:     path,
		OLEDImagePaths:    []string{path},
		OLEDImageInterval: config.DefaultOLEDImageInterval,
	}
	cfg.Normalize()

	display := &fakeDisplay{notify: make(chan struct{}, 4)}
	sampler := &fakeSampler{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := New(display, sampler, cfg, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.Start(ctx)
	select {
	case <-display.notify:
	case <-time.After(2 * time.Second):
		service.Stop()
		t.Fatal("timed out waiting for first OLED draw")
	}

	time.Sleep(1200 * time.Millisecond)
	service.Stop()

	if got := display.Calls(); got != 1 {
		t.Fatalf("display calls = %d, want 1", got)
	}
	if got := sampler.Calls(); got != 0 {
		t.Fatalf("sampler calls = %d, want 0 for fixed image", got)
	}
}

func TestSelectedImagePath(t *testing.T) {
	if got := selectedImagePath([]string{"a.pbm", "b.pbm"}, 1); got != "b.pbm" {
		t.Fatalf("selected path = %q", got)
	}
	if got := selectedImagePath([]string{"a.pbm", "b.pbm"}, 9); got != "a.pbm" {
		t.Fatalf("fallback path = %q", got)
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

type fakeDisplay struct {
	mu     sync.Mutex
	calls  int
	notify chan struct{}
}

func (d *fakeDisplay) Display(context.Context, *image.Gray, int) error {
	d.mu.Lock()
	d.calls++
	ch := d.notify
	d.mu.Unlock()
	if ch != nil {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	return nil
}

func (*fakeDisplay) Clear(context.Context) error { return nil }
func (*fakeDisplay) Off(context.Context) error   { return nil }
func (*fakeDisplay) Close() error                { return nil }

func (d *fakeDisplay) Calls() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.calls
}

type fakeSampler struct {
	mu    sync.Mutex
	calls int
}

func (s *fakeSampler) Snapshot(context.Context) (status.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return status.Snapshot{}, nil
}

func (*fakeSampler) CPUTemperatureC(context.Context) (float64, error) {
	return 0, nil
}

func (s *fakeSampler) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}
