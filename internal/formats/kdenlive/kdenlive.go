// SPDX-License-Identifier: Apache-2.0

// Package kdenlive reads and writes the subtitle JSON of the Kdenlive video
// editor.
//
// Kdenlive keeps a subtitle track as a JSON array. Each element carries a
// layer, the start position in seconds, and the dialogue as an ASS event
// line. The dialogue holds the end time, the style, the margins, and the
// text, with the two-character sequence \N for a line break.
//
// The reader maps the start position and the end time of the dialogue onto
// a cue and keeps the text. The writer emits one element per cue with layer
// zero. Styling, positioning, and effects degrade on write, and the writer
// records each loss.
package kdenlive

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

// FormatName is the registry name of the format.
const FormatName = "kdenlive"

// dialoguePrefix opens the dialogue field of an event line.
const dialoguePrefix = "Dialogue:"

// fieldCount is the number of comma-separated fields in an event line:
// Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text.
const fieldCount = 10

// DefaultStyle returns the implicit style of a Kdenlive track. The format
// carries no style table, so the writer always names the Default style.
func DefaultStyle() model.Style {
	return model.Style{
		Name:         "Default",
		Font:         "Arial",
		Size:         20,
		Primary:      model.NewColour(255, 255, 255, 255),
		Secondary:    model.NewColour(255, 255, 255, 255),
		Outline:      model.NewColour(0, 0, 0, 255),
		OutlineWidth: 0,
		Alignment:    model.AnchorBottomCentre,
	}
}

// entry is one element of the subtitle JSON array.
type entry struct {
	// Layer is the subtitle track layer.
	Layer int `json:"layer"`
	// StartPos is the cue start in seconds.
	StartPos float64 `json:"startPos"`
	// Dialogue is the event line of the cue.
	Dialogue string `json:"dialogue"`
}

// Reader parses a Kdenlive subtitle JSON document.
type Reader struct{}

// NewReader returns a Kdenlive reader.
func NewReader() *Reader { return &Reader{} }

// Name returns the registry name of the format.
func (r *Reader) Name() string { return FormatName }

// Parse reads a Kdenlive subtitle JSON document from source.
func (r *Reader) Parse(source io.Reader) (*model.Document, error) {
	data, err := io.ReadAll(source)
	if err != nil {
		return nil, fmt.Errorf("read kdenlive: %w", err)
	}
	var entries []entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse kdenlive: %w", err)
	}

	doc := &model.Document{
		Metadata:        map[string]string{},
		VideoDimensions: model.Point{X: 1280, Y: 720},
		Styles:          []model.Style{DefaultStyle()},
	}
	for i, e := range entries {
		cue, err := e.toCue()
		if err != nil {
			return nil, fmt.Errorf("parse kdenlive entry %d: %w", i, err)
		}
		doc.Cues = append(doc.Cues, cue)
	}
	return doc, nil
}

// toCue converts one entry into a cue.
func (e entry) toCue() (model.Cue, error) {
	fields, err := splitDialogue(e.Dialogue)
	if err != nil {
		return model.Cue{}, err
	}
	end, err := parseTime(fields[2])
	if err != nil {
		return model.Cue{}, err
	}
	start := time.Duration(e.StartPos * float64(time.Second))
	text := strings.ReplaceAll(fields[fieldCount-1], `\N`, "\n")
	span := model.TextSpan{Text: stripTags(text)}
	return model.Cue{Start: start, End: end, Spans: []model.TextSpan{span}}, nil
}

// splitDialogue reads an event line into its fields. A leading Dialogue:
// label is optional, and the Text field keeps the commas inside it.
func splitDialogue(dialogue string) ([]string, error) {
	line := strings.TrimSpace(dialogue)
	if strings.HasPrefix(line, dialoguePrefix) {
		line = strings.TrimSpace(strings.TrimPrefix(line, dialoguePrefix))
	}
	fields := strings.SplitN(line, ",", fieldCount)
	if len(fields) != fieldCount {
		return nil, fmt.Errorf("dialogue %q: want %d comma-separated fields, got %d", dialogue, fieldCount, len(fields))
	}
	return fields, nil
}

// stripTags removes the override blocks of an event line. The format has no
// styling of its own, so the reader keeps the text and drops the tags.
func stripTags(text string) string {
	var b strings.Builder
	depth := 0
	for _, r := range text {
		switch {
		case r == '{':
			depth++
		case r == '}' && depth > 0:
			depth--
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// parseTime reads an ASS timestamp of the form h:mm:ss.cc.
func parseTime(s string) (time.Duration, error) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("parse ASS time %q: want h:mm:ss.cc", s)
	}
	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("parse ASS time %q: bad hours", s)
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("parse ASS time %q: bad minutes", s)
	}
	sec, frac, _ := strings.Cut(parts[2], ".")
	seconds, err := strconv.Atoi(sec)
	if err != nil {
		return 0, fmt.Errorf("parse ASS time %q: bad seconds", s)
	}
	var centiseconds int
	if frac != "" {
		for len(frac) < 2 {
			frac += "0"
		}
		if centiseconds, err = strconv.Atoi(frac[:2]); err != nil {
			return 0, fmt.Errorf("parse ASS time %q: bad fraction", s)
		}
	}
	return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second + time.Duration(centiseconds)*10*time.Millisecond, nil
}

