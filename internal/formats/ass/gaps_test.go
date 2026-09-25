// SPDX-License-Identifier: Apache-2.0

package ass

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

type failingReader struct{ err error }

func (f failingReader) Read([]byte) (int, error) { return 0, f.err }

func TestWriterName(t *testing.T) {
	if got := NewWriter().Name(); got != FormatName {
		t.Fatalf("writer name = %q", got)
	}
}

func TestParseScannerError(t *testing.T) {
	if _, err := NewReader().Parse(failingReader{errors.New("boom")}); err == nil {
		t.Fatal("a failing reader must surface an error")
	}
}

func TestParseStyleErrors(t *testing.T) {
	format := []string{
		"name", "fontname", "fontsize", "primarycolour", "secondarycolour",
		"outlinecolour", "backcolour", "bold", "italic", "underline",
		"borderstyle", "outline", "shadow", "alignment",
	}
	base := []string{
		"Default", "Arial", "20", "&H00FFFFFF", "&H0000FF&", "&H00000000",
		"&H00000000", "0", "0", "0", "1", "2", "2", "2",
	}
	tests := []struct {
		name string
		at   int
		bad  string
	}{
		{"primary colour", 3, "&Hzz&"},
		{"secondary colour", 4, "&Hzz&"},
		{"outline colour", 5, "&Hzz&"},
		{"back colour", 6, "&Hzz&"},
		{"border style", 10, "x"},
		{"outline width", 11, "x"},
		{"shadow depth", 12, "x"},
		{"alignment", 13, "x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := append([]string(nil), base...)
			fields[tt.at] = tt.bad
			if _, err := parseStyle(fields, format); err == nil {
				t.Fatalf("a bad %s must fail", tt.name)
			}
		})
	}
}

// TestParseStyleDefaults covers an empty style name, an empty font name, a
// blank number, and a format without a border style or alignment.
func TestParseStyleDefaults(t *testing.T) {
	format := []string{
		"name", "fontname", "fontsize", "primarycolour", "secondarycolour",
		"outlinecolour", "backcolour", "outline", "shadow",
	}
	fields := []string{
		"", "", "20", "&H00FFFFFF", "&H0000FF&", "&H00000000", "&H00000000", "", "",
	}
	style, err := parseStyle(fields, format)
	if err != nil {
		t.Fatalf("parseStyle: %v", err)
	}
	if style.Name != "Default" {
		t.Errorf("name = %q, want Default", style.Name)
	}
	if style.Font != "Arial" {
		t.Errorf("font = %q, want Arial", style.Font)
	}
	if style.OutlineWidth != 0 || style.ShadowDepth != 0 {
		t.Errorf("blank widths must take the default: %+v", style)
	}
	if style.Box || style.Alignment != model.AnchorBottomCentre {
		t.Errorf("missing border style and alignment must take the default: %+v", style)
	}
}

func TestParseFlagTruthy(t *testing.T) {
	if !parseFlag("yes") || !parseFlag("-1") || !parseFlag("2") {
		t.Fatal("truthy flag forms must read as true")
	}
	if parseFlag("no") || parseFlag("0") || parseFlag("") || parseFlag("false") {
		t.Fatal("false flag forms must read as false")
	}
}

const gapsHeader = `[Script Info]
ScriptType: v4.00+
PlayResX: 1280
PlayResY: 720

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,Arial,20,&H00FFFFFF,&H00777777,&H00000000,&H64000000,0,0,0,0,100,100,0,0,1,2,2,2,10,10,10,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`

