// SPDX-License-Identifier: Apache-2.0

package ytt

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

const fixturePath = "testdata/sample.ytt"

func readFixture(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(data)
}

func parseFixture(t *testing.T) *model.Document {
	t.Helper()
	doc, err := NewReader().Parse(strings.NewReader(readFixture(t)))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return doc
}

func ptr[T any](v T) *T { return &v }

func TestReaderParsesFixture(t *testing.T) {
	doc := parseFixture(t)
	if len(doc.Cues) != 15 {
		t.Fatalf("got %d cues, want 15", len(doc.Cues))
	}
	if len(doc.Styles) != 1 {
		t.Fatalf("reader must emit one implicit style, got %d", len(doc.Styles))
	}
	if got := doc.VideoDimensions; got.X != defaultWidth || got.Y != defaultHeight {
		t.Fatalf("dimensions = %+v", got)
	}
	first := doc.Cues[0]
	if first.Start != 0 || first.End != 2*time.Second {
		t.Fatalf("first cue timing = %v..%v", first.Start, first.End)
	}
	if first.Text() != "Bold line" {
		t.Fatalf("first cue text = %q", first.Text())
	}
	if first.Spans[0].Bold == nil || !*first.Spans[0].Bold {
		t.Fatalf("bold pen not applied: %+v", first.Spans[0])
	}
}

func TestReaderAppliesColours(t *testing.T) {
	doc := parseFixture(t)
	red := doc.Cues[4].Spans[0].Fore
	if red == nil || red.R != 255 || red.G != 0 || red.B != 0 || red.A != 40 {
		t.Fatalf("foreground colour = %+v, want red with alpha 40", red)
	}
	shadow := doc.Cues[5].Spans[0].Shadows
	if len(shadow) != 1 || shadow[0].Kind != model.ShadowHard {
		t.Fatalf("hard shadow = %+v", shadow)
	}
	if shadow[0].Colour.G != 255 {
		t.Fatalf("shadow colour = %+v, want green", shadow[0].Colour)
	}
}

func TestReaderAppliesFontAndScale(t *testing.T) {
	doc := parseFixture(t)
	font := doc.Cues[6].Spans[0].Font
	if font == nil || *font != "Courier New" {
		t.Fatalf("font = %v, want Courier New", font)
	}
	size := doc.Cues[7].Spans[0].Size
	if size == nil {
		t.Fatal("font scale not applied")
	}
	// sz=200 means a real factor of 1.25 on the base size of 20.
	if got, want := *size, 25.0; got != want {
		t.Fatalf("size = %v, want %v", got, want)
	}
}

func TestReaderAppliesScriptAndPacked(t *testing.T) {
	doc := parseFixture(t)
	sub := doc.Cues[9].Spans[0].Script
	if sub == nil || sub.Kind != model.ScriptSubscript {
		t.Fatalf("subscript = %+v", sub)
	}
	sup := doc.Cues[10].Spans[0].Script
	if sup == nil || sup.Kind != model.ScriptSuperscript {
		t.Fatalf("superscript = %+v", sup)
	}
	packed := doc.Cues[11].Spans[0].Packed
	if packed == nil || !*packed {
		t.Fatalf("packed = %v", packed)
	}
}

func TestReaderVerticalAndPosition(t *testing.T) {
	doc := parseFixture(t)
	vertical := doc.Cues[8]
	if vertical.Layout == nil || vertical.Layout.Anchor == nil {
		t.Fatalf("vertical cue has no layout: %+v", vertical.Layout)
	}
	if *vertical.Layout.Anchor != model.AnchorTopLeft {
		t.Fatalf("anchor = %d, want top left", *vertical.Layout.Anchor)
	}
	if pos := vertical.Layout.Position; pos == nil || pos.X != 0 || pos.Y != 72 {
		t.Fatalf("position = %+v, want (0, 72)", pos)
	}
	if v := vertical.Spans[0].Vertical; v == nil || v.Mode != model.VerticalColumnsRTL {
		t.Fatalf("vertical mode = %+v", v)
	}
	rotated := doc.Cues[14]
	if v := rotated.Spans[0].Vertical; v == nil || v.Mode != model.VerticalRotatedReversed {
		t.Fatalf("rotated mode = %+v", v)
	}
}

func TestReaderRubyDropsParenthesis(t *testing.T) {
	doc := parseFixture(t)
	cue := doc.Cues[12]
	if len(cue.Spans) != 2 {
		t.Fatalf("got %d spans, want base and reading only: %+v", len(cue.Spans), cue.Spans)
	}
	if cue.Spans[0].Text != "漢" || cue.Spans[0].Ruby != nil {
		t.Fatalf("base span = %+v", cue.Spans[0])
	}
	ann := cue.Spans[1]
	if ann.Text != "かん" || ann.Ruby == nil || ann.Ruby.Position != model.RubyOver {
		t.Fatalf("annotation span = %+v", ann)
	}
}

