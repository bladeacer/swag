// SPDX-License-Identifier: Apache-2.0

package i18n

import (
	"os"
	"testing"
)

// envFrom builds the environment lookup of a test.
func envFrom(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}

func TestDetect(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want Locale
	}{
		{"British English", map[string]string{"LANG": "en_GB.UTF-8"}, BritishLocale},
		{"American English", map[string]string{"LANG": "en_US.UTF-8"}, DefaultLocale},
		{"a codeset and a modifier", map[string]string{"LANG": "en_GB.UTF-8@euro"}, BritishLocale},
		{"a bare tag", map[string]string{"LANG": "en-GB"}, BritishLocale},
		{"an unsupported language", map[string]string{"LANG": "de_DE.UTF-8"}, DefaultLocale},
		{"another English region", map[string]string{"LANG": "en_AU.UTF-8"}, DefaultLocale},
		{"the C locale", map[string]string{"LANG": "C"}, DefaultLocale},
		{"the C locale with a codeset", map[string]string{"LANG": "C.UTF-8"}, DefaultLocale},
		{"the POSIX locale", map[string]string{"LANG": "POSIX"}, DefaultLocale},
		{"a tag with no language", map[string]string{"LANG": ".UTF-8"}, DefaultLocale},
		{"space around the value", map[string]string{"LANG": "  en_GB.UTF-8  "}, BritishLocale},
		{"no variable carries a value", nil, DefaultLocale},
		{"every variable is empty", map[string]string{"LC_ALL": "", "LC_MESSAGES": "", "LANG": ""}, DefaultLocale},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Detect(envFrom(tt.env)); got != tt.want {
				t.Errorf("Detect(%v) = %q, want %q", tt.env, got, tt.want)
			}
		})
	}
}

// TestDetectPrecedence covers the POSIX order: LC_ALL wins over LC_MESSAGES,
// and LC_MESSAGES wins over LANG. The first variable that carries a value
// decides, so an unsupported value does not fall through to a later one.
func TestDetectPrecedence(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want Locale
	}{
		{"LC_ALL wins", map[string]string{"LC_ALL": "en_US.UTF-8", "LC_MESSAGES": "en_GB.UTF-8", "LANG": "en_GB.UTF-8"}, DefaultLocale},
		{"LC_MESSAGES wins over LANG", map[string]string{"LC_MESSAGES": "en_GB.UTF-8", "LANG": "en_US.UTF-8"}, BritishLocale},
		{"an empty LC_ALL leaves LC_MESSAGES", map[string]string{"LC_ALL": "", "LC_MESSAGES": "en_GB.UTF-8", "LANG": "en_US.UTF-8"}, BritishLocale},
		{"an unsupported LC_ALL stops the search", map[string]string{"LC_ALL": "fr_FR.UTF-8", "LC_MESSAGES": "en_GB.UTF-8"}, DefaultLocale},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Detect(envFrom(tt.env)); got != tt.want {
				t.Errorf("Detect(%v) = %q, want %q", tt.env, got, tt.want)
			}
		})
	}
}

// TestDetectReadsTheProcessEnvironment covers the live environment, which
// the command line reads through os.Getenv.
func TestDetectReadsTheProcessEnvironment(t *testing.T) {
	t.Setenv("LC_ALL", "en_GB.UTF-8")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "en_US.UTF-8")
	if got := Detect(os.Getenv); got != BritishLocale {
		t.Errorf("Detect over the process environment = %q, want %q", got, BritishLocale)
	}
}
