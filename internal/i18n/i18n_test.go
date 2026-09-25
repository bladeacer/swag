package i18n

import "testing"

func TestNormalise(t *testing.T) {
	tests := []struct {
		in   string
		want Locale
	}{
		{"en-GB", DefaultLocale},
		{"en_GB", DefaultLocale},
		{"en-gb", DefaultLocale},
		{"  en-GB  ", DefaultLocale},
		{"en", DefaultLocale},    // the language fallback to the registered en-GB
		{"en-AU", DefaultLocale}, // another English region, no catalogue of its own
		{"en-US", americanLocale},
		{"en-us", americanLocale}, // the tag match ignores the case
		{"EN-US", americanLocale}, // the tag match ignores the case
		{"de-DE", DefaultLocale},  // an unknown language resolves to the default
		{"", DefaultLocale},
	}
	for _, tt := range tests {
		if got := Normalise(tt.in); got != tt.want {
			t.Errorf("Normalise(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestResolveReportsAMatch covers the report of a shipped locale, so the
// command line can refuse a typo instead of falling back in silence.
func TestResolveReportsAMatch(t *testing.T) {
	tests := []struct {
		in   string
		want Locale
		ok   bool
	}{
		{"en-GB", DefaultLocale, true},
		{"en-US", americanLocale, true},
		{"en_us", americanLocale, true},
		{"en", DefaultLocale, false}, // no catalogue is registered under the bare language
		{"en-AU", DefaultLocale, false},
		{"de-DE", DefaultLocale, false},
		{"", DefaultLocale, false},
	}
	for _, tt := range tests {
		got, ok := Resolve(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("Resolve(%q) = %q, %v, want %q, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestRegisterAndLookup(t *testing.T) {
	Register(Locale("test-xx"), map[Key]string{
		MsgConvertSuccess: "TEST %s",
	})
	t.Cleanup(func() { delete(catalogues, Locale("test-xx")) })

	tr := New("test-xx")
	if got := tr.S(MsgConvertSuccess); got != "TEST %s" {
		t.Fatalf("S() = %q, want the registered message", got)
	}
	if got := tr.F(MsgConvertSuccess, "out.ass"); got != "TEST out.ass" {
		t.Fatalf("F() = %q, want formatted message", got)
	}
	// A key the test locale does not carry falls back to en-GB.
	if got := tr.S(MsgFormatUnknown); got != enGB[MsgFormatUnknown] {
		t.Fatalf("fallback failed: %q", got)
	}
}

func TestUnknownKeyReturnsKey(t *testing.T) {
	tr := New("en-GB")
	if got := tr.S(Key("does.not.exist")); got != "does.not.exist" {
		t.Fatalf("unknown key must return the key: %q", got)
	}
}

func TestLocaleReportsSelected(t *testing.T) {
	if got := New("en-GB").Locale(); got != DefaultLocale {
		t.Fatalf("Locale() = %q, want %q", got, DefaultLocale)
	}
}

// TestSupportedIsSortedAndComplete pins the locale list. The command line
// prints it in an error message, so the order must not wobble between runs.
func TestSupportedIsSortedAndComplete(t *testing.T) {
	locales := Supported()
	if len(locales) != 2 {
		t.Fatalf("Supported() = %v, want the two English locales", locales)
	}
	if locales[0] != DefaultLocale || locales[1] != americanLocale {
		t.Fatalf("Supported() = %v, want %s then %s", locales, DefaultLocale, americanLocale)
	}
}

// TestSecondLocaleIsKnown keeps the second locale in step with the default
// one. A key that lands in en-GB without a variant is fine, because the
// lookup falls back, but a key that only the second catalogue carries is a
// mistake.
func TestSecondLocaleIsKnown(t *testing.T) {
	for key := range enUS {
		if _, ok := enGB[key]; !ok {
			t.Errorf("the %s catalogue carries an unknown key %q", americanLocale, key)
		}
	}
	if len(enUS) == 0 {
		t.Fatalf("the %s catalogue must carry the entries whose wording differs", americanLocale)
	}
}

// TestSecondLocaleLookup covers the lookup of a variant entry and the
// fallback for a key the second catalogue does not carry.
func TestSecondLocaleLookup(t *testing.T) {
	tr := New("en-US")
	if got := tr.Locale(); got != americanLocale {
		t.Fatalf("Locale() = %q, want %q", got, americanLocale)
	}
	if got := tr.S(MsgVersionLicence); got != "Apache-2.0 license" {
		t.Fatalf("the American spelling is missing: %q", got)
	}
	if got := tr.S(MsgConvertStart); got != enGB[MsgConvertStart] {
		t.Fatalf("a shared key must fall back to en-GB: %q", got)
	}
	if got := New("en-GB").S(MsgVersionLicence); got != "Apache-2.0 licence" {
		t.Fatalf("the British spelling is missing: %q", got)
	}
}
