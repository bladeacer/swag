package sub

import (
	"bytes"
	"fmt"
	"io"

	"github.com/bladeacer/swag/internal/envelope"
)

// LossPolicy decides what ConvertWith does with a degraded write.
type LossPolicy uint8

const (
	// LossReport returns the loss report of the write. It is the default.
	LossReport LossPolicy = iota
	// LossStrict returns an error when the target drops a feature. The
	// loss report still comes back with the error.
	LossStrict
	// LossSilent drops the loss report.
	LossSilent
)

// Options configures ConvertWith.
type Options struct {
	// Format names the input format. An empty value means auto-detection.
	Format string
	// Target names the output format. It is required.
	Target string
	// Styles renames a style on the way out. The key is the source name
	// and the value the target name. A missing key keeps the name.
	Styles map[string]string
	// Font replaces the font of every style and every span when it is not
	// empty. The writers may then snap the name to their own font list.
	Font string
	// Loss decides the handling of a degraded write.
	Loss LossPolicy
	// StrictCompat turns the integrity block off, so the output stays
	// inside the original specification of the target format. A lossy
	// write still reports every dropped feature, and it appends no block.
	// The flag is off by default, because a reader outside swag ignores
	// the block and reads the cues.
	StrictCompat bool
}

// ConvertWith parses source with the options and renders the document to
// the target format. It applies the style mapping and the font override
// before the write. The loss policy then decides the report and the error.
func ConvertWith(fileName string, source io.Reader, opts Options, sink io.Writer) ([]string, error) {
	if opts.Target == "" {
		return nil, fmt.Errorf("convert %s: no target format", fileName)
	}
	doc, err := Parse(fileName, source, opts.Format)
	if err != nil {
		return nil, err
	}
	applyOptions(doc, opts)
	losses, err := renderTarget(doc, opts, sink)
	if err != nil {
		return nil, err
	}
	switch opts.Loss {
	case LossStrict:
		if len(losses) > 0 {
			return losses, fmt.Errorf("convert %s to %s: %d features lost", fileName, opts.Target, len(losses))
		}
	case LossSilent:
		return nil, nil
	}
	return losses, nil
}

// renderTarget renders doc to sink. A strict conversion renders into
// memory first, removes the integrity block, and writes the plain result.
// The block is an extension of this project, and a target format that
// cannot hold it never gains one.
func renderTarget(doc *Document, opts Options, sink io.Writer) ([]string, error) {
	target := sink
	var buf bytes.Buffer
	if opts.StrictCompat {
		target = &buf
	}
	losses, err := Render(doc, opts.Target, target)
	if err != nil {
		return nil, err
	}
	if !opts.StrictCompat {
		return losses, nil
	}
	if _, err := sink.Write(envelope.Strip(buf.Bytes())); err != nil {
		return nil, fmt.Errorf("render %s: %w", opts.Target, err)
	}
	return losses, nil
}

// applyOptions rewrites the style names and the fonts of doc in place.
func applyOptions(doc *Document, opts Options) {
	for i := range doc.Styles {
		if name, ok := opts.Styles[doc.Styles[i].Name]; ok && name != "" {
			doc.Styles[i].Name = name
		}
	}
	if opts.Font == "" {
		return
	}
	for i := range doc.Styles {
		doc.Styles[i].Font = opts.Font
	}
	for i := range doc.Cues {
		for j := range doc.Cues[i].Spans {
			font := opts.Font
			doc.Cues[i].Spans[j].Font = &font
		}
	}
}
