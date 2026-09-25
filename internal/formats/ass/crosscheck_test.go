// SPDX-License-Identifier: Apache-2.0

package ass

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/swag/internal/formats/ytt"
	"github.com/bladeacer/swag/internal/model"
)

// The cross-check suite runs a document through the second format and back.
// Each direction keeps the timing and the text, reports the features it
// cannot express, and keeps the features the other format does share.

func readYTTFixture(t *testing.T) *model.Document {
	t.Helper()
	data, err := os.ReadFile("../ytt/testdata/sample.ytt")
	if err != nil {
		t.Skipf("YTT fixture is unavailable: %v", err)
	}
	doc, err := ytt.NewReader().Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("parse YTT fixture: %v", err)
	}
	return doc
}

// assertStable checks that two documents carry the same cues, timing, and
// text. A writer can re-split spans and rename styles, so only the cue-level
// shape is compared.
func assertStable(t *testing.T, want, got *model.Document) {
	t.Helper()
	if len(want.Cues) != len(got.Cues) {
		t.Fatalf("cue count changed: %d to %d", len(want.Cues), len(got.Cues))
	}
	for i := range want.Cues {
		if want.Cues[i].Start != got.Cues[i].Start || want.Cues[i].End != got.Cues[i].End {
			t.Errorf("cue %d timing changed: %v..%v to %v..%v", i,
				want.Cues[i].Start, want.Cues[i].End, got.Cues[i].Start, got.Cues[i].End)
		}
		if want.Cues[i].Text() != got.Cues[i].Text() {
			t.Errorf("cue %d text changed: %q to %q", i, want.Cues[i].Text(), got.Cues[i].Text())
		}
	}
}

func lossText(losses []string) string { return strings.Join(losses, "; ") }

// TestCrossCheckASSToYTT converts the ASS colour fixture to YouTube Timed
// Text and back. The pass must report the ASS effects YouTube Timed Text
// cannot express, and must keep the timing, text, and styling that it can.
func TestCrossCheckASSToYTT(t *testing.T) {
	original := parseFixture(t, "colour.ass")

	var yttOut strings.Builder
	losses, err := ytt.NewWriter().Render(original, &yttOut)
	if err != nil {
		t.Fatalf("render YTT: %v", err)
	}
	joined := lossText(losses)
	for _, want := range []string{
		"cue fade", "cue move", "keyframe animation", "not a YouTube font",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("YTT loss report is missing %q: %v", want, losses)
		}
	}

	mid, err := ytt.NewReader().Parse(strings.NewReader(yttOut.String()))
	if err != nil {
		t.Fatalf("re-parse YTT: %v\n%s", err, yttOut.String())
	}
	assertStable(t, original, mid)

	// Styling that YouTube Timed Text does share survives the pass.
	if !isTrue(mid.Cues[1].Spans[0].Bold) {
		t.Errorf("bold did not survive into YTT: %+v", mid.Cues[1].Spans[0])
	}
	if red := mid.Cues[4].Spans[0].Fore; red == nil || red.R != 255 || red.G != 0 || red.B != 0 {
		t.Errorf("red did not survive into YTT: %+v", red)
	}

	var assOut strings.Builder
	if _, err := NewWriter().Render(mid, &assOut); err != nil {
		t.Fatalf("render ASS: %v", err)
	}
	final := reparse(t, assOut.String())
	assertStable(t, original, final)
}

// TestCrossCheckASSToYTTCJK checks the CJK features, which both formats
// share: ruby, vertical layout, packing, and direction.
func TestCrossCheckASSToYTTCJK(t *testing.T) {
	original := parseFixture(t, "cjk.ass")

	var yttOut strings.Builder
	if _, err := ytt.NewWriter().Render(original, &yttOut); err != nil {
		t.Fatalf("render YTT: %v", err)
	}
	mid, err := ytt.NewReader().Parse(strings.NewReader(yttOut.String()))
	if err != nil {
		t.Fatalf("re-parse YTT: %v\n%s", err, yttOut.String())
	}
	assertStable(t, original, mid)

	if v := mid.Cues[3].Spans[0].Vertical; v == nil || v.Mode != model.VerticalColumnsRTL {
		t.Errorf("vertical columns did not survive: %+v", v)
	}
	if p := mid.Cues[7].Spans[0].Packed; p == nil || !*p {
		t.Errorf("packing did not survive: %+v", p)
	}
	if d := mid.Cues[9].Spans[1].Direction; d == nil || *d != model.DirRightToLeft {
		t.Errorf("right to left direction did not survive: %+v", d)
	}
	base := findSpan(mid.Cues[0], "漢")
	ann := findSpan(mid.Cues[0], "かん")
	if base == nil || ann == nil || ann.Ruby == nil || ann.Ruby.Position != model.RubyOver {
		t.Errorf("ruby did not survive: base %+v annotation %+v", base, ann)
	}

	var assOut strings.Builder
	if _, err := NewWriter().Render(mid, &assOut); err != nil {
		t.Fatalf("render ASS: %v", err)
	}
	final := reparse(t, assOut.String())
	assertStable(t, original, final)
	if a := findSpan(final.Cues[2], "ふ"); a == nil || a.Ruby == nil || a.Ruby.Position != model.RubyUnder {
		t.Errorf("under ruby did not survive the full pass: %+v", a)
	}
}

