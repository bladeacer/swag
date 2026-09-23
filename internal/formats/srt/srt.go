// Package srt reads and writes SubRip (.srt) subtitles.
//
// SubRip carries plain text with two inline HTML-like tags: <i> (italic)
// and <b> (bold). Everything else the IR can express degrades on write,
// and the writer records each loss in its report.
package srt

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

// FormatName is the registry name of the format.
const FormatName = "srt"

// The inline tags SubRip understands.
const (
	tagItalicOn  = "<i>"
	tagItalicOff = "</i>"
	tagBoldOn    = "<b>"
	tagBoldOff   = "</b>"
)

// Reader parses a SubRip document.
type Reader struct{}

// NewReader returns a SubRip reader.
func NewReader() *Reader { return &Reader{} }

// Name returns the registry name of the format.
func (r *Reader) Name() string { return FormatName }

// Parse reads a SubRip document from source.
func (r *Reader) Parse(source io.Reader) (*model.Document, error) {
	doc := &model.Document{
		Metadata:        map[string]string{},
		VideoDimensions: model.Point{X: 1280, Y: 720},
		Styles:          []model.Style{DefaultStyle()},
	}
	scanner := bufio.NewScanner(source)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		trimmed := strings.TrimSpace(line)

		switch {
		case trimmed == "":
			// A blank line closes the current cue.
		case isCounter(trimmed):
			// A cue counter line: the next non-blank line is the timing.
		case strings.Contains(trimmed, "-->"):
			start, end, err := parseTiming(trimmed)
			if err != nil {
				return nil, err
			}
			doc.Cues = append(doc.Cues, model.Cue{Start: start, End: end})
		case len(doc.Cues) > 0:
			cue := &doc.Cues[len(doc.Cues)-1]
			if len(cue.Spans) > 0 {
				// A new physical line continues the cue: mark the break on
				// the span that ended the line before this one.
				cue.Spans[len(cue.Spans)-1].Text += "\n"
			}
			cue.Spans = append(cue.Spans, SplitTags(line)...)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read srt: %w", err)
	}
	return doc, nil
}

// isCounter reports whether the line is a decimal cue counter.
func isCounter(line string) bool {
	_, err := strconv.Atoi(line)
	return err == nil
}

// parseTiming parses "00:00:01,000 --> 00:00:04,000".
func parseTiming(line string) (time.Duration, time.Duration, error) {
	arrow := strings.Index(line, "-->")
	if arrow < 0 {
		return 0, 0, fmt.Errorf("parse srt timing %q: missing -->", line)
	}
	start, err := parseTimestamp(strings.TrimSpace(line[:arrow]))
	if err != nil {
		return 0, 0, err
	}
	end, err := parseTimestamp(strings.TrimSpace(line[arrow+3:]))
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}

// parseTimestamp parses hh:mm:ss,mmm; the hours field may carry more than
// two digits.
func parseTimestamp(ts string) (time.Duration, error) {
	ts = strings.ReplaceAll(ts, ",", ".")
	parts := strings.Split(ts, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("parse srt timestamp %q: want hh:mm:ss,mmm", ts)
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, fmt.Errorf("parse srt timestamp %q: bad hours", ts)
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("parse srt timestamp %q: bad minutes", ts)
	}
	sec, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0, fmt.Errorf("parse srt timestamp %q: bad seconds", ts)
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute +
		time.Duration(sec*float64(time.Second)), nil
}

// SplitTags splits one text line into spans on italic and bold toggles.
// A line can carry any number of toggles. Other markup passes through as
// text.
func SplitTags(line string) []model.TextSpan {
	var spans []model.TextSpan
	bold, italic := false, false

	flush := func(text string) {
		if text == "" {
			return
		}
		span := model.TextSpan{Text: text}
		b, i := bold, italic
		if b {
			span.Bold = &b
		}
		if i {
			span.Italic = &i
		}
		spans = append(spans, span)
	}

	for line != "" {
		switch {
		case cutPrefix(&line, tagItalicOn):
			flush("")
			italic = true
		case cutPrefix(&line, tagItalicOff):
			italic = false
		case cutPrefix(&line, tagBoldOn):
			bold = true
		case cutPrefix(&line, tagBoldOff):
			bold = false
		default:
			// No tag at the front: consume up to the next tag, or the rest.
			text := line
			if idx := nextTagIndex(line); idx >= 0 {
				text = line[:idx]
			}
			flush(text)
			if idx := nextTagIndex(line); idx >= 0 {
				line = line[idx:]
			} else {
				line = ""
			}
		}
	}
	return spans
}

// cutPrefix removes prefix from the head of s and reports whether it was
// present.
func cutPrefix(s *string, prefix string) bool {
	if !strings.HasPrefix(*s, prefix) {
		return false
	}
	*s = (*s)[len(prefix):]
	return true
}

// nextTagIndex returns the index of the earliest inline tag in line, or -1.
func nextTagIndex(line string) int {
	best := -1
	for _, tag := range []string{tagItalicOn, tagItalicOff, tagBoldOn, tagBoldOff} {
		if idx := strings.Index(line, tag); idx >= 0 && (best < 0 || idx < best) {
			best = idx
		}
	}
	return best
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
