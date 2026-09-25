// SPDX-License-Identifier: Apache-2.0

package ass

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/swag/internal/formats/json1"
	"github.com/bladeacer/swag/internal/formats/kdenlive"
	"github.com/bladeacer/swag/internal/formats/sbv"
	"github.com/bladeacer/swag/internal/formats/srt"
	"github.com/bladeacer/swag/internal/formats/srv3"
	"github.com/bladeacer/swag/internal/formats/ttml"
	"github.com/bladeacer/swag/internal/formats/vtt"
	"github.com/bladeacer/swag/internal/formats/ytt"
	"github.com/bladeacer/swag/internal/model"
)

// The cross-check suite runs a document through a second format and back.
// Each direction keeps the timing and the text, reports the features it
// cannot express, and keeps the features the other format does share.

// crossTarget describes one shipped writer for the suite.
type crossTarget struct {
	name   string
	write  func(*model.Document, io.Writer) ([]string, error)
	read   func(io.Reader) (*model.Document, error)
	losses []string
}

// crossTargets lists every shipped writer. The ASS colour fixture is the
// source of the suite, so the loss lists name what that fixture provokes.
var crossTargets = []crossTarget{
	{
		name:  "ass",
		write: func(d *model.Document, w io.Writer) ([]string, error) { return NewWriter().Render(d, w) },
		read:  func(r io.Reader) (*model.Document, error) { return NewReader().Parse(r) },
	},
	{
		name:   "ytt",
		write:  func(d *model.Document, w io.Writer) ([]string, error) { return ytt.NewWriter().Render(d, w) },
		read:   func(r io.Reader) (*model.Document, error) { return ytt.NewReader().Parse(r) },
		losses: []string{"cue fade", "cue move", "keyframe animation", `font "Fancy Font"`},
	},
	{
		name:   "srv3",
		write:  func(d *model.Document, w io.Writer) ([]string, error) { return srv3.NewWriter().Render(d, w) },
		read:   func(r io.Reader) (*model.Document, error) { return srv3.NewReader().Parse(r) },
		losses: []string{"cue fade", "cue move", "keyframe animation", `font "Fancy Font"`},
	},
	{
		name:   "srt",
		write:  srt.NewWriter().Render,
		read:   srt.NewReader().Parse,
		losses: []string{"positioning", "animation", "foreground colour", "transparency", "shadow effects", "script offset"},
	},
	{
		name:   "sbv",
		write:  sbv.NewWriter().Render,
		read:   sbv.NewReader().Parse,
		losses: []string{"positioning", "inline styling"},
	},
	{
		name:  "vtt",
		write: vtt.NewWriter().Render,
		read:  vtt.NewReader().Parse,
		losses: []string{
			"font", "font size", "foreground colour", "secondary colour", "background colour",
			"shadow effects", "outline width", "vertical alignment", "cue move", "cue fade",
			"animation", "script offset",
		},
	},
	{
		name:   "json1",
		write:  json1.NewWriter().Render,
		read:   json1.NewReader().Parse,
		losses: nil,
	},
	{
		name:   "ttml",
		write:  ttml.NewWriter().Render,
		read:   ttml.NewReader().Parse,
		losses: []string{"shadow effects", "cue move", "cue fade", "animation", "script offset"},
	},
	{
		name:   "kdenlive",
		write:  kdenlive.NewWriter().Render,
		read:   kdenlive.NewReader().Parse,
		losses: []string{"inline styling", "positioning", "animation", "script offset"},
	},
}

