package gofont

import (
	"math/bits"
	"sort"
)

func calcSearchParams(numTables int) (searchRange, entrySelector, rangeShift uint16) {
	entrySelector = uint16(bits.Len(uint(numTables)) - 1)
	power := uint16(1 << entrySelector)
	if power > uint16(numTables) {
		power >>= 1
		entrySelector--
	}
	searchRange = power * 16
	rangeShift = uint16(numTables)*16 - searchRange
	return
}

func (ttf *TrueTypeFont) Serialize() ([]byte, error) {
	// Rebuild cmap from abstract rune map if it was modified
	if ttf.runeToGlyphID != nil {
		ttf.cmap = rebuildCmap(ttf.runeToGlyphID, ttf.cmap)
	}

	// Collect all table data
	type tableEntry struct {
		tag  string
		data []byte
	}

	var tables []tableEntry

	if ttf.head != nil {
		tables = append(tables, tableEntry{"head", writeHead(ttf.head)})
	}
	if ttf.os2 != nil {
		tables = append(tables, tableEntry{"OS/2", writeOS2(ttf.os2)})
	}
	if ttf.name != nil {
		tables = append(tables, tableEntry{"name", writeName(ttf.name)})
	}
	if ttf.maxp != nil {
		tables = append(tables, tableEntry{"maxp", writeMaxp(ttf.maxp)})
	}
	if ttf.hhea != nil {
		tables = append(tables, tableEntry{"hhea", writeHhea(ttf.hhea)})
	}
	if ttf.hmtx != nil {
		tables = append(tables, tableEntry{"hmtx", writeHmtx(ttf.hmtx)})
	}
	if ttf.cmap != nil {
		tables = append(tables, tableEntry{"cmap", writeCmap(ttf.cmap)})
	}
	// glyf and loca must be generated together (loca offsets depend on glyf data)
	if ttf.glyf != nil && ttf.head != nil {
		glyfData, locaOffsets := writeGlyf(ttf.glyf)
		tables = append(tables, tableEntry{"glyf", glyfData})
		loca := &Loca{offsets: locaOffsets}
		tables = append(tables, tableEntry{"loca", writeLoca(loca, ttf.head.indexToLocFormat)})
	}
	if ttf.post != nil {
		tables = append(tables, tableEntry{"post", writePost(ttf.post)})
	}

	numTables := len(tables)

	// Sort tables alphabetically by tag (TTF spec requires this)
	sort.Slice(tables, func(i, j int) bool {
		return tables[i].tag < tables[j].tag
	})

	// Calculate offsets
	// Offset table: 12 bytes
	// Directory: 16 bytes per table
	headerSize := 12 + 16*numTables

	offset := uint32(headerSize)
	type tableOffset struct {
		tag      string
		data     []byte
		offset   uint32
		length   uint32
		checksum uint32
	}
	tableOffsets := make([]tableOffset, numTables)

	for i, t := range tables {
		// Align to 4-byte boundary
		if offset%4 != 0 {
			offset += 4 - offset%4
		}

		// For head table, checksum must be calculated with checksumAdjustment zeroed
		checksumData := t.data
		if t.tag == "head" && len(t.data) >= 12 {
			checksumData = make([]byte, len(t.data))
			copy(checksumData, t.data)
			checksumData[8] = 0
			checksumData[9] = 0
			checksumData[10] = 0
			checksumData[11] = 0
		}

		tableOffsets[i] = tableOffset{
			tag:      t.tag,
			data:     t.data,
			offset:   offset,
			length:   uint32(len(t.data)),
			checksum: calcTableChecksum(checksumData),
		}
		offset += uint32(len(t.data))
	}

	totalSize := offset

	// Build the file
	fileData := make([]byte, totalSize)
	binary := BinaryFrom(fileData, false)

	// Write offset table
	binary.PutU32(ttf.version)
	searchRange, entrySelector, rangeShift := calcSearchParams(numTables)
	binary.PutU16(uint16(numTables))
	binary.PutU16(searchRange)
	binary.PutU16(entrySelector)
	binary.PutU16(rangeShift)

	// Write directory entries
	for _, to := range tableOffsets {
		tagBytes := []byte(to.tag)
		if len(tagBytes) < 4 {
			padded := make([]byte, 4)
			copy(padded, tagBytes)
			tagBytes = padded
		}
		binary.Append(tagBytes[:4])
		binary.PutU32(to.checksum)
		binary.PutU32(to.offset)
		binary.PutU32(to.length)
	}

	// Write table data
	for _, to := range tableOffsets {
		copy(fileData[to.offset:], to.data)
	}

	// Patch head.checksumAdjustment
	// The head table has a special field at offset 8 (checksumAdjustment)
	// that must be set to 0xB1B0AFBA - checksum(entire file)
	for _, to := range tableOffsets {
		if to.tag == "head" && len(to.data) >= 12 {
			// Zero out the checksumAdjustment field first
			headStart := to.offset
			fileData[headStart+8] = 0
			fileData[headStart+9] = 0
			fileData[headStart+10] = 0
			fileData[headStart+11] = 0

			// Calculate whole file checksum
			wholeChecksum := calcTableChecksum(fileData)
			adjustment := 0xB1B0AFBA - wholeChecksum

			// Write adjustment back into head table
			binary := BinaryFrom(fileData[headStart+8:headStart+12], false)
			binary.PutU32(adjustment)
			break
		}
	}

	return fileData, nil
}
