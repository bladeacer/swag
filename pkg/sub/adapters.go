package sub

import (
	"io"

	"github.com/bladeacer/swag/internal/formats/sbv"
	"github.com/bladeacer/swag/internal/formats/srt"
)

// srtFormat adapts the internal SRT package to the public registry.
type srtFormat struct{}

func (srtFormat) Name() string       { return srt.FormatName }
func (srtFormat) Extensions() []string { return []string{"srt"} }

func (srtFormat) Parse(source io.Reader) (*Document, error) {
	return srt.NewReader().Parse(source)
}

func (srtFormat) Render(doc *Document, sink io.Writer) ([]string, error) {
	return srt.NewWriter().Render(doc, sink)
}

// sbvFormat adapts the internal SBV package to the public registry.
type sbvFormat struct{}

func (sbvFormat) Name() string         { return sbv.FormatName }
func (sbvFormat) Extensions() []string { return []string{"sbv"} }

func (sbvFormat) Parse(source io.Reader) (*Document, error) {
	return sbv.NewReader().Parse(source)
}

func (sbvFormat) Render(doc *Document, sink io.Writer) ([]string, error) {
	return sbv.NewWriter().Render(doc, sink)
}