const gapsDialogue = `Dialogue: 0,0:00:01.00,0:00:02.00,Default,,0,0,0,,{\clip(1,2,3)}tier four
Dialogue: 0,0:00:02.00,0:00:03.00,Default,,0,0,0,,{\bord}blank border
Dialogue: 0,0:00:03.00,0:00:04.00,Default,,0,0,0,,{\c}blank fore {\2a}blank secondary {\4a}blank shadow {\alpha}blank all
Dialogue: 0,0:00:04.00,0:00:05.00,Default,,0,0,0,,{\c&Hzz&}bad colour
Dialogue: 0,0:00:05.00,0:00:06.00,Default,,0,0,0,,{\k bad}bad karaoke
Dialogue: 0,0:00:06.00,0:00:07.00,Default,,0,0,0,,{\ytvert5}bad vertical
Dialogue: 0,0:00:07.00,0:00:08.00,Default,,0,0,0,,{\b}blank flag {\b maybe}text flag {\b2}number flag
Dialogue: 0,0:00:08.00,0:00:09.00,Default,,0,0,0,,{\pos(1)}bad pos {\move(1,2,3)}bad move
Dialogue: 0,0:00:09.00,0:00:10.00,Default,,0,0,0,,{\fad(1)}short fad {\fade(1,2,3)}short fade
Dialogue: 0,0:00:10.00,0:00:11.00,Default,,0,0,0,,{\fade(a,b,c,d,e,f,g)}bad fade numbers
Dialogue: 0,0:00:11.00,0:00:12.00,Default,,0,0,0,,{\fade(-1,300,0,0,0,0,0)}clamped fade
Dialogue: 0,0:00:12.00,0:00:13.00,Default,,0,0,0,,{\t(0,1000,)}empty keyframe
Dialogue: 0,0:00:13.00,0:00:14.00,Default,,0,0,0,,{\t(0,1000,\2c&HFF0000&\1a&H10&\2a&H20&\3a&H30&\4a&H40&\alpha&H50&\bord1&\shad2\x1)}keyframe values
Dialogue: 0,0:00:14.00,0:00:15.00,Default,,0,0,0,,{\ytshake(a)}bad shake
Dialogue: 0,0:00:15.00,0:00:16.00,Default,,0,0,0,,{\ytshake(1,2,3)}radius and times
Dialogue: 0,0:00:16.00,0:00:17.00,Default,,0,0,0,,{\ytshake(1,2,3,4)}radii and times
Dialogue: 0,0:00:17.00,0:00:18.00,Default,,0,0,0,,{\ytchroma(&HFF0000&,255,5,6,100)}five chroma arguments
Dialogue: 0,0:00:18.00,0:00:19.00,Default,,0,0,0,,{\ytkt(Foo)}unknown karaoke type
Dialogue: 0,0:00:19.00,0:00:20.00,Default,,0,0,0,,{\ytruby}[unclosed
Dialogue: 0,0:00:20.00,0:00:21.00,Default,,0,0,0,,{\ytruby}[noslash]
Dialogue: 0,0:00:21.00,0:00:22.00,Default,,0,0,0,,{\ytruby}[base/reading] after
Dialogue: 0,0:00:22.00,0:00:23.00,Default,,0,0,0,,{\1a&Hzz&}bad alpha {\alpha&Hzz&}bad all alpha {\pos(a,b)}bad pos values {\move(a,b,c,d)}bad move values {\fad(a,b)}bad fad values {\t}bare keyframe {\ytshake(1,2)}two radii
`

func parseGaps(t *testing.T, lines string) *model.Document {
	t.Helper()
	doc, err := NewReader().Parse(strings.NewReader(gapsHeader + lines))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return doc
}

// TestParseTagFallbacks covers the tag forms that reset, default, or carry
// too few values.
func TestParseTagFallbacks(t *testing.T) {
	doc := parseGaps(t, gapsDialogue)
	if len(doc.Cues) != 22 {
		t.Fatalf("got %d cues, want 22", len(doc.Cues))
	}
	// A blank border tag returns to the style width.
	if got := doc.Cues[1].Spans[0].OutlineWidth; got != nil {
		t.Errorf("a blank \\bord must reset to the style: %+v", got)
	}
	// A blank alpha tag returns to the style colour.
	if got := doc.Cues[2].Spans[0].Secondary; got != nil {
		t.Errorf("a blank \\2a must reset to the style: %+v", got)
	}
	if got := doc.Cues[2].Spans[0].Fore; got != nil {
		t.Errorf("a blank \\c must reset to the style: %+v", got)
	}
	// A bad colour value stays on the previous colour.
	if got := doc.Cues[3].Spans[0].Fore; got != nil {
		t.Errorf("a bad colour must change nothing: %+v", got)
	}
	// Incomplete argument tags are ignored.
	if doc.Cues[7].Layout != nil && doc.Cues[7].Layout.Position != nil {
		t.Errorf("\\pos(1) must be ignored: %+v", doc.Cues[7].Layout)
	}
	// A clamped complex fade keeps its alphas inside the range.
	if fade := doc.Cues[10].Layout.Fade; fade == nil || fade.StartAlpha != 0 || fade.MidAlpha != 255 {
		t.Errorf("clamped fade = %+v", fade)
	}
	// A karaoke type that is not a cursor is ignored.
	for _, a := range doc.Cues[17].Animations {
		if a.Karaoke != nil {
			t.Errorf("\\ytkt(Foo) must give no karaoke type: %+v", a.Karaoke)
		}
	}
	// Ruby without a closing bracket or a separator keeps its text.
	if got := doc.Cues[18].Text(); got != "[unclosed" {
		t.Errorf("unclosed ruby text = %q", got)
	}
	if got := doc.Cues[19].Text(); got != "noslash" {
		t.Errorf("ruby without a separator = %q", got)
	}
	if base := findSpan(doc.Cues[20], "base"); base == nil {
		t.Errorf("ruby base missing: %+v", doc.Cues[20].Spans)
	}
	if ann := findSpan(doc.Cues[20], "reading"); ann == nil || ann.Ruby == nil {
		t.Errorf("ruby reading missing: %+v", doc.Cues[20].Spans)
	}
	// A blank flag returns to the style value, and an unreadable one too.
	if got := doc.Cues[6].Spans[0].Bold; got != nil {
		t.Errorf("a blank \\b must reset to the style: %+v", got)
	}
	if got := doc.Cues[6].Spans[0].Italic; got != nil {
		t.Errorf("a text \\b value must reset to the style: %+v", got)
	}
	// A bad alpha value and a bad argument tag change nothing.
	if got := doc.Cues[21].Spans[0].Fore; got != nil {
		t.Errorf("a bad \\1a must change nothing: %+v", got)
	}
	twoRadii := firstAnimation(doc.Cues[21], func(a model.Animation) bool { return a.Shake != nil })
	if twoRadii == nil || twoRadii.Shake.RadiusX != 1 || twoRadii.Shake.RadiusY != 2 {
		t.Errorf("a two value shake must read both radii: %+v", twoRadii)
	}
}

