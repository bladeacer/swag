package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bladeacer/swag/internal/i18n"
	"github.com/bladeacer/swag/pkg/sub"
)

const srtFixture = "1\n00:00:00,000 --> 00:00:01,000\nhello\n"

// voiceVTTFixture carries a voice span and inline styling, so a conversion
// exercises the speaker name and the style tags at once.
const voiceVTTFixture = "WEBVTT\n\n00:00:01.000 --> 00:00:04.000\n<v Roger Bingham><i>We are</i> in the Milky Way.</v>\n"

// assKaraokeFixture carries karaoke, which SRT cannot hold, so a convert
// to SRT reports a loss.
const assKaraokeFixture = `[Script Info]
ScriptType: v4.00+
PlayResX: 1280
PlayResY: 720

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,Arial,20,&H00FFFFFF,&H00777777,&H00000000,&H64000000,0,0,0,0,100,100,0,0,1,2,2,2,10,10,10,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
Dialogue: 0,0:00:01.00,0:00:06.00,Default,,0,0,0,,{\k50}ka{\k50}ra
`

// writeSubtitle writes a fixture file and returns its path.
func writeSubtitle(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

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
	tr := i18n.New("en-GB")
	if got := targetName(&ConvertCmd{}, tr); got != "?" {
		t.Fatalf("targetName = %q, want ?", got)
	}
	if got := targetName(&ConvertCmd{Output: "x.sbv"}, tr); got != "sbv" {
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
	sink, closer, err := openOutput("", i18n.New("en-GB"))
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
	sink, closer, err := openOutput(path, i18n.New("en-GB"))
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
	path := filepath.Join(t.TempDir(), "no-such-dir", "x.sbv")
	_, _, err := openOutput(path, i18n.New("en-GB"))
	if err == nil {
		t.Fatal("unwritable output path must fail")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("the error must name the output file: %v", err)
	}
}

func TestOutputLabel(t *testing.T) {
	tr := i18n.New("en-GB")
	if got := outputLabel("", tr); got != "standard output" {
		t.Fatalf("outputLabel(\"\") = %q", got)
	}
	if got := outputLabel("a.sbv", tr); got != "a.sbv" {
		t.Fatalf("outputLabel = %q", got)
	}
}

func TestBannerDoesNotPanic(t *testing.T) {
	banner(i18n.New("en-GB"))
}

func TestTargetNameFromFormat(t *testing.T) {
	if got := targetName(&ConvertCmd{Format: "ass", Output: "x.srt"}, i18n.New("en-GB")); got != "ass" {
		t.Fatalf("targetName = %q, want ass (the -f flag wins)", got)
	}
}

// newRunContext returns the runContext a kong run would inject.
func newRunContext(verbose bool) *runContext {
	return &runContext{CLI: &CLI{Verbose: verbose}, T: i18n.New("en-GB")}
}

func TestConvertCmdRunWritesFile(t *testing.T) {
	in := writeSubtitle(t, "in.srt", srtFixture)
	out := filepath.Join(t.TempDir(), "out.sbv")
	c := &ConvertCmd{Input: in, Output: out}
	if err := c.Run(newRunContext(false)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(data), "hello") {
		t.Fatalf("output lacks the cue text: %s", data)
	}
}

func TestConvertCmdRunVerboseReportsLosses(t *testing.T) {
	in := writeSubtitle(t, "in.ass", assKaraokeFixture)
	out := filepath.Join(t.TempDir(), "out.srt")
	c := &ConvertCmd{Input: in, Output: out, Verbose: true}
	if err := c.Run(newRunContext(false)); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestConvertCmdRunCLIVerboseReportsLosses(t *testing.T) {
	in := writeSubtitle(t, "in.ass", assKaraokeFixture)
	out := filepath.Join(t.TempDir(), "out.srt")
	c := &ConvertCmd{Input: in, Output: out}
	if err := c.Run(newRunContext(true)); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestConvertCmdRunInputMissing(t *testing.T) {
	c := &ConvertCmd{Input: filepath.Join(t.TempDir(), "nope.srt"), Output: "x.sbv"}
	if err := c.Run(newRunContext(false)); err == nil {
		t.Fatal("a missing input must fail")
	}
}

func TestConvertCmdRunOpenFails(t *testing.T) {
	in := writeSubtitle(t, "in.srt", srtFixture)
	original := openInput
	openInput = func(string) (io.ReadCloser, error) { return nil, errors.New("denied") }
	t.Cleanup(func() { openInput = original })

	c := &ConvertCmd{Input: in, Output: filepath.Join(t.TempDir(), "out.sbv")}
	if err := c.Run(newRunContext(false)); err == nil {
		t.Fatal("an unreadable input must fail")
	}
}

func TestConvertCmdRunParseFails(t *testing.T) {
	in := writeSubtitle(t, "in.srt", "1\n00:00:0,000 --> broken\nx\n")
	c := &ConvertCmd{Input: in, Output: filepath.Join(t.TempDir(), "out.sbv")}
	if err := c.Run(newRunContext(false)); err == nil {
		t.Fatal("unparsable input must fail")
	}
}

func TestConvertCmdRunTargetUnknown(t *testing.T) {
	in := writeSubtitle(t, "in.srt", srtFixture)
	c := &ConvertCmd{Input: in, Output: filepath.Join(t.TempDir(), "out.txt")}
	if err := c.Run(newRunContext(false)); err == nil {
		t.Fatal("an unknown output extension must fail")
	}
}

func TestConvertCmdRunOutputFails(t *testing.T) {
	in := writeSubtitle(t, "in.srt", srtFixture)
	c := &ConvertCmd{Input: in, Format: "srt", Output: filepath.Join(t.TempDir(), "no-such-dir", "out.srt")}
	if err := c.Run(newRunContext(false)); err == nil {
		t.Fatal("an unwritable output must fail")
	}
}

func TestConvertCmdRunRenderFails(t *testing.T) {
	in := writeSubtitle(t, "in.srt", srtFixture)
	c := &ConvertCmd{Input: in, Format: "bogus", Output: filepath.Join(t.TempDir(), "out.bin")}
	if err := c.Run(newRunContext(false)); err == nil {
		t.Fatal("an unknown target format must fail")
	}
}

func TestRunConverts(t *testing.T) {
	in := writeSubtitle(t, "in.srt", srtFixture)
	out := filepath.Join(t.TempDir(), "out.sbv")
	if code := run([]string{"convert", "-i", in, "-o", out}); code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("output missing: %v", err)
	}
}

func TestRunRejectsBadFlag(t *testing.T) {
	if code := run([]string{"--bogus-flag"}); code != 1 {
		t.Fatalf("run exit code = %d, want 1", code)
	}
}

func TestRunReportsConvertFailure(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.srt")
	if code := run([]string{"convert", "-i", missing, "-o", "out.sbv"}); code != 1 {
		t.Fatalf("run exit code = %d, want 1", code)
	}
}

func TestMainTakesRunCode(t *testing.T) {
	originalExit := exit
	originalArgs := os.Args
	code := -1
	exit = func(c int) { code = c }
	t.Cleanup(func() {
		exit = originalExit
		os.Args = originalArgs
	})

	in := writeSubtitle(t, "in.srt", srtFixture)
	out := filepath.Join(t.TempDir(), "out.sbv")
	os.Args = []string{"swag", "convert", "-i", in, "-o", out}
	main()
	if code != 0 {
		t.Fatalf("main exit code = %d, want 0", code)
	}
}

func TestConvertCmdRunFontOverride(t *testing.T) {
	in := writeSubtitle(t, "in.ass", assKaraokeFixture)
	out := filepath.Join(t.TempDir(), "out.ass")
	c := &ConvertCmd{Input: in, Output: out, Font: "Verdana"}
	if err := c.Run(newRunContext(false)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(data), "Verdana") {
		t.Fatalf("the font override is missing:\n%s", data)
	}
}

func TestConvertCmdRunStrictFails(t *testing.T) {
	in := writeSubtitle(t, "in.ass", assKaraokeFixture)
	out := filepath.Join(t.TempDir(), "out.srt")
	c := &ConvertCmd{Input: in, Output: out, Strict: true}
	if err := c.Run(newRunContext(false)); err == nil {
		t.Fatal("--strict must fail when the target drops a feature")
	}
}

// TestBareRunArgs covers the shapes that carry no work: a bare run under
// air, and the separator that `make run` passes.
func TestBareRunArgs(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{nil, true},
		{[]string{"--"}, true},
		{[]string{"convert"}, false},
		{[]string{"-i", "in.srt"}, false},
	}
	for _, tt := range tests {
		if got := bareRun(tt.args); got != tt.want {
			t.Errorf("bareRun(%v) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

// TestBareRunExitsZero keeps `air` and `make run` working. A run with no
// arguments shows the banner and the first step, and it must not fail.
func TestBareRunExitsZero(t *testing.T) {
	for _, args := range [][]string{nil, {"--"}} {
		if code := run(args); code != 0 {
			t.Errorf("run(%v) exit code = %d, want 0", args, code)
		}
	}
}

// TestRunWithoutTheCommandWord covers the default command. Its flags stand
// at the top level, so the command word is optional for a full conversion.
func TestRunWithoutTheCommandWord(t *testing.T) {
	in := writeSubtitle(t, "in.srt", srtFixture)
	out := filepath.Join(t.TempDir(), "out.vtt")
	if code := run([]string{"-i", in, "-o", out, "-v"}); code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.HasPrefix(string(data), "WEBVTT") {
		t.Fatalf("output must be WebVTT:\n%s", data)
	}
}

// TestRunConvertsEveryFormat walks the command line through every registered
// target and back to WebVTT, so the wiring of each format works end to end.
func TestRunConvertsEveryFormat(t *testing.T) {
	in := writeSubtitle(t, "in.vtt", voiceVTTFixture)
	for _, name := range sub.Registered() {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			mid := filepath.Join(dir, "mid."+name)
			if code := run([]string{"convert", "-i", in, "-o", mid}); code != 0 {
				t.Fatalf("convert to %s exit code = %d, want 0", name, code)
			}
			back := filepath.Join(dir, "back.vtt")
			if code := run([]string{"convert", "-i", mid, "-o", back}); code != 0 {
				t.Fatalf("convert %s back exit code = %d, want 0", name, code)
			}
			data, err := os.ReadFile(back)
			if err != nil {
				t.Fatalf("read the output: %v", err)
			}
			if !strings.HasPrefix(string(data), "WEBVTT") {
				t.Fatalf("the second leg must write WebVTT:\n%s", data)
			}
			if !strings.Contains(string(data), "Milky") {
				t.Fatalf("the cue text is missing after the %s pass:\n%s", name, data)
			}
		})
	}
}

func TestRunReportsADirectoryInput(t *testing.T) {
	dir := t.TempDir()
	if code := run([]string{"convert", "-i", dir, "-o", filepath.Join(dir, "out.sbv")}); code != 1 {
		t.Fatalf("run exit code = %d, want 1", code)
	}
}

func TestRunAcceptsALocaleFlag(t *testing.T) {
	t.Setenv("SWAG_LOCALE", "fr-FR")
	in := writeSubtitle(t, "in.srt", srtFixture)
	out := filepath.Join(t.TempDir(), "out.sbv")
	if code := run([]string{"--locale", "fr-FR", "convert", "-i", in, "-o", out}); code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
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
