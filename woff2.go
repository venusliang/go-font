package gofont

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"

	"github.com/andybalholm/brotli"
)

const woff2Signature = 0x774F4632 // "wOF2"

// woff2KnownTags is the list of 63 predefined table tags for WOFF2.
var woff2KnownTags = [63]string{
	"cmap", "head", "hhea", "hmtx",
	"maxp", "name", "OS/2", "post",
	"cvt ", "fpgm", "glyf", "loca",
	"prep", "CFF ", "VORG", "EBDT",
	"EBLC", "gasp", "hdmx", "kern",
	"LTSH", "PCLT", "VDMX", "vhea",
	"vmtx", "BASE", "GDEF", "GPOS",
	"GSUB", "EBSC", "JSTF", "MATH",
	"CBDT", "CBLC", "COLR", "CPAL",
	"SVG ", "sbix", "acnt", "avar",
	"bdat", "bloc", "bsln", "cvar",
	"fdsc", "feat", "fmtx", "fvar",
	"gvar", "hsty", "just", "lcar",
	"mort", "morx", "opbd", "prop",
	"trak", "Zapf", "Silf", "Glat",
	"Gloc", "Feat", "Sill",
}

// --- Variable-length integer encoding ---

// readUIntBase128 reads a UIntBase128 value from data starting at *offset.
func readUIntBase128(data []byte, offset *int) (uint32, error) {
	var accum uint32
	for i := 0; i < 5; i++ {
		if *offset >= len(data) {
			return 0, errors.New("WOFF2: unexpected end of UIntBase128")
		}
		b := data[*offset]
		*offset++
		if i == 0 && b == 0x80 {
			return 0, errors.New("WOFF2: UIntBase128 leading zero")
		}
		if accum&0xFE000000 != 0 {
			return 0, errors.New("WOFF2: UIntBase128 overflow")
		}
		accum = (accum << 7) | uint32(b&0x7F)
		if b&0x80 == 0 {
			return accum, nil
		}
	}
	return 0, errors.New("WOFF2: UIntBase128 exceeds 5 bytes")
}

// writeUIntBase128 appends a UIntBase128 encoded value to buf.
func writeUIntBase128(buf []byte, val uint32) []byte {
	if val == 0 {
		return append(buf, 0)
	}
	var tmp [5]byte
	n := 0
	for i := 4; i >= 0; i-- {
		group := (val >> (uint(i) * 7)) & 0x7F
		if n > 0 || group != 0 {
			if i > 0 {
				group |= 0x80
			}
			tmp[n] = byte(group)
			n++
		}
	}
	return append(buf, tmp[:n]...)
}

// read255UInt16 reads a 255UInt16 value from data starting at *offset.
func read255UInt16(data []byte, offset *int) (uint16, error) {
	if *offset >= len(data) {
		return 0, errors.New("WOFF2: unexpected end of 255UInt16")
	}
	code := data[*offset]
	*offset++
	switch code {
	case 253:
		if *offset+1 >= len(data) {
			return 0, errors.New("WOFF2: unexpected end of 255UInt16")
		}
		val := binary.BigEndian.Uint16(data[*offset:])
		*offset += 2
		return val, nil
	case 254:
		if *offset >= len(data) {
			return 0, errors.New("WOFF2: unexpected end of 255UInt16")
		}
		val := uint16(data[*offset]) + 253*2
		*offset++
		return val, nil
	case 255:
		if *offset >= len(data) {
			return 0, errors.New("WOFF2: unexpected end of 255UInt16")
		}
		val := uint16(data[*offset]) + 253
		*offset++
		return val, nil
	default:
		return uint16(code), nil
	}
}

// signInt16 returns -1 if the bit at pos in flag is set, 1 otherwise.
func signInt16(flag byte, pos uint) int16 {
	if flag&(1<<pos) != 0 {
		return -1
	}
	return 1
}

// --- WOFF2 table directory entry ---

type woff2TableEntry struct {
	tag              string
	flags            byte
	transformVersion int
	origLength       uint32
	transformLength  uint32
	data             []byte
}

// --- ParseWOFF2 ---

