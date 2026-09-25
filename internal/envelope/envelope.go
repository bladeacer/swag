// SPDX-License-Identifier: Apache-2.0

// Package envelope carries a whole document inside a plain subtitle file.
//
// A plain format such as SubRip, SBV, or WebVTT holds text and timing only,
// so a writer of one reports every other feature as a loss. The envelope
// adds a block at the end of the file that holds the full intermediate
// representation, so a swag reader restores every feature and a plain
// subtitle player stops at the last cue and ignores the block.
//
// Two block forms cover the formats:
//
//	NOTE swag-ir 1                 the plain form of SubRip, SBV, and the
//	<base64 of a JSON1 document>   WebVTT note block
//
//	<!-- swag-ir 1                  an XML comment, for TTML
//	<base64 of a JSON1 document>
//	-->
//
// Both forms are extensions of this project. A file outside swag stays a
// valid plain subtitle file, and a file inside swag keeps every feature
// across a conversion.
//
// A format that cannot hold either form, such as the strict JSON array of
// Kdenlive, carries no block. Its own writer reports the loss instead, and a
// conversion through JSON1 keeps the content.
package envelope

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/bladeacer/swag/internal/formats/json1"
	"github.com/bladeacer/swag/internal/model"
)

// Marker opens the plain block. A reader looks for it at the start of a
// line.
const Marker = "NOTE swag-ir"

// XMLMarker opens the block of an XML document, inside a comment.
const XMLMarker = "<!-- swag-ir"

// xmlClose ends an XML comment. A payload reader stops at it, so an empty
// block reports a missing payload rather than a bad one.
const xmlClose = "-->"

// Version is the version of the block shape.
const Version = "1"

// maxLineBytes bounds one line. The envelope payload is a single line, and
// a large document runs to several megabytes.
const maxLineBytes = 32 * 1024 * 1024

// ReadLines reads a whole text file into lines with the carriage return
// removed. A plain format reads the whole list, because the envelope sits at
// the end of the file.
func ReadLines(source io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(source)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, strings.TrimRight(scanner.Text(), "\r"))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// Decode reads a base64 payload into a document.
func Decode(payload string) (*model.Document, error) {
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("envelope: decode the payload: %w", err)
	}
	doc, err := json1.NewReader().Parse(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("envelope: read the payload: %w", err)
	}
	return doc, nil
}

// markerFound reports whether the line opens a block, and it checks the
// version. A line that only starts with the marker text is not a marker.
func markerFound(line, marker string) (bool, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed != marker && !strings.HasPrefix(trimmed, marker+" ") {
		return false, nil
	}
	fields := strings.Fields(trimmed)
	if len(fields) < 3 {
		return true, fmt.Errorf("envelope: the marker carries no version")
	}
	if fields[2] != Version {
		return true, fmt.Errorf("envelope: unsupported version %q", fields[2])
	}
	return true, nil
}

// payloadAfter returns the first line after start that carries content. A
// blank line is skipped, and the end of an XML comment closes the block, so
// an empty block reports a missing payload instead of a bad one.
func payloadAfter(lines []string, start int) (string, error) {
	for _, line := range lines[start:] {
		text := strings.TrimSpace(line)
		if text == "" {
			continue
		}
		if text == xmlClose {
			break
		}
		return text, nil
	}
	return "", fmt.Errorf("envelope: the block carries no payload")
}

// Write appends the plain block for doc to sink. The caller writes the cues
// first, so a plain consumer reads them and stops before the block.
func Write(sink io.Writer, doc *model.Document) error {
	if _, err := io.WriteString(sink, Marker+" "+Version+"\n"); err != nil {
		return err
	}
	encoder := base64.NewEncoder(base64.StdEncoding, sink)
	if _, err := json1.NewWriter().Render(doc, encoder); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	_, err := io.WriteString(sink, "\n")
	return err
}

// WriteXML appends the block for doc to sink as an XML comment, with indent
// before each line. The caller writes it inside an element, where an XML
// reader ignores it.
func WriteXML(sink io.Writer, indent string, doc *model.Document) error {
	if _, err := io.WriteString(sink, indent+XMLMarker+" "+Version+"\n"+indent); err != nil {
		return err
	}
	encoder := base64.NewEncoder(base64.StdEncoding, sink)
	if _, err := json1.NewWriter().Render(doc, encoder); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	_, err := io.WriteString(sink, "\n"+indent+xmlClose+"\n")
	return err
}

// Extract splits lines into the text before the block and the document the
// block carries. A file without a block returns every line and a nil
// document. A block that cannot be read returns an error, so a damaged file
// does not pass as a plain one.
func Extract(lines []string) ([]string, *model.Document, error) {
	for i, line := range lines {
		found, err := markerFound(line, Marker)
		if err != nil {
			return lines[:i], nil, err
		}
		if !found {
			continue
		}
		payload, err := payloadAfter(lines, i+1)
		if err != nil {
			return lines[:i], nil, err
		}
		doc, err := Decode(payload)
		if err != nil {
			return lines[:i], nil, err
		}
		return lines[:i], doc, nil
	}
	return lines, nil, nil
}

// ExtractXML returns the document that the block of an XML file carries, or
// nil when the file has no block. The block sits inside a comment, so the
// XML reader of the format ignores it and no line has to be removed.
func ExtractXML(data []byte) (*model.Document, error) {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		found, err := markerFound(line, XMLMarker)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		payload, err := payloadAfter(lines, i+1)
		if err != nil {
			return nil, err
		}
		return Decode(payload)
	}
	return nil, nil
}
