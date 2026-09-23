package sbv

import (
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

const sampleInput = `0:00:01.000,0:00:04.000
Plain line.

0:00:04.500,0:00:08.000
Two display lines
on one cue.

0:00:08.500,0:00:12.000
Last cue.
`

func TestReaderParseSample(t *testing.T) {
	doc, err := NewReader().Parse(strings.NewReader(sampleInput))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(doc.Cues) != 3 {
		t.Fatalf("got %d cues, want 3", len(doc.Cues))
	}
	first := doc.Cues[0]
	if first.Start != time.Second || first.End != 4*time.Second {
		t.Fatalf("timing wrong: %v..%v", first.Start, first.End)
	}
	second := doc.Cues[1]
	if got := second.Text(); got != "Two display lines\non one cue." {
		t.Fatalf("multi-line text = %q", got)
	}
	if len(doc.Styles) != 1 {
		t.Fatal("reader must emit one implicit style")
	}
}

func TestReaderParseHoursOverNine(t *testing.T) {
	input := "100:00:00.250,100:00:01.000\nLate.\n"
	doc, err := NewReader().Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := 100*time.Hour + 250*time.Millisecond
	if doc.Cues[0].Start != want {
		t.Fatalf("Start = %v, want %v", doc.Cues[0].Start, want)
	}
}

func TestReaderParseBadTiming(t *testing.T) {
	input := "aa:bb.cc,dd:ee.ff\nBroken.\n"
	if _, err := NewReader().Parse(strings.NewReader(input)); err == nil {
		t.Fatal("malformed timing must fail")
	}
}

func TestReaderTextLineWithComma(t *testing.T) {
	// A text line that carries a comma but no timestamp shape must not
	// panic: lenient readers skip it rather than fail the file.
	input := "0:00:01.000,0:00:02.000\nHello, world.\n"
	doc, err := NewReader().Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(doc.Cues) != 1 || doc.Cues[0].Text() != "Hello, world." {
		t.Fatalf("comma text line mishandled: %+v", doc.Cues)
	}
}

func TestWriterRenderRoundTrip(t *testing.T) {
	doc, err := NewReader().Parse(strings.NewReader(sampleInput))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "0:00:01.000,0:00:04.000") {
		t.Fatalf("timing line missing:\n%s", got)
	}
	if !strings.Contains(got, "Two display lines\non one cue.") {
		t.Fatalf("multi-line text not preserved:\n%s", got)
	}
	doc2, err := NewReader().Parse(strings.NewReader(got))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	for i := range doc.Cues {
		if doc.Cues[i].Text() != doc2.Cues[i].Text() {
			t.Fatalf("cue %d text changed: %q vs %q", i, doc.Cues[i].Text(), doc2.Cues[i].Text())
		}
	}
}

func TestWriterLossNotesForStyling(t *testing.T) {
	bold := true
	italic := true
	doc := &model.Document{
		Cues: []model.Cue{{
			Start: time.Second,
			End:   2 * time.Second,
			Spans: []model.TextSpan{
				{Text: "styled", Bold: &bold, Italic: &italic},
				{Text: "kanji"},
				{Text: "reading", Ruby: &model.Ruby{}},
			},
		}},
	}
	var out strings.Builder
	losses, err := NewWriter().Render(doc, &out)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	joined := strings.Join(losses, "; ")
	if !strings.Contains(joined, "inline styling") || !strings.Contains(joined, "ruby") {
		t.Fatalf("loss report incomplete: %v", losses)
	}
	if !strings.Contains(out.String(), "(reading)") {
		t.Fatalf("ruby must fall back to brackets:\n%s", out.String())
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{0, "0:00:00.000"},
		{time.Hour + 500*time.Millisecond, "1:00:00.500"},
		{100*time.Hour + 2*time.Minute + 3*time.Second, "100:02:03.000"},
		{-time.Second, "0:00:00.000"},
	}
	for _, tt := range tests {
		if got := formatTime(tt.in); got != tt.want {
			t.Errorf("formatTime(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