// ParseWOFF2 parses a WOFF2 (Web Open Font Format 2) file and returns a TrueTypeFont.
func ParseWOFF2(data []byte) (TrueTypeFont, error) {
	if len(data) < 48 {
		return TrueTypeFont{}, errors.New("WOFF2 file too small")
	}

	bin := BinaryFrom(data, false)
	signature := bin.U32()
	if signature != woff2Signature {
		return TrueTypeFont{}, errors.New("invalid WOFF2 signature")
	}

	flavor := bin.U32()
	_ = bin.U32()           // total length
	numTables := bin.U16()
	_ = bin.U16()           // reserved
	_ = bin.U32()           // totalSfntSize
	totalCompressedSize := bin.U32()
	_ = bin.U16()           // majorVersion
	_ = bin.U16()           // minorVersion
	_ = bin.U32()           // metaOffset
	_ = bin.U32()           // metaLength
	_ = bin.U32()           // metaOrigLength
	_ = bin.U32()           // privOffset
	_ = bin.U32()           // privLength

	entries := make([]woff2TableEntry, numTables)
	tagIndex := make(map[string]int)
	var uncompressedSize uint32
	dirOffset := 48

	for i := 0; i < int(numTables); i++ {
		if dirOffset >= len(data) {
			return TrueTypeFont{}, errors.New("WOFF2: table directory out of bounds")
		}
		flags := data[dirOffset]
		dirOffset++
		tagIdx := int(flags & 0x3F)
		transformVersion := int((flags & 0xC0) >> 6)

		var tag string
		if tagIdx == 63 {
			if dirOffset+4 > len(data) {
				return TrueTypeFont{}, errors.New("WOFF2: custom tag out of bounds")
			}
			tag = string(data[dirOffset : dirOffset+4])
			dirOffset += 4
		} else {
			tag = woff2KnownTags[tagIdx]
		}

		origLength, err := readUIntBase128(data, &dirOffset)
		if err != nil {
			return TrueTypeFont{}, err
		}

		var transformLength uint32
		if (tag == "glyf" || tag == "loca") && transformVersion == 0 {
			transformLength, err = readUIntBase128(data, &dirOffset)
			if err != nil {
				return TrueTypeFont{}, err
			}
			if tag != "loca" && transformLength == 0 {
				return TrueTypeFont{}, errors.New("WOFF2: glyf transformLength must be non-zero")
			}
			if math.MaxUint32-uncompressedSize < transformLength {
				return TrueTypeFont{}, errors.New("WOFF2: uncompressed size overflow")
			}
			uncompressedSize += transformLength
		} else if transformVersion == 0 || (transformVersion == 3 && (tag == "glyf" || tag == "loca")) {
			if math.MaxUint32-uncompressedSize < origLength {
				return TrueTypeFont{}, errors.New("WOFF2: uncompressed size overflow")
			}
			uncompressedSize += origLength
		} else if tag == "hmtx" && transformVersion == 1 {
			transformLength, err = readUIntBase128(data, &dirOffset)
			if err != nil {
				return TrueTypeFont{}, err
			}
			if math.MaxUint32-uncompressedSize < transformLength {
				return TrueTypeFont{}, errors.New("WOFF2: uncompressed size overflow")
			}
			uncompressedSize += transformLength
		} else {
			return TrueTypeFont{}, errors.New("WOFF2: invalid transform for table " + tag)
		}

		if _, exists := tagIndex[tag]; exists {
			return TrueTypeFont{}, errors.New("WOFF2: duplicate table " + tag)
		}
		tagIndex[tag] = i
		entries[i] = woff2TableEntry{
			tag:              tag,
			flags:            flags,
			transformVersion: transformVersion,
			origLength:       origLength,
			transformLength:  transformLength,
		}
	}

	iGlyf, hasGlyf := tagIndex["glyf"]
	_, hasLoca := tagIndex["loca"]
	if hasGlyf != hasLoca {
		return TrueTypeFont{}, errors.New("WOFF2: glyf and loca must both be present or absent")
	}
	if hasLoca && entries[tagIndex["loca"]].transformLength != 0 {
		return TrueTypeFont{}, errors.New("WOFF2: loca transformLength must be zero")
	}

	compStart := dirOffset
	compEnd := compStart + int(totalCompressedSize)
	if compEnd > len(data) {
		return TrueTypeFont{}, errors.New("WOFF2: compressed data out of bounds")
	}

	rBrotli := brotli.NewReader(bytes.NewReader(data[compStart:compEnd]))
	decompressed, err := io.ReadAll(rBrotli)
	if err != nil {
		return TrueTypeFont{}, errors.New("WOFF2: Brotli decompression failed: " + err.Error())
	}
	if uint32(len(decompressed)) != uncompressedSize {
		return TrueTypeFont{}, errors.New("WOFF2: decompressed size mismatch")
	}

	var offset uint32
	for i := range entries {
		if entries[i].tag == "loca" && entries[i].transformVersion == 0 {
			continue
		}
		n := entries[i].origLength
		if entries[i].transformLength != 0 {
			n = entries[i].transformLength
		}
		if uint32(len(decompressed))-offset < n {
			return TrueTypeFont{}, errors.New("WOFF2: table data out of bounds in decompressed stream")
		}
		entries[i].data = decompressed[offset : offset+n]
		offset += n
	}

	if hasGlyf && entries[iGlyf].transformVersion == 0 {
		glyfData, locaData, err := reconstructGlyfLoca(entries[iGlyf].data, entries[tagIndex["loca"]].origLength)
		if err != nil {
			return TrueTypeFont{}, err
		}
		entries[iGlyf].data = glyfData
		entries[tagIndex["loca"]].data = locaData
	}

	if iHmtx, hasHmtx := tagIndex["hmtx"]; hasHmtx && entries[iHmtx].transformVersion == 1 {
		iHead, ok := tagIndex["head"]
		if !ok {
			return TrueTypeFont{}, errors.New("WOFF2: hmtx transform requires head table")
		}
		if !hasGlyf || !hasLoca {
			return TrueTypeFont{}, errors.New("WOFF2: hmtx transform requires glyf/loca tables")
		}
		iMaxp, ok := tagIndex["maxp"]
		if !ok {
			return TrueTypeFont{}, errors.New("WOFF2: hmtx transform requires maxp table")
		}
		iHhea, ok := tagIndex["hhea"]
		if !ok {
			return TrueTypeFont{}, errors.New("WOFF2: hmtx transform requires hhea table")
		}
		hmtxData, err := reconstructHmtx(
			entries[iHmtx].data, entries[iHead].data,
			entries[iGlyf].data, entries[tagIndex["loca"]].data,
			entries[iMaxp].data, entries[iHhea].data,
		)
		if err != nil {
			return TrueTypeFont{}, err
		}
		entries[iHmtx].data = hmtxData
	}

	if iHead, ok := tagIndex["head"]; ok && len(entries[iHead].data) >= 18 {
		binary.BigEndian.PutUint32(entries[iHead].data[8:], 0)
		flags := binary.BigEndian.Uint16(entries[iHead].data[16:])
		flags |= 0x0800
		binary.BigEndian.PutUint16(entries[iHead].data[16:], flags)
	}

	genericEntries := make([]rebuildTableEntry, len(entries))
	for i, e := range entries {
		genericEntries[i] = rebuildTableEntry{tag: e.tag, data: e.data}
	}
	return rebuildTTF(flavor, genericEntries)
}

