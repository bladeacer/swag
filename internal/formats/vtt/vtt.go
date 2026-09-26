// SPDX-License-Identifier: Apache-2.0

// Package vtt reads and writes WebVTT subtitles.
//
// The reader maps a cue timing line onto the IR cue, the align setting onto
// the cue anchor, and the position setting onto the cue position. It maps
// the inline tags <i>, <b>, and <u> onto span overrides, and the voice
// annotation <v Speaker> onto TextSpan.Voice. It strips every other tag and
// keeps its text.
//
// The writer emits inline tags for the style overrides it can express, the
// voice annotation of a span, and the align and position settings. It
// reports the features it drops, and a lossy write appends the integrity
// block of internal/envelope as a NOTE block, which the format ignores.
//
// # The voice annotation
//
// A voice span names the speaker of the text, as in
// <v Roger Bingham>We are in the Milky Way. The name reaches the IR through
// TextSpan.Voice and leaves it again, so a WebVTT to WebVTT round trip
// keeps the speaker. A format without a voice form drops the name and keeps
// the text.
package vtt

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/bladeacer/swag/internal/envelope"
	"github.com/bladeacer/swag/internal/model"
)

// FormatName is the registry name of the format.
const FormatName = "vtt"

// The default frame, which the position setting uses when the document
// carries no video size.
const (
	defaultWidth  = 1280
	defaultHeight = 720
)

// Reader parses a WebVTT document.
type Reader struct{}

// NewReader returns a WebVTT reader.
func NewReader() *Reader { return &Reader{} }

// Name returns the registry name of the format.
func (r *Reader) Name() string { return FormatName }

// DefaultStyle returns the implicit style of WebVTT, which carries no style
// table of its own.
func DefaultStyle() model.Style {
	return model.Style{
		Name:      "Default",
		Font:      "Arial",
		Size:      20,
		Primary:   model.NewColour(255, 255, 255, 255),
		Secondary: model.NewColour(255, 255, 255, 255),
		Alignment: model.AnchorBottomCentre,
	}
}

// Parse reads a WebVTT document from source.
func (r *Reader) Parse(source io.Reader) (*model.Document, error) {
	scanner := bufio.NewScanner(source)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, strings.TrimRight(scanner.Text(), "\r"))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read vtt: %w", err)
	}

	body, embedded, err := envelope.Extract(lines)
	if err != nil {
		return nil, fmt.Errorf("parse vtt: %w", err)
	}
	if embedded != nil {
		return embedded, nil
	}
	lines = body

	doc := &model.Document{
		Styles:          []model.Style{DefaultStyle()},
		VideoDimensions: model.Point{X: defaultWidth, Y: defaultHeight},
	}

	i := 0
	if i < len(lines) {
		signature := strings.TrimPrefix(lines[i], "\ufeff")
		if strings.HasPrefix(signature, "WEBVTT") {
			i++
		}
	}
	for i < len(lines) {
		if strings.TrimSpace(lines[i]) == "" {
			i++
			continue
		}
		start := i
		for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
			i++
		}
		block := lines[start:i]
		timing := -1
		for j, line := range block {
			if strings.Contains(line, "-->") {
				timing = j
				break
			}
		}
		if timing < 0 {
			// A block with no timing line, for example a NOTE, is skipped.
			continue
		}
		cue, err := parseTiming(block[timing])
		if err != nil {
			return nil, err
		}
		cue.Spans = parseSpans(strings.Join(block[timing+1:], "\n"))
		doc.Cues = append(doc.Cues, cue)
	}
	return doc, nil
}