// TestCrossCheckASSToYTTKaraoke checks that karaoke timing survives both
// directions.
func TestCrossCheckASSToYTTKaraoke(t *testing.T) {
	original := parseFixture(t, "karaoke.ass")

	var yttOut strings.Builder
	if _, err := ytt.NewWriter().Render(original, &yttOut); err != nil {
		t.Fatalf("render YTT: %v", err)
	}
	mid, err := ytt.NewReader().Parse(strings.NewReader(yttOut.String()))
	if err != nil {
		t.Fatalf("re-parse YTT: %v\n%s", err, yttOut.String())
	}
	assertStable(t, original, mid)

	// The first cue tiles its syllables in both formats.
	want := []struct{ start, end time.Duration }{
		{0, 420 * time.Millisecond},
		{420 * time.Millisecond, 800 * time.Millisecond},
		{800 * time.Millisecond, 1300 * time.Millisecond},
	}
	if !mid.Cues[0].Karaoke() {
		t.Fatalf("the karaoke cue lost its timing: %+v", mid.Cues[0].Spans)
	}
	for i, w := range want {
		if mid.Cues[0].Spans[i].Start != w.start || mid.Cues[0].Spans[i].End != w.end {
			t.Errorf("span %d = %v..%v, want %v..%v", i,
				mid.Cues[0].Spans[i].Start, mid.Cues[0].Spans[i].End, w.start, w.end)
		}
	}

	var assOut strings.Builder
	if _, err := NewWriter().Render(mid, &assOut); err != nil {
		t.Fatalf("render ASS: %v", err)
	}
	final := reparse(t, assOut.String())
	assertStable(t, original, final)
	for i, w := range want {
		if final.Cues[0].Spans[i].Start != w.start || final.Cues[0].Spans[i].End != w.end {
			t.Errorf("full pass span %d = %v..%v, want %v..%v", i,
				final.Cues[0].Spans[i].Start, final.Cues[0].Spans[i].End, w.start, w.end)
		}
	}
}

// TestCrossCheckYTTToASS runs the other direction over the YTT fixture. ASS
// can express every feature of the sample, so the pass reports no loss, and
// a second YTT pass settles on the same text and timing.
func TestCrossCheckYTTToASS(t *testing.T) {
	original := readYTTFixture(t)

	var assOut strings.Builder
	losses, err := NewWriter().Render(original, &assOut)
	if err != nil {
		t.Fatalf("render ASS: %v", err)
	}
	if len(losses) > 0 {
		t.Fatalf("ASS must express every feature of the YTT sample, got losses: %v", losses)
	}

	mid := reparse(t, assOut.String())
	assertStable(t, original, mid)

	// Position, anchor, and vertical layout survive into ASS.
	vertical := mid.Cues[8]
	if vertical.Layout == nil || vertical.Layout.Anchor == nil || *vertical.Layout.Anchor != model.AnchorTopLeft {
		t.Errorf("anchor did not survive into ASS: %+v", vertical.Layout)
	}
	if pos := vertical.Layout.Position; pos == nil || pos.X != 0 || pos.Y != 72 {
		t.Errorf("position did not survive into ASS: %+v", pos)
	}
	if v := vertical.Spans[0].Vertical; v == nil || v.Mode != model.VerticalColumnsRTL {
		t.Errorf("vertical mode did not survive into ASS: %+v", v)
	}
	ann := findSpan(mid.Cues[12], "かん")
	if ann == nil || ann.Ruby == nil || ann.Ruby.Position != model.RubyOver {
		t.Errorf("ruby did not survive into ASS: %+v", ann)
	}

	var yttOut strings.Builder
	if _, err := ytt.NewWriter().Render(mid, &yttOut); err != nil {
		t.Fatalf("render YTT again: %v", err)
	}
	final, err := ytt.NewReader().Parse(strings.NewReader(yttOut.String()))
	if err != nil {
		t.Fatalf("re-parse YTT: %v\n%s", err, yttOut.String())
	}
	assertStable(t, mid, final)
}
