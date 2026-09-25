// SPDX-License-Identifier: Apache-2.0

package ytt

import (
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

func TestParseHexColourBodyError(t *testing.T) {
	if _, err := parseHexColour("#GGGGGG"); err == nil {
		t.Fatal("a six character non-hex body must fail")
	}
}

func TestParseOpacityNumberError(t *testing.T) {
	if _, err := parseOpacity("abc"); err == nil {
		t.Fatal("a non-numeric opacity must fail")
	}
}

// TestWriterBoxBackground covers the background box branch of the pen
// builder. A style with Box set and no shadow renders a background colour.
func TestWriterBoxBackground(t *testing.T) {
	box := DefaultStyle()
	box.Box = true
	box.OutlineWidth = 2
	doc := &model.Document{
		Styles: []model.Style{box},
		Cues:   []model.Cue{{Spans: []model.TextSpan{{Text: "boxed"}}}},
	}
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out.String(), `bc="`) {
		t.Fatalf("a box style must write a background colour:\n%s", out.String())
	}
}

// TestWriterGlowOutline covers the glow edge branch. A style with an outline
// width and no shadow writes the outline as a glow edge.
func TestWriterGlowOutline(t *testing.T) {
	glow := DefaultStyle()
	glow.OutlineWidth = 2
	doc := &model.Document{
		Styles: []model.Style{glow},
		Cues:   []model.Cue{{Spans: []model.TextSpan{{Text: "outlined"}}}},
	}
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out.String(), `et="3"`) {
		t.Fatalf("an outline must write the glow edge type:\n%s", out.String())
	}
}

func parse(t *testing.T, input string) (*model.Document, error) {
	t.Helper()
	return NewReader().Parse(strings.NewReader(input))
}

func TestReaderBreakAfterExplicitSpan(t *testing.T) {
	input := `<timedtext format="3"><head></head><body>` +
		`<p t="0" d="1000"><s p="1">a</s><br/>b</p></body></timedtext>`
	doc, err := parse(t, input)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := doc.Cues[0].Text(); got != "a\nb" {
		t.Fatalf("text = %q, want a break after the span", got)
	}
}

func TestReaderSpanPenError(t *testing.T) {
	input := `<timedtext format="3"><head></head><body>` +
		`<p t="0" d="1000"><s p="x">a</s></p></body></timedtext>`
	if _, err := parse(t, input); err == nil {
		t.Fatal("a bad span pen id must fail")
	}
}

func TestReaderDropsEmptySpanAfterPadding(t *testing.T) {
	input := `<timedtext format="3"><head></head><body>` +
		`<p t="0" d="1000"><s p="1">&#8203;</s><s p="1">b</s></p></body></timedtext>`
	doc, err := parse(t, input)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := doc.Cues[0].Text(); got != "b" {
		t.Fatalf("text = %q, want only the visible run", got)
	}
}

func TestReaderEmptyParagraph(t *testing.T) {
	input := `<timedtext format="3"><head></head><body><p t="0" d="1000"></p></body></timedtext>`
	doc, err := parse(t, input)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(doc.Cues[0].Spans) != 0 {
		t.Fatalf("an empty paragraph must give no spans: %+v", doc.Cues[0].Spans)
	}
}

// TestWriterEmptyCueSpans covers the early return for a cue with no spans.
func TestWriterEmptyCueSpans(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues:   []model.Cue{{Start: 0, End: time.Second}},
	}
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(out.String(), "<p ") {
		t.Fatalf("a cue with no spans must write no line:\n%s", out.String())
	}
}

func TestReaderParagraphEndsAtEOF(t *testing.T) {
	// The paragraph runs past the end of its parent element. The XML
	// decoder reports unexpected end of input, so the read fails.
	input := `<timedtext format="3"><head></head><body><p t="0" d="1000">x</body></timedtext>`
	if _, err := parse(t, input); err == nil {
		t.Fatal("a paragraph that runs past its parent must fail")
	}
}

func TestReaderMalformedTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"paragraph token",
			`<timedtext format="3"><head></head><body><p t="0" d="1000">&bogus;</p></body></timedtext>`,
		},
		{
			"span decode",
			`<timedtext format="3"><head></head><body><p t="0" d="1000"><s p="1">&bogus;</s></p></body></timedtext>`,
		},
		{
			"break skip",
			`<timedtext format="3"><head></head><body><p t="0" d="1000">x<br>`,
		},
		{
			"unknown element skip",
			`<timedtext format="3"><head></head><body><p t="0" d="1000">x<foo>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parse(t, tt.input); err == nil {
				t.Fatalf("malformed input must fail: %s", tt.input)
			}
		})
	}
}

func TestPenAttributeErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"italic", `<timedtext format="3"><head><pen id="1" i="maybe" /></head><body></body></timedtext>`},
		{"underline", `<timedtext format="3"><head><pen id="1" u="maybe" /></head><body></body></timedtext>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parse(t, tt.input); err == nil {
				t.Fatalf("a bad pen attribute must fail: %s", tt.input)
			}
		})
	}
}

func TestWindowPositionYScalarError(t *testing.T) {
	input := `<timedtext format="3"><head><wp id="1" ap="0" av="x" /></head>` +
		`<body><p t="0" d="1000" wp="1">x</p></body></timedtext>`
	if _, err := parse(t, input); err == nil {
		t.Fatal("a bad vertical percentage must fail")
	}
}

// TestWindowPositionDefaults covers the default used when a window position
// omits its percentages.
func TestWindowPositionDefaults(t *testing.T) {
	input := `<timedtext format="3"><head><wp id="1" ap="0" /></head>` +
		`<body><p t="0" d="1000" wp="1">x</p></body></timedtext>`
	doc, err := parse(t, input)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	pos := doc.Cues[0].Layout.Position
	if pos.X != 640 || pos.Y != 648 {
		t.Fatalf("default position = %+v, want the default percentages", pos)
	}
}
