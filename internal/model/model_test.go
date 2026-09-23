package model

import "testing"

func TestColourWithAlpha(t *testing.T) {
	c := NewColour(1, 2, 3, 255)
	got := c.WithAlpha(128)
	if got.A != 128 || got.R != 1 || got.G != 2 || got.B != 3 {
		t.Fatalf("WithAlpha changed the wrong channels: %+v", got)
	}
	if c.A != 255 {
		t.Fatalf("WithAlpha mutated the receiver: %+v", c)
	}
}

func TestColourOpaqueAndTransparent(t *testing.T) {
	tests := []struct {
		name       string
		colour     Colour
		wantOpaque bool
		wantClear  bool
	}{
		{"opaque white", NewColour(255, 255, 255, 255), true, false},
		{"half grey", NewColour(0, 0, 0, 128), false, false},
		{"clear black", NewColour(0, 0, 0, 0), false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.colour.Opaque(); got != tt.wantOpaque {
				t.Errorf("Opaque() = %v, want %v", got, tt.wantOpaque)
			}
			if got := tt.colour.Transparent(); got != tt.wantClear {
				t.Errorf("Transparent() = %v, want %v", got, tt.wantClear)
			}
		})
	}
}

func TestAnchorNumpadOrder(t *testing.T) {
	tests := []struct {
		anchor Anchor
		want   uint8
	}{
		{AnchorBottomLeft, 1},
		{AnchorBottomCentre, 2},
		{AnchorBottomRight, 3},
		{AnchorMiddleLeft, 4},
		{AnchorCentre, 5},
		{AnchorMiddleRight, 6},
		{AnchorTopLeft, 7},
		{AnchorTopCentre, 8},
		{AnchorTopRight, 9},
	}
	for _, tt := range tests {
		if got := uint8(tt.anchor); got != tt.want {
			t.Errorf("anchor %d = %d, want %d", tt.want, got, tt.want)
		}
	}
}

func TestDocumentHoldsStylesAndCues(t *testing.T) {
	doc := Document{
		VideoDimensions: Point{X: 1280, Y: 720},
		Styles: []Style{
			{Name: "Default", Font: "Roboto", Size: 20, Primary: NewColour(255, 255, 255, 254)},
		},
		Cues: []Cue{
			{Spans: []TextSpan{{Text: "hello"}}},
		},
	}
	if len(doc.Styles) != 1 || doc.Styles[0].Name != "Default" {
		t.Fatalf("styles not stored: %+v", doc.Styles)
	}
	if doc.Cues[0].Text() != "hello" {
		t.Fatalf("cue text = %q", doc.Cues[0].Text())
	}
	if doc.VideoDimensions.X != 1280 || doc.VideoDimensions.Y != 720 {
		t.Fatalf("dimensions not stored: %+v", doc.VideoDimensions)
	}
}
