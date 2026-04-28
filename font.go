package gofonts

// Font is an interface for font types that can parse binary data.
type Font interface {
	Parse([]byte) error
}
