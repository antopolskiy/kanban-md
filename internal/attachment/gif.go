package attachment

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	gifScreenSize     = 13
	gifDescriptorSize = 9
	gifTableFlag      = 0x80
	gifTableSizeMask  = 0x07
	gifColorBytes     = 3
	gifExtension      = 0x21
	gifImage          = 0x2c
	gifTrailer        = 0x3b
	maxGIFFrames      = 1024
)

// validateGIFBounds walks GIF blocks before DecodeAll allocates frame buffers.
// The decoder still validates compressed pixels and all animation frames.
func validateGIFBounds(data []byte) error {
	r := bytes.NewReader(data)
	var screen [gifScreenSize]byte
	if _, err := io.ReadFull(r, screen[:]); err != nil {
		return err
	}
	const screenPackedOffset = 10
	if err := skipGIFColorTable(r, screen[screenPackedOffset]); err != nil {
		return err
	}
	frames, pixels := 0, 0
	for {
		block, err := r.ReadByte()
		if err != nil {
			return err
		}
		switch block {
		case gifTrailer:
			return nil
		case gifExtension:
			if _, err := r.ReadByte(); err != nil { // Extension label.
				return err
			}
		case gifImage:
			framePixels, err := readGIFFrame(r)
			if err != nil {
				return err
			}
			frames++
			pixels += framePixels
			if frames > maxGIFFrames || pixels > maxPixels {
				return errors.New("GIF animation exceeds frame or total pixel limit")
			}
		default:
			return fmt.Errorf("invalid GIF block 0x%x", block)
		}
		if err := skipGIFBlocks(r); err != nil {
			return err
		}
	}
}

func readGIFFrame(r *bytes.Reader) (int, error) {
	var descriptor [gifDescriptorSize]byte
	if _, err := io.ReadFull(r, descriptor[:]); err != nil {
		return 0, err
	}
	const widthOffset, heightOffset, packedOffset = 4, 6, 8
	width := int(binary.LittleEndian.Uint16(descriptor[widthOffset:heightOffset]))
	height := int(binary.LittleEndian.Uint16(descriptor[heightOffset:packedOffset]))
	if err := validateDimensions(width, height); err != nil {
		return 0, err
	}
	if err := skipGIFColorTable(r, descriptor[packedOffset]); err != nil {
		return 0, err
	}
	if _, err := r.ReadByte(); err != nil { // LZW minimum code size.
		return 0, err
	}
	return width * height, nil
}

func skipGIFColorTable(r *bytes.Reader, packed byte) error {
	if packed&gifTableFlag == 0 {
		return nil
	}
	size := gifColorBytes << ((packed & gifTableSizeMask) + 1)
	_, err := io.CopyN(io.Discard, r, int64(size))
	return err
}

func skipGIFBlocks(r *bytes.Reader) error {
	for {
		size, err := r.ReadByte()
		if err != nil {
			return err
		}
		if size == 0 {
			return nil
		}
		if _, err := io.CopyN(io.Discard, r, int64(size)); err != nil {
			return err
		}
	}
}