func TestReaderKaraokeTiling(t *testing.T) {
	doc := parseFixture(t)
	cue := doc.Cues[13]
	if !cue.Karaoke() {
		t.Fatal("cue must be karaoke")
	}
	if len(cue.Spans) != 4 {
		t.Fatalf("got %d spans, want 4", len(cue.Spans))
	}
	want := []struct{ start, end time.Duration }{
		{0, time.Second},
		{time.Second, 2 * time.Second},
		{2 * time.Second, 3 * time.Second},
		{3 * time.Second, 3 * time.Second},
	}
	for i, w := range want {
		if got := cue.Spans[i].Start; got != w.start {
			t.Errorf("span %d start = %v, want %v", i, got, w.start)
		}
		if got := cue.Spans[i].End; got != w.end {
			t.Errorf("span %d end = %v, want %v", i, got, w.end)
		}
	}
}

func TestReaderRejectsBadColour(t *testing.T) {
	input := `<timedtext format="3"><head><pen id="1" fc="notacolour" /></head><body></body></timedtext>`
	if _, err := NewReader().Parse(strings.NewReader(input)); err == nil {
		t.Fatal("bad colour must fail")
	}
}

func TestReaderRejectsBadXML(t *testing.T) {
	if _, err := NewReader().Parse(strings.NewReader("not xml at all")); err == nil {
		t.Fatal("bad XML must fail")
	}
}

func TestWriterRoundTripFixture(t *testing.T) {
	doc := parseFixture(t)
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	again, err := NewReader().Parse(strings.NewReader(out.String()))
	if err != nil {
		t.Fatalf("re-parse: %v\n%s", err, out.String())
	}
	if len(again.Cues) != len(doc.Cues) {
		t.Fatalf("cue count changed: %d to %d", len(doc.Cues), len(again.Cues))
	}
	for i := range doc.Cues {
		if doc.Cues[i].Text() != again.Cues[i].Text() {
			t.Errorf("cue %d text changed: %q to %q", i, doc.Cues[i].Text(), again.Cues[i].Text())
		}
		if doc.Cues[i].Start != again.Cues[i].Start || doc.Cues[i].End != again.Cues[i].End {
			t.Errorf("cue %d timing changed", i)
		}
	}
}

func TestWriterDeduplicatesPens(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues: []model.Cue{
			{Spans: []model.TextSpan{{Text: "one"}}},
			{Spans: []model.TextSpan{{Text: "two"}}},
		},
	}
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got := strings.Count(out.String(), "<pen "); got != 1 {
		t.Fatalf("got %d pens, want 1 after deduplication:\n%s", got, out.String())
	}
}

func TestWriterColourQuirks(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues: []model.Cue{{Spans: []model.TextSpan{
			{Text: "white", Fore: ptr(model.NewColour(255, 255, 255, 255))},
		}}},
	}
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, `fc="#FEFEFE"`) {
		t.Fatalf("white must shift to #FEFEFE:\n%s", got)
	}
	if !strings.Contains(got, `fo="254"`) {
		t.Fatalf("opacity must cap at 254:\n%s", got)
	}

	dark := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues: []model.Cue{{Spans: []model.TextSpan{
			{Text: "dark", Fore: ptr(model.NewColour(0, 0, 0, 255))},
		}}},
	}
	out.Reset()
	if _, err := NewWriter().Render(dark, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out.String(), `fc="#010101"`) {
		t.Fatalf("pure black must lift to #010101:\n%s", out.String())
	}
}

func TestWriterFontSnapAndScale(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues: []model.Cue{{Spans: []model.TextSpan{
			{Text: "big", Font: ptr("Fancy Font"), Size: ptr(25.0)},
		}}},
	}
	var out strings.Builder
	losses, err := NewWriter().Render(doc, &out)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out.String(), `fs="4"`) {
		t.Fatalf("unknown font must snap to Roboto (fs 4):\n%s", out.String())
	}
	if !strings.Contains(out.String(), `sz="200"`) {
		t.Fatalf("scale 1.25 must write sz 200:\n%s", out.String())
	}
	joined := strings.Join(losses, "; ")
	if !strings.Contains(joined, "Fancy Font") {
		t.Fatalf("loss report must name the snapped font: %v", losses)
	}
}

func TestWriterZeroWidthSpacePadding(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues: []model.Cue{{Spans: []model.TextSpan{
			{Text: "Bold", Bold: ptr(true)},
			{Text: "Red", Fore: ptr(model.NewColour(255, 0, 0, 255))},
		}}},
	}
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out.String(), "&#8203;") {
		t.Fatalf("multiple runs need zero-width space padding:\n%s", out.String())
	}
}

