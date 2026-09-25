package sub

import (
	"errors"
	"strings"
	"testing"
)

const assSample = `[Script Info]
ScriptType: v4.00+
PlayResX: 1280
PlayResY: 720

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour
Style: Default,Arial,20,&H00FFFFFF,&H000000FF,&H00000000,&H00000000

[Events]
Format: Layer, Start, End, Style, Text
Dialogue: 0,0:00:01.00,0:00:02.00,Default,Hello from ASS
`

const srv3Sample = `<?xml version="1.0" encoding="utf-8" ?>
<timedtext format="3">
<head>
<pen id="1" fc="#FEFEFE" fo="254" />
</head>
<body>
<p t="0" d="2000" p="1">Hello from SRV3.</p>
</body>
</timedtext>
`

// TestParseEveryFormat covers the parse adapter of each registered format.
func TestParseEveryFormat(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		source  string
		wantCue int
	}{
		{"srt", "in.srt", srtSample, 2},
		{"sbv", "in.sbv", sbvSample, 1},
		{"ytt", "in.ytt", yttSample, 1},
		{"srv3", "in.srv3", srv3Sample, 1},
		{"ass", "in.ass", assSample, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Parse(tt.file, strings.NewReader(tt.source), "")
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if len(doc.Cues) != tt.wantCue {
				t.Fatalf("got %d cues, want %d", len(doc.Cues), tt.wantCue)
			}
		})
	}
}

// TestRenderEveryFormat covers the render adapter of each registered format.
func TestRenderEveryFormat(t *testing.T) {
	doc, err := Parse("in.ass", strings.NewReader(assSample), "")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for _, name := range []string{"srt", "sbv", "ytt", "srv3", "ass"} {
		t.Run(name, func(t *testing.T) {
			var out strings.Builder
			if _, err := Render(doc, name, &out); err != nil {
				t.Fatalf("Render %s: %v", name, err)
			}
			if out.Len() == 0 {
				t.Fatalf("Render %s wrote nothing", name)
			}
		})
	}
}

// TestIdentifySBVWithLeadingBlankLine covers the blank line skip inside the
// SBV shape check.
func TestIdentifySBVWithLeadingBlankLine(t *testing.T) {
	got, err := Identify("unnamed", strings.NewReader("\n0:00:01.000,0:00:04.000\nHi\n"))
	if err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if got != "sbv" {
		t.Fatalf("Identify = %q, want sbv", got)
	}
}

// TestIdentifyOnlyBlankLines covers the empty shape checks.
func TestIdentifyOnlyBlankLines(t *testing.T) {
	got, err := Identify("unnamed", strings.NewReader("\n\n"))
	if err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if got != "" {
		t.Fatalf("Identify = %q, want an unknown format", got)
	}
}

// TestParseWrapsReaderError covers the error wrap of a failing format
// reader.
func TestParseWrapsReaderError(t *testing.T) {
	source := "1\n00:00:0,000 --> broken\nx\n"
	_, err := Parse("in.srt", strings.NewReader(source), "")
	if err == nil || !strings.Contains(err.Error(), "parse in.srt as srt") {
		t.Fatalf("Parse must name the file and the format: %v", err)
	}
}

// failWriter fails every write.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("disk gone") }

// TestRenderWrapsWriterError covers the error wrap of a failing format
// writer.
func TestRenderWrapsWriterError(t *testing.T) {
	doc, err := Parse("in.srt", strings.NewReader(srtSample), "")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := Render(doc, "srt", failWriter{}); err == nil {
		t.Fatal("a failing sink must surface an error")
	}
}

// TestParseFormatReaderError covers the direct read error of a format.
func TestParseFormatReaderError(t *testing.T) {
	boom := errors.New("boom")
	if _, err := Parse("in.srt", errReader{boom}, "srt"); !errors.Is(err, boom) {
		t.Fatalf("Parse must wrap the read error, got %v", err)
	}
}
