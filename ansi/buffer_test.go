package ansi

import (
	"bytes"
	"testing"
)

func TestBuffer_PrintableRuneWidth(t *testing.T) {
	t.Parallel()

	var bb bytes.Buffer
	bb.WriteString("\x1B[38;2;249;38;114mfoo")
	b := Buffer{bb}

	if n := b.PrintableRuneWidth(); n != 3 {
		t.Fatalf("width should be 3, got %d", n)
	}
}

func TestPrintableRuneWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "plain text",
			input: "hello",
			want:  5,
		},
		{
			name:  "CSI color sequence",
			input: "\x1B[38;2;249;38;114mfoo\x1B[0m",
			want:  3,
		},
		{
			name:  "DCS sequence skipped",
			input: "\x1BPq#0;2;0;0;0#1;2;100;100;0-\x1B\\foo",
			want:  3,
		},
		{
			name:  "OSC sequence with ST",
			input: "\x1B]8;;https://example.com\x1B\\link\x1B]8;;\x1B\\",
			want:  4,
		},
		{
			name:  "OSC sequence with BEL",
			input: "\x1B]0;window title\afoo",
			want:  3,
		},
		{
			name:  "sixel image data has zero width",
			input: "\x1BPq\n#0;2;0;0;0#1;2;100;100;0\n#0~~@@vv@@~~@@~~$\n#1!14telestrings\n-\n#0!!@@vv@@!!@@!!$\n#1!14stealth\n\x1B\\",
			want:  0,
		},
		{
			name:  "text before and after DCS",
			input: "ab\x1BPqdata\x1B\\cd",
			want:  4,
		},
		{
			name:  "APC sequence skipped",
			input: "\x1B_application data\x1B\\text",
			want:  4,
		},
		{
			name:  "PM sequence skipped",
			input: "\x1B^privacy message\x1B\\text",
			want:  4,
		},
		{
			name:  "SOS sequence skipped",
			input: "\x1BXsome string\x1B\\text",
			want:  4,
		},
		{
			name:  "mixed CSI and DCS",
			input: "\x1B[1mfoo\x1BPqsixeldata\x1B\\\x1B[0mbar",
			want:  6,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := PrintableRuneWidth(tc.input); got != tc.want {
				t.Errorf("PrintableRuneWidth() = %d, want %d", got, tc.want)
			}
		})
	}
}

// go test -bench=Benchmark_PrintableRuneWidth -benchmem -count=4
func Benchmark_PrintableRuneWidth(b *testing.B) {
	s := "\x1B[38;2;249;38;114mfoo"

	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()
		b.ResetTimer()
		for pb.Next() {
			PrintableRuneWidth(s)
		}
	})
}
