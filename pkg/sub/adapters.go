package sub

import (
	"io"

	"github.com/bladeacer/swag/internal/formats/ass"
	"github.com/bladeacer/swag/internal/formats/sbv"
	"github.com/bladeacer/swag/internal/formats/srt"
	"github.com/bladeacer/swag/internal/formats/srv3"
	"github.com/bladeacer/swag/internal/formats/ytt"
)

// srtFormat adapts the internal SRT package to the public registry.
type srtFormat struct{}

func (srtFormat) Name() string         { return srt.FormatName }
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

// assFormat adapts the internal ASS reader to the public registry. The
// writer lands with v0.5.0, so the format reads only for now.
type assFormat struct{}

func (assFormat) Name() string         { return ass.FormatName }
func (assFormat) Extensions() []string { return []string{"ass", "ssa"} }

func (assFormat) Parse(source io.Reader) (*Document, error) {
	return ass.NewReader().Parse(source)
}

// yttFormat adapts the internal YTT package to the public registry.
type yttFormat struct{}

func (yttFormat) Name() string         { return ytt.FormatName }
func (yttFormat) Extensions() []string { return []string{"ytt"} }

func (yttFormat) Parse(source io.Reader) (*Document, error) {
	return ytt.NewReader().Parse(source)
}

func (yttFormat) Render(doc *Document, sink io.Writer) ([]string, error) {
	return ytt.NewWriter().Render(doc, sink)
}

// srv3Format adapts the internal SRV3 package to the public registry. SRV3
// shares the YTT pen model, so the adapter delegates to the same code.
type srv3Format struct{}

func (srv3Format) Name() string         { return srv3.FormatName }
func (srv3Format) Extensions() []string { return []string{"srv3"} }

func (srv3Format) Parse(source io.Reader) (*Document, error) {
	return srv3.NewReader().Parse(source)
}

func (srv3Format) Render(doc *Document, sink io.Writer) ([]string, error) {
	return srv3.NewWriter().Render(doc, sink)
}
