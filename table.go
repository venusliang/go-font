package gofont

type TableDirectory struct {
	tag      uint32 // table tag name
	checkSum uint32 // checksum for this table
	offset   uint32 // offset from beginning of TrueType font file
	length   uint32 // length of this table in byte
}

func calcTableChecksum(table []byte) uint32 {
	var sum uint32
	n := len(table)
	for i := 0; i+3 < n; i += 4 {
		sum += uint32(table[i])<<24 | uint32(table[i+1])<<16 | uint32(table[i+2])<<8 | uint32(table[i+3])
	}
	// Handle trailing bytes with zero-padding
	rem := n % 4
	if rem > 0 {
		start := n - rem
		var last uint32
		for i := 0; i < rem; i++ {
			last |= uint32(table[start+i]) << uint(24-8*i)
		}
		sum += last
	}
	return sum
}

type Table interface {
	Read([]byte) error
	Write() []byte
}

// 