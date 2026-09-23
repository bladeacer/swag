package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bladeacer/swag/internal/i18n"
)

func TestResolveTargetFromFormatFlag(t *testing.T) {
	c := ConvertCmd{Format: "sbv", Output: "out.srt"}
	got, err := resolveTarget(&c, i18n.New("en-GB"))
	if err != nil {
		t.Fatalf("resolveTarget: %v", err)
	}
	if got != "sbv" {
		t.Fatalf("resolveTarget = %q, want sbv (the -f flag wins)", got)
	}
}

func TestResolveTargetFromOutputExtension(t *testing.T) {
	c := ConvertCmd{Output: "out.srt"}
	got, err := resolveTarget(&c, i18n.New("en-GB"))
	if err != nil {
		t.Fatalf("resolveTarget: %v", err)
	}
	if got != "srt" {
		t.Fatalf("resolveTarget = %q, want srt", got)
	}
}

func TestResolveTargetUnknownExtension(t *testing.T) {
	c := ConvertCmd{Output: "out.txt"}
	if _, err := resolveTarget(&c, i18n.New("en-GB")); err == nil {
		t.Fatal("unknown output extension must fail")
	}
}

func TestResolveTargetNothingGiven(t *testing.T) {
	c := ConvertCmd{}
	if _, err := resolveTarget(&c, i18n.New("en-GB")); err == nil {
		t.Fatal("missing -o and -f must fail")
	}
}

func TestTargetNameFallback(t *testing.T) {
	if got := targetName(&ConvertCmd{}); got != "?" {
		t.Fatalf("targetName = %q, want ?", got)
	}
	if got := targetName(&ConvertCmd{Output: "x.sbv"}); got != "sbv" {
		t.Fatalf("targetName = %q, want sbv", got)
	}
}

func TestCheckInput(t *testing.T) {
	tr := i18n.New("en-GB")
	if err := checkInput(filepath.Join(t.TempDir(), "missing.srt"), tr); err == nil {
		t.Fatal("missing file must fail")
	}
	dir := t.TempDir()
	if err := checkInput(dir, tr); err == nil {
		t.Fatal("directory input must fail")
	}
	f := filepath.Join(t.TempDir(), "ok.srt")
	if err := os.WriteFile(f, []byte("1\n00:00:00,000 --> 00:00:01,000\nx\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := checkInput(f, tr); err != nil {
		t.Fatalf("existing file must pass: %v", err)
	}
}

func TestOpenOutputStdout(t *testing.T) {
	sink, closer, err := openOutput("")
	if err != nil {
		t.Fatalf("openOutput: %v", err)
	}
	defer closer()
	if sink != os.Stdout {
		t.Fatal("empty -o must write to stdout")
	}
}

func TestOpenOutputFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.sbv")
	sink, closer, err := openOutput(path)
	if err != nil {
		t.Fatalf("openOutput: %v", err)
	}
	closer()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	if sink == nil {
		t.Fatal("sink must not be nil")
	}
}

func TestOpenOutputBadPath(t *testing.T) {
	if _, _, err := openOutput(filepath.Join(t.TempDir(), "no-such-dir", "x.sbv")); err == nil {
		t.Fatal("unwritable output path must fail")
	}
}

func TestOutputLabel(t *testing.T) {
	if got := outputLabel(""); got != "standard output" {
		t.Fatalf("outputLabel(\"\") = %q", got)
	}
	if got := outputLabel("a.sbv"); got != "a.sbv" {
		t.Fatalf("outputLabel = %q", got)
	}
}

func TestBannerDoesNotPanic(t *testing.T) {
	banner(i18n.New("en-GB"))
}

func TestRunContextCarriesLocale(t *testing.T) {
	ictx := &runContext{CLI: &CLI{Locale: "en-GB"}, T: i18n.New("en-GB")}
	if ictx.T.Locale() != i18n.DefaultLocale {
		t.Fatalf("locale not carried: %q", ictx.T.Locale())
	}
	// The error path formats through the catalogue.
	msg := ictx.T.F(i18n.MsgInputUnreadable, errors.New("x"))
	if msg == "" {
		t.Fatal("message must not be empty")
	}
}
