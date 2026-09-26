// SPDX-License-Identifier: Apache-2.0

package ass

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

// lossNotes collects the degradations of one write.
type lossNotes struct{ notes []string }

func (l *lossNotes) add(format string, args ...any) {
	l.notes = append(l.notes, fmt.Sprintf(format, args...))
}

// Writer emits ASS documents from the IR.
type Writer struct{}

// NewWriter returns an ASS writer.
func NewWriter() *Writer { return &Writer{} }

// Name returns the registry name of the format.
func (w *Writer) Name() string { return FormatName }

// styleFormat is the field order the writer emits for the style table.
const styleFormat = "Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, " +
	"OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, " +
	"Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding"

// Render writes doc to sink as ASS. The returned slice carries one entry
// per feature the format cannot express.
func (w *Writer) Render(doc *model.Document, sink io.Writer) ([]string, error) {
	var losses lossNotes

	base := baseStyle(doc.Styles)
	styles := doc.Styles
	if len(styles) == 0 {
		styles = []model.Style{base}
	}

	var out strings.Builder
	out.Grow(len(doc.Cues) * 96)
	writeScriptInfo(&out, doc)
	writeStyles(&out, styles)
	out.WriteString("[Events]\n")
	out.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")
	var groups []model.RubyGroup
	for i := range doc.Cues {
		groups = model.RubyGroupsInto(groups[:0], doc.Cues[i].Spans)
		text := w.renderCue(doc.Cues[i], base, &losses, groups)
		fmt.Fprintf(&out, "Dialogue: 0,%s,%s,%s,,0,0,0,,%s\n",
			formatASSTime(doc.Cues[i].Start), formatASSTime(doc.Cues[i].End), base.Name, text)
	}

	if _, err := io.WriteString(sink, out.String()); err != nil {
		return losses.notes, fmt.Errorf("write ass: %w", err)
	}
	return losses.notes, nil
}

// writeScriptInfo writes the Script Info section from the document metadata
// and the play resolution.
func writeScriptInfo(out *strings.Builder, doc *model.Document) {
	meta := func(key, def string) string {
		if doc.Metadata != nil && doc.Metadata[key] != "" {
			return doc.Metadata[key]
		}
		return def
	}
	out.WriteString("[Script Info]\n")
	fmt.Fprintf(out, "Title: %s\n", meta("Title", "swag"))
	fmt.Fprintf(out, "ScriptType: %s\n", meta("ScriptType", "v4.00+"))
	fmt.Fprintf(out, "WrapStyle: %s\n", meta("WrapStyle", "0"))
	fmt.Fprintf(out, "ScaledBorderAndShadow: yes\n")
	width, height := doc.VideoDimensions.X, doc.VideoDimensions.Y
	if width <= 0 {
		width = defaultWidth
	}
	if height <= 0 {
		height = defaultHeight
	}
	fmt.Fprintf(out, "PlayResX: %s\n", num(width))
	fmt.Fprintf(out, "PlayResY: %s\n", num(height))
	fmt.Fprintf(out, "Collisions: %s\n", meta("Collisions", "Normal"))
	out.WriteString("\n")
}

// writeStyles writes the V4+ Styles section.
func writeStyles(out *strings.Builder, styles []model.Style) {
	out.WriteString("[V4+ Styles]\n")
	out.WriteString(styleFormat)
	out.WriteString("\n")
	for _, s := range styles {
		border := 1
		if s.Box {
			border = 3
		}
		fmt.Fprintf(out, "Style: %s,%s,%s,%s,%s,%s,%s,%s,%s,%s,0,100,100,0,0,%d,%s,%s,%d,10,10,10,1\n",
			styleName(s), styleFont(s), num(s.Size),
			formatColour(s.Primary), formatColour(s.Secondary),
			formatColour(s.Outline), formatColour(s.Shadow),
			flag(s.Bold), flag(s.Italic), flag(s.Underline),
			border, num(s.OutlineWidth), num(s.ShadowDepth), int(alignmentValue(s.Alignment)))
	}
	out.WriteString("\n")
}

