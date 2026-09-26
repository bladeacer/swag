// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pterm/pterm"

	"github.com/bladeacer/swag/internal/config"
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

// writeConfig writes a configuration file and returns its path.
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write the configuration: %v", err)
	}
	return path
}

// unsetLocale clears SWAG_LOCALE for the test, so a configuration setting
// for the locale is not shadowed by the environment of the host.
func unsetLocale(t *testing.T) {
	t.Helper()
	original, had := os.LookupEnv("SWAG_LOCALE")
	if err := os.Unsetenv("SWAG_LOCALE"); err != nil {
		t.Fatalf("unset SWAG_LOCALE: %v", err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("SWAG_LOCALE", original)
		}
	})
}

// TestRunReadsTheConfiguration proves a flag option of the file reaches the
// conversion, because -n is absent.
func TestRunReadsTheConfiguration(t *testing.T) {
	t.Setenv("SWAG_CONFIG", writeConfig(t, "font = \"Verdana\"\nverbose = true\n"))
	in := writeSubtitle(t, "in.ass", assKaraokeFixture)
	out := filepath.Join(t.TempDir(), "out.ass")
	capturePterm(t)

	if code := run([]string{"-i", in, "-o", out}); code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read the output: %v", err)
	}
	if !strings.Contains(string(data), "Verdana") {
		t.Fatalf("the font setting is missing:\n%s", data)
	}
}

// TestRunConfigurationSetsTheLocale proves the file feeds the locale flag.
// An unknown locale in the file fails the check, so the setting was read.
func TestRunConfigurationSetsTheLocale(t *testing.T) {
	unsetLocale(t)
	t.Setenv("SWAG_CONFIG", writeConfig(t, "locale = \"de-DE\"\n"))
	in := writeSubtitle(t, "in.srt", srtFixture)
	out := filepath.Join(t.TempDir(), "out.sbv")

	if code := run([]string{"-i", in, "-o", out}); code != 1 {
		t.Fatalf("run exit code = %d, want 1 for the unknown file locale", code)
	}
}

// TestEnvironmentBeatsTheConfiguration proves the precedence: a locale from
// the environment wins over the unknown one in the file.
func TestEnvironmentBeatsTheConfiguration(t *testing.T) {
	t.Setenv("SWAG_CONFIG", writeConfig(t, "locale = \"de-DE\"\n"))
	t.Setenv("SWAG_LOCALE", "en-US")
	in := writeSubtitle(t, "in.srt", srtFixture)
	out := filepath.Join(t.TempDir(), "out.sbv")
	capturePterm(t)

	if code := run([]string{"-i", in, "-o", out}); code != 0 {
		t.Fatalf("run exit code = %d, want 0 because the environment wins", code)
	}
}

// TestCommandLineBeatsTheConfiguration proves the precedence in the other
// direction: the flag wins over the file.
func TestCommandLineBeatsTheConfiguration(t *testing.T) {
	t.Setenv("SWAG_CONFIG", writeConfig(t, "font = \"Verdana\"\n"))
	in := writeSubtitle(t, "in.ass", assKaraokeFixture)
	out := filepath.Join(t.TempDir(), "out.ass")
	capturePterm(t)

	if code := run([]string{"-i", in, "-o", out, "-n", "Courier"}); code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read the output: %v", err)
	}
	if strings.Contains(string(data), "Verdana") {
		t.Fatalf("the flag must win over the file:\n%s", data)
	}
	if !strings.Contains(string(data), "Courier") {
		t.Fatalf("the flag value is missing:\n%s", data)
	}
}

// TestRunReportsABadConfiguration covers a file that the tool cannot read.
func TestRunReportsABadConfiguration(t *testing.T) {
	t.Setenv("SWAG_CONFIG", writeConfig(t, "bogus = 1\n"))
	out := capturePterm(t)

	if code := run([]string{"-i", "in.srt", "-o", "out.sbv"}); code != 1 {
		t.Fatalf("run exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "configuration could not be read") {
		t.Fatalf("the report is missing:\n%s", out.String())
	}
}

// TestHelpAndVersionSurviveABadConfiguration keeps the two reports working
// even when the file is broken.
func TestHelpAndVersionSurviveABadConfiguration(t *testing.T) {
	t.Setenv("SWAG_CONFIG", writeConfig(t, "bogus = 1\n"))
	capturePterm(t)
	if code := run([]string{"--help"}); code != 0 {
		t.Fatalf("run --help exit code = %d, want 0", code)
	}
	if code := run([]string{"--version"}); code != 0 {
		t.Fatalf("run --version exit code = %d, want 0", code)
	}
}

// TestBareRunSurvivesAValidConfiguration covers the bare run, which reports
// a broken file before the banner.
func TestBareRunReportsABadConfiguration(t *testing.T) {
	t.Setenv("SWAG_CONFIG", writeConfig(t, "bogus = 1\n"))
	capturePterm(t)
	if code := run(nil); code != 1 {
		t.Fatalf("a bare run with a broken file exited %d, want 1", code)
	}
}

// TestSettingsFromFileError covers a configuration location that cannot be
// resolved.
func TestSettingsFromFileError(t *testing.T) {
	original := configFile
	configFile = func(string, func(string) string) (string, error) { return "", errors.New("boom") }
	t.Cleanup(func() { configFile = original })

	if _, err := settingsFromFile(); err == nil {
		t.Fatal("a failed resolution must fail the load")
	}
	capturePterm(t)
	if code := run([]string{"config"}); code != 1 {
		t.Fatalf("run exit code = %d, want 1", code)
	}
}

// TestRunConfigInit covers the default file that the command writes, and the
// refusal to replace it.
func TestRunConfigInit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.toml")
	t.Setenv("SWAG_CONFIG", path)
	out := capturePterm(t)

	if code := run([]string{"config", "--init"}); code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "Wrote the default configuration file") {
		t.Fatalf("the report is missing:\n%s", out.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the written file: %v", err)
	}
	if !strings.Contains(string(data), "# The swag configuration file.") {
		t.Fatalf("the written file is not the default:\n%s", data)
	}

	if code := run([]string{"config", "--init"}); code != 1 {
		t.Fatalf("the second run exited %d, want 1", code)
	}
}

