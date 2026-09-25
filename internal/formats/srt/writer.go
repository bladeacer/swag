package srt

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bladeacer/swag/internal/envelope"
	"github.com/bladeacer/swag/internal/model"
)

// lossNotes collects the degradations of one write.
type lossNotes struct {
	notes []string
}

func (l *lossNotes) add(format string, args ...any) {
	l.notes = append(l.notes, fmt.Sprintf(format, args...))
}

// Writer emits SubRip documents from the IR.
type Writer struct{}

// NewWriter returns a SubRip writer.
func NewWriter() *Writer { return &Writer{} }

// Name returns the registry name of the format.
func (w *Writer) Name() string { return FormatName }

// Render writes doc to sink as SubRip. The returned slice carries one
// entry per feature that a plain SubRip consumer cannot read. A write with
// at least one such feature also appends the integrity block, which keeps
// the whole document for a swag reader.
func (w *Writer) Render(doc *model.Document, sink io.Writer) ([]string, error) {
	var losses lossNotes
	var out strings.Builder

	for i := range doc.Cues {
		cue := doc.Cues[i]
		recordLosses(cue, &losses)

		fmt.Fprintf(&out, "%d\n", i+1)
		fmt.Fprintf(&out, "%s --> %s\n", formatTime(cue.Start), formatTime(cue.End))
		out.WriteString(renderText(cue))
		out.WriteString("\n\n")
	}

	if _, err := io.WriteString(sink, out.String()); err != nil {
		return losses.notes, fmt.Errorf("write srt: %w", err)
	}
	// The cue loop ends the file with a blank line, so the block starts a
	// fresh paragraph. A plain consumer stops at the last cue.
	if len(losses.notes) > 0 {
		if err := envelope.Write(sink, doc); err != nil {
			return losses.notes, fmt.Errorf("write srt: %w", err)
		}
	}
	return losses.notes, nil
}

// recordLosses notes the features of cue that SubRip cannot carry.
func recordLosses(cue model.Cue, losses *lossNotes) {
	if cue.Karaoke() {
		losses.add("karaoke timing")
	}
	if cue.Layout != nil {
		losses.add("positioning")
	}
	for _, a := range cue.Animations {
		losses.add("animation")
		_ = a
	}
	for i, span := range cue.Spans {
		if span.Ruby != nil {
			losses.add("ruby text in span %d", i)
		}
		if span.Vertical != nil && span.Vertical.Mode != model.VerticalNone {
			losses.add("vertical text in span %d", i)
		}
		if span.Script != nil && span.Script.Kind != model.ScriptRegular {
			losses.add("script offset in span %d", i)
		}
		if span.Fore != nil && !span.Fore.Opaque() {
			losses.add("transparency in span %d", i)
		}
		if span.Fore != nil && *span.Fore != DefaultStyle().Primary {
			losses.add("foreground colour in span %d", i)
		}
		if len(span.Shadows) > 0 {
			losses.add("shadow effects in span %d", i)
		}
		if span.Direction != nil && *span.Direction == model.DirRightToLeft {
			losses.add("right-to-left marking in span %d", i)
		}
		if span.Strikeout != nil && *span.Strikeout {
			losses.add("strikeout in span %d", i)
		}
		if scaleChanged(span.ScaleX) || scaleChanged(span.ScaleY) {
			losses.add("glyph scale in span %d", i)
		}
		if span.Voice != nil {
			losses.add("voice name in span %d", i)
		}
	}
}

// scaleChanged reports whether a glyph scale override differs from the
// original size. SubRip carries no scale, so the writer records a loss for
// it.
func scaleChanged(v *float64) bool { return v != nil && *v != 100 }

// renderText renders the spans of cue as one SRT text block.
func renderText(cue model.Cue) string {
	groups := model.RubyGroups(cue.Spans)
	var lines []string
	var b strings.Builder
	for _, group := range groups {
		b.WriteString(renderSpan(group.Base))
		for _, ann := range group.Annotations {
			// Ruby has no SRT form: the reading rides in brackets.
			b.WriteString("(" + ann.Text + ")")
		}
	}
	lines = append(lines, b.String())
	return strings.Join(lines, "\n")
}

// renderSpan renders one span with its italic and bold tags.
func renderSpan(span model.TextSpan) string {
	var b strings.Builder
	if isTrue(span.Bold) {
		b.WriteString(tagBoldOn)
	}
	if isTrue(span.Italic) {
		b.WriteString(tagItalicOn)
	}
	b.WriteString(span.Text)
	if isTrue(span.Italic) {
		b.WriteString(tagItalicOff)
	}
	if isTrue(span.Bold) {
		b.WriteString(tagBoldOff)
	}
	return b.String()
}

// isTrue dereferences an optional bool.
func isTrue(v *bool) bool {
	return v != nil && *v
}

// formatTime renders a duration as hh:mm:ss,mmm.
func formatTime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	ms := int64(d / time.Millisecond)
	hours := ms / (3600 * 1000)
	minutes := (ms % (3600 * 1000)) / (60 * 1000)
	seconds := (ms % (60 * 1000)) / 1000
	millis := ms % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, millis)
}
