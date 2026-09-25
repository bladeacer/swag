package sub

import (
	"io"

	"github.com/bladeacer/swag/internal/formats/ass"
	"github.com/bladeacer/swag/internal/formats/json1"
	"github.com/bladeacer/swag/internal/formats/kdenlive"
	"github.com/bladeacer/swag/internal/formats/sbv"
	"github.com/bladeacer/swag/internal/formats/srt"
	"github.com/bladeacer/swag/internal/formats/srv3"
	"github.com/bladeacer/swag/internal/formats/ttml"
	"github.com/bladeacer/swag/internal/formats/vtt"
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

// assFormat adapts the internal ASS package to the public registry.
type assFormat struct{}

func (assFormat) Name() string         { return ass.FormatName }
func (assFormat) Extensions() []string { return []string{"ass", "ssa"} }

func (assFormat) Parse(source io.Reader) (*Document, error) {
	return ass.NewReader().Parse(source)
}

func (assFormat) Render(doc *Document, sink io.Writer) ([]string, error) {
	return ass.NewWriter().Render(doc, sink)
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

// json1Format adapts the internal json1 package to the public registry.
type json1Format struct{}

func (json1Format) Name() string         { return json1.FormatName }
func (json1Format) Extensions() []string { return []string{"json1"} }

func (json1Format) Parse(source io.Reader) (*Document, error) {
	return json1.NewReader().Parse(source)
}

func (json1Format) Render(doc *Document, sink io.Writer) ([]string, error) {
	return json1.NewWriter().Render(doc, sink)
}

// ttmlFormat adapts the internal TTML package to the public registry.
type ttmlFormat struct{}

func (ttmlFormat) Name() string         { return ttml.FormatName }
func (ttmlFormat) Extensions() []string { return []string{"ttml", "dfxp"} }

func (ttmlFormat) Parse(source io.Reader) (*Document, error) {
	return ttml.NewReader().Parse(source)
}

func (ttmlFormat) Render(doc *Document, sink io.Writer) ([]string, error) {
	return ttml.NewWriter().Render(doc, sink)
}

// vttFormat adapts the internal WebVTT package to the public registry.
type vttFormat struct{}

func (vttFormat) Name() string         { return vtt.FormatName }
func (vttFormat) Extensions() []string { return []string{"vtt"} }

func (vttFormat) Parse(source io.Reader) (*Document, error) {
	return vtt.NewReader().Parse(source)
}

func (vttFormat) Render(doc *Document, sink io.Writer) ([]string, error) {
	return vtt.NewWriter().Render(doc, sink)
}

// kdenliveFormat adapts the internal Kdenlive package to the public
// registry.
type kdenliveFormat struct{}

func (kdenliveFormat) Name() string         { return kdenlive.FormatName }
func (kdenliveFormat) Extensions() []string { return []string{"kdenlive"} }

func (kdenliveFormat) Parse(source io.Reader) (*Document, error) {
	return kdenlive.NewReader().Parse(source)
}

func (kdenliveFormat) Render(doc *Document, sink io.Writer) ([]string, error) {
	return kdenlive.NewWriter().Render(doc, sink)
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
