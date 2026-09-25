// SPDX-License-Identifier: Apache-2.0

package ytt

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

// lossNotes collects the degradations of one write.
type lossNotes struct {
	notes []string
}

func (l *lossNotes) add(format string, args ...any) {
	l.notes = append(l.notes, fmt.Sprintf(format, args...))
}

// Writer emits YouTube Timed Text documents from the IR.
type Writer struct{}

// NewWriter returns a YouTube Timed Text writer.
func NewWriter() *Writer { return &Writer{} }

// Name returns the registry name of the format.
func (w *Writer) Name() string { return FormatName }

// renderedSegment is one <s> run of a line.
type renderedSegment struct {
	penID int
	text  string
	// time is the karaoke offset in milliseconds. A negative value means
	// the run carries no timing.
	time int
}

// renderedLine is one <p> element, ready to write.
type renderedLine struct {
	start    int
	duration int
	wpID     int
	wsID     int
	// penID is the pen of the whole line when the line has one run. It is
	// -1 when the line uses <s> runs instead.
	penID int
	text  string
	runs  []renderedSegment
}

// Render writes doc to sink as YouTube Timed Text. The returned slice
// carries one entry per feature the format cannot express.
func (w *Writer) Render(doc *model.Document, sink io.Writer) ([]string, error) {
	var losses lossNotes

	dims := doc.VideoDimensions
	if dims.X <= 0 || dims.Y <= 0 {
		dims = model.Point{X: defaultWidth, Y: defaultHeight}
	}
	base := DefaultStyle()
	if len(doc.Styles) > 0 {
		base = doc.Styles[0]
	}

	pens := newPenTable()
	windows := newWindowTable()
	positions := newPositionTable(dims)

	var lines []renderedLine
	for i := range doc.Cues {
		lines = append(lines, w.renderCue(doc.Cues[i], base, pens, windows, positions, &losses)...)
	}

	var out strings.Builder
	out.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\" ?>\n")
	out.WriteString("<timedtext format=\"3\">\n<head>\n")
	writePens(&out, pens)
	writeWindows(&out, windows)
	writePositions(&out, positions)
	out.WriteString("</head>\n<body>\n")
	for _, line := range lines {
		writeLine(&out, line)
	}
	out.WriteString("</body>\n</timedtext>\n")

	if _, err := io.WriteString(sink, out.String()); err != nil {
		return losses.notes, fmt.Errorf("write ytt: %w", err)
	}
	return losses.notes, nil
}

// renderCue turns one cue into one or more lines. A cue with more than one
// shadow kind is layered: YTT carries one edge per pen, so the writer emits
// one copy of the line per kind.
func (w *Writer) renderCue(
	cue model.Cue,
	base model.Style,
	pens *penTable,
	windows *windowTable,
	positions *positionTable,
	losses *lossNotes,
) []renderedLine {
	recordCueLosses(cue, losses)

	spans := model.NormaliseKaraoke(cue.Spans)
	if len(spans) == 0 {
		return nil
	}

	wpID := positions.id(cue.Layout)
	wsID := windows.id(windowFor(spans, layoutAnchor(cue.Layout)))

	// The YouTube player clips a leading italic overhang and a leading
	// shadow. Steal one space from the line start so neither is cut off.
	needSpace := false
	if r := model.Resolve(base, spans[0]); r.Italic || len(r.Shadows) > 0 {
		needSpace = true
	}

	kinds := shadowKinds(spans, base)
	layers := len(kinds)
	if layers == 0 {
		layers = 1
	}

	startMS := int(cue.Start / time.Millisecond)
	durationMS := int((cue.End - cue.Start) / time.Millisecond)

	lines := make([]renderedLine, 0, layers)
	for layer := 0; layer < layers; layer++ {
		kind, hasKind := activeKind(kinds, layer)
		runs := w.buildRuns(spans, base, kind, hasKind, pens, losses)
		line := renderedLine{
			start:    startMS,
			duration: durationMS,
			wpID:     wpID,
			wsID:     wsID,
			penID:    -1,
		}
		if len(runs) == 1 && runs[0].time < 0 {
			line.penID = runs[0].penID
			line.text = runs[0].text
		} else {
			line.runs = runs
		}
		if needSpace {
			prependSpace(&line)
		}
		lines = append(lines, line)
	}
	return lines
}

