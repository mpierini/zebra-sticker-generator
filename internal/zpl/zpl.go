// Package zpl renders lines of text as a ZPL II label.
package zpl

import (
	"fmt"
	"math"
	"strconv"
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
// at 1x magnification. It returns an error if the label is too small to
// hold at least one character inside its margins.
func NewLabel(widthInches, heightInches float64, dpi int) (Label, error) {
	label := Label{
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
	if label.MaxCols() < 1 || label.MaxRows() < 1 {
		return Label{}, fmt.Errorf("zpl: label %.2fx%.2f in at %d dpi is too small", widthInches, heightInches, dpi)
	}
	return label, nil
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
	// The origin is the top-left corner of the art, not the label's center.
	// The label width minus the art width is the empty space beside the art;
	// halving it leaves equal space on the left and right, centering the art.
	// max(..., 0) pins the art to the edge if it is wider than the label.
	// The vertical origin works the same way with the art's height.
	originX := max((label.WidthDots-cols*label.Pitch)/2, 0)
	originY := max((label.HeightDots-len(lines)*label.LineHeight)/2, 0)

	var builder strings.Builder
	// Reserving enough space up front means the whole document is built
	// with a single allocation.
	builder.Grow(estimatedSize(lines))

	// ^XA starts the label, ^PW/^LL set width/length, ^CI28 selects UTF-8.
	builder.WriteString("^XA\n^PW")
	writeInt(&builder, label.WidthDots)
	builder.WriteString("\n^LL")
	writeInt(&builder, label.HeightDots)
	builder.WriteString("\n^CI28\n")
	for lineIndex, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		// ^FO positions the field, ^A<font>N selects the font in normal
		// orientation, and ^FD...^FS is the text. The height and width are
		// left out on purpose: passing font A's nominal 9,5 makes Labelary
		// round up to 2x magnification, doubling every character.
		builder.WriteString("^FO")
		writeInt(&builder, originX)
		builder.WriteByte(',')
		writeInt(&builder, originY+lineIndex*label.LineHeight)
		builder.WriteString("^A")
		builder.WriteString(label.Font)
		builder.WriteByte('N')
		if strings.ContainsAny(line, controlChars) {
			// ^FH lets control characters be written as _XX hex escapes.
			builder.WriteString("^FH_^FD")
			writeEscaped(&builder, line)
		} else {
			builder.WriteString("^FD")
			builder.WriteString(line)
		}
		builder.WriteString("^FS\n")
	}
	builder.WriteString("^XZ\n")
	return builder.String()
}

// The estimate doesn't need to be exact, only big enough that the builder
// never has to reallocate. These constants over-count the ZPL that Render
// writes around the text.
const (
	// documentOverhead covers the ^XA/^PW/^LL/^CI28 header and ^XZ footer
	// (about 35 bytes with 6-digit dimensions).
	documentOverhead = 64
	// lineOverhead covers ^FO<x>,<y>^A<font>N^FH_^FD before each row's text
	// and ^FS\n after it (about 31 bytes with 6-digit coordinates).
	lineOverhead = 40
)

// estimatedSize returns an upper bound on the bytes Render writes for
// ordinary lines. Lines containing ZPL control characters grow when escaped
// (each becomes 3 bytes), which the per-line slack usually absorbs; if not,
// the builder simply makes one extra allocation.
func estimatedSize(lines []string) int {
	size := documentOverhead
	for _, line := range lines {
		size += len(line) + lineOverhead
	}
	return size
}

// writeInt writes value in decimal. Unlike fmt.Fprintf or strconv.Itoa it
// formats into a stack buffer, so it does not allocate.
func writeInt(builder *strings.Builder, value int) {
	var digits [20]byte
	builder.Write(strconv.AppendInt(digits[:0], int64(value), 10))
}

// controlChars are the characters ZPL treats specially inside field data.
const controlChars = "^~_"

// writeEscaped writes line with each control character replaced by its
// ^FH hex escape: _ is 0x5F, ^ is 0x5E and ~ is 0x7E. It writes straight to
// the builder instead of using strings.Replacer, whose io.Writer parameter
// would force the builder onto the heap and cost an extra allocation.
func writeEscaped(builder *strings.Builder, line string) {
	for index := 0; index < len(line); index++ {
		switch char := line[index]; char {
		case '_':
			builder.WriteString("_5F")
		case '^':
			builder.WriteString("_5E")
		case '~':
			builder.WriteString("_7E")
		default:
			builder.WriteByte(char)
		}
	}
}
