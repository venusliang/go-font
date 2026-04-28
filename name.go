package gofont

import "errors"

type NameRecord struct {
	platformID uint16
	encodingID uint16
	languageID uint16
	nameID     uint16
	length     uint16
	offset     uint16
}

type LangTagRecord struct {
	length uint16
	offset uint16
}

type Name struct {
	format         uint16
	count          uint16
	stringOffset   uint16
	nameRecords    []NameRecord
	langTagCount   uint16
	langTagRecords []LangTagRecord
	stringStorage  []byte
}

func parseName(data []byte) (*Name, error) {
	if len(data) < 6 {
		return nil, errors.New("'name' 表长度不足")
	}

	binary := BinaryFrom(data, false)

	name := &Name{}
	name.format = binary.U16()
	name.count = binary.U16()
	name.stringOffset = binary.U16()

	for i := 0; i < int(name.count); i++ {
		record := NameRecord{}
		record.platformID = binary.U16()
		record.encodingID = binary.U16()
		record.languageID = binary.U16()
		record.nameID = binary.U16()
		record.length = binary.U16()
		record.offset = binary.U16()
		name.nameRecords = append(name.nameRecords, record)
	}
	if name.format == 1 {
		name.langTagCount = binary.U16()
		for i := 0; i < int(name.langTagCount); i++ {
			record := LangTagRecord{}
			record.length = binary.U16()
			record.offset = binary.U16()
			name.langTagRecords = append(name.langTagRecords, record)
		}
	}
	raw := data[name.stringOffset:]
	name.stringStorage = make([]byte, len(raw))
	copy(name.stringStorage, raw)

	return name, nil
}

func writeName(name *Name) []byte {
	headerSize := 6
	recordSize := 12 * int(name.count)
	if name.format == 1 {	
		recordSize += 4 * int(name.langTagCount)
	}
	totalSize := headerSize + recordSize + len(name.stringStorage)
	data := make([]byte, totalSize)

	binary := BinaryFrom(data, false)
	binary.PutU16(name.format)
	binary.PutU16(name.count)
	binary.PutU16(name.stringOffset)
	for _, record := range name.nameRecords {
		binary.PutU16(record.platformID)
		binary.PutU16(record.encodingID)
		binary.PutU16(record.languageID)
		binary.PutU16(record.nameID)
		binary.PutU16(record.length)
		binary.PutU16(record.offset)
	}
	if name.format == 1 {
		binary.PutU16(name.langTagCount)
		for _, record := range name.langTagRecords {
			binary.PutU16(record.length)
			binary.PutU16(record.offset)
		}
	}
	copy(data[binary.Offset():], name.stringStorage)

	return data
}
