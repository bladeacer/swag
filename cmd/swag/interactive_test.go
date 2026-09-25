// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bladeacer/swag/internal/i18n"
	"github.com/bladeacer/swag/internal/tui"
)

// scriptInteractive replaces the two streams of the terminal commands with
// a scripted reader and a buffer, and restores them afterwards.
func scriptInteractive(t *testing.T, script string, out io.Writer) {
	t.Helper()
	originalIn, originalOut := terminalIn, terminalOut
	terminalIn, terminalOut = strings.NewReader(script), out
	t.Cleanup(func() { terminalIn, terminalOut = originalIn, originalOut })
}

// stubPrompter returns fixed answers and writes nothing.
type stubPrompter struct {
	text   string
	choice string
}

func (s stubPrompter) ask(string, string) (string, error) { return s.text, nil }
func (s stubPrompter) askChoice(string, []string, string) (string, error) {
	return s.choice, nil
}

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

// brokenReadCloser fails every read.
type brokenReadCloser struct{}

func (brokenReadCloser) Read([]byte) (int, error) { return 0, errors.New("boom") }
func (brokenReadCloser) Close() error             { return nil }

func TestSwapExtension(t *testing.T) {
	if got := swapExtension("/tmp/in.ass", "vtt"); got != "/tmp/in.vtt" {
		t.Errorf("swapExtension = %q, want /tmp/in.vtt", got)
	}
	if got := swapExtension("/tmp/in", "srt"); got != "/tmp/in.srt" {
		t.Errorf("swapExtension without an extension = %q", got)
	}
}

func TestLinePrompterAsk(t *testing.T) {
	var out strings.Builder
	p := newPrompter(strings.NewReader("chosen.ass\n"), &out, i18n.New("en-GB")).(*linePrompter)
	answer, err := p.ask("Input file", "default.ass")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if answer != "chosen.ass" {
		t.Fatalf("ask = %q, want the answer", answer)
	}
	if !strings.Contains(out.String(), "default.ass") {
		t.Fatalf("the prompt must name the default: %q", out.String())
	}

	// An empty answer takes the default.
	out.Reset()
	p = newPrompter(strings.NewReader("\n"), &out, i18n.New("en-GB")).(*linePrompter)
	answer, err = p.ask("Input file", "default.ass")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if answer != "default.ass" {
		t.Fatalf("ask = %q, want the default", answer)
	}
}

// TestLinePrompterAskWithoutDefault covers the bare prompt, which carries no
// default value.
func TestLinePrompterAskWithoutDefault(t *testing.T) {
	var out strings.Builder
	p := newPrompter(strings.NewReader("in.ass\n"), &out, i18n.New("en-GB")).(*linePrompter)
	answer, err := p.ask("Input file", "")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if answer != "in.ass" {
		t.Fatalf("ask = %q", answer)
	}
	if strings.Contains(out.String(), "[]") {
		t.Fatalf("a bare prompt must carry no empty default: %q", out.String())
	}
	// An empty answer to a bare prompt returns the empty default.
	p = newPrompter(strings.NewReader("\n"), &out, i18n.New("en-GB")).(*linePrompter)
	answer, err = p.ask("Input file", "")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if answer != "" {
		t.Fatalf("ask = %q, want the empty default", answer)
	}
}

func TestLinePrompterAskReadError(t *testing.T) {
	var out strings.Builder
	p := newPrompter(strings.NewReader(""), &out, i18n.New("en-GB")).(*linePrompter)
	if _, err := p.ask("Input file", "default"); err == nil {
		t.Fatal("an empty input must fail the read")
	}
}

func TestLinePrompterAskWriteError(t *testing.T) {
	p := newPrompter(strings.NewReader("x\n"), &failAtWriter{failAt: 1}, i18n.New("en-GB")).(*linePrompter)
	if _, err := p.ask("Input file", "default"); err == nil {
		t.Fatal("a failing sink must surface the error")
	}
}

