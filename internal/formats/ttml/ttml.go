// SPDX-License-Identifier: Apache-2.0

// Package ttml reads the YouTube TTML dialect and writes general TTML.
//
// TTML is XML. A <style> element in the head holds named styling, a
// <region> element holds a placement, and the body holds <p> lines with
// inline <span> runs. The YouTube dialect keeps the paragraph timing in the
// begin and end attributes and the karaoke timing in the begin attribute of
// a run.
//
// The reader maps a named style onto the document styles and applies it to
// the runs of the paragraph, a region onto the cue anchor and position, and
// a run attribute onto a span override. The writer emits the document
// styles and one region per cue placement, then renders the spans with
// their inline attributes. It reports the features TTML cannot express.
package ttml

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

// FormatName is the registry name of the format.
const FormatName = "ttml"

// The frame rate for the frame-count timestamp form.
const frameRate = 30

// The play resolution a percentage placement refers to.
const (
	defaultWidth  = 1280
	defaultHeight = 720
)

// DefaultStyle returns the implicit style of a TTML document.
func DefaultStyle() model.Style {
	return model.Style{
		Name:      "Default",
		Font:      "Arial",
		Size:      20,
		Primary:   model.NewColour(255, 255, 255, 255),
		Secondary: model.NewColour(255, 255, 255, 255),
		Outline:   model.NewColour(0, 0, 0, 255),
		Alignment: model.AnchorBottomCentre,
	}
}

// Reader parses a TTML document.
type Reader struct{}

// NewReader returns a TTML reader.
func NewReader() *Reader { return &Reader{} }

// Name returns the registry name of the format.
func (r *Reader) Name() string { return FormatName }

// styleAttrs holds the raw styling attributes of a style, a paragraph, or a
// run. An empty string means "not set here", so a merge can layer them.
type styleAttrs struct {
	Colour         string
	Background     string
	FontFamily     string
	FontSize       string
	FontWeight     string
	FontStyle      string
	TextDecoration string
	TextOutline    string
	Opacity        string
	// Style names another style, which this one extends.
	Style string
}

// overlay returns a copy of a with the set fields of b on top.
func (a styleAttrs) overlay(b styleAttrs) styleAttrs {
	if b.Colour != "" {
		a.Colour = b.Colour
	}
	if b.Background != "" {
		a.Background = b.Background
	}
	if b.FontFamily != "" {
		a.FontFamily = b.FontFamily
	}
	if b.FontSize != "" {
		a.FontSize = b.FontSize
	}
	if b.FontWeight != "" {
		a.FontWeight = b.FontWeight
	}
	if b.FontStyle != "" {
		a.FontStyle = b.FontStyle
	}
	if b.TextDecoration != "" {
		a.TextDecoration = b.TextDecoration
	}
	if b.TextOutline != "" {
		a.TextOutline = b.TextOutline
	}
	if b.Opacity != "" {
		a.Opacity = b.Opacity
	}
	return a
}

// resolveStyle follows the style references of an element and returns the
// merged attributes. The element's own attributes win over the styles it
// extends. A reference cycle is an error.
func resolveStyle(styles map[string]styleAttrs, own styleAttrs) (styleAttrs, error) {
	var chain []styleAttrs
	seen := map[string]bool{}
	ref := own.Style
	for ref != "" {
		if seen[ref] {
			return styleAttrs{}, fmt.Errorf("style cycle at %q", ref)
		}
		seen[ref] = true
		s, ok := styles[ref]
		if !ok {
			return styleAttrs{}, fmt.Errorf("unknown style %q", ref)
		}
		chain = append(chain, s)
		ref = s.Style
	}
	out := styleAttrs{}
	for i := len(chain) - 1; i >= 0; i-- {
		out = out.overlay(chain[i])
	}
	return out.overlay(own), nil
}

