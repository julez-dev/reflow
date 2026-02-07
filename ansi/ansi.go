package ansi

const Marker = '\x1B'

// IsTerminator reports whether c terminates a CSI escape sequence.
func IsTerminator(c rune) bool {
	return (c >= 0x40 && c <= 0x5a) || (c >= 0x61 && c <= 0x7a)
}

// IsStringSequenceStart reports whether c, following an ESC, begins a
// string-type escape sequence terminated by ST (ESC \).
// These are: DCS (P), OSC (]), SOS (X), PM (^), APC (_).
func IsStringSequenceStart(c rune) bool {
	return c == 'P' || c == ']' || c == 'X' || c == '^' || c == '_'
}
