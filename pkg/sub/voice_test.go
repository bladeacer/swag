package sub

import (
	"reflect"
	"strings"
	"testing"
)

// voiceVTT carries a voice span, the WebVTT form that names the speaker of
// the text: <v Roger Bingham>We are in the Milky Way.
const voiceVTT = "WEBVTT\n\n00:00:01.000 --> 00:00:04.000\n<v Roger Bingham>We are in the Milky Way.</v>\n"

// TestVoiceSpanAcrossFormats pins what each format does with a speaker name.
// WebVTT and json1 carry the name, the plain formats keep it in the
// integrity block, and every other format drops it and says so.
func TestVoiceSpanAcrossFormats(t *testing.T) {
	tests := []struct {
		format      string
		wantLoss    bool
		carriesName bool
	}{
		{format: "vtt", carriesName: true},
		{format: "json1", carriesName: true},
		{format: "srt", wantLoss: true, carriesName: true},
		{format: "sbv", wantLoss: true, carriesName: true},
		{format: "ass", wantLoss: true},
		{format: "ytt", wantLoss: true},
		{format: "srv3", wantLoss: true},
		{format: "ttml", wantLoss: true},
		{format: "kdenlive", wantLoss: true},
	}
	source, err := Parse("in.vtt", strings.NewReader(voiceVTT), "")
	if err != nil {
		t.Fatalf("parse the sample: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			var out strings.Builder
			losses, err := Render(source, tt.format, &out)
			if err != nil {
				t.Fatalf("render %s: %v", tt.format, err)
			}
			joined := strings.Join(losses, "; ")
			if tt.wantLoss && !strings.Contains(joined, "voice") {
				t.Errorf("the %s report must name the lost voice: %v", tt.format, losses)
			}
			if !tt.wantLoss && len(losses) != 0 {
				t.Errorf("the %s writer must keep the voice: %v", tt.format, losses)
			}

			recovered, err := Parse("out."+tt.format, strings.NewReader(out.String()), "")
			if err != nil {
				t.Fatalf("re-parse %s: %v", tt.format, err)
			}
			voice := recovered.Cues[0].Spans[0].Voice
			if tt.carriesName {
				if voice == nil || *voice != "Roger Bingham" {
					t.Errorf("the voice name did not survive %s: %+v", tt.format, voice)
				}
				return
			}
			if voice != nil {
				t.Errorf("the %s reader must not invent a voice: %q", tt.format, *voice)
			}
		})
	}
}

// TestVoiceSpanStaysOutOfThePlainText checks the promise of the integrity
// block: the plain text renders without the speaker name, so a player that
// knows only SubRip shows the subtitle alone, while the block a swag reader
// uses still holds the name.
func TestVoiceSpanStaysOutOfThePlainText(t *testing.T) {
	source, err := Parse("in.vtt", strings.NewReader(voiceVTT), "")
	if err != nil {
		t.Fatalf("parse the sample: %v", err)
	}
	var out strings.Builder
	if _, err := Render(source, "srt", &out); err != nil {
		t.Fatalf("render srt: %v", err)
	}
	cues, _, found := strings.Cut(out.String(), "NOTE swag-ir")
	if !found {
		t.Fatalf("the integrity block is missing:\n%s", out.String())
	}
	if !strings.Contains(cues, "We are in the Milky Way.") {
		t.Errorf("the cue text is missing:\n%s", cues)
	}
	if strings.Contains(cues, "Roger Bingham") {
		t.Errorf("the speaker name must stay out of the cue text:\n%s", cues)
	}
}

// TestVoiceSpanChainEndsWhereItStarted walks the name from WebVTT through a
// middle format that keeps it and back to WebVTT, which is the three-way
// conversion the block exists for. The document that comes back must equal
// the document that went in.
func TestVoiceSpanChainEndsWhereItStarted(t *testing.T) {
	for _, mid := range []string{"srt", "sbv", "json1"} {
		t.Run(mid, func(t *testing.T) {
			start, err := Parse("in.vtt", strings.NewReader(voiceVTT), "")
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			var first strings.Builder
			if _, err := Render(start, mid, &first); err != nil {
				t.Fatalf("render %s: %v", mid, err)
			}
			middle, err := Parse("mid."+mid, strings.NewReader(first.String()), "")
			if err != nil {
				t.Fatalf("parse %s: %v", mid, err)
			}
			var last strings.Builder
			if _, err := Render(middle, "vtt", &last); err != nil {
				t.Fatalf("render vtt: %v", err)
			}
			end, err := Parse("out.vtt", strings.NewReader(last.String()), "")
			if err != nil {
				t.Fatalf("parse vtt: %v", err)
			}
			if !reflect.DeepEqual(start, end) {
				t.Errorf("vtt to %s to vtt changed the document:\n want %+v\n  got %+v", mid, start, end)
			}
		})
	}
}