// toStyle converts the attributes into a document style.
func (a styleAttrs) toStyle(name string) (model.Style, error) {
	s := DefaultStyle()
	s.Name = name
	if a.Colour != "" {
		c, err := parseColour(a.Colour)
		if err != nil {
			return model.Style{}, err
		}
		s.Primary = c
	}
	if a.Background != "" {
		c, err := parseColour(a.Background)
		if err != nil {
			return model.Style{}, err
		}
		s.Box = true
		s.Outline = c
	}
	if a.FontFamily != "" {
		s.Font = a.FontFamily
	}
	if a.FontSize != "" {
		size, err := parseLength(a.FontSize)
		if err != nil {
			return model.Style{}, err
		}
		s.Size = size
	}
	if a.FontWeight != "" {
		bold, err := parseBold(a.FontWeight)
		if err != nil {
			return model.Style{}, err
		}
		s.Bold = bold
	}
	if a.FontStyle != "" {
		italic, err := parseItalic(a.FontStyle)
		if err != nil {
			return model.Style{}, err
		}
		s.Italic = italic
	}
	if a.TextDecoration != "" {
		underline, _, err := parseDecoration(a.TextDecoration)
		if err != nil {
			return model.Style{}, err
		}
		s.Underline = underline
	}
	if a.TextOutline != "" {
		width, colour, err := parseOutline(a.TextOutline)
		if err != nil {
			return model.Style{}, err
		}
		s.OutlineWidth = width
		if colour != nil {
			s.Outline = *colour
		}
	}
	if a.Opacity != "" {
		alpha, err := parseOpacity(a.Opacity)
		if err != nil {
			return model.Style{}, err
		}
		s.Primary = s.Primary.WithAlpha(alpha)
	}
	return s, nil
}

// applySpan sets the span overrides that the attributes carry.
func (a styleAttrs) applySpan(span *model.TextSpan) error {
	if a.Colour != "" {
		c, err := parseColour(a.Colour)
		if err != nil {
			return err
		}
		span.Fore = &c
	}
	if a.Background != "" {
		c, err := parseColour(a.Background)
		if err != nil {
			return err
		}
		span.Back = &c
	}
	if a.FontFamily != "" {
		font := a.FontFamily
		span.Font = &font
	}
	if a.FontSize != "" {
		size, err := parseLength(a.FontSize)
		if err != nil {
			return err
		}
		span.Size = &size
	}
	if a.FontWeight != "" {
		bold, err := parseBold(a.FontWeight)
		if err != nil {
			return err
		}
		span.Bold = &bold
	}
	if a.FontStyle != "" {
		italic, err := parseItalic(a.FontStyle)
		if err != nil {
			return err
		}
		span.Italic = &italic
	}
	if a.TextDecoration != "" {
		underline, strikeout, err := parseDecoration(a.TextDecoration)
		if err != nil {
			return err
		}
		span.Underline = &underline
		span.Strikeout = &strikeout
	}
	if a.TextOutline != "" {
		width, _, err := parseOutline(a.TextOutline)
		if err != nil {
			return err
		}
		span.OutlineWidth = &width
	}
	if a.Opacity != "" {
		alpha, err := parseOpacity(a.Opacity)
		if err != nil {
			return err
		}
		fore := model.Colour{A: alpha}
		if span.Fore != nil {
			fore = span.Fore.WithAlpha(alpha)
		}
		span.Fore = &fore
	}
	return nil
}

// xmlTT is the root element of a TTML document.
type xmlTT struct {
	XMLName xml.Name `xml:"tt"`
	Head    xmlHead  `xml:"head"`
	Body    xmlBody  `xml:"body"`
}

// xmlHead holds the named styles and the regions.
type xmlHead struct {
	Styles  []xmlStyle  `xml:"styling>style"`
	Regions []xmlRegion `xml:"layout>region"`
}

// xmlStyle is one <style> element.
type xmlStyle struct {
	ID string `xml:"id,attr"`
	styleAttrFields
}

// styleAttrFields carries the styling attributes in their raw string form.
type styleAttrFields struct {
	Style          string `xml:"style,attr"`
	Colour         string `xml:"color,attr"`
	Background     string `xml:"backgroundColor,attr"`
	FontFamily     string `xml:"fontFamily,attr"`
	FontSize       string `xml:"fontSize,attr"`
	FontWeight     string `xml:"fontWeight,attr"`
	FontStyle      string `xml:"fontStyle,attr"`
	TextDecoration string `xml:"textDecoration,attr"`
	TextOutline    string `xml:"textOutline,attr"`
	Opacity        string `xml:"opacity,attr"`
}

// attrs returns the style attributes of the element.
func (x xmlStyle) attrs() styleAttrs {
	return styleAttrs{
		Style:          x.Style,
		Colour:         x.Colour,
		Background:     x.Background,
		FontFamily:     x.FontFamily,
		FontSize:       x.FontSize,
		FontWeight:     x.FontWeight,
		FontStyle:      x.FontStyle,
		TextDecoration: x.TextDecoration,
		TextOutline:    x.TextOutline,
		Opacity:        x.Opacity,
	}
}