// renderCue builds one Dialogue text field.
func (w *Writer) renderCue(cue model.Cue, base model.Style, losses *lossNotes, groups []model.RubyGroup) string {
	var b strings.Builder
	if tags := cueLevelTags(cue, base, losses); len(tags) > 0 {
		writeBraces(&b, tags)
	}

	state := spanState{}
	prevOverrides := false
	kursor := time.Duration(0)
	var prevStart, prevEnd time.Duration
	havePrev := false
	for _, group := range groups {
		baseSpan := group.Base
		// ASS has no voice form, so a speaker name drops on write.
		if baseSpan.Voice != nil {
			losses.add("a voice name has no ASS form")
		}
		overrides := styleTags(baseSpan, losses)

		var tags []string
		if prevOverrides || len(overrides) > 0 {
			tags = append(tags, `\r`)
		}
		prevOverrides = len(overrides) > 0
		tags = append(tags, overrides...)
		tags = append(tags, state.scriptTag(baseSpan)...)
		tags = append(tags, state.packedTag(baseSpan)...)
		tags = append(tags, state.verticalTag(baseSpan)...)
		tags = append(tags, state.directionTag(baseSpan)...)

		text := baseSpan.Text
		if len(group.Annotations) > 0 {
			ann := group.Annotations[0]
			if ann.Ruby.Position == model.RubyParenthetical {
				text = baseSpan.Text + "(" + ann.Text + ")"
			} else {
				if ann.Ruby.Position == model.RubyUnder {
					tags = append(tags, `\ytruby2`)
				} else {
					tags = append(tags, `\ytruby`)
				}
				text = "[" + baseSpan.Text + "/" + ann.Text + "]"
			}
			if len(group.Annotations) > 1 {
				losses.add("one ruby base with several readings keeps the first reading")
			}
		}

		timed := baseSpan.Start != 0 || baseSpan.End != 0
		// A style tag can split one syllable into several spans, which
		// share its window. Only the first span of a syllable takes a
		// \k tag, so the later spans inherit the window on read.
		sameWindow := timed && havePrev && baseSpan.Start == prevStart && baseSpan.End == prevEnd
		if timed && !sameWindow {
			var tag string
			tag, kursor = karaokeTag(baseSpan, kursor, losses)
			tags = append(tags, tag)
		}
		if timed {
			prevStart, prevEnd, havePrev = baseSpan.Start, baseSpan.End, true
		} else {
			havePrev = false
		}

		writeBraces(&b, tags)
		b.WriteString(escapeASSText(text))
	}
	return b.String()
}

// spanState tracks the span-level properties that the writer emits as
// changes rather than as full values.
type spanState struct {
	script    model.ScriptKind
	packed    bool
	vertical  model.VerticalMode
	haveVert  bool
	direction model.Direction
}

// scriptTag returns the tag that changes the script offset, if it changed.
func (s *spanState) scriptTag(span model.TextSpan) []string {
	want := model.ScriptRegular
	if span.Script != nil {
		want = span.Script.Kind
	}
	if want == s.script {
		return nil
	}
	s.script = want
	switch want {
	case model.ScriptSuperscript:
		return []string{`\ytsup`}
	case model.ScriptSubscript:
		return []string{`\ytsub`}
	}
	return []string{`\ytsur`}
}

// packedTag returns the tag that changes the packed flag, if it changed.
func (s *spanState) packedTag(span model.TextSpan) []string {
	want := span.Packed != nil && *span.Packed
	if want == s.packed {
		return nil
	}
	s.packed = want
	if want {
		return []string{`\ytpack1`}
	}
	return []string{`\ytpack0`}
}

// verticalTag returns the tag that changes the vertical mode, if it
// changed. A blank \ytvert returns the run to horizontal text, so a span
// can leave vertical text.
func (s *spanState) verticalTag(span model.TextSpan) []string {
	want := model.VerticalNone
	if span.Vertical != nil {
		want = span.Vertical.Mode
	}
	if s.haveVert && want == s.vertical {
		return nil
	}
	clear := s.haveVert && s.vertical != model.VerticalNone && want == model.VerticalNone
	s.vertical, s.haveVert = want, true
	switch want {
	case model.VerticalColumnsRTL:
		return []string{`\ytvert9`}
	case model.VerticalColumnsLTR:
		return []string{`\ytvert7`}
	case model.VerticalRotated:
		return []string{`\ytvert1`}
	case model.VerticalRotatedReversed:
		return []string{`\ytvert3`}
	}
	if clear {
		return []string{`\ytvert`}
	}
	return nil
}

// directionTag returns the tag that changes the reading direction, if it
// changed.
func (s *spanState) directionTag(span model.TextSpan) []string {
	want := model.DirLeftToRight
	if span.Direction != nil {
		want = *span.Direction
	}
	if want == s.direction {
		return nil
	}
	s.direction = want
	if want == model.DirRightToLeft {
		return []string{`\ytdir4`}
	}
	return []string{`\ytdir6`}
}

