// SPDX-License-Identifier: Apache-2.0

package ttml

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

func parseFixture(t *testing.T, name string) *model.Document {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	doc, err := NewReader().Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return doc
}

func ptr[T any](v T) *T { return &v }

// withBody wraps a body fragment into a minimal TTML document.
func withBody(body string) string {
	return `<?xml version="1.0" encoding="utf-8"?><tt><head></head><body><div>` + body + `</div></body></tt>`
}

// withHead wraps a head and a body fragment into a TTML document.
func withHead(head, body string) string {
	return `<?xml version="1.0" encoding="utf-8"?><tt><head>` + head + `</head><body><div>` + body + `</div></body></tt>`
}

func TestNames(t *testing.T) {
	if got := NewReader().Name(); got != FormatName {
		t.Errorf("reader name = %q, want %q", got, FormatName)
	}
	if got := NewWriter().Name(); got != FormatName {
		t.Errorf("writer name = %q, want %q", got, FormatName)
	}
	if DefaultStyle().Name != "Default" {
		t.Errorf("default style name = %q", DefaultStyle().Name)
	}
}

func TestParseFixture(t *testing.T) {
	doc := parseFixture(t, "youtube.ttml")

	if len(doc.Styles) != 4 {
		t.Fatalf("styles = %d, want 4", len(doc.Styles))
	}
	// The red style extends base, so the base values survive the merge.
	red := doc.Styles[2]
	if red.Name != "red" || red.Font != "Roboto" || red.Size != 22 || !red.Bold {
		t.Errorf("red style did not merge base: %+v", red)
	}
	if red.Primary != model.NewColour(255, 0, 0, 128) {
		t.Errorf("red primary = %+v, want red with half alpha", red.Primary)
	}
	boxed := doc.Styles[3]
	if !boxed.Box || !boxed.Underline || boxed.OutlineWidth != 2 {
		t.Errorf("boxed style = %+v", boxed)
	}

	if len(doc.Cues) != 5 {
		t.Fatalf("cues = %d, want 5", len(doc.Cues))
	}

	first := doc.Cues[0]
	if first.Start != time.Second || first.End != 4*time.Second {
		t.Errorf("first cue timing = %v..%v", first.Start, first.End)
	}
	if first.Text() != "Hello world." {
		t.Errorf("first cue text = %q", first.Text())
	}
	if first.Layout == nil || first.Layout.Anchor == nil || *first.Layout.Anchor != model.AnchorBottomLeft {
		t.Errorf("first cue layout = %+v", first.Layout)
	}
	if pos := first.Layout.Position; pos == nil || pos.X != 128 || pos.Y != 648 {
		t.Errorf("first cue position = %+v", pos)
	}
	// The span takes the red style and keeps the base font.
	if s := first.Spans[1]; s.Fore == nil || s.Font == nil || *s.Font != "Roboto" || !isTrue(s.Bold) {
		t.Errorf("styled span = %+v", s)
	}

	second := doc.Cues[1]
	if second.Start != 1500*time.Millisecond || second.End != 3500*time.Millisecond {
		t.Errorf("second cue timing = %v..%v", second.Start, second.End)
	}
	if second.Layout == nil || second.Layout.Anchor == nil || *second.Layout.Anchor != model.AnchorTopRight {
		t.Errorf("second cue layout = %+v", second.Layout)
	}
	if !second.Karaoke() {
		t.Fatalf("second cue lost karaoke: %+v", second.Spans)
	}
	if second.Spans[0].Start != 0 || second.Spans[0].End != time.Second {
		t.Errorf("span 0 timing = %v..%v", second.Spans[0].Start, second.Spans[0].End)
	}
	if second.Spans[2].Start != time.Second || second.Spans[2].End != 2*time.Second {
		t.Errorf("span 2 timing = %v..%v", second.Spans[2].Start, second.Spans[2].End)
	}

	if third := doc.Cues[2]; third.Text() != "Line one\nline two" {
		t.Errorf("third cue text = %q", third.Text())
	}

	fourth := doc.Cues[3]
	if !isTrue(fourth.Spans[0].Italic) || !isTrue(fourth.Spans[0].Strikeout) {
		t.Errorf("fourth cue span = %+v", fourth.Spans[0])
	}

	fifth := doc.Cues[4]
	if fifth.Start != 20*time.Second || fifth.End != 20*time.Second {
		t.Errorf("fifth cue timing = %v..%v", fifth.Start, fifth.End)
	}
	if fifth.Layout == nil || fifth.Layout.Position != nil || fifth.Layout.Anchor != nil {
		t.Errorf("fifth cue layout = %+v", fifth.Layout)
	}
	if fifth.Text() != "No end and nested text." {
		t.Errorf("fifth cue text = %q", fifth.Text())
	}
	if span := fifth.Spans[1]; span.Fore == nil || span.Fore.A != 64 {
		t.Errorf("fifth cue span = %+v", span)
	}
}

