package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/l-you/pironman5-go/internal/config"
)

func TestParseOptionalFlags(t *testing.T) {
	opts, err := Parse([]string{"-rb", "35", "-re", "off", "-om", "fixed", "-op", "image", "-oj", "/tmp/a.pbm,/tmp/b.pbm", "-ot", "3", "start", "--background"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Command != "start" || !opts.Background {
		t.Fatalf("unexpected command/background: %#v", opts)
	}
	if !opts.RGBBrightness.HasValue || opts.RGBBrightness.Value != "35" {
		t.Fatalf("rgb brightness = %#v", opts.RGBBrightness)
	}
	if !opts.RGBEnable.HasValue || opts.RGBEnable.Value != "off" {
		t.Fatalf("rgb enable = %#v", opts.RGBEnable)
	}
	if !opts.OLEDPageMode.HasValue || opts.OLEDPageMode.Value != "fixed" {
		t.Fatalf("oled page mode = %#v", opts.OLEDPageMode)
	}
	if !opts.OLEDPage.HasValue || opts.OLEDPage.Value != "image" {
		t.Fatalf("oled page = %#v", opts.OLEDPage)
	}
	if !opts.OLEDImagePath.HasValue || opts.OLEDImagePath.Value != "/tmp/a.pbm,/tmp/b.pbm" {
		t.Fatalf("oled image path = %#v", opts.OLEDImagePath)
	}
	if !opts.OLEDImageInterval.HasValue || opts.OLEDImageInterval.Value != "3" {
		t.Fatalf("oled image interval = %#v", opts.OLEDImageInterval)
	}
}

func TestRunAppliesConfigFlags(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	var out bytes.Buffer
	err := Run(context.Background(), []string{"-cp", configPath, "-rb", "35", "-re", "false", "-rs", "solid", "-om", "fixed", "-op", "image", "-oj", "/tmp/a.pbm,/tmp/b.pbm", "-ot", "3"}, &out, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var cfg config.File
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.System.RGBBrightness != 35 || cfg.System.RGBEnable || cfg.System.RGBStyle != "solid" {
		t.Fatalf("unexpected rgb config: %#v", cfg.System)
	}
	if cfg.System.OLEDPageMode != config.OLEDPageModeFixed || cfg.System.OLEDPage != config.OLEDPageImage {
		t.Fatalf("unexpected oled page config: %#v", cfg.System)
	}
	if cfg.System.OLEDImagePath != "/tmp/a.pbm" {
		t.Fatalf("oled image path = %q", cfg.System.OLEDImagePath)
	}
	if !reflect.DeepEqual(cfg.System.OLEDImagePaths, []string{"/tmp/a.pbm", "/tmp/b.pbm"}) {
		t.Fatalf("oled image paths = %#v", cfg.System.OLEDImagePaths)
	}
	if cfg.System.OLEDImageInterval != 3 {
		t.Fatalf("oled image interval = %d", cfg.System.OLEDImageInterval)
	}
}

func TestRunPrintsConfigPathWithoutCreatingConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	var out bytes.Buffer
	if err := Run(context.Background(), []string{"-cp"}, &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("unexpected config file at %s", configPath)
	}
	if out.String() == "" {
		t.Fatal("expected config path output")
	}
}
