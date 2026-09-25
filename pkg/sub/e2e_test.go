package sub

import (
	"reflect"
	"strings"
	"testing"
)

// TestEveryFormatThroughJSON1 proves the lossless hop. Each format reads a
// document, the JSON1 writer renders it, and the JSON1 reader returns the
// same document. A chain through the exchange format therefore keeps every
// feature of every reader.
func TestEveryFormatThroughJSON1(t *testing.T) {
	tests := []struct {
		name   string
		file   string
		source string
	}{
		{"srt", "in.srt", srtSample},
		{"sbv", "in.sbv", sbvSample},
		{"ytt", "in.ytt", yttSample},
		{"srv3", "in.srv3", srv3Sample},
		{"ass", "in.ass", assSample},
		{"json1", "in.json1", json1Sample},
		{"vtt", "in.vtt", vttSample},
		{"ttml", "in.ttml", ttmlSample},
		{"kdenlive", "in.kdenlive", kdenliveSample},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Parse(tt.file, strings.NewReader(tt.source), "")
			if err != nil {
				t.Fatalf("parse %s: %v", tt.name, err)
			}
			var out strings.Builder
			losses, err := Render(doc, "json1", &out)
			if err != nil {
				t.Fatalf("render json1: %v", err)
			}
			if len(losses) != 0 {
				t.Fatalf("json1 must be lossless: %v", losses)
			}
			back, err := Parse("out.json1", strings.NewReader(out.String()), "")
			if err != nil {
				t.Fatalf("parse json1: %v", err)
			}
			if !reflect.DeepEqual(doc, back) {
				t.Errorf("the exchange round trip changed the %s document:\n want %+v\n  got %+v", tt.name, doc, back)
			}
		})
	}
}

// TestASSFixturesThroughJSON1 runs the richer ASS fixtures through the
// exchange format, so ruby groups, karaoke windows, and the ASS overrides
// all meet the lossless hop.
func TestASSFixturesThroughJSON1(t *testing.T) {
	for _, name := range []string{"colour.ass", "karaoke.ass", "cjk.ass", "overrides.ass"} {
		t.Run(name, func(t *testing.T) {
			doc := assFixture(t, name)
			var first, second strings.Builder
			if _, err := Render(doc, "json1", &first); err != nil {
				t.Fatalf("render json1: %v", err)
			}
			back, err := Parse("mid.json1", strings.NewReader(first.String()), "")
			if err != nil {
				t.Fatalf("parse json1: %v", err)
			}
			if !reflect.DeepEqual(doc, back) {
				t.Fatal("the document changed on the way through the exchange format")
			}
			if _, err := Render(back, "json1", &second); err != nil {
				t.Fatalf("render json1 again: %v", err)
			}
			if first.String() != second.String() {
				t.Error("a second exchange write differs from the first one")
			}
		})
	}
}

// TestJSON1ToEveryFormat reads the exchange format and writes every
// registered target, so one document exercises every writer adapter from a
// single source.
func TestJSON1ToEveryFormat(t *testing.T) {
	doc := assFixture(t, "colour.ass")
	for _, name := range Registered() {
		t.Run(name, func(t *testing.T) {
			var out strings.Builder
			if _, err := Render(doc, name, &out); err != nil {
				t.Fatalf("render %s: %v", name, err)
			}
			if out.Len() == 0 {
				t.Fatalf("the %s writer wrote nothing", name)
			}
		})
	}
}
