// SPDX-License-Identifier: Apache-2.0

// Package ass reads Advanced SubStation Alpha (ASS) subtitles.
//
// The reader parses the Script Info, V4+ Styles, and Events sections. It
// maps each style onto the IR and flattens the style of a cue onto the
// spans of that cue, because the IR carries one document style and
// per-span overrides. The override tags of each event line are read by the
// shared richtext package.
package ass

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/bladeacer/swag/internal/model"
)

// FormatName is the registry name of the format.
const FormatName = "ass"

// The default play resolution. ASS defaults to a small frame, but the IR
// uses the common 720p frame so coordinates stay comparable between
// formats.
const (
	defaultWidth  = 1280
	defaultHeight = 720
)

// eventLine is one Dialogue line with its fields already split.
type eventLine struct {
	fields []string
}

// Reader parses an ASS document.
type Reader struct{}

// NewReader returns an ASS reader.
func NewReader() *Reader { return &Reader{} }

// Name returns the registry name of the format.
func (r *Reader) Name() string { return FormatName }

// Parse reads an ASS document from source.
func (r *Reader) Parse(source io.Reader) (*model.Document, error) {
	scanner := bufio.NewScanner(source)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	info := map[string]string{}
	var styles []model.Style
	styleByName := map[string]model.Style{}
	var events []eventLine

	section := ""
	styleFormat := []string{"name", "fontname", "fontsize", "primarycolour", "secondarycolour",
		"outlinecolour", "backcolour", "bold", "italic", "underline", "borderstyle",
		"outline", "shadow", "alignment"}
	eventFormat := []string{"layer", "start", "end", "style", "name", "marginl",
		"marginr", "marginv", "effect", "text"}

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			section = strings.ToLower(strings.Trim(trimmed, "[]"))
			continue
		}
		key, value, ok := splitKeyValue(trimmed)
		if !ok {
			continue
		}
		switch section {
		case "script info":
			info[key] = value
		case "v4+ styles", "v4 styles":
			switch strings.ToLower(key) {
			case "format":
				styleFormat = splitFormat(value)
			case "style":
				style, err := parseStyle(splitFields(value), styleFormat)
				if err != nil {
					return nil, fmt.Errorf("parse ass style: %w", err)
				}
				styles = append(styles, style)
				styleByName[strings.ToLower(style.Name)] = style
			}
		case "events":
			switch strings.ToLower(key) {
			case "format":
				eventFormat = splitFormat(value)
			case "dialogue":
				events = append(events, eventLine{fields: splitN(value, len(eventFormat))})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read ass: %w", err)
	}

	base := baseStyle(styles)
	width, height := playRes(info)

	doc := &model.Document{
		Metadata:        info,
		Styles:          orderedStyles(styles, base),
		VideoDimensions: model.Point{X: width, Y: height},
	}
	for _, ev := range events {
		cue, err := parseEvent(ev, eventFormat, styleByName, base)
		if err != nil {
			return nil, fmt.Errorf("parse ass event: %w", err)
		}
		doc.Cues = append(doc.Cues, cue)
	}
	return doc, nil
}

// splitKeyValue splits a "Key: value" line at the first colon. ASS uses
// the colon for style and event prefixes too, so the split is at the first
// one only.
func splitKeyValue(line string) (key, value string, ok bool) {
	i := strings.IndexByte(line, ':')
	if i < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]), true
}

