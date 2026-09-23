package model

import "time"

// Resolve returns the effective rendering of span inside style: the span
// override where it is set, the style value where it is not. The span is
// not modified.
func Resolve(style Style, span TextSpan) Resolved {
	r := Resolved{
		Text:         span.Text,
		Font:         style.Font,
		Size:         style.Size,
		Bold:         style.Bold,
		Italic:       style.Italic,
		Underline:    style.Underline,
		Fore:         style.Primary,
		Secondary:    style.Secondary,
		OutlineWidth: style.OutlineWidth,
	}
	if span.Font != nil {
		r.Font = *span.Font
	}
	if span.Size != nil {
		r.Size = *span.Size
	}
	if span.Bold != nil {
		r.Bold = *span.Bold
	}
	if span.Italic != nil {
		r.Italic = *span.Italic
	}
	if span.Underline != nil {
		r.Underline = *span.Underline
	}
	if span.Fore != nil {
		r.Fore = *span.Fore
	}
	if span.Secondary != nil {
		r.Secondary = *span.Secondary
	}
	if span.OutlineWidth != nil {
		r.OutlineWidth = *span.OutlineWidth
	}
	r.Shadows = append([]Shadow(nil), span.Shadows...)
	if len(r.Shadows) == 0 && style.ShadowDepth > 0 {
		r.Shadows = []Shadow{{Kind: ShadowHard, Colour: style.Shadow}}
	}
	if len(r.Shadows) == 0 && style.OutlineWidth > 0 && style.Box {
		r.Box = true
		r.BoxColour = style.Outline
	}
	return r
}

// Resolved is the effective rendering of a span in a style.
type Resolved struct {
	Text         string
	Font         string
	Size         float64
	Bold         bool
	Italic       bool
	Underline    bool
	Fore         Colour
	Secondary    Colour
	OutlineWidth float64
	Shadows      []Shadow
	// Box reports whether the span renders with a background box instead of
	// an outline, and BoxColour carries the box colour.
	Box       bool
	BoxColour Colour
}

// Karaoke reports whether cue carries karaoke timing: any span with a
// non-zero start or end offset makes the cue a karaoke cue.
func (c Cue) Karaoke() bool {
	for _, s := range c.Spans {
		if s.Start != 0 || s.End != 0 {
			return true
		}
	}
	return false
}

// Text returns the concatenated text of all spans of the cue.
func (c Cue) Text() string {
	var b []byte
	for _, s := range c.Spans {
		b = append(b, s.Text...)
	}
	return string(b)
}

// RubyGroup is one base span with its annotation spans.
type RubyGroup struct {
	Base        TextSpan
	Annotations []TextSpan
}

// RubyGroups walks spans and pairs each base span with the annotation spans
// that follow it. Spans without ruby pass through as groups with no
// annotations. The walk does not separate a base from its annotations, so
// converters can treat each group as one unit.
func RubyGroups(spans []TextSpan) []RubyGroup {
	var groups []RubyGroup
	for i := 0; i < len(spans); {
		g := RubyGroup{Base: spans[i]}
		i++
		for i < len(spans) && spans[i].Ruby != nil {
			g.Annotations = append(g.Annotations, spans[i])
			i++
		}
		groups = append(groups, g)
	}
	return groups
}

// SungAt reports the spans of cue that are sung (their karaoke window has
// started) at offset from the cue start. For an untimed cue every span is
// sung from offset zero.
func (c Cue) SungAt(offset time.Duration) []TextSpan {
	if !c.Karaoke() {
		return c.Spans
	}
	var sung []TextSpan
	for _, s := range c.Spans {
		if offset >= s.Start {
			sung = append(sung, s)
		}
	}
	return sung
}

// Duration returns the duration of the cue.
func (c Cue) Duration() time.Duration {
	return c.End - c.Start
}
