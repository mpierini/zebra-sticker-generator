// Package imagein turns user-supplied image data into an image.Image.
//
// It accepts raw image bytes (PNG, JPEG, GIF), base64-encoded image bytes,
// or a data URL such as those produced by a browser's FileReader.readAsDataURL.
package imagein

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"
)

// ErrEmpty is returned when no image data was supplied.
var ErrEmpty = errors.New("imagein: no image data")

// Decode detects the encoding of data and decodes it into an image.
func Decode(data []byte) (image.Image, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, ErrEmpty
	}

	if _, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("imagein: decode image: %w", err)
		}
		return img, nil
	}

	raw, err := decodeBase64(string(data))
	if err != nil {
		return nil, fmt.Errorf("imagein: input is neither a supported image nor base64: %w", err)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("imagein: decode base64 image: %w", err)
	}
	return img, nil
}

// decodeBase64 strips an optional data URL prefix and any whitespace, then
// decodes with or without padding.
func decodeBase64(encoded string) ([]byte, error) {
	if strings.HasPrefix(encoded, "data:") {
		commaIndex := strings.Index(encoded, ",")
		if commaIndex < 0 {
			return nil, errors.New("malformed data URL")
		}
		if !strings.HasSuffix(encoded[:commaIndex], ";base64") {
			return nil, errors.New("data URL is not base64-encoded")
		}
		encoded = encoded[commaIndex+1:]
	}
	encoded = strings.Join(strings.Fields(encoded), "")

	if raw, err := base64.StdEncoding.DecodeString(encoded); err == nil {
		return raw, nil
	}
	return base64.RawStdEncoding.DecodeString(strings.TrimRight(encoded, "="))
}
