package gofont

import (
	"encoding/binary"
	"errors"
)

const eotMagicNumber uint16 = 0x504C

const (
	eotFlagCompressed = 0x00000004 // TTEMBED_TTCOMPRESSED (MTX, not supported)
	eotFlagXOREncrypt = 0x10000000 // TTEMBED_XORENCRYPTDATA
)

// ParseEOT parses an EOT (Embedded OpenType) file and returns a TrueTypeFont.
func ParseEOT(data []byte) (TrueTypeFont, error) {
	if len(data) < 82 {
		return TrueTypeFont{}, errors.New("EOT file too small")
	}

	le := binary.LittleEndian

	eotSize := le.Uint32(data[0:4])
	fontDataSize := le.Uint32(data[4:8])
	version := le.Uint32(data[8:12])
	flags := le.Uint32(data[12:16])
	magicNumber := le.Uint16(data[34:36])

	if magicNumber != eotMagicNumber {
		return TrueTypeFont{}, errors.New("invalid EOT magic number")
	}

	if version != 0x00010000 && version != 0x00020001 && version != 0x00020002 {
		return TrueTypeFont{}, errors.New("unsupported EOT version")
	}

	if flags&eotFlagCompressed != 0 {
		return TrueTypeFont{}, errors.New("EOT MTX compression not supported")
	}

	// Skip variable-length name fields
	off := uint32(82) // past fixed header + Padding1

	readNameField := func() error {
		if off+2 > uint32(len(data)) {
			return errors.New("EOT: name size out of bounds")
		}
		nameSize := le.Uint16(data[off : off+2])
		off += 2
		if uint32(nameSize)+off > uint32(len(data)) {
			return errors.New("EOT: name data out of bounds")
		}
		off += uint32(nameSize)
		// Padding after name
		if off+2 > uint32(len(data)) {
			return errors.New("EOT: padding out of bounds")
		}
		off += 2
		return nil
	}

	// FamilyName
	if err := readNameField(); err != nil {
		return TrueTypeFont{}, err
	}
	// StyleName
	if err := readNameField(); err != nil {
		return TrueTypeFont{}, err
	}
	// VersionName
	if err := readNameField(); err != nil {
		return TrueTypeFont{}, err
	}
	// FullName
	if err := readNameField(); err != nil {
		return TrueTypeFont{}, err
	}
	// RootString
	if err := readNameField(); err != nil {
		return TrueTypeFont{}, err
	}

	// Version 0x00020001 and 0x00020002 have additional fields
	if version >= 0x00020001 {
		if off+12 > uint32(len(data)) {
			return TrueTypeFont{}, errors.New("EOT: extended fields out of bounds")
		}
		// RootStringCheckSum (4) + EUDCCodePage (4) + Padding6 (2) + SignatureSize (2)
		signatureSize := le.Uint16(data[off+10 : off+12])
		off += 12
		if uint32(signatureSize)+off > uint32(len(data)) {
			return TrueTypeFont{}, errors.New("EOT: signature data out of bounds")
		}
		off += uint32(signatureSize)
		if off+8 > uint32(len(data)) {
			return TrueTypeFont{}, errors.New("EOT: EUDC fields out of bounds")
		}
		eudcFlags := le.Uint32(data[off : off+4])
		eudcFontSize := le.Uint32(data[off+4 : off+8])
		off += 8
		_ = eudcFlags
		off += eudcFontSize
	}

	// Extract FontData from end of EOT
	fontDataStart := eotSize - fontDataSize
	if fontDataStart != off {
		// The font data may not align exactly with our calculated offset
		// Trust EOTSize - FontDataSize as the authoritative position
	}
	if fontDataStart+fontDataSize > uint32(len(data)) {
		return TrueTypeFont{}, errors.New("EOT: font data out of bounds")
	}

	fontData := make([]byte, fontDataSize)
	copy(fontData, data[fontDataStart:fontDataStart+fontDataSize])

	// XOR decryption if needed
	if flags&eotFlagXOREncrypt != 0 {
		xorDecrypt(fontData)
	}

	return Parse(fontData)
}

