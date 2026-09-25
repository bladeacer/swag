// SPDX-License-Identifier: Apache-2.0

package ass

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

func renderDoc(t *testing.T, doc *model.Document) (string, []string) {
	t.Helper()
	var out strings.Builder
	losses, err := NewWriter().Render(doc, &out)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return out.String(), losses
}

func reparse(t *testing.T, text string) *model.Document {
	t.Helper()
	doc, err := NewReader().Parse(strings.NewReader(text))
	if err != nil {
		t.Fatalf("re-parse: %v\n%s", err, text)
	}
	return doc
}

// spanView is the comparable rendering of one span. Fields that the style
// carries are resolved, and the override pointers stay as they are.
type spanView struct {
	text        string
	start       time.Duration
	end         time.Duration
	font        string
	size        float64
	bold        bool
	italic      bool
	underline   bool
	strikeout   bool
	scaleX      float64
	scaleY      float64
	fore        model.Colour
	secondary   model.Colour
	outline     float64
	shadowDepth float64
	shadows     []model.Shadow
	back        *model.Colour
	vertical    *model.Vertical
	script      *model.Script
	direction   *model.Direction
	packed      *bool
	ruby        *model.Ruby
}

func viewSpan(style model.Style, span model.TextSpan) spanView {
	r := model.Resolve(style, span)
	return spanView{
		text:        span.Text,
		start:       span.Start,
		end:         span.End,
		font:        r.Font,
		size:        r.Size,
		bold:        r.Bold,
		italic:      r.Italic,
		underline:   r.Underline,
		strikeout:   r.Strikeout,
		scaleX:      r.ScaleX,
		scaleY:      r.ScaleY,
		fore:        r.Fore,
		secondary:   r.Secondary,
		outline:     r.OutlineWidth,
		shadowDepth: r.ShadowDepth,
		shadows:     r.Shadows,
		back:        span.Back,
		vertical:    span.Vertical,
		script:      span.Script,
		direction:   span.Direction,
		packed:      span.Packed,
		ruby:        span.Ruby,
	}
}

func viewSpans(style model.Style, spans []model.TextSpan) []spanView {
	var out []spanView
	for _, span := range spans {
		v := viewSpan(style, span)
		// The writer merges neighbours that render the same way, and the
		// reader coalesces them again, so the comparison merges first.
		if n := len(out); n > 0 && sameSpanExceptText(out[n-1], v) {
			out[n-1].text += v.text
			continue
		}
		out = append(out, v)
	}
	if out == nil {
		out = []spanView{}
	}
	return out
}

// sameSpanExceptText reports whether two spans render identically apart
// from their text.
func sameSpanExceptText(a, b spanView) bool {
	a.text, b.text = "", ""
	return reflect.DeepEqual(a, b)
}

type layoutView struct {
	position *model.Point
	anchor   *model.Anchor
	fade     *model.Fade
	move     *model.Move
}

func viewLayout(l *model.Layout) layoutView {
	if l == nil {
		return layoutView{}
	}
	return layoutView{position: l.Position, anchor: l.Anchor, fade: l.Fade, move: l.Move}
}