// formatTime renders a duration as h:mm:ss.cc.
func formatTime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	ms := d / time.Millisecond
	hours := ms / 3600000
	minutes := (ms / 60000) % 60
	seconds := (ms / 1000) % 60
	centis := (ms % 1000) / 10
	return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, seconds, centis)
}

// lossNotes collects the degradations of one write.
type lossNotes struct{ notes []string }

func (l *lossNotes) add(format string, args ...any) {
	l.notes = append(l.notes, fmt.Sprintf(format, args...))
}

// Writer renders Kdenlive subtitle JSON documents.
type Writer struct{}

// NewWriter returns a Kdenlive writer.
func NewWriter() *Writer { return &Writer{} }

// Name returns the registry name of the format.
func (w *Writer) Name() string { return FormatName }

// Render writes doc to sink as Kdenlive subtitle JSON. The returned slice
// carries one entry per feature the format cannot express.
func (w *Writer) Render(doc *model.Document, sink io.Writer) ([]string, error) {
	var losses lossNotes
	entries := make([]entry, 0, len(doc.Cues))
	for i := range doc.Cues {
		cue := doc.Cues[i]
		recordLosses(cue, &losses)
		entries = append(entries, entry{
			Layer:    0,
			StartPos: float64(cue.Start) / float64(time.Second),
			Dialogue: dialogueLine(cue),
		})
	}

	encoder := json.NewEncoder(sink)
	encoder.SetIndent("", "  ")
	// Turn off HTML escaping so a text with an angle bracket or an
	// ampersand survives the round trip unchanged.
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(entries); err != nil {
		return losses.notes, fmt.Errorf("write kdenlive: %w", err)
	}
	return losses.notes, nil
}

// dialogueLine renders the event line of a cue.
func dialogueLine(cue model.Cue) string {
	text := strings.ReplaceAll(renderText(cue), "\n", `\N`)
	return fmt.Sprintf("%s 0,%s,%s,Default,,0,0,0,,%s",
		dialoguePrefix, formatTime(cue.Start), formatTime(cue.End), text)
}

// renderText renders the spans of cue, falling back to brackets for ruby
// annotations.
func renderText(cue model.Cue) string {
	var b strings.Builder
	for _, group := range model.RubyGroups(cue.Spans) {
		b.WriteString(group.Base.Text)
		for _, ann := range group.Annotations {
			b.WriteString("(" + ann.Text + ")")
		}
	}
	return b.String()
}

// recordLosses notes the features of cue that the format cannot carry.
func recordLosses(cue model.Cue, losses *lossNotes) {
	if cue.Karaoke() {
		losses.add("karaoke timing")
	}
	if cue.Layout != nil {
		losses.add("positioning")
	}
	for range cue.Animations {
		losses.add("animation")
	}
	for _, span := range cue.Spans {
		if isTrue(span.Bold) || isTrue(span.Italic) || isTrue(span.Underline) || isTrue(span.Strikeout) ||
			span.Font != nil || span.Size != nil || span.Fore != nil || span.Back != nil ||
			span.OutlineWidth != nil || len(span.Shadows) > 0 || span.Secondary != nil {
			losses.add("inline styling")
		}
		if scaleChanged(span.ScaleX) || scaleChanged(span.ScaleY) {
			losses.add("glyph scale")
		}
		if span.Vertical != nil && span.Vertical.Mode != model.VerticalNone {
			losses.add("vertical text")
		}
		if span.Packed != nil && *span.Packed {
			losses.add("packing")
		}
		if span.Direction != nil && *span.Direction == model.DirRightToLeft {
			losses.add("right-to-left marking")
		}
		if span.Script != nil && span.Script.Kind != model.ScriptRegular {
			losses.add("script offset")
		}
		if span.Ruby != nil {
			losses.add("ruby text")
		}
		if span.Voice != nil {
			losses.add("voice name")
		}
	}
}

// isTrue dereferences an optional bool.
func isTrue(v *bool) bool { return v != nil && *v }

// scaleChanged reports whether a glyph scale override differs from the
// original size. The format carries no scale, so the writer records a loss.
func scaleChanged(v *float64) bool { return v != nil && *v != 100 }
