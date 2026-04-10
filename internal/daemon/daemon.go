package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"

	"github.com/l-you/pironman5-go/internal/buildinfo"
	"github.com/l-you/pironman5-go/internal/config"
	"github.com/l-you/pironman5-go/internal/fan"
	"github.com/l-you/pironman5-go/internal/hardware"
	"github.com/l-you/pironman5-go/internal/logging"
	"github.com/l-you/pironman5-go/internal/oled"
	"github.com/l-you/pironman5-go/internal/rgb"
	"github.com/l-you/pironman5-go/internal/status"
	"github.com/l-you/pironman5-go/internal/variant"
)

type Runtime struct {
	cfgPath string
	cfg     config.File
	log     *slog.Logger
	fan     *fan.Service
	rgb     *rgb.Service
	oled    *oled.Service
}

func New(configPath string) (*Runtime, error) {
	v := variant.Standard()
	cfg, err := config.LoadOrCreate(configPath, v.DefaultConfig)
	if err != nil {
		return nil, err
	}
	log := logging.New(buildinfo.AppName, buildinfo.DefaultLogDir, cfg.System.DebugLevel)
	fanRPM := &atomic.Int64{}
	baseSampler := status.SystemSampler{}
	sampler := &fanAwareSampler{base: baseSampler, fanRPM: fanRPM}

	gpioOut, err := hardware.NewGPIODOutput(cfg.System.GPIOFanPin)
	if err != nil {
		log.Warn("gpio fan unavailable; using noop output", "error", err)
	}
	var gpio fan.DigitalOutput = gpioOut
	if gpio == nil {
		gpio = hardware.NoopDigitalOutput{}
	}

	strip, err := hardware.NewSPILEDStrip()
	if err != nil {
		log.Warn("ws2812 unavailable; using noop strip", "error", err)
	}
	var rgbStrip rgb.Strip = strip
	if rgbStrip == nil {
		rgbStrip = hardware.NoopStrip{}
	}

	display, err := hardware.NewSSD1306Display()
	if err != nil {
		log.Warn("oled unavailable; using noop display", "error", err)
	}
	var oledDisplay oled.Display = display
	if oledDisplay == nil {
		oledDisplay = hardware.NoopDisplay{}
	}

	fanService := fan.New(gpio, hardware.NewPWMFan(), baseSampler.CPUTemperatureC, cfg.System, log)
	fanService.SetOnChanged(func(state fan.State) { fanRPM.Store(int64(state.PWMRPM)) })

	return &Runtime{
		cfgPath: configPath,
		cfg:     cfg,
		log:     log,
		fan:     fanService,
		rgb:     rgb.New(rgbStrip, cfg.System, log),
		oled:    oled.New(oledDisplay, sampler, cfg.System, log),
	}, nil
}

func (r *Runtime) Run(ctx context.Context) error {
	r.log.Info("starting pironman5", "config", r.cfgPath)
	if err := writePID(buildinfo.PIDPath, os.Getpid()); err != nil {
		r.log.Warn("write pid file", "error", err)
	}
	defer removePID(buildinfo.PIDPath, os.Getpid())

	r.fan.Start(ctx)
	r.rgb.Start(ctx)
	r.oled.Start(ctx)

	<-ctx.Done()
	r.log.Info("stopping pironman5")
	r.oled.Stop()
	r.rgb.Stop()
	r.fan.Stop()
	r.log.Info("pironman5 stopped")
	return nil
}

func writePID(path string, pid int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create pid directory: %w", err)
	}
	return os.WriteFile(path, []byte(strconv.Itoa(pid)+"\n"), 0o644)
}

func removePID(path string, pid int) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	stored, err := strconv.Atoi(stringTrimSpace(data))
	if err == nil && stored == pid {
		_ = os.Remove(path)
	}
}

func stringTrimSpace(data []byte) string {
	start, end := 0, len(data)
	for start < end && (data[start] == ' ' || data[start] == '\n' || data[start] == '\t' || data[start] == '\r') {
		start++
	}
	for end > start && (data[end-1] == ' ' || data[end-1] == '\n' || data[end-1] == '\t' || data[end-1] == '\r') {
		end--
	}
	return string(data[start:end])
}

type fanAwareSampler struct {
	base   status.SystemSampler
	fanRPM *atomic.Int64
}

func (s *fanAwareSampler) Snapshot(ctx context.Context) (status.Snapshot, error) {
	snap, err := s.base.Snapshot(ctx)
	if err != nil {
		return status.Snapshot{}, err
	}
	snap.FanRPM = int(s.fanRPM.Load())
	return snap, nil
}

func (s *fanAwareSampler) CPUTemperatureC(ctx context.Context) (float64, error) {
	return s.base.CPUTemperatureC(ctx)
}