// buildRuns turns the spans of a cue into <s> runs, expanding ruby groups
// into the four-run sequence YouTube expects and choosing the shadow of the
// current layer.
func (w *Writer) buildRuns(
	spans []model.TextSpan,
	base model.Style,
	kind model.ShadowKind,
	hasKind bool,
	pens *penTable,
	losses *lossNotes,
) []renderedSegment {
	var runs []renderedSegment

	for _, group := range model.RubyGroups(spans) {
		r := model.Resolve(base, group.Base)
		shadow := chosenShadow(r.Shadows, kind, hasKind)
		basePen := buildPen(r, group.Base, base, shadow, losses)

		if len(group.Annotations) == 0 {
			runs = append(runs, renderedSegment{
				penID: pens.id(basePen),
				text:  group.Base.Text,
				time:  runTime(group.Base),
			})
			continue
		}

		// A base with readings needs the base pen, a parenthesis pen, and
		// the reading pens. The reading always uses the formatting of the
		// base, which is what the YouTube player does.
		baseMark := basePen
		baseMark.ruby = rubyBase
		runs = append(runs, renderedSegment{penID: pens.id(baseMark), text: group.Base.Text, time: runTime(group.Base)})

		for _, ann := range group.Annotations {
			if ann.Ruby.Position == model.RubyParenthetical {
				runs = append(runs, renderedSegment{penID: pens.id(basePen), text: "(" + ann.Text + ")", time: runTime(ann)})
				continue
			}
			paren := basePen
			paren.ruby = rubyParenthesis
			reading := basePen
			if ann.Ruby.Position == model.RubyUnder {
				reading.ruby = rubyUnder
			} else {
				reading.ruby = rubyOver
			}
			runs = append(runs,
				renderedSegment{penID: pens.id(paren), text: "(", time: runTime(ann)},
				renderedSegment{penID: pens.id(reading), text: ann.Text, time: runTime(ann)},
				renderedSegment{penID: pens.id(paren), text: ")", time: runTime(ann)},
			)
		}
	}
	return runs
}

// runTime returns the relative karaoke offset of a span in milliseconds, or
// -1 when the span carries no timing.
func runTime(span model.TextSpan) int {
	if span.Start == 0 && span.End == 0 {
		return -1
	}
	return int(span.Start / time.Millisecond)
}

// prependSpace steals one space at the line start. YouTube trims the
// leading italic overhang and the leading shadow, so the space keeps them
// inside the drawn box.
func prependSpace(line *renderedLine) {
	if line.penID >= 0 {
		line.text = " " + line.text
		return
	}
	if len(line.runs) > 0 {
		line.runs[0].text = " " + line.runs[0].text
	}
}

// recordCueLosses notes the features of cue that YouTube Timed Text cannot
// express.
func recordCueLosses(cue model.Cue, losses *lossNotes) {
	for _, span := range cue.Spans {
		if span.Strikeout != nil && *span.Strikeout {
			losses.add("strikeout")
		}
		if scaleChanged(span.ScaleX) || scaleChanged(span.ScaleY) {
			losses.add("glyph scale")
		}
		if span.Voice != nil {
			losses.add("voice name")
		}
	}
	for _, a := range cue.Animations {
		switch {
		case a.Fade != nil:
			losses.add("fade animation")
		case a.Move != nil:
			losses.add("move animation")
		case a.Shake != nil:
			losses.add("shake animation")
		case a.Chroma != nil:
			losses.add("chroma animation")
		case a.Keyframes != nil:
			losses.add("keyframe animation")
		case a.Karaoke != nil:
			losses.add("karaoke type")
		}
	}
	if cue.Layout != nil {
		if cue.Layout.Fade != nil {
			losses.add("cue fade")
		}
		if cue.Layout.Move != nil {
			losses.add("cue move")
		}
	}
}

// scaleChanged reports whether a glyph scale override differs from the
// original size. YouTube Timed Text carries no scale, so the writer records
// a loss for it.
func scaleChanged(v *float64) bool { return v != nil && *v != 100 }

// layoutAnchor returns the cue-level anchor of a layout, if it has one.
func layoutAnchor(layout *model.Layout) *model.Anchor {
	if layout == nil {
		return nil
	}
	return layout.Anchor
}

