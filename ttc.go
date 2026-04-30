package gofont

import (
	"errors"
	"fmt"
)

// ParseTTC parses a TTC (TrueType Collection) file and returns all fonts within it.
func ParseTTC(data []byte) ([]TrueTypeFont, error) {
	if len(data) < 12 {
		return nil, errors.New("TTC file too small")
	}

	if string(data[0:4]) != "ttcf" {
		return nil, errors.New("invalid TTC tag")
	}

	binary := BinaryFrom(data[4:], false)

	version := binary.U32()
	if version != 0x00010000 && version != 0x00020000 {
		return nil, fmt.Errorf("unsupported TTC version: 0x%08X", version)
	}

	numFonts := binary.U32()
	if numFonts == 0 {
		return nil, errors.New("TTC contains no fonts")
	}

	if 12+int(numFonts)*4 > len(data) {
		return nil, errors.New("TTC offset table out of bounds")
	}

	offsets := make([]uint32, numFonts)
	for i := range offsets {
		offsets[i] = binary.U32()
	}

	fonts := make([]TrueTypeFont, numFonts)
	for i, off := range offsets {
		if int(off) >= len(data) {
			return nil, fmt.Errorf("TTC font %d offset out of bounds", i)
		}
		ttf, err := parseFromOffset(data, off)
		if err != nil {
			return nil, fmt.Errorf("TTC font %d: %w", i, err)
		}
		fonts[i] = ttf
	}

	return fonts, nil
}

// SerializeTTC serializes multiple TrueTypeFont objects into a TTC file.
func SerializeTTC(fonts []TrueTypeFont) ([]byte, error) {
	if len(fonts) == 0 {
		return nil, errors.New("no fonts to serialize")
	}

	// Serialize each font to independent TTF binary data
	ttfDataList := make([][]byte, len(fonts))
	for i, ttf := range fonts {
		data, err := ttf.Serialize()
		if err != nil {
			return nil, fmt.Errorf("font %d: %w", i, err)
		}
		ttfDataList[i] = data
	}

	// TTC header: "ttcf" (4) + version (4) + numFonts (4) + offsets (4×N)
	headerSize := 12 + 4*len(fonts)

	// Calculate offset for each font's TTF data
	// Each TTF data is padded to 4-byte alignment
	offsets := make([]uint32, len(fonts))
	currentOffset := uint32(headerSize)
	for i, ttfData := range ttfDataList {
		offsets[i] = currentOffset
		currentOffset += uint32(len(ttfData))
		// Pad to 4-byte boundary (except last)
		if i < len(ttfDataList)-1 && currentOffset%4 != 0 {
			currentOffset += 4 - currentOffset%4
		}
	}

	totalSize := currentOffset
	ttcData := make([]byte, totalSize)
	binary := BinaryFrom(ttcData, false)

	// TTC header
	binary.Append([]byte("ttcf"))
	binary.PutU32(0x00010000) // version 1.0
	binary.PutU32(uint32(len(fonts)))
	for _, off := range offsets {
		binary.PutU32(off)
	}

	// Write each font's TTF data, adjusting table offsets to be TTC-relative
	for i, ttfData := range ttfDataList {
		fontStart := offsets[i]
		copy(ttcData[fontStart:], ttfData)

		// Read numTables from the font's offset table header
		numTables := uint16(ttfData[4])<<8 | uint16(ttfData[5])

		// Adjust each table directory entry's offset field to be TTC-relative
		// Table directory starts at byte 12 within each TTF
		// Each entry: tag(4) + checksum(4) + offset(4) + length(4) = 16 bytes
		for j := 0; j < int(numTables); j++ {
			entryStart := fontStart + uint32(12+j*16)
			origOffset := uint32(ttcData[entryStart+8])<<24 | uint32(ttcData[entryStart+9])<<16 | uint32(ttcData[entryStart+10])<<8 | uint32(ttcData[entryStart+11])
			newOffset := origOffset + fontStart
			ttcData[entryStart+8] = byte(newOffset >> 24)
			ttcData[entryStart+9] = byte(newOffset >> 16)
			ttcData[entryStart+10] = byte(newOffset >> 8)
			ttcData[entryStart+11] = byte(newOffset)
		}
	}

	return ttcData, nil
}
