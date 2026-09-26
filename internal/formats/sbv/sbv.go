// Package sbv reads and writes YouTube SBV (.sbv) subtitles.
//
// SBV carries plain text only: one timing line (start,end joined by a
// comma), one or more text lines, then a blank line. Multi-line text
// renders on separate display lines. Styling, positioning, and effects
// degrade on write, and the writer records each loss.
//
// A lossy write also appends the integrity block of internal/envelope, which
// holds the whole document. A plain subtitle player stops at the last cue
// and ignores the block, and a swag reader restores every feature from it.
// A document that fits in plain SBV writes no block, so a plain file stays
// plain.
package sbv

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/bladeacer/swag/internal/envelope"
	"github.com/bladeacer/swag/internal/model"
)

// FormatName is the registry name of the format.
const FormatName = "sbv"

// Reader parses an SBV document.
type Reader struct{}

// NewReader returns an SBV reader.
func NewReader() *Reader { return &Reader{} }

// Name returns the registry name of the format.
func (r *Reader) Name() string { return FormatName }

// Parse reads an SBV document from source. It streams the body one line at a
// time, so a large file builds no slice of lines.
func (r *Reader) Parse(source io.Reader) (*model.Document, error) {
	doc := &model.Document{
		Metadata:        map[string]string{},
		VideoDimensions: model.Point{X: 1280, Y: 720},
		Styles:          []model.Style{DefaultStyle()},
	}
	embedded, err := envelope.Read(source, func(line string) error {
		return parseLine(doc, line)
	})
	if err != nil {
		return nil, fmt.Errorf("parse sbv: %w", err)
	}
	if embedded != nil {
		return embedded, nil
	}
	return doc, nil
}

// parseLine folds one SBV body line into doc.
func parseLine(doc *model.Document, line string) error {
	trimmed := strings.TrimSpace(line)

	switch {
	case trimmed == "":
		// A blank line closes the current cue.
	case looksLikeTiming(trimmed):
		start, end, err := parseTiming(trimmed)
		if err != nil {
			return err
		}
		doc.Cues = append(doc.Cues, model.Cue{Start: start, End: end})
	case len(doc.Cues) > 0:
		cue := &doc.Cues[len(doc.Cues)-1]
		if len(cue.Spans) > 0 {
			// Each extra text line is its own display line.
			cue.Spans[len(cue.Spans)-1].Text += "\n"
		}
		cue.Spans = append(cue.Spans, model.TextSpan{Text: line})
	}
	return nil
}

// looksLikeTiming reports whether the line carries the SBV timing shape:
// two timestamps joined by a comma, with the fractional dot in the last
// segment. Text lines never match.
func looksLikeTiming(line string) bool {
	comma := strings.Index(line, ",")
	if comma < 0 {
		return false
	}
	head, tail := line[:comma], line[comma+1:]
	// A text line with a comma rarely carries two colons, and its last
	// segment has no dot followed by digits.
	return strings.Count(head, ":") >= 1 &&
		strings.Count(tail, ":") >= 1 &&
		strings.Contains(head, ".") && strings.Contains(tail, ".")
}

// parseTiming parses "0:00:01.000,0:00:04.000".
func parseTiming(line string) (time.Duration, time.Duration, error) {
	comma := strings.Index(line, ",")
	if comma < 0 {
		return 0, 0, fmt.Errorf("parse sbv timing %q: missing comma", line)
	}
	start, err := parseTimestamp(strings.TrimSpace(line[:comma]))
	if err != nil {
		return 0, 0, err
	}
	end, err := parseTimestamp(strings.TrimSpace(line[comma+1:]))
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}

