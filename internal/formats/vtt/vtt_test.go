// SPDX-License-Identifier: Apache-2.0

package vtt

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

const sample = "WEBVTT\n\n" +
	"intro\n" +
	"00:00:01.000 --> 00:00:04.000 align:start position:25%\n" +
	"<i>Hello</i> &amp; <b>world</b>\n\n" +
	"NOTE this is a comment\n\n" +
	"00:00:05.500 --> 00:00:08.000\n" +
	"<u>Underlined</u> text\n"

func parse(t *testing.T, text string) *model.Document {
	t.Helper()
	doc, err := NewReader().Parse(strings.NewReader(text))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return doc
}

func TestParseSample(t *testing.T) {
	doc := parse(t, sample)
	if len(doc.Cues) != 2 {
		t.Fatalf("got %d cues, want 2", len(doc.Cues))
	}
	first := doc.Cues[0]
	if first.Start != time.Second || first.End != 4*time.Second {
		t.Fatalf("first cue timing = %v..%v", first.Start, first.End)
	}
	if first.Text() != "Hello & world" {
		t.Fatalf("first cue text = %q", first.Text())
	}
	if first.Layout == nil || first.Layout.Anchor == nil || *first.Layout.Anchor != model.AnchorBottomLeft {
		t.Fatalf("align:start = %+v", first.Layout)
	}
	if first.Layout.Position == nil || first.Layout.Position.X != 320 {
		t.Fatalf("position:25%% = %+v", first.Layout.Position)
	}
	if len(first.Spans) != 3 {
		t.Fatalf("first cue spans = %+v", first.Spans)
	}
	if !isTrue(first.Spans[0].Italic) {
		t.Errorf("italic tag missing: %+v", first.Spans[0])
	}
	if !isTrue(first.Spans[2].Bold) {
		t.Errorf("bold tag missing: %+v", first.Spans[2])
	}
	second := doc.Cues[1]
	if !isTrue(second.Spans[0].Underline) {
		t.Errorf("underline tag missing: %+v", second.Spans[0])
	}
}