// styleTags returns the tags for the style overrides of a span. The caller
// places them after a reset tag, so each tag sets an absolute value.
func styleTags(span model.TextSpan, losses *lossNotes) []string {
	var tags []string
	if span.Font != nil {
		tags = append(tags, `\fn`+*span.Font)
	}
	if span.Size != nil {
		tags = append(tags, `\fs`+num(*span.Size))
	}
	if span.Bold != nil {
		tags = append(tags, `\b`+boolDigit(*span.Bold))
	}
	if span.Italic != nil {
		tags = append(tags, `\i`+boolDigit(*span.Italic))
	}
	if span.Underline != nil {
		tags = append(tags, `\u`+boolDigit(*span.Underline))
	}
	if span.Fore != nil {
		tags = append(tags, colourTags(`\c`, `\1a`, *span.Fore)...)
	}
	if span.Secondary != nil {
		tags = append(tags, colourTags(`\2c`, `\2a`, *span.Secondary)...)
	}
	if span.Back != nil {
		tags = append(tags, colourTags(`\3c`, `\3a`, *span.Back)...)
	}
	if span.OutlineWidth != nil {
		tags = append(tags, `\bord`+num(*span.OutlineWidth))
	}
	if span.ShadowDepth != nil {
		tags = append(tags, `\shad`+num(*span.ShadowDepth))
	}
	if span.Strikeout != nil {
		tags = append(tags, `\s`+boolDigit(*span.Strikeout))
	}
	if span.ScaleX != nil {
		tags = append(tags, `\fscx`+num(*span.ScaleX))
	}
	if span.ScaleY != nil {
		tags = append(tags, `\fscy`+num(*span.ScaleY))
	}
	// ASS carries one outline channel and one shadow channel. A glow writes
	// through the outline channel, and every other kind writes through the
	// shadow channel, so a soft or bevel shadow keeps a visible shape.
	solid := 0
	for _, shadow := range span.Shadows {
		if shadow.Kind != model.ShadowGlow {
			solid++
		}
	}
	if solid > 1 {
		losses.add("ASS carries one shadow, so only the last shadow survives")
	}
	for _, shadow := range span.Shadows {
		if shadow.Kind == model.ShadowGlow {
			tags = append(tags, colourTags(`\3c`, `\3a`, shadow.Colour)...)
			continue
		}
		tags = append(tags, colourTags(`\4c`, `\4a`, shadow.Colour)...)
	}
	return tags
}

// cueLevelTags returns the tags that apply to the whole cue: placement,
// effects, vertical layout, and direction.
func cueLevelTags(cue model.Cue, base model.Style, losses *lossNotes) []string {
	var tags []string
	if cue.Layout != nil {
		if cue.Layout.Anchor != nil && *cue.Layout.Anchor != base.Alignment {
			tags = append(tags, fmt.Sprintf(`\an%d`, int(*cue.Layout.Anchor)))
		}
		if cue.Layout.Position != nil {
			tags = append(tags, fmt.Sprintf(`\pos(%s,%s)`,
				num(cue.Layout.Position.X), num(cue.Layout.Position.Y)))
		}
		if cue.Layout.Move != nil {
			tags = append(tags, moveTag(*cue.Layout.Move))
		}
		if cue.Layout.Fade != nil {
			tags = append(tags, fadeTag(*cue.Layout.Fade))
		}
	}
	for _, a := range cue.Animations {
		switch {
		case a.Fade != nil:
			tags = append(tags, fadeTag(*a.Fade))
		case a.Move != nil:
			tags = append(tags, moveTag(*a.Move))
		case a.Shake != nil:
			tags = append(tags, shakeTag(*a.Shake))
		case a.Chroma != nil:
			tags = append(tags, chromaTag(*a.Chroma, losses))
		case a.Keyframes != nil:
			tags = append(tags, keyframeTags(a.Keyframes, losses)...)
		case a.Karaoke != nil:
			tags = append(tags, karaokeTypeTag(*a.Karaoke, losses)...)
		}
	}

	return tags
}

