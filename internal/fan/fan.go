package fan

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/l-you/pironman5-go/internal/config"
)

type DigitalOutput interface {
	Set(context.Context, bool) error
	Close() error
}

type PWM interface {
	Ready() bool
	Supported() bool
	State(context.Context) (int, error)
	SetState(context.Context, int) error
	SpeedRPM(context.Context) (int, error)
	Off(context.Context) error
	Close() error
}

type TemperatureReader func(context.Context) (float64, error)

type State struct {
	Level     int
	GPIOState bool
	PWMRPM    int
}

type Service struct {
	gpio      DigitalOutput
	pwm       PWM
	temp      TemperatureReader
	log       *slog.Logger
	mu        sync.RWMutex
	cfg       config.System
	level     int
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	onChanged func(State)
}

func New(gpio DigitalOutput, pwm PWM, temp TemperatureReader, cfg config.System, log *slog.Logger) *Service {
	return &Service{
		gpio:      gpio,
		pwm:       pwm,
		temp:      temp,
		cfg:       cfg,
		log:       log.With("service", "fan"),
		onChanged: func(State) {},
	}
}

func (s *Service) SetOnChanged(fn func(State)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if fn == nil {
		s.onChanged = func(State) {}
		return
	}
	s.onChanged = fn
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
	_ = s.gpio.Set(context.Background(), false)
	_ = s.pwm.Off(context.Background())
	_ = s.gpio.Close()
	_ = s.pwm.Close()
}

func (s *Service) loop(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(s.interval())
	defer ticker.Stop()
	for {
		s.runOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) runOnce(ctx context.Context) {
	cfg, callback := s.snapshot()
	state := State{Level: s.level}
	if s.pwm.Ready() && s.pwm.Supported() {
		pwmState, err := s.pwm.State(ctx)
		if err != nil {
			s.log.Warn("read pwm fan state", "error", err)
		} else {
			state.Level = clampLevel(pwmState)
			s.level = state.Level
		}
		if rpm, err := s.pwm.SpeedRPM(ctx); err == nil {
			state.PWMRPM = rpm
		}
		state.GPIOState = state.Level >= cfg.GPIOFanMode
		if err := s.gpio.Set(ctx, state.GPIOState); err != nil {
			s.log.Warn("sync gpio fan", "error", err)
		}
		callback(state)
		return
	}

	tempC, err := s.temp(ctx)
	if err != nil {
		s.log.Warn("read cpu temperature", "error", err)
		return
	}
	newLevel, changed, direction := LevelForTemperature(s.level, tempC)
	if changed {
		s.log.Info("set fan level", "level", Levels[newLevel].Name, "temperature_c", tempC, "direction", direction)
	}
	s.level = newLevel
	state.Level = newLevel
	state.GPIOState = newLevel >= cfg.GPIOFanMode
	if err := s.gpio.Set(ctx, state.GPIOState); err != nil {
		s.log.Warn("set gpio fan", "error", err)
	}
	if s.pwm.Ready() && !s.pwm.Supported() {
		if err := s.pwm.SetState(ctx, newLevel); err != nil {
			s.log.Warn("set pwm fan", "error", err)
		}
	}
	callback(state)
}

func (s *Service) snapshot() (config.System, func(State)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg, s.onChanged
}

func (s *Service) interval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Duration(s.cfg.DataInterval) * time.Second
}

type Level struct {
	Name    string
	Low     float64
	High    float64
	Percent int
}

var Levels = []Level{
	{Name: "OFF", Low: -200, High: 55, Percent: 0},
	{Name: "LOW", Low: 45, High: 65, Percent: 40},
	{Name: "MEDIUM", Low: 55, High: 75, Percent: 80},
	{Name: "HIGH", Low: 65, High: 100, Percent: 100},
}

func LevelForTemperature(current int, tempC float64) (int, bool, string) {
	current = clampLevel(current)
	level := current
	direction := ""
	if tempC < Levels[level].Low {
		level--
		direction = "low"
	} else if tempC > Levels[level].High {
		level++
		direction = "high"
	}
	level = clampLevel(level)
	return level, level != current, direction
}

func clampLevel(level int) int {
	if level < 0 {
		return 0
	}
	if level >= len(Levels) {
		return len(Levels) - 1
	}
	return level
}