// TestRoundTripFixtures checks that every committed fixture survives one
// ASS to IR to ASS pass with the same colours, timing, and structure. The
// comparison is semantic, so it allows the style names to flatten and the
// tag order to change.
func TestRoundTripFixtures(t *testing.T) {
	fixtures := []struct {
		name   string
		losses []string
	}{
		{name: "karaoke.ass"},
		{name: "colour.ass"},
		{name: "cjk.ass"},
		// The chroma fixture carries a conservative loss: the four
		// argument form names one offset, even though the symmetric
		// spread survives the pass.
		{name: "overrides.ass", losses: []string{"more than one copy"}},
	}
	for _, fixture := range fixtures {
		name := fixture.name
		t.Run(name, func(t *testing.T) {
			original := parseFixture(t, name)
			text, losses := renderDoc(t, original)
			again := reparse(t, text)

			base := baseStyle(original.Styles)
			baseAgain := baseStyle(again.Styles)

			if len(again.Cues) != len(original.Cues) {
				t.Fatalf("cue count changed: %d to %d", len(original.Cues), len(again.Cues))
			}
			for i := range original.Cues {
				want, got := original.Cues[i], again.Cues[i]
				if want.Start != got.Start || want.End != got.End {
					t.Errorf("cue %d timing changed: %v..%v to %v..%v", i, want.Start, want.End, got.Start, got.End)
				}
				if want.Text() != got.Text() {
					t.Errorf("cue %d text changed: %q to %q", i, want.Text(), got.Text())
				}
				if !reflect.DeepEqual(viewSpans(base, want.Spans), viewSpans(baseAgain, got.Spans)) {
					t.Errorf("cue %d spans changed:\n want %+v\n  got %+v", i,
						viewSpans(base, want.Spans), viewSpans(baseAgain, got.Spans))
				}
				if !reflect.DeepEqual(viewLayout(want.Layout), viewLayout(got.Layout)) {
					t.Errorf("cue %d layout changed:\n want %+v\n  got %+v", i,
						viewLayout(want.Layout), viewLayout(got.Layout))
				}
				if !reflect.DeepEqual(animationViews(want), animationViews(got)) {
					t.Errorf("cue %d animations changed:\n want %+v\n  got %+v", i,
						animationViews(want), animationViews(got))
				}
			}
			// An ASS source carries the features of the writer tiers, so
			// the pass is lossless apart from a documented chroma limit.
			for _, loss := range losses {
				allowed := false
				for _, want := range fixture.losses {
					if strings.Contains(loss, want) {
						allowed = true
						break
					}
				}
				if !allowed {
					t.Errorf("unexpected round trip loss: %q\n%s", loss, text)
				}
			}
		})
	}
}

// animationViews flattens the animation list into comparable values.
func animationViews(cue model.Cue) []any {
	var out []any
	for _, a := range cue.Animations {
		switch {
		case a.Shake != nil:
			out = append(out, *a.Shake)
		case a.Chroma != nil:
			out = append(out, *a.Chroma)
		case a.Keyframes != nil:
			out = append(out, a.Keyframes)
		case a.Karaoke != nil:
			out = append(out, *a.Karaoke)
		case a.Fade != nil:
			out = append(out, *a.Fade)
		case a.Move != nil:
			out = append(out, *a.Move)
		}
	}
	return out
}

func TestWriterHeaderAndStyleTable(t *testing.T) {
	doc := parseFixture(t, "colour.ass")
	text, _ := renderDoc(t, doc)
	for _, want := range []string{
		"[Script Info]",
		"[V4+ Styles]",
		"[Events]",
		"PlayResX: 1280",
		"PlayResY: 720",
		"Format: Name, Fontname, Fontsize",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("output is missing %q:\n%s", want, text)
		}
	}
	// The Default style is written first, and the box flag maps back to
	// BorderStyle 3.
	if !strings.Contains(text, "Style: Boxed,Verdana,20,&H00FFFFFF,&H000000FF,&H00000000,&H00000000,0,0,0,0,100,100,0,0,3,2,0,2,") {
		t.Errorf("box style line wrong:\n%s", text)
	}
}

