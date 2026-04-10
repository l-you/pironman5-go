package imageconv

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/l-you/pironman5-go/internal/oled"
	"github.com/l-you/pironman5-go/internal/pbm"
	xdraw "golang.org/x/image/draw"
)

func ConvertToPBM(inputPath, outputPath string) error {
	src, err := decodeImage(inputPath)
	if err != nil {
		return err
	}
	converted := Convert(src, oled.Width, oled.Height)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := pbm.EncodeFile(outputPath, converted); err != nil {
		return err
	}
	return nil
}

func Convert(src image.Image, width, height int) *image.Gray {
	dst := image.NewGray(image.Rect(0, 0, width, height))
	bounds := src.Bounds()
	if bounds.Dx() < 1 || bounds.Dy() < 1 {
		return dst
	}
	scale := math.Min(float64(width)/float64(bounds.Dx()), float64(height)/float64(bounds.Dy()))
	scaledWidth := max(1, int(math.Round(float64(bounds.Dx())*scale)))
	scaledHeight := max(1, int(math.Round(float64(bounds.Dy())*scale)))
	resized := image.NewRGBA(image.Rect(0, 0, scaledWidth, scaledHeight))
	xdraw.CatmullRom.Scale(resized, resized.Bounds(), src, bounds, xdraw.Over, nil)
	x0 := (width - scaledWidth) / 2
	y0 := (height - scaledHeight) / 2
	for y := 0; y < scaledHeight; y++ {
		for x := 0; x < scaledWidth; x++ {
			if !isWhitePixel(resized.At(x, y), x, y) {
				continue
			}
			dst.SetGray(x0+x, y0+y, color.Gray{Y: 255})
		}
	}
	return dst
}

func decodeImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	return img, nil
}

func isWhitePixel(c color.Color, x, y int) bool {
	r, g, b, a := c.RGBA()
	if a == 0 {
		return false
	}
	luma := float64(299*r+587*g+114*b) / 1000
	luma *= float64(a) / 0xffff
	bayer := [4][4]float64{
		{0, 8, 2, 10},
		{12, 4, 14, 6},
		{3, 11, 1, 9},
		{15, 7, 13, 5},
	}
	threshold := 0xffff * (0.35 + bayer[y%4][x%4]/16*0.3)
	return luma >= threshold
}