// xmlRegion is one <region> element.
type xmlRegion struct {
	ID           string `xml:"id,attr"`
	Origin       string `xml:"origin,attr"`
	DisplayAlign string `xml:"displayAlign,attr"`
	TextAlign    string `xml:"textAlign,attr"`
}

// xmlBody holds the lines of a document.
type xmlBody struct {
	Paragraphs []xmlParagraph `xml:"div>p"`
}

// xmlParagraph is one <p> element. Its content is mixed: plain text, <span>
// runs, and line breaks, so it carries its own unmarshaller.
type xmlParagraph struct {
	Begin  string
	End    string
	Dur    string
	Region string
	// Inline carries the styling attributes of the paragraph itself.
	Inline styleAttrs
	Runs   []xmlRun
}

// xmlRun is one item in the content of a <p> element.
type xmlRun struct {
	Text  string
	Break bool
	Span  *xmlSpan
}

// xmlSpan is one <span> element with its own styling and timing.
type xmlSpan struct {
	Begin string
	Attrs styleAttrs
	Text  string
}

// UnmarshalXML reads a <p> element and keeps the order of its text runs,
// spans, and line breaks.
func (p *xmlParagraph) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		switch a.Name.Local {
		case "begin":
			p.Begin = a.Value
		case "end":
			p.End = a.Value
		case "dur":
			p.Dur = a.Value
		case "region":
			p.Region = a.Value
		}
	}
	p.Inline = styleAttrsFromAttrs(start.Attr)
	for {
		tok, err := d.Token()
		if err != nil {
			return fmt.Errorf("parse paragraph: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "span":
				span := &xmlSpan{}
				if err := d.DecodeElement(span, &t); err != nil {
					return fmt.Errorf("parse span: %w", err)
				}
				p.Runs = append(p.Runs, xmlRun{Span: span})
			case "br":
				p.Runs = append(p.Runs, xmlRun{Break: true})
				if err := d.Skip(); err != nil {
					return fmt.Errorf("parse break: %w", err)
				}
			default:
				if err := d.Skip(); err != nil {
					return fmt.Errorf("parse paragraph: %w", err)
				}
			}
		case xml.CharData:
			if len(t) > 0 {
				p.Runs = append(p.Runs, xmlRun{Text: string(t)})
			}
		case xml.EndElement:
			if t.Name == start.Name {
				return nil
			}
		}
	}
}

// UnmarshalXML reads a <span> element with its attributes and its text.
func (s *xmlSpan) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	s.Attrs = styleAttrsFromAttrs(start.Attr)
	for _, a := range start.Attr {
		if a.Name.Local == "begin" {
			s.Begin = a.Value
		}
	}
	var text strings.Builder
	for {
		tok, err := d.Token()
		if err != nil {
			return fmt.Errorf("parse span: %w", err)
		}
		switch t := tok.(type) {
		case xml.CharData:
			text.Write(t)
		case xml.StartElement:
			// A nested element carries no styling of its own in the
			// YouTube dialect, so the reader keeps its text and drops the
			// element.
		case xml.EndElement:
			if t.Name == start.Name {
				s.Text = text.String()
				return nil
			}
		}
	}
}

// styleAttrsFromAttrs reads the styling attributes of an element. It matches
// on the local name, so the tts prefix does not matter.
func styleAttrsFromAttrs(attrs []xml.Attr) styleAttrs {
	var a styleAttrs
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "color":
			a.Colour = attr.Value
		case "backgroundColor":
			a.Background = attr.Value
		case "fontFamily":
			a.FontFamily = attr.Value
		case "fontSize":
			a.FontSize = attr.Value
		case "fontWeight":
			a.FontWeight = attr.Value
		case "fontStyle":
			a.FontStyle = attr.Value
		case "textDecoration":
			a.TextDecoration = attr.Value
		case "textOutline":
			a.TextOutline = attr.Value
		case "opacity":
			a.Opacity = attr.Value
		case "style":
			a.Style = attr.Value
		}
	}
	return a
}