// --- Shared TTF rebuilding (used by both WOFF and WOFF2) ---

// rebuildTableEntry is a generic table entry used to rebuild a TTF from decompressed table data.
type rebuildTableEntry struct {
	tag  string
	data []byte
}

// rebuildTTF reconstructs a TTF byte stream from decompressed table entries and parses it.
func rebuildTTF(flavor uint32, entries []rebuildTableEntry) (TrueTypeFont, error) {
	numTables := uint16(len(entries))
	searchRange, entrySelector, rangeShift := calcSearchParams(int(numTables))

	sorted := make([]rebuildTableEntry, len(entries))
	copy(sorted, entries)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].tag < sorted[j-1].tag; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}

	headerSize := 12 + 16*int(numTables)
	type tableInfo struct {
		tag      string
		data     []byte
		offset   uint32
		length   uint32
		checksum uint32
	}
	tables := make([]tableInfo, numTables)
	sfntOffset := uint32(headerSize)

	for i, e := range sorted {
		data := e.data
		if data == nil {
			data = []byte{}
		}
		paddedLen := uint32(len(data))
		if paddedLen%4 != 0 {
			paddedLen += 4 - paddedLen%4
		}
		checksum := calcTableChecksum(data)
		// head table checksum is calculated with checksumAdjustment zeroed
		if e.tag == "head" && len(data) >= 12 {
			headCopy := make([]byte, len(data))
			copy(headCopy, data)
			headCopy[8] = 0
			headCopy[9] = 0
			headCopy[10] = 0
			headCopy[11] = 0
			checksum = calcTableChecksum(headCopy)
		}

		tables[i] = tableInfo{
			tag:      e.tag,
			data:     data,
			offset:   sfntOffset,
			length:   uint32(len(data)),
			checksum: checksum,
		}
		sfntOffset += paddedLen
	}

	ttfData := make([]byte, sfntOffset)
	bin := BinaryFrom(ttfData, false)

	bin.PutU32(flavor)
	bin.PutU16(numTables)
	bin.PutU16(searchRange)
	bin.PutU16(entrySelector)
	bin.PutU16(rangeShift)

	for _, t := range tables {
		tagBytes := []byte(t.tag)
		if len(tagBytes) < 4 {
			padded := make([]byte, 4)
			copy(padded, tagBytes)
			tagBytes = padded
		}
		bin.Append(tagBytes[:4])
		bin.PutU32(t.checksum)
		bin.PutU32(t.offset)
		bin.PutU32(t.length)
	}

	for _, t := range tables {
		copy(ttfData[t.offset:], t.data)
		paddedLen := t.length
		if paddedLen%4 != 0 {
			paddedLen += 4 - paddedLen%4
		}
		for j := t.length; j < paddedLen; j++ {
			ttfData[t.offset+j] = 0
		}
	}

	for _, t := range tables {
		if t.tag == "head" && len(t.data) >= 12 {
			headStart := t.offset
			ttfData[headStart+8] = 0
			ttfData[headStart+9] = 0
			ttfData[headStart+10] = 0
			ttfData[headStart+11] = 0
			wholeChecksum := calcTableChecksum(ttfData)
			adjustment := 0xB1B0AFBA - wholeChecksum
			bin := BinaryFrom(ttfData[headStart+8:headStart+12], false)
			bin.PutU32(adjustment)
			break
		}
	}

	return Parse(ttfData)
}

