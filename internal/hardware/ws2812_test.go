package hardware

import (
	"testing"

	"github.com/l-you/pironman5-go/internal/rgb"
)

func TestEncodeWS2812GRBLength(t *testing.T) {
	data := EncodeWS2812GRB([]rgb.Color{{R: 0x80, G: 0x00, B: 0x01}})
	if len(data) != 25 {
		t.Fatalf("encoded length = %d, want 25", len(data))
	}
	if data[0] != 0 || data[len(data)-1] != 0 {
		t.Fatalf("expected reset padding around frame")
	}
}
