// SPDX-License-Identifier: Apache-2.0

package ytt

import (
	"errors"
	"strings"
	"testing"

	"github.com/bladeacer/swag/internal/model"
)

func TestReaderAndWriterNames(t *testing.T) {
	if got := NewReader().Name(); got != FormatName {
		t.Fatalf("reader name = %q", got)
	}
	if got := NewWriter().Name(); got != FormatName {
		t.Fatalf("writer name = %q", got)
	}
}

func TestReaderBooleanForms(t *testing.T) {
	input := `<timedtext format="3"><head><pen id="1" b="false" i="true" u="1" /></head>` +
		`<body><p t="0" d="1000" p="1">x</p></body></timedtext>`
	doc, err := NewReader().Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	span := doc.Cues[0].Spans[0]
	if isTrue(span.Bold) || !isTrue(span.Italic) || !isTrue(span.Underline) {
		t.Fatalf("boolean pen forms wrong: %+v", span)
	}
}

func TestReaderLineBreaksAndUnknownElements(t *testing.T) {
	input := `<timedtext format="3"><head></head><body>` +
		`<p t="0" d="1000">one<br/>two<foo>skip</foo>three</p>` +
		`</body></timedtext>`
	doc, err := NewReader().Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	// The break splits the line, and the unknown element is skipped.
	if got := doc.Cues[0].Text(); got != "one\ntwothree" {
		t.Fatalf("text = %q", got)
	}
}

func TestReaderSpanInheritsLinePen(t *testing.T) {
	input := `<timedtext format="3"><head><pen id="1" b="1" /></head><body>` +
		`<p t="0" d="1000" p="1"><s>plain run</s></p></body></timedtext>`
	doc, err := NewReader().Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !isTrue(doc.Cues[0].Spans[0].Bold) {
		t.Fatalf("span must inherit the line pen: %+v", doc.Cues[0].Spans[0])
	}
}

func TestReaderDropsPaddingSpan(t *testing.T) {
	input := `<timedtext format="3"><head><pen id="1" b="1" /><pen id="2" i="1" /></head><body>` +
		`<p t="0" d="1000"><s p="1">a</s>&#8203;<s p="2">b</s></p></body></timedtext>`
	doc, err := NewReader().Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for _, span := range doc.Cues[0].Spans {
		if strings.Contains(span.Text, zeroWidthSpace) {
			t.Fatalf("padding must be dropped: %+v", span)
		}
	}
}

func TestReaderDirectionFromPitch(t *testing.T) {
	input := `<timedtext format="3"><head><ws id="1" pd="1" /></head><body>` +
		`<p t="0" d="1000" ws="1">x</p></body></timedtext>`
	doc, err := NewReader().Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	d := doc.Cues[0].Spans[0].Direction
	if d == nil || *d != model.DirRightToLeft {
		t.Fatalf("pd 1 must mark right to left: %+v", d)
	}
}

func TestReaderDecimalPosition(t *testing.T) {
	input := `<timedtext format="3"><head><wp id="1" ap="7" ah="12.5" av="90" /></head><body>` +
		`<p t="0" d="1000" wp="1">x</p></body></timedtext>`
	doc, err := NewReader().Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if pos := doc.Cues[0].Layout.Position; pos.X != 160 {
		t.Fatalf("decimal position = %+v", pos)
	}
}

func TestReaderErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"start time", `<timedtext><head></head><body><p t="x" d="1">a</p></body></timedtext>`},
		{"duration", `<timedtext><head></head><body><p t="0" d="x">a</p></body></timedtext>`},
		{"position id", `<timedtext><head></head><body><p t="0" d="1" wp="x">a</p></body></timedtext>`},
		{"style id", `<timedtext><head></head><body><p t="0" d="1" ws="x">a</p></body></timedtext>`},
		{"pen id", `<timedtext><head></head><body><p t="0" d="1" p="x">a</p></body></timedtext>`},
		{"span time", `<timedtext><head></head><body><p t="0" d="1"><s t="x">a</s></p></body></timedtext>`},
		{"pen colour", `<timedtext><head><pen id="1" fc="nope" /></head><body></body></timedtext>`},
		{"pen opacity", `<timedtext><head><pen id="1" fc="#FFFFFF" fo="300" /></head><body></body></timedtext>`},
		{"pen boolean", `<timedtext><head><pen id="1" b="maybe" /></head><body></body></timedtext>`},
		{"pen edge", `<timedtext><head><pen id="1" et="9" /></head><body></body></timedtext>`},
		{"pen font style", `<timedtext><head><pen id="1" fs="x" /></head><body></body></timedtext>`},
		{"pen scale", `<timedtext><head><pen id="1" sz="x" /></head><body></body></timedtext>`},
		{"pen ruby", `<timedtext><head><pen id="1" rb="x" /></head><body></body></timedtext>`},
		{"pen offset", `<timedtext><head><pen id="1" of="x" /></head><body></body></timedtext>`},
		{"pen packed", `<timedtext><head><pen id="1" hg="x" /></head><body></body></timedtext>`},
		{"window pitch", `<timedtext><head><ws id="1" pd="x" /></head><body><p t="0" d="1" ws="1">a</p></body></timedtext>`},
		{"window skew", `<timedtext><head><ws id="1" pd="2" sd="x" /></head><body><p t="0" d="1" ws="1">a</p></body></timedtext>`},
		{"position anchor", `<timedtext><head><wp id="1" ap="x" /></head><body><p t="0" d="1" wp="1">a</p></body></timedtext>`},
		{"position x", `<timedtext><head><wp id="1" ah="x" /></head><body><p t="0" d="1" wp="1">a</p></body></timedtext>`},
		{"background opacity", `<timedtext><head><pen id="1" bc="#FFFFFF" bo="300" /></head><body></body></timedtext>`},
		{"edge opacity", `<timedtext><head><pen id="1" ec="#FFFFFF" eo="300" /></head><body></body></timedtext>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewReader().Parse(strings.NewReader(tt.input)); err == nil {
				t.Fatalf("input must fail: %s", tt.input)
			}
		})
	}
}

func TestReaderUnknownPenIsEmpty(t *testing.T) {
	input := `<timedtext format="3"><head></head><body><p t="0" d="1000" p="99">x</p></body></timedtext>`
	doc, err := NewReader().Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if doc.Cues[0].Spans[0].Bold != nil {
		t.Fatalf("unknown pen must set nothing: %+v", doc.Cues[0].Spans[0])
	}
}

func TestReaderKeepsLeadingSpaceWhenPlain(t *testing.T) {
	input := `<timedtext format="3"><head></head><body><p t="0" d="1000"> leading</p></body></timedtext>`
	doc, err := NewReader().Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := doc.Cues[0].Text(); got != " leading" {
		t.Fatalf("plain leading space must stay: %q", got)
	}
}

func TestWriterReportsEveryAnimation(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues: []model.Cue{{
			Spans:  []model.TextSpan{{Text: "x"}},
			Layout: &model.Layout{Fade: &model.Fade{}, Move: &model.Move{}},
			Animations: []model.Animation{
				{Move: &model.Move{}},
				{Shake: &model.Shake{}},
				{Chroma: &model.Chroma{}},
				{Keyframes: []model.Keyframe{{Steps: []model.KeyframeStep{{Property: "fore"}}}}},
				{Karaoke: &model.Karaoke{Kind: model.KaraokeFade}},
			},
		}},
	}
	var out strings.Builder
	losses, err := NewWriter().Render(doc, &out)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	joined := strings.Join(losses, "; ")
	for _, want := range []string{"cue fade", "cue move", "move animation", "shake", "chroma", "keyframe", "karaoke type"} {
		if !strings.Contains(joined, want) {
			t.Errorf("loss report missing %q: %v", want, losses)
		}
	}
}

func TestWriterEmitsEveryVerticalMode(t *testing.T) {
	modes := []struct {
		vertical model.Vertical
		marker   string
	}{
		{model.Vertical{Mode: model.VerticalColumnsLTR}, `pd="2" sd="1"`},
		{model.Vertical{Mode: model.VerticalRotated}, `pd="3" sd="0"`},
		{model.Vertical{Mode: model.VerticalRotatedReversed}, `pd="3" sd="1"`},
	}
	for _, tt := range modes {
		doc := &model.Document{
			Styles: []model.Style{DefaultStyle()},
			Cues:   []model.Cue{{Spans: []model.TextSpan{{Text: "v", Vertical: &tt.vertical}}}},
		}
		var out strings.Builder
		if _, err := NewWriter().Render(doc, &out); err != nil {
			t.Fatalf("Render: %v", err)
		}
		if !strings.Contains(out.String(), tt.marker) {
			t.Errorf("vertical mode marker %q missing:\n%s", tt.marker, out.String())
		}
	}
}

func TestWriterDirectionWindow(t *testing.T) {
	rtl := model.DirRightToLeft
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues:   []model.Cue{{Spans: []model.TextSpan{{Text: "x", Direction: &rtl}}}},
	}
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out.String(), `pd="1"`) {
		t.Fatalf("right-to-left window missing:\n%s", out.String())
	}
}

func TestWriterRubyFallbacks(t *testing.T) {
	under := model.RubyUnder
	paren := model.RubyParenthetical
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues: []model.Cue{
			{Spans: []model.TextSpan{
				{Text: "漢"},
				{Text: "かん", Ruby: &model.Ruby{Position: under}},
			}},
			{Spans: []model.TextSpan{
				{Text: "字"},
				{Text: "じ", Ruby: &model.Ruby{Position: paren}},
			}},
		},
	}
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out.String(), `rb="5"`) {
		t.Fatalf("ruby under must write rb 5:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "(じ)") {
		t.Fatalf("parenthetical ruby must fall back to brackets:\n%s", out.String())
	}
}

func TestWriterReportsScaleFloor(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues:   []model.Cue{{Spans: []model.TextSpan{{Text: "t", Size: ptr(5.0)}}}},
	}
	var out strings.Builder
	losses, err := NewWriter().Render(doc, &out)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(strings.Join(losses, ";"), "75%") {
		t.Fatalf("scale floor loss not reported: %v", losses)
	}
	if !strings.Contains(out.String(), `sz="0"`) {
		t.Fatalf("scale floor must write sz 0:\n%s", out.String())
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestWriterPropagatesWriteError(t *testing.T) {
	doc := &model.Document{Styles: []model.Style{DefaultStyle()}}
	if _, err := NewWriter().Render(doc, failWriter{}); err == nil {
		t.Fatal("write error must surface")
	}
}

func TestEdgeTypeValueTable(t *testing.T) {
	tests := map[model.ShadowKind]int{
		model.ShadowHard:    1,
		model.ShadowBevel:   2,
		model.ShadowGlow:    3,
		model.ShadowSoft:    4,
		model.ShadowKind(0): 0,
	}
	for kind, want := range tests {
		if got := edgeTypeValue(kind); got != want {
			t.Errorf("edgeTypeValue(%d) = %d, want %d", kind, got, want)
		}
	}
}

func TestEscapeText(t *testing.T) {
	got := escapeText("a<b>&c")
	if got != "a&lt;b&gt;&amp;c" {
		t.Fatalf("escapeText = %q", got)
	}
}

func TestFontNameUnknown(t *testing.T) {
	if got := FontName(FontStyle(99)); got != Roboto {
		t.Fatalf("unknown font style = %q, want %s", got, Roboto)
	}
	if _, ok := LookupFont(""); ok {
		t.Fatal("empty font name must be outside the allow-list")
	}
}

func TestAnchorOutOfRange(t *testing.T) {
	if got := anchorFromAP(-1); got != model.AnchorBottomCentre {
		t.Errorf("anchorFromAP(-1) = %d", got)
	}
	if got := anchorFromAP(99); got != model.AnchorBottomCentre {
		t.Errorf("anchorFromAP(99) = %d", got)
	}
	if got := apFromAnchor(model.Anchor(0)); got != apFromAnchor(model.AnchorBottomCentre) {
		t.Errorf("apFromAnchor(0) = %d", got)
	}
	if ah, av := defaultPercent(model.Anchor(0)); ah != 50 || av != 90 {
		t.Errorf("defaultPercent(0) = (%d,%d)", ah, av)
	}
}

func TestParseOpacityEmptyBody(t *testing.T) {
	if a, err := parseOpacity("   "); err != nil || a != 255 {
		t.Fatalf("blank opacity = %d, %v", a, err)
	}
}

func TestPositionErrorPaths(t *testing.T) {
	if _, err := attrEdgeKind("bad"); err == nil {
		t.Fatal("bad edge kind must fail")
	}
	if _, err := attrInt("bad", 0); err == nil {
		t.Fatal("bad integer must fail")
	}
	if _, err := attrFloat("bad", 0); err == nil {
		t.Fatal("bad float must fail")
	}
	if _, err := attrTrue("bad"); err == nil {
		t.Fatal("bad boolean must fail")
	}
	if _, err := attrColour("bad", ""); err == nil {
		t.Fatal("bad colour must fail")
	}
	if c, err := attrColour("", ""); err != nil || c != nil {
		t.Fatalf("missing colour must stay nil: %v %v", c, err)
	}
}

func TestPositionBadAnchor(t *testing.T) {
	wp := xmlWindowPos{ID: 1, Anchor: "bad"}
	if _, err := wp.toPosition(1280, 720); err == nil {
		t.Fatal("bad anchor must fail")
	}
}

func isTrue(v *bool) bool { return v != nil && *v }
