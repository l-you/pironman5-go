package imageconv

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/l-you/pironman5-go/internal/oled"
	"github.com/l-you/pironman5-go/internal/pbm"
)

func TestConvertToPBM(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "source.jpg")
	outputPath := filepath.Join(t.TempDir(), "oled.pbm")
	src := image.NewRGBA(image.Rect(0, 0, 32, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 32; x++ {
			if x < 24 {
				src.Set(x, y, color.White)
			}
		}
	}
	file, err := os.Create(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(file, src, nil); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ConvertToPBM(inputPath, outputPath); err != nil {
		t.Fatal(err)
	}
	got, err := pbm.DecodeFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds().Dx() != oled.Width || got.Bounds().Dy() != oled.Height {
		t.Fatalf("bounds = %v", got.Bounds())
	}
	white := 0
	for _, pixel := range got.Pix {
		if pixel == 255 {
			white++
		}
	}
	if white == 0 {
		t.Fatal("converted image rendered no white pixels")
	}
}
