package sub

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// assFixture reads an ASS fixture from the format test data.
func assFixture(t *testing.T, name string) *Document {
	t.Helper()
	data, err := os.ReadFile("../../internal/formats/ass/testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	doc, err := Parse(name, strings.NewReader(string(data)), "")
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return doc
}

// TestIntegrityThroughPlainFormats proves a three-way conversion. The ASS
// source carries ruby, karaoke, and styling. A plain writer cannot express
// them, so it reports the loss and appends the integrity block. The plain
// reader restores the exact document, and a second ASS write equals the
// first one.
func TestIntegrityThroughPlainFormats(t *testing.T) {
	for _, plain := range []string{"srt", "sbv"} {
		t.Run(plain, func(t *testing.T) {
			source := assFixture(t, "colour.ass")

			var plainOut strings.Builder
			losses, err := Render(source, plain, &plainOut)
			if err != nil {
				t.Fatalf("render %s: %v", plain, err)
			}
			if len(losses) == 0 {
				t.Fatalf("the fixture must provoke a loss in %s", plain)
			}
			if !strings.Contains(plainOut.String(), "NOTE swag-ir") {
				t.Fatalf("the integrity block is missing from the %s output", plain)
			}

			recovered, err := Parse("out."+plain, strings.NewReader(plainOut.String()), "")
			if err != nil {
				t.Fatalf("re-parse %s: %v", plain, err)
			}
			if !reflect.DeepEqual(source, recovered) {
				t.Errorf("the %s round trip changed the document", plain)
			}

			var first, second strings.Builder
			if _, err := Render(source, "ass", &first); err != nil {
				t.Fatalf("render ASS: %v", err)
			}
			if _, err := Render(recovered, "ass", &second); err != nil {
				t.Fatalf("render ASS from the recovered document: %v", err)
			}
			if first.String() != second.String() {
				t.Errorf("the ASS output changed after the %s pass", plain)
			}
		})
	}
}

// TestIntegrityKeepsKaraokeTiming checks the karaoke fixture, which carries
// per-syllable timing that a plain format cannot hold.
func TestIntegrityKeepsKaraokeTiming(t *testing.T) {
	source := assFixture(t, "karaoke.ass")
	var out strings.Builder
	if _, err := Render(source, "srt", &out); err != nil {
		t.Fatalf("render srt: %v", err)
	}
	recovered, err := Parse("out.srt", strings.NewReader(out.String()), "")
	if err != nil {
		t.Fatalf("re-parse srt: %v", err)
	}
	if !reflect.DeepEqual(source, recovered) {
		t.Error("the karaoke timing did not survive the SubRip pass")
	}
}

// TestPlainDocumentStaysPlain checks that a document whose features all fit
// in SubRip writes no block, so a plain file stays plain.
func TestPlainDocumentStaysPlain(t *testing.T) {
	doc, err := Parse("in.srt", strings.NewReader(srtSample), "")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out strings.Builder
	losses, err := Render(doc, "srt", &out)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(losses) != 0 {
		t.Fatalf("losses = %v, want none", losses)
	}
	if strings.Contains(out.String(), "NOTE swag-ir") {
		t.Errorf("a plain document must not gain a block:\n%s", out.String())
	}
}

// TestStandardPlainFileStillReads checks that a hand-written SubRip file
// without a block parses as before. The tool stays compatible with the base
// format.
func TestStandardPlainFileStillReads(t *testing.T) {
	const source = "1\n00:00:01,000 --> 00:00:04,000\n<i>Hello</i> world.\n\n2\n00:00:04,500 --> 00:00:08,000\nSecond line.\n"
	doc, err := Parse("in.srt", strings.NewReader(source), "")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(doc.Cues) != 2 {
		t.Fatalf("cues = %d, want 2", len(doc.Cues))
	}
	if got := doc.Cues[0].Text(); got != "Hello world." {
		t.Errorf("first cue text = %q", got)
	}
}