// Parse reads a TTML document from source.
func (r *Reader) Parse(source io.Reader) (*model.Document, error) {
	var root xmlTT
	if err := xml.NewDecoder(source).Decode(&root); err != nil {
		return nil, fmt.Errorf("parse ttml: %w", err)
	}

	styles := make(map[string]styleAttrs, len(root.Head.Styles))
	for _, s := range root.Head.Styles {
		styles[s.ID] = s.attrs()
	}

	doc := &model.Document{
		Metadata:        map[string]string{},
		VideoDimensions: model.Point{X: defaultWidth, Y: defaultHeight},
		Styles:          []model.Style{DefaultStyle()},
	}
	for _, s := range root.Head.Styles {
		resolved, err := resolveStyle(styles, s.attrs())
		if err != nil {
			return nil, fmt.Errorf("parse ttml style %q: %w", s.ID, err)
		}
		style, err := resolved.toStyle(s.ID)
		if err != nil {
			return nil, fmt.Errorf("parse ttml style %q: %w", s.ID, err)
		}
		doc.Styles = append(doc.Styles, style)
	}

	for _, para := range root.Body.Paragraphs {
		cue, err := parseParagraph(para, styles, root.Head.Regions)
		if err != nil {
			return nil, fmt.Errorf("parse ttml: %w", err)
		}
		doc.Cues = append(doc.Cues, cue)
	}
	return doc, nil
}

// parseParagraph turns one <p> element into a cue.
func parseParagraph(para xmlParagraph, styles map[string]styleAttrs, regions []xmlRegion) (model.Cue, error) {
	start, err := parseTime(para.Begin)
	if err != nil {
		return model.Cue{}, fmt.Errorf("paragraph begin: %w", err)
	}
	end, err := paragraphEnd(para, start)
	if err != nil {
		return model.Cue{}, err
	}

	base, err := resolveStyle(styles, para.Inline)
	if err != nil {
		return model.Cue{}, err
	}
	for _, region := range regions {
		if region.ID != "" && region.ID == para.Region {
			cue := model.Cue{Start: start, End: end}
			layout, err := region.toLayout(defaultWidth, defaultHeight)
			if err != nil {
				return model.Cue{}, err
			}
			cue.Layout = layout
			return buildCue(para, cue, base, styles)
		}
	}
	return buildCue(para, model.Cue{Start: start, End: end}, base, styles)
}

// paragraphEnd reads the end time from the end attribute, or from the dur
// attribute when end is absent.
func paragraphEnd(para xmlParagraph, start time.Duration) (time.Duration, error) {
	if para.End != "" {
		end, err := parseTime(para.End)
		if err != nil {
			return 0, fmt.Errorf("paragraph end: %w", err)
		}
		return end, nil
	}
	if para.Dur != "" {
		dur, err := parseTime(para.Dur)
		if err != nil {
			return 0, fmt.Errorf("paragraph duration: %w", err)
		}
		return start + dur, nil
	}
	return start, nil
}

// buildCue walks the runs of a paragraph into the spans of cue.
func buildCue(para xmlParagraph, cue model.Cue, base styleAttrs, styles map[string]styleAttrs) (model.Cue, error) {
	var times []time.Duration
	var explicit []bool

	appendText := func(text string) error {
		if n := len(cue.Spans); n > 0 && !explicit[n-1] {
			cue.Spans[n-1].Text += text
			return nil
		}
		span := model.TextSpan{Text: text}
		if err := base.applySpan(&span); err != nil {
			return err
		}
		cue.Spans = append(cue.Spans, span)
		times = append(times, -1)
		explicit = append(explicit, false)
		return nil
	}

	for _, run := range para.Runs {
		switch {
		case run.Break:
			if err := appendText("\n"); err != nil {
				return model.Cue{}, err
			}
		case run.Span != nil:
			resolved, err := resolveStyle(styles, run.Span.Attrs)
			if err != nil {
				return model.Cue{}, err
			}
			span := model.TextSpan{Text: run.Span.Text}
			if err := base.overlay(resolved).applySpan(&span); err != nil {
				return model.Cue{}, err
			}
			begin := time.Duration(-1)
			if run.Span.Begin != "" {
				if begin, err = parseTime(run.Span.Begin); err != nil {
					return model.Cue{}, fmt.Errorf("span begin: %w", err)
				}
			}
			cue.Spans = append(cue.Spans, span)
			times = append(times, begin)
			explicit = append(explicit, true)
		case run.Text != "":
			if err := appendText(run.Text); err != nil {
				return model.Cue{}, err
			}
		}
	}
	applyKaraoke(cue.Spans, times, cue.Duration())
	return cue, nil
}

// applyKaraoke converts the relative begin times of a paragraph into the span
// offsets of the IR. Each timed span ends when the next timed span begins,
// and the last one ends with the cue.
func applyKaraoke(spans []model.TextSpan, times []time.Duration, duration time.Duration) {
	timed := false
	for _, t := range times {
		if t >= 0 {
			timed = true
			break
		}
	}
	if !timed {
		return
	}
	for i := range spans {
		start := times[i]
		if start < 0 {
			start = 0
		}
		end := duration
		for j := i + 1; j < len(times); j++ {
			if times[j] >= 0 {
				end = times[j]
				break
			}
		}
		spans[i].Start = start
		spans[i].End = end
	}
}