// --- reconstructGlyfLoca reverses the WOFF2 glyf/loca transform ---

func reconstructGlyfLoca(data []byte, origLocaLength uint32) (glyfData, locaData []byte, err error) {
	if len(data) < 36 {
		return nil, nil, errors.New("WOFF2: transformed glyf data too small")
	}

	_ = binary.BigEndian.Uint16(data[0:2])
	optionFlags := binary.BigEndian.Uint16(data[2:4])
	numGlyphs := binary.BigEndian.Uint16(data[4:6])
	indexFormat := binary.BigEndian.Uint16(data[6:8])
	nContourStreamSize := binary.BigEndian.Uint32(data[8:12])
	nPointsStreamSize := binary.BigEndian.Uint32(data[12:16])
	flagStreamSize := binary.BigEndian.Uint32(data[16:20])
	glyphStreamSize := binary.BigEndian.Uint32(data[20:24])
	compositeStreamSize := binary.BigEndian.Uint32(data[24:28])
	bboxBitmapAndStreamSize := binary.BigEndian.Uint32(data[28:32])
	instructionStreamSize := binary.BigEndian.Uint32(data[32:36])

	locaLength := (uint32(numGlyphs) + 1) * 2
	if indexFormat != 0 {
		locaLength *= 2
	}
	if locaLength != origLocaLength {
		return nil, nil, errors.New("WOFF2: loca origLength mismatch")
	}

	streamOff := uint32(36)
	nContourStream := data[streamOff : streamOff+nContourStreamSize]
	streamOff += nContourStreamSize
	nPointsStream := data[streamOff : streamOff+nPointsStreamSize]
	streamOff += nPointsStreamSize
	flagStream := data[streamOff : streamOff+flagStreamSize]
	streamOff += flagStreamSize
	glyphStream := data[streamOff : streamOff+glyphStreamSize]
	streamOff += glyphStreamSize
	compositeStream := data[streamOff : streamOff+compositeStreamSize]
	streamOff += compositeStreamSize
	bboxBitmapPlusStream := data[streamOff : streamOff+bboxBitmapAndStreamSize]
	streamOff += bboxBitmapAndStreamSize
	instructionStream := data[streamOff : streamOff+instructionStreamSize]

	bitmapSize := ((uint32(numGlyphs) + 31) >> 5) << 2
	var bboxBitmap []byte
	var bboxStreamData []byte
	if uint32(len(bboxBitmapPlusStream)) >= bitmapSize {
		bboxBitmap = bboxBitmapPlusStream[:bitmapSize]
		bboxStreamData = bboxBitmapPlusStream[bitmapSize:]
	}

	var overlapBitmap []byte
	if optionFlags&0x0001 != 0 && streamOff+bitmapSize <= uint32(len(data)) {
		overlapBitmap = data[streamOff : streamOff+bitmapSize]
	}

	ncOff := 0
	npOff := 0
	flOff := 0
	gsOff := 0
	csOff := 0
	bbOff := 0
	isOff := 0

	glyfBuf := make([]byte, 0, len(data))
	locaBuf := make([]byte, locaLength)
	locaBin := BinaryFrom(locaBuf, false)
	glyfOffset := uint32(0)

	for glyphID := uint16(0); glyphID < numGlyphs; glyphID++ {
		if indexFormat == 0 {
			locaBin.PutU16(uint16(glyfOffset >> 1))
		} else {
			locaBin.PutU32(glyfOffset)
		}

		explicitBbox := false
		if bboxBitmap != nil {
			byteIdx := glyphID / 8
			bitIdx := glyphID % 8
			if int(byteIdx) < len(bboxBitmap) {
				explicitBbox = (bboxBitmap[byteIdx] & (1 << (7 - bitIdx))) != 0
			}
		}

		hasOverlap := false
		if overlapBitmap != nil {
			byteIdx := glyphID / 8
			bitIdx := glyphID % 8
			if int(byteIdx) < len(overlapBitmap) {
				hasOverlap = (overlapBitmap[byteIdx] & (1 << (7 - bitIdx))) != 0
			}
		}

		if ncOff+1 >= len(nContourStream) {
			continue
		}
		nContours := int16(binary.BigEndian.Uint16(nContourStream[ncOff:]))
		ncOff += 2

		if nContours == 0 {
			if explicitBbox {
				return nil, nil, errors.New("WOFF2: empty glyph with explicit bbox")
			}
			continue
		}

		if nContours > 0 {
			var xMin, yMin, xMax, yMax int16

			if explicitBbox && len(bboxStreamData)-bbOff >= 8 {
				xMin = int16(binary.BigEndian.Uint16(bboxStreamData[bbOff:]))
				bbOff += 2
				yMin = int16(binary.BigEndian.Uint16(bboxStreamData[bbOff:]))
				bbOff += 2
				xMax = int16(binary.BigEndian.Uint16(bboxStreamData[bbOff:]))
				bbOff += 2
				yMax = int16(binary.BigEndian.Uint16(bboxStreamData[bbOff:]))
				bbOff += 2
			}

			var nPoints uint16
			endPtsOfContours := make([]uint16, nContours)
			for i := int16(0); i < nContours; i++ {
				nPoint, err := read255UInt16(nPointsStream, &npOff)
				if err != nil {
					return nil, nil, err
				}
				if math.MaxUint16-nPoints < nPoint {
					return nil, nil, errors.New("WOFF2: point count overflow")
				}
				nPoints += nPoint
				endPtsOfContours[i] = nPoints - 1
			}

			outlineFlags := make([]byte, nPoints)
			xCoordinates := make([]int16, nPoints)
			yCoordinates := make([]int16, nPoints)
			var x, y int16

			for iPoint := uint16(0); iPoint < nPoints; iPoint++ {
				if flOff >= len(flagStream) {
					return nil, nil, errors.New("WOFF2: flagStream exhausted")
				}
				flag := flagStream[flOff]
				flOff++
				onCurve := (flag & 0x80) == 0
				flag &= 0x7F

				var dx, dy int16
				if flag < 10 {
					if gsOff >= len(glyphStream) {
						return nil, nil, errors.New("WOFF2: glyphStream exhausted")
					}
					coord0 := int16(glyphStream[gsOff])
					gsOff++
					dy = signInt16(flag, 0) * (int16(flag&0x0E)<<7 + coord0)
				} else if flag < 20 {
					if gsOff >= len(glyphStream) {
						return nil, nil, errors.New("WOFF2: glyphStream exhausted")
					}
					coord0 := int16(glyphStream[gsOff])
					gsOff++
					dx = signInt16(flag, 0) * (int16((flag-10)&0x0E)<<7 + coord0)
				} else if flag < 84 {
					if gsOff >= len(glyphStream) {
						return nil, nil, errors.New("WOFF2: glyphStream exhausted")
					}
					coord0 := int16(glyphStream[gsOff])
					gsOff++
					dx = signInt16(flag, 0) * (1 + int16((flag-20)&0x30) + coord0>>4)
					dy = signInt16(flag, 1) * (1 + int16((flag-20)&0x0C)<<2 + coord0&0x0F)
				} else if flag < 120 {
					if gsOff+1 >= len(glyphStream) {
						return nil, nil, errors.New("WOFF2: glyphStream exhausted")
					}
					coord0 := int16(glyphStream[gsOff])
					gsOff++
					coord1 := int16(glyphStream[gsOff])
					gsOff++
					dx = signInt16(flag, 0) * (1 + int16((flag-84)/12)<<8 + coord0)
					dy = signInt16(flag, 1) * (1 + (int16((flag-84)%12)>>2)<<8 + coord1)
				} else if flag < 124 {
					if gsOff+2 >= len(glyphStream) {
						return nil, nil, errors.New("WOFF2: glyphStream exhausted")
					}
					coord0 := int16(glyphStream[gsOff])
					gsOff++
					coord1 := int16(glyphStream[gsOff])
					gsOff++
					coord2 := int16(glyphStream[gsOff])
					gsOff++
					dx = signInt16(flag, 0) * (coord0<<4 + coord1>>4)
					dy = signInt16(flag, 1) * ((coord1&0x0F)<<8 + coord2)
				} else {
					if gsOff+3 >= len(glyphStream) {
						return nil, nil, errors.New("WOFF2: glyphStream exhausted")
					}
					coord0 := int16(glyphStream[gsOff])
					gsOff++
					coord1 := int16(glyphStream[gsOff])
					gsOff++
					coord2 := int16(glyphStream[gsOff])
					gsOff++
					coord3 := int16(glyphStream[gsOff])
					gsOff++
					dx = signInt16(flag, 0) * (coord0<<8 + coord1)
					dy = signInt16(flag, 1) * (coord2<<8 + coord3)
				}

				xCoordinates[iPoint] = dx
				yCoordinates[iPoint] = dy

				var outlineFlag byte
				if onCurve {
					outlineFlag |= 0x01
				}
				if hasOverlap {
					outlineFlag |= 0x40
				}
				outlineFlags[iPoint] = outlineFlag

				if !explicitBbox {
					x += dx
					y += dy
					if iPoint == 0 {
						xMin, xMax = x, x
						yMin, yMax = y, y
					} else {
						if x < xMin {
							xMin = x
						} else if x > xMax {
							xMax = x
						}
						if y < yMin {
							yMin = y
						} else if y > yMax {
							yMax = y
						}
					}
				}
			}

			instructionLength, err := read255UInt16(glyphStream, &gsOff)
			if err != nil {
				return nil, nil, err
			}
			var instructions []byte
			if int(instructionLength) <= len(instructionStream)-isOff {
				instructions = instructionStream[isOff : isOff+int(instructionLength)]
				isOff += int(instructionLength)
			}

			glyphStart := len(glyfBuf)
			glyfBuf = append(glyfBuf, make([]byte, 10)...)
			binary.BigEndian.PutUint16(glyfBuf[glyphStart:], uint16(nContours))
			binary.BigEndian.PutUint16(glyfBuf[glyphStart+2:], uint16(xMin))
			binary.BigEndian.PutUint16(glyfBuf[glyphStart+4:], uint16(yMin))
			binary.BigEndian.PutUint16(glyfBuf[glyphStart+6:], uint16(xMax))
			binary.BigEndian.PutUint16(glyfBuf[glyphStart+8:], uint16(yMax))

			for _, ep := range endPtsOfContours {
				glyfBuf = append(glyfBuf, 0, 0)
				binary.BigEndian.PutUint16(glyfBuf[len(glyfBuf)-2:], ep)
			}

			glyfBuf = append(glyfBuf, 0, 0)
			binary.BigEndian.PutUint16(glyfBuf[len(glyfBuf)-2:], instructionLength)
			glyfBuf = append(glyfBuf, instructions...)
			glyfBuf = append(glyfBuf, outlineFlags...)

			for _, xc := range xCoordinates {
				glyfBuf = append(glyfBuf, 0, 0)
				binary.BigEndian.PutUint16(glyfBuf[len(glyfBuf)-2:], uint16(xc))
			}
			for _, yc := range yCoordinates {
				glyfBuf = append(glyfBuf, 0, 0)
				binary.BigEndian.PutUint16(glyfBuf[len(glyfBuf)-2:], uint16(yc))
			}

			for len(glyfBuf)%4 != 0 {
				glyfBuf = append(glyfBuf, 0)
			}
			glyfOffset = uint32(len(glyfBuf))

		} else {
			if !explicitBbox {
				return nil, nil, errors.New("WOFF2: composite glyph must have explicit bbox")
			}
			if len(bboxStreamData)-bbOff < 8 {
				return nil, nil, errors.New("WOFF2: bboxStream exhausted")
			}
			xMin := int16(binary.BigEndian.Uint16(bboxStreamData[bbOff:]))
			bbOff += 2
			yMin := int16(binary.BigEndian.Uint16(bboxStreamData[bbOff:]))
			bbOff += 2
			xMax := int16(binary.BigEndian.Uint16(bboxStreamData[bbOff:]))
			bbOff += 2
			yMax := int16(binary.BigEndian.Uint16(bboxStreamData[bbOff:]))
			bbOff += 2

			glyphStart := len(glyfBuf)
			glyfBuf = append(glyfBuf, make([]byte, 10)...)
			binary.BigEndian.PutUint16(glyfBuf[glyphStart:], uint16(nContours))
			binary.BigEndian.PutUint16(glyfBuf[glyphStart+2:], uint16(xMin))
			binary.BigEndian.PutUint16(glyfBuf[glyphStart+4:], uint16(yMin))
			binary.BigEndian.PutUint16(glyfBuf[glyphStart+6:], uint16(xMax))
			binary.BigEndian.PutUint16(glyfBuf[glyphStart+8:], uint16(yMax))

			hasInstructions := false
			for {
				if csOff+1 >= len(compositeStream) {
					return nil, nil, errors.New("WOFF2: compositeStream exhausted")
				}
				compositeFlag := binary.BigEndian.Uint16(compositeStream[csOff:])
				csOff += 2

				argsAreWords := (compositeFlag & 0x0001) != 0
				haveScale := (compositeFlag & 0x0008) != 0
				moreComponents := (compositeFlag & 0x0020) != 0
				haveXYScales := (compositeFlag & 0x0040) != 0
				have2by2 := (compositeFlag & 0x0080) != 0
				haveInstructions := (compositeFlag & 0x0100) != 0

				numBytes := 4
				if argsAreWords {
					numBytes += 2
				}
				if haveScale {
					numBytes += 2
				} else if haveXYScales {
					numBytes += 4
				} else if have2by2 {
					numBytes += 8
				}

				if csOff+numBytes > len(compositeStream) {
					return nil, nil, errors.New("WOFF2: compositeStream exhausted")
				}

				glyfBuf = append(glyfBuf, 0, 0)
				binary.BigEndian.PutUint16(glyfBuf[len(glyfBuf)-2:], compositeFlag)
				glyfBuf = append(glyfBuf, compositeStream[csOff:csOff+numBytes]...)
				csOff += numBytes

				if haveInstructions {
					hasInstructions = true
				}
				if !moreComponents {
					break
				}
			}

			if hasInstructions {
				instructionLength, err := read255UInt16(glyphStream, &gsOff)
				if err != nil {
					return nil, nil, err
				}
				glyfBuf = append(glyfBuf, 0, 0)
				binary.BigEndian.PutUint16(glyfBuf[len(glyfBuf)-2:], instructionLength)
				if int(instructionLength) <= len(instructionStream)-isOff {
					glyfBuf = append(glyfBuf, instructionStream[isOff:isOff+int(instructionLength)]...)
					isOff += int(instructionLength)
				}
			}

			for len(glyfBuf)%4 != 0 {
				glyfBuf = append(glyfBuf, 0)
			}
			glyfOffset = uint32(len(glyfBuf))
		}
	}

	if indexFormat == 0 {
		locaBin.PutU16(uint16(glyfOffset >> 1))
	} else {
		locaBin.PutU32(glyfOffset)
	}

	return glyfBuf, locaBuf, nil
}

