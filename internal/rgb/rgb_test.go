package rgb

import (
	"testing"
	"time"

	"github.com/l-you/pironman5-go/internal/config"
)

func TestGenerateSolidScalesBrightness(t *testing.T) {
	colors, delay := Generate(config.System{RGBLEDCount: 2, RGBEnable: true, RGBColor: "#804020", RGBBrightness: 50, RGBStyle: "solid"}, 0)
	if delay != idleFrameInterval {
		t.Fatalf("delay = %v, want %v", delay, idleFrameInterval)
	}
	want := Color{R: 64, G: 32, B: 16}
	for _, got := range colors {
		if got != want {
			t.Fatalf("color = %#v, want %#v", got, want)
		}
	}
}

func TestGenerateBreathingKeepsFrameDelayIndependentFromSpeed(t *testing.T) {
	cfg := config.System{RGBLEDCount: 1, RGBEnable: true, RGBColor: "#ffffff", RGBBrightness: 100, RGBStyle: "breathing"}
	cfg.RGBSpeed = 25
	colorsSlow, delaySlow := Generate(cfg, time.Second)
	cfg.RGBSpeed = 75
	colorsFast, delayFast := Generate(cfg, time.Second)
	if delaySlow != animationFrameInterval || delayFast != animationFrameInterval {
		t.Fatalf("delays = %v/%v, want %v", delaySlow, delayFast, animationFrameInterval)
	}
	if colorsSlow[0] == colorsFast[0] {
		t.Fatalf("speed should change phase, got same color %#v", colorsSlow[0])
	}
}

func TestGenerateFlowReverse(t *testing.T) {
	colors, delay := Generate(config.System{RGBLEDCount: 4, RGBEnable: true, RGBColor: "#ffffff", RGBBrightness: 100, RGBStyle: "flow_reverse", RGBSpeed: 50}, 300*time.Millisecond)
	if delay != 300*time.Millisecond {
		t.Fatalf("delay = %v, want 300ms", delay)
	}
	if colors[2] != (Color{R: 255, G: 255, B: 255}) {
		t.Fatalf("expected reverse flow at index 2: %#v", colors)
	}
}

func TestGenerateHueCycleKeepsFrameDelayIndependentFromSpeed(t *testing.T) {
	cfg := config.System{RGBLEDCount: 1, RGBEnable: true, RGBColor: "#ffffff", RGBBrightness: 100, RGBStyle: "hue_cycle"}
	cfg.RGBSpeed = 0
	_, delaySlow := Generate(cfg, time.Second)
	cfg.RGBSpeed = 100
	_, delayFast := Generate(cfg, time.Second)
	if delaySlow != animationFrameInterval || delayFast != animationFrameInterval {
		t.Fatalf("delays = %v/%v, want %v", delaySlow, delayFast, animationFrameInterval)
	}
}

func TestHSLToRGBPrimary(t *testing.T) {
	if got := HSLToRGB(0, 1, 1); got != (Color{R: 255, G: 0, B: 0}) {
		t.Fatalf("red = %#v", got)
	}
}