// toLayout converts a region into a cue layout.
func (reg xmlRegion) toLayout(width, height float64) (*model.Layout, error) {
	layout := &model.Layout{}
	if reg.Origin != "" {
		x, y, err := parseOrigin(reg.Origin)
		if err != nil {
			return nil, err
		}
		pos := model.Point{X: x * width, Y: y * height}
		layout.Position = &pos
	}
	if reg.DisplayAlign != "" || reg.TextAlign != "" {
		anchor, err := parseAnchor(reg.DisplayAlign, reg.TextAlign)
		if err != nil {
			return nil, err
		}
		layout.Anchor = &anchor
	}
	return layout, nil
}

// parseAnchor maps displayAlign and textAlign onto a numpad anchor. A
// missing value takes the TTML default: bottom row, centre column.
func parseAnchor(displayAlign, textAlign string) (model.Anchor, error) {
	var row int
	switch displayAlign {
	case "", "after":
		row = 0
	case "center":
		row = 1
	case "before":
		row = 2
	default:
		return 0, fmt.Errorf("parse displayAlign %q: want before, center, or after", displayAlign)
	}
	var col int
	switch textAlign {
	case "", "center":
		col = 1
	case "start", "left":
		col = 0
	case "end", "right":
		col = 2
	default:
		return 0, fmt.Errorf("parse textAlign %q: want start, center, or end", textAlign)
	}
	return model.Anchor(1 + 3*row + col), nil
}

// parseOrigin reads an origin such as "10% 90%".
func parseOrigin(s string) (float64, float64, error) {
	fields := strings.Fields(s)
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("parse origin %q: want two percentages", s)
	}
	x, err := parsePercent(fields[0])
	if err != nil {
		return 0, 0, err
	}
	y, err := parsePercent(fields[1])
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

// parsePercent reads a percentage as a fraction.
func parsePercent(s string) (float64, error) {
	if !strings.HasSuffix(s, "%") {
		return 0, fmt.Errorf("parse percentage %q: missing %% sign", s)
	}
	v, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
	if err != nil {
		return 0, fmt.Errorf("parse percentage %q: %w", s, err)
	}
	return v / 100, nil
}

// parseTime reads a TTML time expression. It accepts the clock forms
// HH:MM:SS, HH:MM:SS.mmm, HH:MM:SS:FF, and MM:SS, plus the offset forms
// with an h, m, s, ms, or f unit.
func parseTime(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("parse time: empty value")
	}
	if strings.Contains(s, ":") {
		return parseClock(s)
	}
	return parseOffset(s)
}

// parseClock reads a clock form. The last field of the four-field form
// carries frames, and the last field of every other form carries the
// fractional seconds.
func parseClock(s string) (time.Duration, error) {
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 4 {
		return 0, fmt.Errorf("parse time %q: want two to four clock fields", s)
	}
	units := []time.Duration{time.Hour, time.Minute, time.Second}
	if len(parts) == 2 {
		units = []time.Duration{time.Minute, time.Second}
	}
	var total time.Duration
	for i, part := range parts {
		v, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return 0, fmt.Errorf("parse time %q: %w", s, err)
		}
		if i == len(parts)-1 && len(parts) == 4 {
			total += time.Duration(v / frameRate * float64(time.Second))
			continue
		}
		total += time.Duration(v * float64(units[i]))
	}
	return total, nil
}

// parseOffset reads a time with a unit suffix, for example "1.5s" or
// "100ms".
func parseOffset(s string) (time.Duration, error) {
	unit := ""
	switch {
	case strings.HasSuffix(s, "ms"):
		unit = "ms"
	case strings.HasSuffix(s, "h"):
		unit = "h"
	case strings.HasSuffix(s, "m"):
		unit = "m"
	case strings.HasSuffix(s, "s"):
		unit = "s"
	case strings.HasSuffix(s, "f"):
		unit = "f"
	default:
		return 0, fmt.Errorf("parse time %q: missing unit", s)
	}
	v, err := strconv.ParseFloat(strings.TrimSuffix(s, unit), 64)
	if err != nil {
		return 0, fmt.Errorf("parse time %q: %w", s, err)
	}
	switch unit {
	case "ms":
		return time.Duration(v * float64(time.Millisecond)), nil
	case "h":
		return time.Duration(v * float64(time.Hour)), nil
	case "m":
		return time.Duration(v * float64(time.Minute)), nil
	case "f":
		return time.Duration(v / frameRate * float64(time.Second)), nil
	default:
		return time.Duration(v * float64(time.Second)), nil
	}
}

