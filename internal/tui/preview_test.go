// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

func TestHex(t *testing.T) {
	if got := Hex(model.NewColour(255, 128, 0, 255)); got != "#ff8000" {
		t.Fatalf("Hex = %q, want #ff8000", got)
	}
}

func TestSwatch(t *testing.T) {
	plain := Swatch(model.NewColour(1, 2, 3, 255), false)
	if plain != "[#010203]" {
		t.Fatalf("a plain swatch = %q, want the hex value", plain)
	}
	colour := Swatch(model.NewColour(1, 2, 3, 255), true)
	if !strings.Contains(colour, "\x1b[48;2;1;2;3m") || !strings.Contains(colour, reset) {
		t.Fatalf("a colour swatch = %q, want a background and a reset", colour)
	}
}

func TestTint(t *testing.T) {
	if got := Tint(model.NewColour(1, 2, 3, 255), "text", false); got != "text" {
		t.Fatalf("a plain tint = %q, want the text", got)
	}
	got := Tint(model.NewColour(1, 2, 3, 255), "text", true)
	if !strings.Contains(got, "\x1b[38;2;1;2;3m") || !strings.HasSuffix(got, reset) {
		t.Fatalf("a colour tint = %q, want a foreground and a reset", got)
	}
}

func TestClock(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{0, "0:00:00.000"},
		{time.Second + 500*time.Millisecond, "0:00:01.500"},
		{time.Hour + 2*time.Minute + 3*time.Second, "1:02:03.000"},
		{-time.Second, "0:00:00.000"},
	}
	for _, tt := range tests {
		if got := Clock(tt.in); got != tt.want {
			t.Errorf("Clock(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestStyleRows covers every part of a style row: the name, the font, the
// size, the flags, and the box or the outline swatch.
func TestStyleRows(t *testing.T) {
	styles := []model.Style{
		{
			Name: "Default", Font: "Arial", Size: 20,
			Primary: model.NewColour(255, 255, 255, 255),
			Bold:    true, Italic: true, Underline: true,
		},
		{
			Name: "Boxed", Box: true,
			Primary: model.NewColour(0, 0, 0, 255),
			Outline: model.NewColour(0, 0, 255, 255),
		},
		{
			Name: "Outlined", OutlineWidth: 2,
			Primary: model.NewColour(9, 9, 9, 255),
			Outline: model.NewColour(1, 1, 1, 255),
		},
		// A style with no name, no font, and no size still renders.
		{Primary: model.NewColour(4, 5, 6, 255)},
	}
	rows := StyleRows(styles, false)
	if len(rows) != len(styles) {
		t.Fatalf("rows = %d, want %d", len(rows), len(styles))
	}
	joined := strings.Join(rows, "\n")
	for _, want := range []string{"Default", "Arial", "20", "bold", "italic", "underline", "box", "outline"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the preview is missing %q:\n%s", want, joined)
		}
	}
	if !strings.Contains(rows[0], "[#ffffff]") {
		t.Errorf("a plain preview must show the hex colour: %q", rows[0])
	}
	// The colour form carries an ANSI background instead of the hex value.
	coloured := strings.Join(StyleRows(styles, true), "\n")
	if !strings.Contains(coloured, "\x1b[48;2;255;255;255m") {
		t.Errorf("the colour preview must carry a background:\n%q", coloured)
	}
}

func TestTimelineRows(t *testing.T) {
	doc := &model.Document{
		Cues: []model.Cue{
			{Start: 0, End: 4 * time.Second, Spans: []model.TextSpan{{Text: "plain"}}},
			{
				Start: time.Second, End: 5 * time.Second,
				Spans: []model.TextSpan{
					{Text: "ka", Start: 0, End: time.Second},
					{Text: "ra", Start: time.Second, End: 2 * time.Second},
				},
			},
		},
	}
	rows, dropped := TimelineRows(doc, 8, 0, false)
	if dropped != 0 {
		t.Fatalf("dropped = %d, want 0", dropped)
	}
	// The plain cue takes one row and the karaoke cue takes a text row and a
	// bar row.
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3:\n%s", len(rows), strings.Join(rows, "\n"))
	}
	if !strings.Contains(rows[0], "0:00:00.000-0:00:04.000") || !strings.Contains(rows[0], "plain") {
		t.Errorf("the first row = %q", rows[0])
	}
	if !strings.Contains(rows[2], "#") {
		t.Errorf("the karaoke bar = %q, want a sung cell", rows[2])
	}
}

func TestTimelineRowsLimit(t *testing.T) {
	doc := &model.Document{
		Cues: []model.Cue{
			{Start: 0, End: time.Second, Spans: []model.TextSpan{{Text: "one"}}},
			{Start: time.Second, End: 2 * time.Second, Spans: []model.TextSpan{{Text: "two"}}},
			{Start: 2 * time.Second, End: 3 * time.Second, Spans: []model.TextSpan{{Text: "three"}}},
		},
	}
	rows, dropped := TimelineRows(doc, 8, 1, false)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if dropped != 2 {
		t.Fatalf("dropped = %d, want 2", dropped)
	}
	if strings.Contains(strings.Join(rows, "\n"), "two") {
		t.Fatalf("the limit must leave the later cues out:\n%s", strings.Join(rows, "\n"))
	}
}

// TestTimelineRowsColour covers the coloured bar, whose runs change the
// foreground colour as the karaoke windows change.
func TestTimelineRowsColour(t *testing.T) {
	four := 4 * time.Second
	doc := &model.Document{
		Cues: []model.Cue{{
			Start: 0, End: four,
			Spans: []model.TextSpan{
				{Text: "a", Start: 0, End: time.Second, Fore: &model.Colour{R: 255, A: 255}},
				{Text: "b", Start: 3 * time.Second, End: four, Secondary: &model.Colour{R: 0, G: 128, B: 0, A: 255}},
			},
		}},
	}
	rows, _ := TimelineRows(doc, 8, 0, true)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want a text row and a bar row", len(rows))
	}
	bar := rows[1]
	if !strings.Contains(bar, "\x1b[38;2;255;0;0m") {
		t.Errorf("the sung run is missing its colour: %q", bar)
	}
	if !strings.Contains(bar, "\x1b[38;2;0;128;0m") {
		t.Errorf("the unsung run is missing its colour: %q", bar)
	}
	if !strings.HasSuffix(bar, reset) {
		t.Errorf("the bar must end with a reset: %q", bar)
	}
}