// TestParseKeyframeValues covers every keyframe property name.
func TestParseKeyframeValues(t *testing.T) {
	doc := parseGaps(t, gapsDialogue)
	var steps []model.KeyframeStep
	for _, a := range doc.Cues[12].Animations {
		for _, kf := range a.Keyframes {
			steps = append(steps, kf.Steps...)
		}
	}
	want := []string{
		"secondary", "forealpha", "secondaryalpha", "outlinealpha",
		"shadowalpha", "alpha", "outlinewidth", "shadowdepth", "x",
	}
	if len(steps) != len(want) {
		t.Fatalf("got %d steps, want %d: %+v", len(steps), len(want), steps)
	}
	for i, name := range want {
		if steps[i].Property != name {
			t.Errorf("step %d property = %q, want %q", i, steps[i].Property, name)
		}
	}
}

// TestParseShakeForms covers the shake argument shapes.
func TestParseShakeForms(t *testing.T) {
	doc := parseGaps(t, gapsDialogue)
	bad := firstAnimation(doc.Cues[13], func(a model.Animation) bool { return a.Shake != nil })
	if bad == nil || bad.Shake.RadiusX != 20 || bad.Shake.RadiusY != 20 {
		t.Fatalf("a bad shake must keep the default radii: %+v", bad)
	}
	radiusTimes := firstAnimation(doc.Cues[14], func(a model.Animation) bool { return a.Shake != nil })
	if radiusTimes == nil || radiusTimes.Shake.RadiusY != radiusTimes.Shake.RadiusX {
		t.Fatalf("a three value shake must reuse the radius: %+v", radiusTimes)
	}
	radiiTimes := firstAnimation(doc.Cues[15], func(a model.Animation) bool { return a.Shake != nil })
	if radiiTimes == nil || radiiTimes.Shake.RadiusX != 1 || radiiTimes.Shake.RadiusY != 2 {
		t.Fatalf("a four value shake must read both radii: %+v", radiiTimes)
	}
}

func TestParseEventTimeErrors(t *testing.T) {
	bad := gapsHeader + "Dialogue: 0,x:00:01.00,0:00:02.00,Default,,0,0,0,,bad start\n"
	if _, err := NewReader().Parse(strings.NewReader(bad)); err == nil {
		t.Fatal("a bad start time must fail")
	}
	badEnd := gapsHeader + "Dialogue: 0,0:00:01.00,x:00:02.00,Default,,0,0,0,,bad end\n"
	if _, err := NewReader().Parse(strings.NewReader(badEnd)); err == nil {
		t.Fatal("a bad end time must fail")
	}
}

func TestParseASSTimeErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"shape", "bad"},
		{"hours", "x:00:00.00"},
		{"minutes", "0:x:00.00"},
		{"seconds", "0:00:x.00"},
		{"centiseconds", "0:00:01.xx"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseASSTime(tt.in); err == nil {
				t.Fatalf("parseASSTime(%q) must fail", tt.in)
			}
		})
	}
}

