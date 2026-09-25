// SPDX-License-Identifier: Apache-2.0

package kdenlive

import (
	"errors"
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
	doc := parseFixture(t, "track.json")
	if len(doc.Cues) != 3 {
		t.Fatalf("cues = %d, want 3", len(doc.Cues))
	}
	if doc.Cues[0].Start != time.Second || doc.Cues[0].End != 4*time.Second {
		t.Errorf("first cue timing = %v..%v", doc.Cues[0].Start, doc.Cues[0].End)
	}
	if doc.Cues[0].Text() != "Hello world." {
		t.Errorf("first cue text = %q", doc.Cues[0].Text())
	}
	if doc.Cues[1].Start != 4500*time.Millisecond || doc.Cues[1].End != 8*time.Second {
		t.Errorf("second cue timing = %v..%v", doc.Cues[1].Start, doc.Cues[1].End)
	}
	if doc.Cues[2].Text() != "Two\nlines." {
		t.Errorf("third cue text = %q", doc.Cues[2].Text())
	}
}

func TestParseWithoutPrefix(t *testing.T) {
	source := `[{"layer":0,"startPos":2,"dialogue":"0,0:00:02.00,0:00:03.00,Default,,0,0,0,,No prefix."}]`
	doc, err := NewReader().Parse(strings.NewReader(source))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if doc.Cues[0].Text() != "No prefix." {
		t.Errorf("text = %q", doc.Cues[0].Text())
	}
}

// errReader fails every read.
type errReader struct{ err error }

func (e errReader) Read([]byte) (int, error) { return 0, e.err }

func TestParseErrors(t *testing.T) {
	boom := errors.New("boom")
	if _, err := NewReader().Parse(errReader{boom}); !errors.Is(err, boom) {
		t.Errorf("read error = %v, want %v", err, boom)
	}
	if _, err := NewReader().Parse(strings.NewReader("{")); err == nil {
		t.Error("malformed JSON must fail")
	}
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"fields", `[{"startPos":1,"dialogue":"a,b"}]`, "comma-separated fields"},
		{"time", `[{"startPos":1,"dialogue":"0,0:00:01.00,zz,Default,,0,0,0,,text"}]`, "parse ASS time"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewReader().Parse(strings.NewReader(tt.in))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestParseTimeErrors(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"0:00", "want h:mm:ss.cc"},
		{"x:00:00.00", "bad hours"},
		{"0:xx:00.00", "bad minutes"},
		{"0:00:xx.00", "bad seconds"},
		{"0:00:00.xx", "bad fraction"},
	}
	for _, tt := range tests {
		if _, err := parseTime(tt.in); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("parseTime(%q) error = %v, want %q", tt.in, err, tt.want)
		}
	}
}

func TestParseTimeShortFraction(t *testing.T) {
	got, err := parseTime("0:00:00.5")
	if err != nil || got != 500*time.Millisecond {
		t.Errorf("parseTime(0:00:00.5) = %v, %v", got, err)
	}
}

func TestStripTags(t *testing.T) {
	if got := stripTags(`{\i1}Hello{\i0} world`); got != "Hello world" {
		t.Errorf("stripTags = %q", got)
	}
	// A closing brace without an opening one stays as text.
	if got := stripTags("a}b"); got != "a}b" {
		t.Errorf("stripTags = %q", got)
	}
}

func TestFormatTime(t *testing.T) {
	if got := formatTime(-time.Second); got != "0:00:00.00" {
		t.Errorf("formatTime(-1s) = %q", got)
	}
	if got := formatTime(90*time.Minute + 1500*time.Millisecond); got != "1:30:01.50" {
		t.Errorf("formatTime = %q", got)
	}
}

func TestRenderRoundTrip(t *testing.T) {
	doc := parseFixture(t, "track.json")
	var out strings.Builder
	losses, err := NewWriter().Render(doc, &out)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(losses) != 0 {
		t.Errorf("losses = %v, want none", losses)
	}
	back, err := NewReader().Parse(strings.NewReader(out.String()))
	if err != nil {
		t.Fatalf("re-parse: %v\n%s", err, out.String())
	}
	if len(back.Cues) != len(doc.Cues) {
		t.Fatalf("cues = %d, want %d", len(back.Cues), len(doc.Cues))
	}
	for i := range doc.Cues {
		if back.Cues[i].Start != doc.Cues[i].Start || back.Cues[i].End != doc.Cues[i].End {
			t.Errorf("cue %d timing = %v..%v, want %v..%v", i,
				back.Cues[i].Start, back.Cues[i].End, doc.Cues[i].Start, doc.Cues[i].End)
		}
		if back.Cues[i].Text() != doc.Cues[i].Text() {
			t.Errorf("cue %d text = %q, want %q", i, back.Cues[i].Text(), doc.Cues[i].Text())
		}
	}
}

func TestRenderLosses(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues: []model.Cue{{
			Start: time.Second, End: 2 * time.Second,
			Layout:     &model.Layout{Position: &model.Point{X: 1}},
			Animations: []model.Animation{{}},
			Spans: []model.TextSpan{
				{
					Text: "text", Start: 0, End: time.Second,
					Bold: ptr(true), ScaleX: ptr(150.0),
					Vertical:  &model.Vertical{Mode: model.VerticalColumnsRTL},
					Packed:    ptr(true),
					Direction: ptr(model.DirRightToLeft),
					Script:    &model.Script{Kind: model.ScriptSubscript},
					Shadows:   []model.Shadow{{Kind: model.ShadowSoft}},
				},
				{Text: "かん", Ruby: &model.Ruby{}},
			},
		}},
	}
	var out strings.Builder
	losses, err := NewWriter().Render(doc, &out)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	joined := strings.Join(losses, "; ")
	for _, want := range []string{
		"karaoke timing", "positioning", "animation", "inline styling", "glyph scale",
		"vertical text", "packing", "right-to-left marking", "script offset", "ruby text",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("losses are missing %q: %v", want, losses)
		}
	}
}

// failWriter fails every write.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestRenderWriteError(t *testing.T) {
	doc := &model.Document{Cues: []model.Cue{{Start: 0, End: time.Second, Spans: []model.TextSpan{{Text: "text"}}}}}
	if _, err := NewWriter().Render(doc, failWriter{}); err == nil {
		t.Error("a failing sink must return an error")
	}
}