func TestLinePrompterAskChoice(t *testing.T) {
	options := []string{"ass", "srt", "vtt"}
	var out strings.Builder

	p := newPrompter(strings.NewReader("2\n"), &out, i18n.New("en-GB")).(*linePrompter)
	answer, err := p.askChoice("Target format", options, "ass")
	if err != nil {
		t.Fatalf("askChoice: %v", err)
	}
	if answer != "srt" {
		t.Fatalf("askChoice = %q, want the second option", answer)
	}
	if !strings.Contains(out.String(), "3) vtt") {
		t.Fatalf("the options must be listed: %q", out.String())
	}

	// A name picks the option too.
	p = newPrompter(strings.NewReader("VTT\n"), &out, i18n.New("en-GB")).(*linePrompter)
	if answer, err = p.askChoice("Target format", options, "ass"); err != nil || answer != "vtt" {
		t.Fatalf("askChoice by name = %q, %v", answer, err)
	}

	// An empty answer takes the default.
	p = newPrompter(strings.NewReader("\n"), &out, i18n.New("en-GB")).(*linePrompter)
	if answer, err = p.askChoice("Target format", options, "ass"); err != nil || answer != "ass" {
		t.Fatalf("askChoice default = %q, %v", answer, err)
	}
}

func TestLinePrompterAskChoiceErrors(t *testing.T) {
	options := []string{"ass", "srt"}
	var out strings.Builder

	// A number outside the list is refused.
	p := newPrompter(strings.NewReader("9\n"), &out, i18n.New("en-GB")).(*linePrompter)
	if _, err := p.askChoice("Target format", options, "ass"); err == nil {
		t.Fatal("an out-of-range number must fail")
	}

	// So is an unknown name.
	p = newPrompter(strings.NewReader("docx\n"), &out, i18n.New("en-GB")).(*linePrompter)
	if _, err := p.askChoice("Target format", options, "ass"); err == nil {
		t.Fatal("an unknown name must fail")
	}

	// A locale catalogue with no options has nothing to choose.
	p = newPrompter(strings.NewReader("\n"), &out, i18n.New("en-GB")).(*linePrompter)
	if _, err := p.askChoice("Target format", nil, "ass"); err == nil {
		t.Fatal("an empty option list must fail")
	}

	// A read error surfaces.
	p = newPrompter(strings.NewReader(""), &out, i18n.New("en-GB")).(*linePrompter)
	if _, err := p.askChoice("Target format", options, "ass"); err == nil {
		t.Fatal("an empty input must fail the read")
	}

	// So does a write error on every line of the question: the label, an
	// option, and the pick line.
	for _, failAt := range []int{1, 2, len(options) + 2} {
		p = newPrompter(strings.NewReader("1\n"), &failAtWriter{failAt: failAt}, i18n.New("en-GB")).(*linePrompter)
		if _, err := p.askChoice("Target format", options, "ass"); err == nil {
			t.Errorf("a sink that fails on write %d must surface the error", failAt)
		}
	}
}

// TestLinePrompterAskChoiceWithoutDefault covers the pick line of a
// question with no default, which carries no value to name.
func TestLinePrompterAskChoiceWithoutDefault(t *testing.T) {
	var out strings.Builder
	p := newPrompter(strings.NewReader("1\n"), &out, i18n.New("en-GB")).(*linePrompter)
	answer, err := p.askChoice("Target format", []string{"ass", "srt"}, "")
	if err != nil {
		t.Fatalf("askChoice: %v", err)
	}
	if answer != "ass" {
		t.Fatalf("askChoice = %q, want the first option", answer)
	}
	if strings.Contains(out.String(), "Enter for") {
		t.Fatalf("a question with no default must not promise one:\n%s", out.String())
	}
}

// TestLinePrompterLastLineWithoutNewline covers an answer that ends at the
// end of the input without a newline.
func TestLinePrompterLastLineWithoutNewline(t *testing.T) {
	var out strings.Builder
	p := newPrompter(strings.NewReader("vtt"), &out, i18n.New("en-GB")).(*linePrompter)
	answer, err := p.ask("Target format", "")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if answer != "vtt" {
		t.Fatalf("ask = %q, want the last line", answer)
	}
}

