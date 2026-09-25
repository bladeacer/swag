package sub

import (
	"os"
	"strings"
	"testing"
)

// The fuzz targets run the seed corpus during a normal test run and a longer
// search under `go test -fuzz`. The nightly workflow runs each target for
// several minutes, so a crash in a reader shows up without a user report.

// fuzzReader seeds a target and checks that a parsed document survives a
// lossless JSON1 round trip. A reader must not panic on any input, and a
// document it accepts must render.
func fuzzReader(f *testing.F, name string, seeds []string) {
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data string) {
		doc, err := Parse("fuzz."+name, strings.NewReader(data), name)
		if err != nil {
			return
		}
		var out strings.Builder
		if _, err := Render(doc, "json1", &out); err != nil {
			t.Fatalf("render parsed %s as json1: %v", name, err)
		}
	})
}

// seedFile returns the content of a fixture file, or nothing when the file
// is unavailable.
func seedFile(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return []string{string(data)}
}

// FuzzParseSRT fuzzes the SubRip reader.
func FuzzParseSRT(f *testing.F) {
	fuzzReader(f, "srt", []string{
		srtSample,
		"1\n00:00:01,000 --> 00:00:04,000\n<i>Hello</i> world.\n",
		"1\n00:00:0,000 --> broken\nx\n",
	})
}

// FuzzParseSBV fuzzes the SBV reader.
func FuzzParseSBV(f *testing.F) {
	fuzzReader(f, "sbv", []string{
		sbvSample,
		"0:00:01.000,0:00:04.000\nHello world.\n\n",
		"0:00:01,2\nbroken\n",
	})
}

// FuzzParseYTT fuzzes the YouTube Timed Text reader.
func FuzzParseYTT(f *testing.F) {
	seeds := []string{yttSample, "<timedtext><body><p t=\"0\" d=\"1000\">Hi</p></body></timedtext>", "<timedtext"}
	fuzzReader(f, "ytt", append(seeds, seedFile("../../internal/formats/ytt/testdata/sample.ytt")...))
}

// FuzzParseSRV3 fuzzes the SRV3 reader.
func FuzzParseSRV3(f *testing.F) {
	seeds := []string{srv3Sample}
	fuzzReader(f, "srv3", append(seeds, seedFile("../../internal/formats/srv3/testdata/sample.srv3")...))
}

// FuzzParseASS fuzzes the ASS reader.
func FuzzParseASS(f *testing.F) {
	seeds := []string{assSample, "[Script Info]\n[Events]\nDialogue: broken\n"}
	for _, name := range []string{"colour.ass", "karaoke.ass", "cjk.ass", "overrides.ass"} {
		seeds = append(seeds, seedFile("../../internal/formats/ass/testdata/"+name)...)
	}
	fuzzReader(f, "ass", seeds)
}

// FuzzParseJSON1 fuzzes the lossless JSON1 reader.
func FuzzParseJSON1(f *testing.F) {
	fuzzReader(f, "json1", []string{json1Sample, "{}", "[]", `{"version":"1"}`})
}

// FuzzParseVTT fuzzes the WebVTT reader.
func FuzzParseVTT(f *testing.F) {
	fuzzReader(f, "vtt", []string{
		vttSample,
		"WEBVTT\n\nNOTE note\n\n00:00:01.000 --> 00:00:04.000 align:start\n<i>Hi</i>\n",
		"WEBVTT\n\nbroken\n",
	})
}

// FuzzParseTTML fuzzes the TTML reader.
func FuzzParseTTML(f *testing.F) {
	seeds := []string{ttmlSample, "<tt><body><div><p begin=\"1s\" end=\"2s\">Hi</p></div></body></tt>", "<tt"}
	fuzzReader(f, "ttml", append(seeds, seedFile("../../internal/formats/ttml/testdata/youtube.ttml")...))
}

// FuzzParseKdenlive fuzzes the Kdenlive subtitle JSON reader.
func FuzzParseKdenlive(f *testing.F) {
	seeds := []string{kdenliveSample, "[]", "[{}]"}
	fuzzReader(f, "kdenlive", append(seeds, seedFile("../../internal/formats/kdenlive/testdata/track.json")...))
}