// parseTimestamp parses h:mm:ss.mmm. The hours field can carry more than
// one digit. It finds the two colons instead of splitting the string, so a
// parse builds no slice for the parts.
func parseTimestamp(ts string) (time.Duration, error) {
	if strings.Count(ts, ":") != 2 {
		return 0, fmt.Errorf("parse sbv timestamp %q: want h:mm:ss.mmm", ts)
	}
	first := strings.Index(ts, ":")
	second := first + 1 + strings.Index(ts[first+1:], ":")
	h, err := strconv.Atoi(strings.TrimSpace(ts[:first]))
	if err != nil {
		return 0, fmt.Errorf("parse sbv timestamp %q: bad hours", ts)
	}
	m, err := strconv.Atoi(ts[first+1 : second])
	if err != nil {
		return 0, fmt.Errorf("parse sbv timestamp %q: bad minutes", ts)
	}
	sec, err := strconv.ParseFloat(ts[second+1:], 64)
	if err != nil {
		return 0, fmt.Errorf("parse sbv timestamp %q: bad seconds", ts)
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute +
		time.Duration(sec*float64(time.Second)), nil
}

// DefaultStyle returns the implicit style of a plain format.
func DefaultStyle() model.Style {
	return model.Style{
		Name:         "Default",
		Font:         "Roboto",
		Size:         20,
		Primary:      model.NewColour(255, 255, 255, 254),
		Secondary:    model.NewColour(120, 120, 120, 254),
		Outline:      model.NewColour(0, 0, 0, 254),
		OutlineWidth: 2,
		Alignment:    model.AnchorBottomCentre,
	}
}

// Writer emits SBV documents from the IR.
type Writer struct{}

// NewWriter returns an SBV writer.
func NewWriter() *Writer { return &Writer{} }

// Name returns the registry name of the format.
func (w *Writer) Name() string { return FormatName }

// Render writes doc to sink as SBV. The returned slice carries one entry
// per feature that a plain SBV consumer cannot read. A write with at least
// one such feature also appends the integrity block, which keeps the whole
// document for a swag reader.
func (w *Writer) Render(doc *model.Document, sink io.Writer) ([]string, error) {
	var losses []string
	note := func(what string) { losses = append(losses, what) }

	var out strings.Builder
	out.Grow(len(doc.Cues) * 64)
	var groups []model.RubyGroup
	for i := range doc.Cues {
		cue := doc.Cues[i]
		groups = model.RubyGroupsInto(groups[:0], cue.Spans)
		if cue.Karaoke() {
			note("karaoke timing")
		}
		if cue.Layout != nil {
			note("positioning")
		}
		for _, span := range cue.Spans {
			if isTrue(span.Bold) || isTrue(span.Italic) || isTrue(span.Strikeout) {
				note("inline styling")
			}
			if scaleChanged(span.ScaleX) || scaleChanged(span.ScaleY) {
				note("glyph scale")
			}
			if span.Ruby != nil {
				note("ruby text")
			}
			if span.Voice != nil {
				note("voice name")
			}
		}

		fmt.Fprintf(&out, "%s,%s\n", formatTime(cue.Start), formatTime(cue.End))
		out.WriteString(renderText(groups))
		out.WriteString("\n\n")
	}

	if _, err := io.WriteString(sink, out.String()); err != nil {
		return losses, fmt.Errorf("write sbv: %w", err)
	}
	// The cue loop ends the file with a blank line, so the block starts a
	// fresh paragraph. A plain consumer stops at the last cue.
	if len(losses) > 0 {
		if err := envelope.Write(sink, doc); err != nil {
			return losses, fmt.Errorf("write sbv: %w", err)
		}
	}
	return losses, nil
}

// renderText renders the groups of one cue, falling back to brackets for
// ruby annotations. The caller owns the grouping, so a render reuses it for
// the next cue.
func renderText(groups []model.RubyGroup) string {
	var b strings.Builder
	for _, group := range groups {
		b.WriteString(group.Base.Text)
		for _, ann := range group.Annotations {
			b.WriteString("(" + ann.Text + ")")
		}
	}
	return b.String()
}

// scaleChanged reports whether a glyph scale override differs from the
// original size. SBV carries no scale, so the writer records a loss for it.
func scaleChanged(v *float64) bool { return v != nil && *v != 100 }

// isTrue dereferences an optional bool.
func isTrue(v *bool) bool {
	return v != nil && *v
}

// formatTime renders a duration as h:mm:ss.mmm.
func formatTime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	ms := int64(d / time.Millisecond)
	hours := ms / (3600 * 1000)
	minutes := (ms % (3600 * 1000)) / (60 * 1000)
	seconds := (ms % (60 * 1000)) / 1000
	millis := ms % 1000
	return fmt.Sprintf("%d:%02d:%02d.%03d", hours, minutes, seconds, millis)
}
