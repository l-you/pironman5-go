package fan

import "testing"

func TestLevelForTemperatureUsesUpstreamHysteresis(t *testing.T) {
	tests := []struct {
		name      string
		current   int
		temp      float64
		wantLevel int
		changed   bool
	}{
		{name: "off stays below high", current: 0, temp: 50, wantLevel: 0, changed: false},
		{name: "off increases above high", current: 0, temp: 56, wantLevel: 1, changed: true},
		{name: "low holds inside band", current: 1, temp: 50, wantLevel: 1, changed: false},
		{name: "medium decreases below low", current: 2, temp: 54, wantLevel: 1, changed: true},
		{name: "high clamps", current: 3, temp: 120, wantLevel: 3, changed: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, changed, _ := LevelForTemperature(tt.current, tt.temp)
			if level != tt.wantLevel || changed != tt.changed {
				t.Fatalf("LevelForTemperature(%d, %.1f) = (%d, %t), want (%d, %t)", tt.current, tt.temp, level, changed, tt.wantLevel, tt.changed)
			}
		})
	}
}
