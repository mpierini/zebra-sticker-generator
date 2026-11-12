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

// maxPixels caps the decoded image size. A small compressed file can declare
// enormous dimensions, and decoding allocates memory for every pixel, so the
// header is checked before decoding (see the security considerations in the
// image package docs). 50 megapixels covers high-end phone and camera photos
// while bounding decoding to roughly 200 MB (4 bytes per RGBA pixel); the
// ASCII art itself is at most a few hundred characters wide.
const maxPixels = 50_000_000

var (
	errEmpty    = errors.New("imagein: no image data")
	errTooLarge = errors.New("imagein: image is too large")
)

// Decode detects the encoding of data and decodes it into an image.
func Decode(data []byte) (image.Image, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, errEmpty
	}

	raw := data
	// If the bytes aren't a recognizable image header, treat them as base64.
	if _, _, err := image.DecodeConfig(bytes.NewReader(data)); err != nil {
		raw, err = decodeBase64(string(data))
		if err != nil {
			return nil, fmt.Errorf("imagein: input is neither a supported image nor base64: %w", err)
		}
	}
	return decodeLimited(raw)
}

// decodeLimited reads only the image header to check its dimensions, and
// decodes the pixels only if the image is within maxPixels.
func decodeLimited(raw []byte) (image.Image, error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("imagein: read image header: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 {
		return nil, fmt.Errorf("imagein: invalid image dimensions %dx%d", config.Width, config.Height)
	}
	// int64 so huge declared dimensions can't overflow the multiplication.
	if pixels := int64(config.Width) * int64(config.Height); pixels > maxPixels {
		return nil, fmt.Errorf("%w: %dx%d is %d pixels, limit is %d",
			errTooLarge, config.Width, config.Height, pixels, maxPixels)
	}

	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("imagein: decode image: %w", err)
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
