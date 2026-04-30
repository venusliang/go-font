package gofont

import "fmt"

type TrueTypeFont struct {
	data          []byte
	version       uint32
	numTables     uint16
	searchRange   uint16
	entrySelector uint16
	rangeShift    uint16
	directorys    map[string]TableDirectory
	head          *Head
	os2           *OS2
	cmap          *CMap
	maxp          *Maxp
	name          *Name
	hhea          *Hhea
	hmtx          *Hmtx
	loca          *Loca
	glyf          []*Glyph
	post          *Post
	kern          *Kern
	gpos          *GPOS
	gsub          *GSUB
	runeToGlyphID map[rune]uint16
}

// Parse parses TTF binary data and returns a TrueTypeFont.
func Parse(data []byte) (TrueTypeFont, error) {
	return parseFromOffset(data, 0)
}

// parseFromOffset parses a TTF starting at the given byte offset within data.
// Used by both Parse() (offset=0) and ParseTTC() (offset from TTC header).
func parseFromOffset(data []byte, offset uint32) (ttf TrueTypeFont, err error) {
	ttf.data = data
	binary := BinaryFrom(data[offset:], false)

	ttf.version = binary.U32()
	if ttf.version != 0x00010000 && ttf.version != 0x74727565 {
		err = fmt.Errorf("invalid version: %x", ttf.version)
		return
	}

	ttf.numTables = binary.U16()
	ttf.searchRange = binary.U16()
	ttf.entrySelector = binary.U16()
	ttf.rangeShift = binary.U16()

	ttf.directorys = make(map[string]TableDirectory)

	for i := 0; i < int(ttf.numTables); i++ {
		name := string(binary.Bytes(4))
		directory := TableDirectory{
			tag:      binary.U32(),
			checkSum: binary.U32(),
			offset:   binary.U32(),
			length:   binary.U32(),
		}
		ttf.directorys[name] = directory
		end := int(directory.offset) + int(directory.length)
		if int(directory.offset) < 0 || int(directory.length) < 0 || end > len(data) {
			err = fmt.Errorf("invalid table directory: %s", name)
			return
		}
		table := data[directory.offset:end]

		if name != "head" {
			checkSum := calcTableChecksum(table)
			if checkSum != directory.checkSum {
				err = fmt.Errorf("invalid checksum: %s", name)
				return
			}
		} else {
			tableDataCopy := make([]byte, len(table))
			copy(tableDataCopy, table)
			for i := 0; i < 4; i++ {
				tableDataCopy[8+i] = 0
			}
			checkSum := calcTableChecksum(tableDataCopy)
			if checkSum != directory.checkSum {
				err = fmt.Errorf("invalid checksum: %s", name)
				return
			}
		}

		switch name {
		case "head":
			ttf.head, err = parseHead(table)
		case "OS/2":
			ttf.os2, err = parseOS2(table)
		case "cmap":
			ttf.cmap, err = parseCmap(directory, table)
		case "maxp":
			ttf.maxp, err = parseMaxp(table)
		case "name":
			ttf.name, err = parseName(table)
		case "hhea":
			ttf.hhea, err = parseHhea(table)
		case "post":
			ttf.post, err = parsePost(table)
		case "kern":
			ttf.kern, err = parseKern(table)
		case "GPOS":
			ttf.gpos, err = parseGpos(table)
		case "GSUB":
			ttf.gsub, err = parseGsub(table)
		}
		if err != nil {
			return
		}
	}

	// Parse dependent tables (require data from other tables)
	if dir, ok := ttf.directorys["hmtx"]; ok && ttf.hhea != nil && ttf.maxp != nil {
		table := ttf.getTableData(dir)
		ttf.hmtx, err = parseHmtx(table, int(ttf.hhea.numberOfHMetrics), int(ttf.maxp.numGlyphs))
		if err != nil {
			return
		}
	}

	if dir, ok := ttf.directorys["loca"]; ok && ttf.head != nil && ttf.maxp != nil {
		table := ttf.getTableData(dir)
		ttf.loca, err = parseLoca(table, int(ttf.maxp.numGlyphs), ttf.head.indexToLocFormat)
		if err != nil {
			return
		}
	}

	if ttf.loca != nil {
		if dir, ok := ttf.directorys["glyf"]; ok {
			table := ttf.getTableData(dir)
			ttf.glyf, err = parseGlyf(table, ttf.loca)
			if err != nil {
				return
			}
		}
	}

	return
}

func (ttf *TrueTypeFont) getTableData(dir TableDirectory) []byte {
	end := int(dir.offset) + int(dir.length)
	return ttf.data[dir.offset:end]
}
