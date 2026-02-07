package ansi

import (
	"bytes"
	"io"
	"unicode/utf8"
)

// writerState tracks the ANSI parsing state inside Writer.
type writerState int

const (
	writerGround         writerState = iota
	writerEscape                     // saw ESC
	writerCSI                        // inside CSI (ESC [)
	writerStringSequence             // inside DCS/OSC/APC/PM/SOS
	writerStringEscape               // inside string sequence, saw ESC
)

type Writer struct {
	Forward io.Writer

	state      writerState
	ansiseq    bytes.Buffer
	lastseq    bytes.Buffer
	seqchanged bool
	runeBuf    []byte
}

// Write is used to write content to the ANSI buffer.
func (w *Writer) Write(b []byte) (int, error) {
	for _, c := range string(b) {
		switch w.state {
		case writerGround:
			if c == Marker {
				w.state = writerEscape
				w.seqchanged = true
				_, _ = w.ansiseq.WriteRune(c)
			} else {
				_, err := w.writeRune(c)
				if err != nil {
					return 0, err
				}
			}

		case writerEscape:
			_, _ = w.ansiseq.WriteRune(c)
			if c == '[' {
				w.state = writerCSI
			} else if IsStringSequenceStart(c) {
				w.state = writerStringSequence
				// Flush the ESC + introducer to Forward immediately
				_, _ = w.ansiseq.WriteTo(w.Forward)
			} else if IsTerminator(c) {
				// Two-character escape sequence
				w.state = writerGround
				_, _ = w.ansiseq.WriteTo(w.Forward)
			}

		case writerCSI:
			_, _ = w.ansiseq.WriteRune(c)
			if IsTerminator(c) {
				w.state = writerGround

				if bytes.HasSuffix(w.ansiseq.Bytes(), []byte("[0m")) {
					w.lastseq.Reset()
					w.seqchanged = false
				} else if c == 'm' {
					_, _ = w.lastseq.Write(w.ansiseq.Bytes())
				}

				_, _ = w.ansiseq.WriteTo(w.Forward)
			}

		case writerStringSequence:
			if c == Marker {
				w.state = writerStringEscape
			} else if c == '\a' {
				// BEL terminates OSC
				_, _ = w.Forward.Write([]byte{'\a'})
				w.state = writerGround
			} else {
				// Pass through string sequence content
				_, err := w.writeRuneTo(c, w.Forward)
				if err != nil {
					return 0, err
				}
			}

		case writerStringEscape:
			if c == '\\' {
				// ST = ESC backslash — end of string sequence
				_, _ = w.Forward.Write([]byte{Marker, '\\'})
				w.state = writerGround
			} else if c == Marker {
				// Another ESC, forward previous one
				_, _ = w.Forward.Write([]byte{Marker})
			} else {
				// False alarm, forward the ESC and this char
				_, _ = w.Forward.Write([]byte{Marker})
				_, err := w.writeRuneTo(c, w.Forward)
				if err != nil {
					return 0, err
				}
				w.state = writerStringSequence
			}
		}
	}

	return len(b), nil
}

func (w *Writer) writeRune(r rune) (int, error) {
	return w.writeRuneTo(r, w.Forward)
}

func (w *Writer) writeRuneTo(r rune, dst io.Writer) (int, error) {
	if w.runeBuf == nil {
		w.runeBuf = make([]byte, utf8.UTFMax)
	}
	n := utf8.EncodeRune(w.runeBuf, r)
	return dst.Write(w.runeBuf[:n])
}

func (w *Writer) LastSequence() string {
	return w.lastseq.String()
}

func (w *Writer) ResetAnsi() {
	if !w.seqchanged {
		return
	}
	_, _ = w.Forward.Write([]byte("\x1b[0m"))
}

func (w *Writer) RestoreAnsi() {
	_, _ = w.Forward.Write(w.lastseq.Bytes())
}
