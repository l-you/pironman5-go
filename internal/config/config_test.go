package config

import (
	"os"
	"path/filepath"
	"testing"
)

func testDefaults() System {
	return System{
		DataInterval:         1,
		DebugLevel:           "INFO",
		RGBColor:             "#0a1aff",
		RGBBrightness:        50,
		RGBStyle:             "breathing",
		RGBSpeed:             50,
		RGBEnable:            true,
		RGBLEDCount:          4,
		TemperatureUnit:      "C",
		OLEDEnable:           true,
		OLEDRotation:         0,
		OLEDDisk:             "total",
		OLEDNetworkInterface: "all",
		GPIOFanPin:           6,
		GPIOFanMode:          0,
	}
}

func TestLoadMigratesLegacyAuto(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"auto":{"rgb_brightness":35,"debug_level":"debug"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, rewrite, err := Load(path, testDefaults())
	if err != nil {
		t.Fatal(err)
	}
	if !rewrite {
		t.Fatal("expected legacy auto config to request rewrite")
	}
	if cfg.System.RGBBrightness != 35 {
		t.Fatalf("rgb brightness = %d, want 35", cfg.System.RGBBrightness)
	}
	if cfg.System.DebugLevel != "DEBUG" {
		t.Fatalf("debug level = %q, want DEBUG", cfg.System.DebugLevel)
	}
}

func TestSaveAndLoadOrCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	cfg, err := LoadOrCreate(path, testDefaults())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.System.GPIOFanPin != 6 {
		t.Fatalf("gpio fan pin = %d, want 6", cfg.System.GPIOFanPin)
	}
	cfg.System.RGBStyle = "solid"
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := Load(path, testDefaults())
	if err != nil {
		t.Fatal(err)
	}
	if loaded.System.RGBStyle != "solid" {
		t.Fatalf("rgb style = %q, want solid", loaded.System.RGBStyle)
	}
}

func TestParseHexColor(t *testing.T) {
	got, err := ParseHexColor("#0a1aff")
	if err != nil {
		t.Fatal(err)
	}
	if got.R != 10 || got.G != 26 || got.B != 255 {
		t.Fatalf("color = %#v", got)
	}
}
