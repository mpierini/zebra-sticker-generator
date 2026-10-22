// Command zsticker converts an image into ASCII art and emits it as a ZPL
// label for Zebra printers.
//
// The image is read from -f or stdin and may be raw image bytes, base64, or a
// data URL.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/mpierini/zebra-sticker-generator/internal/ascii"
	"github.com/mpierini/zebra-sticker-generator/internal/imagein"
	"github.com/mpierini/zebra-sticker-generator/internal/zpl"
)

const usage = `zsticker converts an image into ASCII art and outputs it as a ZPL label
for Zebra printers.

Usage:
  zsticker [flags] -f image.png
  base64 -i image.png | zsticker [flags]

Input may be a PNG, JPEG or GIF file, base64-encoded image data, or a data URL
(data:image/png;base64,...). ZPL is written to stdout unless -o is given.

Examples:
  zsticker -f cat.png -o cat.zpl
  zsticker -f cat.png -ascii -invert > cat.zpl
  zsticker -f cat.png -contrast 1.5 -o cat.zpl
  zsticker -f cat.png -width-in 2 -height-in 1 -o small.zpl

Preview the ZPL at https://labelary.com/viewer.html

Flags:
`

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "zsticker:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("zsticker", flag.ContinueOnError)
	flags.SetOutput(stderr)
	inputPath := flags.String("f", "", "input image file (default: read stdin)")
	outputPath := flags.String("o", "", "output ZPL file (default: stdout)")
	cols := flags.Int("cols", 0, "characters per row (0: fit label width)")
	widthInches := flags.Float64("width-in", 4, "label width in inches")
	heightInches := flags.Float64("height-in", 6, "label height in inches")
	dpi := flags.Int("dpi", 203, "printer resolution in dots per inch")
	invert := flags.Bool("invert", false, "swap light and dark")
	autoLevel := flags.Bool("autolevel", true, "stretch the image's tones to use the full ramp (-autolevel=false to disable)")
	contrast := flags.Float64("contrast", 1, "contrast multiplier; >1 increases contrast, <1 reduces it")
	ramp := flags.String("ramp", ascii.DefaultRamp, "characters from lightest to darkest")
	preview := flags.Bool("ascii", false, "also print the ASCII art to stderr")
	flags.Usage = func() {
		fmt.Fprint(flags.Output(), usage)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	var data []byte
	var err error
	if *inputPath != "" {
		data, err = os.ReadFile(*inputPath)
	} else {
		data, err = io.ReadAll(stdin)
	}
	if err != nil {
		return err
	}

	img, err := imagein.Decode(data)
	if err != nil {
		return err
	}

	label := zpl.NewLabel(*widthInches, *heightInches, *dpi)
	if label.MaxCols() < 1 || label.MaxRows() < 1 {
		return fmt.Errorf("label %.2fx%.2f in at %d dpi is too small", *widthInches, *heightInches, *dpi)
	}
	if *cols <= 0 || *cols > label.MaxCols() {
		*cols = label.MaxCols()
	}

	lines := ascii.Convert(img, ascii.Options{
		Cols:       *cols,
		MaxRows:    label.MaxRows(),
		CellAspect: label.CellAspect(),
		Ramp:       *ramp,
		Invert:     *invert,
		AutoLevel:  *autoLevel,
		Contrast:   *contrast,
	})

	if *preview {
		for _, line := range lines {
			fmt.Fprintln(stderr, line)
		}
	}

	document := zpl.Render(lines, label)
	if *outputPath != "" {
		return os.WriteFile(*outputPath, []byte(document), 0o644)
	}
	_, err = io.WriteString(stdout, document)
	return err
}