func TestWriterEmitsTierOneToThreeTags(t *testing.T) {
	vertical := model.Vertical{Mode: model.VerticalColumnsRTL}
	rtl := model.DirRightToLeft
	sup := model.Script{Kind: model.ScriptSuperscript}
	packed := true
	doc := &model.Document{
		Styles: []model.Style{baseStyle(nil)},
		Cues: []model.Cue{{
			Start: time.Second, End: 5 * time.Second,
			Layout: &model.Layout{
				Anchor:   ptr(model.AnchorTopLeft),
				Position: &model.Point{X: 100, Y: 100},
				Move:     &model.Move{From: model.Point{X: 1, Y: 2}, To: model.Point{X: 3, Y: 4}},
				Fade:     &model.Fade{In: 500 * time.Millisecond, Out: 500 * time.Millisecond},
			},
			Spans: []model.TextSpan{
				{
					Text: "styled", Font: ptr("Verdana"), Size: ptr(30.0), Bold: ptr(true),
					Italic: ptr(true), Underline: ptr(true), Fore: ptr(model.NewColour(255, 0, 0, 255)),
					Secondary: ptr(model.NewColour(0, 255, 0, 255)), Back: ptr(model.NewColour(0, 0, 255, 255)),
					OutlineWidth: ptr(4.0), Vertical: &vertical, Direction: &rtl, Script: &sup, Packed: &packed,
					Shadows: []model.Shadow{{Kind: model.ShadowHard, Colour: model.NewColour(9, 8, 7, 6)}},
				},
			},
		}},
	}
	text, losses := renderDoc(t, doc)
	for _, want := range []string{
		`\an7`, `\pos(100,100)`, `\move(1,2,3,4)`, `\fad(500,500)`,
		`\fnVerdana`, `\fs30`, `\b1`, `\i1`, `\u1`,
		`\c&H0000FF&`, `\1a&H00&`, `\2c&H00FF00&`, `\3c&HFF0000&`,
		`\bord4`, `\ytvert9`, `\ytdir4`, `\ytsup`, `\ytpack1`, `\4c&H070809&`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("output is missing %q:\n%s", want, text)
		}
	}
	if len(losses) != 0 {
		t.Errorf("all Tier 1 to 3 tags must be expressible: %v", losses)
	}
}

func TestWriterEmitsKaraokeAndAnnotations(t *testing.T) {
	under := model.RubyUnder
	doc := &model.Document{
		Styles: []model.Style{baseStyle(nil)},
		Cues: []model.Cue{{
			Start: 0, End: 3 * time.Second,
			Layout: &model.Layout{Fade: &model.Fade{}},
			Animations: []model.Animation{
				{Keyframes: []model.Keyframe{{
					Start: 0, End: time.Second, Easing: 2,
					Steps: []model.KeyframeStep{{Property: "outline", Value: "&H0000FF&"}},
				}}},
				{Shake: &model.Shake{RadiusX: 10, RadiusY: 5, Start: 100 * time.Millisecond, End: 200 * time.Millisecond}},
				{Chroma: &model.Chroma{Offsets: []model.Point{{X: 5}}, InTime: 270 * time.Millisecond, OutTime: 270 * time.Millisecond}},
				{Karaoke: &model.Karaoke{Kind: model.KaraokeFade}},
			},
			Spans: []model.TextSpan{
				{Text: "Ka", Start: 0, End: time.Second},
				{Text: "ra", Start: time.Second, End: 2 * time.Second},
				{Text: "漢", Start: 2 * time.Second, End: 3 * time.Second},
				{Text: "かん", Start: 2 * time.Second, End: 3 * time.Second, Ruby: &model.Ruby{Position: under}},
			},
		}},
	}
	text, losses := renderDoc(t, doc)
	for _, want := range []string{
		`\k100`, `\t(0,1000,2,\3c&H0000FF&)`, `\ytshake(10,5,100,200)`,
		`\ytchroma`, `\ytktFade`, `\ytruby2`, "[漢/かん]",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("output is missing %q:\n%s", want, text)
		}
	}
	if len(losses) != 0 {
		t.Errorf("unexpected losses: %v", losses)
	}
}

