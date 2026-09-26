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

func TestStripPlainBlock(t *testing.T) {
	var out strings.Builder
	out.WriteString("1\n00:00:01,000 --> 00:00:04,000\nplain\n\n")
	if err := Write(&out, richDocument()); err != nil {
		t.Fatalf("Write: %v", err)
	}
	stripped := string(Strip([]byte(out.String())))
	if strings.Contains(stripped, Marker) {
		t.Errorf("the marker survived the strip:\n%s", stripped)
	}
	if !strings.HasSuffix(stripped, "plain\n") {
		t.Errorf("the cue text did not survive:\n%q", stripped)
	}
}

// TestStripPlainMarkerWithoutVersion covers the exact marker form, which
// carries no version field.
func TestStripPlainMarkerWithoutVersion(t *testing.T) {
	got := string(Strip([]byte("cue\n" + Marker + "\nPAYLOAD\n")))
	if got != "cue" {
		t.Errorf("Strip = %q, want %q", got, "cue")
	}
}

func TestStripXMLBlock(t *testing.T) {
	var out strings.Builder
	out.WriteString("<tt>\n  <body>\n")
	if err := WriteXML(&out, "  ", richDocument()); err != nil {
		t.Fatalf("WriteXML: %v", err)
	}
	out.WriteString("  </body>\n</tt>\n")
	stripped := string(Strip([]byte(out.String())))
	if strings.Contains(stripped, XMLMarker) || strings.Contains(stripped, xmlClose) {
		t.Errorf("the comment survived the strip:\n%s", stripped)
	}
	want := "<tt>\n  <body>\n  </body>\n</tt>\n"
	if stripped != want {
		t.Errorf("Strip = %q, want %q", stripped, want)
	}
}

// TestStripXMLMarkerWithoutVersion covers the exact XML marker form.
func TestStripXMLMarkerWithoutVersion(t *testing.T) {
	got := string(Strip([]byte("<tt>\n" + XMLMarker + "\nPAYLOAD\n" + xmlClose + "\n</tt>\n")))
	want := "<tt>\n</tt>\n"
	if got != want {
		t.Errorf("Strip = %q, want %q", got, want)
	}
}

// TestStripOpenComment covers a comment with no closing line. The input must
// stay unchanged, so a damaged render does not lose its tail.
func TestStripOpenComment(t *testing.T) {
	in := []byte("<tt>\n" + XMLMarker + " " + Version + "\nPAYLOAD\n</tt>\n")
	if got := Strip(in); string(got) != string(in) {
		t.Errorf("an open comment must leave the input unchanged, got %q", got)
	}
}

func TestStripWithoutBlock(t *testing.T) {
	in := []byte("1\n00:00:01,000 --> 00:00:02,000\nplain\n")
	if got := Strip(in); string(got) != string(in) {
		t.Errorf("Strip changed a plain document: %q", got)
	}
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

// collectRead returns a handler that records every line it sees.
func collectRead(lines *[]string) func(string) error {
	return func(line string) error {
		*lines = append(*lines, line)
		return nil
	}
}

// TestReadStreamsTheBody covers a file without a block, so the handler sees
// every line and Read returns no document.
func TestReadStreamsTheBody(t *testing.T) {
	var seen []string
	doc, err := Read(strings.NewReader("one\r\ntwo\n\nthree"), collectRead(&seen))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if doc != nil {
		t.Fatalf("document = %+v, want nil", doc)
	}
	want := []string{"one", "two", "", "three"}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("lines = %q, want %q", seen, want)
	}
}

// TestReadRecoversTheBlock covers the streaming scan of a file with an
// integrity block. The handler sees only the lines before the marker, and
// Read returns the document of the block.
func TestReadRecoversTheBlock(t *testing.T) {
	doc := richDocument()
	payload, err := encode(doc)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	source := "1\n00:00:01,000 --> 00:00:04,000\nplain\n\n" + Marker + " " + Version + "\n" + payload + "\n"
	var seen []string
	got, err := Read(strings.NewReader(source), collectRead(&seen))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !reflect.DeepEqual(got, doc) {
		t.Fatalf("recovered document differs:\nwant %+v\ngot  %+v", doc, got)
	}
	if len(seen) != 4 {
		t.Fatalf("body = %v, want the four lines before the block", seen)
	}
}

// TestReadErrors covers every damaged block that a stream can reach.
func TestReadErrors(t *testing.T) {
	payload, err := encode(richDocument())
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"no version", Marker + "\n", "no version"},
		{"unknown version", Marker + " 9\n" + payload + "\n", "unsupported version"},
		{"no payload", Marker + " " + Version + "\n\n\n", "no payload"},
		{"bad base64", Marker + " " + Version + "\n!!!!\n", "decode the payload"},
		{"bad payload", Marker + " " + Version + "\n" + base64.StdEncoding.EncodeToString([]byte("nope")) + "\n", "read the payload"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Read(strings.NewReader(tt.source), func(string) error { return nil })
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

// TestReadScanError covers a source that fails before any line.
func TestReadScanError(t *testing.T) {
	boom := errors.New("boom")
	if _, err := Read(errReader{boom}, func(string) error { return nil }); !errors.Is(err, boom) {
		t.Fatalf("error = %v, want %v", err, boom)
	}
}

// TestReadPayloadScanError covers a source that carries the marker and then
// fails before the payload.
func TestReadPayloadScanError(t *testing.T) {
	boom := errors.New("boom")
	source := &partialReader{data: Marker + " " + Version + "\n", err: boom}
	if _, err := Read(source, func(string) error { return nil }); !errors.Is(err, boom) {
		t.Fatalf("error = %v, want %v", err, boom)
	}
}

// TestReadHandlerError covers a body line that the handler refuses.
func TestReadHandlerError(t *testing.T) {
	boom := errors.New("boom")
	if _, err := Read(strings.NewReader("one\ntwo\n"), func(string) error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("error = %v, want %v", err, boom)
	}
}

// partialReader returns its data once, then fails every later read.
type partialReader struct {
	data string
	err  error
	done bool
}

func (r *partialReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, r.err
	}
	r.done = true
	return copy(p, r.data), nil
}
