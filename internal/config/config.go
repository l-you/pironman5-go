package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type File struct {
	System System `json:"system"`
}

type System struct {
	DataInterval         int    `json:"data_interval"`
	DebugLevel           string `json:"debug_level"`
	RGBColor             string `json:"rgb_color"`
	RGBBrightness        int    `json:"rgb_brightness"`
	RGBStyle             string `json:"rgb_style"`
	RGBSpeed             int    `json:"rgb_speed"`
	RGBEnable            bool   `json:"rgb_enable"`
	RGBLEDCount          int    `json:"rgb_led_count"`
	TemperatureUnit      string `json:"temperature_unit"`
	OLEDEnable           bool   `json:"oled_enable"`
	OLEDRotation         int    `json:"oled_rotation"`
	OLEDDisk             string `json:"oled_disk"`
	OLEDNetworkInterface string `json:"oled_network_interface"`
	GPIOFanPin           int    `json:"gpio_fan_pin"`
	GPIOFanMode          int    `json:"gpio_fan_mode"`
}

var (
	DebugLevels = []string{"DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"}
	RGBStyles   = []string{"solid", "breathing", "flow", "flow_reverse", "rainbow", "rainbow_reverse", "hue_cycle"}
)

func LoadOrCreate(path string, defaults System) (File, error) {
	cfg, rewrite, err := Load(path, defaults)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return File{}, err
		}
		cfg = File{System: defaults}
		rewrite = true
	}
	cfg.System.Normalize()
	if err := cfg.System.Validate(); err != nil {
		return File{}, err
	}
	if rewrite {
		if err := Save(path, cfg); err != nil {
			return File{}, err
		}
	}
	return cfg, nil
}

func Load(path string, defaults System) (File, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, false, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return File{System: defaults}, true, nil
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return File{}, false, fmt.Errorf("decode config %s: %w", path, err)
	}

	system := defaults
	rewrite := false
	if raw, ok := root["system"]; ok {
		if err := json.Unmarshal(raw, &system); err != nil {
			return File{}, false, fmt.Errorf("decode system config: %w", err)
		}
	} else if raw, ok := root["auto"]; ok {
		if err := json.Unmarshal(raw, &system); err != nil {
			return File{}, false, fmt.Errorf("decode legacy auto config: %w", err)
		}
		rewrite = true
	} else {
		rewrite = true
	}
	system.Normalize()
	return File{System: system}, rewrite, nil
}

func Save(path string, cfg File) error {
	cfg.System.Normalize()
	if err := cfg.System.Validate(); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".config-*.json")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("encode config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace config %s: %w", path, err)
	}
	return nil
}

func (s *System) Normalize() {
	s.DebugLevel = strings.ToUpper(s.DebugLevel)
	s.TemperatureUnit = strings.ToUpper(s.TemperatureUnit)
	if s.DebugLevel == "" {
		s.DebugLevel = "INFO"
	}
}

func (s System) Validate() error {
	if s.DataInterval < 1 {
		return fmt.Errorf("data_interval must be >= 1")
	}
	if !slices.Contains(DebugLevels, strings.ToUpper(s.DebugLevel)) {
		return fmt.Errorf("debug_level must be one of %v", DebugLevels)
	}
	if _, err := ParseHexColor(s.RGBColor); err != nil {
		return err
	}
	if err := validateRange("rgb_brightness", s.RGBBrightness, 0, 100); err != nil {
		return err
	}
	if err := validateRange("rgb_speed", s.RGBSpeed, 0, 100); err != nil {
		return err
	}
	if s.RGBLEDCount < 1 {
		return fmt.Errorf("rgb_led_count must be >= 1")
	}
	if !slices.Contains(RGBStyles, s.RGBStyle) {
		return fmt.Errorf("rgb_style must be one of %v", RGBStyles)
	}
	if s.TemperatureUnit != "C" && s.TemperatureUnit != "F" {
		return fmt.Errorf("temperature_unit must be C or F")
	}
	if s.OLEDRotation != 0 && s.OLEDRotation != 180 {
		return fmt.Errorf("oled_rotation must be 0 or 180")
	}
	if strings.TrimSpace(s.OLEDDisk) == "" {
		return fmt.Errorf("oled_disk must not be empty")
	}
	if strings.TrimSpace(s.OLEDNetworkInterface) == "" {
		return fmt.Errorf("oled_network_interface must not be empty")
	}
	if s.GPIOFanPin < 0 || s.GPIOFanPin > 40 {
		return fmt.Errorf("gpio_fan_pin must be between 0 and 40")
	}
	if err := validateRange("gpio_fan_mode", s.GPIOFanMode, 0, 4); err != nil {
		return err
	}
	return nil
}

func validateRange(name string, value, min, max int) error {
	if value < min || value > max {
		return fmt.Errorf("%s must be between %d and %d", name, min, max)
	}
	return nil
}

type RGB struct {
	R uint8
	G uint8
	B uint8
}

func ParseHexColor(value string) (RGB, error) {
	hexValue := strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(hexValue) != 6 {
		return RGB{}, fmt.Errorf("rgb_color must be six hex characters, optionally prefixed with #")
	}
	parsed, err := strconv.ParseUint(hexValue, 16, 32)
	if err != nil {
		return RGB{}, fmt.Errorf("rgb_color must be valid hex: %w", err)
	}
	return RGB{R: uint8(parsed >> 16), G: uint8(parsed >> 8), B: uint8(parsed)}, nil
}
