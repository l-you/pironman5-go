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

const (
	animationFrameInterval = 20 * time.Millisecond
	idleFrameInterval      = time.Second
	errorBackoffInterval   = 5 * time.Second
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
	started := time.Now()
	for {
		cfg := s.snapshot()
		pattern, delay := Generate(cfg, time.Since(started))
		if err := s.strip.Show(ctx, pattern); err != nil {
			s.log.Warn("show rgb frame", "error", err)
			delay = errorBackoffInterval
		}
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

func Generate(cfg config.System, elapsed time.Duration) ([]Color, time.Duration) {
	if cfg.RGBLEDCount < 1 {
		return nil, idleFrameInterval
	}
	colors := make([]Color, cfg.RGBLEDCount)
	if !cfg.RGBEnable {
		return colors, idleFrameInterval
	}
	base, err := parseBaseColor(cfg)
	if err != nil {
		return colors, idleFrameInterval
	}

	switch cfg.RGBStyle {
	case "solid":
		fill(colors, scale(base, cfg.RGBBrightness))
		return colors, idleFrameInterval
	case "breathing":
		cycle := mapDelay(cfg.RGBSpeed, 20*time.Second, 200*time.Millisecond)
		brightness := (1 - math.Cos(2*math.Pi*phase(elapsed, cycle))) / 2
		fill(colors, scaleRatio(base, float64(cfg.RGBBrightness)/100*brightness))
		return colors, animationFrameInterval
	case "flow", "flow_reverse":
		step := mapDelay(cfg.RGBSpeed, 500*time.Millisecond, 100*time.Millisecond)
		index := int(elapsed/step) % cfg.RGBLEDCount
		if cfg.RGBStyle == "flow_reverse" {
			index = cfg.RGBLEDCount - 1 - index
		}
		colors[index] = scale(base, cfg.RGBBrightness)
		return colors, step
	case "rainbow", "rainbow_reverse":
		reverse := cfg.RGBStyle == "rainbow_reverse"
		hueOffset := 360 * phase(elapsed, mapDelay(cfg.RGBSpeed, 36*time.Second, 1800*time.Millisecond))
		for i := range colors {
			idx := i
			if reverse {
				idx = cfg.RGBLEDCount - 1 - i
			}
			hue := float64(idx)*360/float64(cfg.RGBLEDCount) + hueOffset
			colors[i] = HSLToRGB(hue, 1, float64(cfg.RGBBrightness)/100)
		}
		return colors, animationFrameInterval
	case "hue_cycle":
		hue := 360 * phase(elapsed, mapDelay(cfg.RGBSpeed, 36*time.Second, 1800*time.Millisecond))
		fill(colors, HSLToRGB(hue, 1, float64(cfg.RGBBrightness)/100))
		return colors, animationFrameInterval
	default:
		fill(colors, scale(base, cfg.RGBBrightness))
		return colors, idleFrameInterval
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
	return scaleRatio(color, float64(clampInt(percent, 0, 100))/100)
}

func scaleRatio(color Color, ratio float64) Color {
	ratio = math.Max(0, math.Min(1, ratio))
	return Color{
		R: uint8(math.Round(float64(color.R) * ratio)),
		G: uint8(math.Round(float64(color.G) * ratio)),
		B: uint8(math.Round(float64(color.B) * ratio)),
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
	return Color{R: floatToByte(r), G: floatToByte(g), B: floatToByte(b)}
}

func floatToByte(value float64) uint8 {
	value = math.Max(0, math.Min(1, value))
	return uint8(math.Round(value * 255))
}

func phase(elapsed, period time.Duration) float64 {
	if period <= 0 {
		return 0
	}
	return float64(elapsed%period) / float64(period)
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
	speed = clampInt(speed, 0, 100)
	delta := slow - fast
	return slow - time.Duration(int64(delta)*int64(speed)/100)
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
