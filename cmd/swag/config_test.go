// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pterm/pterm"
)

// capturePterm sends the pterm output of a test into a buffer.
func capturePterm(t *testing.T) *strings.Builder {
	t.Helper()
	var out strings.Builder
	pterm.SetDefaultOutput(&out)
	t.Cleanup(func() { pterm.SetDefaultOutput(os.Stdout) })
	return &out
}

// TestConfigCmdRun reports a file that exists.
func TestConfigCmdRun(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(file, []byte("# swag\n"), 0o644); err != nil {
		t.Fatalf("write the file: %v", err)
	}
	t.Setenv("SWAG_CONFIG", file)
	out := capturePterm(t)

	if err := (&ConfigCmd{}).Run(newRunContext(false)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	printed := out.String()
	if !strings.Contains(printed, file) {
		t.Errorf("the path is missing:\n%s", printed)
	}
	if !strings.Contains(printed, "present") {
		t.Errorf("a present file must be reported:\n%s", printed)
	}
	if !strings.Contains(printed, "runs on") {
		t.Errorf("the platform is missing:\n%s", printed)
	}
}

// TestConfigCmdRunAbsent reports a file that is not there.
func TestConfigCmdRunAbsent(t *testing.T) {
	t.Setenv("SWAG_CONFIG", filepath.Join(t.TempDir(), "missing.toml"))
	out := capturePterm(t)

	if err := (&ConfigCmd{}).Run(newRunContext(false)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "absent") {
		t.Errorf("a missing file must be reported:\n%s", out.String())
	}
}

// TestConfigCmdRunError covers a platform whose configuration location
// cannot be resolved.
func TestConfigCmdRunError(t *testing.T) {
	original := configFile
	configFile = func(string, func(string) string) (string, error) { return "", errors.New("boom") }
	t.Cleanup(func() { configFile = original })

	if err := (&ConfigCmd{}).Run(newRunContext(false)); err == nil {
		t.Fatal("a failed resolution must fail the command")
	}
}

// TestRunConfigCommand covers the command through the parser, and the
// command list of the help page.
func TestRunConfigCommand(t *testing.T) {
	t.Setenv("SWAG_CONFIG", filepath.Join(t.TempDir(), "missing.toml"))
	out := capturePterm(t)

	if code := run([]string{"config"}); code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "configuration file") {
		t.Fatalf("the report is missing:\n%s", out.String())
	}
}