// withWorkingDir pins the working directory for a test, so the working
// directory configuration file is read from a known place.
func withWorkingDir(t *testing.T, dir string) {
	t.Helper()
	original := workingDir
	workingDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { workingDir = original })
}

// TestWorkingDirectoryFileOverridesTheGlobalFile proves the chain: a font in
// the working directory file wins over the font in the global file.
func TestWorkingDirectoryFileOverridesTheGlobalFile(t *testing.T) {
	t.Setenv("SWAG_CONFIG", writeConfig(t, "font = \"Verdana\"\n"))
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.LocalFileName), []byte("font = \"Courier New\"\n"), 0o644); err != nil {
		t.Fatalf("write the working directory file: %v", err)
	}
	withWorkingDir(t, dir)
	in := writeSubtitle(t, "in.ass", assKaraokeFixture)
	out := filepath.Join(t.TempDir(), "out.ass")
	capturePterm(t)

	if code := run([]string{"-i", in, "-o", out}); code != 0 {
		t.Fatalf("run exit code = %d, want 0", code)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read the output: %v", err)
	}
	if !strings.Contains(string(data), "Courier New") {
		t.Fatalf("the working directory file did not win:\n%s", data)
	}
	if strings.Contains(string(data), "Verdana") {
		t.Fatalf("the global file leaked through:\n%s", data)
	}
}

// TestWorkingDirectoryFileKeepsTheGlobalSettings proves that the working
// directory file changes one setting and keeps the rest of the global file.
func TestWorkingDirectoryFileKeepsTheGlobalSettings(t *testing.T) {
	t.Setenv("SWAG_CONFIG", writeConfig(t, "font = \"Verdana\"\nfrom = \"ass\"\n"))
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.LocalFileName), []byte("verbose = true\n"), 0o644); err != nil {
		t.Fatalf("write the working directory file: %v", err)
	}
	withWorkingDir(t, dir)
	settings, err := settingsFromFile()
	if err != nil {
		t.Fatalf("settingsFromFile: %v", err)
	}
	if settings.Font != "Verdana" || settings.From != "ass" {
		t.Fatalf("the global settings were dropped: %+v", settings)
	}
	if settings.Verbose == nil || !*settings.Verbose {
		t.Fatalf("the working directory setting was dropped: %+v", settings.Verbose)
	}
}

// TestWorkingDirectoryFileError covers a broken working directory file.
func TestWorkingDirectoryFileError(t *testing.T) {
	t.Setenv("SWAG_CONFIG", writeConfig(t, "font = \"Verdana\"\n"))
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.LocalFileName), []byte("bogus = 1\n"), 0o644); err != nil {
		t.Fatalf("write the working directory file: %v", err)
	}
	withWorkingDir(t, dir)
	if _, err := settingsFromFile(); err == nil {
		t.Fatal("a broken working directory file must fail the load")
	}
	capturePterm(t)
	if code := run([]string{"config"}); code != 1 {
		t.Fatalf("run exit code = %d, want 1", code)
	}
}

// TestConfigCmdWorkingDirectoryError covers the report of both paths when
// the working directory cannot be read.
func TestConfigCmdWorkingDirectoryError(t *testing.T) {
	t.Setenv("SWAG_CONFIG", writeConfig(t, "font = \"Verdana\"\n"))
	original := workingDir
	workingDir = func() (string, error) { return "", errors.New("boom") }
	t.Cleanup(func() { workingDir = original })

	if err := (&ConfigCmd{}).Run(newRunContext(false)); err == nil {
		t.Fatal("a failed working directory must fail the command")
	}
}

// TestWorkingDirectoryError covers a working directory that cannot be read.
func TestWorkingDirectoryError(t *testing.T) {
	original := workingDir
	workingDir = func() (string, error) { return "", errors.New("boom") }
	t.Cleanup(func() { workingDir = original })

	if _, err := settingsFromFile(); err == nil {
		t.Fatal("a failed working directory must fail the load")
	}
	capturePterm(t)
	if code := run([]string{"config"}); code != 1 {
		t.Fatalf("run exit code = %d, want 1", code)
	}
}

// TestConfigCmdReportsTheWorkingDirectoryFile covers the report of both
// paths.
func TestConfigCmdReportsTheWorkingDirectoryFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.LocalFileName), []byte("# swag\n"), 0o644); err != nil {
		t.Fatalf("write the working directory file: %v", err)
	}
	withWorkingDir(t, dir)
	t.Setenv("SWAG_CONFIG", filepath.Join(t.TempDir(), "missing.toml"))
	out := capturePterm(t)

	if err := (&ConfigCmd{}).Run(newRunContext(false)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "working directory configuration file") {
		t.Errorf("the working directory path is missing:\n%s", out.String())
	}
}

// TestConfigInitWriteError covers a path that cannot be written.
func TestConfigInitWriteError(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatalf("write the blocker: %v", err)
	}
	t.Setenv("SWAG_CONFIG", filepath.Join(blocker, "config.toml"))

	if err := (&ConfigCmd{Init: true}).Run(newRunContext(false)); err == nil {
		t.Fatal("an unwritable path must fail the command")
	}
}