// TestCrossCheckEveryWriter runs the ASS colour fixture through every
// shipped writer. Each target keeps the cue count, the timing, and the
// text, and it reports the features it drops.
func TestCrossCheckEveryWriter(t *testing.T) {
	source := parseFixture(t, "colour.ass")
	for _, target := range crossTargets {
		t.Run(target.name, func(t *testing.T) {
			var out strings.Builder
			losses, err := target.write(source, &out)
			if err != nil {
				t.Fatalf("render %s: %v", target.name, err)
			}
			joined := lossText(losses)
			for _, want := range target.losses {
				if !strings.Contains(joined, want) {
					t.Errorf("%s loss report is missing %q: %v", target.name, want, losses)
				}
			}
			if len(target.losses) == 0 && len(losses) > 0 {
				t.Errorf("%s must carry the fixture without loss: %v", target.name, losses)
			}
			again, err := target.read(strings.NewReader(out.String()))
			if err != nil {
				t.Fatalf("re-parse %s: %v\n%s", target.name, err, out.String())
			}
			assertStable(t, source, again)
		})
	}
}

// TestCrossCheckOverrideLosses runs the override fixture through every
// shipped writer. The fixture is an ASS source, so the ASS writer carries it
// without loss. Each other writer reports the overrides it drops, and the
// text and the cue timing survive everywhere.
func TestCrossCheckOverrideLosses(t *testing.T) {
	source := parseFixture(t, "overrides.ass")
	expect := map[string][]string{
		"ass":  nil,
		"ytt":  {"strikeout", "glyph scale", "chroma animation", "karaoke type"},
		"srv3": {"strikeout", "glyph scale", "chroma animation", "karaoke type"},
		"srt":  {"strikeout", "glyph scale", "positioning", "animation", "shadow effects"},
		"sbv":  {"inline styling", "glyph scale", "positioning"}, "vtt": {
			"shadow effects", "vertical alignment", "strikeout", "glyph scale", "animation", "vertical text",
		},
		"json1":    nil,
		"ttml":     {"shadow effects", "glyph scale", "animation", "vertical text"},
		"kdenlive": {"inline styling", "positioning", "glyph scale", "animation", "vertical text"},
	}
	for _, target := range crossTargets {
		t.Run(target.name, func(t *testing.T) {
			var out strings.Builder
			losses, err := target.write(source, &out)
			if err != nil {
				t.Fatalf("render %s: %v", target.name, err)
			}
			joined := lossText(losses)
			for _, want := range expect[target.name] {
				if !strings.Contains(joined, want) {
					t.Errorf("%s loss report is missing %q: %v", target.name, want, losses)
				}
			}
			if len(expect[target.name]) == 0 && len(losses) > 0 {
				t.Errorf("%s must carry the fixture without loss: %v", target.name, losses)
			}
			again, err := target.read(strings.NewReader(out.String()))
			if err != nil {
				t.Fatalf("re-parse %s: %v\n%s", target.name, err, out.String())
			}
			assertStable(t, source, again)
		})
	}
}

// TestCrossCheckChains runs one ASS source through a sequence of formats and
// back. The chains cover two-way passes and passes of three or more formats.
// The text and the cue timing survive every chain, so the readers and the
// writers compose.
// chainSource is one reader fixture for the chain suite. A source that
// carries ruby text stays inside the formats that carry ruby, so the text
// comparison stays fair.
type chainSource struct {
	name   string
	load   func(*testing.T) *model.Document
	chains [][]string
}

// parseSRTSource reads a small SubRip document for the chain suite.
func parseSRTSource(t *testing.T) *model.Document {
	t.Helper()
	doc, err := srt.NewReader().Parse(strings.NewReader(
		"1\n00:00:01,000 --> 00:00:04,000\n<i>Hello</i> world.\n\n" +
			"2\n00:00:04,500 --> 00:00:08,000\nSecond line.\n"))
	if err != nil {
		t.Fatalf("parse SRT source: %v", err)
	}
	return doc
}

// parseSBVSource reads a small SBV document for the chain suite.
func parseSBVSource(t *testing.T) *model.Document {
	t.Helper()
	doc, err := sbv.NewReader().Parse(strings.NewReader(
		"0:00:01.000,0:00:04.000\nHello world.\n\n" +
			"0:00:04.500,0:00:08.000\nSecond line.\n"))
	if err != nil {
		t.Fatalf("parse SBV source: %v", err)
	}
	return doc
}

