package ansi

import (
	"bytes"

	"github.com/mattn/go-runewidth"
)

// Buffer is a buffer aware of ANSI escape sequences.
type Buffer struct {
	bytes.Buffer
}

// PrintableRuneWidth returns the cell width of all printable runes in the
// buffer.
func (w Buffer) PrintableRuneWidth() int {
	return PrintableRuneWidth(w.String())
}

// PrintableRuneWidth returns the cell width of the given string.
func PrintableRuneWidth(s string) int {
	const (
		ground         = iota
		escape         // saw ESC, waiting for next char
		csiSequence    // inside CSI sequence (ESC [), waiting for terminator
		stringSequence // inside DCS/OSC/APC/PM/SOS, waiting for ST (ESC \)
		stringEscape   // inside string sequence, saw ESC, checking for backslash
	)

	var n int
	state := ground

	for _, c := range s {
		switch state {
		case ground:
			if c == Marker {
				state = escape
			} else {
				n += runewidth.RuneWidth(c)
			}
		case escape:
			if c == '[' {
				state = csiSequence
			} else if IsStringSequenceStart(c) {
				state = stringSequence
			} else if IsTerminator(c) {
				// Two-character escape (e.g. ESC c), done
				state = ground
			}
			// else: intermediate byte, stay in escape
		case csiSequence:
			if IsTerminator(c) {
				state = ground
			}
		case stringSequence:
			if c == Marker {
				state = stringEscape
			} else if c == '\a' {
				// BEL terminates OSC sequences
				state = ground
			}
		case stringEscape:
			if c == '\\' {
				// ST (String Terminator) = ESC backslash
				state = ground
			} else if c == Marker {
				// Another ESC — stay in stringEscape
			} else {
				// False alarm, back to string sequence
				state = stringSequence
			}
		}
	}

	return n
}
