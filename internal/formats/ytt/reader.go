// SPDX-License-Identifier: Apache-2.0

package ytt

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

// zeroWidthSpace is the padding character YouTube uses between runs. The
// reader drops it so a round trip does not stack padding.
const zeroWidthSpace = "\u200b"

// Reader parses a YouTube Timed Text document.
type Reader struct{}

// NewReader returns a YouTube Timed Text reader.
func NewReader() *Reader { return &Reader{} }

// Name returns the registry name of the format.
func (r *Reader) Name() string { return FormatName }

// Parse reads a YouTube Timed Text document from source.
func (r *Reader) Parse(source io.Reader) (*model.Document, error) {
	var root xmlTimedText
	decoder := xml.NewDecoder(source)
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("parse ytt: %w", err)
	}

	style := DefaultStyle()
	width := float64(defaultWidth)
	height := float64(defaultHeight)

	pens := make(map[int]pen, len(root.Head.Pens))
	for _, xp := range root.Head.Pens {
		p, err := xp.toPen()
		if err != nil {
			return nil, fmt.Errorf("parse ytt: %w", err)
		}
		pens[xp.ID] = p
	}
	styles := make(map[int]xmlWindowStyle, len(root.Head.Styles))
	for _, ws := range root.Head.Styles {
		styles[ws.ID] = ws
	}
	positions := make(map[int]xmlWindowPos, len(root.Head.Positions))
	for _, wp := range root.Head.Positions {
		positions[wp.ID] = wp
	}

	doc := &model.Document{
		Metadata:        map[string]string{},
		Styles:          []model.Style{style},
		VideoDimensions: model.Point{X: width, Y: height},
		Cues:            make([]model.Cue, 0, len(root.Body.Paragraphs)),
	}
	if root.Format != "" {
		doc.Metadata["ytt.format"] = root.Format
	}

	for _, para := range root.Body.Paragraphs {
		cue, err := parseParagraph(para, pens, styles, positions, style.Size, width, height)
		if err != nil {
			return nil, fmt.Errorf("parse ytt: %w", err)
		}
		doc.Cues = append(doc.Cues, cue)
	}
	return doc, nil
}

// parseParagraph turns one <p> element into a cue.
func parseParagraph(
	para xmlParagraph,
	pens map[int]pen,
	styles map[int]xmlWindowStyle,
	positions map[int]xmlWindowPos,
	baseSize, width, height float64,
) (model.Cue, error) {
	startMS, err := attrInt(para.Start, 0)
	if err != nil {
		return model.Cue{}, fmt.Errorf("paragraph start: %w", err)
	}
	durationMS, err := attrInt(para.Duration, 0)
	if err != nil {
		return model.Cue{}, fmt.Errorf("paragraph duration: %w", err)
	}

	cue := model.Cue{
		Start: time.Duration(startMS) * time.Millisecond,
		End:   time.Duration(startMS+durationMS) * time.Millisecond,
	}

	linePen, err := penFor(pens, para.Pen)
	if err != nil {
		return model.Cue{}, err
	}

	if posID, ok, err := attrYTTInt(para.Position); err != nil {
		return model.Cue{}, fmt.Errorf("paragraph position: %w", err)
	} else if ok {
		if wp, found := positions[posID]; found {
			layout, err := wp.toPosition(width, height)
			if err != nil {
				return model.Cue{}, err
			}
			cue.Layout = layout
		}
	}

	var vertical *model.Vertical
	var direction *model.Direction
	if styleID, ok, err := attrYTTInt(para.Style); err != nil {
		return model.Cue{}, fmt.Errorf("paragraph style: %w", err)
	} else if ok {
		if ws, found := styles[styleID]; found {
			vertical, direction, err = parseWindowStyle(ws)
			if err != nil {
				return model.Cue{}, err
			}
		}
	}

	spans, times, err := buildSpans(para, linePen, pens, baseSize)
	if err != nil {
		return model.Cue{}, err
	}
	stripPlatformSpace(spans)
	applyKaraoke(spans, times, cue.Duration())
	for i := range spans {
		if vertical != nil {
			v := *vertical
			spans[i].Vertical = &v
		}
		if direction != nil {
			d := *direction
			spans[i].Direction = &d
		}
	}
	cue.Spans = spans
	return cue, nil
}

