package ascii

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"
)

// halfBlack returns an image whose left half is black and right half white.
func halfBlack(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 0, width/2, height), image.Black, image.Point{}, draw.Src)
	return img
}

func TestConvertHalfBlack(t *testing.T) {
	lines := Convert(halfBlack(200, 100), Options{Cols: 20, CellAspect: 2})
	if len(lines) != 5 {
		t.Fatalf("rows = %d, want 5", len(lines))
	}
	for _, line := range lines {
		if len(line) != 20 {
			t.Fatalf("cols = %d, want 20", len(line))
		}
		if line[:8] != strings.Repeat("N", 8) || line[12:] != strings.Repeat(" ", 8) {
			t.Errorf("line = %q, want N on the left and spaces on the right", line)
		}
	}
}

func TestConvertInvert(t *testing.T) {
	lines := Convert(halfBlack(200, 100), Options{Cols: 20, CellAspect: 2, Invert: true})
	if line := lines[0]; line[:8] != strings.Repeat(" ", 8) || line[12:] != strings.Repeat("N", 8) {
		t.Errorf("line = %q, want spaces on the left and N on the right", line)
	}
}

// grayRamp returns a horizontal gradient from darkest to lightest.
func grayRamp(width, height int, darkest, lightest uint8) image.Image {
	img := image.NewGray(image.Rect(0, 0, width, height))
	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			step := int(lightest-darkest) * col / (width - 1)
			img.SetGray(col, row, color.Gray{Y: darkest + uint8(step)})
		}
	}
	return img
}

func TestConvertAutoLevelUsesFullRamp(t *testing.T) {
	// A dull image spanning only mid-greys.
	dull := grayRamp(110, 10, 100, 160)
	opts := Options{Cols: 11, CellAspect: 2}

	flat := Convert(dull, opts)[0]
	if strings.ContainsAny(flat, " N") {
		t.Fatalf("without auto-level, dull image should avoid the ramp ends: %q", flat)
	}

	opts.AutoLevel = true
	leveled := Convert(dull, opts)[0]
	if leveled[0] != 'N' || leveled[len(leveled)-1] != ' ' {
		t.Errorf("with auto-level, want N at the dark end and space at the light end: %q", leveled)
	}
}

func TestConvertAutoLevelBlankImage(t *testing.T) {
	blank := grayRamp(10, 10, 128, 128)
	lines := Convert(blank, Options{Cols: 5, CellAspect: 1, AutoLevel: true})
	if strings.ContainsAny(lines[0], " N") {
		t.Errorf("flat grey should stay mid-ramp, got %q", lines[0])
	}
}

func TestConvertContrast(t *testing.T) {
	dull := grayRamp(110, 10, 100, 160)
	normal := Convert(dull, Options{Cols: 11, CellAspect: 2})[0]
	boosted := Convert(dull, Options{Cols: 11, CellAspect: 2, Contrast: 3})[0]
	ramp := DefaultRamp
	spread := func(line string) int {
		return strings.IndexByte(ramp, line[0]) - strings.IndexByte(ramp, line[len(line)-1])
	}
	if spread(boosted) <= spread(normal) {
		t.Errorf("contrast 3 should widen the tonal spread: normal %q, boosted %q", normal, boosted)
	}
}

func TestConvertTransparentIsWhite(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img.Set(0, 0, color.Transparent)
	for _, line := range Convert(img, Options{Cols: 5, CellAspect: 1}) {
		if strings.TrimSpace(line) != "" {
			t.Errorf("line = %q, want blank", line)
		}
	}
}

func TestConvertMaxRowsKeepsAspect(t *testing.T) {
	// A tall image that would need 50 rows at 20 cols; capped to 10 rows,
	// cols should shrink to 4 to keep the shape.
	lines := Convert(halfBlack(100, 500), Options{Cols: 20, MaxRows: 10, CellAspect: 2})
	if len(lines) != 10 || len(lines[0]) != 4 {
		t.Errorf("grid = %dx%d, want 4x10", len(lines[0]), len(lines))
	}
}
