package gofonts

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(binary.BigEndian.Uint16([]byte{0x00, 0x01}))
	fmt.Println(ttf.directorys)
}
