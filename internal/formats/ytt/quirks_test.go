// SPDX-License-Identifier: Apache-2.0

package ytt

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bladeacer/swag/internal/model"
)

// TestFontFamilyTable pins the canonical name and every allow-list alias of
// each YouTube font choice. The reader and the writer both read this table,
// so a changed row reaches both.
func TestFontFamilyTable(t *testing.T) {
	tests := []struct {
		style     FontStyle
		canonical string
		names     []string
	}{
		{FontDefault, Roboto, nil},
		{FontMonospaceSerif, "Courier New", []string{"Courier New"}},
		{FontProportionalSerif, "Times New Roman", []string{"Times New Roman", "Georgia"}},
		{FontMonospaceSans, "Lucida Console", []string{"Lucida Console"}},
		{FontProportionalSans, Roboto, []string{
			"Arial", "Arial Black", "Arial Narrow", "Impact", "Roboto",
			"Tahoma", "Trebuchet MS", "Verdana",
		}},
		{FontCasual, "Comic Sans MS", []string{"Comic Sans MS"}},
		{FontCursive, "Monotype Corsiva", []string{"Monotype Corsiva"}},
		{FontSmallCaps, "Arial Small Caps", nil},
	}
	for _, tt := range tests {
		if got := FontName(tt.style); got != tt.canonical {
			t.Errorf("FontName(%d) = %q, want %q", tt.style, got, tt.canonical)
		}
		for _, name := range tt.names {
			if got, ok := LookupFont(name); !ok || got != tt.style {
				t.Errorf("LookupFont(%q) = %d, %v, want %d, true", name, got, ok, tt.style)
			}
			if got, ok := LookupFont(strings.ToUpper(name)); !ok || got != tt.style {
				t.Errorf("LookupFont of the upper case %q = %d, %v, want %d, true", name, got, ok, tt.style)
			}
		}
	}
}

// TestFontSnapsUnknownToRoboto pins the fallback for a name outside the
// allow-list, including the empty name and a name with surrounding space.
func TestFontSnapsUnknownToRoboto(t *testing.T) {
	for _, name := range []string{"", "  ", "Fancy Font", "Roboto Condensed", "Arial Unicode", "Iosevka", "宋体"} {
		style, ok := LookupFont(name)
		if ok {
			t.Errorf("LookupFont(%q) reported a match", name)
		}
		if style != FontProportionalSans {
			t.Errorf("LookupFont(%q) = %d, want the default sans", name, style)
		}
		if got := FontName(style); got != Roboto {
			t.Errorf("FontName of the fallback = %q, want %q", got, Roboto)
		}
	}
}

// TestWriterEmitsEveryWritableFont covers the writer path of each font
// choice that the allow-list can resolve, so every "fs" value is exercised.
func TestWriterEmitsEveryWritableFont(t *testing.T) {
	for style := FontMonospaceSerif; style <= FontCursive; style++ {
		name := FontName(style)
		if got, ok := LookupFont(name); !ok || got != style {
			t.Fatalf("the canonical name %q does not resolve to %d", name, style)
		}
		doc := &model.Document{
			Styles: []model.Style{DefaultStyle()},
			Cues:   []model.Cue{{Spans: []model.TextSpan{{Text: "x", Font: &name}}}},
		}
		var out strings.Builder
		if _, err := NewWriter().Render(doc, &out); err != nil {
			t.Fatalf("Render %d: %v", style, err)
		}
		want := fmt.Sprintf(`fs="%d"`, int(style))
		if !strings.Contains(out.String(), want) {
			t.Errorf("font %q must write %s:\n%s", name, want, out.String())
		}
	}
}

// TestWriterSnapsTheSmallCapsFamily pins the documented asymmetry. The
// reader maps the "fs" value 7 onto "Arial Small Caps", which is outside the
// allow-list, so a write snaps it back to Roboto and records the loss.
func TestWriterSnapsTheSmallCapsFamily(t *testing.T) {
	name := FontName(FontSmallCaps)
	if name != "Arial Small Caps" {
		t.Fatalf("the small caps family = %q", name)
	}
	if _, ok := LookupFont(name); ok {
		t.Fatalf("%q must stay outside the allow-list", name)
	}
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues:   []model.Cue{{Spans: []model.TextSpan{{Text: "Small", Font: &name}}}},
	}
	var out strings.Builder
	losses, err := NewWriter().Render(doc, &out)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out.String(), `fs="4"`) {
		t.Fatalf("the small caps family must snap to the default sans:\n%s", out.String())
	}
	if !strings.Contains(strings.Join(losses, "; "), name) {
		t.Fatalf("the loss report must name the snapped family: %v", losses)
	}
}

// TestWriterShadowStealsTheLineStart covers the shadow variant of the
// prefetch space. YouTube trims the leading shadow, so the line gains one
// space at the start.
func TestWriterShadowStealsTheLineStart(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues: []model.Cue{{Spans: []model.TextSpan{{
			Text:    "Shadow",
			Shadows: []model.Shadow{{Kind: model.ShadowSoft, Colour: model.NewColour(0, 0, 0, 255)}},
		}}}},
	}
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out.String(), "> Shadow</p>") {
		t.Fatalf("a leading shadow must steal a space:\n%s", out.String())
	}
}

// TestClampOpacityCeiling pins the alpha ceiling of every colour slot. The
// upload drops a colour attribute whose opacity is 255, so the writer caps
// the value at 254 and keeps the attribute.
func TestClampOpacityCeiling(t *testing.T) {
	tests := []struct{ in, want uint8 }{
		{255, 254},
		{254, 254},
		{128, 128},
		{0, 0},
	}
	for _, tt := range tests {
		if got := clampOpacity(model.NewColour(10, 20, 30, tt.in)); got.A != tt.want {
			t.Errorf("clampOpacity of %d = %d, want %d", tt.in, got.A, tt.want)
		}
	}
}

// TestWhiteShiftAndDarkLift pins the two colour workarounds of the upload
// path, and leaves every other colour unchanged.
func TestWhiteShiftAndDarkLift(t *testing.T) {
	whites := []struct {
		name string
		in   model.Colour
		want model.Colour
	}{
		{"white", model.NewColour(255, 255, 255, 255), model.NewColour(0xFE, 0xFE, 0xFE, 255)},
		{"near white", model.NewColour(254, 254, 254, 255), model.NewColour(254, 254, 254, 255)},
		{"red", model.NewColour(255, 0, 0, 255), model.NewColour(255, 0, 0, 255)},
	}
	for _, tt := range whites {
		if got := shiftWhite(tt.in); got != tt.want {
			t.Errorf("shiftWhite(%s) = %+v, want %+v", tt.name, got, tt.want)
		}
	}
	darks := []struct {
		in   model.Colour
		want model.Colour
	}{
		{model.NewColour(0, 0, 0, 255), model.NewColour(1, 1, 1, 255)},
		{model.NewColour(0, 0, 1, 255), model.NewColour(0, 0, 1, 255)},
	}
	for _, tt := range darks {
		if got := brightenDark(tt.in); got != tt.want {
			t.Errorf("brightenDark(%+v) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

// TestScaleRoundTripSweep walks the whole range of the virtual percentage,
// so the re-mapping and its inverse stay exact.
func TestScaleRoundTripSweep(t *testing.T) {
	for sz := 0; sz <= 400; sz++ {
		if got := ScaleToYTT(ScaleFromYTT(sz)); got != sz {
			t.Fatalf("round trip of sz %d = %d", sz, got)
		}
	}
	if got := ScaleToYTT(0.5); got != 0 {
		t.Errorf("a scale below the minimum = %d, want 0", got)
	}
}
