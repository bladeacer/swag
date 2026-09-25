// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pterm/pterm"

	"github.com/bladeacer/swag/internal/i18n"
	"github.com/bladeacer/swag/internal/model"
)

func TestTerminalColour(t *testing.T) {
	originalColor, originalRaw := pterm.PrintColor, pterm.RawOutput
	t.Cleanup(func() { pterm.PrintColor, pterm.RawOutput = originalColor, originalRaw })

	pterm.PrintColor, pterm.RawOutput = true, false
	if !terminalColour() {
		t.Error("a plain terminal carries colour")
	}
	pterm.PrintColor = false
	if terminalColour() {
		t.Error("a terminal with colour off carries none")
	}
	pterm.PrintColor, pterm.RawOutput = true, true
	if terminalColour() {
		t.Error("a raw terminal carries no colour")
	}
}

func TestPreviewRows(t *testing.T) {
	tr := i18n.New("en-GB")

	empty := previewRows(tr, &model.Document{}, 10, false)
	joined := strings.Join(empty, "\n")
	if !strings.Contains(joined, tr.S(i18n.MsgPreviewNoStyles)) {
		t.Errorf("an empty document needs the style note:\n%s", joined)
	}
	if !strings.Contains(joined, tr.S(i18n.MsgPreviewNoCues)) {
		t.Errorf("an empty document needs the cue note:\n%s", joined)
	}

	doc := &model.Document{
		Styles: []model.Style{{Name: "Default", Primary: model.NewColour(255, 255, 255, 255)}},
		Cues: []model.Cue{
			{Start: 0, End: time.Second, Spans: []model.TextSpan{{Text: "one"}}},
			{Start: time.Second, End: 2 * time.Second, Spans: []model.TextSpan{{Text: "two"}}},
			{Start: 2 * time.Second, End: 3 * time.Second, Spans: []model.TextSpan{{Text: "three"}}},
		},
	}
	rows := previewRows(tr, doc, 1, true)
	joined = strings.Join(rows, "\n")
	if !strings.Contains(joined, "Default") {
		t.Errorf("the style row is missing:\n%s", joined)
	}
	if !strings.Contains(joined, "one") || strings.Contains(joined, "two") {
		t.Errorf("the limit did not apply:\n%s", joined)
	}
	if !strings.Contains(joined, tr.F(i18n.MsgPreviewMore, 2)) {
		t.Errorf("the summary row is missing:\n%s", joined)
	}
}

func TestPreviewCmdRun(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.ass")
	if err := os.WriteFile(in, []byte(assKaraokeFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	var screen strings.Builder
	scriptInteractive(t, "", &screen)

	c := &PreviewCmd{Input: in, Limit: 0}
	if err := c.Run(newRunContext(false)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, want := range []string{"swag preview", "Styles", "Timeline", "kara"} {
		if !strings.Contains(screen.String(), want) {
			t.Errorf("the preview is missing %q:\n%s", want, screen.String())
		}
	}
}

func TestPreviewCmdRunErrors(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "in.srt")
	if err := os.WriteFile(good, []byte(srtFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	var screen strings.Builder
	scriptInteractive(t, "", &screen)

	// A missing input fails the check.
	if err := (&PreviewCmd{Input: filepath.Join(dir, "nope.srt")}).Run(newRunContext(false)); err == nil {
		t.Error("a missing input must fail")
	}
	// A file that does not parse fails.
	broken := writeSubtitle(t, "broken.srt", "1\n00:00:0,000 --> broken\nx\n")
	if err := (&PreviewCmd{Input: broken}).Run(newRunContext(false)); err == nil {
		t.Error("an unparsable input must fail")
	}

	original := openInput
	t.Cleanup(func() { openInput = original })
	openInput = func(string) (io.ReadCloser, error) { return nil, errors.New("denied") }
	if err := (&PreviewCmd{Input: good}).Run(newRunContext(false)); err == nil {
		t.Error("a failed open must fail the command")
	}
	openInput = original

	// A failing screen fails the command.
	originalOut := terminalOut
	terminalOut = &failAtWriter{failAt: 1}
	t.Cleanup(func() { terminalOut = originalOut })
	if err := (&PreviewCmd{Input: good}).Run(newRunContext(false)); err == nil {
		t.Error("a failing screen must fail the command")
	}
}

// TestRunPreviewCommand covers the command word and the flags of the
// preview.
func TestRunPreviewCommand(t *testing.T) {
	in := writeSubtitle(t, "in.vtt", voiceVTTFixture)
	var screen strings.Builder
	scriptInteractive(t, "", &screen)

	if code := run([]string{"preview", "-i", in, "--limit", "1"}); code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
	if !strings.Contains(screen.String(), "Timeline") {
		t.Fatalf("the preview is missing its timeline:\n%s", screen.String())
	}
	if code := run([]string{"preview", "-F", "vtt", "-i", in}); code != 0 {
		t.Fatalf("run with --from exit code = %d, want 0", code)
	}
}

// TestRunPreviewHelp covers the help page of the preview command.
func TestRunPreviewHelp(t *testing.T) {
	if code := run([]string{"preview", "--help"}); code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
}
