package imagein

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func samplePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 7, 3))
	img.Set(1, 1, color.Black)
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestDecode(t *testing.T) {
	raw := samplePNG(t)
	encoded := base64.StdEncoding.EncodeToString(raw)
	wrapped := encoded[:10] + "\n" + encoded[10:] + "\n"

	cases := map[string][]byte{
		"raw":        raw,
		"base64":     []byte(encoded),
		"unpadded":   []byte(base64.RawStdEncoding.EncodeToString(raw)),
		"wrapped":    []byte(wrapped),
		"data URL":   []byte("data:image/png;base64," + encoded),
		"whitespace": []byte("  " + encoded + "  "),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			img, err := Decode(input)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if got := img.Bounds().Size(); got != image.Pt(7, 3) {
				t.Errorf("size = %v, want 7x3", got)
			}
		})
	}
}

func TestDecodeErrors(t *testing.T) {
	cases := map[string]string{
		"empty":            "   ",
		"garbage":          "not an image!",
		"base64 non-image": base64.StdEncoding.EncodeToString([]byte("hello world")),
		"data URL no b64":  "data:image/png,abc",
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Decode([]byte(input)); err == nil {
				t.Error("expected error")
			}
		})
	}
	if _, err := Decode([]byte(strings.Repeat(" ", 3))); err != errEmpty {
		t.Errorf("empty input error = %v, want errEmpty", err)
	}
}

// pngHeaderOnly returns a PNG containing just the signature and an IHDR
// chunk declaring the given size. It is a few dozen bytes, yet claims
// dimensions that would need gigabytes to decode, which is exactly the case
// the size limit guards against.
//
// A PNG file is an 8-byte signature followed by "chunks", each laid out as
// a 4-byte data length, a 4-byte type name, the data, and a CRC-32 checksum
// of the type and data. IHDR ("image header") must be the first chunk; it
// holds the width, height, bit depth, color type, and compression, filter
// and interlace methods. image.DecodeConfig only needs IHDR to report the
// dimensions, so no pixel data (IDAT chunks) is required.
func pngHeaderOnly(width, height uint32) []byte {
	var header bytes.Buffer
	header.WriteString("\x89PNG\r\n\x1a\n")
	chunk := []byte("IHDR")
	chunk = binary.BigEndian.AppendUint32(chunk, width)
	chunk = binary.BigEndian.AppendUint32(chunk, height)
	// Bit depth 8, grayscale, default compression, filter and interlace.
	chunk = append(chunk, 8, 0, 0, 0, 0)
	binary.Write(&header, binary.BigEndian, uint32(len(chunk)-4))
	header.Write(chunk)
	binary.Write(&header, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return header.Bytes()
}

func TestDecodeRejectsOversizedImages(t *testing.T) {
	huge := pngHeaderOnly(10_000, 10_000)
	cases := map[string][]byte{
		"raw":      huge,
		"base64":   []byte(base64.StdEncoding.EncodeToString(huge)),
		"data URL": []byte("data:image/png;base64," + base64.StdEncoding.EncodeToString(huge)),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Decode(input)
			if !errors.Is(err, errTooLarge) {
				t.Errorf("error = %v, want errTooLarge", err)
			}
		})
	}
}
