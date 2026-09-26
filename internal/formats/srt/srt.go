// Package srt reads and writes SubRip (.srt) subtitles.
//
// SubRip carries plain text with two inline HTML-like tags: <i> (italic)
// and <b> (bold). Everything else the IR can express degrades on write,
// and the writer records each loss in its report.
//
// A lossy write also appends the integrity block of internal/envelope, which
// holds the whole document. A plain subtitle player stops at the last cue
// and ignores the block, and a swag reader restores every feature from it.
// A document that fits in plain SubRip writes no block, so a plain file
// stays plain.
package srt

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

// Parse reads a SubRip document from source. It streams the body one line at
// a time, so a large file builds no slice of lines.
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
		return nil, fmt.Errorf("parse srt: %w", err)
	}
	if embedded != nil {
		return embedded, nil
	}
	return doc, nil
}

// parseLine folds one SubRip body line into doc.
func parseLine(doc *model.Document, line string) error {
	trimmed := strings.TrimSpace(line)

	switch {
	case trimmed == "":
		// A blank line closes the current cue.
	case isCounter(trimmed):
		// A cue counter line: the next non-blank line is the timing.
	case strings.Contains(trimmed, "-->"):
		start, end, err := parseTiming(trimmed)
		if err != nil {
			return err
		}
		doc.Cues = append(doc.Cues, model.Cue{Start: start, End: end})
	case len(doc.Cues) > 0:
		cue := &doc.Cues[len(doc.Cues)-1]
		if len(cue.Spans) > 0 {
			// A new physical line continues the cue: mark the break on
			// the span that ended the line before this one.
			cue.Spans[len(cue.Spans)-1].Text += "\n"
		}
		cue.Spans = appendSpans(cue.Spans, line)
	}
	return nil
}

// isCounter reports whether the line is a decimal cue counter. It rejects a
// line that carries a non-digit before it calls the parser, because the
// parser builds an error for every failed parse and most lines are text.
func isCounter(line string) bool {
	for i := 0; i < len(line); i++ {
		if line[i] < '0' || line[i] > '9' {
			return false
		}
	}
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

// parseTimestamp parses hh:mm:ss,mmm. The hours field can carry more than
// two digits. It finds the two colons instead of splitting the string, so a
// parse builds no slice for the parts.
func parseTimestamp(ts string) (time.Duration, error) {
	ts = strings.ReplaceAll(ts, ",", ".")
	if strings.Count(ts, ":") != 2 {
		return 0, fmt.Errorf("parse srt timestamp %q: want hh:mm:ss,mmm", ts)
	}
	first := strings.Index(ts, ":")
	second := first + 1 + strings.Index(ts[first+1:], ":")
	h, err := strconv.Atoi(strings.TrimSpace(ts[:first]))
	if err != nil {
		return 0, fmt.Errorf("parse srt timestamp %q: bad hours", ts)
	}
	m, err := strconv.Atoi(ts[first+1 : second])
	if err != nil {
		return 0, fmt.Errorf("parse srt timestamp %q: bad minutes", ts)
	}
	sec, err := strconv.ParseFloat(ts[second+1:], 64)
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
	return appendSpans(nil, line)
}

// appendSpans splits one text line into spans on italic and bold toggles and
// appends them to dst. A reader passes the span slice of the cue, so the
// call builds no intermediate slice for every line.
func appendSpans(dst []model.TextSpan, line string) []model.TextSpan {
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
		dst = append(dst, span)
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
	return dst
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

// inlineTags lists the SubRip inline tags in one place, so the scan reuses
// the slice rather than building a fresh one for every call.
var inlineTags = []string{tagItalicOn, tagItalicOff, tagBoldOn, tagBoldOff}

// nextTagIndex returns the index of the earliest inline tag in line, or -1.
func nextTagIndex(line string) int {
	best := -1
	for _, tag := range inlineTags {
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