// parseTiming reads a timing line and its settings into a cue.
func parseTiming(line string) (model.Cue, error) {
	arrow := strings.Index(line, "-->")
	start, err := parseTimestamp(strings.TrimSpace(line[:arrow]))
	if err != nil {
		return model.Cue{}, err
	}
	rest := strings.TrimSpace(line[arrow+3:])
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return model.Cue{}, fmt.Errorf("parse vtt timing %q: no end time", line)
	}
	end, err := parseTimestamp(fields[0])
	if err != nil {
		return model.Cue{}, err
	}
	cue := model.Cue{Start: start, End: end}
	for _, setting := range fields[1:] {
		name, value, ok := strings.Cut(setting, ":")
		if !ok {
			continue
		}
		switch name {
		case "align":
			cue.Layout = withAnchor(cue.Layout, alignAnchor(value))
		case "position":
			if pct, err := strconv.ParseFloat(strings.TrimSuffix(value, "%"), 64); err == nil {
				cue.Layout = withPosition(cue.Layout, &model.Point{X: pct / 100 * defaultWidth, Y: 0})
			}
		}
	}
	return cue, nil
}

// alignAnchor maps a WebVTT align value onto a horizontal numpad anchor.
func alignAnchor(value string) *model.Anchor {
	var a model.Anchor
	switch value {
	case "start", "left":
		a = model.AnchorBottomLeft
	case "center", "middle":
		a = model.AnchorBottomCentre
	case "end", "right":
		a = model.AnchorBottomRight
	default:
		return nil
	}
	return &a
}

// withAnchor returns layout with the anchor set.
func withAnchor(layout *model.Layout, anchor *model.Anchor) *model.Layout {
	if anchor == nil {
		return layout
	}
	if layout == nil {
		layout = &model.Layout{}
	}
	layout.Anchor = anchor
	return layout
}

// withPosition returns layout with the position set.
func withPosition(layout *model.Layout, pos *model.Point) *model.Layout {
	if layout == nil {
		layout = &model.Layout{}
	}
	layout.Position = pos
	return layout
}

// parseTimestamp reads a WebVTT timestamp of the form HH:MM:SS.mmm or
// MM:SS.mmm.
func parseTimestamp(s string) (time.Duration, error) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, fmt.Errorf("parse vtt timestamp %q: want MM:SS.mmm or HH:MM:SS.mmm", s)
	}
	secPart := parts[len(parts)-1]
	sec, frac, _ := strings.Cut(secPart, ".")
	seconds, err := strconv.Atoi(sec)
	if err != nil {
		return 0, fmt.Errorf("parse vtt timestamp %q: bad seconds", s)
	}
	var minutes int
	if len(parts) == 3 {
		minutes, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("parse vtt timestamp %q: bad hours", s)
		}
	}
	mins, err := strconv.Atoi(parts[len(parts)-2])
	if err != nil {
		return 0, fmt.Errorf("parse vtt timestamp %q: bad minutes", s)
	}
	var ms int
	if frac != "" {
		for len(frac) < 3 {
			frac += "0"
		}
		frac = frac[:3]
		ms, err = strconv.Atoi(frac)
		if err != nil {
			return 0, fmt.Errorf("parse vtt timestamp %q: bad milliseconds", s)
		}
	}
	return time.Duration(minutes)*time.Hour + time.Duration(mins)*time.Minute +
		time.Duration(seconds)*time.Second + time.Duration(ms)*time.Millisecond, nil
}

// parseSpans turns a cue payload into spans. The tags <i>, <b>, and <u>
// open and close a style, and <v Speaker> and </v> open and close a voice
// annotation. Every other tag is dropped and its text stays.
func parseSpans(text string) []model.TextSpan {
	var spans []model.TextSpan
	var bold, italic, underline bool
	var voice *string
	var buf strings.Builder
	flush := func() {
		if buf.Len() == 0 {
			return
		}
		span := model.TextSpan{Text: decodeEntities(buf.String())}
		if bold {
			span.Bold = ptr(true)
		}
		if italic {
			span.Italic = ptr(true)
		}
		if underline {
			span.Underline = ptr(true)
		}
		if voice != nil {
			span.Voice = ptr(*voice)
		}
		spans = append(spans, span)
		buf.Reset()
	}
	for i := 0; i < len(text); {
		if text[i] != '<' {
			buf.WriteByte(text[i])
			i++
			continue
		}
		end := strings.IndexByte(text[i:], '>')
		if end < 0 {
			buf.WriteString(text[i:])
			break
		}
		flush()
		switch tag := text[i+1 : i+end]; {
		case tag == "i":
			italic = true
		case tag == "/i":
			italic = false
		case tag == "b":
			bold = true
		case tag == "/b":
			bold = false
		case tag == "u":
			underline = true
		case tag == "/u":
			underline = false
		case tag == "/v":
			voice = nil
		case tag == "v":
			// A voice annotation with no name still opens a voice span,
			// and an empty name is the name it carries.
			voice = ptr("")
		case strings.HasPrefix(tag, "v "):
			voice = ptr(strings.TrimSpace(tag[2:]))
		}
		i += end + 1
	}
	flush()
	return spans
}