// TestWriterEmitsOverrideTags checks the span overrides that have no style
// equivalent, the custom chroma form, and the cursor forms.
func TestWriterEmitsOverrideTags(t *testing.T) {
	strike := true
	scaleX, scaleY, depth := 150.0, 50.0, 4.0
	doc := &model.Document{
		Styles: []model.Style{baseStyle(nil)},
		Cues: []model.Cue{{
			Spans: []model.TextSpan{{
				Text:        "styled",
				Strikeout:   &strike,
				ScaleX:      &scaleX,
				ScaleY:      &scaleY,
				ShadowDepth: &depth,
			}},
			Animations: []model.Animation{
				{Chroma: &model.Chroma{
					Colours: []model.Colour{
						model.NewColour(255, 0, 0, 255),
						model.NewColour(0, 255, 0, 255),
						model.NewColour(0, 0, 255, 255),
					},
					Alpha: 191, InTime: 100 * time.Millisecond, OutTime: 200 * time.Millisecond,
				}},
				{Karaoke: &model.Karaoke{Kind: model.KaraokeCursor, Cursor: "star", CursorTags: `\b1`, CursorLeft: true}},
				{Karaoke: &model.Karaoke{
					Kind:           model.KaraokeCursor,
					CursorInterval: 100 * time.Millisecond,
					CursorFrames:   []model.KaraokeFrame{{Tags: `\i1`, Text: "spin"}, {Tags: `\i0`, Text: "star"}},
				}},
			},
		}},
	}
	text, losses := renderDoc(t, doc)
	for _, want := range []string{
		`\shad4`, `\s1`, `\fscx150`, `\fscy50`,
		`\ytchroma(&H0000FF&,&H00FF00&,&HFF0000&,&H40&,0,0,100,200)`,
		`\ytkt(LCursor,\b1,star)`,
		`\ytkt(Cursor,100,\i1,spin,\i0,star)`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("output is missing %q:\n%s", want, text)
		}
	}
	if len(losses) != 0 {
		t.Errorf("unexpected losses: %v", losses)
	}
}

func TestWriterHandlesScriptAndDirectionChanges(t *testing.T) {
	rtl := model.DirRightToLeft
	ltr := model.DirLeftToRight
	sub := model.Script{Kind: model.ScriptSubscript}
	doc := &model.Document{
		Styles: []model.Style{baseStyle(nil)},
		Cues: []model.Cue{{Spans: []model.TextSpan{
			{Text: "plain"},
			{Text: "rtl", Direction: &rtl},
			{Text: "back", Direction: &ltr},
			{Text: "sub", Script: &sub},
			{Text: "plain again"},
		}}},
	}
	text, losses := renderDoc(t, doc)
	if !strings.Contains(text, `\ytdir4`) || !strings.Contains(text, `\ytdir6`) {
		t.Errorf("direction changes must be emitted:\n%s", text)
	}
	if !strings.Contains(text, `\ytsub`) {
		t.Errorf("subscript tag missing:\n%s", text)
	}
	again := reparse(t, text)
	if got := again.Cues[0].Spans[2].Direction; got == nil || *got != model.DirLeftToRight {
		t.Errorf("a span that returns to left to right needs an explicit tag: %+v", got)
	}
	if got := again.Cues[0].Spans[1].Direction; got == nil || *got != model.DirRightToLeft {
		t.Errorf("right-to-left span lost its direction: %+v", got)
	}
	if len(losses) != 0 {
		t.Errorf("unexpected losses: %v", losses)
	}
}

func TestWriterReportsLosses(t *testing.T) {
	soft := true
	doc := &model.Document{
		Styles: []model.Style{baseStyle(nil)},
		Cues: []model.Cue{{
			Spans: []model.TextSpan{
				// A gap before the first karaoke segment cannot be
				// expressed: ASS starts the first \k at zero.
				{
					Text:   "late",
					Start:  time.Second,
					End:    2 * time.Second,
					Packed: &soft,
					Shadows: []model.Shadow{
						{Kind: model.ShadowSoft, Colour: model.NewColour(0, 0, 0, 255)},
						{Kind: model.ShadowHard, Colour: model.NewColour(0, 0, 0, 255)},
					},
				},
			},
			Animations: []model.Animation{
				{Chroma: &model.Chroma{Offsets: []model.Point{{X: 1}, {X: 2}, {X: 3}}}},
			},
		}},
	}
	_, losses := renderDoc(t, doc)
	joined := strings.Join(losses, "; ")
	for _, want := range []string{"karaoke gap", "shadow of kind", "more than one copy"} {
		if !strings.Contains(joined, want) {
			t.Errorf("loss report is missing %q: %v", want, losses)
		}
	}
}

func TestWriterReportsVerticalClear(t *testing.T) {
	vertical := model.Vertical{Mode: model.VerticalColumnsRTL}
	doc := &model.Document{
		Styles: []model.Style{baseStyle(nil)},
		Cues: []model.Cue{{Spans: []model.TextSpan{
			{Text: "vertical", Vertical: &vertical},
			{Text: "horizontal"},
		}}},
	}
	_, losses := renderDoc(t, doc)
	if !strings.Contains(strings.Join(losses, "; "), "vertical mode") {
		t.Fatalf("leaving vertical text must be reported: %v", losses)
	}
}

