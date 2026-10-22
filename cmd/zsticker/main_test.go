package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestRunBase64Stdin(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 40, 40))
	for row := 0; row < 40; row++ {
		for col := 0; col < 40; col++ {
			img.SetGray(col, row, color.Gray{Y: uint8(col * 6)})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	stdin := strings.NewReader(base64.StdEncoding.EncodeToString(encoded.Bytes()))

	var stdout, stderr bytes.Buffer
	if err := run([]string{"-ascii"}, stdin, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v", err)
	}
	output := stdout.String()
	if !strings.HasPrefix(output, "^XA") || !strings.HasSuffix(output, "^XZ\n") {
		t.Errorf("output is not a ZPL document:\n%s", output)
	}
	if stderr.Len() == 0 {
		t.Error("-ascii wrote no preview to stderr")
	}
}

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"-h"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("run -h: %v", err)
	}
	if !strings.Contains(stderr.String(), "Usage:") || !strings.Contains(stderr.String(), "-width-in") {
		t.Errorf("help output missing usage or flags:\n%s", stderr.String())
	}
}

func TestRunEmptyInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run(nil, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Error("expected error for empty input")
	}
}
