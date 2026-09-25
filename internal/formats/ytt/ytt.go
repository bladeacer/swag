// SPDX-License-Identifier: Apache-2.0

// Package ytt reads and writes YouTube Timed Text (YTT, format 3).
//
// YouTube Timed Text is XML. A <pen> element carries the style of a text
// run, a <ws> element carries the window of a line, and a <wp> element
// carries a named window position. The body holds <p> lines, each with
// <s> runs whose "t" attribute gives karaoke timing.
//
// The package reproduces the platform quirks of the YouTube upload path on
// write. Each quirk carries a test, and the comment beside the code names
// the behaviour it copies. The quirks are: the font allow-list snap, the
// font scale remap, the opacity ceiling at 254, the white shift, the dark
// text lift, the zero-width space padding between runs, the karaoke
// zero-duration bump, the italic prefetch space, the shadow space, and
// multi-shadow line layering.
//
// The SRV3 format shares this model. The package exposes its reader and
// writer so the srv3 package can reuse them.
package ytt

import "github.com/bladeacer/swag/internal/model"

// FormatName is the registry name of the format.
const FormatName = "ytt"

// The play resolution the coordinate percentages refer to. YouTube Timed
// Text does not store a resolution, so the reader and the writer both use
// the common 720p frame.
const (
	defaultWidth  = 1280
	defaultHeight = 720
)

// DefaultStyle returns the implicit style of a YouTube document. Roboto at
// size 20 matches the default pen of the YouTube player.
func DefaultStyle() model.Style {
	return model.Style{
		Name:      "Default",
		Font:      Roboto,
		Size:      20,
		Primary:   model.NewColour(255, 255, 255, 254),
		Secondary: model.NewColour(120, 120, 120, 254),
		Outline:   model.NewColour(0, 0, 0, 254),
		// A pen carries its own edge, so the implicit style has no outline.
		// An outline on the way in becomes the glow edge type on write.
		OutlineWidth: 0,
		Alignment:    model.AnchorBottomCentre,
	}
}
