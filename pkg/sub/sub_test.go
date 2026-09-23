package sub

import (
	"errors"
	"io"
	"strings"
	"testing"
)

const srtSample = `1
00:00:01,000 --> 00:00:04,000
Hello from the public API.

2
00:00:04,500 --> 00:00:08,000
<i>Second</i> cue.
`

const sbvSample = `0:00:01.000,0:00:04.000
Hello from the public API.
`

func TestRegisteredIncludesBuiltins(t *testing.T) {
	names := Registered()
	joined := strings.Join(names, ",")
	if !strings.Contains(joined, "srt") || !strings.Contains(joined, "sbv") {
		t.Fatalf("built-in formats missing: %v", names)
	}
	if !isSorted(names) {
		t.Fatalf("Registered must be sorted: %v", names)
	}
}

func isSorted(names []string) bool {
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			return false
		}
	}
	return true
}

func TestDetectFormatByExtension(t *testing.T) {
	if got := DetectFormat("movie.srt"); got != "srt" {
		t.Fatalf("DetectFormat(srt) = %q", got)
	}
	if got := DetectFormat("MOVIE.SBV"); got != "sbv" {
		t.Fatalf("DetectFormat must be case-insensitive: %q", got)
	}
	if got := DetectFormat("movie.txt"); got != "" {
		t.Fatalf("unknown extension must return empty: %q", got)
	}
	if got := DetectFormat("noext"); got != "" {
		t.Fatalf("no extension must return empty: %q", got)
	}
}

func TestIdentifyByContent(t *testing.T) {
	got, err := Identify("unnamed", strings.NewReader(srtSample))
	if err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if got != "srt" {
		t.Fatalf("content sniff = %q, want srt", got)
	}
	got, err = Identify("unnamed", strings.NewReader(sbvSample))
	if err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if got != "sbv" {
		t.Fatalf("content sniff = %q, want sbv", got)
	}
}

func TestIdentifyUnknownContent(t *testing.T) {
	got, err := Identify("unnamed", strings.NewReader("just some words\n"))
	if err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if got != "" {
		t.Fatalf("plain text must stay unidentified: %q", got)
	}
}

func TestIdentifyReadError(t *testing.T) {
	want := errors.New("disk gone")
	_, err := Identify("x", io.MultiReader(strings.NewReader("-->x"), errReader{want}))
	if !errors.Is(err, want) {
		t.Fatalf("Identify must wrap the read error, got %v", err)
	}
}

type errReader struct{ err error }

func (e errReader) Read([]byte) (int, error) { return 0, e.err }

func TestParseWithExplicitFormat(t *testing.T) {
	doc, err := Parse("in.txt", strings.NewReader(srtSample), "srt")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(doc.Cues) != 2 {
		t.Fatalf("got %d cues, want 2", len(doc.Cues))
	}
}

func TestParseAutoDetect(t *testing.T) {
	doc, err := Parse("sub.sbv", strings.NewReader(sbvSample), "")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(doc.Cues) != 1 {
		t.Fatalf("got %d cues, want 1", len(doc.Cues))
	}
}

func TestParseUnknownFormatName(t *testing.T) {
	if _, err := Parse("in", strings.NewReader(srtSample), "ass"); err == nil {
		t.Fatal("unregistered format must fail")
	}
}

func TestParseUnidentifiable(t *testing.T) {
	_, err := Parse("unnamed", strings.NewReader("nothing"), "")
	if err == nil || !strings.Contains(err.Error(), "not known") {
		t.Fatalf("unidentifiable input must fail with guidance: %v", err)
	}
}

func TestRenderAndLosses(t *testing.T) {
	doc, err := Parse("in.srt", strings.NewReader(srtSample), "")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	var out strings.Builder
	losses, err := Render(doc, "sbv", &out)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out.String(), "0:00:01.000,0:00:04.000") {
		t.Fatalf("SBV timing missing:\n%s", out.String())
	}
	// The italic tag of cue 2 has no SBV form: it must be reported.
	joined := strings.Join(losses, ";")
	if !strings.Contains(joined, "inline styling") {
		t.Fatalf("loss report must carry the inline styling loss: %v", losses)
	}
}

func TestRenderUnknownFormat(t *testing.T) {
	doc := &Document{}
	if _, err := Render(doc, "ytt", io.Discard); err == nil {
		t.Fatal("unregistered writer must fail")
	}
}

func TestConvertEndToEnd(t *testing.T) {
	var out strings.Builder
	losses, err := Convert("in.srt", strings.NewReader(srtSample), "sbv", &out)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !strings.Contains(out.String(), "Hello from the public API.") {
		t.Fatalf("text missing from the conversion:\n%s", out.String())
	}
	_ = losses
}

func TestConvertUnreadableInput(t *testing.T) {
	want := errors.New("boom")
	if _, err := Convert("x", errReader{want}, "sbv", io.Discard); !errors.Is(err, want) {
		t.Fatalf("Convert must surface the read error, got %v", err)
	}
}