// The entity replacers are built once. A replacer holds a trie that costs
// time and memory to build, and a reader or writer would otherwise build it
// for every cue payload.
var (
	// entityDecoder resolves the character references of a cue payload.
	entityDecoder = strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&nbsp;", "\u00a0",
		"&lrm;", "\u200e",
		"&rlm;", "\u200f",
	)
	// entityEncoder escapes the characters that carry meaning in a payload.
	entityEncoder = strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
)

// decodeEntities resolves the character references of a cue payload.
func decodeEntities(s string) string {
	return entityDecoder.Replace(s)
}

// encodeText escapes the characters that carry meaning in a cue payload.
func encodeText(s string) string {
	return entityEncoder.Replace(s)
}

// lossNotes collects the degradations of one write.
type lossNotes struct{ notes []string }

func (l *lossNotes) add(format string, args ...any) {
	l.notes = append(l.notes, fmt.Sprintf(format, args...))
}

// Writer renders WebVTT documents.
type Writer struct{}

// NewWriter returns a WebVTT writer.
func NewWriter() *Writer { return &Writer{} }

// Name returns the registry name of the format.
func (w *Writer) Name() string { return FormatName }

// Render writes doc to sink as WebVTT. The returned slice carries one entry
// per feature the format cannot express.
func (w *Writer) Render(doc *model.Document, sink io.Writer) ([]string, error) {
	var losses lossNotes
	width := doc.VideoDimensions.X
	if width <= 0 {
		width = defaultWidth
	}

	var out strings.Builder
	out.WriteString("WEBVTT\n\n")
	for i := range doc.Cues {
		cue := doc.Cues[i]
		recordLosses(cue, &losses)
		fmt.Fprintf(&out, "%s --> %s%s\n", formatTime(cue.Start), formatTime(cue.End), cueSettings(cue, width))
		out.WriteString(renderSpans(cue))
		out.WriteString("\n\n")
	}

	if _, err := io.WriteString(sink, out.String()); err != nil {
		return losses.notes, fmt.Errorf("write vtt: %w", err)
	}
	// A lossy write keeps the whole document in a NOTE block, which the
	// format treats as a comment. The cue loop ends the file with a blank
	// line, so the block stands on its own.
	if len(losses.notes) > 0 {
		if err := envelope.Write(sink, doc); err != nil {
			return losses.notes, fmt.Errorf("write vtt: %w", err)
		}
	}
	return losses.notes, nil
}

// recordLosses notes the features of cue that WebVTT cannot carry.
func recordLosses(cue model.Cue, losses *lossNotes) {
	if cue.Karaoke() {
		losses.add("karaoke timing")
	}
	if cue.Layout != nil {
		if cue.Layout.Fade != nil {
			losses.add("cue fade")
		}
		if cue.Layout.Move != nil {
			losses.add("cue move")
		}
		if cue.Layout.Anchor != nil && anchorRow(*cue.Layout.Anchor) != 0 {
			losses.add("vertical alignment")
		}
	}
	for _, a := range cue.Animations {
		losses.add("animation")
		_ = a
	}
	for _, span := range cue.Spans {
		if span.Fore != nil {
			losses.add("foreground colour")
		}
		if span.Secondary != nil {
			losses.add("secondary colour")
		}
		if span.Back != nil {
			losses.add("background colour")
		}
		if span.Font != nil {
			losses.add("font")
		}
		if span.Size != nil {
			losses.add("font size")
		}
		if span.Strikeout != nil && *span.Strikeout {
			losses.add("strikeout")
		}
		if scaleChanged(span.ScaleX) || scaleChanged(span.ScaleY) {
			losses.add("glyph scale")
		}
		if len(span.Shadows) > 0 {
			losses.add("shadow effects")
		}
		if span.OutlineWidth != nil {
			losses.add("outline width")
		}
		if span.Vertical != nil && span.Vertical.Mode != model.VerticalNone {
			losses.add("vertical text")
		}
		if span.Script != nil && span.Script.Kind != model.ScriptRegular {
			losses.add("script offset")
		}
		if span.Direction != nil && *span.Direction == model.DirRightToLeft {
			losses.add("right-to-left marking")
		}
		if span.Packed != nil && *span.Packed {
			losses.add("packing")
		}
		if span.Ruby != nil {
			losses.add("ruby text")
		}
	}
}