// SerializeEOT serializes the font as an EOT (Embedded OpenType) file.
func (ttf *TrueTypeFont) SerializeEOT() ([]byte, error) {
	ttfData, err := ttf.Serialize()
	if err != nil {
		return nil, err
	}

	// Encode name strings as UTF-16LE
	familyName := encodeUTF16LE(ttf.nameStringForID(1))
	styleName := encodeUTF16LE(ttf.nameStringForID(2))
	versionName := encodeUTF16LE(ttf.nameStringForID(5))
	fullName := encodeUTF16LE(ttf.nameStringForID(4))

	// Calculate total size
	// Fixed header (82) + 5 name fields (each: 2 size + data + 2 padding)
	namesSize := (2 + len(familyName) + 2) +
		(2 + len(styleName) + 2) +
		(2 + len(versionName) + 2) +
		(2 + len(fullName) + 2) +
		(2 + 0 + 2) // RootString (empty)
	eotSize := uint32(82 + namesSize + len(ttfData))

	// Build EOT data
	eotData := make([]byte, eotSize)
	le := binary.LittleEndian

	le.PutUint32(eotData[0:4], eotSize)
	le.PutUint32(eotData[4:8], uint32(len(ttfData)))
	le.PutUint32(eotData[8:12], 0x00010000) // version
	le.PutUint32(eotData[12:16], 0)         // flags (no compression, no encryption)

	// FontPANOSE from OS/2
	if ttf.os2 != nil {
		copy(eotData[16:26], ttf.os2.panose[:])
	}

	// Charset: default to ANSI (0x01)
	eotData[26] = 0x01

	// Italic from head.macStyle
	if ttf.head != nil && ttf.head.macStyle&uint16(MacStyleItalic) != 0 {
		eotData[27] = 1
	}

	// Weight from OS/2
	if ttf.os2 != nil {
		le.PutUint32(eotData[28:32], uint32(ttf.os2.usWeightClass))
		le.PutUint16(eotData[32:34], ttf.os2.fsType)
	}

	// Magic number
	le.PutUint16(eotData[34:36], eotMagicNumber)

	// Unicode ranges from OS/2
	if ttf.os2 != nil {
		le.PutUint32(eotData[36:40], ttf.os2.ulUnicodeRange1)
		le.PutUint32(eotData[40:44], ttf.os2.ulUnicodeRange2)
		le.PutUint32(eotData[44:48], ttf.os2.ulUnicodeRange3)
		le.PutUint32(eotData[48:52], ttf.os2.ulUnicodeRange4)
		le.PutUint32(eotData[52:56], ttf.os2.ulCodePageRange1)
		le.PutUint32(eotData[56:60], ttf.os2.ulCodePageRange2)
	}

	// CheckSumAdjustment from head
	if ttf.head != nil {
		le.PutUint32(eotData[60:64], ttf.head.checksumAdjustment)
	}

	// Reserved (64-79): already zero
	// Padding1 (80-81): already zero

	// Write variable name fields
	off := 82
	writeNameField := func(nameData []byte) {
		binary.LittleEndian.PutUint16(eotData[off:], uint16(len(nameData)))
		off += 2
		copy(eotData[off:], nameData)
		off += len(nameData)
		off += 2 // padding
	}

	writeNameField(familyName)
	writeNameField(styleName)
	writeNameField(versionName)
	writeNameField(fullName)
	writeNameField(nil) // RootString (empty)

	// Copy TTF data
	copy(eotData[off:], ttfData)

	return eotData, nil
}

// encodeUTF16LE encodes a Go string to little-endian UTF-16 bytes.
func encodeUTF16LE(s string) []byte {
	out := make([]byte, 0, len(s)*2)
	for _, r := range s {
		if r <= 0xFFFF {
			out = append(out, byte(r), byte(r>>8))
		}
	}
	return out
}

// xorDecrypt applies XOR 0x50 decryption in place.
func xorDecrypt(data []byte) {
	for i := range data {
		data[i] ^= 0x50
	}
}