func TestWriterReportsMultipleReadings(t *testing.T) {
	over := model.RubyOver
	doc := &model.Document{
		Styles: []model.Style{baseStyle(nil)},
		Cues: []model.Cue{{Spans: []model.TextSpan{
			{Text: "漢"},
			{Text: "かん", Ruby: &model.Ruby{Position: over}},
			{Text: "かん2", Ruby: &model.Ruby{Position: over}},
		}}},
	}
	text, losses := renderDoc(t, doc)
	if !strings.Contains(text, "[漢/かん]") {
		t.Errorf("first reading missing:\n%s", text)
	}
	if !strings.Contains(strings.Join(losses, "; "), "several readings") {
		t.Fatalf("extra readings must be reported: %v", losses)
	}
}

func TestWriterParentheticalRuby(t *testing.T) {
	paren := model.RubyParenthetical
	doc := &model.Document{
		Styles: []model.Style{baseStyle(nil)},
		Cues: []model.Cue{{Spans: []model.TextSpan{
			{Text: "字"},
			{Text: "じ", Ruby: &model.Ruby{Position: paren}},
		}}},
	}
	text, _ := renderDoc(t, doc)
	if !strings.Contains(text, "字(じ)") {
		t.Fatalf("parenthetical ruby must fall back to brackets:\n%s", text)
	}
	if strings.Contains(text, `\ytruby`) {
		t.Fatalf("parenthetical ruby needs no ruby tag:\n%s", text)
	}
}

func TestWriterEmptyStyleList(t *testing.T) {
	doc := &model.Document{Cues: []model.Cue{{Spans: []model.TextSpan{{Text: "bare"}}}}}
	text, _ := renderDoc(t, doc)
	if !strings.Contains(text, "Style: Default,Arial,20,") {
		t.Fatalf("an empty style list needs the fallback style:\n%s", text)
	}
}

func TestWriterEscapesText(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{baseStyle(nil)},
		Cues:   []model.Cue{{Spans: []model.TextSpan{{Text: "a\\b\nc"}}}},
	}
	text, _ := renderDoc(t, doc)
	if !strings.Contains(text, `a\\b\Nc`) {
		t.Fatalf("backslash and line break must be escaped:\n%s", text)
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestWriterPropagatesWriteError(t *testing.T) {
	doc := &model.Document{Styles: []model.Style{baseStyle(nil)}}
	if _, err := NewWriter().Render(doc, failWriter{}); err == nil {
		t.Fatal("a write error must surface")
	}
}

func TestWriterMetadataRoundTrip(t *testing.T) {
	doc := parseFixture(t, "karaoke.ass")
	text, _ := renderDoc(t, doc)
	if !strings.Contains(text, "Title: swag karaoke fixture") {
		t.Fatalf("title must carry over:\n%s", text)
	}
	again := reparse(t, text)
	if again.Metadata["Title"] != "swag karaoke fixture" {
		t.Fatalf("title did not read back: %+v", again.Metadata)
	}
}

func TestFormatASSTime(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{0, "0:00:00.00"},
		{time.Second, "0:00:01.00"},
		{time.Minute + 500*time.Millisecond, "0:01:00.50"},
		{-time.Second, "0:00:00.00"},
	}
	for _, tt := range tests {
		if got := formatASSTime(tt.in); got != tt.want {
			t.Errorf("formatASSTime(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatColourIsTransparency(t *testing.T) {
	// Alpha 255 is opaque, so the transparency byte is zero.
	if got := formatColour(model.NewColour(0x12, 0x34, 0x56, 255)); got != "&H00563412" {
		t.Errorf("formatColour = %q", got)
	}
	if got := formatColour(model.NewColour(0, 0, 0, 0)); got != "&HFF000000" {
		t.Errorf("formatColour of a clear colour = %q", got)
	}
}

func ptr[T any](v T) *T { return &v }
