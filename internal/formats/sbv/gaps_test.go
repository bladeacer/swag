// SPDX-License-Identifier: Apache-2.0

package sbv

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

type failingReader struct{ err error }

func (f failingReader) Read([]byte) (int, error) { return 0, f.err }

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestNames(t *testing.T) {
	if got := NewReader().Name(); got != FormatName {
		t.Fatalf("reader name = %q", got)
	}
	if got := NewWriter().Name(); got != FormatName {
		t.Fatalf("writer name = %q", got)
	}
}

func TestParseScannerError(t *testing.T) {
	if _, err := NewReader().Parse(failingReader{errors.New("boom")}); err == nil {
		t.Fatal("a failing reader must surface an error")
	}
}

func TestParseTimingErrors(t *testing.T) {
	if _, _, err := parseTiming("no comma"); err == nil {
		t.Fatal("a timing line without a comma must fail")
	}
	if _, _, err := parseTiming("bad,also bad"); err == nil {
		t.Fatal("a bad start timestamp must fail")
	}
	if _, _, err := parseTiming("0:00:01.000,bad"); err == nil {
		t.Fatal("a bad end timestamp must fail")
	}
}

func TestParseTimestampErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"shape", "0:00"},
		{"hours", "x:00:00.000"},
		{"minutes", "0:x:00.000"},
		{"seconds", "0:00:x.000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseTimestamp(tt.in); err == nil {
				t.Fatalf("parseTimestamp(%q) must fail", tt.in)
			}
		})
	}
}

func TestWriterReportsKaraoke(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues: []model.Cue{{
			Start: 0,
			End:   2 * time.Second,
			Spans: []model.TextSpan{
				{Text: "Ka", Start: 0, End: time.Second},
				{Text: "ra", Start: time.Second, End: 2 * time.Second},
			},
		}},
	}
	var out strings.Builder
	losses, err := NewWriter().Render(doc, &out)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(strings.Join(losses, "; "), "karaoke timing") {
		t.Fatalf("the karaoke loss is missing: %v", losses)
	}
}

func TestWriterWriteError(t *testing.T) {
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues:   []model.Cue{{Spans: []model.TextSpan{{Text: "x"}}}},
	}
	if _, err := NewWriter().Render(doc, failingWriter{}); err == nil {
		t.Fatal("a failing writer must surface an error")
	}
}

// TestParseBrokenEnvelope covers a damaged integrity block. A damaged block
// must fail the parse instead of passing as a plain file.
func TestParseBrokenEnvelope(t *testing.T) {
	source := "0:00:01.000,0:00:02.000\nhi\n\nNOTE swag-ir 1\n!!!!\n"
	_, err := NewReader().Parse(strings.NewReader(source))
	if err == nil || !strings.Contains(err.Error(), "parse sbv") {
		t.Fatalf("a damaged block must fail the parse: %v", err)
	}
}

// failAfterWriter accepts the first write and fails every later one, so a
// test can reach the integrity block that follows the cues.
type failAfterWriter struct{ writes int }

func (w *failAfterWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes > 1 {
		return 0, errors.New("boom")
	}
	return len(p), nil
}

// TestWriterEnvelopeWriteError covers a failure inside the integrity block.
func TestWriterEnvelopeWriteError(t *testing.T) {
	bold := true
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues:   []model.Cue{{Spans: []model.TextSpan{{Text: "x", Bold: &bold}}}},
	}
	if _, err := NewWriter().Render(doc, &failAfterWriter{}); err == nil {
		t.Fatal("a failing writer must surface an error inside the block")
	}
}
