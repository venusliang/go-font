package gofont

import "errors"

type Post struct {
	version            uint32
	italicAngle        int32 // Fixed 16.16
	underlinePosition  int16
	underlineThickness int16
	isFixedPitch       uint32
	minMemType42       uint32
	maxMemType42       uint32
	minMemType1        uint32
	maxMemType1        uint32

	// Format 2.0 fields
	numGlyphs      uint16
	glyphNameIndex []uint16
	stringData     []byte // Pascal strings for custom names (index >= 258)
}

func parsePost(data []byte) (*Post, error) {
	if len(data) < 32 {
		return nil, errors.New("post table too small")
	}

	binary := BinaryFrom(data, false)

	post := &Post{
		version:            binary.U32(),
		italicAngle:        int32(binary.U32()),
		underlinePosition:  binary.I16(),
		underlineThickness: binary.I16(),
		isFixedPitch:       binary.U32(),
		minMemType42:       binary.U32(),
		maxMemType42:       binary.U32(),
		minMemType1:        binary.U32(),
		maxMemType1:        binary.U32(),
	}

	if post.version == 0x00020000 {
		if len(data) < 34 {
			return nil, errors.New("post format 2.0 table too small")
		}
		post.numGlyphs = binary.U16()
		post.glyphNameIndex = make([]uint16, post.numGlyphs)
		for i := 0; i < int(post.numGlyphs); i++ {
			post.glyphNameIndex[i] = binary.U16()
		}

		// Read Pascal strings (custom names for indices >= 258)
		maxIdx := uint16(0)
		for _, idx := range post.glyphNameIndex {
			if idx > maxIdx {
				maxIdx = idx
			}
		}
		// Standard Macintosh glyph names go up to index 257
		// Custom names start at index 258
		if maxIdx >= 258 {
			remaining := data[binary.Offset():]
			post.stringData = make([]byte, len(remaining))
			copy(post.stringData, remaining)
		}
	}

	return post, nil
}

func writePost(post *Post) []byte {
	size := 32
	if post.version == 0x00020000 {
		size += 2 + len(post.glyphNameIndex)*2 + len(post.stringData)
	}

	data := make([]byte, size)
	binary := BinaryFrom(data, false)

	binary.PutU32(post.version)
	binary.PutU32(uint32(post.italicAngle))
	binary.PutU16(uint16(post.underlinePosition))
	binary.PutU16(uint16(post.underlineThickness))
	binary.PutU32(post.isFixedPitch)
	binary.PutU32(post.minMemType42)
	binary.PutU32(post.maxMemType42)
	binary.PutU32(post.minMemType1)
	binary.PutU32(post.maxMemType1)

	if post.version == 0x00020000 {
		binary.PutU16(post.numGlyphs)
		for _, idx := range post.glyphNameIndex {
			binary.PutU16(idx)
		}
		if len(post.stringData) > 0 {
			binary.Append(post.stringData)
		}
	}

	return data
}
