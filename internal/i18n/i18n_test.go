package i18n

import "testing"

func TestNormalise(t *testing.T) {
	tests := []struct {
		in   string
		want Locale
	}{
		{"en-GB", BritishLocale},
		{"en_GB", BritishLocale},
		{"en-gb", BritishLocale},
		{"  en-GB  ", BritishLocale},
		{"en", DefaultLocale},    // no catalogue is registered under the bare language
		{"en-AU", DefaultLocale}, // another English region, no catalogue of its own
		{"en-US", DefaultLocale},
		{"en-us", DefaultLocale}, // the tag match ignores the case
		{"EN-US", DefaultLocale}, // the tag match ignores the case
		{"de-DE", DefaultLocale}, // an unknown language resolves to the default
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
		{"en-GB", BritishLocale, true},
		{"en-US", DefaultLocale, true},
		{"en_us", DefaultLocale, true},
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
	// A key the test locale does not carry falls back to the default
	// catalogue.
	if got := tr.S(MsgFormatUnknown); got != enUS[MsgFormatUnknown] {
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
	if got := New("en-GB").Locale(); got != BritishLocale {
		t.Fatalf("Locale() = %q, want %q", got, BritishLocale)
	}
	if got := For(BritishLocale).Locale(); got != BritishLocale {
		t.Fatalf("For() = %q, want %q", got, BritishLocale)
	}
}

// TestSupportedIsSortedAndComplete pins the locale list. The command line
// prints it in an error message, so the order must not wobble between runs.
func TestSupportedIsSortedAndComplete(t *testing.T) {
	locales := Supported()
	if len(locales) != 2 {
		t.Fatalf("Supported() = %v, want the two English locales", locales)
	}
	if locales[0] != BritishLocale || locales[1] != DefaultLocale {
		t.Fatalf("Supported() = %v, want %s then %s", locales, BritishLocale, DefaultLocale)
	}
}

// TestSecondLocaleIsKnown keeps the British catalogue in step with the
// default one. A key that only the British catalogue carries is a mistake,
// because a lookup of a missing key falls back to the default catalogue.
func TestSecondLocaleIsKnown(t *testing.T) {
	for key := range enGB {
		if _, ok := enUS[key]; !ok {
			t.Errorf("the %s catalogue carries an unknown key %q", BritishLocale, key)
		}
	}
	if len(enGB) == 0 {
		t.Fatalf("the %s catalogue must carry the entries whose wording differs", BritishLocale)
	}
}

// TestBritishLocaleLookup covers the lookup of a variant entry and the
// fallback for a key the British catalogue does not carry.
func TestBritishLocaleLookup(t *testing.T) {
	tr := New("en-GB")
	if got := tr.Locale(); got != BritishLocale {
		t.Fatalf("Locale() = %q, want %q", got, BritishLocale)
	}
	if got := tr.S(MsgVersionLicence); got != "Apache-2.0 licence" {
		t.Fatalf("the British spelling is missing: %q", got)
	}
	if got := tr.S(MsgConvertStart); got != enUS[MsgConvertStart] {
		t.Fatalf("a shared key must fall back to the default catalogue: %q", got)
	}
	if got := New("en-US").S(MsgVersionLicence); got != "Apache-2.0 license" {
		t.Fatalf("the American spelling is missing: %q", got)
	}
}