func TestLayouts(t *testing.T) {
	tr := i18n.New("en-GB")
	plain := resultLayout(tr, "in.ass", "ass", "vtt", "in.vtt", nil)
	if len(plain.Rows) != 3 {
		t.Fatalf("rows = %v, want three", plain.Rows)
	}
	if !strings.Contains(plain.Status, "in.vtt") {
		t.Fatalf("the status must name the output: %q", plain.Status)
	}
	lossy := resultLayout(tr, "in.ass", "ass", "srt", "in.srt", []string{"karaoke timing"})
	if len(lossy.Rows) != 4 {
		t.Fatalf("rows = %v, want the loss row as well", lossy.Rows)
	}
	if !strings.Contains(lossy.Status, "1") {
		t.Fatalf("the status must count the losses: %q", lossy.Status)
	}
	if rows := tui.Changed(plain, lossy); len(rows) == 0 {
		t.Fatal("a lossy result must change the frame")
	}
}

// TestInteractiveCmdRunWithoutPrompts covers the whole path when every
// choice comes from a flag, so nothing is asked.
func TestInteractiveCmdRunWithoutPrompts(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.srt")
	if err := os.WriteFile(in, []byte(srtFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	out := filepath.Join(dir, "out.vtt")
	var screen strings.Builder
	scriptInteractive(t, "", &screen)

	c := &InteractiveCmd{Input: in, Target: "vtt", Output: out}
	if err := c.Run(newRunContext(false)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.HasPrefix(string(data), "WEBVTT") {
		t.Fatalf("output must be WebVTT:\n%s", data)
	}
	// The frame names the conversion and the outcome.
	for _, want := range []string{"swag interactive", "Wrote", "vtt"} {
		if !strings.Contains(screen.String(), want) {
			t.Errorf("the frame is missing %q:\n%s", want, screen.String())
		}
	}
}

// TestInteractiveCmdRunAsksForEverything covers the prompts, with the input
// answer, a target chosen by number, and an empty output answer that takes
// the default beside the input.
func TestInteractiveCmdRunAsksForEverything(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.srt")
	if err := os.WriteFile(in, []byte(srtFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	var screen strings.Builder
	// The input answer, the number of the first registered format, and an
	// empty line for the default output.
	scriptInteractive(t, in+"\n1\n\n", &screen)

	c := &InteractiveCmd{}
	if err := c.Run(newRunContext(false)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	out := filepath.Join(dir, "in.ass")
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("the default output must sit beside the input: %v", err)
	}
	if !strings.Contains(screen.String(), "Target format") {
		t.Fatalf("the target question is missing:\n%s", screen.String())
	}
}

// TestInteractiveCmdRunLosses covers a lossy conversion, whose second frame
// adds the loss rows.
func TestInteractiveCmdRunLosses(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.ass")
	if err := os.WriteFile(in, []byte(assKaraokeFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	out := filepath.Join(dir, "out.srt")
	var screen strings.Builder
	scriptInteractive(t, "", &screen)

	c := &InteractiveCmd{Input: in, Target: "srt", Output: out}
	if err := c.Run(newRunContext(false)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(screen.String(), "karaoke timing") {
		t.Fatalf("the frame must carry the loss:\n%s", screen.String())
	}
}

func TestInteractiveCmdRunStrictFails(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.ass")
	if err := os.WriteFile(in, []byte(assKaraokeFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	var screen strings.Builder
	scriptInteractive(t, "", &screen)

	c := &InteractiveCmd{Input: in, Target: "srt", Output: filepath.Join(dir, "out.srt"), Strict: true}
	if err := c.Run(newRunContext(false)); err == nil {
		t.Fatal("a strict conversion that loses a feature must fail")
	}
}

func TestInteractiveCmdRunErrors(t *testing.T) {
	tr := newRunContext(false)
	dir := t.TempDir()
	good := filepath.Join(dir, "in.srt")
	if err := os.WriteFile(good, []byte(srtFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	tests := []struct {
		name string
		cmd  InteractiveCmd
	}{
		{"missing input", InteractiveCmd{Input: filepath.Join(dir, "nope.srt"), Target: "vtt"}},
		{"bad target", InteractiveCmd{Input: good, Target: "bogus", Output: filepath.Join(dir, "o.bin")}},
		{"bad output", InteractiveCmd{Input: good, Target: "vtt", Output: filepath.Join(dir, "no", "o.vtt")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var screen strings.Builder
			scriptInteractive(t, "", &screen)
			cmd := tt.cmd
			if err := cmd.Run(tr); err == nil {
				t.Errorf("%s must fail", tt.name)
			}
		})
	}
}

// TestInteractiveCmdRunUnknownContent covers a file that exists but carries
// no signature the tool knows, so the target question arrives with no
// default to offer.
func TestInteractiveCmdRunUnknownContent(t *testing.T) {
	dir := t.TempDir()
	odd := filepath.Join(dir, "in.txt")
	if err := os.WriteFile(odd, []byte("not a subtitle\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	out := filepath.Join(dir, "out.srt")
	var screen strings.Builder
	// The target answer by name, then the output answer. The input format
	// comes from the flag, because the content carries no signature.
	scriptInteractive(t, "srt\n"+out+"\n", &screen)

	cmd := &InteractiveCmd{Input: odd, From: "srt"}
	if err := cmd.Run(newRunContext(false)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(screen.String(), "Target format") {
		t.Fatalf("the target question is missing:\n%s", screen.String())
	}
	if !strings.Contains(screen.String(), "Choose a number:") {
		t.Fatalf("a question with no default must not promise one:\n%s", screen.String())
	}
}

// TestInteractiveCmdRunBadInputFormat covers a named input format that the
// registry cannot read, so the parse fails before the first frame.
func TestInteractiveCmdRunBadInputFormat(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.srt")
	if err := os.WriteFile(in, []byte(srtFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	var screen strings.Builder
	scriptInteractive(t, "", &screen)

	cmd := &InteractiveCmd{Input: in, From: "bogus", Target: "vtt", Output: filepath.Join(dir, "out.vtt")}
	if err := cmd.Run(newRunContext(false)); err == nil {
		t.Fatal("an unknown input format must fail")
	}
}

// TestInteractiveCmdRunReadErrors covers a failing open, a failing read, and
// a failing identification of the input file. Each case needs a real file,
// because the input check runs first.
func TestInteractiveCmdRunReadErrors(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "in.srt")
	if err := os.WriteFile(good, []byte(srtFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	originalOpen, originalIdentify := openInput, identify
	t.Cleanup(func() { openInput, identify = originalOpen, originalIdentify })
	tr := newRunContext(false)

	var screen strings.Builder
	scriptInteractive(t, "", &screen)

	openInput = func(string) (io.ReadCloser, error) { return nil, errors.New("denied") }
	if err := (&InteractiveCmd{Input: good, Target: "vtt"}).Run(tr); err == nil {
		t.Fatal("a failed open must fail the command")
	}

	openInput = func(string) (io.ReadCloser, error) { return brokenReadCloser{}, nil }
	if err := (&InteractiveCmd{Input: good, Target: "vtt"}).Run(tr); err == nil {
		t.Fatal("a failed read must fail the command")
	}

	openInput = originalOpen
	identify = func(string, io.Reader) (string, error) { return "", errors.New("boom") }
	if err := (&InteractiveCmd{Input: good, Target: "vtt"}).Run(tr); err == nil {
		t.Fatal("a failed identification must fail the command")
	}
	identify = originalIdentify

	// The round trip back to the real identification still works, and the
	// font override reaches the conversion.
	out := filepath.Join(dir, "out.vtt")
	if err := (&InteractiveCmd{Input: good, Target: "vtt", Output: out, Font: "Courier"}).Run(tr); err != nil {
		t.Fatalf("Run with a font override: %v", err)
	}
}

// TestInteractiveCmdRunPromptErrors covers each question that fails before
// an answer arrives.
func TestInteractiveCmdRunPromptErrors(t *testing.T) {
	tr := newRunContext(false)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.srt")
	if err := os.WriteFile(in, []byte(srtFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var screen strings.Builder
	// The input question fails on an empty reader.
	scriptInteractive(t, "", &screen)
	if err := (&InteractiveCmd{}).Run(tr); err == nil {
		t.Fatal("a failed input question must fail the command")
	}
	// The target question fails on an empty reader.
	scriptInteractive(t, "", &screen)
	if err := (&InteractiveCmd{Input: in}).Run(tr); err == nil {
		t.Fatal("a failed target question must fail the command")
	}
	// The output question fails on an empty reader.
	scriptInteractive(t, "", &screen)
	if err := (&InteractiveCmd{Input: in, Target: "vtt"}).Run(tr); err == nil {
		t.Fatal("a failed output question must fail the command")
	}
}

// TestInteractiveCmdRunDrawErrors covers a failing screen on the first frame
// and on the second one.
func TestInteractiveCmdRunDrawErrors(t *testing.T) {
	tr := newRunContext(false)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.srt")
	if err := os.WriteFile(in, []byte(srtFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	for _, failAt := range []int{1, 2} {
		original := terminalOut
		terminalOut = &failAtWriter{failAt: failAt}
		cmd := &InteractiveCmd{Input: in, Target: "vtt", Output: filepath.Join(dir, "out.vtt")}
		err := cmd.Run(tr)
		terminalOut = original
		if err == nil {
			t.Errorf("a screen that fails on write %d must fail the command", failAt)
		}
	}
}

// TestRunInteractiveCommand drives the command through the parser, so the
// command word and its shorthands are covered.
func TestRunInteractiveCommand(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.srt")
	if err := os.WriteFile(in, []byte(srtFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	out := filepath.Join(dir, "out.vtt")
	var screen strings.Builder
	scriptInteractive(t, "", &screen)

	code := run([]string{"interactive", "-i", in, "-f", "vtt", "-o", out})
	if code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("output missing: %v", err)
	}
}

// TestRunInteractiveAsksForInput covers the interactive command entered
// with no flags at all, so the first question appears.
func TestRunInteractiveAsksForInput(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.srt")
	if err := os.WriteFile(in, []byte(srtFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	out := filepath.Join(dir, "out.vtt")
	var screen strings.Builder
	scriptInteractive(t, in+"\nvtt\n"+out+"\n", &screen)

	code := run([]string{"interactive"})
	if code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("output missing: %v", err)
	}
	if !strings.Contains(screen.String(), "Input file") {
		t.Fatalf("the input question is missing:\n%s", screen.String())
	}
}

// TestOrderFormats covers the target order of the interactive picker: the
// preferred formats first, then the rest of the registry.
func TestOrderFormats(t *testing.T) {
	all := []string{"ass", "json1", "kdenlive", "sbv", "srt", "ttml", "vtt", "ytt"}
	got := orderFormats([]string{"vtt", "srt", "bogus", "vtt"}, all)
	want := []string{"vtt", "srt", "ass", "json1", "kdenlive", "sbv", "ttml", "ytt"}
	if len(got) != len(want) {
		t.Fatalf("orderFormats = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("orderFormats = %v, want %v", got, want)
		}
	}
	// An empty list keeps the registry order.
	if got := orderFormats(nil, all); len(got) != len(all) || got[0] != all[0] {
		t.Fatalf("an empty preferred list = %v, want %v", got, all)
	}
}

// TestRunInteractiveStrictCompat proves the flag reaches the write of the
// interactive command.
func TestRunInteractiveStrictCompat(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.ass")
	if err := os.WriteFile(in, []byte(assKaraokeFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	out := filepath.Join(dir, "out.srt")
	var screen strings.Builder
	scriptInteractive(t, "", &screen)

	code := run([]string{"interactive", "-i", in, "-f", "srt", "-o", out, "--strict-compat"})
	if code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if strings.Contains(string(data), "NOTE swag-ir") {
		t.Errorf("a strict interactive write must carry no block:\n%s", data)
	}
}

// TestInteractiveHelpPage covers the help page of the new command.
func TestInteractiveHelpPage(t *testing.T) {
	if code := run([]string{"interactive", "--help"}); code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
}