// parseVTTSource reads a small WebVTT document for the chain suite.
func parseVTTSource(t *testing.T) *model.Document {
	t.Helper()
	doc, err := vtt.NewReader().Parse(strings.NewReader(
		"WEBVTT\n\n00:00:01.000 --> 00:00:04.000\n<i>Hello</i> world.\n\n" +
			"00:00:04.500 --> 00:00:08.000\nSecond line.\n"))
	if err != nil {
		t.Fatalf("parse VTT source: %v", err)
	}
	return doc
}

// parseKdenliveSource reads a small Kdenlive subtitle document for the
// chain suite.
func parseKdenliveSource(t *testing.T) *model.Document {
	t.Helper()
	doc, err := kdenlive.NewReader().Parse(strings.NewReader(
		`[{"layer":0,"startPos":1,"dialogue":"Dialogue: 0,0:00:01.00,0:00:04.00,Default,,0,0,0,,Hello world."},` +
			`{"layer":0,"startPos":4.5,"dialogue":"Dialogue: 0,0:00:04.50,0:00:08.00,Default,,0,0,0,,Second line."}]`))
	if err != nil {
		t.Fatalf("parse Kdenlive source: %v", err)
	}
	return doc
}

// readSRV3Fixture reads the SRV3 sample for the chain suite.
func readSRV3Fixture(t *testing.T) *model.Document {
	t.Helper()
	data, err := os.ReadFile("../srv3/testdata/sample.srv3")
	if err != nil {
		t.Skipf("SRV3 fixture is unavailable: %v", err)
	}
	doc, err := srv3.NewReader().Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("parse SRV3 fixture: %v", err)
	}
	return doc
}

// parseTTMLSource reads a small TTML document for the chain suite.
func parseTTMLSource(t *testing.T) *model.Document {
	t.Helper()
	doc, err := ttml.NewReader().Parse(strings.NewReader(
		`<tt xmlns="http://www.w3.org/ns/ttml"><head></head><body><div>` +
			`<p begin="00:00:01.000" end="00:00:04.000"><span tts:fontStyle="italic">Hello</span> world.</p>` +
			`<p begin="00:00:04.500" end="00:00:08.000">Second line.</p>` +
			`</div></body></tt>`))
	if err != nil {
		t.Fatalf("parse TTML source: %v", err)
	}
	return doc
}

// parseJSON1Source reads a lossless JSON1 document for the chain suite. The
// JSON1 form carries every IR field, so the source starts from the ASS
// colour fixture.
func parseJSON1Source(t *testing.T) *model.Document {
	t.Helper()
	start := parseFixture(t, "colour.ass")
	var buf strings.Builder
	if _, err := json1.NewWriter().Render(start, &buf); err != nil {
		t.Fatalf("render JSON1 source: %v", err)
	}
	doc, err := json1.NewReader().Parse(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("parse JSON1 source: %v", err)
	}
	return doc
}

