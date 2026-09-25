// SPDX-License-Identifier: Apache-2.0

package ytt

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/bladeacer/swag/internal/model"
)

// xmlTimedText is the root element of a YTT or SRV3 document.
type xmlTimedText struct {
	XMLName xml.Name `xml:"timedtext"`
	Format  string   `xml:"format,attr"`
	Head    xmlHead  `xml:"head"`
	Body    xmlBody  `xml:"body"`
}

// xmlHead holds the named pens, window styles, and window positions.
type xmlHead struct {
	Pens      []xmlPen         `xml:"pen"`
	Styles    []xmlWindowStyle `xml:"ws"`
	Positions []xmlWindowPos   `xml:"wp"`
}

// xmlPen is one <pen> element. The attributes stay as strings because the
// format uses "0" and "1" for booleans and mixes colours with numbers.
type xmlPen struct {
	ID          int    `xml:"id,attr"`
	Bold        string `xml:"b,attr"`
	Italic      string `xml:"i,attr"`
	Underline   string `xml:"u,attr"`
	ForeColour  string `xml:"fc,attr"`
	ForeOpacity string `xml:"fo,attr"`
	BackColour  string `xml:"bc,attr"`
	BackOpacity string `xml:"bo,attr"`
	EdgeColour  string `xml:"ec,attr"`
	EdgeOpacity string `xml:"eo,attr"`
	EdgeType    string `xml:"et,attr"`
	FontStyle   string `xml:"fs,attr"`
	FontScale   string `xml:"sz,attr"`
	Ruby        string `xml:"rb,attr"`
	Offset      string `xml:"of,attr"`
	Packed      string `xml:"hg,attr"`
}

// xmlWindowStyle is one <ws> element: the window of a line.
type xmlWindowStyle struct {
	ID            int    `xml:"id,attr"`
	Justification string `xml:"ju,attr"`
	Pitch         string `xml:"pd,attr"`
	Skew          string `xml:"sd,attr"`
	WindowOpacity string `xml:"wfo,attr"`
}

// xmlWindowPos is one <wp> element: a named window position.
type xmlWindowPos struct {
	ID     int    `xml:"id,attr"`
	Anchor string `xml:"ap,attr"`
	X      string `xml:"ah,attr"`
	Y      string `xml:"av,attr"`
}

// xmlBody holds the lines of a document.
type xmlBody struct {
	Paragraphs []xmlParagraph `xml:"p"`
}

// xmlParagraph is one <p> element. Its content is mixed: plain text, <s>
// runs, and line breaks, so it carries its own unmarshaller.
type xmlParagraph struct {
	Start    string
	Duration string
	Position string
	Style    string
	Pen      string
	Runs     []xmlRun
}

// xmlRun is one item in the content of a <p> element.
type xmlRun struct {
	// Text is a plain text run.
	Text string
	// Break is a <br/> element.
	Break bool
	// Span is an <s> element with its own pen and karaoke time.
	Span *xmlSpan
}

// xmlSpan is one <s> element.
type xmlSpan struct {
	Pen  string
	Time string
	Text string
}

// UnmarshalXML reads a <p> element and keeps the order of its text runs,
// spans, and line breaks.
func (p *xmlParagraph) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		switch a.Name.Local {
		case "t":
			p.Start = a.Value
		case "d":
			p.Duration = a.Value
		case "wp":
			p.Position = a.Value
		case "ws":
			p.Style = a.Value
		case "p":
			p.Pen = a.Value
		}
	}
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("parse paragraph: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "s":
				span := &xmlSpan{}
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "p":
						span.Pen = a.Value
					case "t":
						span.Time = a.Value
					}
				}
				var text string
				if err := d.DecodeElement(&text, &t); err != nil {
					return fmt.Errorf("parse span: %w", err)
				}
				span.Text = text
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

