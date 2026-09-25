package i18n

import "testing"

// TestNormaliseLanguageFallback covers the base-language fallback for a
// tag whose language part is registered and whose region part is not.
func TestNormaliseLanguageFallback(t *testing.T) {
	Register(Locale("fr"), map[Key]string{})
	t.Cleanup(func() { delete(catalogues, Locale("fr")) })

	if got := Normalise("fr-CA"); got != Locale("fr") {
		t.Fatalf("Normalise(fr-CA) = %q, want fr", got)
	}
}
