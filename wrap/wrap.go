package wrap

import (
	"bytes"
	"strings"
	"unicode"

	"github.com/julez-dev/reflow/ansi"
	"github.com/mattn/go-runewidth"
)

var (
	defaultNewline  = []rune{'\n'}
	defaultTabWidth = 4
)

type wrapState int

const (
	wGround         wrapState = iota
	wEscape                   // saw ESC
	wCSI                      // inside CSI (ESC [)
	wStringSequence           // inside DCS/OSC/APC/PM/SOS
	wStringEscape             // inside string sequence, saw ESC
)

type Wrap struct {
	Limit         int
	Newline       []rune
	KeepNewlines  bool
	PreserveSpace bool
	TabWidth      int

	buf             *bytes.Buffer
	lineLen         int
	state           wrapState
	forcefulNewline bool
}

// NewWriter returns a new instance of a wrapping writer, initialized with
// default settings.
func NewWriter(limit int) *Wrap {
	return &Wrap{
		Limit:        limit,
		Newline:      defaultNewline,
		KeepNewlines: true,
		// Keep whitespaces following a forceful line break. If disabled,
		// leading whitespaces in a line are only kept if the line break
		// was not forceful, meaning a line break that was already present
		// in the input
		PreserveSpace: false,
		TabWidth:      defaultTabWidth,

		buf: &bytes.Buffer{},
	}
}

// Bytes is shorthand for declaring a new default Wrap instance,
// used to immediately wrap a byte slice.
func Bytes(b []byte, limit int) []byte {
	f := NewWriter(limit)
	_, _ = f.Write(b)

	return f.Bytes()
}

func (w *Wrap) addNewLine() {
	_, _ = w.buf.WriteRune('\n')
	w.lineLen = 0
}

// String is shorthand for declaring a new default Wrap instance,
// used to immediately wrap a string.
func String(s string, limit int) string {
	return string(Bytes([]byte(s), limit))
}

func (w *Wrap) Write(b []byte) (int, error) {
	s := strings.Replace(string(b), "\t", strings.Repeat(" ", w.TabWidth), -1)
	if !w.KeepNewlines {
		s = strings.Replace(s, "\n", "", -1)
	}

	width := ansi.PrintableRuneWidth(s)

	if w.Limit <= 0 || w.lineLen+width <= w.Limit {
		w.lineLen += width
		return w.buf.Write(b)
	}

	for _, c := range s {
		switch w.state {
		case wGround:
			if c == ansi.Marker {
				w.state = wEscape
				_, _ = w.buf.WriteRune(c)
			} else if inGroup(w.Newline, c) {
				w.addNewLine()
				w.forcefulNewline = false
			} else {
				width := runewidth.RuneWidth(c)

				if w.lineLen+width > w.Limit {
					w.addNewLine()
					w.forcefulNewline = true
				}

				if w.lineLen == 0 {
					if w.forcefulNewline && !w.PreserveSpace && unicode.IsSpace(c) {
						continue
					}
				} else {
					w.forcefulNewline = false
				}

				w.lineLen += width
				_, _ = w.buf.WriteRune(c)
			}

		case wEscape:
			_, _ = w.buf.WriteRune(c)
			if c == '[' {
				w.state = wCSI
			} else if ansi.IsStringSequenceStart(c) {
				w.state = wStringSequence
			} else if ansi.IsTerminator(c) {
				w.state = wGround
			}

		case wCSI:
			_, _ = w.buf.WriteRune(c)
			if ansi.IsTerminator(c) {
				w.state = wGround
			}

		case wStringSequence:
			_, _ = w.buf.WriteRune(c)
			if c == ansi.Marker {
				w.state = wStringEscape
			} else if c == '\a' {
				w.state = wGround
			}

		case wStringEscape:
			_, _ = w.buf.WriteRune(c)
			if c == '\\' {
				w.state = wGround
			} else if c == ansi.Marker {
				// Another ESC, stay
			} else {
				w.state = wStringSequence
			}
		}
	}

	return len(b), nil
}

// Bytes returns the wrapped result as a byte slice.
func (w *Wrap) Bytes() []byte {
	return w.buf.Bytes()
}

// String returns the wrapped result as a string.
func (w *Wrap) String() string {
	return w.buf.String()
}

func inGroup(a []rune, c rune) bool {
	for _, v := range a {
		if v == c {
			return true
		}
	}
	return false
}