// karaokeTag returns the \k tag for a span and the new karaoke cursor. The
// value is a duration in centiseconds, so the writer reports a gap when the
// spans do not tile.
func karaokeTag(span model.TextSpan, cursor time.Duration, losses *lossNotes) (string, time.Duration) {
	if span.Start > cursor {
		losses.add("a karaoke gap cannot be expressed in ASS")
	}
	start := span.Start
	if start < cursor {
		start = cursor
	}
	duration := span.End - start
	if duration < 10*time.Millisecond {
		duration = 10 * time.Millisecond
	}
	cs := int((duration + 5*time.Millisecond) / (10 * time.Millisecond))
	return fmt.Sprintf(`\k%d`, cs), start + time.Duration(cs)*10*time.Millisecond
}

func moveTag(m model.Move) string {
	base := fmt.Sprintf(`\move(%s,%s,%s,%s)`, num(m.From.X), num(m.From.Y), num(m.To.X), num(m.To.Y))
	if m.Start == 0 && m.End == 0 {
		return base
	}
	return fmt.Sprintf(`\move(%s,%s,%s,%s,%d,%d)`, num(m.From.X), num(m.From.Y), num(m.To.X), num(m.To.Y),
		int(m.Start/time.Millisecond), int(m.End/time.Millisecond))
}

func fadeTag(f model.Fade) string {
	if f.StartAlpha == 0 && f.MidAlpha == 0 && f.EndAlpha == 0 {
		return fmt.Sprintf(`\fad(%d,%d)`, int(f.In/time.Millisecond), int(f.Out/time.Millisecond))
	}
	return fmt.Sprintf(`\fade(%d,%d,%d,%d,%d,%d,%d)`,
		f.StartAlpha, f.MidAlpha, f.EndAlpha,
		int(f.StartIn/time.Millisecond), int(f.EndIn/time.Millisecond),
		int(f.StartOut/time.Millisecond), int(f.EndOut/time.Millisecond))
}

func shakeTag(s model.Shake) string {
	if s.Start != 0 || s.End != 0 {
		return fmt.Sprintf(`\ytshake(%s,%s,%d,%d)`, num(s.RadiusX), num(s.RadiusY),
			int(s.Start/time.Millisecond), int(s.End/time.Millisecond))
	}
	return fmt.Sprintf(`\ytshake(%s,%s)`, num(s.RadiusX), num(s.RadiusY))
}

func chromaTag(c model.Chroma, losses *lossNotes) string {
	var offsetX, offsetY float64
	if len(c.Offsets) > 0 {
		offsetX = -c.Offsets[0].X
		offsetY = -c.Offsets[0].Y
	}
	copies := 3
	if len(c.Colours) > 0 {
		copies = len(c.Colours)
	}
	// The argument form names one offset and spreads the copies from it. The
	// writer records a loss only when the spread can not reproduce the
	// offsets of the IR.
	if len(c.Offsets) > 1 && !sameOffsets(c.Offsets, spreadOffsets(offsetX, offsetY, copies)) {
		losses.add("a chroma with more than one offset keeps the first offset only")
	}
	if len(c.Colours) > 0 {
		var b strings.Builder
		b.WriteString(`\ytchroma(`)
		for _, col := range c.Colours {
			fmt.Fprintf(&b, `&H%02X%02X%02X&,`, col.B, col.G, col.R)
		}
		fmt.Fprintf(&b, `&H%02X&,%s,%s,%d,%d)`, 255-c.Alpha, num(offsetX), num(offsetY),
			int(c.InTime/time.Millisecond), int(c.OutTime/time.Millisecond))
		return b.String()
	}
	return fmt.Sprintf(`\ytchroma(%s,%s,%d,%d)`, num(offsetX), num(offsetY),
		int(c.InTime/time.Millisecond), int(c.OutTime/time.Millisecond))
}

