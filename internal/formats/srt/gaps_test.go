// SPDX-License-Identifier: Apache-2.0

package srt

import (
	"errors"
	"strings"
	"testing"

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
	if _, _, err := parseTiming("no arrow here"); err == nil {
		t.Fatal("a timing line without --> must fail")
	}
	if _, _, err := parseTiming("bad --> 00:00:02,000"); err == nil {
		t.Fatal("a bad start timestamp must fail")
	}
	if _, _, err := parseTiming("00:00:01,000 --> bad"); err == nil {
		t.Fatal("a bad end timestamp must fail")
	}
}

func TestParseTimestampErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"shape", "00:00"},
		{"hours", "x:00:00,000"},
		{"minutes", "00:x:00,000"},
		{"seconds", "00:00:x,000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseTimestamp(tt.in); err == nil {
				t.Fatalf("parseTimestamp(%q) must fail", tt.in)
			}
		})
	}
}

func TestWriterReportsVerticalAndDirection(t *testing.T) {
	vertical := model.Vertical{Mode: model.VerticalColumnsRTL}
	rtl := model.DirRightToLeft
	doc := &model.Document{
		Styles: []model.Style{DefaultStyle()},
		Cues: []model.Cue{{Spans: []model.TextSpan{
			{Text: "vertical", Vertical: &vertical},
			{Text: " right to left", Direction: &rtl},
		}}},
	}
	var out strings.Builder
	losses, err := NewWriter().Render(doc, &out)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	joined := strings.Join(losses, "; ")
	for _, want := range []string{"vertical text", "right-to-left"} {
		if !strings.Contains(joined, want) {
			t.Errorf("loss report is missing %q: %v", want, losses)
		}
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
