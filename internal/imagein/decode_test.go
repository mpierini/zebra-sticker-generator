package imagein

import (
	"bytes"
	"encoding/base64"
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
	if _, err := Decode([]byte(strings.Repeat(" ", 3))); err != ErrEmpty {
		t.Errorf("empty input error = %v, want ErrEmpty", err)
	}
}
