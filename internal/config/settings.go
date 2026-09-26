// SPDX-License-Identifier: Apache-2.0

package config

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

// DefaultFile is the configuration file that the tool writes when a caller
// asks for it. Every fixed setting is active with its built-in default
// value, and the keybinds table carries the built-in bindings, so writing
// the file changes no behaviour until the user edits it. The copy at the
// repository root is the one a reader opens, and a test keeps the two in
// step.
//
//go:embed swag.toml
var DefaultFile string

// Settings is the configuration that a file carries. Every field is
// optional, and a missing field keeps the default of the tool. A field whose
// value could be named zero, such as Jobs or a bool, is a pointer, so the
// tool can tell an absent setting from a setting of zero when it layers one
// file over another.
type Settings struct {
	// Locale names the message locale, for example "en-GB".
	Locale string `toml:"locale"`
	// Verbose prints the conversion report, as the -v flag does.
	Verbose *bool `toml:"verbose"`
	// Strict fails a conversion that drops a feature, as the -s flag does.
	Strict *bool `toml:"strict"`
	// StrictCompat writes no integrity block, as the -c flag does.
	StrictCompat *bool `toml:"strict_compat"`
	// Font replaces the font of every style and span, as the -n flag does.
	Font string `toml:"font"`
	// From names the input format, as the -F flag does.
	From string `toml:"from"`
	// Format names the target format, as the -f flag does.
	Format string `toml:"format"`
	// Preferred lists the target formats of a batch run when the -f flag is
	// absent, and it orders the target question of the interactive mode.
	Preferred []string `toml:"preferred"`
	// Jobs is the number of conversions that run at once in a batch run. A
	// missing value uses the automatic count, which keeps two cores free,
	// and zero uses every core.
	Jobs *int `toml:"jobs"`
	// Keybinds maps an interactive action onto the tokens of its binding.
	// The interactive mode reads the table, and an action it does not ship
	// is an error. An empty list removes the default binding.
	Keybinds map[string][]string `toml:"keybinds"`
}

// Default returns the settings of a configuration file that names nothing.
// The values keep the defaults of the tool, so a missing file changes no
// behaviour.
func Default() Settings {
	return Settings{}
}

// Merge returns the settings of override layered over base. A setting that
// override carries replaces the setting of base, and a setting that override
// leaves absent keeps the value of base. The keybinds table merges one action
// at a time, so an override can change one binding and keep the rest. The
// caller layers the working directory file over the global file, so the
// closer file wins.
func Merge(base, override Settings) Settings {
	out := base
	if override.Locale != "" {
		out.Locale = override.Locale
	}
	if override.Verbose != nil {
		out.Verbose = override.Verbose
	}
	if override.Strict != nil {
		out.Strict = override.Strict
	}
	if override.StrictCompat != nil {
		out.StrictCompat = override.StrictCompat
	}
	if override.Font != "" {
		out.Font = override.Font
	}
	if override.From != "" {
		out.From = override.From
	}
	if override.Format != "" {
		out.Format = override.Format
	}
	if override.Preferred != nil {
		out.Preferred = append([]string(nil), override.Preferred...)
	}
	if override.Jobs != nil {
		out.Jobs = override.Jobs
	}
	if len(override.Keybinds) > 0 {
		out.Keybinds = make(map[string][]string, len(base.Keybinds)+len(override.Keybinds))
		for action, tokens := range base.Keybinds {
			out.Keybinds[action] = append([]string(nil), tokens...)
		}
		for action, tokens := range override.Keybinds {
			out.Keybinds[action] = append([]string(nil), tokens...)
		}
	}
	return out
}

// settingTypes names the TOML type of every setting. A value of another type
// is an error, so a broken document never half-loads in silence. The check
// catches the keybind table in particular, because the decoding library
// passes a scalar for a map field without a report.
var settingTypes = map[string]string{
	"locale":        "String",
	"verbose":       "Bool",
	"strict":        "Bool",
	"strict_compat": "Bool",
	"font":          "String",
	"from":          "String",
	"format":        "String",
	"preferred":     "Array",
	"jobs":          "Integer",
	"keybinds":      "Hash",
}

// typeLabel names a TOML type for a reader. The library calls a table a
// hash and a boolean a bool, and the label says table and boolean instead.
func typeLabel(tomlType string) string {
	switch tomlType {
	case "Hash":
		return "table"
	case "Bool":
		return "boolean"
	}
	return strings.ToLower(tomlType)
}

// checkTypes refuses a setting whose TOML type differs from the schema.
func checkTypes(meta toml.MetaData) error {
	for name, want := range settingTypes {
		if !meta.IsDefined(name) {
			continue
		}
		if got := meta.Type(name); got != want {
			return fmt.Errorf("the setting %q must be a %s, not a %s", name, typeLabel(want), typeLabel(got))
		}
	}
	return nil
}

// Decode reads the settings from a TOML stream. A broken document, a value
// of the wrong type, and an unknown key are each an error, so a typo never
// passes in silence.
func Decode(r io.Reader) (Settings, error) {
	var settings Settings
	meta, err := toml.NewDecoder(r).Decode(&settings)
	if err != nil {
		return Settings{}, fmt.Errorf("parse: %w", err)
	}
	if err := checkTypes(meta); err != nil {
		return Settings{}, err
	}
	if unknown := meta.Undecoded(); len(unknown) > 0 {
		return Settings{}, fmt.Errorf("unknown setting %q", unknown[0].String())
	}
	return normalise(settings), nil
}

// cachedEntry holds a decoded file and the stamp of the file it came from.
// The stamp catches a change between two loads, so the cache never serves a
// stale document. A missing file carries a false exists value and the
// defaults, so a file that appears later still reaches the caller.
type cachedEntry struct {
	modTime  time.Time
	size     int64
	exists   bool
	settings Settings
	err      error
}