// TestCrossCheckChains runs one fixture per reader through a sequence of
// formats and compares the result with the source. The chains cover
// two-way passes and passes of three or more formats, so the readers and
// the writers compose.
func TestCrossCheckChains(t *testing.T) {
	byName := map[string]crossTarget{}
	for _, target := range crossTargets {
		byName[target.name] = target
	}
	sources := []chainSource{
		{
			name: "colour.ass",
			load: func(t *testing.T) *model.Document { return parseFixture(t, "colour.ass") },
			chains: [][]string{
				{"ytt", "ass"},
				{"srv3", "ass"},
				{"srt", "ass"},
				{"sbv", "ass"},
				{"ytt", "srt", "ass"},
				{"srv3", "sbv", "ass"},
				{"ytt", "srv3", "srt", "sbv", "ass"},
			},
		},
		{
			name:   "karaoke.ass",
			load:   func(t *testing.T) *model.Document { return parseFixture(t, "karaoke.ass") },
			chains: [][]string{{"ytt", "ass"}, {"srt", "sbv", "ass"}},
		},
		{
			name:   "overrides.ass",
			load:   func(t *testing.T) *model.Document { return parseFixture(t, "overrides.ass") },
			chains: [][]string{{"ytt", "ass"}, {"srv3", "srt", "sbv", "ass"}},
		},
		{
			name:   "ytt-sample",
			load:   readYTTFixture,
			chains: [][]string{{"ass", "ytt"}, {"srv3", "ytt"}, {"ass", "srv3", "ass", "ytt"}},
		},
		{
			name:   "srv3-source",
			load:   readSRV3Fixture,
			chains: [][]string{{"ass", "srv3"}, {"ytt", "ass", "srv3"}, {"srt", "sbv", "ass", "srv3"}},
		},
		{
			name:   "srt-source",
			load:   parseSRTSource,
			chains: [][]string{{"ass", "srt"}, {"sbv", "ass", "srt"}, {"ytt", "ass", "sbv", "srt"}},
		},
		{
			name:   "sbv-source",
			load:   parseSBVSource,
			chains: [][]string{{"ass", "sbv"}, {"srt", "ass", "sbv"}, {"ass", "ytt", "srt", "sbv"}},
		},
		{
			name:   "vtt-source",
			load:   parseVTTSource,
			chains: [][]string{{"ass", "vtt"}, {"json1", "ass", "vtt"}, {"srt", "sbv", "ass", "vtt"}},
		},
		{
			name:   "json1-source",
			load:   parseJSON1Source,
			chains: [][]string{{"ass", "json1"}, {"vtt", "srt", "ass", "json1"}, {"ytt", "ass", "json1"}},
		},
		{
			name:   "ttml-source",
			load:   parseTTMLSource,
			chains: [][]string{{"ass", "ttml"}, {"json1", "ass", "ttml"}, {"sbv", "srt", "ass", "ttml"}},
		},
		{
			name:   "kdenlive-source",
			load:   parseKdenliveSource,
			chains: [][]string{{"ass", "kdenlive"}, {"vtt", "ass", "kdenlive"}, {"srt", "sbv", "ass", "kdenlive"}},
		},
	}
	for _, source := range sources {
		t.Run(source.name, func(t *testing.T) {
			doc := source.load(t)
			for _, chain := range source.chains {
				t.Run(strings.Join(chain, "-"), func(t *testing.T) {
					current := doc
					for _, step := range chain {
						target := byName[step]
						var out strings.Builder
						if _, err := target.write(current, &out); err != nil {
							t.Fatalf("write %s: %v", step, err)
						}
						next, err := target.read(strings.NewReader(out.String()))
						if err != nil {
							t.Fatalf("read %s: %v\n%s", step, err, out.String())
						}
						current = next
					}
					assertStable(t, doc, current)
				})
			}
		})
	}
}

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

// TestCrossCheckPlainToASS starts from SubRip, which is the plainest source
// the project reads. The ASS pass keeps the text and the italic override,
// and the SubRip pass back settles on the same cues.
func TestCrossCheckPlainToASS(t *testing.T) {
	const source = "1\n00:00:01,000 --> 00:00:04,000\n<i>Hello</i> world.\n\n" +
		"2\n00:00:04,500 --> 00:00:08,000\nSecond line.\n"

	start, err := srt.NewReader().Parse(strings.NewReader(source))
	if err != nil {
		t.Fatalf("parse SRT: %v", err)
	}

	var assOut strings.Builder
	if _, err := NewWriter().Render(start, &assOut); err != nil {
		t.Fatalf("render ASS: %v", err)
	}
	mid := reparse(t, assOut.String())
	assertStable(t, start, mid)
	if !isTrue(mid.Cues[0].Spans[0].Italic) {
		t.Errorf("the italic override did not survive into ASS: %+v", mid.Cues[0].Spans[0])
	}

	var srtOut strings.Builder
	if _, err := srt.NewWriter().Render(mid, &srtOut); err != nil {
		t.Fatalf("render SRT: %v", err)
	}
	final, err := srt.NewReader().Parse(strings.NewReader(srtOut.String()))
	if err != nil {
		t.Fatalf("re-parse SRT: %v\n%s", err, srtOut.String())
	}
	assertStable(t, start, final)
}
