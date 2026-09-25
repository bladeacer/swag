// SPDX-License-Identifier: Apache-2.0

package ytt

import "github.com/bladeacer/swag/internal/model"

// buildPen turns a resolved span into a pen, applying the colour and font
// quirks of the YouTube upload path. shadow carries the shadow of the
// current layer, or nil when the span has no shadow at all.
func buildPen(r model.Resolved, span model.TextSpan, base model.Style, shadow *model.Shadow, losses *lossNotes) pen {
	p := pen{script: scriptUnset}

	p.bold = r.Bold
	p.italic = r.Italic
	p.underline = r.Underline

	// Platform quirks: hold alpha at or below 254, shift pure white, and
	// lift pure black, so the colour survives the upload.
	fore := clampOpacity(shiftWhite(brightenDark(r.Fore)))
	p.fore = &fore

	switch {
	case r.Box:
		back := clampOpacity(r.BoxColour)
		p.back = &back
	case span.Back != nil:
		back := clampOpacity(*span.Back)
		p.back = &back
	}

	switch {
	case shadow != nil:
		edge := clampOpacity(shadow.Colour)
		p.edge = &edge
		p.edgeKind = shadow.Kind
	case len(r.Shadows) == 0 && r.OutlineWidth > 0 && !r.Box:
		// A plain outline becomes the glow edge type, which the YouTube
		// player draws around the glyphs.
		edge := clampOpacity(base.Outline)
		p.edge = &edge
		p.edgeKind = model.ShadowGlow
	}

	fontStyle, known := LookupFont(r.Font)
	p.fontStyle, p.hasFont = fontStyle, true
	if !known {
		losses.add("font %q is not a YouTube font; the write uses %s", r.Font, Roboto)
	}

	if base.Size > 0 && r.Size > 0 && r.Size != base.Size {
		scale := r.Size / base.Size
		p.fontScale, p.hasScale = ScaleToYTT(scale), true
		if scale < 0.75 {
			losses.add("font size is the 75%% minimum after the YouTube scale remap")
		}
	}

	if span.Script != nil {
		switch span.Script.Kind {
		case model.ScriptSubscript:
			p.script = scriptSubscript
		case model.ScriptSuperscript:
			p.script = scriptSuperscript
		case model.ScriptRegular:
			p.script = scriptRegular
		}
	}
	if span.Packed != nil && *span.Packed {
		p.packed = true
	}
	return p
}