// shadowKinds returns the distinct shadow kinds of the spans, in the fixed
// order of the IR.
func shadowKinds(spans []model.TextSpan, base model.Style) []model.ShadowKind {
	seen := map[model.ShadowKind]bool{}
	var kinds []model.ShadowKind
	for _, s := range spans {
		for _, sh := range model.Resolve(base, s).Shadows {
			if !seen[sh.Kind] {
				seen[sh.Kind] = true
				kinds = append(kinds, sh.Kind)
			}
		}
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
	return kinds
}

// activeKind returns the shadow kind of one layer.
func activeKind(kinds []model.ShadowKind, layer int) (model.ShadowKind, bool) {
	if len(kinds) == 0 {
		return 0, false
	}
	if len(kinds) == 1 {
		return kinds[0], true
	}
	return kinds[layer], true
}

// chosenShadow returns the shadow of the given kind, or nil when the span
// has none. When hasKind is false the span carries no shadow at all, so the
// caller falls back to the outline.
func chosenShadow(shadows []model.Shadow, kind model.ShadowKind, hasKind bool) *model.Shadow {
	if !hasKind {
		return nil
	}
	for i := range shadows {
		if shadows[i].Kind == kind {
			return &shadows[i]
		}
	}
	return nil
}

// writePens writes the pen elements in id order.
func writePens(out *strings.Builder, pens *penTable) {
	for i, p := range pens.list {
		fmt.Fprintf(out, "<pen id=\"%d\"", i+1)
		writePenAttributes(out, p)
		out.WriteString(" />\n")
	}
}

// writePenAttributes writes the attributes of one pen, from the first to
// the last of the format.
func writePenAttributes(out *strings.Builder, p pen) {
	if p.bold {
		out.WriteString(` b="1"`)
	}
	if p.italic {
		out.WriteString(` i="1"`)
	}
	if p.underline {
		out.WriteString(` u="1"`)
	}
	writePenColour(out, "fc", "fo", p.fore)
	writePenColour(out, "bc", "bo", p.back)
	writePenColour(out, "ec", "eo", p.edge)
	if p.edgeKind != 0 {
		fmt.Fprintf(out, ` et="%d"`, edgeTypeValue(p.edgeKind))
	}
	if p.hasFont {
		fmt.Fprintf(out, ` fs="%d"`, int(p.fontStyle))
	}
	if p.hasScale {
		fmt.Fprintf(out, ` sz="%d"`, p.fontScale)
	}
	if p.ruby != rubyNone {
		fmt.Fprintf(out, ` rb="%d"`, p.ruby)
	}
	if p.script != scriptUnset {
		fmt.Fprintf(out, ` of="%d"`, p.script)
	}
	if p.packed {
		out.WriteString(` hg="1"`)
	}
}

// writePenColour writes a colour and its opacity when the pen sets it.
func writePenColour(out *strings.Builder, colourAttr, opacityAttr string, c *model.Colour) {
	if c == nil {
		return
	}
	fmt.Fprintf(out, ` %s="%s"`, colourAttr, formatHexColour(*c))
	fmt.Fprintf(out, ` %s="%d"`, opacityAttr, c.A)
}

// edgeTypeValue maps a shadow kind onto the "et" attribute.
func edgeTypeValue(kind model.ShadowKind) int {
	switch kind {
	case model.ShadowHard:
		return 1
	case model.ShadowBevel:
		return 2
	case model.ShadowGlow:
		return 3
	case model.ShadowSoft:
		return 4
	}
	return 0
}

// writeLine writes one <p> element.
func writeLine(out *strings.Builder, line renderedLine) {
	fmt.Fprintf(out, `<p t="%d" d="%d"`, line.start, line.duration)
	if line.wpID >= 0 {
		fmt.Fprintf(out, ` wp="%d"`, line.wpID)
	}
	if line.wsID >= 0 {
		fmt.Fprintf(out, ` ws="%d"`, line.wsID)
	}
	if line.penID >= 0 {
		fmt.Fprintf(out, ` p="%d"`, line.penID)
		out.WriteString(">")
		out.WriteString(escapeText(line.text))
		out.WriteString("</p>\n")
		return
	}
	out.WriteString(">")
	for i, run := range line.runs {
		out.WriteString(`<s`)
		if run.penID >= 0 {
			fmt.Fprintf(out, ` p="%d"`, run.penID)
		}
		if run.time > 0 {
			fmt.Fprintf(out, ` t="%d"`, run.time)
		}
		out.WriteString(">")
		out.WriteString(escapeText(run.text))
		out.WriteString("</s>")
		if i == 0 && len(line.runs) > 1 {
			// YouTube drops the pen of the first run unless some text
			// sits outside a run. A zero-width space keeps the pen and
			// adds no visible glyph.
			out.WriteString("&#8203;")
		}
	}
	out.WriteString("</p>\n")
}

// escapeText escapes XML text without touching the line breaks, which the
// format uses as line separators. A bytes.Buffer cannot fail a write, so the
// escape error needs no branch.
func escapeText(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}
