// SPDX-License-Identifier: Apache-2.0

package ass

import (
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
		t.Fatalf("Parse: %v", err)
	}
	return doc
}

func TestName(t *testing.T) {
	if got := NewReader().Name(); got != FormatName {
		t.Fatalf("Name = %q", got)
	}
}

func TestParseScriptInfoAndStyles(t *testing.T) {
	doc := parseFixture(t, "karaoke.ass")
	if doc.Metadata["Title"] == "" {
		t.Fatalf("script info not read: %+v", doc.Metadata)
	}
	if doc.VideoDimensions.X != 1280 || doc.VideoDimensions.Y != 720 {
		t.Fatalf("play resolution = %+v", doc.VideoDimensions)
	}
	if len(doc.Styles) != 2 {
		t.Fatalf("got %d styles, want 2", len(doc.Styles))
	}
	base := doc.Styles[0]
	if base.Name != "Default" {
		t.Fatalf("base style = %q, want Default first", base.Name)
	}
	if base.Primary != model.NewColour(255, 255, 255, 255) {
		t.Fatalf("primary = %+v", base.Primary)
	}
	if base.Secondary != model.NewColour(0x77, 0x77, 0x77, 255) {
		t.Fatalf("secondary = %+v", base.Secondary)
	}
	if base.Shadow.A != 155 {
		t.Fatalf("back colour alpha = %d, want 155", base.Shadow.A)
	}
	if base.Box || base.Alignment != model.AnchorBottomCentre {
		t.Fatalf("base flags = %+v", base)
	}
}

func TestParseBoxStyle(t *testing.T) {
	doc := parseFixture(t, "karaoke.ass")
	var box model.Style
	for _, s := range doc.Styles {
		if s.Name == "KaraokeBox" {
			box = s
		}
	}
	if !box.Box {
		t.Fatalf("BorderStyle 3 must set Box: %+v", box)
	}
	if box.Font != "Verdana" || box.Size != 25 {
		t.Fatalf("box font = %q %v", box.Font, box.Size)
	}
}

func TestParseKaraokeTiming(t *testing.T) {
	doc := parseFixture(t, "karaoke.ass")
	cue := doc.Cues[0]
	if !cue.Karaoke() {
		t.Fatal("first cue must be karaoke")
	}
	if got, want := cue.Text(), "Nyan passuru hotto keeki"; got != want {
		t.Fatalf("text = %q, want %q", got, want)
	}
	if len(cue.Spans) != 7 {
		t.Fatalf("got %d spans, want 7", len(cue.Spans))
	}
	want := []struct{ start, end time.Duration }{
		{0, 420 * time.Millisecond},
		{420 * time.Millisecond, 800 * time.Millisecond},
		{800 * time.Millisecond, 1300 * time.Millisecond},
	}
	for i, w := range want {
		if cue.Spans[i].Start != w.start || cue.Spans[i].End != w.end {
			t.Errorf("span %d = %v..%v, want %v..%v", i, cue.Spans[i].Start, cue.Spans[i].End, w.start, w.end)
		}
	}
}

func TestParseSecondaryColour(t *testing.T) {
	doc := parseFixture(t, "karaoke.ass")
	cue := doc.Cues[4]
	if cue.Spans[0].Secondary == nil {
		t.Fatalf("\\2a must set the secondary colour: %+v", cue.Spans[0])
	}
	if a := cue.Spans[0].Secondary.A; a != 0 {
		t.Fatalf("\\2a&HFF& must make the unsung text invisible, alpha = %d", a)
	}
}

func TestParseMidSyllableColour(t *testing.T) {
	doc := parseFixture(t, "karaoke.ass")
	cue := doc.Cues[5]
	if len(cue.Spans) != 7 {
		t.Fatalf("got %d spans, want 7: %+v", len(cue.Spans), cue.Spans)
	}
	green := cue.Spans[2].Fore
	if green == nil || green.R != 0 || green.G != 255 || green.B != 0 {
		t.Fatalf("\\c&H00FF00& must give green: %+v", green)
	}
	if cue.Spans[3].Fore != nil {
		t.Fatalf("\\c with no value must reset to the style: %+v", cue.Spans[3].Fore)
	}
	if !isTrue(cue.Spans[5].Italic) || isTrue(cue.Spans[6].Italic) {
		t.Fatalf("italic toggles wrong: %+v %+v", cue.Spans[5].Italic, cue.Spans[6].Italic)
	}
}

