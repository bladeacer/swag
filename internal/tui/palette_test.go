// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

func TestQueries(t *testing.T) {
	got := queries()
	for i := 0; i < indexedColours; i++ {
		want := fmt.Sprintf("%s4;%d;?%s", oscIntroducer, i, oscTerminator)
		if !strings.Contains(got, want) {
			t.Errorf("the query is missing the indexed colour %d: %q", i, got)
		}
	}
	for _, want := range []string{
		oscIntroducer + "10;?" + oscTerminator,
		oscIntroducer + "11;?" + oscTerminator,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the query is missing %q", want)
		}
	}
	if n := strings.Count(got, oscIntroducer); n != indexedColours+2 {
		t.Errorf("the query carries %d sequences, want %d", n, indexedColours+2)
	}
}

func TestPlainPalette(t *testing.T) {
	palette := PlainPalette()
	if palette.Known {
		t.Error("the plain palette must not report a terminal answer")
	}
	if palette.Foreground != model.NewColour(255, 255, 255, 255) {
		t.Errorf("the foreground = %v, want white", palette.Foreground)
	}
	if palette.Background != model.NewColour(0, 0, 0, 255) {
		t.Errorf("the background = %v, want black", palette.Background)
	}
}

// TestProbeReadsThePalette covers the happy path: a terminal answers the
// indexed colour, the foreground, and the background.
func TestProbeReadsThePalette(t *testing.T) {
	input := oscIntroducer + "4;1;rgb:ffff/0000/0000" + oscTerminator +
		oscIntroducer + "10;rgb:ffff/ffff/ffff" + "\x07" +
		oscIntroducer + "11;rgb:1010/2020/3030" + oscTerminator
	var out strings.Builder
	palette := Probe(strings.NewReader(input), &out, time.Second)

	if !palette.Known {
		t.Fatal("the answer must mark the palette as known")
	}
	if palette.Foreground != model.NewColour(255, 255, 255, 255) {
		t.Errorf("the foreground = %v, want white", palette.Foreground)
	}
	if palette.Background != model.NewColour(0x10, 0x20, 0x30, 255) {
		t.Errorf("the background = %v, want #102030", palette.Background)
	}
	if palette.Indexed[1] != model.NewColour(255, 0, 0, 255) {
		t.Errorf("the indexed colour 1 = %v, want red", palette.Indexed[1])
	}
	if !strings.Contains(out.String(), oscIntroducer+"4;0;?"+oscTerminator) {
		t.Errorf("the probe must write the queries: %q", out.String())
	}
}

func TestProbeWriteError(t *testing.T) {
	palette := Probe(strings.NewReader(""), failWriter{}, time.Second)
	if palette != PlainPalette() {
		t.Errorf("a failed write must return the plain palette: %+v", palette)
	}
}

// TestProbeFallsBackWhenTheTerminalIsSilent covers the plain fallback: a
// terminal that ignores the query never answers, and the probe must not
// wait longer than the timeout.
func TestProbeFallsBackWhenTheTerminalIsSilent(t *testing.T) {
	in := newBlockingReader()
	defer in.Release()
	var out strings.Builder
	palette := Probe(in, &out, 20*time.Millisecond)
	if palette != PlainPalette() {
		t.Errorf("a silent terminal must return the plain palette: %+v", palette)
	}
}

// TestProbeKeepsAPartialAnswer covers a terminal that answers the foreground
// and then stays quiet. The timeout must not throw the answer away.
func TestProbeKeepsAPartialAnswer(t *testing.T) {
	in := &tailReader{
		data:    oscIntroducer + "10;rgb:ffff/ffff/ffff" + oscTerminator,
		release: make(chan struct{}),
	}
	defer close(in.release)
	var out strings.Builder
	palette := Probe(in, &out, 100*time.Millisecond)
	if !palette.Known {
		t.Fatal("a partial answer must still count")
	}
	if palette.Foreground != model.NewColour(255, 255, 255, 255) {
		t.Errorf("the foreground = %v, want white", palette.Foreground)
	}
	if palette.Background != PlainPalette().Background {
		t.Errorf("the background = %v, want the plain fallback", palette.Background)
	}
}

