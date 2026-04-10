package oled

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"sync"
	"time"

	"github.com/l-you/pironman5-go/internal/config"
	"github.com/l-you/pironman5-go/internal/status"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	Width  = 128
	Height = 64
)

type Display interface {
	Display(context.Context, *image.Gray, int) error
	Clear(context.Context) error
	Off(context.Context) error
	Close() error
}

type Service struct {
	display Display
	sampler status.Sampler
	log     *slog.Logger
	mu      sync.RWMutex
	cfg     config.System
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func New(display Display, sampler status.Sampler, cfg config.System, log *slog.Logger) *Service {
	return &Service{display: display, sampler: sampler, cfg: cfg, log: log.With("service", "oled")}
}

func (s *Service) Update(cfg config.System) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
}

func (s *Service) Start(parent context.Context) {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	s.mu.Unlock()

	s.wg.Add(1)
	go s.loop(ctx)
}

func (s *Service) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.wg.Wait()
	_ = s.display.Off(context.Background())
	_ = s.display.Close()
}

func (s *Service) loop(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	page := 0
	lastPageChange := time.Now()
	for {
		cfg := s.snapshot()
		if !cfg.OLEDEnable {
			_ = s.display.Clear(ctx)
		} else {
			snap, err := s.sampler.Snapshot(ctx)
			if err != nil {
				s.log.Warn("sample oled status", "error", err)
			} else {
				if time.Since(lastPageChange) >= 5*time.Second {
					page = (page + 1) % 3
					lastPageChange = time.Now()
				}
				img := Render(page, snap, cfg)
				if err := s.display.Display(ctx, img, cfg.OLEDRotation); err != nil {
					s.log.Warn("display oled page", "error", err)
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) snapshot() config.System {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func Render(page int, snap status.Snapshot, cfg config.System) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, Width, Height))
	switch page % 3 {
	case 0:
		renderPerformance(img, snap, cfg)
	case 1:
		renderIPs(img, snap)
	default:
		renderDisks(img, snap)
	}
	return img
}

func renderPerformance(img *image.Gray, snap status.Snapshot, cfg config.System) {
	cpu := clampPercent(snap.CPUPercent)
	memory := clampPercent(snap.MemoryPercent)
	temp := snap.CPUTempC
	unit := cfg.TemperatureUnit
	if unit == "F" {
		temp = temp*9/5 + 32
	}
	drawText(img, "CPU", 0, 0)
	drawText(img, fmt.Sprintf("%3.0f%%", cpu), 0, 14)
	drawBar(img, 0, 28, 56, 7, cpu)
	drawText(img, fmt.Sprintf("%3.0f%c%s", temp, 176, unit), 72, 0)
	drawText(img, "FAN", 72, 18)
	drawText(img, fmt.Sprintf("%d", snap.FanRPM), 72, 32)
	drawText(img, "RAM", 0, 42)
	drawText(img, fmt.Sprintf("%s/%s", status.FormatBytesString(snap.MemoryUsed), status.FormatBytesString(snap.MemoryTotal)), 32, 42)
	drawBar(img, 0, 56, 126, 7, memory)
}

func renderIPs(img *image.Gray, snap status.Snapshot) {
	drawText(img, "IP", 0, 0)
	if len(snap.IPs) == 0 {
		drawText(img, "DISCONNECTED", 0, 20)
		return
	}
	for i, ip := range snap.IPs {
		if i >= 3 {
			break
		}
		drawText(img, ip.Interface, 0, 16+i*16)
		drawText(img, ip.Address, 40, 16+i*16)
	}
}

func renderDisks(img *image.Gray, snap status.Snapshot) {
	drawText(img, "DISK", 0, 0)
	if len(snap.Disks) == 0 {
		drawText(img, "NO MOUNTS", 0, 20)
		return
	}
	for i, disk := range snap.Disks {
		if i >= 3 {
			break
		}
		y := 16 + i*16
		name := disk.Name
		if len(name) > 7 {
			name = name[:7]
		}
		drawText(img, name, 0, y)
		drawText(img, fmt.Sprintf("%s/%s", status.FormatBytesString(disk.Used), status.FormatBytesString(disk.Total)), 42, y)
		drawBar(img, 0, y+12, 126, 4, clampPercent(disk.Percent))
	}
}

func drawText(img *image.Gray, text string, x, y int) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.White),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x, y+12),
	}
	d.DrawString(text)
}

func drawBar(img *image.Gray, x, y, width, height int, percent float64) {
	drawRect(img, x, y, width, height, false)
	fill := int(float64(width-2) * clampPercent(percent) / 100)
	if fill > 0 {
		fillRect(img, x+1, y+1, fill, height-2)
	}
}

func drawRect(img *image.Gray, x, y, width, height int, fill bool) {
	for yy := y; yy < y+height; yy++ {
		for xx := x; xx < x+width; xx++ {
			if fill || yy == y || yy == y+height-1 || xx == x || xx == x+width-1 {
				setWhite(img, xx, yy)
			}
		}
	}
}

func fillRect(img *image.Gray, x, y, width, height int) {
	drawRect(img, x, y, width, height, true)
}

func setWhite(img *image.Gray, x, y int) {
	if image.Pt(x, y).In(img.Bounds()) {
		img.SetGray(x, y, color.Gray{Y: 255})
	}
}

func clampPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
