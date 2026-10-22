// Package zpl renders lines of text as a ZPL II label.
package zpl

import (
	"fmt"
	"math"
	"strings"
)

// Label describes the label and the fixed-width font used to print on it.
// All measurements are in printer dots.
type Label struct {
	WidthDots  int
	HeightDots int
	Margin     int
	// Font is a built-in Zebra font name; it must be fixed-width. It is
	// always printed at its native size.
	Font string
	// Pitch is the horizontal advance per character, including the gap.
	Pitch int
	// LineHeight is the vertical advance per line.
	LineHeight int
}

// NewLabel returns a label of the given size using Zebra's built-in font A
// at 1x magnification.
func NewLabel(widthInches, heightInches float64, dpi int) Label {
	return Label{
		// Printers address the label in dots: inches x dots-per-inch
		// (4 in x 203 dpi = 812 dots, 6 in x 203 dpi = 1218 dots).
		WidthDots:  int(math.Round(widthInches * float64(dpi))),
		HeightDots: int(math.Round(heightInches * float64(dpi))),
		// 16 dots is about 2 mm at 203 dpi: a safe border so nothing prints
		// off the edge if the label is slightly misaligned.
		Margin: 16,
		// Font A is a fixed-width bitmap font, which ASCII art needs so the
		// columns line up. At its native size its glyphs are 5 dots wide and
		// at most 9 dots tall including descenders (measured with Labelary).
		Font: "A",
		// Each glyph is followed by a 1-dot gap, so each character advances
		// 5 + 1 = 6 dots.
		Pitch: 6,
		// 9-dot glyph + 1 dot of spacing so rows don't touch.
		LineHeight: 10,
	}
}

// MaxCols is the number of characters that fit across the printable width.
func (label Label) MaxCols() int { return (label.WidthDots - 2*label.Margin) / label.Pitch }

// MaxRows is the number of lines that fit down the printable height.
func (label Label) MaxRows() int { return (label.HeightDots - 2*label.Margin) / label.LineHeight }

// CellAspect is a character cell's height divided by its width.
func (label Label) CellAspect() float64 {
	return float64(label.LineHeight) / float64(label.Pitch)
}

// Render returns a ZPL document printing lines centered on the label.
func Render(lines []string, label Label) string {
	cols := 0
	for _, line := range lines {
		cols = max(cols, len([]rune(line)))
	}
	originX := max((label.WidthDots-cols*label.Pitch)/2, 0)
	originY := max((label.HeightDots-len(lines)*label.LineHeight)/2, 0)

	var builder strings.Builder
	// ^XA starts the label, ^PW/^LL set width/length, ^CI28 selects UTF-8.
	fmt.Fprintf(&builder, "^XA\n^PW%d\n^LL%d\n^CI28\n", label.WidthDots, label.HeightDots)
	for lineIndex, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		// ^FO positions the field, ^A<font>N selects the font in normal
		// orientation, and ^FD...^FS is the text. The height and width are
		// left out on purpose: passing font A's nominal 9,5 makes Labelary
		// round up to 2x magnification, doubling every character.
		fmt.Fprintf(&builder, "^FO%d,%d^A%sN",
			originX, originY+lineIndex*label.LineHeight, label.Font)
		if strings.ContainsAny(line, "^~_") {
			// ^FH lets control characters be written as _XX hex escapes.
			builder.WriteString("^FH_^FD")
			builder.WriteString(escaper.Replace(line))
		} else {
			builder.WriteString("^FD")
			builder.WriteString(line)
		}
		builder.WriteString("^FS\n")
	}
	builder.WriteString("^XZ\n")
	return builder.String()
}

var escaper = strings.NewReplacer("_", "_5F", "^", "_5E", "~", "_7E")
