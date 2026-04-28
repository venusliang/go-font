package gofonts

type TableDirectory struct {
	tag      uint32 // table tag name
	checkSum uint32 // checksum for this table
	offset   uint32 // offset from beginning of TrueType font file
	length   uint32 // length of this table in byte
}

func calcTableChecksum(table []byte) uint32 {
	if len(table)%4 != 0 {
		l := 4 - len(table)%4
		pad := make([]byte, l)
		table = append(table, pad...)
	}

	var sum uint32
	for i := 0; i < len(table); i += 4 {
		sum += uint32(table[i])<<24 | uint32(table[i+1])<<16 | uint32(table[i+2])<<8 | uint32(table[i+3])
	}
	return sum
}

type Table interface {
	Read([]byte) error
	Write() []byte
}

// 