// SPDX-License-Identifier: Apache-2.0

// Package json1 reads and writes the swag JSON exchange format. The format
// carries the whole intermediate representation, so a document survives a
// round trip without loss. The file carries a version, and the reader lifts
// an older file to the current version before it returns the document, so a
// file written by an earlier release keeps working.
//
// Version history:
//
//	1  the first shape, without the voice of a span
//	2  adds TextSpan.Voice
package json1

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/bladeacer/swag/internal/model"
)

// FormatName is the registry name of the format.
const FormatName = "json1"

// FormatVersion is the version the writer writes and the version the reader
// returns.
const FormatVersion = "2"

// envelope wraps one document with the format version.
type envelope struct {
	Version  string          `json:"version"`
	Document *model.Document `json:"document"`
}

// migration is one step of the version chain. next names the version the
// step produces, and apply lifts a document in place.
type migration struct {
	next  string
	apply func(*model.Document)
}

// migrations maps a file version to the step that lifts a document from that
// version to the next one. The reader follows the chain to FormatVersion, so
// a file from any earlier release opens with the current reader.
var migrations = map[string]migration{
	"1": {next: "2", apply: migrateV1},
}

// migrateV1 lifts a version 1 document to version 2. Version 1 wrote no
// span voice and could leave the metadata map nil, and version 2 reads a nil
// map as an empty one. The step moves the map to that shape. A version 1
// file carries no voice, and a nil voice keeps the meaning "no voice", so
// the step leaves the spans alone.
func migrateV1(doc *model.Document) {
	if doc.Metadata == nil {
		doc.Metadata = map[string]string{}
	}
}

// migrate follows the chain from version to FormatVersion. An unknown
// version, including an empty one, has no step and is an error.
func migrate(version string, doc *model.Document) error {
	for version != FormatVersion {
		step, ok := migrations[version]
		if !ok {
			return fmt.Errorf("parse json1: unsupported version %q", version)
		}
		step.apply(doc)
		version = step.next
	}
	return nil
}

// Reader parses a json1 document.
type Reader struct{}

// NewReader returns a json1 reader.
func NewReader() *Reader { return &Reader{} }

// Name returns the registry name of the format.
func (r *Reader) Name() string { return FormatName }

// Parse reads a json1 document from source. A file from an earlier format
// version is migrated to the current one first.
func (r *Reader) Parse(source io.Reader) (*model.Document, error) {
	var env envelope
	if err := json.NewDecoder(source).Decode(&env); err != nil {
		return nil, fmt.Errorf("parse json1: %w", err)
	}
	if env.Document == nil {
		return nil, fmt.Errorf("parse json1: no document")
	}
	if err := migrate(env.Version, env.Document); err != nil {
		return nil, err
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