// --- reconstructHmtx reverses the WOFF2 hmtx transform ---

func reconstructHmtx(data, head, glyf, loca, maxp, hhea []byte) ([]byte, error) {
	if len(data) < 1 {
		return nil, errors.New("WOFF2: hmtx transform data too small")
	}
	if len(head) < 52 {
		return nil, errors.New("WOFF2: head table too small for hmtx transform")
	}
	indexFormat := binary.BigEndian.Uint16(head[50:52])
	if len(maxp) < 6 {
		return nil, errors.New("WOFF2: maxp table too small for hmtx transform")
	}
	numGlyphs := binary.BigEndian.Uint16(maxp[4:6])
	if len(hhea) < 36 {
		return nil, errors.New("WOFF2: hhea table too small for hmtx transform")
	}
	numHMetrics := binary.BigEndian.Uint16(hhea[34:36])

	if numHMetrics < 1 {
		return nil, errors.New("WOFF2: numHMetrics must be >= 1")
	}
	if numGlyphs < numHMetrics {
		return nil, errors.New("WOFF2: numGlyphs must be >= numHMetrics")
	}

	flags := data[0]
	reconstructProportional := flags&0x01 != 0
	reconstructMonospaced := flags&0x02 != 0
	if !reconstructProportional && !reconstructMonospaced {
		return nil, errors.New("WOFF2: hmtx must reconstruct at least one LSB array")
	}

	off := 1
	advanceWidths := make([]uint16, numHMetrics)
	for i := 0; i < int(numHMetrics); i++ {
		if off+1 >= len(data) {
			return nil, errors.New("WOFF2: hmtx transform data truncated")
		}
		advanceWidths[i] = binary.BigEndian.Uint16(data[off:])
		off += 2
	}

	lsbs := make([]int16, numGlyphs)
	if !reconstructProportional {
		for i := 0; i < int(numHMetrics); i++ {
			if off+1 >= len(data) {
				return nil, errors.New("WOFF2: hmtx transform data truncated")
			}
			lsbs[i] = int16(binary.BigEndian.Uint16(data[off:]))
			off += 2
		}
	}
	if !reconstructMonospaced {
		for i := int(numHMetrics); i < int(numGlyphs); i++ {
			if off+1 >= len(data) {
				return nil, errors.New("WOFF2: hmtx transform data truncated")
			}
			lsbs[i] = int16(binary.BigEndian.Uint16(data[off:]))
			off += 2
		}
	}

	iGlyphMin := uint16(0)
	iGlyphMax := numGlyphs
	if !reconstructProportional {
		iGlyphMin = numHMetrics
	}
	if !reconstructMonospaced {
		iGlyphMax = numHMetrics
	}

	locaEntrySize := 2
	if indexFormat != 0 {
		locaEntrySize = 4
	}

	for iGlyph := iGlyphMin; iGlyph < iGlyphMax; iGlyph++ {
		locaOff := iGlyph * uint16(locaEntrySize)
		var glyphOffset uint32
		if indexFormat != 0 {
			glyphOffset = binary.BigEndian.Uint32(loca[locaOff:])
		} else {
			glyphOffset = uint32(binary.BigEndian.Uint16(loca[locaOff:])) << 1
		}

		nextLocaOff := (iGlyph + 1) * uint16(locaEntrySize)
		var nextGlyphOffset uint32
		if indexFormat != 0 {
			nextGlyphOffset = binary.BigEndian.Uint32(loca[nextLocaOff:])
		} else {
			nextGlyphOffset = uint32(binary.BigEndian.Uint16(loca[nextLocaOff:])) << 1
		}

		if nextGlyphOffset == glyphOffset {
			lsbs[iGlyph] = 0
		} else {
			if int(glyphOffset)+4 > len(glyf) {
				return nil, errors.New("WOFF2: glyf table too small for hmtx transform")
			}
			lsbs[iGlyph] = int16(binary.BigEndian.Uint16(glyf[glyphOffset+2:]))
		}
	}

	size := int(numHMetrics)*4 + (int(numGlyphs)-int(numHMetrics))*2
	result := make([]byte, size)
	bin := BinaryFrom(result, false)
	for i := 0; i < int(numHMetrics); i++ {
		bin.PutU16(advanceWidths[i])
		bin.PutU16(uint16(lsbs[i]))
	}
	for i := int(numHMetrics); i < int(numGlyphs); i++ {
		bin.PutU16(uint16(lsbs[i]))
	}

	return result, nil
}

