package gofont

import (
	"bytes"
	"compress/zlib"
	"errors"
	"io"
)

const woffSignature = 0x774F4646 // "wOFF"

// ParseWOFF parses a WOFF (Web Open Font Format) file and returns a TrueTypeFont.
// It decompresses the WOFF table data and delegates to the standard TTF parser.
func ParseWOFF(data []byte) (TrueTypeFont, error) {
	if len(data) < 44 {
		return TrueTypeFont{}, errors.New("WOFF file too small")
	}

	binary := BinaryFrom(data, false)

	signature := binary.U32()
	if signature != woffSignature {
		return TrueTypeFont{}, errors.New("invalid WOFF signature")
	}

	flavor := binary.U32()       // 0x00010000 (TTF) or 0x4F54544F (OTF)
	_ = binary.U32()             // total WOFF length
	numTables := binary.U16()
	_ = binary.U16()             // reserved
	totalSfntSize := binary.U32() // total uncompressed size
	_ = binary.U16()             // majorVersion
	_ = binary.U16()             // minorVersion
	_ = binary.U32()             // metaOffset
	_ = binary.U32()             // metaLength
	_ = binary.U32()             // metaOrigLength
	_ = binary.U32()             // privOffset
	_ = binary.U32()             // privLength

	// Read WOFF table directory
	type woffEntry struct {
		tag         string
		checksum    uint32
		compData    []byte
		origLength  uint32
	}
	entries := make([]woffEntry, numTables)
	for i := range entries {
		tag := string(binary.Read(4))
		offset := binary.U32()
		compLength := binary.U32()
		origLength := binary.U32()
		origChecksum := binary.U32()

		if int(offset+compLength) > len(data) {
			return TrueTypeFont{}, errors.New("WOFF table data out of bounds")
		}

		compData := data[offset : offset+compLength]

		// Decompress using zlib
		var decompData []byte
		if compLength == origLength {
			// Not compressed (compressed size == original size)
			decompData = make([]byte, len(compData))
			copy(decompData, compData)
		} else {
			r, err := zlib.NewReader(bytes.NewReader(compData))
			if err != nil {
				return TrueTypeFont{}, err
			}
			decompData, err = io.ReadAll(r)
			r.Close()
			if err != nil {
				return TrueTypeFont{}, err
			}
		}

		if uint32(len(decompData)) != origLength {
			return TrueTypeFont{}, errors.New("WOFF decompressed size mismatch")
		}

		entries[i] = woffEntry{
			tag:      tag,
			checksum: origChecksum,
			compData: decompData,
			origLength: origLength,
		}
	}

	// Rebuild TTF byte stream
	searchRange, entrySelector, rangeShift := calcSearchParams(int(numTables))
	headerSize := 12 + int(numTables)*16

	ttfData := make([]byte, totalSfntSize)
	ttfBin := BinaryFrom(ttfData, false)

	// Write TTF header
	ttfBin.PutU32(flavor)
	ttfBin.PutU16(numTables)
	ttfBin.PutU16(searchRange)
	ttfBin.PutU16(uint16(entrySelector))
	ttfBin.PutU16(rangeShift)

	// Calculate table offsets (4-byte aligned)
	offsets := make([]uint32, numTables)
	off := uint32(headerSize)
	for i, e := range entries {
		offsets[i] = off
		paddedLen := e.origLength
		if paddedLen%4 != 0 {
			paddedLen += 4 - paddedLen%4
		}
		off += paddedLen
	}

	// Write TTF table directory
	for i, e := range entries {
		ttfBin.Append([]byte(e.tag))
		ttfBin.PutU32(e.checksum)
		ttfBin.PutU32(offsets[i])
		ttfBin.PutU32(e.origLength)
	}

	// Write TTF table data
	for i, e := range entries {
		copy(ttfData[offsets[i]:], e.compData)
		paddedLen := e.origLength
		if paddedLen%4 != 0 {
			paddedLen += 4 - paddedLen%4
		}
		// Zero-fill padding
		for j := e.origLength; j < paddedLen; j++ {
			ttfData[offsets[i]+j] = 0
		}
	}

	return Parse(ttfData)
}

// SerializeWOFF serializes the font as a WOFF (Web Open Font Format) file.
func (ttf *TrueTypeFont) SerializeWOFF() ([]byte, error) {
	// Get the TTF data first
	ttfData, err := ttf.Serialize()
	if err != nil {
		return nil, err
	}

	if len(ttfData) < 12 {
		return nil, errors.New("TTF data too small")
	}

	// Parse TTF header
	binary := BinaryFrom(ttfData, false)
	flavor := binary.U32()
	numTables := binary.U16()
	_ = binary.U16() // searchRange
	_ = binary.U16() // entrySelector
	_ = binary.U16() // rangeShift

	// Read TTF table directory
	type ttfTable struct {
		tag      string
		checksum uint32
		offset   uint32
		length   uint32
	}
	tables := make([]ttfTable, numTables)
	for i := range tables {
		tables[i].tag = string(binary.Read(4))
		tables[i].checksum = binary.U32()
		tables[i].offset = binary.U32()
		tables[i].length = binary.U32()
	}

	// Compress each table
	type compTable struct {
		tag      string
		checksum uint32
		compData []byte
		origLen  uint32
	}
	compTables := make([]compTable, numTables)
	for i, t := range tables {
		tableData := ttfData[t.offset : t.offset+t.length]
		var buf bytes.Buffer
		w := zlib.NewWriter(&buf)
		w.Write(tableData)
		w.Close()
		compData := buf.Bytes()

		// If compression doesn't help, store uncompressed
		// WOFF spec allows storing uncompressed if compressed >= original
		if len(compData) >= len(tableData) {
			compData = make([]byte, len(tableData))
			copy(compData, tableData)
		}

		compTables[i] = compTable{
			tag:      t.tag,
			checksum: t.checksum,
			compData: compData,
			origLen:  t.length,
		}
	}

	// Build WOFF
	headerSize := 44
	dirEntrySize := 20
	dirSize := int(numTables) * dirEntrySize
	dataStart := headerSize + dirSize

	// Calculate offsets for compressed data
	woffOffsets := make([]uint32, numTables)
	off := uint32(dataStart)
	for i, ct := range compTables {
		woffOffsets[i] = off
		off += uint32(len(ct.compData))
	}

	totalLength := off

	// Allocate WOFF buffer
	woffData := make([]byte, totalLength)
	woffBin := BinaryFrom(woffData, false)

	// Write WOFF header
	woffBin.PutU32(woffSignature)
	woffBin.PutU32(flavor)
	woffBin.PutU32(totalLength)
	woffBin.PutU16(numTables)
	woffBin.PutU16(0) // reserved
	woffBin.PutU32(uint32(len(ttfData)))
	woffBin.PutU16(0) // majorVersion
	woffBin.PutU16(0) // minorVersion
	woffBin.PutU32(0) // metaOffset
	woffBin.PutU32(0) // metaLength
	woffBin.PutU32(0) // metaOrigLength
	woffBin.PutU32(0) // privOffset
	woffBin.PutU32(0) // privLength

	// Write table directory
	for i, ct := range compTables {
		woffBin.Append([]byte(ct.tag))
		woffBin.PutU32(woffOffsets[i])
		woffBin.PutU32(uint32(len(ct.compData)))
		woffBin.PutU32(ct.origLen)
		woffBin.PutU32(ct.checksum)
	}

	// Write compressed table data
	for _, ct := range compTables {
		woffBin.Append(ct.compData)
	}

	return woffData, nil
}