// toPen converts a <pen> element into the shared pen model.
func (x xmlPen) toPen() (pen, error) {
	p := pen{script: scriptUnset}
	var err error
	if p.bold, err = attrTrue(x.Bold); err != nil {
		return pen{}, fmt.Errorf("pen %d: %w", x.ID, err)
	}
	if p.italic, err = attrTrue(x.Italic); err != nil {
		return pen{}, fmt.Errorf("pen %d: %w", x.ID, err)
	}
	if p.underline, err = attrTrue(x.Underline); err != nil {
		return pen{}, fmt.Errorf("pen %d: %w", x.ID, err)
	}
	if p.fore, err = attrColour(x.ForeColour, x.ForeOpacity); err != nil {
		return pen{}, fmt.Errorf("pen %d foreground: %w", x.ID, err)
	}
	if p.back, err = attrColour(x.BackColour, x.BackOpacity); err != nil {
		return pen{}, fmt.Errorf("pen %d background: %w", x.ID, err)
	}
	if p.edge, err = attrColour(x.EdgeColour, x.EdgeOpacity); err != nil {
		return pen{}, fmt.Errorf("pen %d edge: %w", x.ID, err)
	}
	if p.edgeKind, err = attrEdgeKind(x.EdgeType); err != nil {
		return pen{}, fmt.Errorf("pen %d: %w", x.ID, err)
	}
	if strings.TrimSpace(x.FontStyle) != "" {
		style, err := attrInt(x.FontStyle, int(FontDefault))
		if err != nil {
			return pen{}, fmt.Errorf("pen %d font style: %w", x.ID, err)
		}
		p.fontStyle, p.hasFont = FontStyle(style), true
	}
	if strings.TrimSpace(x.FontScale) != "" {
		scale, err := attrInt(x.FontScale, 100)
		if err != nil {
			return pen{}, fmt.Errorf("pen %d font scale: %w", x.ID, err)
		}
		p.fontScale, p.hasScale = scale, true
	}
	if p.ruby, err = attrInt(x.Ruby, rubyNone); err != nil {
		return pen{}, fmt.Errorf("pen %d ruby: %w", x.ID, err)
	}
	if strings.TrimSpace(x.Offset) != "" {
		if p.script, err = attrInt(x.Offset, scriptRegular); err != nil {
			return pen{}, fmt.Errorf("pen %d offset: %w", x.ID, err)
		}
	}
	if p.packed, err = attrTrue(x.Packed); err != nil {
		return pen{}, fmt.Errorf("pen %d: %w", x.ID, err)
	}
	return p, nil
}

// toPosition converts a <wp> element into a layout anchor and a position in
// video pixels.
func (x xmlWindowPos) toPosition(width, height float64) (*model.Layout, error) {
	ap, err := attrInt(x.Anchor, 0)
	if err != nil {
		return nil, fmt.Errorf("window position %d anchor: %w", x.ID, err)
	}
	xPercent, err := attrFloat(x.X, 50)
	if err != nil {
		return nil, fmt.Errorf("window position %d x: %w", x.ID, err)
	}
	yPercent, err := attrFloat(x.Y, 90)
	if err != nil {
		return nil, fmt.Errorf("window position %d y: %w", x.ID, err)
	}
	anchor := anchorFromAP(ap)
	pos := model.Point{X: xPercent / 100 * width, Y: yPercent / 100 * height}
	return &model.Layout{Position: &pos, Anchor: &anchor}, nil
}

// attrTrue reads a "0" or "1" attribute. A missing attribute is false.
func attrTrue(s string) (bool, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return false, nil
	}
	switch s {
	case "0", "false":
		return false, nil
	case "1", "true":
		return true, nil
	}
	return false, fmt.Errorf("parse boolean %q: want 0 or 1", s)
}

// attrInt reads an integer attribute with a default for a missing value.
func attrInt(s string, def int) (int, error) {
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

// attrFloat reads a decimal attribute with a default for a missing value.
func attrFloat(s string, def float64) (float64, error) {
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

// attrColour reads a colour and its opacity into one RGBA colour. A missing
// colour stays nil, which means "inherit".
func attrColour(hex, opacity string) (*model.Colour, error) {
	if strings.TrimSpace(hex) == "" {
		return nil, nil
	}
	c, err := parseHexColour(hex)
	if err != nil {
		return nil, err
	}
	a, err := parseOpacity(opacity)
	if err != nil {
		return nil, err
	}
	c.A = a
	return &c, nil
}

// attrEdgeKind maps an "et" value onto a shadow kind. Zero means no edge.
func attrEdgeKind(s string) (model.ShadowKind, error) {
	v, err := attrInt(s, 0)
	if err != nil {
		return 0, fmt.Errorf("parse edge type %q: %w", s, err)
	}
	switch v {
	case 0:
		return 0, nil
	case 1:
		return model.ShadowHard, nil
	case 2:
		return model.ShadowBevel, nil
	case 3:
		return model.ShadowGlow, nil
	case 4:
		return model.ShadowSoft, nil
	}
	return 0, fmt.Errorf("parse edge type %q: want 0 to 4", s)
}

// attrYTTInt reads an integer attribute and reports whether it was present.
func attrYTTInt(s string) (int, bool, error) {
	if strings.TrimSpace(s) == "" {
		return 0, false, nil
	}
	v, err := attrInt(s, 0)
	return v, true, err
}
