// SPDX-License-Identifier: Apache-2.0

package ytt

import "github.com/bladeacer/swag/internal/model"

// The ruby values of the "rb" pen attribute. The base value marks the text
// that carries the reading; the parenthesis value marks the fallback text
// for clients without ruby support; the over and under values mark the
// reading itself.
const (
	rubyNone        = 0
	rubyBase        = 1
	rubyParenthesis = 2
	rubyOver        = 4
	rubyOverAlt     = 6
	rubyUnder       = 5
	rubyUnderAlt    = 7
)

// The script values of the "of" pen attribute.
const (
	scriptSubscript   = 0
	scriptRegular     = 1
	scriptSuperscript = 2
	// scriptUnset means the pen carries no offset attribute.
	scriptUnset = -1
)

// pen is the style of one text run. The reader builds it from a <pen>
// element and applies it to a span. The writer builds it from a resolved
// span and deduplicates it into a <pen> element. A nil colour means the
// document does not set it.
type pen struct {
	bold      bool
	italic    bool
	underline bool

	fore *model.Colour
	back *model.Colour
	edge *model.Colour
	// edgeKind is the shadow kind of the edge. Zero means no edge.
	edgeKind model.ShadowKind

	fontStyle FontStyle
	hasFont   bool
	fontScale int
	hasScale  bool

	ruby   int
	script int
	packed bool
}

// applyTo writes the pen attributes onto span as overrides. baseSize is the
// font size of the document style, which the scale value is relative to.
// The parenthesis ruby value marks fallback text, so the caller drops
// those spans before it reaches here.
func (p pen) applyTo(span *model.TextSpan, baseSize float64) {
	if p.bold {
		v := true
		span.Bold = &v
	}
	if p.italic {
		v := true
		span.Italic = &v
	}
	if p.underline {
		v := true
		span.Underline = &v
	}
	if p.fore != nil {
		c := *p.fore
		span.Fore = &c
	}
	if p.back != nil {
		c := *p.back
		span.Back = &c
	}
	if p.edge != nil && p.edgeKind != 0 {
		span.Shadows = append(span.Shadows, model.Shadow{Kind: p.edgeKind, Colour: *p.edge})
	}
	if p.hasFont {
		name := FontName(p.fontStyle)
		span.Font = &name
	}
	if p.hasScale {
		size := baseSize * ScaleFromYTT(p.fontScale)
		span.Size = &size
	}
	switch p.ruby {
	case rubyOver, rubyOverAlt:
		pos := model.RubyOver
		span.Ruby = &model.Ruby{Position: pos}
	case rubyUnder, rubyUnderAlt:
		pos := model.RubyUnder
		span.Ruby = &model.Ruby{Position: pos}
	case rubyBase, rubyNone, rubyParenthesis:
		// A base span carries no Ruby field; the parenthesis value is
		// handled by the reader before this call.
	}
	switch p.script {
	case scriptSubscript:
		kind := model.ScriptSubscript
		span.Script = &model.Script{Kind: kind}
	case scriptSuperscript:
		kind := model.ScriptSuperscript
		span.Script = &model.Script{Kind: kind}
	case scriptRegular, scriptUnset:
		// The regular offset is the default, so no override is needed.
	}
	if p.packed {
		v := true
		span.Packed = &v
	}
}

// isRubyParenthesis reports whether the pen marks fallback parenthesis
// text, which the reader drops so the IR keeps one reading per base.
func (p pen) isRubyParenthesis() bool {
	return p.ruby == rubyParenthesis
}
