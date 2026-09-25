// SPDX-License-Identifier: Apache-2.0

package ass

import (
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

const syntheticDoc = `[Script Info]
ScriptType: v4.00+
PlayResX: 1280
PlayResY: 720

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,Arial,20,&H00FFFFFF,&H00777777,&H00000000,&H64000000,0,0,0,0,100,100,0,0,1,2,2,2,10,10,10,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
Dialogue: 0,0:00:01.00,0:00:05.00,Default,,0,0,0,,{\ytshake(10,5,100,200)}shake
Dialogue: 0,0:00:05.00,0:00:09.00,Default,,0,0,0,,{\ytshake}default shake
Dialogue: 0,0:00:09.00,0:00:13.00,Default,,0,0,0,,{\ytshake(8)}radius shake
Dialogue: 0,0:00:13.00,0:00:17.00,Default,,0,0,0,,{\ytchroma}chroma
Dialogue: 0,0:00:17.00,0:00:21.00,Default,,0,0,0,,{\ytchroma(100,200)}chroma times
Dialogue: 0,0:00:21.00,0:00:25.00,Default,,0,0,0,,{\ytchroma(5,6,100,200)}chroma offsets
Dialogue: 0,0:00:25.00,0:00:29.00,Default,,0,0,0,,{\ytchroma(&HFF0000&,&H00FF00&,&H0000FF&,255,5,6,100,200)}chroma colours
Dialogue: 0,0:00:29.00,0:00:33.00,Default,,0,0,0,,{\ytchroma(&HFF0000&,255,5,6,100,200)}chroma one colour
Dialogue: 0,0:00:33.00,0:00:37.00,Default,,0,0,0,,{\ytktFade}fade karaoke
Dialogue: 0,0:00:37.00,0:00:41.00,Default,,0,0,0,,{\ytktGlitch}glitch karaoke
Dialogue: 0,0:00:41.00,0:00:45.00,Default,,0,0,0,,{\ytkt(Cursor,star)}cursor karaoke
Dialogue: 0,0:00:45.00,0:00:49.00,Default,,0,0,0,,{\fade(255,0,255,0,1000,2000,3000)}complex fade
Dialogue: 0,0:00:49.00,0:00:53.00,Default,,0,0,0,,{\t(0,1000,2,\fs30\c&H0000FF&)}keyframe
Dialogue: 0,0:00:53.00,0:00:57.00,Default,,0,0,0,,{\move(1,2,3,4,100,500)}move timed
Dialogue: 0,0:00:57.00,0:01:01.00,Default,,0,0,0,,{\ytdir6}left to right
Dialogue: 0,0:01:01.00,0:01:05.00,Default,,0,0,0,,{\1a}alpha reset
Dialogue: 0,0:01:05.00,0:01:09.00,Default,,0,0,0,,{\b true}{\i no}{\u yes}flags
Dialogue: 0,0:01:09.00,0:01:13.00,Default,,0,0,0,,{\an99}{\pos}{\move}{\fad(1)}{\t(9)}unfinished
`

func parseSynthetic(t *testing.T) *model.Document {
	t.Helper()
	doc, err := NewReader().Parse(strings.NewReader(syntheticDoc))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return doc
}

func firstAnimation(cue model.Cue, pick func(model.Animation) bool) *model.Animation {
	for i := range cue.Animations {
		if pick(cue.Animations[i]) {
			return &cue.Animations[i]
		}
	}
	return nil
}

func TestShakeArguments(t *testing.T) {
	doc := parseSynthetic(t)
	shake := firstAnimation(doc.Cues[0], func(a model.Animation) bool { return a.Shake != nil })
	if shake == nil {
		t.Fatalf("shake missing: %+v", doc.Cues[0].Animations)
	}
	if shake.Shake.RadiusX != 10 || shake.Shake.RadiusY != 5 {
		t.Fatalf("radii = %+v", shake.Shake)
	}
	if shake.Shake.Start != 100*time.Millisecond || shake.Shake.End != 200*time.Millisecond {
		t.Fatalf("window = %v..%v", shake.Shake.Start, shake.Shake.End)
	}
	def := firstAnimation(doc.Cues[1], func(a model.Animation) bool { return a.Shake != nil })
	if def.Shake.RadiusX != 20 || def.Shake.RadiusY != 20 {
		t.Fatalf("default radii = %+v", def.Shake)
	}
	single := firstAnimation(doc.Cues[2], func(a model.Animation) bool { return a.Shake != nil })
	if single.Shake.RadiusX != 8 || single.Shake.RadiusY != 8 {
		t.Fatalf("single radius = %+v", single.Shake)
	}
}

func TestChromaArguments(t *testing.T) {
	doc := parseSynthetic(t)
	def := firstAnimation(doc.Cues[3], func(a model.Animation) bool { return a.Chroma != nil })
	if def.Chroma.InTime != 270*time.Millisecond || len(def.Chroma.Offsets) != 3 {
		t.Fatalf("default chroma = %+v", def.Chroma)
	}
	times := firstAnimation(doc.Cues[4], func(a model.Animation) bool { return a.Chroma != nil })
	if times.Chroma.InTime != 100*time.Millisecond || times.Chroma.OutTime != 200*time.Millisecond {
		t.Fatalf("chroma times = %+v", times.Chroma)
	}
	offsets := firstAnimation(doc.Cues[5], func(a model.Animation) bool { return a.Chroma != nil })
	if offsets.Chroma.Offsets[0].X != -5 || offsets.Chroma.Offsets[2].X != 5 {
		t.Fatalf("chroma offsets = %+v", offsets.Chroma.Offsets)
	}
	colours := firstAnimation(doc.Cues[6], func(a model.Animation) bool { return a.Chroma != nil })
	if len(colours.Chroma.Offsets) != 3 {
		t.Fatalf("chroma colour copies = %+v", colours.Chroma.Offsets)
	}
	one := firstAnimation(doc.Cues[7], func(a model.Animation) bool { return a.Chroma != nil })
	if len(one.Chroma.Offsets) != 1 || one.Chroma.Offsets[0].X != 0 {
		t.Fatalf("single colour copy = %+v", one.Chroma.Offsets)
	}
}

func TestKaraokeTypes(t *testing.T) {
	doc := parseSynthetic(t)
	fade := firstAnimation(doc.Cues[8], func(a model.Animation) bool { return a.Karaoke != nil })
	if fade.Karaoke.Kind != model.KaraokeFade {
		t.Fatalf("fade karaoke = %+v", fade.Karaoke)
	}
	glitch := firstAnimation(doc.Cues[9], func(a model.Animation) bool { return a.Karaoke != nil })
	if glitch.Karaoke.Kind != model.KaraokeGlitch {
		t.Fatalf("glitch karaoke = %+v", glitch.Karaoke)
	}
	cursor := firstAnimation(doc.Cues[10], func(a model.Animation) bool { return a.Karaoke != nil })
	if cursor.Karaoke.Kind != model.KaraokeCursor || cursor.Karaoke.Cursor != "star" {
		t.Fatalf("cursor karaoke = %+v", cursor.Karaoke)
	}
}

func TestComplexFade(t *testing.T) {
	doc := parseSynthetic(t)
	fade := firstAnimation(doc.Cues[11], func(a model.Animation) bool { return a.Fade != nil })
	if fade == nil {
		t.Fatalf("complex fade missing: %+v", doc.Cues[11].Animations)
	}
	if fade.Fade.StartAlpha != 255 || fade.Fade.EndAlpha != 255 {
		t.Fatalf("fade alphas = %+v", fade.Fade)
	}
	if fade.Fade.EndOut != 3*time.Second {
		t.Fatalf("fade end = %v", fade.Fade.EndOut)
	}
}

func TestKeyframeProperties(t *testing.T) {
	doc := parseSynthetic(t)
	var steps []model.KeyframeStep
	var easing float64
	for _, a := range doc.Cues[12].Animations {
		for _, kf := range a.Keyframes {
			steps = append(steps, kf.Steps...)
			easing = kf.Easing
		}
	}
	if len(steps) != 2 {
		t.Fatalf("steps = %+v", steps)
	}
	if steps[0].Property != "fontsize" || steps[0].Value != "30" {
		t.Fatalf("fontsize step = %+v", steps[0])
	}
	if steps[1].Property != "fore" {
		t.Fatalf("colour step = %+v", steps[1])
	}
	if easing != 2 {
		t.Fatalf("easing = %v", easing)
	}
}

func TestMoveTimesAndDirection(t *testing.T) {
	doc := parseSynthetic(t)
	move := doc.Cues[13].Layout.Move
	if move.Start != 100*time.Millisecond || move.End != 500*time.Millisecond {
		t.Fatalf("move times = %v..%v", move.Start, move.End)
	}
	if d := doc.Cues[14].Spans[0].Direction; d == nil || *d != model.DirLeftToRight {
		t.Fatalf("\\ytdir6 = %+v", d)
	}
}

func TestAlphaResetAndFlags(t *testing.T) {
	doc := parseSynthetic(t)
	if doc.Cues[15].Spans[0].Fore != nil {
		t.Fatalf("\\1a with no value must reset: %+v", doc.Cues[15].Spans[0].Fore)
	}
	span := doc.Cues[16].Spans[0]
	if !isTrue(span.Bold) || isTrue(span.Italic) || !isTrue(span.Underline) {
		t.Fatalf("flag forms wrong: %+v", span)
	}
}

func TestIncompleteTagsAreIgnored(t *testing.T) {
	doc := parseSynthetic(t)
	cue := doc.Cues[17]
	// \an99 is out of range, and the argument tags carry too few values, so
	// none of them may change the cue.
	if cue.Layout != nil && cue.Layout.Move != nil {
		t.Fatalf("incomplete move must be ignored: %+v", cue.Layout.Move)
	}
	if len(cue.Animations) != 0 {
		t.Fatalf("incomplete fade and keyframe must be ignored: %+v", cue.Animations)
	}
}

func TestColourForms(t *testing.T) {
	if c, err := parseColour("&H0000FF&", 255); err != nil || c != model.NewColour(255, 0, 0, 255) {
		t.Fatalf("six digit colour = %+v, %v", c, err)
	}
	if c, err := parseColour("16777215", 255); err != nil || c != model.NewColour(255, 255, 255, 255) {
		t.Fatalf("decimal colour = %+v, %v", c, err)
	}
	if _, err := parseColour("", 255); err == nil {
		t.Fatal("empty colour must fail")
	}
	if _, err := parseColour("&Hzz&", 255); err == nil {
		t.Fatal("bad hex must fail")
	}
	if a, err := parseTransparency("&H80&"); err != nil || a != 127 {
		t.Fatalf("alpha = %d, %v", a, err)
	}
	if a, err := parseTransparency(""); err != nil || a != 255 {
		t.Fatalf("blank alpha = %d, %v", a, err)
	}
	if _, err := parseTransparency("&H100&"); err == nil {
		t.Fatal("out of range alpha must fail")
	}
}

func TestHelperEdgeCases(t *testing.T) {
	if anchor(0) != model.AnchorBottomCentre || anchor(99) != model.AnchorBottomCentre {
		t.Fatal("out of range alignment must fall back")
	}
	if v, err := parseInt("", 3); err != nil || v != 3 {
		t.Fatalf("blank integer = %d, %v", v, err)
	}
	if _, err := parseInt("x", 0); err == nil {
		t.Fatal("bad integer must fail")
	}
	if _, err := parseFloat("x", 0); err == nil {
		t.Fatal("bad float must fail")
	}
	if _, v, ok := splitKeyValue("no colon"); ok || v != "" {
		t.Fatalf("line without a colon = %q %v", v, ok)
	}
	if got := splitN("a,b,c", 0); len(got) != 1 {
		t.Fatalf("splitN with n 0 = %v", got)
	}
	if millis("x") != 0 {
		t.Fatal("bad milliseconds must be zero")
	}
	if karaokeFromArgs(nil) != nil {
		t.Fatal("empty karaoke args must give nothing")
	}
}