func TestKaraokeBar(t *testing.T) {
	tests := []struct {
		name  string
		cue   model.Cue
		width int
		want  string
	}{
		{
			name:  "untimed",
			cue:   model.Cue{Start: 0, End: 4 * time.Second, Spans: []model.TextSpan{{Text: "x"}}},
			width: 8,
			want:  "........",
		},
		{
			name: "full",
			cue: model.Cue{Start: 0, End: 4 * time.Second, Spans: []model.TextSpan{
				{Text: "x", Start: 0, End: 4 * time.Second},
			}},
			width: 8,
			want:  "########",
		},
		{
			name: "gap",
			cue: model.Cue{Start: 0, End: 4 * time.Second, Spans: []model.TextSpan{
				{Text: "a", Start: 0, End: time.Second},
				{Text: "b", Start: 3 * time.Second, End: 4 * time.Second},
			}},
			width: 8,
			want:  "##....##",
		},
		{
			name: "short span",
			cue: model.Cue{Start: 0, End: 100 * time.Second, Spans: []model.TextSpan{
				{Text: "a", Start: 0, End: time.Millisecond},
			}},
			width: 8,
			want:  "#.......",
		},
		{
			name: "beyond the end",
			cue: model.Cue{Start: 0, End: time.Second, Spans: []model.TextSpan{
				{Text: "a", Start: 0, End: 10 * time.Second},
			}},
			width: 8,
			want:  "########",
		},
		{
			name:  "no width",
			cue:   model.Cue{Start: 0, End: time.Second, Spans: []model.TextSpan{{Text: "x", Start: 0, End: time.Second}}},
			width: 0,
			want:  "",
		},
		{
			name:  "no duration",
			cue:   model.Cue{Spans: []model.TextSpan{{Text: "x", Start: 0, End: time.Second}}},
			width: 4,
			want:  "....",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KaraokeBar(tt.cue, tt.width); got != tt.want {
				t.Errorf("KaraokeBar = %q, want %q", got, tt.want)
			}
		})
	}
}