// fileStamp identifies the state of one file for the merge cache. A missing
// file carries a zero stamp, so a file that appears later misses the cache.
type fileStamp struct {
	modTime time.Time
	size    int64
	exists  bool
}

// stamp returns the file stamp of an entry.
func (e cachedEntry) stamp() fileStamp {
	return fileStamp{modTime: e.modTime, size: e.size, exists: e.exists}
}

var (
	cacheMu sync.Mutex
	cache   = map[string]cachedEntry{}

	mergeMu    sync.Mutex
	mergeCache = map[mergeKey]Settings{}
)

// mergeKey names the two files of a merge and the state of each. The paths
// and the two stamps decide whether a kept merge is still valid.
type mergeKey struct {
	base, override           string
	baseStamp, overrideStamp fileStamp
}

// Load reads the settings from the file at path. A missing file returns the
// defaults and no error, so the tool works with no configuration. Any other
// failure names the file, so a broken document is easy to find.
//
// A decoded file is kept for the next load of the same path. The cache uses
// the size and the modification time of the file, so an edited file is read
// again. The interactive mode resolves the global file and the working
// directory file in one run, and the cache keeps a repeated lookup cheap.
func Load(path string) (Settings, error) {
	entry, err := loadEntry(path)
	if err != nil {
		return Settings{}, err
	}
	return entry.settings, entry.err
}

// loadEntry returns the entry of one file. A missing file returns the
// defaults with a zero stamp and no error. A stat or read failure returns an
// error that names the file. A decode failure stays in the entry, so a
// caller can keep the read error for the merged cache.
func loadEntry(path string) (cachedEntry, error) {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cachedEntry{settings: Default()}, nil
	}
	if err != nil {
		return cachedEntry{}, fmt.Errorf("config: read %s: %w", path, err)
	}
	if entry, ok := cached(path, info); ok {
		return entry, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cachedEntry{}, fmt.Errorf("config: read %s: %w", path, err)
	}
	settings, err := Decode(bytes.NewReader(data))
	if err != nil {
		err = fmt.Errorf("config: %s: %w", path, err)
	}
	entry := cachedEntry{modTime: info.ModTime(), size: info.Size(), exists: true, settings: settings, err: err}
	store(path, entry)
	return entry, nil
}

// LoadMerged reads the base file and the override file and layers the
// override over the base. The closer file wins, so a caller passes the
// global file first and the working directory file second.
//
// The pair of paths and the state of each file are kept, so a repeated
// lookup of two unchanged files returns the kept merge. A missing file keeps
// the defaults and changes no behaviour.
func LoadMerged(basePath, overridePath string) (Settings, error) {
	base, err := loadEntry(basePath)
	if err != nil {
		return Settings{}, err
	}
	if base.err != nil {
		return Settings{}, base.err
	}
	override, err := loadEntry(overridePath)
	if err != nil {
		return Settings{}, err
	}
	if override.err != nil {
		return Settings{}, override.err
	}
	key := mergeKey{base: basePath, override: overridePath, baseStamp: base.stamp(), overrideStamp: override.stamp()}
	if merged, ok := keptMerge(key); ok {
		return merged, nil
	}
	merged := Merge(base.settings, override.settings)
	keepMerge(key, merged)
	return merged, nil
}

// keptMerge returns the kept merge of a pair of unchanged files.
func keptMerge(key mergeKey) (Settings, bool) {
	mergeMu.Lock()
	defer mergeMu.Unlock()
	merged, ok := mergeCache[key]
	return merged, ok
}

// keepMerge stores a merge for the next lookup of the same pair of files.
func keepMerge(key mergeKey, merged Settings) {
	mergeMu.Lock()
	defer mergeMu.Unlock()
	mergeCache[key] = merged
}

// cached returns the decoded file for a path when the file is unchanged
// since the last load.
func cached(path string, info os.FileInfo) (cachedEntry, bool) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	entry, ok := cache[path]
	if !ok || entry.size != info.Size() || !entry.modTime.Equal(info.ModTime()) {
		return cachedEntry{}, false
	}
	return entry, true
}

// store keeps a decoded file for the next load of the same path.
func store(path string, entry cachedEntry) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cache[path] = entry
}

// normalise trims the list settings, so a name that carries space still
// matches the registry.
func normalise(settings Settings) Settings {
	var preferred []string
	for _, name := range settings.Preferred {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			preferred = append(preferred, trimmed)
		}
	}
	settings.Preferred = preferred
	return settings
}

// Flag returns the setting that mirrors a command line flag, and reports
// whether the setting carries a value. The name is the long form of the flag
// without the leading dashes. The interactive command names its target with
// --target, so that name reaches the same setting as --format.
func (s Settings) Flag(name string) (any, bool) {
	switch name {
	case "locale":
		return s.Locale, s.Locale != ""
	case "verbose":
		if s.Verbose != nil {
			return *s.Verbose, true
		}
	case "strict":
		if s.Strict != nil {
			return *s.Strict, true
		}
	case "strict-compat":
		if s.StrictCompat != nil {
			return *s.StrictCompat, true
		}
	case "font":
		return s.Font, s.Font != ""
	case "from":
		return s.From, s.From != ""
	case "format", "target":
		return s.Format, s.Format != ""
	case "jobs":
		if s.Jobs != nil {
			return *s.Jobs, true
		}
	}
	return nil, false
}

// Write creates the configuration file at path with content. It refuses to
// replace an existing file, so a hand-edited file never disappears.
func Write(path, content string) error {
	if Exists(path) {
		return fmt.Errorf("config: the file %s is present", path)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("config: create %s: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("config: write %s: %w", path, err)
	}
	return nil
}
