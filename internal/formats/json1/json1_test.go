// SPDX-License-Identifier: Apache-2.0

package json1

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

func ptr[T any](v T) *T { return &v }

// richDocument returns a document that carries every field of the IR, so
// the round-trip test can prove that the format is lossless.
func richDocument() *model.Document {
	over, under, paren := model.RubyOver, model.RubyUnder, model.RubyParenthetical
	return &model.Document{
		Metadata:        map[string]string{"Title": "json1 fixture"},
		VideoDimensions: model.Point{X: 1280, Y: 720},
		Styles: []model.Style{
			{
				Name: "Default", Font: "Arial", Size: 20,
				Bold: true, Italic: true, Underline: true,
				Primary:      model.NewColour(255, 255, 255, 255),
				Secondary:    model.NewColour(120, 120, 120, 200),
				Outline:      model.NewColour(0, 0, 0, 128),
				Shadow:       model.NewColour(34, 34, 34, 255),
				OutlineWidth: 2, ShadowDepth: 3,
				Alignment: model.AnchorBottomCentre, Box: true,
			},
			{Name: "Second", Font: "Verdana", Size: 30, Alignment: model.AnchorTopRight},
		},
		Cues: []model.Cue{{
			Start: time.Second, End: 4 * time.Second,
			Layout: &model.Layout{
				Position: &model.Point{X: 100, Y: 200},
				Anchor:   ptr(model.AnchorTopLeft),
				Fade:     &model.Fade{In: time.Second, Out: 2 * time.Second, StartAlpha: 255, MidAlpha: 10, EndAlpha: 0, StartIn: 1, EndIn: 2, StartOut: 3, EndOut: 4},
				Move:     &model.Move{From: model.Point{X: 1, Y: 2}, To: model.Point{X: 3, Y: 4}, Start: time.Second, End: 2 * time.Second},
			},
			Spans: []model.TextSpan{
				{
					Text: "styled", Start: 100 * time.Millisecond, End: 500 * time.Millisecond,
					Font: ptr("Trebuchet MS"), Size: ptr(40.0),
					Bold: ptr(true), Italic: ptr(false), Underline: ptr(true),
					Strikeout: ptr(true), ScaleX: ptr(150.0), ScaleY: ptr(50.0),
					Fore: ptr(model.NewColour(255, 0, 0, 255)), Secondary: ptr(model.NewColour(0, 255, 0, 255)),
					Back:         ptr(model.NewColour(0, 0, 255, 255)),
					Shadows:      []model.Shadow{{Kind: model.ShadowSoft, Colour: model.NewColour(9, 8, 7, 6)}, {Kind: model.ShadowGlow, Colour: model.NewColour(1, 2, 3, 4)}},
					OutlineWidth: ptr(4.0), ShadowDepth: ptr(2.0),
					Vertical:  &model.Vertical{Mode: model.VerticalColumnsLTR, Packed: true},
					Script:    &model.Script{Kind: model.ScriptSuperscript},
					Direction: ptr(model.DirRightToLeft),
					Packed:    ptr(true),
				},
				{Text: "漢"},
				{Text: "かん", Ruby: &model.Ruby{Position: over}},
				{Text: "字"},
				{Text: "じ", Ruby: &model.Ruby{Position: under}},
				{Text: "空", Ruby: &model.Ruby{Position: paren}},
			},
			Animations: []model.Animation{
				{Fade: &model.Fade{In: time.Second}},
				{Move: &model.Move{From: model.Point{X: 1, Y: 2}, To: model.Point{X: 3, Y: 4}}},
				{Shake: &model.Shake{RadiusX: 5, RadiusY: 6, Start: time.Second, End: 2 * time.Second}},
				{Chroma: &model.Chroma{Offsets: []model.Point{{X: 1, Y: 2}}, InTime: 270 * time.Millisecond, OutTime: 270 * time.Millisecond, Colours: []model.Colour{model.NewColour(255, 0, 0, 255)}, Alpha: 191}},
				{Keyframes: []model.Keyframe{{Start: 0, End: time.Second, Easing: 2, Steps: []model.KeyframeStep{{Property: "fore", Value: "&H00FF00&"}}}}},
				{Karaoke: &model.Karaoke{Kind: model.KaraokeCursor, Cursor: "star", CursorTags: `\b1`, CursorLeft: true, CursorInterval: 100 * time.Millisecond, CursorFrames: []model.KaraokeFrame{{Tags: `\i1`, Text: "spin"}}}},
			},
		}},
	}
}

func TestRoundTripLossless(t *testing.T) {
	doc := richDocument()
	var out strings.Builder
	losses, err := NewWriter().Render(doc, &out)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(losses) != 0 {
		t.Fatalf("json1 must be lossless: %v", losses)
	}
	got, err := NewReader().Parse(strings.NewReader(out.String()))
	if err != nil {
		t.Fatalf("Parse: %v\n%s", err, out.String())
	}
	if !reflect.DeepEqual(doc, got) {
		t.Fatalf("round trip changed the document:\n want %+v\n  got %+v", doc, got)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"bad json", "{not json", "parse json1"},
		{"wrong version", `{"version":"2","document":{}}`, "unsupported version"},
		{"no document", `{"version":"1"}`, "no document"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewReader().Parse(strings.NewReader(tt.input))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Parse(%s) error = %v, want %q", tt.input, err, tt.want)
			}
		})
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("disk gone") }

func TestRenderWriteError(t *testing.T) {
	if _, err := NewWriter().Render(richDocument(), failWriter{}); err == nil {
		t.Fatal("a failing sink must surface an error")
	}
}

func TestNames(t *testing.T) {
	if NewReader().Name() != FormatName || NewWriter().Name() != FormatName {
		t.Fatalf("names must be %q", FormatName)
	}
}
