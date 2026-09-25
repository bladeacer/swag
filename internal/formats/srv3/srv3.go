// SPDX-License-Identifier: Apache-2.0

// Package srv3 reads and writes YouTube SRV3 subtitles.
//
// SRV3 is the XML that the YouTube timed-text endpoint returns. It shares
// the pen model, the window style, and the window position with YouTube
// Timed Text (see internal/formats/ytt), and it differs only as an older
// dialect of the same document. The reader and the writer therefore reuse
// the ytt package, and the registry keeps a separate name and extension so
// a caller can ask for either dialect.
package srv3

import (
	"io"

	"github.com/bladeacer/swag/internal/formats/ytt"
	"github.com/bladeacer/swag/internal/model"
)

// FormatName is the registry name of the format.
const FormatName = "srv3"

// Reader parses an SRV3 document.
type Reader struct{}

// NewReader returns an SRV3 reader.
func NewReader() *Reader { return &Reader{} }

// Name returns the registry name of the format.
func (r *Reader) Name() string { return FormatName }

// Parse reads an SRV3 document from source.
func (r *Reader) Parse(source io.Reader) (*model.Document, error) {
	return ytt.NewReader().Parse(source)
}

// Writer emits SRV3 documents from the IR.
type Writer struct{}

// NewWriter returns an SRV3 writer.
func NewWriter() *Writer { return &Writer{} }

// Name returns the registry name of the format.
func (w *Writer) Name() string { return FormatName }

// Render writes doc to sink as SRV3. The returned slice carries one entry
// per feature the format cannot express.
func (w *Writer) Render(doc *model.Document, sink io.Writer) ([]string, error) {
	return ytt.NewWriter().Render(doc, sink)
}
