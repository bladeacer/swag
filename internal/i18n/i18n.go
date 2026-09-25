// Package i18n holds the message catalogue of swag.
//
// The CLI never formats display text inline: it looks messages up here, so
// a new language lands as one catalogue file. The default locale is en-GB.
// Lookups fall back to en-GB for keys that a locale does not carry.
package i18n

import (
	"fmt"
	"sort"
	"strings"
)

// Locale identifies a message set by its language tag, for example "en-GB".
type Locale string

// DefaultLocale is the locale of the shipped catalogue.
const DefaultLocale Locale = "en-GB"

// catalogues maps a locale to its message set. The en-GB entry is compiled
// in; further locales can register through Register.
var catalogues = map[Locale]map[Key]string{
	DefaultLocale: enGB,
}

// Register adds or replaces the message set of a locale. Missing keys fall
// back to en-GB at lookup time.
func Register(l Locale, messages map[Key]string) {
	catalogues[l] = messages
}

// Supported reports the registered locales in alphabetical order.
func Supported() []Locale {
	locales := make([]Locale, 0, len(catalogues))
	for l := range catalogues {
		locales = append(locales, l)
	}
	sort.Slice(locales, func(i, j int) bool { return locales[i] < locales[j] })
	return locales
}

// Resolve maps a user-supplied language tag onto a registered locale and
// reports whether the tag matched one. It accepts the "en-GB", "en_GB", and
// "en-gb" forms, and it falls back to a locale registered under the bare
// language, so "fr-CA" selects a registered "fr". A tag that is not shipped
// returns the default locale with a false report, which lets the command
// line refuse a typo instead of falling back in silence.
func Resolve(tag string) (Locale, bool) {
	tag = strings.ReplaceAll(strings.TrimSpace(tag), "_", "-")
	for l := range catalogues {
		if strings.EqualFold(string(l), tag) {
			return l, true
		}
	}
	if i := strings.Index(tag, "-"); i > 0 {
		base := tag[:i]
		for l := range catalogues {
			if strings.EqualFold(string(l), base) {
				return l, true
			}
		}
	}
	return DefaultLocale, false
}

// Normalise maps a user-supplied language tag to a registered locale. An
// unknown tag resolves to the default locale, because a library caller
// wants a catalogue rather than an error.
func Normalise(tag string) Locale {
	locale, _ := Resolve(tag)
	return locale
}

// Key names one message in the catalogue.
type Key string

// The message keys of the CLI.
const (
	MsgBannerTitle      Key = "banner.title"
	MsgBannerTagline    Key = "banner.tagline"
	MsgCliDescription   Key = "cli.description"
	MsgVersionLicence   Key = "version.licence"
	MsgOutputStdout     Key = "output.stdout"
	MsgConvertStart     Key = "convert.start"
	MsgConvertSuccess   Key = "convert.success"
	MsgConvertFailed    Key = "convert.failed"
	MsgConvertLosses    Key = "convert.losses"
	MsgUsageFailed      Key = "error.usage"
	MsgUsageBare        Key = "usage.bare"
	MsgInputMissing     Key = "error.input-missing"
	MsgInputUnreadable  Key = "error.input-unreadable"
	MsgInputIsDirectory Key = "error.input-is-directory"
	MsgTargetMissing    Key = "error.target-missing"
	MsgFormatUnknown    Key = "error.format-unknown"
	MsgOutputCreate     Key = "error.output-create"
	MsgLocaleUnknown    Key = "error.locale-unknown"

	MsgInteractiveTitle      Key = "interactive.title"
	MsgInteractiveInput      Key = "interactive.input"
	MsgInteractiveTarget     Key = "interactive.target"
	MsgInteractiveOutput     Key = "interactive.output"
	MsgInteractivePrompt     Key = "interactive.prompt"
	MsgInteractivePromptBare Key = "interactive.prompt-bare"
	MsgInteractivePick       Key = "interactive.pick"
	MsgInteractiveChoice     Key = "interactive.choice"
	MsgInteractiveRead       Key = "interactive.read"
	MsgInteractiveNone       Key = "interactive.none"
	MsgInteractiveInputLine  Key = "interactive.line.input"
	MsgInteractiveTargetLine Key = "interactive.line.target"
	MsgInteractiveOutputLine Key = "interactive.line.output"
	MsgInteractiveLosses     Key = "interactive.losses"
)

// enGB is the default message set. Values follow the simple-english rules:
// complete sentences, one instruction per message, condition first.
var enGB = map[Key]string{
	MsgBannerTitle:      "swag (Subtitles With A Gopher)",
	MsgBannerTagline:    "Read, write, and convert subtitles.",
	MsgCliDescription:   "Subtitles With A Gopher: read, write, and convert subtitles.",
	MsgVersionLicence:   "Apache-2.0 licence",
	MsgOutputStdout:     "standard output",
	MsgConvertStart:     "Converting %s to %s format.",
	MsgConvertSuccess:   "Wrote %s.",
	MsgConvertFailed:    "Conversion failed. %s",
	MsgConvertLosses:    "Features the target format does not carry (%d):",
	MsgUsageFailed:      "The command line could not be read. %s",
	MsgUsageBare:        "Give the input file with -i and the target format with -f. Run swag --help to read every flag.",
	MsgInputMissing:     "The input file does not exist. Give the path of a subtitle file with -i.",
	MsgInputUnreadable:  "The input file could not be read. %s",
	MsgInputIsDirectory: "The input path %s is a directory. Give the path of a subtitle file.",
	MsgTargetMissing:    "No target format was given. Name it with -f or give an output file with -o.",
	MsgFormatUnknown:    "The format of %s is not known. Name the target format with -f.",
	MsgOutputCreate:     "The output file %s could not be created.",
	MsgLocaleUnknown:    "The locale %s is not shipped. The shipped locales are %s.",

	MsgInteractiveTitle:      "swag interactive",
	MsgInteractiveInput:      "Input file",
	MsgInteractiveTarget:     "Target format",
	MsgInteractiveOutput:     "Output file",
	MsgInteractivePrompt:     "%s [%s]:",
	MsgInteractivePromptBare: "%s:",
	MsgInteractivePick:       "Choose a number, or press Enter for %s:",
	MsgInteractiveChoice:     "The answer %s is not one of the numbers. Answer with the number of a choice.",
	MsgInteractiveRead:       "The answer could not be read. %s",
	MsgInteractiveNone:       "No format is registered, so there is nothing to choose.",
	MsgInteractiveInputLine:  "Input: %s (detected as %s)",
	MsgInteractiveTargetLine: "Target: %s",
	MsgInteractiveOutputLine: "Output: %s",
	MsgInteractiveLosses:     "The target format does not carry %d features.",
}

// T is a catalogue bound to one locale. It is safe for concurrent use.
type T struct {
	locale Locale
}

// New returns a catalogue for the locale. Unknown locales resolve to the
// default locale.
func New(tag string) *T {
	return &T{locale: Normalise(tag)}
}

// Locale reports the locale of the catalogue.
func (t *T) Locale() Locale {
	return t.locale
}

// S returns the message for key. Unknown keys return the key itself, so a
// missing translation stays visible instead of printing an empty string.
func (t *T) S(key Key) string {
	if msg, ok := catalogues[t.locale][key]; ok {
		return msg
	}
	if msg, ok := catalogues[DefaultLocale][key]; ok {
		return msg
	}
	return string(key)
}

// F returns the message for key formatted with args.
func (t *T) F(key Key, args ...any) string {
	return fmt.Sprintf(t.S(key), args...)
}
