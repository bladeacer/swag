// Package model defines the intermediate representation (IR) of swag.
//
// Every subtitle format reads into this representation and writes from it.
// The IR carries the union of the feature sets of the target formats: RGBA
// colours, span-level karaoke timing, ruby annotations, vertical layout,
// script offsets, and multi-shadow effects.
//
// Two conventions hold across the package. First, pointer and slice fields
// mean "the source set this"; nil or empty means "inherit from the style".
// Second, a Colour has no unset state, so an optional colour is a pointer.
//
// # Karaoke timing
//
// Karaoke lives on the spans of a Cue, not in a side table. A span that
// carries a non-zero Start or End is a karaoke segment: Start is when the
// span begins to render as sung, and End is when the next segment takes
// over. Both values are offsets from Cue.Start, so Cue.SungAt resolves the
// sung spans for any point in the cue without a clock. A cue is a karaoke
// cue when any span is timed (Cue.Karaoke); the unsung colour then comes
// from Style.Secondary, or from TextSpan.Secondary when the source set it.
//
// Writers must keep three invariants:
//
//   - Segments tile the cue in order: each segment starts at or after the
//     end of the one before it, and the first segment starts at zero.
//   - A gap between segments renders as unsung text, not as silence.
//   - Zero-duration segments are valid on read but carry no timing
//     information; formats that reject them (YTT upload) need the one-
//     millisecond bump before write.
//
// Ruby annotations ride the same slice. A base span (Ruby == nil) is
// followed immediately in Cue.Spans by one or more annotation spans
// (Ruby != nil); the group is the unit that converters move together.
// RubyGroups extracts groups, and it keeps a leading annotation as the
// base of its own group rather than dropping it, so no text is lost on a
// malformed line. Annotations carry no timing of their own: they inherit
// the window of their base, so re-timing a base re-times its readings.
// Renderers without ruby support must fall back to bracketed text in
// reading order, never to dropping the annotation.
package model

import "time"

// Document is a whole subtitle file in the intermediate representation.
type Document struct {
	// Metadata carries file-level key/value pairs, for example the title
	// from an ASS Script Info section.
	Metadata map[string]string

	// Styles holds the style definitions of the document, in source order.
	// A reader emits at least one style per document.
	Styles []Style

	// Cues holds the timed cues, in source order.
	Cues []Cue

	// VideoDimensions is the play resolution the coordinates refer to,
	// for example Point{X: 1280, Y: 720}.
	VideoDimensions Point
}

// Style carries the default rendering of a document: font, size, colours,
// widths, and alignment. Formats map their own style concepts onto these
// fields (for example, ASS BorderStyle 3 becomes Box true).
type Style struct {
	Name         string
	Font         string
	Size         float64
	Bold         bool
	Italic       bool
	Underline    bool
	Primary      Colour
	Secondary    Colour
	Outline      Colour
	Shadow       Colour
	OutlineWidth float64
	ShadowDepth  float64
	Alignment    Anchor
	// Box requests a background box instead of an outline (ASS
	// BorderStyle 3).
	Box bool
}

// Cue is one timed subtitle. Timing runs from the media start.
type Cue struct {
	Start time.Duration
	End   time.Duration

	// Spans holds the text of the cue, in reading order. Ruby base spans
	// are followed immediately by their annotation spans.
	Spans []TextSpan

	// Layout carries cue-level placement: position, anchor, and motion.
	// nil means "use the format default".
	Layout *Layout

	// Animations carries cue-level effects such as fades and shakes.
	Animations []Animation
}

// Layout carries cue-level placement.
type Layout struct {
	// Position is the anchor point location in video pixels.
	Position *Point
	// Anchor is the numpad anchor of the cue. nil means "use the style".
	Anchor *Anchor
	// Fade describes a simple or complex fade of the whole cue.
	Fade *Fade
	// Move describes motion from one point to another.
	Move *Move
}

// Shadow is one visual effect drawn behind the text.
type Shadow struct {
	Kind   ShadowKind
	Colour Colour
}

// ShadowKind enumerates the visual effects. A span carries at most one
// shadow per kind.
type ShadowKind uint8

