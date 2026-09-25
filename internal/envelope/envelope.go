// SPDX-License-Identifier: Apache-2.0

// Package envelope carries a whole document inside a plain subtitle file.
//
// A plain format such as SubRip or SBV holds text and timing only, so a
// writer of one reports every other feature as a loss. The envelope adds a
// block at the end of the file that holds the full intermediate
// representation, so a swag reader restores every feature and a plain
// subtitle player stops at the last cue and ignores the block.
//
// The block looks like this:
//
//	NOTE swag-ir 1
//	<base64 of a JSON1 document>
//
// The format is our own extension. A file outside swag stays a plain
// subtitle file, and a file inside swag keeps every feature across a
// conversion.
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

// Marker opens the block. A reader looks for it at the start of a line.
const Marker = "NOTE swag-ir"

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

// Write appends the block for doc to sink. The caller writes the cues first,
// so a plain consumer reads them and stops before the block.
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

// Extract splits lines into the text before the block and the document the
// block carries. A file without a block returns every line and a nil
// document. A block that cannot be read returns an error, so a damaged file
// does not pass as a plain one.
func Extract(lines []string) ([]string, *model.Document, error) {
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != Marker && !strings.HasPrefix(trimmed, Marker+" ") {
			continue
		}
		body := lines[:i]
		fields := strings.Fields(trimmed)
		if len(fields) < 3 {
			return body, nil, fmt.Errorf("envelope: the marker carries no version")
		}
		if fields[2] != Version {
			return body, nil, fmt.Errorf("envelope: unsupported version %q", fields[2])
		}
		payload := ""
		for _, rest := range lines[i+1:] {
			if text := strings.TrimSpace(rest); text != "" {
				payload = text
				break
			}
		}
		if payload == "" {
			return body, nil, fmt.Errorf("envelope: the block carries no payload")
		}
		raw, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return body, nil, fmt.Errorf("envelope: decode the payload: %w", err)
		}
		doc, err := json1.NewReader().Parse(bytes.NewReader(raw))
		if err != nil {
			return body, nil, fmt.Errorf("envelope: read the payload: %w", err)
		}
		return body, doc, nil
	}
	return lines, nil, nil
}