// parseLength reads a length such as "20" or "20px".
func parseLength(s string) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(s), "px"), 64)
	if err != nil {
		return 0, fmt.Errorf("parse length %q: %w", s, err)
	}
	return v, nil
}

// parseColour reads a colour of the form #RRGGBB or #RRGGBBAA.
func parseColour(s string) (model.Colour, error) {
	hex := strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(hex) != 6 && len(hex) != 8 {
		return model.Colour{}, fmt.Errorf("parse colour %q: want #RRGGBB or #RRGGBBAA", s)
	}
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return model.Colour{}, fmt.Errorf("parse colour %q: %w", s, err)
	}
	c := model.Colour{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}
	if len(hex) == 8 {
		c.A = uint8(v)
		c.R = uint8(v >> 24)
		c.G = uint8(v >> 16)
		c.B = uint8(v >> 8)
	}
	return c, nil
}

// parseBold reads a fontWeight value.
func parseBold(s string) (bool, error) {
	switch strings.TrimSpace(s) {
	case "bold":
		return true, nil
	case "normal":
		return false, nil
	default:
		return false, fmt.Errorf("parse fontWeight %q: want bold or normal", s)
	}
}

// parseItalic reads a fontStyle value.
func parseItalic(s string) (bool, error) {
	switch strings.TrimSpace(s) {
	case "italic", "oblique":
		return true, nil
	case "normal":
		return false, nil
	default:
		return false, fmt.Errorf("parse fontStyle %q: want italic or normal", s)
	}
}

// parseDecoration reads a textDecoration value and reports underline and
// strikeout.
func parseDecoration(s string) (bool, bool, error) {
	underline, strikeout := false, false
	for _, field := range strings.Fields(s) {
		switch field {
		case "underline":
			underline = true
		case "lineThrough":
			strikeout = true
		case "none", "noUnderline", "noLineThrough":
		default:
			return false, false, fmt.Errorf("parse textDecoration %q: unknown value %q", s, field)
		}
	}
	return underline, strikeout, nil
}

// parseOutline reads a textOutline value such as "2px #000000" or
// "#000000 2px". The colour is optional.
func parseOutline(s string) (float64, *model.Colour, error) {
	var width float64
	var colour *model.Colour
	for _, field := range strings.Fields(s) {
		if strings.HasPrefix(field, "#") {
			c, err := parseColour(field)
			if err != nil {
				return 0, nil, err
			}
			colour = &c
			continue
		}
		w, err := parseLength(field)
		if err != nil {
			return 0, nil, err
		}
		width = w
	}
	return width, colour, nil
}

// parseOpacity reads an opacity from 0 to 1 and returns the alpha channel.
func parseOpacity(s string) (uint8, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, fmt.Errorf("parse opacity %q: %w", s, err)
	}
	if v < 0 || v > 1 {
		return 0, fmt.Errorf("parse opacity %q: want 0 to 1", s)
	}
	return uint8(math.Round(v * 255)), nil
}

// lossNotes collects the degradations of one write.
type lossNotes struct{ notes []string }

func (l *lossNotes) add(format string, args ...any) {
	l.notes = append(l.notes, fmt.Sprintf(format, args...))
}

// Writer renders TTML documents.
type Writer struct{}

// NewWriter returns a TTML writer.
func NewWriter() *Writer { return &Writer{} }

// Name returns the registry name of the format.
func (w *Writer) Name() string { return FormatName }