func TestParseTimeForms(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		{"00:00:01.000", time.Second},
		{"01:30.500", 90*time.Second + 500*time.Millisecond},
		{"00:00:01:15", time.Second + 500*time.Millisecond},
		{"1.5s", 1500 * time.Millisecond},
		{"250ms", 250 * time.Millisecond},
		{"1h", time.Hour},
		{"2m", 2 * time.Minute},
		{"3s", 3 * time.Second},
		{"60f", 2 * time.Second},
	}
	for _, tt := range tests {
		got, err := parseTime(tt.in)
		if err != nil {
			t.Errorf("parseTime(%q) error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseTime(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseTimeErrors(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", "empty value"},
		{"1:2:3:4:5", "two to four"},
		{"aa:bb", "parse time"},
		{"12", "missing unit"},
		{"zzs", "parse time"},
	}
	for _, tt := range tests {
		_, err := parseTime(tt.in)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("parseTime(%q) error = %v, want %q", tt.in, err, tt.want)
		}
	}
}

func TestParsePercent(t *testing.T) {
	got, err := parsePercent("10%")
	if err != nil || got != 0.1 {
		t.Errorf("parsePercent(10%%) = %v, %v", got, err)
	}
	if _, err := parsePercent("10"); err == nil {
		t.Error("parsePercent without a sign must fail")
	}
	if _, err := parsePercent("x%"); err == nil {
		t.Error("parsePercent with a bad number must fail")
	}
}

func TestParseOrigin(t *testing.T) {
	if _, _, err := parseOrigin("10%"); err == nil {
		t.Error("parseOrigin with one field must fail")
	}
	if _, _, err := parseOrigin("x% 10%"); err == nil {
		t.Error("parseOrigin with a bad x must fail")
	}
	if _, _, err := parseOrigin("10% y%"); err == nil {
		t.Error("parseOrigin with a bad y must fail")
	}
}

func TestParseColour(t *testing.T) {
	got, err := parseColour("#FF8000")
	if err != nil || got != model.NewColour(255, 128, 0, 255) {
		t.Errorf("parseColour = %+v, %v", got, err)
	}
	got, err = parseColour("#ff800080")
	if err != nil || got != model.NewColour(255, 128, 0, 128) {
		t.Errorf("parseColour with alpha = %+v, %v", got, err)
	}
	for _, bad := range []string{"#fff", "#zzzzzz"} {
		if _, err := parseColour(bad); err == nil {
			t.Errorf("parseColour(%q) must fail", bad)
		}
	}
}

func TestParseOutline(t *testing.T) {
	width, colour, err := parseOutline("2px #000000")
	if err != nil || width != 2 || colour == nil || *colour != model.NewColour(0, 0, 0, 255) {
		t.Errorf("parseOutline = %v, %+v, %v", width, colour, err)
	}
	width, colour, err = parseOutline("#000000")
	if err != nil || width != 0 || colour == nil {
		t.Errorf("parseOutline with colour only = %v, %+v, %v", width, colour, err)
	}
	if _, _, err := parseOutline("zz"); err == nil {
		t.Error("parseOutline with a bad width must fail")
	}
	if _, _, err := parseOutline("#zz"); err == nil {
		t.Error("parseOutline with a bad colour must fail")
	}
}

func TestParseOpacity(t *testing.T) {
	if got, err := parseOpacity("0.5"); err != nil || got != 128 {
		t.Errorf("parseOpacity(0.5) = %d, %v", got, err)
	}
	if _, err := parseOpacity("zz"); err == nil {
		t.Error("parseOpacity with a bad number must fail")
	}
	if _, err := parseOpacity("5"); err == nil {
		t.Error("parseOpacity out of range must fail")
	}
}

