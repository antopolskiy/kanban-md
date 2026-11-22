package attachment

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
	"testing"
)

func encodedImage(t *testing.T, format string) []byte {
	t.Helper()
	pixels := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	pixels.Set(1, 1, color.NRGBA{R: 255, A: 255})
	var buf bytes.Buffer
	var err error
	switch format {
	case formatPNG:
		err = png.Encode(&buf, pixels)
	case formatJPEG:
		err = jpeg.Encode(&buf, pixels, nil)
	case formatGIF:
		err = gif.Encode(&buf, pixels, nil)
	default:
		t.Fatalf("unknown test image format %q", format)
	}
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestReadSupportedFormats(t *testing.T) {
	for _, format := range []string{formatPNG, formatJPEG, formatGIF} {
		t.Run(format, func(t *testing.T) {
			data := encodedImage(t, format)
			img, err := Read(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if img.Format != format || !bytes.Equal(img.Data, data) {
				t.Fatalf("image changed: format %q, bytes equal %v", img.Format, bytes.Equal(img.Data, data))
			}
		})
	}
}

func TestReadInvalidImages(t *testing.T) {
	pngData := encodedImage(t, formatPNG)
	jpegData := encodedImage(t, formatJPEG)
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"text", []byte("this is not an image")},
		{"svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)},
		{"webp", []byte("RIFF0000WEBPVP8 ")},
		{"png header only", pngData[:33]},
		{"png missing end", pngData[:len(pngData)-12]},
		{"jpeg missing pixels", jpegData[:len(jpegData)-20]},
		{"gif missing trailer", gifBytes(1, 1, 1, false)},
		{"gif missing frames", gifBytes(1, 1, 0, true)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img, err := Read(bytes.NewReader(tt.data))
			if err == nil || len(img.Data) != 0 || img.Format != "" {
				t.Fatalf("Read returned %+v, %v; want zero image and error", img, err)
			}
		})
	}
}

type endlessReader struct{ count int }

func (r *endlessReader) Read(p []byte) (int, error) {
	clear(p)
	r.count += len(p)
	return len(p), nil
}

func TestReadByteLimit(t *testing.T) {
	r := &endlessReader{}
	if _, err := Read(r); err == nil || !strings.Contains(err.Error(), "byte limit") {
		t.Fatalf("Read oversized input: %v", err)
	}
	if r.count != MaxBytes+1 {
		t.Fatalf("read %d bytes, want %d", r.count, MaxBytes+1)
	}
	data := make([]byte, MaxBytes)
	copy(data, encodedImage(t, formatPNG))
	img, err := Read(bytes.NewReader(data))
	if err != nil || !bytes.Equal(img.Data, data) {
		t.Fatalf("image exactly at limit was not preserved: %v", err)
	}
}

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

func TestReadReaderErrors(t *testing.T) {
	want := errors.New("source unavailable")
	r := io.MultiReader(bytes.NewReader(encodedImage(t, formatPNG)), failingReader{err: want})
	img, err := Read(r)
	if !errors.Is(err, want) || img.Data != nil || img.Format != "" {
		t.Fatalf("Read failed source: %+v, %v", img, err)
	}
	if _, err := Read(nil); err == nil {
		t.Fatal("Read accepted nil reader")
	}
}

func pngDimensions(t *testing.T, width, height uint32) []byte {
	t.Helper()
	data := encodedImage(t, formatPNG)
	binary.BigEndian.PutUint32(data[16:20], width)
	binary.BigEndian.PutUint32(data[20:24], height)
	binary.BigEndian.PutUint32(data[29:33], crc32.ChecksumIEEE(data[12:29]))
	return data
}

func TestReadDimensionLimits(t *testing.T) {
	tests := []struct {
		name          string
		width, height uint32
	}{
		{"wide", maxDimension + 1, 1},
		{"tall", 1, maxDimension + 1},
		{"too many pixels", 8192, 8192},
		{"zero width", 0, 1},
		{"zero height", 1, 0},
		{"integer boundary", 1 << 31, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Read(bytes.NewReader(pngDimensions(t, tt.width, tt.height))); err == nil {
				t.Fatal("Read accepted invalid dimensions")
			}
		})
	}
	if err := validateDimensions(maxDimension, 1); err != nil {
		t.Fatalf("maximum side should be allowed: %v", err)
	}
	if err := validateDimensions(8192, 4096); err != nil {
		t.Fatalf("maximum pixel count should be allowed: %v", err)
	}
}

// gifBytes builds a tiny GIF whose descriptors can advertise allocation bombs
// without allocating those pixel buffers in the test itself.
func gifBytes(width, height uint16, frames int, trailer bool) []byte {
	data := []byte("GIF89a")
	data = binary.LittleEndian.AppendUint16(data, width)
	data = binary.LittleEndian.AppendUint16(data, height)
	data = append(data, 0x80, 0, 0, 0, 0, 0, 255, 255, 255)
	for range frames {
		data = append(data, gifImage, 0, 0, 0, 0)
		data = binary.LittleEndian.AppendUint16(data, width)
		data = binary.LittleEndian.AppendUint16(data, height)
		data = append(data, 0, 2, 2, 0x44, 0x01, 0)
	}
	if trailer {
		data = append(data, gifTrailer)
	}
	return data
}

func TestReadGIFBounds(t *testing.T) {
	oversizedFrame := gifBytes(1, 1, 1, true)
	const firstFrameWidthOffset = 24
	binary.LittleEndian.PutUint16(oversizedFrame[firstFrameWidthOffset:], maxDimension+1)
	tests := []struct {
		name string
		data []byte
	}{
		{"screen dimensions", gifBytes(maxDimension+1, 1, 1, true)},
		{"frame dimensions", oversizedFrame},
		{"frame count", gifBytes(1, 1, maxGIFFrames+1, true)},
		{"total pixels", gifBytes(1024, 1024, 33, true)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Read(bytes.NewReader(tt.data)); err == nil || !strings.Contains(err.Error(), "limit") {
				t.Fatalf("Read allocation bomb: %v", err)
			}
		})
	}
	data := gifBytes(1, 1, 2, true)
	if img, err := Read(bytes.NewReader(data)); err != nil || !bytes.Equal(img.Data, data) {
		t.Fatalf("Read valid animated GIF: %v", err)
	}
	// Corrupt the final frame while retaining its dimensions and the trailer.
	data[len(data)-5] = 0xff
	if _, err := Read(bytes.NewReader(data)); err == nil {
		t.Fatal("Read accepted corrupt later GIF frame")
	}
}

func TestReadAnimatedGIFWithExtensionsAndLocalPalette(t *testing.T) {
	first := image.NewPaletted(image.Rect(0, 0, 1, 1), color.Palette{color.Black, color.White})
	second := image.NewPaletted(image.Rect(0, 0, 1, 1), color.Palette{color.Black, color.RGBA{R: 255, A: 255}})
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, &gif.GIF{
		Image: []*image.Paletted{first, second},
		Delay: []int{1, 1},
	}); err != nil {
		t.Fatal(err)
	}
	img, err := Read(bytes.NewReader(buf.Bytes()))
	if err != nil || img.Format != formatGIF || !bytes.Equal(img.Data, buf.Bytes()) {
		t.Fatalf("Read animated GIF: %+v, %v", img, err)
	}
}

func TestReadTruncatedGIFBlocks(t *testing.T) {
	data := gifBytes(1, 1, 2, true)
	for size := range len(data) {
		if _, err := Read(bytes.NewReader(data[:size])); err == nil {
			t.Fatalf("Read accepted GIF truncated to %d of %d bytes", size, len(data))
		}
	}
}
