package model

import (
	"testing"
	"time"
)

func TestResolvePrefersSpanOverrides(t *testing.T) {
	style := Style{
		Name:         "Default",
		Font:         "Roboto",
		Size:         20,
		Primary:      NewColour(255, 255, 255, 254),
		Secondary:    NewColour(120, 120, 120, 254),
		OutlineWidth: 2,
	}
	font := "Verdana"
	size := 40.0
	fore := NewColour(255, 0, 0, 254)
	span := TextSpan{Text: "big red", Font: &font, Size: &size, Fore: &fore}

	got := Resolve(style, span)
	if got.Font != "Verdana" || got.Size != 40 || got.Fore != fore {
		t.Fatalf("span overrides not applied: %+v", got)
	}
	if got.Secondary != style.Secondary {
		t.Fatalf("unset override must inherit from the style: %+v", got)
	}
}

func TestResolveInheritsStyleBooleans(t *testing.T) {
	style := Style{Name: "Base", Font: "Roboto", Size: 20, Bold: true, Italic: true}
	got := Resolve(style, TextSpan{Text: "plain"})
	if !got.Bold || !got.Italic {
		t.Fatalf("style booleans not inherited: %+v", got)
	}
	off := false
	got = Resolve(style, TextSpan{Text: "not italic", Italic: &off})
	if !got.Bold {
		t.Fatal("bold must stay on")
	}
	if got.Italic {
		t.Fatal("span must override italic off")
	}
}

// TestResolveSpanOverrides covers the span overrides that have no style
// equivalent, and the span shadow depth that replaces the style depth.
func TestResolveSpanOverrides(t *testing.T) {
	style := Style{Name: "Base", Size: 20, ShadowDepth: 2, Shadow: NewColour(1, 2, 3, 255)}
	strike := true
	scaleX, scaleY := 150.0, 50.0
	depth := 4.0
	span := TextSpan{Text: "x", Strikeout: &strike, ScaleX: &scaleX, ScaleY: &scaleY, ShadowDepth: &depth}
	got := Resolve(style, span)
	if !got.Strikeout || got.ScaleX != 150 || got.ScaleY != 50 || got.ShadowDepth != 4 {
		t.Fatalf("span override fields not applied: %+v", got)
	}
	if len(got.Shadows) != 1 || got.Shadows[0].Kind != ShadowHard {
		t.Fatalf("a positive span shadow depth needs a hard shadow: %+v", got.Shadows)
	}
}

// TestResolveDefaults covers a span with no overrides: the scale is 100%,
// and the shadow depth comes from the style.
func TestResolveDefaults(t *testing.T) {
	got := Resolve(Style{Name: "Base", Size: 20}, TextSpan{Text: "x"})
	if got.ScaleX != 100 || got.ScaleY != 100 || got.Strikeout || got.ShadowDepth != 0 {
		t.Fatalf("defaults wrong: %+v", got)
	}
}

// TestResolveSpanShadowDepthZero covers a span that clears the style shadow
// with a zero depth.
func TestResolveSpanShadowDepthZero(t *testing.T) {
	zero := 0.0
	got := Resolve(Style{Name: "Base", Size: 20, ShadowDepth: 2}, TextSpan{Text: "x", ShadowDepth: &zero})
	if len(got.Shadows) != 0 || got.ShadowDepth != 0 {
		t.Fatalf("a zero span depth must remove the shadow: %+v", got)
	}
}

func TestResolveShadowsFromStyle(t *testing.T) {
	shadow := NewColour(34, 34, 34, 254)
	style := Style{Name: "Shadowed", Font: "Roboto", Size: 20, Shadow: shadow, ShadowDepth: 2}
	got := Resolve(style, TextSpan{Text: "x"})
	if len(got.Shadows) != 1 || got.Shadows[0].Kind != ShadowHard || got.Shadows[0].Colour != shadow {
		t.Fatalf("style shadow not carried into the resolved span: %+v", got)
	}
	custom := []Shadow{{Kind: ShadowGlow, Colour: shadow}}
	got = Resolve(style, TextSpan{Text: "x", Shadows: custom})
	if len(got.Shadows) != 1 || got.Shadows[0].Kind != ShadowGlow {
		t.Fatalf("span shadow must replace the style shadow: %+v", got)
	}
}