func TestParseTimestampForms(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		{"00:00:01.000", time.Second},
		{"01:02:03.004", time.Hour + 2*time.Minute + 3*time.Second + 4*time.Millisecond},
		{"02:03.500", 2*time.Minute + 3500*time.Millisecond},
		{"00:00:01.5", time.Second + 500*time.Millisecond},
	}
	for _, tt := range tests {
		got, err := parseTimestamp(tt.in)
		if err != nil {
			t.Errorf("parseTimestamp(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseTimestamp(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseTimestampErrors(t *testing.T) {
	for _, in := range []string{"", "1", "a:00:01.000", "00:a:01.000", "00:00:a.000", "00:00:01.xyz"} {
		if _, err := parseTimestamp(in); err == nil {
			t.Errorf("parseTimestamp(%q) must fail", in)
		}
	}
}

func TestParseTimingErrors(t *testing.T) {
	if _, err := NewReader().Parse(strings.NewReader("WEBVTT\n\n00:00:01.000 -->\n")); err == nil {
		t.Fatal("a missing end time must fail")
	}
	if _, err := NewReader().Parse(strings.NewReader("WEBVTT\n\nbad --> 00:00:02.000\n")); err == nil {
		t.Fatal("a bad start time must fail")
	}
	if _, err := NewReader().Parse(strings.NewReader("WEBVTT\n\n00:00:01.000 --> bad\n")); err == nil {
		t.Fatal("a bad end time must fail")
	}
}

func TestParseWithoutHeader(t *testing.T) {
	doc := parse(t, "00:00:01.000 --> 00:00:02.000\nplain\n")
	if len(doc.Cues) != 1 || doc.Cues[0].Text() != "plain" {
		t.Fatalf("a missing header must still parse: %+v", doc.Cues)
	}
}

func TestParseBrowserMark(t *testing.T) {
	doc := parse(t, "\ufeffWEBVTT\n\n00:00:01.000 --> 00:00:02.000\nplain\n")
	if len(doc.Cues) != 1 {
		t.Fatalf("a byte order mark must be skipped: %d cues", len(doc.Cues))
	}
}

func TestParseUnterminatedTag(t *testing.T) {
	doc := parse(t, "WEBVTT\n\n00:00:01.000 --> 00:00:02.000\na < b\n")
	if got := doc.Cues[0].Text(); got != "a < b" {
		t.Fatalf("a lone angle bracket is literal text: %q", got)
	}
}

func TestAlignAnchorValues(t *testing.T) {
	tests := []struct {
		value string
		want  *model.Anchor
	}{
		{"left", ptr(model.AnchorBottomLeft)},
		{"center", ptr(model.AnchorBottomCentre)},
		{"middle", ptr(model.AnchorBottomCentre)},
		{"right", ptr(model.AnchorBottomRight)},
		{"odd", nil},
	}
	for _, tt := range tests {
		got := alignAnchor(tt.value)
		if tt.want == nil {
			if got != nil {
				t.Errorf("alignAnchor(%q) = %+v, want nil", tt.value, got)
			}
			continue
		}
		if got == nil || *got != *tt.want {
			t.Errorf("alignAnchor(%q) = %+v, want %v", tt.value, got, *tt.want)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	bold, italic, underline := true, true, true
	anchor := model.AnchorBottomLeft
	doc := &model.Document{
		VideoDimensions: model.Point{X: 1280, Y: 720},
		Styles:          []model.Style{DefaultStyle()},
		Cues: []model.Cue{
			{
				Start: time.Second, End: 3 * time.Second,
				Layout: &model.Layout{Anchor: &anchor, Position: &model.Point{X: 320, Y: 0}},
				Spans: []model.TextSpan{
					{Text: "all", Bold: &bold, Italic: &italic, Underline: &underline},
					{Text: " plain"},
				},
			},
			{Start: 4 * time.Second, End: 5 * time.Second, Spans: []model.TextSpan{{Text: "second"}}},
		},
	}
	var out strings.Builder
	losses, err := NewWriter().Render(doc, &out)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(losses) != 0 {
		t.Fatalf("unexpected losses: %v", losses)
	}
	again := parse(t, out.String())
	if len(again.Cues) != len(doc.Cues) {
		t.Fatalf("cue count = %d, want %d", len(again.Cues), len(doc.Cues))
	}
	for i := range doc.Cues {
		if doc.Cues[i].Start != again.Cues[i].Start || doc.Cues[i].End != again.Cues[i].End {
			t.Errorf("cue %d timing changed: %v..%v to %v..%v", i,
				doc.Cues[i].Start, doc.Cues[i].End, again.Cues[i].Start, again.Cues[i].End)
		}
		if doc.Cues[i].Text() != again.Cues[i].Text() {
			t.Errorf("cue %d text changed: %q to %q", i, doc.Cues[i].Text(), again.Cues[i].Text())
		}
	}
	if !isTrue(again.Cues[0].Spans[0].Bold) || !isTrue(again.Cues[0].Spans[0].Italic) || !isTrue(again.Cues[0].Spans[0].Underline) {
		t.Errorf("inline tags did not survive: %+v", again.Cues[0].Spans[0])
	}
	if again.Cues[0].Layout == nil || again.Cues[0].Layout.Position == nil || again.Cues[0].Layout.Position.X != 320 {
		t.Errorf("position did not survive: %+v", again.Cues[0].Layout)
	}
}

func TestWriterLosses(t *testing.T) {
	rtl := model.DirRightToLeft
	sub := model.Script{Kind: model.ScriptSubscript}
	vertical := model.Vertical{Mode: model.VerticalColumnsRTL}
	packed := true
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues: []model.Cue{{
			Start: 0, End: time.Second,
			Layout: &model.Layout{
				Anchor: ptr(model.AnchorTopLeft),
				Fade:   &model.Fade{},
				Move:   &model.Move{},
			},
			Animations: []model.Animation{{Shake: &model.Shake{}}},
			Spans: []model.TextSpan{
				{
					Text: "styled",
					Fore: ptr(model.NewColour(255, 0, 0, 255)), Secondary: ptr(model.NewColour(0, 255, 0, 255)),
					Back:         ptr(model.NewColour(0, 0, 255, 255)),
					Shadows:      []model.Shadow{{Kind: model.ShadowHard, Colour: model.NewColour(0, 0, 0, 255)}},
					OutlineWidth: ptr(2.0), Vertical: &vertical, Script: &sub,
					Direction: &rtl, Packed: &packed,
				},
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
	for _, want := range []string{
		"vertical alignment", "cue fade", "cue move", "animation",
		"foreground colour", "secondary colour", "background colour",
		"shadow effects", "outline width", "vertical text", "script offset",
		"right-to-left marking", "packing", "ruby text",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("loss report is missing %q: %v", want, losses)
		}
	}
	if !strings.Contains(out.String(), "(reading)") {
		t.Errorf("ruby must fall back to brackets:\n%s", out.String())
	}
}

func TestWriterKaraokeLoss(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues:   []model.Cue{{Spans: []model.TextSpan{{Text: "a", Start: 0, End: time.Second}}}},
	}
	var out strings.Builder
	losses, _ := NewWriter().Render(doc, &out)
	if !strings.Contains(strings.Join(losses, "; "), "karaoke timing") {
		t.Fatalf("karaoke loss missing: %v", losses)
	}
}

func TestFormatTime(t *testing.T) {
	if got := formatTime(time.Second + 500*time.Millisecond); got != "00:00:01.500" {
		t.Fatalf("formatTime = %q", got)
	}
	if got := formatTime(-time.Second); got != "00:00:00.000" {
		t.Fatalf("a negative time must clamp: %q", got)
	}
}

func TestWriteError(t *testing.T) {
	doc := &model.Document{Styles: []model.Style{DefaultStyle()}}
	if _, err := NewWriter().Render(doc, failWriter{}); err == nil {
		t.Fatal("a failing sink must surface an error")
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("disk gone") }

type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestParseReadError(t *testing.T) {
	if _, err := NewReader().Parse(failReader{}); err == nil {
		t.Fatal("a failing reader must surface an error")
	}
}

func TestVTTAlign(t *testing.T) {
	tests := []struct {
		anchor model.Anchor
		want   string
	}{
		{model.AnchorBottomLeft, "start"},
		{model.AnchorMiddleLeft, "start"},
		{model.AnchorTopLeft, "start"},
		{model.AnchorBottomCentre, "center"},
		{model.AnchorCentre, "center"},
		{model.AnchorTopCentre, "center"},
		{model.AnchorBottomRight, "end"},
		{model.AnchorMiddleRight, "end"},
		{model.AnchorTopRight, "end"},
		{model.Anchor(0), ""},
	}
	for _, tt := range tests {
		if got := vttAlign(tt.anchor); got != tt.want {
			t.Errorf("vttAlign(%d) = %q, want %q", tt.anchor, got, tt.want)
		}
	}
}

func TestAnchorRow(t *testing.T) {
	tests := []struct {
		anchor model.Anchor
		want   int
	}{
		{model.AnchorBottomLeft, 0},
		{model.AnchorMiddleLeft, 1},
		{model.AnchorTopLeft, 2},
	}
	for _, tt := range tests {
		if got := anchorRow(tt.anchor); got != tt.want {
			t.Errorf("anchorRow(%d) = %d, want %d", tt.anchor, got, tt.want)
		}
	}
}

func TestCueSettingsEmpty(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues:   []model.Cue{{Layout: &model.Layout{}, Spans: []model.TextSpan{{Text: "x"}}}},
	}
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(out.String(), "align:") || strings.Contains(out.String(), "position:") {
		t.Fatalf("an empty layout needs no settings:\n%s", out.String())
	}
}

func TestParseSettingsEdges(t *testing.T) {
	doc := parse(t, "WEBVTT\n\n00:00:01.000 --> 00:00:02.000 align:odd position:abc% vertical\nplain\n")
	if doc.Cues[0].Layout != nil {
		t.Fatalf("bad settings must not change the cue: %+v", doc.Cues[0].Layout)
	}
	doc = parse(t, "WEBVTT\n\n00:00:01.000 --> 00:00:02.000 position:50%\nplain\n")
	if doc.Cues[0].Layout == nil || doc.Cues[0].Layout.Position == nil || doc.Cues[0].Layout.Position.X != 640 {
		t.Fatalf("a position alone must create the layout: %+v", doc.Cues[0].Layout)
	}
}

func TestNames(t *testing.T) {
	if NewReader().Name() != FormatName || NewWriter().Name() != FormatName {
		t.Fatalf("names must be %q", FormatName)
	}
}

// TestParseVoiceSpan covers the voice annotation of the WebVTT spec, which
// names the speaker of the text: <v Roger Bingham>We are in the Milky Way.
func TestParseVoiceSpan(t *testing.T) {
	doc := parse(t, "WEBVTT\n\n00:00:01.000 --> 00:00:02.000\n<v Roger Bingham>We are in the Milky Way.</v>\n")
	span := doc.Cues[0].Spans[0]
	if span.Voice == nil || *span.Voice != "Roger Bingham" {
		t.Fatalf("voice = %+v, want Roger Bingham", span.Voice)
	}
	if span.Text != "We are in the Milky Way." {
		t.Fatalf("text = %q", span.Text)
	}
}

// TestParseVoiceSpanWithoutName keeps a nameless voice annotation, and the
// name it carries is empty.
func TestParseVoiceSpanWithoutName(t *testing.T) {
	doc := parse(t, "WEBVTT\n\n00:00:01.000 --> 00:00:02.000\n<v>nameless</v>\n")
	span := doc.Cues[0].Spans[0]
	if span.Voice == nil || *span.Voice != "" {
		t.Fatalf("voice = %+v, want an empty name", span.Voice)
	}
	// The writer must emit the name-free form, because an empty annotation
	// is not valid WebVTT.
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out.String(), "<v>nameless</v>") {
		t.Fatalf("a nameless voice must write as <v>:\n%s", out.String())
	}
}

// TestVoiceSpanClosesAtEnd covers the text that follows a closed voice
// annotation, which carries no voice of its own.
func TestVoiceSpanClosesAtEnd(t *testing.T) {
	doc := parse(t, "WEBVTT\n\n00:00:01.000 --> 00:00:02.000\n<v Ana>one</v> two\n")
	spans := doc.Cues[0].Spans
	if len(spans) != 2 {
		t.Fatalf("spans = %+v", spans)
	}
	if spans[0].Voice == nil || *spans[0].Voice != "Ana" {
		t.Errorf("first span voice = %+v", spans[0].Voice)
	}
	if spans[1].Voice != nil {
		t.Errorf("text after </v> must carry no voice: %+v", spans[1].Voice)
	}
}

func TestVoiceSpanRoundTrip(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues: []model.Cue{{End: time.Second, Spans: []model.TextSpan{
			{Text: "Hi", Voice: ptr("Roger"), Bold: ptr(true)},
		}}},
	}
	var out strings.Builder
	losses, err := NewWriter().Render(doc, &out)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(losses) != 0 {
		t.Fatalf("WebVTT carries a voice span: %v", losses)
	}
	if !strings.Contains(out.String(), "<v Roger><b>Hi</b></v>") {
		t.Fatalf("the writer must wrap the voice outside the style tags:\n%s", out.String())
	}
	again := parse(t, out.String())
	span := again.Cues[0].Spans[0]
	if span.Voice == nil || *span.Voice != "Roger" || !isTrue(span.Bold) {
		t.Fatalf("the voice and the bold flag must survive: %+v", span)
	}
}
