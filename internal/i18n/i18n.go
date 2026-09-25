// Package i18n holds the message catalogue of swag.
//
// The CLI never formats display text inline: it looks messages up here, so
// a new language lands as one catalogue file. The default locale is en-GB.
// Lookups fall back to en-GB for keys that a locale does not carry.
package i18n

import (
	"fmt"
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

// Supported reports the registered locales.
func Supported() []Locale {
	locales := make([]Locale, 0, len(catalogues))
	for l := range catalogues {
		locales = append(locales, l)
	}
	return locales
}

// Normalise maps a user-supplied language tag to a registered locale. It
// accepts "en-GB", "en_GB", and "en-gb" forms. Unknown tags resolve to the
// default locale.
func Normalise(tag string) Locale {
	tag = strings.ReplaceAll(strings.TrimSpace(tag), "_", "-")
	for l := range catalogues {
		if strings.EqualFold(string(l), tag) {
			return l
		}
	}
	// Fall back to the language part: "fr-CA" to "fr", when registered.
	if i := strings.Index(tag, "-"); i > 0 {
		base := tag[:i]
		for l := range catalogues {
			if strings.EqualFold(string(l), base) {
				return l
			}
		}
	}
	return DefaultLocale
}

// Key names one message in the catalogue.
type Key string

// The message keys of the CLI.
const (
	MsgBannerTitle       Key = "banner.title"
	MsgBannerTagline     Key = "banner.tagline"
	MsgConvertStart      Key = "convert.start"
	MsgConvertSuccess    Key = "convert.success"
	MsgConvertFailed     Key = "convert.failed"
	MsgConvertLosses     Key = "convert.losses"
	MsgUsageFailed       Key = "error.usage"
	MsgInputMissing      Key = "error.input-missing"
	MsgInputUnreadable   Key = "error.input-unreadable"
	MsgFormatUnknown     Key = "error.format-unknown"
	MsgFormatUnsupported Key = "error.format-unsupported"
)

// enGB is the default message set. Values follow the simple-english rules:
// complete sentences, one instruction per message, condition first.
var enGB = map[Key]string{
	MsgBannerTitle:       "swag (Subtitles With A Gopher)",
	MsgBannerTagline:     "Read, write, and convert subtitles.",
	MsgConvertStart:      "Converting %s to %s format.",
	MsgConvertSuccess:    "Wrote %s.",
	MsgConvertFailed:     "Conversion failed. %s",
	MsgConvertLosses:     "Features the target format does not carry (%d):",
	MsgUsageFailed:       "The command line could not be read. %s",
	MsgInputMissing:      "The input file does not exist. Give the path of a subtitle file with -i.",
	MsgInputUnreadable:   "The input file could not be read. %s",
	MsgFormatUnknown:     "The format of %s is not known. Name the format with -f.",
	MsgFormatUnsupported: "The format %s is not supported for this operation.",
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
