package model

import "time"

// Colour is an RGBA colour. Every channel, including alpha, carries the
// full 0-255 range. Formats without alpha write a loss note when they drop
// the channel.
type Colour struct {
	R, G, B, A uint8
}

// NewColour returns a colour with the given channels.
func NewColour(r, g, b, a uint8) Colour {
	return Colour{R: r, G: g, B: b, A: a}
}

// Opaque reports whether the colour is fully opaque.
func (c Colour) Opaque() bool {
	return c.A == 255
}

// Transparent reports whether the colour is fully transparent.
func (c Colour) Transparent() bool {
	return c.A == 0
}

// WithAlpha returns the colour with the alpha channel set to a.
func (c Colour) WithAlpha(a uint8) Colour {
	c.A = a
	return c
}

// Point is a position in video pixels, with the origin at the top-left
// corner of the video.
type Point struct {
	X, Y float64
}

// Anchor is a numpad screen position: 1 is bottom-left, 5 is centre, and 9
// is top-right. The numbering matches ASS alignment values. YTT anchor
// point values (0 is top-left, 8 is bottom-right) need the mapping in the
// YTT reader.
type Anchor uint8

// The numpad anchors.
const (
	AnchorBottomLeft Anchor = iota + 1
	AnchorBottomCentre
	AnchorBottomRight
	AnchorMiddleLeft
	AnchorCentre
	AnchorMiddleRight
	AnchorTopLeft
	AnchorTopCentre
	AnchorTopRight
)

// Fade describes a fade of the whole cue. Start and End are the fade
// durations; for a complex fade the T pairs give the intermediate times
// and opacities (ASS \fad and \fade).
type Fade struct {
	// In and Out are the fade-in and fade-out durations.
	In, Out time.Duration

	// Complex fade fields. StartIn/EndIn bound the fade-in on the cue
	// timeline; StartOut/EndOut bound the fade-out.
	StartIn, EndIn                 time.Duration
	StartOut, EndOut               time.Duration
	StartAlpha, MidAlpha, EndAlpha uint8
}

// Move describes motion of the whole cue from one point to another
// (ASS \move).
type Move struct {
	From, To Point
	// Start and End bound the motion on the cue timeline. Both zero means
	// "across the whole cue".
	Start, End time.Duration
}

// Animation is one cue-level effect. Exactly one field is non-nil.
type Animation struct {
	Fade *Fade
	Move *Move
	// Shake randomly displaces the cue within a radius (ASS \ytshake).
	Shake *Shake
	// Chroma is the chromatic aberration effect (ASS \ytchroma).
	Chroma *Chroma
	// Keyframes animates style values over time (ASS \t).
	Keyframes []Keyframe
	// Karaoke is an advanced karaoke type (ASS \ytkt).
	Karaoke *Karaoke
}

// Shake is the random displacement effect (ASS \ytshake).
type Shake struct {
	RadiusX, RadiusY float64
	// Start and End bound the effect on the cue timeline. Both zero means
	// "across the whole cue".
	Start, End time.Duration
}

// Chroma is the chromatic aberration effect (ASS \ytchroma): copies of the
// cue start apart and merge in, then split apart on the way out.
type Chroma struct {
	// Offsets holds one offset per colour copy, from the cue position.
	Offsets []Point
	// InTime and OutTime are the merge and disperse durations.
	InTime, OutTime time.Duration
	// Colours holds one colour per copy for a custom effect. An empty
	// slice means the default red, green, and blue copies.
	Colours []Colour
	// Alpha is the alpha channel of a custom effect, from 0 to 255.
	Alpha uint8
}

// KaraokeKind enumerates the advanced karaoke types of ASS (\ytkt).
type KaraokeKind uint8

// The karaoke kinds.
const (
	// KaraokeFade fades each syllable in (\ytktFade).
	KaraokeFade KaraokeKind = iota + 1
	// KaraokeGlitch replaces the letters of each syllable with random
	// glyphs (\ytktGlitch).
	KaraokeGlitch
	// KaraokeCursor draws extra text after the current syllable
	// (\ytkt(Cursor,...)).
	KaraokeCursor
)

// Karaoke is an advanced karaoke presentation (ASS \ytkt).
type Karaoke struct {
	Kind KaraokeKind
	// Cursor is the text that follows the sung syllable for the cursor
	// kind.
	Cursor string
	// CursorTags holds the override tags that style the cursor text.
	CursorTags string
	// CursorLeft draws the cursor before the sung syllable (the LCursor
	// and RCursor forms). The default cursor follows the syllable.
	CursorLeft bool
	// CursorInterval is the duration of each frame of an animated cursor.
	CursorInterval time.Duration
	// CursorFrames holds the frames of an animated cursor in order. An
	// empty slice means a static cursor.
	CursorFrames []KaraokeFrame
}

// KaraokeFrame is one frame of an animated \ytkt cursor.
type KaraokeFrame struct {
	// Tags styles the frame text with override tags.
	Tags string
	// Text is the frame text.
	Text string
}

// Keyframe is one animated value change (ASS \t).
type Keyframe struct {
	// Start and End bound the change on the cue timeline.
	Start, End time.Duration
	// Easing is the acceleration of the change; 1 is linear.
	Easing float64
	// Steps holds the values to animate.
	Steps []KeyframeStep
}

// KeyframeStep is one animated value (ASS \t arguments).
type KeyframeStep struct {
	// Property names the animated value: "fore", "outline", "shadow",
	// "fontsize", or a transparency property such as "forealpha".
	Property string
	// Value is the target value of the property.
	Value string
}
