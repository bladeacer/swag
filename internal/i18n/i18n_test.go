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
		{"en", DefaultLocale},
		{"en-US", DefaultLocale}, // language fallback to the registered en-GB
		{"fr-FR", frenchLocale},  // the second shipped locale
		{"fr-fr", frenchLocale},  // the tag match ignores the case
		{"FR-FR", frenchLocale},  // the tag match ignores the case
		{"de-DE", DefaultLocale}, // unknown languages resolve to the default
		{"", DefaultLocale},
	}
	for _, tt := range tests {
		if got := Normalise(tt.in); got != tt.want {
			t.Errorf("Normalise(%q) = %q, want %q", tt.in, got, tt.want)
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

func TestSupportedIncludesDefault(t *testing.T) {
	found := map[Locale]bool{}
	for _, l := range Supported() {
		found[l] = true
	}
	if !found[DefaultLocale] {
		t.Fatal("Supported() must include the default locale")
	}
	if !found[frenchLocale] {
		t.Fatalf("Supported() must include the second locale %q", frenchLocale)
	}
}

// TestSecondLocaleIsComplete keeps the new locale in step with the default
// one. A key that lands in en-GB must gain a translation in the same
// change, so a user of the second locale never reads the raw key.
func TestSecondLocaleIsComplete(t *testing.T) {
	for key := range enGB {
		if _, ok := frFR[key]; !ok {
			t.Errorf("the %s catalogue is missing the key %q", frenchLocale, key)
		}
	}
	for key := range frFR {
		if _, ok := enGB[key]; !ok {
			t.Errorf("the %s catalogue carries an unknown key %q", frenchLocale, key)
		}
	}
}

// TestSecondLocaleLookup covers the lookup of a translated message and the
// format of its placeholders.
func TestSecondLocaleLookup(t *testing.T) {
	tr := New("fr-FR")
	if got := tr.Locale(); got != frenchLocale {
		t.Fatalf("Locale() = %q, want %q", got, frenchLocale)
	}
	if got := tr.F(MsgConvertSuccess, "out.ass"); got != "out.ass écrit." {
		t.Fatalf("F() = %q", got)
	}
}
