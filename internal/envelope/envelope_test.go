// SPDX-License-Identifier: Apache-2.0

package envelope

import (
	"encoding/base64"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

// richDocument returns a document with a feature outside plain text.
func richDocument() *model.Document {
	under := true
	scale := 140.0
	return &model.Document{
		Metadata:        map[string]string{"title": "sample"},
		VideoDimensions: model.Point{X: 1280, Y: 720},
		Styles: []model.Style{{
			Name: "Default", Font: "Roboto", Size: 20,
			Primary:   model.NewColour(255, 255, 255, 254),
			Outline:   model.NewColour(0, 0, 0, 254),
			Alignment: model.AnchorBottomCentre,
		}},
		Cues: []model.Cue{{
			Start: time.Second, End: 4 * time.Second,
			Spans: []model.TextSpan{
				{Text: "漢", Underline: &under},
				{Text: "かん", Ruby: &model.Ruby{Position: model.RubyOver}},
				{Text: "wide", ScaleX: &scale},
			},
		}},
	}
}

// splitLines renders a block and returns its lines.
func splitLines(t *testing.T, doc *model.Document) []string {
	t.Helper()
	var out strings.Builder
	if err := Write(&out, doc); err != nil {
		t.Fatalf("Write: %v", err)
	}
	return strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
}

func TestWriteAndExtract(t *testing.T) {
	doc := richDocument()
	lines := splitLines(t, doc)
	if len(lines) != 2 {
		t.Fatalf("block lines = %d, want 2:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	if lines[0] != Marker+" "+Version {
		t.Errorf("marker line = %q", lines[0])
	}

	body := []string{"1", "00:00:01,000 --> 00:00:04,000", "plain", "", lines[0], lines[1]}
	prefix, got, err := Extract(body)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(prefix) != 4 {
		t.Errorf("body = %v, want the four lines before the block", prefix)
	}
	if !reflect.DeepEqual(got, doc) {
		t.Errorf("recovered document differs:\nwant %+v\ngot  %+v", doc, got)
	}
}

func TestExtractWithoutBlock(t *testing.T) {
	lines := []string{"1", "00:00:01,000 --> 00:00:02,000", "hi", ""}
	body, doc, err := Extract(lines)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if doc != nil {
		t.Errorf("document = %+v, want nil", doc)
	}
	if !reflect.DeepEqual(body, lines) {
		t.Errorf("body = %v, want every line", body)
	}
}

func TestExtractMarkerIsWholeLine(t *testing.T) {
	// A line that only starts with the marker text is not the marker.
	lines := []string{"1", "00:00:01,000 --> 00:00:02,000", "NOTE swag-irx is a word", ""}
	_, doc, err := Extract(lines)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if doc != nil {
		t.Errorf("document = %+v, want nil", doc)
	}
}

func TestExtractErrors(t *testing.T) {
	payload, err := encode(richDocument())
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	tests := []struct {
		name string
		body []string
		want string
	}{
		{"no version", []string{Marker}, "no version"},
		{"unknown version", []string{Marker + " 9", payload}, "unsupported version"},
		{"no payload", []string{Marker + " " + Version, "", ""}, "no payload"},
		{"bad base64", []string{Marker + " " + Version, "!!!!"}, "decode the payload"},
		{"bad payload", []string{Marker + " " + Version, base64.StdEncoding.EncodeToString([]byte("nope"))}, "read the payload"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := Extract(tt.body)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

// encode returns the base64 payload of a document.
func encode(doc *model.Document) (string, error) {
	var out strings.Builder
	if err := Write(&out, doc); err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	return lines[1], nil
}

// splitXML renders an XML block and returns its lines.
func splitXML(t *testing.T, doc *model.Document) []string {
	t.Helper()
	var out strings.Builder
	if err := WriteXML(&out, "  ", doc); err != nil {
		t.Fatalf("WriteXML: %v", err)
	}
	return strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
}

func TestWriteXMLAndExtractXML(t *testing.T) {
	doc := richDocument()
	lines := splitXML(t, doc)
	if len(lines) != 3 {
		t.Fatalf("block lines = %d, want 3:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	if lines[0] != "  "+XMLMarker+" "+Version {
		t.Errorf("marker line = %q", lines[0])
	}
	if lines[2] != "  "+xmlClose {
		t.Errorf("closing line = %q", lines[2])
	}

	file := "<tt>\n  <body>\n" + strings.Join(lines, "\n") + "\n  </body>\n</tt>\n"
	got, err := ExtractXML([]byte(file))
	if err != nil {
		t.Fatalf("ExtractXML: %v", err)
	}
	if !reflect.DeepEqual(got, doc) {
		t.Errorf("recovered document differs:\nwant %+v\ngot  %+v", doc, got)
	}
}

func TestExtractXMLWithoutBlock(t *testing.T) {
	got, err := ExtractXML([]byte("<tt><body><p>plain</p></body></tt>"))
	if err != nil {
		t.Fatalf("ExtractXML: %v", err)
	}
	if got != nil {
		t.Errorf("document = %+v, want nil", got)
	}
}

func TestExtractXMLErrors(t *testing.T) {
	payload := payloadOf(t, richDocument())
	tests := []struct {
		name string
		body string
		want string
	}{
		{"no version", XMLMarker, "no version"},
		{"unknown version", XMLMarker + " 9\n" + payload + "\n" + xmlClose, "unsupported version"},
		{"no payload", XMLMarker + " " + Version + "\n" + xmlClose, "no payload"},
		{"bad base64", XMLMarker + " " + Version + "\n!!!!\n" + xmlClose, "decode the payload"},
		{"bad payload", XMLMarker + " " + Version + "\n" +
			base64.StdEncoding.EncodeToString([]byte("nope")) + "\n" + xmlClose, "read the payload"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExtractXML([]byte(tt.body))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

// payloadOf returns the base64 payload of a document.
func payloadOf(t *testing.T, doc *model.Document) string {
	t.Helper()
	payload, err := encode(doc)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	return payload
}

func TestWriteXMLSinkErrors(t *testing.T) {
	counter := &countWriter{}
	if err := WriteXML(counter, "  ", richDocument()); err != nil {
		t.Fatalf("WriteXML: %v", err)
	}
	for failAt := 1; failAt <= counter.calls; failAt++ {
		sink := &failAtWriter{failAt: failAt}
		if err := WriteXML(sink, "  ", richDocument()); err == nil {
			t.Errorf("a sink that fails on write %d must return an error", failAt)
		}
	}
	healthy := &failAtWriter{failAt: counter.calls + 1}
	if err := WriteXML(healthy, "", richDocument()); err != nil {
		t.Errorf("a healthy sink must accept the block: %v", err)
	}
}

func TestReadLines(t *testing.T) {
	lines, err := ReadLines(strings.NewReader("one\r\ntwo\n\nthree"))
	if err != nil {
		t.Fatalf("ReadLines: %v", err)
	}
	want := []string{"one", "two", "", "three"}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("lines = %q, want %q", lines, want)
	}
}

func TestReadLinesError(t *testing.T) {
	boom := errors.New("boom")
	if _, err := ReadLines(errReader{boom}); !errors.Is(err, boom) {
		t.Errorf("error = %v, want %v", err, boom)
	}
}

// errReader fails every read.
type errReader struct{ err error }

func (e errReader) Read([]byte) (int, error) { return 0, e.err }

// failAtWriter fails on the write call whose number is failAt.
type failAtWriter struct {
	calls  int
	failAt int
}

func (w *failAtWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == w.failAt {
		return 0, errors.New("sink gone")
	}
	return len(p), nil
}

// countWriter counts its write calls.
type countWriter struct{ calls int }

func (w *countWriter) Write(p []byte) (int, error) {
	w.calls++
	return len(p), nil
}

func TestWriteSinkErrors(t *testing.T) {
	counter := &countWriter{}
	if err := Write(counter, richDocument()); err != nil {
		t.Fatalf("Write: %v", err)
	}
	// Every write call in the block can fail, and each failure must surface.
	for failAt := 1; failAt <= counter.calls; failAt++ {
		sink := &failAtWriter{failAt: failAt}
		if err := Write(sink, richDocument()); err == nil {
			t.Errorf("a sink that fails on write %d must return an error", failAt)
		}
	}
	healthy := &failAtWriter{failAt: counter.calls + 1}
	if err := Write(healthy, richDocument()); err != nil {
		t.Errorf("a healthy sink must accept the block: %v", err)
	}
}
