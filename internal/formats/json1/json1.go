// SPDX-License-Identifier: Apache-2.0

// Package json1 reads and writes the swag JSON exchange format. The format
// carries the whole intermediate representation, so a document survives a
// round trip without loss. The file carries a version so a later shape can
// be told apart from this one.
package json1

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/bladeacer/swag/internal/model"
)

// FormatName is the registry name of the format.
const FormatName = "json1"

// FormatVersion is the version of the file shape.
const FormatVersion = "1"

// envelope wraps one document with the format version.
type envelope struct {
	Version  string          `json:"version"`
	Document *model.Document `json:"document"`
}

// Reader parses a json1 document.
type Reader struct{}

// NewReader returns a json1 reader.
func NewReader() *Reader { return &Reader{} }

// Name returns the registry name of the format.
func (r *Reader) Name() string { return FormatName }

// Parse reads a json1 document from source.
func (r *Reader) Parse(source io.Reader) (*model.Document, error) {
	var env envelope
	if err := json.NewDecoder(source).Decode(&env); err != nil {
		return nil, fmt.Errorf("parse json1: %w", err)
	}
	if env.Version != FormatVersion {
		return nil, fmt.Errorf("parse json1: unsupported version %q", env.Version)
	}
	if env.Document == nil {
		return nil, fmt.Errorf("parse json1: no document")
	}
	return env.Document, nil
}

// Writer renders json1 documents.
type Writer struct{}

// NewWriter returns a json1 writer.
func NewWriter() *Writer { return &Writer{} }

// Name returns the registry name of the format.
func (w *Writer) Name() string { return FormatName }

// Render writes doc to sink as json1. The format carries every field of the
// IR, so the loss report is always empty.
func (w *Writer) Render(doc *model.Document, sink io.Writer) ([]string, error) {
	encoder := json.NewEncoder(sink)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(envelope{Version: FormatVersion, Document: doc}); err != nil {
		return nil, fmt.Errorf("write json1: %w", err)
	}
	return nil, nil
}