func TestScanReplies(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []reply
	}{
		{
			name:  "empty",
			input: "",
			want:  nil,
		},
		{
			name:  "foreground and background",
			input: oscIntroducer + "10;rgb:ffff/ffff/ffff" + oscTerminator + oscIntroducer + "11;rgb:0000/0000/0000\x07",
			want: []reply{
				{code: 10, value: "rgb:ffff/ffff/ffff"},
				{code: 11, value: "rgb:0000/0000/0000"},
			},
		},
		{
			name:  "indexed only",
			input: oscIntroducer + "4;1;rgb:ffff/0000/0000" + oscTerminator,
			want:  []reply{{code: 4, index: 1, value: "rgb:ffff/0000/0000"}},
		},
		{
			// Junk, a stray escape, and a real answer.
			name:  "junk before the answer",
			input: "junk\x1bzmore" + oscIntroducer + "10;rgb:0/0/0" + oscTerminator,
			want:  []reply{{code: 10, value: "rgb:0/0/0"}},
		},
		{
			name:  "a bare escape",
			input: "\x1b",
			want:  nil,
		},
		{
			name:  "an unterminated body",
			input: oscIntroducer + "4;0;rgb:0/0/0",
			want:  nil,
		},
		{
			// A title is not one of our queries, so it yields no answer.
			name:  "another sequence",
			input: oscIntroducer + "0;title\x07" + oscIntroducer + "4;2;rgb:0/0/0" + oscTerminator,
			want:  []reply{{code: 4, index: 2, value: "rgb:0/0/0"}},
		},
		{
			name:  "an indexed answer without a value",
			input: oscIntroducer + "4;1\x07",
			want:  nil,
		},
		{
			name:  "an answer without a code",
			input: oscIntroducer + "x;1\x07",
			want:  nil,
		},
		{
			name:  "an answer with one field",
			input: oscIntroducer + "10\x07",
			want:  nil,
		},
		{
			name:  "an indexed answer with a bad index",
			input: oscIntroducer + "4;x;rgb:0/0/0" + oscTerminator,
			want:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectReplies(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("replies = %+v, want %+v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("replies = %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestComplete(t *testing.T) {
	tests := []struct {
		name string
		in   []reply
		want bool
	}{
		{"no answers", nil, false},
		{"foreground", []reply{{code: 10}}, false},
		{"background", []reply{{code: 11}}, false},
		{"indexed", []reply{{code: 4}}, false},
		{"both", []reply{{code: 4}, {code: 10}, {code: 11}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := complete(tt.in); got != tt.want {
				t.Errorf("complete(%+v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseReply(t *testing.T) {
	tests := []struct {
		body string
		want reply
		ok   bool
	}{
		{"10;rgb:ffff/ffff/ffff", reply{code: 10, value: "rgb:ffff/ffff/ffff"}, true},
		{"11;rgb:0/0/0", reply{code: 11, value: "rgb:0/0/0"}, true},
		{"4;3;rgb:0/0/0", reply{code: 4, index: 3, value: "rgb:0/0/0"}, true},
		{"10", reply{}, false},
		{"x;rgb:0/0/0", reply{}, false},
		{"12;rgb:0/0/0", reply{}, false},
		{"4;1", reply{}, false},
		{"4;x;rgb:0/0/0", reply{}, false},
	}
	for _, tt := range tests {
		got, ok := parseReply(tt.body)
		if ok != tt.ok || got != tt.want {
			t.Errorf("parseReply(%q) = (%+v, %v), want (%+v, %v)", tt.body, got, ok, tt.want, tt.ok)
		}
	}
}

func TestApplyIgnoresABadValue(t *testing.T) {
	palette := PlainPalette()
	palette.apply(reply{code: 10, value: "nope"})
	if palette.Known || palette != PlainPalette() {
		t.Errorf("a value that does not parse changed the palette: %+v", palette)
	}
}

func TestApplyReadsTheForegroundAndTheBackground(t *testing.T) {
	palette := PlainPalette()
	palette.apply(reply{code: 10, value: "rgb:ffff/0000/0000"})
	palette.apply(reply{code: 11, value: "rgb:0000/ffff/0000"})
	if !palette.Known {
		t.Error("the answers must mark the palette as known")
	}
	if palette.Foreground != model.NewColour(255, 0, 0, 255) {
		t.Errorf("the foreground = %v, want red", palette.Foreground)
	}
	if palette.Background != model.NewColour(0, 255, 0, 255) {
		t.Errorf("the background = %v, want green", palette.Background)
	}
}

func TestApplyReadsAnIndexedColour(t *testing.T) {
	palette := PlainPalette()
	palette.apply(reply{code: 4, index: 3, value: "#0000ff"})
	if !palette.Known {
		t.Error("the answer must mark the palette as known")
	}
	if palette.Indexed[3] != model.NewColour(0, 0, 255, 255) {
		t.Errorf("the indexed colour 3 = %v, want blue", palette.Indexed[3])
	}
}

func TestApplyIgnoresAnIndexOutOfRange(t *testing.T) {
	for _, index := range []int{-1, indexedColours} {
		palette := PlainPalette()
		palette.apply(reply{code: 4, index: index, value: "#ffffff"})
		if !palette.Known {
			t.Errorf("index %d: the answer must still count", index)
		}
		for i, colour := range palette.Indexed {
			if colour != (model.Colour{}) {
				t.Errorf("index %d: the palette gained a colour at %d", index, i)
			}
		}
	}
}

func TestParseColour(t *testing.T) {
	tests := []struct {
		value string
		want  model.Colour
		ok    bool
	}{
		{"rgb:ffff/ffff/ffff", model.NewColour(255, 255, 255, 255), true},
		{"rgb:ff/00/80", model.NewColour(255, 0, 128, 255), true},
		{"rgb:f/0/0", model.NewColour(255, 0, 0, 255), true},
		{" rgb:0000/0000/0000 ", model.NewColour(0, 0, 0, 255), true},
		{"#010203", model.NewColour(1, 2, 3, 255), true},
		{"#f00", model.NewColour(255, 0, 0, 255), true},
		{"#ff00", model.Colour{}, false},
		{"rgb:1/2", model.Colour{}, false},
		{"rgb:x/0/0", model.Colour{}, false},
		{"rgb:0/x/0", model.Colour{}, false},
		{"rgb:0/0/x", model.Colour{}, false},
		{"rgb:/0/0", model.Colour{}, false},
		{"rgb:12345/0/0", model.Colour{}, false},
		{"nonsense", model.Colour{}, false},
	}
	for _, tt := range tests {
		got, ok := parseColour(tt.value)
		if ok != tt.ok || got != tt.want {
			t.Errorf("parseColour(%q) = (%v, %v), want (%v, %v)", tt.value, got, ok, tt.want, tt.ok)
		}
	}
}

// collectReplies runs scanReplies over input and returns every answer it
// sent.
func collectReplies(input string) []reply {
	answers := make(chan reply, replyBuffer)
	go func() {
		defer close(answers)
		scanReplies(strings.NewReader(input), answers)
	}()
	var got []reply
	for r := range answers {
		got = append(got, r)
	}
	return got
}

// blockingReader blocks every read until the test calls Release. It stands
// in for a terminal that ignores the query.
type blockingReader struct{ release chan struct{} }

func newBlockingReader() *blockingReader {
	return &blockingReader{release: make(chan struct{})}
}

func (b *blockingReader) Read([]byte) (int, error) {
	<-b.release
	return 0, io.EOF
}

func (b *blockingReader) Release() { close(b.release) }

// tailReader returns its data and then blocks, which stands in for a
// terminal that answers only part of the query.
type tailReader struct {
	data    string
	release chan struct{}
}

func (r *tailReader) Read(p []byte) (int, error) {
	if r.data != "" {
		n := copy(p, r.data)
		r.data = r.data[n:]
		return n, nil
	}
	<-r.release
	return 0, io.EOF
}
