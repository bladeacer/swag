package srt

import (
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

const sampleInput = `1
00:00:01,000 --> 00:00:04,000
Plain line.

2
00:00:04,500 --> 00:00:08,000
<i>Italic</i> and <b>bold</b> mixed.

3
00:00:08,500 --> 00:00:12,000
Multi-line
second line.
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
	if text := first.Text(); text != "Plain line." {
		t.Fatalf("text = %q", text)
	}
	second := doc.Cues[1]
	if len(second.Spans) != 4 {
		t.Fatalf("got %d spans, want 4 (italic, plain, bold, plain): %+v", len(second.Spans), second.Spans)
	}
	if !isTrue(second.Spans[0].Italic) || !isTrue(second.Spans[2].Bold) {
		t.Fatalf("inline tags not applied: %+v", second.Spans)
	}
	if third := doc.Cues[2]; len(third.Spans) != 2 {
		t.Fatalf("multi-line cues must keep two spans: %+v", third.Spans)
	}
	if len(doc.Styles) != 1 {
		t.Fatalf("reader must emit one implicit style")
	}
}

func TestReaderParseHourOverFlow(t *testing.T) {
	input := "1\n100:00:00,500 --> 100:00:01,000\nLate.\n"
	doc, err := NewReader().Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := 100*time.Hour + 500*time.Millisecond
	if doc.Cues[0].Start != want {
		t.Fatalf("Start = %v, want %v", doc.Cues[0].Start, want)
	}
}

func TestReaderParseCounterOptional(t *testing.T) {
	input := "00:00:01,000 --> 00:00:02,000\nNo counter.\n"
	doc, err := NewReader().Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(doc.Cues) != 1 || doc.Cues[0].Text() != "No counter." {
		t.Fatalf("cue without counter misread: %+v", doc.Cues)
	}
}

func TestReaderParseBadTiming(t *testing.T) {
	input := "1\nnot a time --> also not\nBroken.\n"
	if _, err := NewReader().Parse(strings.NewReader(input)); err == nil {
		t.Fatal("bad timing must fail")
	}
}

func TestSplitTagsToggleRuns(t *testing.T) {
	spans := SplitTags("<b><i>x</i></b> tail")
	if len(spans) != 2 {
		t.Fatalf("got %d spans, want 2: %+v", len(spans), spans)
	}
	if !isTrue(spans[0].Bold) || !isTrue(spans[0].Italic) || spans[0].Text != "x" {
		t.Fatalf("first span wrong: %+v", spans[0])
	}
	if spans[1].Text != " tail" || isTrue(spans[1].Bold) {
		t.Fatalf("second span wrong: %+v", spans[1])
	}
}

func TestSplitTagsUnclosedTag(t *testing.T) {
	spans := SplitTags("<i>never closed")
	if len(spans) != 1 || !isTrue(spans[0].Italic) {
		t.Fatalf("unclosed tag must keep the style to the end: %+v", spans)
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
	if !strings.Contains(got, "00:00:01,000 --> 00:00:04,000") {
		t.Fatalf("timing line missing:\n%s", got)
	}
	if !strings.Contains(got, "<i>Italic</i> and <b>bold</b> mixed.") {
		t.Fatalf("inline tags not re-emitted:\n%s", got)
	}
	if !strings.Contains(got, "Multi-line\nsecond line.") {
		t.Fatalf("multi-line text not preserved:\n%s", got)
	}
	// Round-trip: rendering the rendered output parses to the same text.
	doc2, err := NewReader().Parse(strings.NewReader(got))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	for i := range doc.Cues {
		if doc.Cues[i].Text() != doc2.Cues[i].Text() {
			t.Fatalf("cue %d text changed: %q vs %q", i, doc.Cues[i].Text(), doc2.Cues[i].Text())
		}
		if doc.Cues[i].Start != doc2.Cues[i].Start || doc.Cues[i].End != doc2.Cues[i].End {
			t.Fatalf("cue %d timing changed", i)
		}
	}
}

func TestWriterLossNotes(t *testing.T) {
	ruby := model.Ruby{Position: model.RubyOver}
	doc := &model.Document{
		Cues: []model.Cue{{
			Start: time.Second,
			End:   2 * time.Second,
			Spans: []model.TextSpan{
				{Text: "base", Start: time.Second},
				{Text: "reading", Ruby: &ruby},
			},
			Layout: &model.Layout{},
		}},
	}
	var out strings.Builder
	losses, err := NewWriter().Render(doc, &out)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	joined := strings.Join(losses, "; ")
	for _, want := range []string{"karaoke", "positioning", "ruby"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("loss report missing %q: %v", want, losses)
		}
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
		{0, "00:00:00,000"},
		{time.Second + 500*time.Millisecond, "00:00:01,500"},
		{time.Hour + 2*time.Minute + 3*time.Second + 4*time.Millisecond, "01:02:03,004"},
		{-time.Second, "00:00:00,000"},
	}
	for _, tt := range tests {
		if got := formatTime(tt.in); got != tt.want {
			t.Errorf("formatTime(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