func TestParseASSTimeFraction(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		{"0:00:01.5", time.Second + 500*time.Millisecond},
		{"0:00:01.123", time.Second + 120*time.Millisecond},
	}
	for _, tt := range tests {
		got, err := parseASSTime(tt.in)
		if err != nil {
			t.Fatalf("parseASSTime(%q): %v", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("parseASSTime(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestColourNumberErrors(t *testing.T) {
	if _, err := parseColour("12x", 255); err == nil {
		t.Fatal("a bad decimal colour must fail")
	}
	if _, err := parseTransparency("zz"); err == nil {
		t.Fatal("a bad alpha body must fail")
	}
}

func TestSpreadOffsetsFloor(t *testing.T) {
	if got := spreadOffsets(1, 2, 0); len(got) != 1 {
		t.Fatalf("spreadOffsets with no copies must give one offset: %+v", got)
	}
}

// TestWriterTagEdges covers the writer branches that the fixtures do not
// reach.
func TestWriterTagEdges(t *testing.T) {
	glow := model.Shadow{Kind: model.ShadowGlow, Colour: model.NewColour(1, 2, 3, 4)}
	base := baseStyle(nil)
	// An alignment outside the numpad range falls back to bottom centre.
	base.Alignment = model.Anchor(0)
	doc := &model.Document{
		Styles: []model.Style{base},
		Cues: []model.Cue{
			{
				Layout: &model.Layout{
					Fade: &model.Fade{
						StartAlpha: 255, MidAlpha: 0, EndAlpha: 255,
						StartIn: 0, EndIn: time.Second, StartOut: 2 * time.Second, EndOut: 3 * time.Second,
					},
				},
				Animations: []model.Animation{
					{Fade: &model.Fade{In: time.Second, Out: 2 * time.Second}},
					{Move: &model.Move{From: model.Point{X: 1, Y: 2}, To: model.Point{X: 3, Y: 4}, Start: time.Second}},
					{Shake: &model.Shake{RadiusX: 7, RadiusY: 8}},
					{Karaoke: &model.Karaoke{Kind: model.KaraokeGlitch}},
					{Karaoke: &model.Karaoke{Kind: model.KaraokeCursor}},
					{Karaoke: &model.Karaoke{Kind: model.KaraokeCursor, Cursor: "star"}},
					{Karaoke: &model.Karaoke{Kind: model.KaraokeKind(99)}},
					{Keyframes: []model.Keyframe{{
						Start: 0, End: time.Second, Easing: 1,
						Steps: []model.KeyframeStep{
							{Property: "secondary", Value: "&HFF0000&"},
							{Property: "shadow", Value: "&H00FF00&"},
							{Property: "forealpha", Value: "&H10&"},
							{Property: "secondaryalpha", Value: "&H20&"},
							{Property: "outlinealpha", Value: "&H30&"},
							{Property: "shadowalpha", Value: "&H40&"},
							{Property: "outlinewidth", Value: "2"},
							{Property: "shadowdepth", Value: "3"},
							{Property: "unknown", Value: "9"},
						},
					}}},
				},
				Spans: []model.TextSpan{
					{Text: "plain", Bold: ptr(false), Shadows: []model.Shadow{glow}},
				},
			},
			{
				Animations: []model.Animation{{Keyframes: []model.Keyframe{{Start: 0, End: time.Second, Easing: 1}}}},
				Spans:      []model.TextSpan{{Text: "empty keyframe"}},
			},
		},
	}
	text, losses := renderDoc(t, doc)
	for _, want := range []string{
		`\fade(255,0,255,0,1000,2000,3000)`, `\fad(1000,2000)`,
		`\move(1,2,3,4,1000,0)`, `\ytshake(7,8)`, `\ytktGlitch`, `\ytkt(Cursor,star)`,
		`\3c&H030201&\3a&HFB&`, `\b0`, `\t(0,1000,\2c`, `\shad3`,
		",2,10,10,10,1",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("output is missing %q:\n%s", want, text)
		}
	}
	joined := strings.Join(losses, "; ")
	for _, want := range []string{"cursor karaoke type with no text", "keyframe with no animated value"} {
		if !strings.Contains(joined, want) {
			t.Errorf("loss report is missing %q: %v", want, losses)
		}
	}
}

// TestWriterDefaultNames covers the fallback style name and font.
func TestWriterDefaultNames(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{{Size: 20, Alignment: model.AnchorBottomCentre}},
		Cues:   []model.Cue{{Spans: []model.TextSpan{{Text: "bare"}}}},
	}
	text, _ := renderDoc(t, doc)
	if !strings.Contains(text, "Style: Default,Arial,20,") {
		t.Fatalf("the fallback style name and font are missing:\n%s", text)
	}
}