func keyframeTags(keyframes []model.Keyframe, losses *lossNotes) []string {
	var tags []string
	for _, kf := range keyframes {
		var inner strings.Builder
		for _, step := range kf.Steps {
			inner.WriteString(`\` + keyframeTagName(step.Property) + step.Value)
		}
		if inner.Len() == 0 {
			continue
		}
		if kf.Easing != 1 && kf.Easing != 0 {
			tags = append(tags, fmt.Sprintf(`\t(%d,%d,%s,%s)`,
				int(kf.Start/time.Millisecond), int(kf.End/time.Millisecond), num(kf.Easing), inner.String()))
		} else {
			tags = append(tags, fmt.Sprintf(`\t(%d,%d,%s)`,
				int(kf.Start/time.Millisecond), int(kf.End/time.Millisecond), inner.String()))
		}
	}
	if len(keyframes) > 0 && len(tags) == 0 {
		losses.add("a keyframe with no animated value cannot be expressed in ASS")
	}
	return tags
}

// keyframeTagName maps an IR keyframe property back onto its ASS tag name.
func keyframeTagName(property string) string {
	switch property {
	case "fore":
		return "c"
	case "secondary":
		return "2c"
	case "outline":
		return "3c"
	case "shadow":
		return "4c"
	case "forealpha":
		return "1a"
	case "secondaryalpha":
		return "2a"
	case "outlinealpha":
		return "3a"
	case "shadowalpha":
		return "4a"
	case "fontsize":
		return "fs"
	case "outlinewidth":
		return "bord"
	case "shadowdepth":
		return "shad"
	}
	return property
}

func karaokeTypeTag(k model.Karaoke, losses *lossNotes) []string {
	switch k.Kind {
	case model.KaraokeFade:
		return []string{`\ytktFade`}
	case model.KaraokeGlitch:
		return []string{`\ytktGlitch`}
	case model.KaraokeCursor:
		if tag := cursorTag(k, losses); tag != "" {
			return []string{tag}
		}
	}
	return nil
}

// cursorTag renders a \ytkt cursor. It emits the animated form when the
// cursor carries frames, and the static form otherwise.
func cursorTag(k model.Karaoke, losses *lossNotes) string {
	name := "Cursor"
	if k.CursorLeft {
		name = "LCursor"
	}
	if len(k.CursorFrames) > 0 {
		var b strings.Builder
		fmt.Fprintf(&b, `\ytkt(%s,%d`, name, int(k.CursorInterval/time.Millisecond))
		for _, frame := range k.CursorFrames {
			b.WriteString("," + frame.Tags + "," + frame.Text)
		}
		b.WriteString(")")
		return b.String()
	}
	if k.Cursor == "" {
		losses.add("a cursor karaoke type with no text cannot be expressed in ASS")
		return ""
	}
	if k.CursorTags != "" {
		return fmt.Sprintf(`\ytkt(%s,%s,%s)`, name, k.CursorTags, k.Cursor)
	}
	return fmt.Sprintf(`\ytkt(%s,%s)`, name, k.Cursor)
}

// sameOffsets reports whether the writer can rebuild b offsets from the
// first offset of a.
func sameOffsets(a, b []model.Point) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// writeBraces writes an override block when it holds at least one tag.
func writeBraces(b *strings.Builder, tags []string) {
	if len(tags) == 0 {
		return
	}
	b.WriteString("{")
	for _, tag := range tags {
		b.WriteString(tag)
	}
	b.WriteString("}")
}

// colourTags returns the colour tag and the alpha tag of one channel set.
func colourTags(colourTag, alphaTag string, c model.Colour) []string {
	return []string{
		fmt.Sprintf(`%s&H%02X%02X%02X&`, colourTag, c.B, c.G, c.R),
		fmt.Sprintf(`%s&H%02X&`, alphaTag, 255-c.A),
	}
}

// escapeASSText escapes the text of a run. A line break becomes \N and a
// literal backslash becomes \\.
func escapeASSText(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, "\n", `\N`)
}

// formatASSTime renders a duration as H:MM:SS.cc.
func formatASSTime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	cs := int64(d / (10 * time.Millisecond))
	h := cs / 360000
	m := (cs / 6000) % 60
	s := (cs / 100) % 60
	frac := cs % 100
	return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, frac)
}

// formatColour renders a colour as &HAABBGGRR&, where AA is the
// transparency and not the alpha.
func formatColour(c model.Colour) string {
	return fmt.Sprintf("&H%02X%02X%02X%02X", 255-c.A, c.B, c.G, c.R)
}

// num renders a number without a trailing zero.
func num(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// boolDigit renders a boolean as the ASS flag.
func boolDigit(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// flag renders a style boolean, where -1 is true.
func flag(b bool) string {
	if b {
		return "-1"
	}
	return "0"
}

// alignmentValue keeps an alignment inside the valid range.
func alignmentValue(a model.Anchor) model.Anchor {
	if a < model.AnchorBottomLeft || a > model.AnchorTopRight {
		return model.AnchorBottomCentre
	}
	return a
}

// styleName keeps a style name non-empty.
func styleName(s model.Style) string {
	if s.Name == "" {
		return "Default"
	}
	return s.Name
}

// styleFont keeps a font name non-empty.
func styleFont(s model.Style) string {
	if s.Font == "" {
		return "Arial"
	}
	return s.Font
}
