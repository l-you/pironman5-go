package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/l-you/pironman5-go/internal/config"
)

func TestParseOptionalFlags(t *testing.T) {
	opts, err := Parse([]string{"-rb", "35", "-re", "off", "-om", "fixed", "-op", "heart", "start", "--background"})
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
	if !opts.OLEDPage.HasValue || opts.OLEDPage.Value != "heart" {
		t.Fatalf("oled page = %#v", opts.OLEDPage)
	}
}

func TestRunAppliesConfigFlags(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	var out bytes.Buffer
	err := Run(context.Background(), []string{"-cp", configPath, "-rb", "35", "-re", "false", "-rs", "solid", "-om", "fixed", "-op", "hearth"}, &out, io.Discard)
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
	if cfg.System.OLEDPageMode != config.OLEDPageModeFixed || cfg.System.OLEDPage != config.OLEDPageHeart {
		t.Fatalf("unexpected oled config: %#v", cfg.System)
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
