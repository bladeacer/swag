// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	settings := Default()
	if !reflect.DeepEqual(settings, Settings{}) {
		t.Fatalf("Default() = %+v, want the zero settings", settings)
	}
	if _, ok := settings.Flag("input"); ok {
		t.Error("a flag with no setting must report false")
	}
}

func TestDecode(t *testing.T) {
	const source = `
locale = "en-US"
verbose = true
strict = true
strict_compat = true
font = "Verdana"
from = "ass"
format = "vtt"
preferred = [" srt ", "vtt", ""]

[keybinds]
input = "i"
quit = "q"
`
	got, err := Decode(strings.NewReader(source))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	want := Settings{
		Locale:       "en-US",
		Verbose:      true,
		Strict:       true,
		StrictCompat: true,
		Font:         "Verdana",
		From:         "ass",
		Format:       "vtt",
		Preferred:    []string{"srt", "vtt"},
		Keybinds:     map[string]string{"input": "i", "quit": "q"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Decode = %+v, want %+v", got, want)
	}
}

func TestDecodeEmpty(t *testing.T) {
	got, err := Decode(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !reflect.DeepEqual(got, Default()) {
		t.Fatalf("Decode of an empty file = %+v, want the defaults", got)
	}
}

func TestDecodeUnknownKey(t *testing.T) {
	_, err := Decode(strings.NewReader("bogus = 1\n"))
	if err == nil || !strings.Contains(err.Error(), "unknown setting") {
		t.Fatalf("an unknown setting must fail, got %v", err)
	}
}

func TestDecodeBadValue(t *testing.T) {
	_, err := Decode(strings.NewReader("verbose = \"yes\"\n"))
	if err == nil {
		t.Fatal("a value of the wrong type must fail")
	}
}

func TestLoadMissingFile(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, Default()) {
		t.Fatalf("Load of a missing file = %+v, want the defaults", got)
	}
}

func TestLoadReadsTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("font = \"Verdana\"\n"), 0o644); err != nil {
		t.Fatalf("write the file: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Font != "Verdana" {
		t.Fatalf("Load = %+v, want the font", got)
	}
}

// TestLoadReadError covers a path that cannot be read, such as a directory.
func TestLoadReadError(t *testing.T) {
	if _, err := Load(t.TempDir()); err == nil {
		t.Fatal("an unreadable path must fail")
	}
}

func TestFlag(t *testing.T) {
	settings := Settings{
		Locale:       "en-US",
		Verbose:      true,
		Strict:       true,
		StrictCompat: true,
		Font:         "Verdana",
		From:         "ass",
		Format:       "vtt",
	}
	tests := []struct {
		name string
		want any
		ok   bool
	}{
		{"locale", "en-US", true},
		{"verbose", true, true},
		{"strict", true, true},
		{"strict-compat", true, true},
		{"font", "Verdana", true},
		{"from", "ass", true},
		{"format", "vtt", true},
		{"target", "vtt", true},
		{"output", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := settings.Flag(tt.name)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("Flag(%q) = (%v, %v), want (%v, %v)", tt.name, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestFlagEmptyStrings covers the settings whose empty value means "not
// set", so the flag keeps its own default.
func TestFlagEmptyStrings(t *testing.T) {
	settings := Default()
	for _, name := range []string{"locale", "font", "from", "format", "target"} {
		if _, ok := settings.Flag(name); ok {
			t.Errorf("an empty %s must report false", name)
		}
	}
}

func TestWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.toml")
	if err := Write(path, DefaultFile); err != nil {
		t.Fatalf("Write: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the file: %v", err)
	}
	if string(data) != DefaultFile {
		t.Fatal("the written file differs from the default file")
	}
	if err := Write(path, "locale = \"en-US\"\n"); err == nil {
		t.Fatal("Write must refuse to replace an existing file")
	}
}

// TestWriteDirectoryError covers a path whose parent is not a directory, so
// the create fails before the write.
func TestWriteDirectoryError(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatalf("write the blocker: %v", err)
	}
	if err := Write(filepath.Join(blocker, "config.toml"), DefaultFile); err == nil {
		t.Fatal("a path under a file must fail")
	}
}

// TestWriteFileError covers a write that fails after the directory exists.
// The path is a directory, so the file create fails.
func TestWriteFileError(t *testing.T) {
	dir := t.TempDir()
	if err := Write(dir, DefaultFile); err == nil {
		t.Fatal("a directory as the file path must fail")
	}
}

// TestDefaultFileHoldsNoSetting proves that writing the default file changes
// no behaviour, because every setting sits in a comment.
func TestDefaultFileHoldsNoSetting(t *testing.T) {
	got, err := Decode(strings.NewReader(DefaultFile))
	if err != nil {
		t.Fatalf("the default file must parse: %v", err)
	}
	if !reflect.DeepEqual(got, Default()) {
		t.Fatalf("the default file = %+v, want the defaults", got)
	}
}

// TestDefaultFileMatchesTheDocs keeps the documented sample in step with the
// file that the tool writes.
func TestDefaultFileMatchesTheDocs(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "configuration.md"))
	if err != nil {
		t.Fatalf("read the configuration page: %v", err)
	}
	if !strings.Contains(string(data), "```toml\n"+DefaultFile+"```") {
		t.Fatal("the configuration page does not carry the default file verbatim")
	}
}
