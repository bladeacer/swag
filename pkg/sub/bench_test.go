package sub

import (
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

// benchSize is the cue count of the benchmark document. Large subtitle files
// run to several thousand cues, so the benchmark uses ten thousand.
const benchSize = 10000

// benchDocument builds a document with size plain cues.
func benchDocument(size int) *Document {
	doc := &Document{
		VideoDimensions: model.Point{X: 1280, Y: 720},
		Styles: []model.Style{{
			Name:         "Default",
			Font:         "Arial",
			Size:         20,
			Primary:      model.NewColour(255, 255, 255, 255),
			Secondary:    model.NewColour(255, 255, 255, 255),
			Outline:      model.NewColour(0, 0, 0, 255),
			OutlineWidth: 2,
			Alignment:    model.AnchorBottomCentre,
		}},
		Cues: make([]model.Cue, 0, size),
	}
	for i := 0; i < size; i++ {
		start := time.Duration(i) * 2 * time.Second
		doc.Cues = append(doc.Cues, model.Cue{
			Start: start,
			End:   start + 1900*time.Millisecond,
			Spans: []model.TextSpan{{Text: "A subtitle line with some words in it."}},
		})
	}
	return doc
}

// benchInputs renders the benchmark document once per format and returns the
// text of each.
func benchInputs(b *testing.B) map[string]string {
	b.Helper()
	doc := benchDocument(benchSize)
	inputs := make(map[string]string, len(Registered()))
	for _, name := range Registered() {
		var out strings.Builder
		if _, err := Render(doc, name, &out); err != nil {
			b.Fatalf("render %s: %v", name, err)
		}
		inputs[name] = out.String()
	}
	return inputs
}

// BenchmarkParse measures the parse of a ten-thousand-cue document in every
// registered format.
func BenchmarkParse(b *testing.B) {
	inputs := benchInputs(b)
	for _, name := range Registered() {
		source := inputs[name]
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(source)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := Parse("bench."+name, strings.NewReader(source), name); err != nil {
					b.Fatalf("parse: %v", err)
				}
			}
		})
	}
}

// BenchmarkRender measures the render of a ten-thousand-cue document in
// every registered format.
func BenchmarkRender(b *testing.B) {
	doc := benchDocument(benchSize)
	for _, name := range Registered() {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := Render(doc, name, discardWriter{}); err != nil {
					b.Fatalf("render: %v", err)
				}
			}
		})
	}
}

// BenchmarkConvert measures the full conversion of a ten-thousand-cue SubRip
// document into every other format.
func BenchmarkConvert(b *testing.B) {
	inputs := benchInputs(b)
	source := inputs["srt"]
	for _, name := range Registered() {
		if name == "srt" {
			continue
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := Convert("bench.srt", strings.NewReader(source), name, discardWriter{}); err != nil {
					b.Fatalf("convert: %v", err)
				}
			}
		})
	}
}

// discardWriter drops every write.
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