func TestWriterMultiShadowLayering(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues: []model.Cue{{Spans: []model.TextSpan{{
			Text: "layered",
			Shadows: []model.Shadow{
				{Kind: model.ShadowSoft, Colour: model.NewColour(0, 0, 0, 255)},
				{Kind: model.ShadowHard, Colour: model.NewColour(0, 0, 0, 255)},
			},
		}}}},
	}
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got := strings.Count(out.String(), "<p "); got != 2 {
		t.Fatalf("got %d lines, want 2 layers:\n%s", got, out.String())
	}
}

func TestWriterKaraokeOffsets(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues: []model.Cue{{
			Start: 0,
			End:   3 * time.Second,
			Spans: []model.TextSpan{
				{Text: "Ka", Start: 0, End: time.Second},
				{Text: "ra", Start: time.Second, End: 2 * time.Second},
				{Text: "oke", Start: 2 * time.Second, End: 3 * time.Second},
			},
		}},
	}
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out.String(), `t="1000"`) || !strings.Contains(out.String(), `t="2000"`) {
		t.Fatalf("karaoke offsets missing:\n%s", out.String())
	}
}

func TestWriterItalicPrefetchSpace(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues:   []model.Cue{{Spans: []model.TextSpan{{Text: "Italic", Italic: ptr(true)}}}},
	}
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out.String(), "> Italic</p>") {
		t.Fatalf("italic line start must steal a space:\n%s", out.String())
	}
}

func TestWriterReportsAnimations(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues: []model.Cue{{
			Spans:      []model.TextSpan{{Text: "x"}},
			Animations: []model.Animation{{Fade: &model.Fade{}}},
		}},
	}
	var out strings.Builder
	losses, err := NewWriter().Render(doc, &out)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(strings.Join(losses, "; "), "fade") {
		t.Fatalf("fade loss not reported: %v", losses)
	}
}

func TestLookupFontAllowList(t *testing.T) {
	allow := []string{
		"Arial", "Arial Black", "Arial Narrow", "Comic Sans MS", "Courier New",
		"Georgia", "Impact", "Roboto", "Tahoma", "Times New Roman",
		"Trebuchet MS", "Verdana",
	}
	for _, name := range allow {
		if _, ok := LookupFont(name); !ok {
			t.Errorf("font %q must be on the allow-list", name)
		}
	}
	if style, ok := LookupFont("Fancy Font"); ok || style != FontProportionalSans {
		t.Errorf("unknown font must snap to the default sans: %v %v", style, ok)
	}
	if got := FontName(FontMonospaceSerif); got != "Courier New" {
		t.Errorf("FontName = %q", got)
	}
}

func TestScaleConversion(t *testing.T) {
	tests := []struct {
		sz    int
		scale float64
	}{
		{100, 1},
		{0, 0.75},
		{200, 1.25},
		{300, 1.5},
	}
	for _, tt := range tests {
		if got := ScaleFromYTT(tt.sz); got != tt.scale {
			t.Errorf("ScaleFromYTT(%d) = %v, want %v", tt.sz, got, tt.scale)
		}
		if got := ScaleToYTT(tt.scale); got != tt.sz {
			t.Errorf("ScaleToYTT(%v) = %d, want %d", tt.scale, got, tt.sz)
		}
	}
	if got := ScaleToYTT(0.1); got != 0 {
		t.Errorf("a scale below the minimum must clamp to 0, got %d", got)
	}
}

func TestAnchorMapping(t *testing.T) {
	for ap := 0; ap <= 8; ap++ {
		anchor := anchorFromAP(ap)
		if back := apFromAnchor(anchor); back != ap {
			t.Errorf("round trip ap %d gave anchor %d and ap %d", ap, anchor, back)
		}
	}
	if got := anchorFromAP(8); got != model.AnchorBottomRight {
		t.Errorf("ap 8 = anchor %d, want bottom right", got)
	}
}

func TestDefaultPercent(t *testing.T) {
	tests := []struct {
		anchor model.Anchor
		ah, av int
	}{
		{model.AnchorTopLeft, 0, 10},
		{model.AnchorBottomCentre, 50, 90},
		{model.AnchorTopRight, 100, 10},
	}
	for _, tt := range tests {
		ah, av := defaultPercent(tt.anchor)
		if ah != tt.ah || av != tt.av {
			t.Errorf("defaultPercent(%d) = (%d,%d), want (%d,%d)", tt.anchor, ah, av, tt.ah, tt.av)
		}
	}
}

func TestParseColourAndOpacity(t *testing.T) {
	c, err := parseHexColour("#12AB34")
	if err != nil {
		t.Fatalf("parseHexColour: %v", err)
	}
	if c.R != 0x12 || c.G != 0xAB || c.B != 0x34 {
		t.Fatalf("colour = %+v", c)
	}
	if _, err := parseHexColour("bad"); err == nil {
		t.Fatal("bad colour must fail")
	}
	if a, err := parseOpacity(""); err != nil || a != 255 {
		t.Fatalf("missing opacity must default to 255: %d %v", a, err)
	}
	if _, err := parseOpacity("300"); err == nil {
		t.Fatal("out of range opacity must fail")
	}
}