func TestParseStyleFlattening(t *testing.T) {
	doc := parseFixture(t, "colour.ass")
	boxed := doc.Cues[7]
	if boxed.Spans[0].Back == nil {
		t.Fatalf("boxed cue must carry a background colour: %+v", boxed.Spans[0])
	}
	heavy := doc.Cues[8]
	span := heavy.Spans[0]
	if span.Font == nil || *span.Font != "Trebuchet MS" {
		t.Fatalf("style font not flattened: %+v", span.Font)
	}
	if span.OutlineWidth == nil || *span.OutlineWidth != 4 {
		t.Fatalf("style outline not flattened: %+v", span.OutlineWidth)
	}
	if !isTrue(span.Bold) {
		t.Fatalf("style bold not flattened: %+v", span.Bold)
	}
}

func TestParseInlineStyling(t *testing.T) {
	doc := parseFixture(t, "colour.ass")
	bold := doc.Cues[1]
	if !isTrue(bold.Spans[0].Bold) {
		t.Fatalf("\\b1 missing: %+v", bold.Spans[0])
	}
	fonts := doc.Cues[2]
	if fonts.Spans[0].Font == nil || *fonts.Spans[0].Font != "Comic Sans MS" {
		t.Fatalf("\\fn missing: %+v", fonts.Spans[0].Font)
	}
	sizes := doc.Cues[3]
	if sizes.Spans[0].Size == nil || *sizes.Spans[0].Size != 40 {
		t.Fatalf("\\fs40 missing: %+v", sizes.Spans[0].Size)
	}
	if sizes.Spans[1].Size == nil || *sizes.Spans[1].Size != 15 {
		t.Fatalf("\\fs15 missing: %+v", sizes.Spans[1].Size)
	}
	if sizes.Spans[2].Size != nil {
		t.Fatalf("\\fs with no value must reset to the style: %+v", sizes.Spans[2].Size)
	}
}

func TestParseTransparency(t *testing.T) {
	doc := parseFixture(t, "colour.ass")
	all := doc.Cues[5]
	if all.Spans[0].Fore == nil || all.Spans[0].Fore.A != 191 {
		t.Fatalf("\\alpha&H40& must give alpha 191: %+v", all.Spans[0].Fore)
	}
	one := doc.Cues[6]
	if one.Spans[0].Fore == nil || one.Spans[0].Fore.A != 127 {
		t.Fatalf("\\1a&H80& must give alpha 127: %+v", one.Spans[0].Fore)
	}
}

func TestParseLayoutTags(t *testing.T) {
	doc := parseFixture(t, "colour.ass")
	corner := doc.Cues[9]
	if corner.Layout == nil || corner.Layout.Anchor == nil || *corner.Layout.Anchor != model.AnchorTopLeft {
		t.Fatalf("\\an7 missing: %+v", corner.Layout)
	}
	if corner.Layout.Position == nil || corner.Layout.Position.X != 100 || corner.Layout.Position.Y != 100 {
		t.Fatalf("\\pos missing: %+v", corner.Layout.Position)
	}
	moved := doc.Cues[11]
	if moved.Layout == nil || moved.Layout.Move == nil {
		t.Fatalf("\\move missing: %+v", moved.Layout)
	}
	if moved.Layout.Move.From.X != 200 || moved.Layout.Move.To.X != 1080 {
		t.Fatalf("move points = %+v", moved.Layout.Move)
	}
}

func TestParseAnimations(t *testing.T) {
	doc := parseFixture(t, "colour.ass")
	faded := doc.Cues[12]
	if faded.Layout == nil || faded.Layout.Fade == nil {
		t.Fatalf("\\fad missing: %+v", faded.Layout)
	}
	if faded.Layout.Fade.In != 500*time.Millisecond {
		t.Fatalf("fade in = %v", faded.Layout.Fade.In)
	}
	keyed := doc.Cues[13]
	var steps []model.KeyframeStep
	for _, a := range keyed.Animations {
		for _, kf := range a.Keyframes {
			steps = append(steps, kf.Steps...)
		}
	}
	if len(steps) == 0 || steps[0].Property != "fore" {
		t.Fatalf("\\t missing: %+v", steps)
	}
}

func TestParseScriptTags(t *testing.T) {
	doc := parseFixture(t, "colour.ass")
	cue := doc.Cues[15]
	if c := findSpan(cue, "superscript"); c == nil || c.Script == nil || c.Script.Kind != model.ScriptSuperscript {
		t.Fatalf("\\ytsup missing: %+v", c)
	}
	if c := findSpan(cue, "subscript"); c == nil || c.Script == nil || c.Script.Kind != model.ScriptSubscript {
		t.Fatalf("\\ytsub missing: %+v", c)
	}
}

func TestParseStyleReset(t *testing.T) {
	doc := parseFixture(t, "colour.ass")
	cue := doc.Cues[16]
	if c := findSpan(cue, "Reset to a named style"); c == nil || c.Size == nil || *c.Size != 15 {
		t.Fatalf("\\rOutlineHeavy missing: %+v", c)
	}
}

