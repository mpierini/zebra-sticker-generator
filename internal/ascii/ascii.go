// Package ascii converts images into ASCII art.
package ascii

import (
	"image"
	"image/color"
	"math"
	"slices"

	"golang.org/x/image/draw"
)

// DefaultRamp is the default ASCII ramp.
//
// An ASCII ramp is a string of characters ordered by how much "ink" each one
// visually covers. Each pixel's brightness is mapped to a position in the
// ramp, so light areas become sparse characters (space, '.') and dark areas
// become dense ones ('#', 'N'). This ramp runs lightest to darkest to match
// black ink on a white label, and deliberately avoids ZPL control characters
// (^ and ~).
//
// The order comes from measuring how many dots each character inks in Zebra
// font A (rendered with Labelary), so darkness rises steadily along the ramp:
// ' '=0 '.'=1 ':'=2 '-'=5 '+'=9 'o'=11 '$'=13 '&'=16 '#'=18 'M'=19 'N'=20.
// Even 'N' covers only about a third of its 6x10 dot cell, so text cannot
// reach solid black; that is the contrast ceiling of ASCII art in this font.
const DefaultRamp = " .:-+o$&#MN"

// Options control the conversion.
type Options struct {
	// Cols is the desired number of characters per row.
	Cols int
	// MaxRows caps the number of rows; 0 means no cap. When the cap is hit,
	// Cols is reduced to preserve the image's aspect ratio.
	MaxRows int
	// CellAspect is a character cell's height divided by its width.
	CellAspect float64
	// Ramp lists characters from lightest to darkest. Empty means DefaultRamp.
	Ramp string
	// Invert swaps light and dark.
	Invert bool
	// AutoLevel stretches the image's tones so its darkest pixels map to the
	// darkest character and its lightest to a space. The extreme 1% at each
	// end is ignored so a few stray pixels can't defeat the stretch.
	AutoLevel bool
	// Contrast scales brightness away from mid-grey after auto-levelling:
	// 1 leaves it unchanged, values above 1 push tones toward black or white,
	// and values below 1 flatten them. 0 means 1.
	Contrast float64
}

// levelClip is the fraction of pixels ignored at each end by AutoLevel.
const levelClip = 0.01

// Convert renders img as rows of ASCII characters.
func Convert(img image.Image, opts Options) []string {
	ramp := []rune(opts.Ramp)
	if len(ramp) == 0 {
		ramp = []rune(DefaultRamp)
	}
	aspect := opts.CellAspect
	if aspect <= 0 {
		aspect = 2
	}

	cols, rows := gridSize(img.Bounds(), opts.Cols, opts.MaxRows, aspect)

	// Shrink the image so each pixel becomes one character. Compositing onto
	// white first makes transparent pixels paper rather than ink.
	scaled := image.NewRGBA(image.Rect(0, 0, cols, rows))
	draw.Draw(scaled, scaled.Bounds(), image.White, image.Point{}, draw.Src)
	// CatmullRom is a high-quality resampling filter: each output pixel is a
	// weighted blend of a 4x4 neighbourhood of source pixels (a Catmull-Rom
	// cubic spline). Compared with nearest-neighbour, which just picks one
	// pixel, it keeps edges smooth and averages away noise, so the downscaled
	// image, and therefore the ASCII art, better reflects the original's
	// shading. draw.Over blends the source onto the white background.
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), img, img.Bounds(), draw.Over, nil)

	// Brightness of each cell from 0 (black) to 1 (white).
	brightness := make([]float64, cols*rows)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			gray := color.GrayModel.Convert(scaled.At(col, row)).(color.Gray).Y
			brightness[row*cols+col] = float64(gray) / 255
		}
	}
	if opts.AutoLevel {
		autoLevel(brightness)
	}
	contrast := opts.Contrast
	if contrast <= 0 {
		contrast = 1
	}

	lines := make([]string, rows)
	rowChars := make([]rune, cols)
	lastIndex := float64(len(ramp) - 1)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			value := brightness[row*cols+col]
			value = clamp((value-0.5)*contrast + 0.5)
			if opts.Invert {
				value = 1 - value
			}
			rowChars[col] = ramp[int(math.Round((1-value)*lastIndex))]
		}
		lines[row] = string(rowChars)
	}
	return lines
}

// autoLevel linearly stretches values in place so the levelClip percentile
// becomes 0 and the (1 - levelClip) percentile becomes 1. Images with no
// tonal range (e.g. a blank page) are left alone.
func autoLevel(values []float64) {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	clipCount := int(float64(len(sorted)) * levelClip)
	low, high := sorted[clipCount], sorted[len(sorted)-1-clipCount]
	if high-low < 1.0/255 {
		return
	}
	for index, value := range values {
		values[index] = clamp((value - low) / (high - low))
	}
}

func clamp(value float64) float64 {
	return math.Min(math.Max(value, 0), 1)
}

// gridSize picks the character grid for an image, keeping its aspect ratio
// once the taller-than-wide character cells are accounted for.
func gridSize(bounds image.Rectangle, cols, maxRows int, aspect float64) (int, int) {
	width, height := float64(bounds.Dx()), float64(bounds.Dy())
	if cols <= 0 {
		cols = 80
	}
	rows := int(math.Round(float64(cols) * height / width / aspect))
	if maxRows > 0 && rows > maxRows {
		rows = maxRows
		cols = int(math.Round(float64(rows) * aspect * width / height))
	}
	return max(cols, 1), max(rows, 1)
}
