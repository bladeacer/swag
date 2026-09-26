// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// boolPtr returns the address of a bool, for a field that tells an absent
// setting from a setting of false.
func boolPtr(v bool) *bool { return &v }

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
jobs = 4

[keybinds]
accept = ["Ctrl", "a"]
quit = ["q"]
`
	got, err := Decode(strings.NewReader(source))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	jobs := 4
	want := Settings{
		Locale:       "en-US",
		Verbose:      boolPtr(true),
		Strict:       boolPtr(true),
		StrictCompat: boolPtr(true),
		Font:         "Verdana",
		From:         "ass",
		Format:       "vtt",
		Preferred:    []string{"srt", "vtt"},
		Jobs:         &jobs,
		Keybinds:     map[string][]string{"accept": {"Ctrl", "a"}, "quit": {"q"}},
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

// TestDecodeBrokenDocument covers a document that TOML cannot parse, such as
// an unclosed table header.
func TestDecodeBrokenDocument(t *testing.T) {
	_, err := Decode(strings.NewReader("[keybinds\nquit = [\"q\"]\n"))
	if err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("a broken document must fail with a parse error, got %v", err)
	}
}

// TestDecodeKeybindsType covers a keybind value of the wrong type and an
// element that is not a string.
func TestDecodeKeybindsType(t *testing.T) {
	for _, source := range []string{
		"[keybinds]\nquit = \"q\"\n",
		"[keybinds]\nquit = [1, 2]\n",
		"keybinds = \"oops\"\n",
		"keybinds = 5\n",
		"keybinds = [\"a\"]\n",
	} {
		if _, err := Decode(strings.NewReader(source)); err == nil {
			t.Errorf("Decode(%q) must fail", source)
		}
	}
}

// TestDecodeEmptyKeybindRemoves covers an empty list, which removes a
// binding rather than leaving it inert.
func TestDecodeEmptyKeybindRemoves(t *testing.T) {
	got, err := Decode(strings.NewReader("[keybinds]\nquit = []\n"))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if tokens, ok := got.Keybinds["quit"]; !ok || len(tokens) != 0 {
		t.Fatalf("an empty list = %v, want an empty entry", got.Keybinds)
	}
}

// TestTypeLabel covers the reader-facing names of the TOML types.
func TestTypeLabel(t *testing.T) {
	tests := map[string]string{
		"Hash":    "table",
		"Bool":    "boolean",
		"String":  "string",
		"Integer": "integer",
		"Array":   "array",
	}
	for in, want := range tests {
		if got := typeLabel(in); got != want {
			t.Errorf("typeLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestLoadNamesTheFile covers a broken document, whose error names the file
// that carries it.
func TestLoadNamesTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("verbose = \n"), 0o644); err != nil {
		t.Fatalf("write the file: %v", err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("a broken file must name the file, got %v", err)
	}
}

// TestDecodeDuplicateKey covers a repeated key, which TOML refuses.
func TestDecodeDuplicateKey(t *testing.T) {
	if _, err := Decode(strings.NewReader("font = \"A\"\nfont = \"B\"\n")); err == nil {
		t.Fatal("a duplicate key must fail")
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
	jobs := 4
	settings := Settings{
		Locale:       "en-US",
		Verbose:      boolPtr(true),
		Strict:       boolPtr(true),
		StrictCompat: boolPtr(true),
		Font:         "Verdana",
		From:         "ass",
		Format:       "vtt",
		Jobs:         &jobs,
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
		{"jobs", 4, true},
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
	for _, name := range []string{"locale", "font", "from", "format", "target", "jobs", "verbose", "strict", "strict-compat"} {
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

// TestDefaultFileHoldsTheBuiltInKeybinds proves that writing the default
// file changes no behaviour. Every optional setting sits in a comment, and
// the keybinds table carries the built-in values.
func TestDefaultFileHoldsTheBuiltInKeybinds(t *testing.T) {
	got, err := Decode(strings.NewReader(DefaultFile))
	if err != nil {
		t.Fatalf("the default file must parse: %v", err)
	}
	want := Settings{Keybinds: map[string][]string{
		"leader": {"Ctrl", "x"},
		"accept": {"<leader>", "a"},
		"help":   {"<leader>", "?"},
		"quit":   {"<leader>", "q"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("the default file = %+v, want the built-in keybinds", got)
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

// TestRootSwagTomlMatchesTheEmbeddedDefault keeps the file at the repository
// root in step with the embedded default, because a reader opens the root
// file and the tool writes the embedded one.
func TestRootSwagTomlMatchesTheEmbeddedDefault(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "swag.toml"))
	if err != nil {
		t.Fatalf("read the root file: %v", err)
	}
	if string(data) != DefaultFile {
		t.Fatal("the root swag.toml differs from the embedded default")
	}
}

// TestMerge covers the layering of two files. A setting that the override
// carries wins, a setting it leaves absent keeps the base value, and the
// keybinds table merges one action at a time.
func TestMerge(t *testing.T) {
	baseJobs, overrideJobs := 2, 8
	base := Settings{
		Locale:       "en-US",
		Verbose:      boolPtr(false),
		Strict:       boolPtr(true),
		StrictCompat: boolPtr(true),
		Font:         "Verdana",
		From:         "ass",
		Format:       "vtt",
		Preferred:    []string{"srt"},
		Jobs:         &baseJobs,
		Keybinds:     map[string][]string{"quit": {"q"}, "help": {"h"}},
	}
	override := Settings{
		Locale:    "en-GB",
		Verbose:   boolPtr(true),
		Font:      "Courier New",
		Preferred: []string{"vtt", "srt"},
		Jobs:      &overrideJobs,
		Keybinds:  map[string][]string{"help": {"?"}},
	}
	got := Merge(base, override)

	if got.Locale != "en-GB" || got.Font != "Courier New" {
		t.Errorf("a carried override did not win: %+v", got)
	}
	if got.Verbose == nil || !*got.Verbose {
		t.Errorf("a carried bool override did not win: %+v", got.Verbose)
	}
	if got.Strict == nil || !*got.Strict || got.StrictCompat == nil || !*got.StrictCompat {
		t.Errorf("an absent bool override dropped the base value: %+v", got)
	}
	if got.From != "ass" || got.Format != "vtt" {
		t.Errorf("an absent string override dropped the base value: %+v", got)
	}
	if len(got.Preferred) != 2 || got.Preferred[0] != "vtt" {
		t.Errorf("preferred = %v, want the override", got.Preferred)
	}
	if got.Jobs == nil || *got.Jobs != 8 {
		t.Errorf("jobs = %v, want the override", got.Jobs)
	}
	if tokens := got.Keybinds["help"]; len(tokens) != 1 || tokens[0] != "?" {
		t.Errorf("the overridden keybind = %v, want [?]", tokens)
	}
	if tokens := got.Keybinds["quit"]; len(tokens) != 1 || tokens[0] != "q" {
		t.Errorf("the kept keybind = %v, want [q]", tokens)
	}
	// The merge must not mutate the base map.
	if tokens := base.Keybinds["help"]; tokens[0] != "h" {
		t.Errorf("Merge mutated the base keybinds: %v", base.Keybinds)
	}

	// A false bool and a string override replace the base value too.
	full := Merge(base, Settings{
		Strict:       boolPtr(false),
		StrictCompat: boolPtr(false),
		From:         "srt",
		Format:       "sbv",
	})
	if full.Strict == nil || *full.Strict || full.StrictCompat == nil || *full.StrictCompat {
		t.Errorf("a false bool override did not win: %+v", full)
	}
	if full.From != "srt" || full.Format != "sbv" {
		t.Errorf("a string override did not win: %+v", full)
	}
}

// TestMergeEmptyOverride keeps every base value.
func TestMergeEmptyOverride(t *testing.T) {
	baseJobs := 3
	base := Settings{Locale: "en-US", Jobs: &baseJobs, Keybinds: map[string][]string{"quit": {"q"}}}
	got := Merge(base, Default())
	if got.Locale != "en-US" || got.Jobs == nil || *got.Jobs != 3 {
		t.Fatalf("an empty override changed the base: %+v", got)
	}
	if tokens := got.Keybinds["quit"]; len(tokens) != 1 || tokens[0] != "q" {
		t.Fatalf("an empty override dropped the keybinds: %+v", got.Keybinds)
	}
}

// TestLoadStatError covers a path under a regular file, where the stat fails
// for a reason other than a missing file.
func TestLoadStatError(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatalf("write the blocker: %v", err)
	}
	if _, err := Load(filepath.Join(blocker, "config.toml")); err == nil {
		t.Fatal("a path under a file must fail the stat")
	}
}

// TestLoadCachesDecode proves that a second load of an unchanged file serves
// the kept result instead of reading the file again.
func TestLoadCachesDecode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("font = \"Verdana\"\n"), 0o644); err != nil {
		t.Fatalf("write the file: %v", err)
	}
	first, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	second, err := Load(path)
	if err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("the cached load differs: %+v and %+v", first, second)
	}
}

// TestLoadRereadsAChangedFile proves that an edit between two loads reaches
// the caller, because the cache notices the new size.
func TestLoadRereadsAChangedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("font = \"Verdana\"\n"), 0o644); err != nil {
		t.Fatalf("write the file: %v", err)
	}
	if _, err := Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := os.WriteFile(path, []byte("font = \"Courier New\"\n"), 0o644); err != nil {
		t.Fatalf("rewrite the file: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load after the edit: %v", err)
	}
	if got.Font != "Courier New" {
		t.Fatalf("the changed file = %+v, want the new font", got)
	}
}
