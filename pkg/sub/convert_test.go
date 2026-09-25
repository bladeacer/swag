package sub

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestConvertWithStyleMapping(t *testing.T) {
	var out strings.Builder
	if _, err := ConvertWith("in.ass", strings.NewReader(assSample), Options{
		Target: "ass",
		Styles: map[string]string{"Default": "Custom"},
	}, &out); err != nil {
		t.Fatalf("ConvertWith: %v", err)
	}
	if !strings.Contains(out.String(), "Style: Custom,") {
		t.Fatalf("the style mapping is missing:\n%s", out.String())
	}
	if strings.Contains(out.String(), "Style: Default,") {
		t.Fatalf("the source style name must not survive:\n%s", out.String())
	}
}

func TestConvertWithFontOverride(t *testing.T) {
	var out strings.Builder
	if _, err := ConvertWith("in.ass", strings.NewReader(assSample), Options{
		Target: "ass",
		Font:   "Verdana",
	}, &out); err != nil {
		t.Fatalf("ConvertWith: %v", err)
	}
	if !strings.Contains(out.String(), "Verdana") {
		t.Fatalf("the font override is missing:\n%s", out.String())
	}
}

func TestConvertWithLossStrict(t *testing.T) {
	var out strings.Builder
	losses, err := ConvertWith("in.srt", strings.NewReader(srtSample), Options{
		Target: "sbv",
		Loss:   LossStrict,
	}, &out)
	if err == nil {
		t.Fatal("LossStrict must fail when the target drops a feature")
	}
	if len(losses) == 0 {
		t.Fatal("LossStrict must still return the loss report")
	}
}

func TestConvertWithLossSilent(t *testing.T) {
	var out strings.Builder
	losses, err := ConvertWith("in.srt", strings.NewReader(srtSample), Options{
		Target: "sbv",
		Loss:   LossSilent,
	}, &out)
	if err != nil {
		t.Fatalf("ConvertWith: %v", err)
	}
	if losses != nil {
		t.Fatalf("LossSilent must drop the report: %v", losses)
	}
}

func TestConvertWithNoTarget(t *testing.T) {
	if _, err := ConvertWith("in.srt", strings.NewReader(srtSample), Options{}, io.Discard); err == nil {
		t.Fatal("a missing target must fail")
	}
}

func TestConvertWithParseError(t *testing.T) {
	boom := errors.New("boom")
	if _, err := ConvertWith("in.srt", errReader{boom}, Options{Target: "sbv"}, io.Discard); !errors.Is(err, boom) {
		t.Fatalf("ConvertWith must return the read error, got %v", err)
	}
}

func TestConvertWithFormatOverride(t *testing.T) {
	var out strings.Builder
	losses, err := ConvertWith("in.unknown", strings.NewReader(srtSample), Options{
		Format: "srt",
		Target: "sbv",
	}, &out)
	if err != nil {
		t.Fatalf("ConvertWith: %v", err)
	}
	if !strings.Contains(out.String(), "Hello from the public API.") {
		t.Fatalf("the text is missing:\n%s", out.String())
	}
	if len(losses) == 0 {
		t.Fatal("the default policy must return the loss report")
	}
}
