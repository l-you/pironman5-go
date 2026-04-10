package rgb

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/l-you/pironman5-go/internal/config"
)

type Color struct {
	R uint8
	G uint8
	B uint8
}

type Strip interface {
	Show(context.Context, []Color) error
	Close() error
}

type Service struct {
	strip  Strip
	log    *slog.Logger
	mu     sync.RWMutex
	cfg    config.System
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New(strip Strip, cfg config.System, log *slog.Logger) *Service {
	return &Service{strip: strip, cfg: cfg, log: log.With("service", "rgb")}
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
	_ = s.strip.Show(context.Background(), nil)
	_ = s.strip.Close()
}

func (s *Service) loop(ctx context.Context) {
	defer s.wg.Done()
	counter := 0
	for {
		cfg := s.snapshot()
		pattern, delay := Generate(cfg, counter)
		if err := s.strip.Show(ctx, pattern); err != nil {
			s.log.Warn("show rgb frame", "error", err)
			delay = 5 * time.Second
		}
		counter++
		if !sleep(ctx, delay) {
			return
		}
	}
}

func (s *Service) snapshot() config.System {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func Generate(cfg config.System, counter int) ([]Color, time.Duration) {
	if cfg.RGBLEDCount < 1 {
		return nil, time.Second
	}
	colors := make([]Color, cfg.RGBLEDCount)
	if !cfg.RGBEnable {
		return colors, time.Second
	}
	base, err := parseBaseColor(cfg)
	if err != nil {
		return colors, time.Second
	}

	switch cfg.RGBStyle {
	case "solid":
		fill(colors, scale(base, cfg.RGBBrightness))
		return colors, time.Second
	case "breathing":
		counter %= 200
		brightness := counter
		if counter >= 100 {
			brightness = 200 - counter
		}
		fill(colors, scale(base, cfg.RGBBrightness*brightness/100))
		return colors, mapDelay(cfg.RGBSpeed, 100*time.Millisecond, time.Millisecond)
	case "flow", "flow_reverse":
		index := counter % cfg.RGBLEDCount
		if cfg.RGBStyle == "flow_reverse" {
			index = cfg.RGBLEDCount - 1 - index
		}
		colors[index] = scale(base, cfg.RGBBrightness)
		return colors, mapDelay(cfg.RGBSpeed, 500*time.Millisecond, 100*time.Millisecond)
	case "rainbow", "rainbow_reverse":
		reverse := cfg.RGBStyle == "rainbow_reverse"
		for i := range colors {
			idx := i
			if reverse {
				idx = cfg.RGBLEDCount - 1 - i
			}
			hue := float64(idx)*360/float64(cfg.RGBLEDCount) + float64(counter%360)
			colors[i] = HSLToRGB(hue, 1, float64(cfg.RGBBrightness)/100)
		}
		return colors, mapDelay(cfg.RGBSpeed, 100*time.Millisecond, 5*time.Millisecond)
	case "hue_cycle":
		fill(colors, HSLToRGB(float64(counter%360), 1, float64(cfg.RGBBrightness)/100))
		return colors, mapDelay(cfg.RGBSpeed, 100*time.Millisecond, 5*time.Millisecond)
	default:
		fill(colors, scale(base, cfg.RGBBrightness))
		return colors, time.Second
	}
}

func parseBaseColor(cfg config.System) (Color, error) {
	parsed, err := config.ParseHexColor(cfg.RGBColor)
	if err != nil {
		return Color{}, fmt.Errorf("parse rgb color: %w", err)
	}
	return Color{R: parsed.R, G: parsed.G, B: parsed.B}, nil
}

func scale(color Color, percent int) Color {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return Color{
		R: uint8(int(color.R) * percent / 100),
		G: uint8(int(color.G) * percent / 100),
		B: uint8(int(color.B) * percent / 100),
	}
}

func fill(colors []Color, color Color) {
	for i := range colors {
		colors[i] = color
	}
}

func HSLToRGB(hue, saturation, brightness float64) Color {
	hue = math.Mod(hue, 360)
	if hue < 0 {
		hue += 360
	}
	hi := int(hue/60) % 6
	f := hue/60 - float64(hi)
	p := brightness * (1 - saturation)
	q := brightness * (1 - f*saturation)
	t := brightness * (1 - (1-f)*saturation)

	var r, g, b float64
	switch hi {
	case 0:
		r, g, b = brightness, t, p
	case 1:
		r, g, b = q, brightness, p
	case 2:
		r, g, b = p, brightness, t
	case 3:
		r, g, b = p, q, brightness
	case 4:
		r, g, b = t, p, brightness
	default:
		r, g, b = brightness, p, q
	}
	return Color{R: uint8(r * 255), G: uint8(g * 255), B: uint8(b * 255)}
}

func sleep(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func mapDelay(speed int, slow, fast time.Duration) time.Duration {
	if speed < 0 {
		speed = 0
	}
	if speed > 100 {
		speed = 100
	}
	delta := slow - fast
	return slow - time.Duration(int64(delta)*int64(speed)/100)
}