// splitFields splits a comma-separated line into trimmed fields. Values
// keep their case, because a font name or a style name is case sensitive.
func splitFields(s string) []string {
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// splitFormat splits a Format line into lower-case field names.
func splitFormat(s string) []string {
	parts := splitFields(s)
	for i := range parts {
		parts[i] = strings.ToLower(parts[i])
	}
	return parts
}

// splitN splits an event line into exactly n fields, so the text field
// keeps its commas.
func splitN(s string, n int) []string {
	if n < 1 {
		n = 1
	}
	return strings.SplitN(s, ",", n)
}

// fieldIndex returns the index of a Format field name, or -1.
func fieldIndex(format []string, name string) int {
	for i, f := range format {
		if f == name {
			return i
		}
	}
	return -1
}

// fieldAt returns the field at the index, or "" when the index is out of
// range.
func fieldAt(fields []string, format []string, name string) string {
	i := fieldIndex(format, name)
	if i < 0 || i >= len(fields) {
		return ""
	}
	return strings.TrimSpace(fields[i])
}

// parseStyle reads one Style line into an IR style.
func parseStyle(fields, format []string) (model.Style, error) {
	var style model.Style
	style.Name = fieldAt(fields, format, "name")
	if style.Name == "" {
		style.Name = "Default"
	}
	style.Font = fieldAt(fields, format, "fontname")
	if style.Font == "" {
		style.Font = "Arial"
	}
	size, err := parseFloat(fieldAt(fields, format, "fontsize"), 20)
	if err != nil {
		return model.Style{}, err
	}
	style.Size = size

	if style.Primary, err = parseColour(fieldAt(fields, format, "primarycolour"), 255); err != nil {
		return model.Style{}, err
	}
	if style.Secondary, err = parseColour(fieldAt(fields, format, "secondarycolour"), 255); err != nil {
		return model.Style{}, err
	}
	if style.Outline, err = parseColour(fieldAt(fields, format, "outlinecolour"), 255); err != nil {
		return model.Style{}, err
	}
	if style.Shadow, err = parseColour(fieldAt(fields, format, "backcolour"), 255); err != nil {
		return model.Style{}, err
	}

	style.Bold = parseFlag(fieldAt(fields, format, "bold"))
	style.Italic = parseFlag(fieldAt(fields, format, "italic"))
	style.Underline = parseFlag(fieldAt(fields, format, "underline"))

	if style.OutlineWidth, err = parseFloat(fieldAt(fields, format, "outline"), 0); err != nil {
		return model.Style{}, err
	}
	if style.ShadowDepth, err = parseFloat(fieldAt(fields, format, "shadow"), 0); err != nil {
		return model.Style{}, err
	}
	border, err := parseInt(fieldAt(fields, format, "borderstyle"), 1)
	if err != nil {
		return model.Style{}, err
	}
	style.Box = border == 3

	align, err := parseInt(fieldAt(fields, format, "alignment"), 2)
	if err != nil {
		return model.Style{}, err
	}
	style.Alignment = anchor(align)
	return style, nil
}

// baseStyle returns the style that other sizes measure against: the style
// named Default when it exists, otherwise the first style. A document with
// no styles gets an Arial fallback.
func baseStyle(styles []model.Style) model.Style {
	for _, s := range styles {
		if strings.EqualFold(s.Name, "Default") {
			return s
		}
	}
	if len(styles) > 0 {
		return styles[0]
	}
	return model.Style{
		Name:      "Default",
		Font:      "Arial",
		Size:      20,
		Primary:   model.NewColour(255, 255, 255, 255),
		Secondary: model.NewColour(255, 255, 255, 255),
		Alignment: model.AnchorBottomCentre,
	}
}

// orderedStyles returns the styles with the base first.
func orderedStyles(styles []model.Style, base model.Style) []model.Style {
	out := []model.Style{base}
	for _, s := range styles {
		if s.Name == base.Name {
			continue
		}
		out = append(out, s)
	}
	return out
}

// playRes reads the play resolution from the Script Info map.
func playRes(info map[string]string) (float64, float64) {
	width, height := float64(defaultWidth), float64(defaultHeight)
	if v, err := strconv.ParseFloat(info["PlayResX"], 64); err == nil && v > 0 {
		width = v
	}
	if v, err := strconv.ParseFloat(info["PlayResY"], 64); err == nil && v > 0 {
		height = v
	}
	return width, height
}

// anchor maps an ASS alignment value onto a numpad anchor.
func anchor(v int) model.Anchor {
	if v < 1 || v > 9 {
		return model.AnchorBottomCentre
	}
	return model.Anchor(v)
}

// parseFlag reads an ASS boolean, where -1 is true and 0 is false.
func parseFlag(s string) bool {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "", "0", "false", "no":
		return false
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v != 0
	}
	return true
}

func parseInt(s string, def int) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return def, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parse integer %q: %w", s, err)
	}
	return v, nil
}

func parseFloat(s string, def float64) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return def, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("parse number %q: %w", s, err)
	}
	return v, nil
}