// buildSpans walks the runs of a paragraph in order and returns the spans
// with their karaoke times in milliseconds, where -1 means untimed. Text
// outside an <s> element takes the pen of the line. Text inside one takes
// its own pen.
func buildSpans(para xmlParagraph, linePen pen, pens map[int]pen, baseSize float64) ([]model.TextSpan, []int, error) {
	// The run count bounds the span count, so the three slices grow no more
	// than once. The pair of readers shares this path.
	spans := make([]model.TextSpan, 0, len(para.Runs))
	times := make([]int, 0, len(para.Runs))
	explicit := make([]bool, 0, len(para.Runs))

	appendText := func(text string) {
		if n := len(spans); n > 0 && !explicit[n-1] {
			spans[n-1].Text += text
			return
		}
		span := model.TextSpan{Text: text}
		linePen.applyTo(&span, baseSize)
		spans = append(spans, span)
		times = append(times, -1)
		explicit = append(explicit, false)
	}

	for _, run := range para.Runs {
		switch {
		case run.Break:
			if n := len(spans); n > 0 && !explicit[n-1] {
				spans[n-1].Text += "\n"
				continue
			}
			appendText("\n")
		case run.Span != nil:
			// A span without a pen of its own takes the pen of the line.
			pn := linePen
			if strings.TrimSpace(run.Span.Pen) != "" {
				var err error
				if pn, err = penFor(pens, run.Span.Pen); err != nil {
					return nil, nil, err
				}
			}
			if pn.isRubyParenthesis() {
				// Parenthesis text is the mobile fallback for the ruby
				// reading. Dropping it keeps one reading per base.
				continue
			}
			text := strings.ReplaceAll(run.Span.Text, zeroWidthSpace, "")
			if text == "" {
				continue
			}
			t := -1
			if v, ok, err := attrYTTInt(run.Span.Time); err != nil {
				return nil, nil, fmt.Errorf("span time: %w", err)
			} else if ok {
				t = v
			}
			span := model.TextSpan{Text: text}
			pn.applyTo(&span, baseSize)
			spans = append(spans, span)
			times = append(times, t)
			explicit = append(explicit, true)
		default:
			if text := strings.ReplaceAll(run.Text, zeroWidthSpace, ""); text != "" {
				appendText(text)
			}
		}
	}
	return spans, times, nil
}

// stripPlatformSpace removes the one space that the writer steals at the
// line start for an italic or shadowed run. YouTube trims that overhang, so
// the space carries no text of its own.
func stripPlatformSpace(spans []model.TextSpan) {
	if len(spans) == 0 {
		return
	}
	first := &spans[0]
	if !strings.HasPrefix(first.Text, " ") {
		return
	}
	italic := first.Italic != nil && *first.Italic
	if italic || len(first.Shadows) > 0 {
		first.Text = first.Text[1:]
	}
}

// applyKaraoke converts the relative appearance times of a paragraph into
// the span offsets of the IR. Each timed span ends when the next timed span
// appears, and the last one ends with the cue.
func applyKaraoke(spans []model.TextSpan, times []int, duration time.Duration) {
	karaoke := false
	for _, t := range times {
		if t > 0 {
			karaoke = true
			break
		}
	}
	if !karaoke {
		return
	}
	endMS := int(duration / time.Millisecond)
	for i := range spans {
		start := times[i]
		if start < 0 {
			start = 0
		}
		end := endMS
		for j := i + 1; j < len(times); j++ {
			if times[j] >= 0 {
				end = times[j]
				break
			}
		}
		spans[i].Start = time.Duration(start) * time.Millisecond
		spans[i].End = time.Duration(end) * time.Millisecond
	}
}

// penFor looks up a pen by the string id of an attribute. An empty id or an
// unknown id gives the zero pen, which sets no overrides.
func penFor(pens map[int]pen, id string) (pen, error) {
	n, ok, err := attrYTTInt(id)
	if err != nil {
		return pen{}, fmt.Errorf("parse pen id %q: %w", id, err)
	}
	if !ok {
		return pen{}, nil
	}
	return pens[n], nil
}

// parseWindowStyle maps a <ws> element onto the vertical mode and direction
// of the IR.
func parseWindowStyle(ws xmlWindowStyle) (*model.Vertical, *model.Direction, error) {
	pd, err := attrInt(ws.Pitch, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("window style %d pitch: %w", ws.ID, err)
	}
	sd, err := attrInt(ws.Skew, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("window style %d skew: %w", ws.ID, err)
	}
	switch {
	case pd == 1:
		d := model.DirRightToLeft
		return nil, &d, nil
	case pd == 2 && sd == 0:
		v := model.Vertical{Mode: model.VerticalColumnsRTL}
		return &v, nil, nil
	case pd == 2 && sd == 1:
		v := model.Vertical{Mode: model.VerticalColumnsLTR}
		return &v, nil, nil
	case pd == 3 && sd == 0:
		v := model.Vertical{Mode: model.VerticalRotated}
		return &v, nil, nil
	case pd == 3 && sd == 1:
		v := model.Vertical{Mode: model.VerticalRotatedReversed}
		return &v, nil, nil
	}
	return nil, nil, nil
}

// anchorFromAP maps a YouTube anchor point (0 top-left to 8 bottom-right)
// onto the numpad anchor of the IR (7 top-left to 3 bottom-right).
func anchorFromAP(ap int) model.Anchor {
	if ap < 0 || ap > 8 {
		ap = 7
	}
	row, col := ap/3, ap%3
	return model.Anchor(7 - 3*row + col)
}

// apFromAnchor is the inverse of anchorFromAP.
func apFromAnchor(a model.Anchor) int {
	if a < model.AnchorBottomLeft || a > model.AnchorTopRight {
		a = model.AnchorBottomCentre
	}
	v := int(a) - 1
	row, col := 2-v/3, v%3
	return row*3 + col
}
