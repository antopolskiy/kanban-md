// Package attachment validates images and stores task-relative attachments.
package attachment

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/gif"
	_ "image/jpeg" // Register the standard JPEG decoder.
	_ "image/png"  // Register the standard PNG decoder.
	"io"
)

// MaxBytes is the largest encoded image accepted by Read and Import.
const MaxBytes = 20 * 1024 * 1024

const (
	maxDimension = 16 * 1024
	maxPixels    = 32 * 1024 * 1024
	formatPNG    = "png"
	formatJPEG   = "jpeg"
	formatGIF    = "gif"
)

// Image holds the original encoded bytes and their detected format
// (png, jpeg, or gif). Callers must not mutate Data during Import.
type Image struct {
	Data   []byte
	Format string
}

// Read reads at most MaxBytes+1 bytes and validates a PNG, JPEG, or GIF image.
// It bounds dimensions before decoding and preserves the original bytes.
// Images may have at most 16,384 pixels per side and 32 Mi pixels in total;
// GIF animations share that pixel budget across at most 1,024 frames.
func Read(r io.Reader) (Image, error) {
	if r == nil {
		return Image{}, errors.New("image reader is nil")
	}
	data, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return Image{}, fmt.Errorf("reading image: %w", err)
	}
	format, err := validate(data)
	if err != nil {
		return Image{}, err
	}
	return Image{Data: data, Format: format}, nil
}

func validate(data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("image is empty")
	}
	if len(data) > MaxBytes {
		return "", fmt.Errorf("image exceeds %d byte limit", MaxBytes)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("reading image dimensions: %w", err)
	}
	switch format {
	case formatPNG, formatJPEG, formatGIF:
	default:
		return "", fmt.Errorf("unsupported image format %q", format)
	}
	if err = validateDimensions(cfg.Width, cfg.Height); err != nil {
		return "", err
	}
	if format == formatGIF {
		if err = validateGIFBounds(data); err != nil {
			return "", fmt.Errorf("validating GIF bounds: %w", err)
		}
		_, err = gif.DecodeAll(bytes.NewReader(data))
	} else {
		_, _, err = image.Decode(bytes.NewReader(data))
	}
	if err != nil {
		return "", fmt.Errorf("decoding image: %w", err)
	}
	return format, nil
}

func validateDimensions(width, height int) error {
	if width <= 0 || height <= 0 || width > maxDimension || height > maxDimension || width*height > maxPixels {
		return fmt.Errorf("image dimensions %dx%d exceed limits (%d per side, %d pixels)",
			width, height, maxDimension, maxPixels)
	}
	return nil
}
