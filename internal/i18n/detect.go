// SPDX-License-Identifier: Apache-2.0

package i18n

import "strings"

// localeVariables names the environment variables that carry the locale, in
// the order of the POSIX standard. The first variable that carries a value
// decides, and the later ones do not apply.
var localeVariables = []string{"LC_ALL", "LC_MESSAGES", "LANG"}

// Detect returns the locale of the environment. The first variable of
// localeVariables that carries a value decides, which follows the POSIX
// precedence. A value carries a codeset and a modifier, so "de_DE.UTF-8"
// reads as "de-DE". A tag that no catalogue carries returns DefaultLocale,
// and so does an unset environment.
func Detect(env func(string) string) Locale {
	for _, name := range localeVariables {
		value := env(name)
		if value == "" {
			continue
		}
		locale, ok := Resolve(cleanTag(value))
		if !ok {
			return DefaultLocale
		}
		return locale
	}
	return DefaultLocale
}

// cleanTag drops the codeset and the modifier from a locale value, so
// "de_DE.UTF-8@euro" reads as "de-DE". A value with no language, such as
// "C.UTF-8", reads as "C", which no catalogue carries.
func cleanTag(value string) string {
	value = strings.TrimSpace(value)
	if i := strings.IndexAny(value, ".@"); i >= 0 {
		value = value[:i]
	}
	return value
}