func TestParseBoldItalicDecoration(t *testing.T) {
	if bold, err := parseBold("normal"); err != nil || bold {
		t.Errorf("parseBold(normal) = %v, %v", bold, err)
	}
	if _, err := parseBold("heavy"); err == nil {
		t.Error("parseBold with an unknown value must fail")
	}
	if italic, err := parseItalic("normal"); err != nil || italic {
		t.Errorf("parseItalic(normal) = %v, %v", italic, err)
	}
	if _, err := parseItalic("cursive"); err == nil {
		t.Error("parseItalic with an unknown value must fail")
	}
	underline, strikeout, err := parseDecoration("none")
	if err != nil || underline || strikeout {
		t.Errorf("parseDecoration(none) = %v, %v, %v", underline, strikeout, err)
	}
	if _, _, err := parseDecoration("blink"); err == nil {
		t.Error("parseDecoration with an unknown value must fail")
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := NewReader().Parse(strings.NewReader("<tt")); err == nil {
		t.Error("truncated XML must fail")
	}
	// A paragraph without a begin attribute has no timing.
	if _, err := NewReader().Parse(strings.NewReader(withBody("<p>text</p>"))); err == nil {
		t.Error("a paragraph without begin must fail")
	}
}

func TestParseParagraphErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"begin", `<p begin="zzs">text</p>`, "paragraph begin"},
		{"end", `<p begin="1s" end="zzs">text</p>`, "paragraph end"},
		{"dur", `<p begin="1s" dur="zzs">text</p>`, "paragraph duration"},
		{"span begin", `<p begin="1s" end="2s"><span begin="zzs">a</span></p>`, "span begin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewReader().Parse(strings.NewReader(withBody(tt.body)))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestParseRegionErrors(t *testing.T) {
	tests := []struct {
		name string
		head string
		want string
	}{
		{"origin", `<layout><region xml:id="r" tts:origin="10%"/></layout>`, "origin"},
		{"display", `<layout><region xml:id="r" tts:displayAlign="sideways"/></layout>`, "displayAlign"},
		{"text", `<layout><region xml:id="r" tts:textAlign="sideways"/></layout>`, "textAlign"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewReader().Parse(strings.NewReader(withHead(tt.head, `<p begin="1s" end="2s" region="r">text</p>`)))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestParseRegionAlignValues(t *testing.T) {
	tests := []struct {
		display string
		text    string
		want    model.Anchor
	}{
		{"center", "start", model.AnchorMiddleLeft},
		{"center", "center", model.AnchorCentre},
		{"center", "end", model.AnchorMiddleRight},
		{"before", "center", model.AnchorTopCentre},
		{"after", "left", model.AnchorBottomLeft},
		{"", "right", model.AnchorBottomRight},
	}
	for _, tt := range tests {
		head := `<layout><region xml:id="r"`
		if tt.display != "" {
			head += ` tts:displayAlign="` + tt.display + `"`
		}
		if tt.text != "" {
			head += ` tts:textAlign="` + tt.text + `"`
		}
		head += ` tts:origin="0% 0%"/></layout>`
		doc, err := NewReader().Parse(strings.NewReader(withHead(head, `<p begin="1s" end="2s" region="r">text</p>`)))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if got := doc.Cues[0].Layout.Anchor; got == nil || *got != tt.want {
			t.Errorf("display %q text %q anchor = %v, want %v", tt.display, tt.text, got, tt.want)
		}
	}
}

func TestParseStyleErrors(t *testing.T) {
	tests := []struct {
		name string
		head string
		want string
	}{
		{"colour", `<styling><style xml:id="s" tts:color="zz"/></styling>`, "parse colour"},
		{"background", `<styling><style xml:id="s" tts:backgroundColor="zz"/></styling>`, "parse colour"},
		{"size", `<styling><style xml:id="s" tts:fontSize="zz"/></styling>`, "parse length"},
		{"weight", `<styling><style xml:id="s" tts:fontWeight="heavy"/></styling>`, "parse fontWeight"},
		{"style attr", `<styling><style xml:id="s" tts:fontStyle="cursive"/></styling>`, "parse fontStyle"},
		{"decoration", `<styling><style xml:id="s" tts:textDecoration="blink"/></styling>`, "parse textDecoration"},
		{"outline", `<styling><style xml:id="s" tts:textOutline="zz"/></styling>`, "parse length"},
		{"opacity", `<styling><style xml:id="s" tts:opacity="5"/></styling>`, "parse opacity"},
		{"unknown ref", `<styling><style xml:id="s" style="nope"/></styling>`, "unknown style"},
		{"cycle", `<styling><style xml:id="a" style="b"/><style xml:id="b" style="a"/></styling>`, "style cycle"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewReader().Parse(strings.NewReader(withHead(tt.head, `<p begin="1s" end="2s">text</p>`)))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestParseSpanErrors(t *testing.T) {
	tests := []struct {
		name string
		attr string
		want string
	}{
		{"colour", `tts:color="zz"`, "parse colour"},
		{"background", `tts:backgroundColor="zz"`, "parse colour"},
		{"size", `tts:fontSize="zz"`, "parse length"},
		{"weight", `tts:fontWeight="heavy"`, "parse fontWeight"},
		{"style", `tts:fontStyle="cursive"`, "parse fontStyle"},
		{"decoration", `tts:textDecoration="blink"`, "parse textDecoration"},
		{"outline", `tts:textOutline="zz"`, "parse length"},
		{"opacity", `tts:opacity="zz"`, "parse opacity"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `<p begin="1s" end="2s"><span ` + tt.attr + `>text</span></p>`
			_, err := NewReader().Parse(strings.NewReader(withBody(body)))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestParseSpanOpacityOnly(t *testing.T) {
	doc, err := NewReader().Parse(strings.NewReader(
		withBody(`<p begin="1s" end="2s"><span tts:opacity="0.5">text</span></p>`)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fore := doc.Cues[0].Spans[0].Fore
	if fore == nil || fore.A != 128 || fore.R != 0 {
		t.Errorf("fore = %+v, want alpha 128", fore)
	}
}

func TestParseSpanNormalValues(t *testing.T) {
	doc, err := NewReader().Parse(strings.NewReader(
		withBody(`<p begin="1s" end="2s"><span tts:fontWeight="normal" tts:fontStyle="normal" tts:textDecoration="none" tts:textOutline="#000000">text</span></p>`)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	span := doc.Cues[0].Spans[0]
	if isTrue(span.Bold) || isTrue(span.Italic) || isTrue(span.Underline) || isTrue(span.Strikeout) {
		t.Errorf("span = %+v, want every flag clear", span)
	}
	if span.OutlineWidth == nil || *span.OutlineWidth != 0 {
		t.Errorf("outline width = %+v, want 0", span.OutlineWidth)
	}
}

func TestParseParagraphStyleError(t *testing.T) {
	_, err := NewReader().Parse(strings.NewReader(withBody(`<p begin="1s" end="2s" style="ghost">text</p>`)))
	if err == nil || !strings.Contains(err.Error(), "unknown style") {
		t.Errorf("error = %v, want unknown style", err)
	}
}

func TestParseSpanStyleError(t *testing.T) {
	_, err := NewReader().Parse(strings.NewReader(
		withBody(`<p begin="1s" end="2s"><span style="ghost">text</span></p>`)))
	if err == nil || !strings.Contains(err.Error(), "unknown style") {
		t.Errorf("error = %v, want unknown style", err)
	}
}

func TestParseParagraphInlineErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"text run", `<p begin="1s" end="2s" tts:color="zz">hi</p>`},
		{"break run", `<p begin="1s" end="2s" tts:color="zz"><br/></p>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewReader().Parse(strings.NewReader(withBody(tt.body)))
			if err == nil || !strings.Contains(err.Error(), "parse colour") {
				t.Errorf("error = %v, want parse colour", err)
			}
		})
	}
}

func TestParseTruncatedElements(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"span", `<p begin="1s" end="2s"><span>text`},
		{"break", `<p begin="1s" end="2s"><br>`},
		{"unknown", `<p begin="1s" end="2s"><c>`},
		{"text", `<p begin="1s" end="2s">text`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewReader().Parse(strings.NewReader(withBody(tt.body))); err == nil {
				t.Error("truncated XML must fail")
			}
		})
	}
}

// renderDoc renders a document and returns the output and the loss report.
func renderDoc(t *testing.T, doc *model.Document) (string, []string) {
	t.Helper()
	var out strings.Builder
	losses, err := NewWriter().Render(doc, &out)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return out.String(), losses
}

func TestRenderRoundTrip(t *testing.T) {
	red := model.NewColour(255, 0, 0, 128)
	blue := model.NewColour(0, 0, 255, 255)
	anchor := model.AnchorTopLeft
	doc := &model.Document{
		VideoDimensions: model.Point{X: 1280, Y: 720},
		Styles: []model.Style{
			{
				Name: "Default", Font: "Roboto", Size: 22, Bold: true, Italic: true,
				Underline: true, Primary: red, Outline: model.NewColour(0, 0, 0, 255),
				OutlineWidth: 2, Alignment: model.AnchorBottomCentre,
			},
			{Name: "Boxed", Box: true, Outline: blue, Primary: model.NewColour(255, 255, 255, 255)},
		},
		Cues: []model.Cue{
			{
				Start: time.Second, End: 4 * time.Second,
				Spans: []model.TextSpan{{
					Text: "Hello", Bold: ptr(true), Italic: ptr(true), Underline: ptr(true),
					Strikeout: ptr(true), Font: ptr("Courier"), Size: ptr(18.0),
					Fore: &red, Back: &blue, OutlineWidth: ptr(1.0),
				}},
			},
			{
				Start: 5 * time.Second, End: 8 * time.Second,
				Layout: &model.Layout{Position: &model.Point{X: 128, Y: 648}, Anchor: &anchor},
				Spans: []model.TextSpan{
					{Text: "a", Start: 0, End: time.Second},
					{Text: "b", Start: time.Second, End: 3 * time.Second},
				},
			},
			{
				Start: 9 * time.Second, End: 11 * time.Second,
				Spans: []model.TextSpan{
					{Text: "漢", Ruby: nil},
					{Text: "かん", Ruby: &model.Ruby{Position: model.RubyOver}},
				},
			},
		},
	}

	out, losses := renderDoc(t, doc)
	if len(losses) != 1 || losses[0] != "ruby text" {
		t.Errorf("losses = %v, want only ruby text", losses)
	}
	for _, want := range []string{"<tt ", `xml:id="style0"`, `xml:id="style1"`, `xml:id="region1"`,
		"tts:origin=", `tts:displayAlign="before"`, `tts:textAlign="start"`,
		`tts:backgroundColor="#0000ff"`, `tts:textDecoration="underline lineThrough"`,
		`tts:fontWeight="bold"`, `tts:fontStyle="italic"`, "かん", "(かん)"} {
		if !strings.Contains(out, want) {
			t.Errorf("output is missing %q:\n%s", want, out)
		}
	}

	back, err := NewReader().Parse(strings.NewReader(out))
	if err != nil {
		t.Fatalf("re-parse: %v\n%s", err, out)
	}
	if len(back.Cues) != len(doc.Cues) {
		t.Fatalf("cues = %d, want %d", len(back.Cues), len(doc.Cues))
	}
	for i := range doc.Cues {
		if back.Cues[i].Start != doc.Cues[i].Start || back.Cues[i].End != doc.Cues[i].End {
			t.Errorf("cue %d timing = %v..%v, want %v..%v", i,
				back.Cues[i].Start, back.Cues[i].End, doc.Cues[i].Start, doc.Cues[i].End)
		}
	}
	if back.Cues[0].Text() != "Hello" {
		t.Errorf("first cue text = %q", back.Cues[0].Text())
	}
	// The colour alpha survives the opacity attribute exactly. The reader
	// prepends its own default style, so the written style is the second
	// one.
	if got := back.Styles[1].Primary; got != model.NewColour(255, 0, 0, 128) {
		t.Errorf("style primary = %+v, want red with alpha 128", got)
	}
}

func TestRenderLosses(t *testing.T) {
	doc := &model.Document{
		VideoDimensions: model.Point{X: 1280, Y: 720},
		Styles:          []model.Style{DefaultStyle()},
		Cues: []model.Cue{{
			Start: time.Second, End: 2 * time.Second,
			Layout: &model.Layout{
				Fade: &model.Fade{In: time.Second},
				Move: &model.Move{From: model.Point{X: 1}, To: model.Point{X: 2}},
			},
			Animations: []model.Animation{{Shake: &model.Shake{RadiusX: 2}}},
			Spans: []model.TextSpan{
				{
					Text: "text", ScaleX: ptr(150.0), ScaleY: ptr(80.0),
					Shadows:   []model.Shadow{{Kind: model.ShadowSoft}},
					Vertical:  &model.Vertical{Mode: model.VerticalColumnsRTL},
					Packed:    ptr(true),
					Direction: ptr(model.DirRightToLeft),
					Script:    &model.Script{Kind: model.ScriptSubscript},
				},
				{Text: "かん", Ruby: &model.Ruby{}},
			},
		}},
	}
	_, losses := renderDoc(t, doc)
	joined := strings.Join(losses, "; ")
	for _, want := range []string{
		"cue fade", "cue move", "animation", "glyph scale", "shadow effects",
		"vertical text", "packing", "right-to-left marking", "script offset", "ruby text",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("losses are missing %q: %v", want, losses)
		}
	}
}

func TestRenderNoStyles(t *testing.T) {
	doc := &model.Document{
		VideoDimensions: model.Point{X: 0, Y: 0},
		Cues:            []model.Cue{{Start: 0, End: time.Second, Spans: []model.TextSpan{{Text: "plain <text>"}}}},
	}
	out, losses := renderDoc(t, doc)
	if len(losses) != 0 {
		t.Errorf("losses = %v, want none", losses)
	}
	if strings.Contains(out, "style=") {
		t.Errorf("a document without styles must not reference one:\n%s", out)
	}
	back, err := NewReader().Parse(strings.NewReader(out))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if back.Cues[0].Text() != "plain <text>" {
		t.Errorf("text = %q", back.Cues[0].Text())
	}
}

func TestRenderEveryAnchor(t *testing.T) {
	anchors := []struct {
		a       model.Anchor
		display string
		text    string
	}{
		{model.AnchorBottomLeft, "after", "start"},
		{model.AnchorBottomCentre, "after", "center"},
		{model.AnchorBottomRight, "after", "end"},
		{model.AnchorMiddleLeft, "center", "start"},
		{model.AnchorCentre, "center", "center"},
		{model.AnchorMiddleRight, "center", "end"},
		{model.AnchorTopLeft, "before", "start"},
		{model.AnchorTopCentre, "before", "center"},
		{model.AnchorTopRight, "before", "end"},
	}
	for _, tt := range anchors {
		a := tt.a
		doc := &model.Document{
			VideoDimensions: model.Point{X: 1280, Y: 720},
			Styles:          []model.Style{DefaultStyle()},
			Cues: []model.Cue{{
				Start: 0, End: time.Second,
				Layout: &model.Layout{Position: &model.Point{X: 640, Y: 360}, Anchor: &a},
				Spans:  []model.TextSpan{{Text: "text"}},
			}},
		}
		out, _ := renderDoc(t, doc)
		if !strings.Contains(out, `tts:displayAlign="`+tt.display+`"`) ||
			!strings.Contains(out, `tts:textAlign="`+tt.text+`"`) {
			t.Errorf("anchor %d is missing its alignment:\n%s", tt.a, out)
		}
	}
}

func TestRenderUnknownAnchor(t *testing.T) {
	bad := model.Anchor(0)
	doc := &model.Document{
		VideoDimensions: model.Point{X: 1280, Y: 720},
		Styles:          []model.Style{DefaultStyle()},
		Cues: []model.Cue{{
			Start: 0, End: time.Second,
			Layout: &model.Layout{Anchor: &bad},
			Spans:  []model.TextSpan{{Text: "text"}},
		}},
	}
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err == nil {
		t.Error("an unknown anchor must fail the write")
	}
}

// failWriter fails every write.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestRenderWriteError(t *testing.T) {
	doc := &model.Document{
		VideoDimensions: model.Point{X: 1280, Y: 720},
		Styles:          []model.Style{DefaultStyle()},
		Cues:            []model.Cue{{Start: 0, End: time.Second, Spans: []model.TextSpan{{Text: "text"}}}},
	}
	if _, err := NewWriter().Render(doc, failWriter{}); err == nil {
		t.Error("a failing sink must return an error")
	}
}

func TestFormatHelpers(t *testing.T) {
	if got := formatTime(-time.Second); got != "00:00:00.000" {
		t.Errorf("formatTime(-1s) = %q", got)
	}
	if got := formatOpacity(255); got != "1.000" {
		t.Errorf("formatOpacity(255) = %q", got)
	}
	// 254 is the alpha ceiling of the YouTube path, and the three decimals
	// bring it back exactly.
	if got := formatOpacity(254); got != "0.996" {
		t.Errorf("formatOpacity(254) = %q", got)
	}
	if got := formatOrigin(model.Point{X: 128, Y: 648}, 1280, 720); got != "10% 90%" {
		t.Errorf("formatOrigin = %q", got)
	}
}
