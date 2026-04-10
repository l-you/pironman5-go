package config

import (
	"os"
	"path/filepath"
	"reflect"
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
		OLEDPageMode:         OLEDPageModeAuto,
		OLEDPage:             OLEDPagePerformance,
		OLEDImagePath:        "",
		OLEDImagePaths:       nil,
		OLEDImageInterval:    DefaultOLEDImageInterval,
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
	if cfg.System.OLEDPageMode != OLEDPageModeAuto || cfg.System.OLEDPage != OLEDPagePerformance {
		t.Fatalf("oled defaults = %q/%q", cfg.System.OLEDPageMode, cfg.System.OLEDPage)
	}
	if cfg.System.OLEDImageInterval != DefaultOLEDImageInterval {
		t.Fatalf("oled image interval = %d, want %d", cfg.System.OLEDImageInterval, DefaultOLEDImageInterval)
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
	cfg.System.OLEDPageMode = OLEDPageModeFixed
	cfg.System.OLEDPage = OLEDPageHeart
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
	if loaded.System.OLEDPageMode != OLEDPageModeFixed || loaded.System.OLEDPage != OLEDPageHeart {
		t.Fatalf("oled page = %q/%q, want fixed/heart", loaded.System.OLEDPageMode, loaded.System.OLEDPage)
	}
}

func TestNormalizeOLEDPageAliases(t *testing.T) {
	if got := NormalizeOLEDPage("hearth"); got != OLEDPageHeart {
		t.Fatalf("hearth alias = %q, want heart", got)
	}
}

func TestNormalizeMigratesSingleOLEDImagePathToList(t *testing.T) {
	cfg := testDefaults()
	cfg.OLEDImagePath = "/tmp/only.pbm"
	cfg.OLEDImagePaths = nil
	cfg.OLEDImageInterval = 0
	cfg.Normalize()
	if cfg.OLEDImagePath != "/tmp/only.pbm" {
		t.Fatalf("oled image path = %q", cfg.OLEDImagePath)
	}
	if !reflect.DeepEqual(cfg.OLEDImagePaths, []string{"/tmp/only.pbm"}) {
		t.Fatalf("oled image paths = %#v", cfg.OLEDImagePaths)
	}
	if cfg.OLEDImageInterval != DefaultOLEDImageInterval {
		t.Fatalf("oled image interval = %d, want %d", cfg.OLEDImageInterval, DefaultOLEDImageInterval)
	}
}

func TestParseOLEDImagePaths(t *testing.T) {
	got := ParseOLEDImagePaths(" /tmp/a.pbm, ,/tmp/b.pbm ")
	want := []string{"/tmp/a.pbm", "/tmp/b.pbm"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %#v, want %#v", got, want)
	}
}

func TestValidateRejectsInvalidOLEDPage(t *testing.T) {
	cfg := testDefaults()
	cfg.OLEDPage = "missing"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid oled page error")
	}
}

func TestValidateRejectsInvalidOLEDImageInterval(t *testing.T) {
	cfg := testDefaults()
	cfg.OLEDImageInterval = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid oled image interval error")
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