// The shadow kinds. YTT edge types and ASS outline/shadow map onto these.
const (
	// ShadowSoft is a soft blurred shadow (YTT edge type 4).
	ShadowSoft ShadowKind = iota + 1
	// ShadowHard is a solid offset shadow (ASS \shadow, YTT edge type 1).
	ShadowHard
	// ShadowBevel is a bevel effect (YTT edge type 2).
	ShadowBevel
	// ShadowGlow is a glow effect (YTT edge type 3).
	ShadowGlow
)

// TextSpan is one run of text with its own overrides. Fields that are
// pointers or slices are overrides; nil or empty inherits from the Style.
type TextSpan struct {
	Text string

	// Start and End are karaoke offsets from the cue start. Both zero
	// means the span is untimed.
	Start time.Duration
	End   time.Duration

	Font      *string
	Size      *float64
	Bold      *bool
	Italic    *bool
	Underline *bool
	// Strikeout requests a line through the text (ASS \s).
	Strikeout *bool
	// ScaleX and ScaleY scale the glyphs in percent (ASS \fscx and
	// \fscy). 100 is the original size.
	ScaleX *float64
	ScaleY *float64

	// Fore is the foreground (sung) colour, Secondary the unsung karaoke
	// colour, Back the background box colour.
	Fore      *Colour
	Secondary *Colour
	Back      *Colour

	// Shadows holds the visual effects, at most one per kind.
	Shadows []Shadow

	// OutlineWidth overrides the style outline width.
	OutlineWidth *float64
	// ShadowDepth overrides the style shadow distance (ASS \shad,
	// \xshad, and \yshad). Zero removes the shadow.
	ShadowDepth *float64

	Vertical  *Vertical
	Script    *Script
	Direction *Direction
	Packed    *bool

	// Ruby marks an annotation span: the reading for the base span that
	// precedes it. Base spans carry a nil Ruby.
	Ruby *Ruby
}

// Vertical describes vertical text layout. Only one Mode applies per span.
type Vertical struct {
	Mode VerticalMode
	// Packed packs text into the space of a full-width character
	// (YTT \ytpack, only meaningful in column modes).
	Packed bool
}

// VerticalMode enumerates the vertical layout modes.
type VerticalMode uint8

// The vertical layout modes, matching the YTT print and scroll directions
// and the ASS \ytvert argument.
const (
	// VerticalNone is ordinary horizontal text.
	VerticalNone VerticalMode = iota
	// VerticalColumnsRTL lays out columns right to left (\ytvert9).
	VerticalColumnsRTL
	// VerticalColumnsLTR lays out columns left to right (\ytvert7).
	VerticalColumnsLTR
	// VerticalRotated rotates the whole cue 90 degrees counter-clockwise
	// (\ytvert1).
	VerticalRotated
	// VerticalRotatedReversed rotates and reverses the line order
	// (\ytvert3).
	VerticalRotatedReversed
)

// Script is a baseline offset of the text.
type Script struct {
	Kind ScriptKind
}

// ScriptKind enumerates the baseline offsets.
type ScriptKind uint8

// The script kinds.
const (
	// ScriptRegular is ordinary text (ASS \ytsur).
	ScriptRegular ScriptKind = iota
	// ScriptSubscript lowers the text (ASS \ytsub, YTT offset 0).
	ScriptSubscript
	// ScriptSuperscript raises the text (ASS \ytsup, YTT offset 2).
	ScriptSuperscript
)

// Direction is the reading direction of the text.
type Direction uint8

// The reading directions.
const (
	// DirLeftToRight is the default direction.
	DirLeftToRight Direction = iota
	// DirRightToLeft marks right-to-left text (\ytdir4).
	DirRightToLeft
)

// Ruby describes a furigana annotation attached to the preceding base span.
type Ruby struct {
	// Position is where the annotation renders relative to its base.
	Position RubyPosition
}

// RubyPosition enumerates the ruby annotation positions.
type RubyPosition uint8

// The ruby positions.
const (
	// RubyOver renders the annotation above the base (default, \ytruby8).
	RubyOver RubyPosition = iota
	// RubyUnder renders the annotation below the base (\ytruby2).
	RubyUnder
	// RubyParenthetical renders the annotation in brackets after the base,
	// the mobile fallback (YTT ruby part 2).
	RubyParenthetical
)
