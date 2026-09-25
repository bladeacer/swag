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

	"github.com/BurntSushi/toml"
)

// DefaultFile is the commented configuration file that the tool writes when
// a caller asks for it. The file carries every setting as a comment, so
// writing it changes no behaviour until the user edits it.
//
//go:embed default.toml
var DefaultFile string

// Settings is the configuration that a file carries. Every field is
// optional, and a missing field keeps the default of the tool. A field that
// carries a value named zero, such as Jobs, is a pointer, so the tool can
// tell an absent setting from a setting of zero.
type Settings struct {
	// Locale names the message locale, for example "en-GB".
	Locale string `toml:"locale"`
	// Verbose prints the conversion report, as the -v flag does.
	Verbose bool `toml:"verbose"`
	// Strict fails a conversion that drops a feature, as the -s flag does.
	Strict bool `toml:"strict"`
	// StrictCompat writes no integrity block, as the -c flag does.
	StrictCompat bool `toml:"strict_compat"`
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
	// Keybinds maps an interactive action onto its key. The interactive
	// mode reads the table, and an action it does not ship is an error.
	Keybinds map[string]string `toml:"keybinds"`
}

// Default returns the settings of a configuration file that names nothing.
// The values keep the defaults of the tool, so a missing file changes no
// behaviour.
func Default() Settings {
	return Settings{}
}

// Decode reads the settings from a TOML stream. An unknown key is an error,
// so a typo never passes in silence.
func Decode(r io.Reader) (Settings, error) {
	var settings Settings
	meta, err := toml.NewDecoder(r).Decode(&settings)
	if err != nil {
		return Settings{}, fmt.Errorf("config: %w", err)
	}
	if unknown := meta.Undecoded(); len(unknown) > 0 {
		return Settings{}, fmt.Errorf("config: unknown setting %q", unknown[0].String())
	}
	return normalise(settings), nil
}

// Load reads the settings from the file at path. A missing file returns the
// defaults and no error, so the tool works with no configuration.
func Load(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("config: read %s: %w", path, err)
	}
	return Decode(bytes.NewReader(data))
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
		return s.Verbose, true
	case "strict":
		return s.Strict, true
	case "strict-compat":
		return s.StrictCompat, true
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