// Render writes doc to sink as TTML. The returned slice carries one entry
// per feature the format cannot express.
func (w *Writer) Render(doc *model.Document, sink io.Writer) ([]string, error) {
	var losses lossNotes
	width := doc.VideoDimensions.X
	if width <= 0 {
		width = defaultWidth
	}
	height := doc.VideoDimensions.Y
	if height <= 0 {
		height = defaultHeight
	}

	var out strings.Builder
	out.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")
	out.WriteString("<tt xmlns=\"http://www.w3.org/ns/ttml\" xmlns:tts=\"http://www.w3.org/ns/ttml#styling\">\n")
	out.WriteString("  <head>\n")
	out.WriteString("    <styling>\n")
	for i, style := range doc.Styles {
		out.WriteString("      ")
		renderStyle(&out, fmt.Sprintf("style%d", i), style)
		out.WriteString("\n")
	}
	out.WriteString("    </styling>\n")
	out.WriteString("    <layout>\n")
	for i := range doc.Cues {
		if doc.Cues[i].Layout == nil {
			continue
		}
		out.WriteString("      ")
		if err := renderRegion(&out, fmt.Sprintf("region%d", i), doc.Cues[i].Layout, width, height); err != nil {
			return losses.notes, err
		}
		out.WriteString("\n")
	}
	out.WriteString("    </layout>\n")
	out.WriteString("  </head>\n")
	out.WriteString("  <body>\n")
	out.WriteString("    <div>\n")
	for i := range doc.Cues {
		cue := doc.Cues[i]
		recordLosses(cue, &losses)
		fmt.Fprintf(&out, "      <p begin=\"%s\" end=\"%s\"", formatTime(cue.Start), formatTime(cue.End))
		if len(doc.Styles) > 0 {
			out.WriteString(" style=\"style0\"")
		}
		if cue.Layout != nil {
			fmt.Fprintf(&out, " region=\"region%d\"", i)
		}
		out.WriteString(">")
		renderSpans(&out, cue)
		out.WriteString("</p>\n")
	}
	out.WriteString("    </div>\n")
	out.WriteString("  </body>\n")
	out.WriteString("</tt>\n")

	if _, err := io.WriteString(sink, out.String()); err != nil {
		return losses.notes, fmt.Errorf("write ttml: %w", err)
	}
	return losses.notes, nil
}

// renderStyle writes one <style> element.
func renderStyle(out *strings.Builder, id string, style model.Style) {
	fmt.Fprintf(out, "<style xml:id=%q", id)
	if style.Font != "" {
		fmt.Fprintf(out, " tts:fontFamily=%q", style.Font)
	}
	if style.Size != 0 {
		fmt.Fprintf(out, " tts:fontSize=%q", formatLength(style.Size))
	}
	if style.Bold {
		out.WriteString(` tts:fontWeight="bold"`)
	}
	if style.Italic {
		out.WriteString(` tts:fontStyle="italic"`)
	}
	if style.Underline {
		out.WriteString(` tts:textDecoration="underline"`)
	}
	fmt.Fprintf(out, " tts:color=%q", formatColour(style.Primary))
	if !style.Primary.Opaque() {
		fmt.Fprintf(out, " tts:opacity=%q", formatOpacity(style.Primary.A))
	}
	if style.Box {
		fmt.Fprintf(out, " tts:backgroundColor=%q", formatColour(style.Outline))
	} else if style.OutlineWidth > 0 {
		fmt.Fprintf(out, " tts:textOutline=%q", formatOutline(style.OutlineWidth, style.Outline))
	}
	out.WriteString("/>")
}

// renderRegion writes one <region> element.
func renderRegion(out *strings.Builder, id string, layout *model.Layout, width, height float64) error {
	displayAlign, textAlign, err := alignOf(layout.Anchor)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "<region xml:id=%q", id)
	if layout.Position != nil {
		fmt.Fprintf(out, " tts:origin=%q", formatOrigin(*layout.Position, width, height))
	}
	if displayAlign != "" {
		fmt.Fprintf(out, " tts:displayAlign=%q", displayAlign)
	}
	if textAlign != "" {
		fmt.Fprintf(out, " tts:textAlign=%q", textAlign)
	}
	out.WriteString("/>")
	return nil
}

// alignOf maps a numpad anchor onto the displayAlign and the textAlign
// values of a region.
func alignOf(anchor *model.Anchor) (string, string, error) {
	if anchor == nil {
		return "", "", nil
	}
	switch *anchor {
	case model.AnchorBottomLeft:
		return "after", "start", nil
	case model.AnchorBottomCentre:
		return "after", "center", nil
	case model.AnchorBottomRight:
		return "after", "end", nil
	case model.AnchorMiddleLeft:
		return "center", "start", nil
	case model.AnchorCentre:
		return "center", "center", nil
	case model.AnchorMiddleRight:
		return "center", "end", nil
	case model.AnchorTopLeft:
		return "before", "start", nil
	case model.AnchorTopCentre:
		return "before", "center", nil
	case model.AnchorTopRight:
		return "before", "end", nil
	}
	return "", "", fmt.Errorf("write ttml: unknown anchor %d", *anchor)
}