func TestResolveBoxFromStyle(t *testing.T) {
	box := NewColour(0, 0, 0, 255)
	style := Style{Name: "Boxed", Font: "Roboto", Size: 20, Outline: box, OutlineWidth: 2, Box: true}
	got := Resolve(style, TextSpan{Text: "x"})
	if !got.Box || got.BoxColour != box {
		t.Fatalf("box style not carried into the resolved span: %+v", got)
	}
	style.Box = false
	if got := Resolve(style, TextSpan{Text: "x"}); got.Box {
		t.Fatal("box must stay off when the style has no box")
	}
}

func TestCueKaraokeDetection(t *testing.T) {
	untimed := Cue{Spans: []TextSpan{{Text: "plain"}}}
	if untimed.Karaoke() {
		t.Fatal("cue without offsets must not be karaoke")
	}
	timed := Cue{Spans: []TextSpan{
		{Text: "ta ", Start: 0, End: 500 * time.Millisecond},
		{Text: "ke", Start: 500 * time.Millisecond, End: time.Second},
	}}
	if !timed.Karaoke() {
		t.Fatal("cue with offsets must be karaoke")
	}
}

func TestCueKaraokeDetectionEndOnly(t *testing.T) {
	cue := Cue{Spans: []TextSpan{{Text: "x", End: time.Second}}}
	if !cue.Karaoke() {
		t.Fatal("a non-zero end offset must make the cue karaoke")
	}
}

func TestCueText(t *testing.T) {
	cue := Cue{Spans: []TextSpan{{Text: "foo"}, {Text: "bar"}, {Text: "!"}}}
	if got := cue.Text(); got != "foobar!" {
		t.Fatalf("Text() = %q, want %q", got, "foobar!")
	}
}

func TestRubyGroups(t *testing.T) {
	spans := []TextSpan{
		{Text: "漢"},
		{Text: "かん", Ruby: &Ruby{Position: RubyOver}},
		{Text: " is "},
		{Text: "読"},
		{Text: "よ", Ruby: &Ruby{Position: RubyOver}},
		{Text: "み", Ruby: &Ruby{Position: RubyOver}},
	}
	groups := RubyGroups(spans)
	if len(groups) != 3 {
		t.Fatalf("got %d groups, want 3", len(groups))
	}
	if groups[0].Base.Text != "漢" || len(groups[0].Annotations) != 1 {
		t.Fatalf("first group wrong: %+v", groups[0])
	}
	if groups[1].Base.Text != " is " || len(groups[1].Annotations) != 0 {
		t.Fatalf("plain span must pass through: %+v", groups[1])
	}
	if groups[2].Base.Text != "読" || len(groups[2].Annotations) != 2 {
		t.Fatalf("second base must carry two annotations: %+v", groups[2])
	}
}

func TestRubyGroupsLeadingAnnotation(t *testing.T) {
	// A leading annotation has no base before it; the group keeps the
	// annotation as its base so no text is lost.
	spans := []TextSpan{{Text: "orphan", Ruby: &Ruby{}}}
	groups := RubyGroups(spans)
	if len(groups) != 1 || groups[0].Base.Text != "orphan" {
		t.Fatalf("leading annotation mishandled: %+v", groups)
	}
}

func TestCueSungAt(t *testing.T) {
	cue := Cue{Spans: []TextSpan{
		{Text: "one ", Start: 0, End: 500 * time.Millisecond},
		{Text: "two ", Start: 500 * time.Millisecond, End: time.Second},
		{Text: "three", Start: time.Second, End: 1500 * time.Millisecond},
	}}
	if got := len(cue.SungAt(0)); got != 1 {
		t.Fatalf("at offset 0 got %d sung spans, want 1", got)
	}
	if got := len(cue.SungAt(600 * time.Millisecond)); got != 2 {
		t.Fatalf("at offset 600ms got %d sung spans, want 2", got)
	}
	if got := len(cue.SungAt(2 * time.Second)); got != 3 {
		t.Fatalf("at offset 2s got %d sung spans, want 3", got)
	}
}

func TestCueSungAtUntimed(t *testing.T) {
	cue := Cue{Spans: []TextSpan{{Text: "a"}, {Text: "b"}}}
	got := cue.SungAt(0)
	if len(got) != 2 {
		t.Fatalf("untimed cue must sing every span: got %d", len(got))
	}
}

func TestCueDuration(t *testing.T) {
	cue := Cue{Start: 10 * time.Second, End: 13 * time.Second}
	if got := cue.Duration(); got != 3*time.Second {
		t.Fatalf("Duration() = %v, want 3s", got)
	}
}
