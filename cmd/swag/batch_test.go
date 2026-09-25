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
)

// writeFileIn writes a fixture inside a named directory, and creates the
// parent directories it needs.
func writeFileIn(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("make the directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestBatchTargets(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"vtt", 1},
		{" srt , vtt ", 2},
		{"srt,,vtt", 2},
		{"", 0},
		{" , ", 0},
	}
	for _, tt := range tests {
		if got := batchTargets(tt.in); len(got) != tt.want {
			t.Errorf("batchTargets(%q) = %v, want %d entries", tt.in, got, tt.want)
		}
	}
}

// TestPlanBatchBesideTheInput covers the walk: it skips a file that no
// format claims, it descends into a subdirectory, and it writes each output
// beside its input when no output directory is given.
func TestPlanBatchBesideTheInput(t *testing.T) {
	dir := t.TempDir()
	writeFileIn(t, dir, "a.srt", srtFixture)
	writeFileIn(t, dir, "b.vtt", voiceVTTFixture)
	writeFileIn(t, dir, "notes.txt", "not a subtitle\n")
	writeFileIn(t, dir, filepath.Join("sub", "c.srt"), srtFixture)

	plans, err := planBatch(dir, "", []string{"json1"})
	if err != nil {
		t.Fatalf("planBatch: %v", err)
	}
	if len(plans) != 3 {
		t.Fatalf("plans = %d, want 3:\n%+v", len(plans), plans)
	}
	want := map[string]string{
		filepath.Join(dir, "a.json1"):        filepath.Join(dir, "a.srt"),
		filepath.Join(dir, "b.json1"):        filepath.Join(dir, "b.vtt"),
		filepath.Join(dir, "sub", "c.json1"): filepath.Join(dir, "sub", "c.srt"),
	}
	for _, plan := range plans {
		if plan.Target != "json1" {
			t.Errorf("target = %q, want json1", plan.Target)
		}
		if want[plan.Output] != plan.Input {
			t.Errorf("plan %+v does not match the expected output map", plan)
		}
	}
}

// TestPlanBatchIntoAnOutputDirectory covers several targets at once, which
// is the multi-writer shape, and the output directory.
func TestPlanBatchIntoAnOutputDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFileIn(t, dir, "a.srt", srtFixture)

	plans, err := planBatch(dir, filepath.Join(dir, "out"), []string{"vtt", "json1"})
	if err != nil {
		t.Fatalf("planBatch: %v", err)
	}
	if len(plans) != 2 {
		t.Fatalf("plans = %d, want one per target", len(plans))
	}
	for _, plan := range plans {
		if !strings.HasPrefix(plan.Output, filepath.Join(dir, "out")) {
			t.Errorf("output = %q, want it under the output directory", plan.Output)
		}
	}
}

func TestPlanBatchError(t *testing.T) {
	if _, err := planBatch(filepath.Join(t.TempDir(), "missing"), "", []string{"vtt"}); err == nil {
		t.Fatal("a missing directory must fail the plan")
	}
}

// TestPlanBatchSubdirectoryError covers a subdirectory that cannot be read,
// which fails the whole plan.
func TestPlanBatchSubdirectoryError(t *testing.T) {
	dir := t.TempDir()
	writeFileIn(t, dir, filepath.Join("sub", "a.srt"), srtFixture)

	original := readDir
	t.Cleanup(func() { readDir = original })
	readDir = func(name string) ([]os.DirEntry, error) {
		if name != dir {
			return nil, errors.New("denied")
		}
		return os.ReadDir(name)
	}
	if _, err := planBatch(dir, "", []string{"vtt"}); err == nil {
		t.Fatal("an unreadable subdirectory must fail the plan")
	}
}

// TestRunBatchPlanError covers a directory the walk cannot read.
func TestRunBatchPlanError(t *testing.T) {
	original := readDir
	t.Cleanup(func() { readDir = original })
	readDir = func(string) ([]os.DirEntry, error) { return nil, errors.New("denied") }

	c := &ConvertCmd{Input: t.TempDir(), Format: "vtt"}
	if err := c.Run(newRunContext(false)); err == nil {
		t.Fatal("an unreadable directory must fail the batch")
	}
}

// TestRunBatchStrictFails covers the strict policy of a batch run.
func TestRunBatchStrictFails(t *testing.T) {
	dir := t.TempDir()
	writeFileIn(t, dir, "a.ass", assKaraokeFixture)

	c := &ConvertCmd{Input: dir, Format: "srt", Strict: true}
	if err := c.Run(newRunContext(false)); err == nil {
		t.Fatal("a strict batch that loses a feature must fail")
	}
}

func TestBatchOutput(t *testing.T) {
	if got := batchOutput("/in", "", "a", "vtt"); got != filepath.Join("/in", "a.vtt") {
		t.Errorf("batchOutput without a directory = %q", got)
	}
	if got := batchOutput("/in", "/out", "a", "vtt"); got != filepath.Join("/out", "a.vtt") {
		t.Errorf("batchOutput with a directory = %q", got)
	}
}