// recordLosses notes the features of cue that TTML cannot carry.
func recordLosses(cue model.Cue, losses *lossNotes) {
	if cue.Layout != nil {
		if cue.Layout.Fade != nil {
			losses.add("cue fade")
		}
		if cue.Layout.Move != nil {
			losses.add("cue move")
		}
	}
	for range cue.Animations {
		losses.add("animation")
	}
	for _, span := range cue.Spans {
		if scaleChanged(span.ScaleX) || scaleChanged(span.ScaleY) {
			losses.add("glyph scale")
		}
		if len(span.Shadows) > 0 {
			losses.add("shadow effects")
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
	}
}

// renderSpans writes the spans of a cue with their inline attributes. A ruby
// annotation falls back to brackets, because general TTML carries no ruby
// form.
func renderSpans(out *strings.Builder, cue model.Cue) {
	for _, group := range model.RubyGroups(cue.Spans) {
		renderSpan(out, group.Base)
		for _, ann := range group.Annotations {
			fmt.Fprintf(out, "(%s)", escapeText(ann.Text))
		}
	}
}

// renderSpan writes one span element with its inline attributes.
func renderSpan(out *strings.Builder, span model.TextSpan) {
	if span.Bold == nil && span.Italic == nil && span.Underline == nil && span.Strikeout == nil &&
		span.Fore == nil && span.Back == nil && span.Font == nil && span.Size == nil &&
		span.OutlineWidth == nil && (span.Start == 0 && span.End == 0) {
		out.WriteString(escapeText(span.Text))
		return
	}
	out.WriteString("<span")
	if span.Start != 0 {
		fmt.Fprintf(out, " begin=%q", formatTime(span.Start))
	}
	if isTrue(span.Bold) {
		out.WriteString(` tts:fontWeight="bold"`)
	}
	if isTrue(span.Italic) {
		out.WriteString(` tts:fontStyle="italic"`)
	}
	if decoration := decorationOf(span); decoration != "" {
		fmt.Fprintf(out, " tts:textDecoration=%q", decoration)
	}
	if span.Font != nil {
		fmt.Fprintf(out, " tts:fontFamily=%q", *span.Font)
	}
	if span.Size != nil {
		fmt.Fprintf(out, " tts:fontSize=%q", formatLength(*span.Size))
	}
	if span.Fore != nil {
		fmt.Fprintf(out, " tts:color=%q", formatColour(*span.Fore))
		if !span.Fore.Opaque() {
			fmt.Fprintf(out, " tts:opacity=%q", formatOpacity(span.Fore.A))
		}
	}
	if span.Back != nil {
		fmt.Fprintf(out, " tts:backgroundColor=%q", formatColour(*span.Back))
	}
	if span.OutlineWidth != nil {
		fmt.Fprintf(out, " tts:textOutline=%q", formatOutline(*span.OutlineWidth, model.Colour{}))
	}
	out.WriteString(">")
	out.WriteString(escapeText(span.Text))
	out.WriteString("</span>")
}

// decorationOf builds the textDecoration value of a span.
func decorationOf(span model.TextSpan) string {
	var parts []string
	if isTrue(span.Underline) {
		parts = append(parts, "underline")
	}
	if isTrue(span.Strikeout) {
		parts = append(parts, "lineThrough")
	}
	return strings.Join(parts, " ")
}

// escapeText escapes the characters that carry meaning in XML.
func escapeText(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// formatColour renders a colour as #RRGGBB.
func formatColour(c model.Colour) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// formatOpacity renders an alpha channel as a fraction from 0 to 1. Three
// decimal places keep the round trip of every channel value exact.
func formatOpacity(a uint8) string {
	return strconv.FormatFloat(float64(a)/255, 'f', 3, 64)
}

// formatLength renders a length in pixels.
func formatLength(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64) + "px"
}

// formatOutline renders an outline width with an optional colour.
func formatOutline(width float64, colour model.Colour) string {
	if colour == (model.Colour{}) {
		return formatLength(width)
	}
	return formatLength(width) + " " + formatColour(colour)
}

// formatOrigin renders a position as two percentages.
func formatOrigin(pos model.Point, width, height float64) string {
	return strconv.FormatFloat(pos.X/width*100, 'f', -1, 64) + "% " +
		strconv.FormatFloat(pos.Y/height*100, 'f', -1, 64) + "%"
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

// isTrue dereferences an optional bool.
func isTrue(v *bool) bool { return v != nil && *v }

// scaleChanged reports whether a glyph scale override differs from the
// original size. TTML carries no glyph scale, so the writer records a loss.
func scaleChanged(v *float64) bool { return v != nil && *v != 100 }
