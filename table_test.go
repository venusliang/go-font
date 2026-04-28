package gofonts

import (
	"testing"
)

func TestCalcTableChecksum(t *testing.T) {
	// Empty table
	if sum := calcTableChecksum(nil); sum != 0 {
		t.Errorf("empty table: got 0x%08X, want 0", sum)
	}

	// 4-byte aligned data
	data := []byte{0x01, 0x02, 0x03, 0x04}
	if sum := calcTableChecksum(data); sum != 0x01020304 {
		t.Errorf("4 bytes: got 0x%08X, want 0x01020304", sum)
	}

	// Unaligned data (3 bytes should be padded to 4)
	data = []byte{0xAA, 0xBB, 0xCC}
	if sum := calcTableChecksum(data); sum != 0xAABBCC00 {
		t.Errorf("3 bytes: got 0x%08X, want 0xAABBCC00", sum)
	}

	// Two 4-byte groups
	data = []byte{0x01, 0x00, 0x01, 0x00, 0x0F, 0xF0, 0x0F, 0xF0}
	expected := uint32(0x01000100) + uint32(0x0FF00FF0)
	if sum := calcTableChecksum(data); sum != expected {
		t.Errorf("8 bytes: got 0x%08X, want 0x%08X", sum, expected)
	}
}

func TestChecksumAgainstDirectory(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	// Verify that our checksum calculation matches the stored directory checksums
	// for a table other than 'head' (head has the checksumAdjustment field zeroed)
	for tableName, dir := range ttf.directorys {
		if tableName == "head" {
			continue
		}
		end := int(dir.offset) + int(dir.length)
		tableData := ttf.data[dir.offset:end]
		sum := calcTableChecksum(tableData)
		if sum != dir.checkSum {
			t.Errorf("%s checksum mismatch: calculated 0x%08X, directory says 0x%08X",
				tableName, sum, dir.checkSum)
		}
	}
}