// anchorRow returns 0 for the bottom row, 1 for the middle row, and 2 for
// the top row of a numpad anchor.
func anchorRow(a model.Anchor) int {
	switch a {
	case model.AnchorMiddleLeft, model.AnchorCentre, model.AnchorMiddleRight:
		return 1
	case model.AnchorTopLeft, model.AnchorTopCentre, model.AnchorTopRight:
		return 2
	}
	return 0
}

// cueSettings renders the align and position settings of a cue.
func cueSettings(cue model.Cue, width float64) string {
	if cue.Layout == nil {
		return ""
	}
	var parts []string
	if cue.Layout.Anchor != nil {
		if align := vttAlign(*cue.Layout.Anchor); align != "" {
			parts = append(parts, "align:"+align)
		}
	}
	if cue.Layout.Position != nil && width > 0 {
		pct := cue.Layout.Position.X / width * 100
		parts = append(parts, "position:"+strconv.FormatFloat(pct, 'f', -1, 64)+"%")
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " ")
}

// vttAlign maps a numpad anchor onto a WebVTT align value.
func vttAlign(a model.Anchor) string {
	switch a {
	case model.AnchorBottomLeft, model.AnchorMiddleLeft, model.AnchorTopLeft:
		return "start"
	case model.AnchorBottomCentre, model.AnchorCentre, model.AnchorTopCentre:
		return "center"
	case model.AnchorBottomRight, model.AnchorMiddleRight, model.AnchorTopRight:
		return "end"
	}
	return ""
}

// renderSpans renders the spans of a cue with their inline tags. A ruby
// annotation falls back to brackets, because WebVTT has no ruby form.
func renderSpans(cue model.Cue) string {
	var b strings.Builder
	for _, group := range model.RubyGroups(cue.Spans) {
		span := group.Base
		var close []string
		if span.Voice != nil {
			// A nameless annotation writes as <v>, because an empty
			// annotation is not valid.
			if *span.Voice == "" {
				b.WriteString("<v>")
			} else {
				b.WriteString("<v " + encodeText(*span.Voice) + ">")
			}
			close = append(close, "</v>")
		}
		if isTrue(span.Bold) {
			b.WriteString("<b>")
			close = append(close, "</b>")
		}
		if isTrue(span.Italic) {
			b.WriteString("<i>")
			close = append(close, "</i>")
		}
		if isTrue(span.Underline) {
			b.WriteString("<u>")
			close = append(close, "</u>")
		}
		b.WriteString(encodeText(span.Text))
		for _, ann := range group.Annotations {
			b.WriteString("(" + encodeText(ann.Text) + ")")
		}
		for i := len(close) - 1; i >= 0; i-- {
			b.WriteString(close[i])
		}
	}
	return b.String()
}

// formatTime renders a duration as HH:MM:SS.mmm.
func formatTime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	ms := int64(d / time.Millisecond)
	h := ms / 3600000
	m := (ms / 60000) % 60
	s := (ms / 1000) % 60
	frac := ms % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, frac)
}

// ptr returns a pointer to a copy of v.
func ptr[T any](v T) *T { return &v }

// isTrue dereferences an optional bool.
func isTrue(v *bool) bool { return v != nil && *v }

// scaleChanged reports whether a glyph scale override differs from the
// original size. WebVTT carries no scale, so the writer records a loss for
// it.
func scaleChanged(v *float64) bool { return v != nil && *v != 100 }