func TestParseTopTitleStyle(t *testing.T) {
	doc := parseFixture(t, "colour.ass")
	cue := doc.Cues[17]
	span := cue.Spans[0]
	if span.Font == nil || *span.Font != "Some Custom Font" {
		t.Fatalf("top title font = %+v", span.Font)
	}
	if span.Size == nil || *span.Size != 40 {
		t.Fatalf("top title size = %+v", span.Size)
	}
	if cue.Layout == nil || cue.Layout.Anchor == nil || *cue.Layout.Anchor != model.AnchorTopCentre {
		t.Fatalf("top title anchor = %+v", cue.Layout)
	}
}

func TestParseRuby(t *testing.T) {
	doc := parseFixture(t, "cjk.ass")
	cue := doc.Cues[0]
	base := findSpan(cue, "漢")
	if base == nil || base.Ruby != nil {
		t.Fatalf("ruby base missing: %+v", base)
	}
	ann := findSpan(cue, "かん")
	if ann == nil || ann.Ruby == nil || ann.Ruby.Position != model.RubyOver {
		t.Fatalf("ruby annotation missing: %+v", ann)
	}
	under := doc.Cues[2]
	if a := findSpan(under, "ふ"); a == nil || a.Ruby == nil || a.Ruby.Position != model.RubyUnder {
		t.Fatalf("\\ytruby2 must place the reading below: %+v", a)
	}
}

func TestParseVerticalAndPacked(t *testing.T) {
	doc := parseFixture(t, "cjk.ass")
	modes := []model.VerticalMode{
		model.VerticalColumnsRTL,
		model.VerticalColumnsLTR,
		model.VerticalRotated,
		model.VerticalRotatedReversed,
	}
	for i, want := range modes {
		cue := doc.Cues[3+i]
		v := cue.Spans[0].Vertical
		if v == nil || v.Mode != want {
			t.Errorf("cue %d vertical = %+v, want %v", i, v, want)
		}
	}
	packed := doc.Cues[7]
	if !isTrue(packed.Spans[0].Packed) {
		t.Fatalf("\\ytpack1 missing: %+v", packed.Spans[0].Packed)
	}
}

func TestParseDirection(t *testing.T) {
	doc := parseFixture(t, "cjk.ass")
	cue := doc.Cues[9]
	if d := cue.Spans[1].Direction; d == nil || *d != model.DirRightToLeft {
		t.Fatalf("\\ytdir4 missing: %+v", d)
	}
}

func TestParseRubyWithKaraoke(t *testing.T) {
	doc := parseFixture(t, "cjk.ass")
	cue := doc.Cues[10]
	if !cue.Karaoke() {
		t.Fatal("ruby line must keep its karaoke timing")
	}
	base := findSpan(cue, "漢")
	if base == nil {
		t.Fatalf("ruby base missing: %+v", cue.Spans)
	}
	ann := findSpan(cue, "かん")
	if ann == nil || ann.Ruby == nil {
		t.Fatalf("ruby annotation missing: %+v", cue.Spans)
	}
	if base.Start != ann.Start || base.End != ann.End {
		t.Fatalf("annotation must inherit the base timing: %v..%v vs %v..%v",
			base.Start, base.End, ann.Start, ann.End)
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		{"0:00:01.00", time.Second},
		{"0:01:00.50", time.Minute + 500*time.Millisecond},
		{"100:00:00.00", 100 * time.Hour},
		{"", 0},
	}
	for _, tt := range tests {
		got, err := parseASSTime(tt.in)
		if err != nil {
			t.Errorf("parseASSTime(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseASSTime(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
	if _, err := parseASSTime("bad"); err == nil {
		t.Fatal("bad time must fail")
	}
}

func TestParseBadStyleLine(t *testing.T) {
	input := "[V4+ Styles]\nFormat: Name, Fontsize, PrimaryColour\nStyle: Default,nope,&H00FFFFFF\n"
	if _, err := NewReader().Parse(strings.NewReader(input)); err == nil {
		t.Fatal("bad font size must fail")
	}
}

func TestEmptyDocument(t *testing.T) {
	doc, err := NewReader().Parse(strings.NewReader("[Script Info]\nTitle: empty\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(doc.Styles) != 1 || doc.Styles[0].Name != "Default" {
		t.Fatalf("empty document needs a fallback style: %+v", doc.Styles)
	}
	if len(doc.Cues) != 0 {
		t.Fatalf("empty document has cues: %+v", doc.Cues)
	}
}

func findSpan(cue model.Cue, text string) *model.TextSpan {
	for i := range cue.Spans {
		if cue.Spans[i].Text == text {
			return &cue.Spans[i]
		}
	}
	return nil
}

func isTrue(v *bool) bool {
	return v != nil && *v
}