// TestRunBatchConverts covers a whole directory run: two files, one target.
func TestRunBatchConverts(t *testing.T) {
	dir := t.TempDir()
	writeFileIn(t, dir, "a.srt", srtFixture)
	writeFileIn(t, dir, "b.srt", srtFixture)

	c := &ConvertCmd{Input: dir, Format: "vtt"}
	if err := c.Run(newRunContext(false)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, name := range []string{"a.vtt", "b.vtt"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if !strings.HasPrefix(string(data), "WEBVTT") {
			t.Errorf("%s must be WebVTT:\n%s", name, data)
		}
	}
}

// TestRunBatchWritesSeveralTargets covers the multi-writer shape and the
// font override.
func TestRunBatchWritesSeveralTargets(t *testing.T) {
	dir := t.TempDir()
	writeFileIn(t, dir, "a.srt", srtFixture)
	outDir := filepath.Join(dir, "out")

	c := &ConvertCmd{Input: dir, Format: "vtt, json1", Output: outDir, Font: "Courier"}
	if err := c.Run(newRunContext(false)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, name := range []string{"a.vtt", "a.json1"} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Errorf("the output %s is missing: %v", name, err)
		}
	}
}

// TestRunBatchKeepsSubdirectories covers an input in a subdirectory, whose
// output directory the run creates. Without that step the write fails.
func TestRunBatchKeepsSubdirectories(t *testing.T) {
	dir := t.TempDir()
	writeFileIn(t, dir, filepath.Join("sub", "a.srt"), srtFixture)
	outDir := filepath.Join(dir, "out")

	c := &ConvertCmd{Input: dir, Format: "vtt", Output: outDir}
	if err := c.Run(newRunContext(false)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "sub", "a.vtt")); err != nil {
		t.Fatalf("the nested output is missing: %v", err)
	}
}

// TestRunBatchVerboseReportsLosses covers the report of a batch run, which
// prints the loss list of each output.
func TestRunBatchVerboseReportsLosses(t *testing.T) {
	dir := t.TempDir()
	writeFileIn(t, dir, "a.ass", assKaraokeFixture)

	c := &ConvertCmd{Input: dir, Format: "srt"}
	if err := c.Run(newRunContext(true)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.srt")); err != nil {
		t.Fatalf("output missing: %v", err)
	}
}

// TestRunBatchErrors covers the failures of a batch run: a missing target
// list, a directory with nothing to convert, and a file that does not parse.
func TestRunBatchErrors(t *testing.T) {
	broken := t.TempDir()
	writeFileIn(t, broken, "bad.srt", "1\n00:00:0,000 --> broken\nx\n")

	tests := []struct {
		name string
		cmd  ConvertCmd
	}{
		{"no format", ConvertCmd{Input: t.TempDir()}},
		{"empty directory", ConvertCmd{Input: t.TempDir(), Format: "vtt"}},
		{"a failing file", ConvertCmd{Input: broken, Format: "vtt"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmd
			if err := cmd.Run(newRunContext(false)); err == nil {
				t.Errorf("%s must fail", tt.name)
			}
		})
	}
}

// TestRunBatchMkdirError covers a failing output directory. A plain file
// stands in for the directory the command cannot create.
func TestRunBatchMkdirError(t *testing.T) {
	dir := t.TempDir()
	writeFileIn(t, dir, "a.srt", srtFixture)
	blocker := writeFileIn(t, dir, "blocker", "not a directory\n")

	c := &ConvertCmd{Input: dir, Format: "vtt", Output: filepath.Join(blocker, "out")}
	if err := c.Run(newRunContext(false)); err == nil {
		t.Fatal("an output directory that cannot be created must fail")
	}
}

// TestConvertOneErrors covers the failures of one batch output: the open,
// the read, the output, and the conversion.
func TestConvertOneErrors(t *testing.T) {
	dir := t.TempDir()
	good := writeFileIn(t, dir, "a.srt", srtFixture)
	tr := i18n.New("en-GB")
	original := openInput
	t.Cleanup(func() { openInput = original })

	plan := batchPlan{Input: good, Target: "vtt", Output: filepath.Join(dir, "a.vtt")}
	c := &ConvertCmd{}
	if _, err := c.convertOne(plan, tr); err != nil {
		t.Fatalf("convertOne: %v", err)
	}

	openInput = func(string) (io.ReadCloser, error) { return nil, errors.New("denied") }
	if _, err := c.convertOne(plan, tr); err == nil {
		t.Error("a failed open must fail the conversion")
	}

	openInput = func(string) (io.ReadCloser, error) { return brokenReadCloser{}, nil }
	if _, err := c.convertOne(plan, tr); err == nil {
		t.Error("a failed read must fail the conversion")
	}
	openInput = original

	badOutput := batchPlan{Input: good, Target: "vtt", Output: filepath.Join(dir, "no", "a.vtt")}
	if _, err := c.convertOne(badOutput, tr); err == nil {
		t.Error("an unwritable output must fail the conversion")
	}

	badTarget := batchPlan{Input: good, Target: "bogus", Output: filepath.Join(dir, "a.bin")}
	if _, err := c.convertOne(badTarget, tr); err == nil {
		t.Error("an unknown target must fail the conversion")
	}
}

// TestRunBatchThroughTheParser covers the command word and the flag path of
// a batch run.
func TestRunBatchThroughTheParser(t *testing.T) {
	dir := t.TempDir()
	writeFileIn(t, dir, "a.srt", srtFixture)
	if code := run([]string{"convert", "-i", dir, "-f", "vtt", "-v"}); code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.vtt")); err != nil {
		t.Fatalf("output missing: %v", err)
	}
}