// --- SerializeWOFF2 ---

// SerializeWOFF2 serializes the font as a WOFF2 file.
// No table transforms are applied (all tables stored raw).
func (ttf *TrueTypeFont) SerializeWOFF2() ([]byte, error) {
	ttfData, err := ttf.Serialize()
	if err != nil {
		return nil, err
	}
	if len(ttfData) < 12 {
		return nil, errors.New("TTF data too small")
	}

	bin := BinaryFrom(ttfData, false)
	flavor := bin.U32()
	numTables := bin.U16()
	_ = bin.U16()
	_ = bin.U16()
	_ = bin.U16()

	type ttfTable struct {
		tag      string
		checksum uint32
		offset   uint32
		length   uint32
	}
	tables := make([]ttfTable, numTables)
	for i := range tables {
		tables[i].tag = string(bin.Read(4))
		tables[i].checksum = bin.U32()
		tables[i].offset = bin.U32()
		tables[i].length = bin.U32()
	}

	var dirBuf []byte
	var tableData []byte

	for _, t := range tables {
		if t.tag == "DSIG" {
			continue
		}

		tagIdx := 63
		for i, known := range woff2KnownTags {
			if known == t.tag {
				tagIdx = i
				break
			}
		}

		transformVersion := byte(0)
		if t.tag == "glyf" || t.tag == "loca" {
			transformVersion = 3
		}

		flags := byte(transformVersion<<6) | byte(tagIdx&0x3F)
		dirBuf = append(dirBuf, flags)

		if tagIdx == 63 {
			dirBuf = append(dirBuf, []byte(t.tag)...)
		}

		dirBuf = writeUIntBase128(dirBuf, t.length)

		tableSlice := ttfData[t.offset : t.offset+t.length]
		tableData = append(tableData, tableSlice...)
	}

	actualNumTables := 0
	for _, t := range tables {
		if t.tag != "DSIG" {
			actualNumTables++
		}
	}

	var compressed bytes.Buffer
	w := brotli.NewWriter(&compressed)
	_, err = w.Write(tableData)
	if err != nil {
		return nil, err
	}
	w.Close()
	compressedData := compressed.Bytes()

	totalLength := uint32(48 + len(dirBuf) + len(compressedData))
	padding := (4 - totalLength%4) % 4
	totalLength += padding

	woff2Data := make([]byte, totalLength)
	bin2 := BinaryFrom(woff2Data, false)

	bin2.PutU32(woff2Signature)
	bin2.PutU32(flavor)
	bin2.PutU32(totalLength)
	bin2.PutU16(uint16(actualNumTables))
	bin2.PutU16(0)
	bin2.PutU32(uint32(len(ttfData)))
	bin2.PutU32(uint32(len(compressedData)))
	bin2.PutU16(1)
	bin2.PutU16(0)
	bin2.PutU32(0)
	bin2.PutU32(0)
	bin2.PutU32(0)
	bin2.PutU32(0)
	bin2.PutU32(0)

	bin2.Append(dirBuf)
	bin2.Append(compressedData)

	return woff2Data, nil
}
