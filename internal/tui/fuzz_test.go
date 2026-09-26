// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"strings"
	"testing"
)

// FuzzParseTokens checks that the token parser never panics. A non-empty
// token list that parses must name at least one key, because a list of
// modifiers alone is an error.
func FuzzParseTokens(f *testing.F) {
	for _, seed := range []string{
		"Ctrl\x00x",
		"<leader>\x00a",
		"Ctrl\x00Shift\x00x",
		"x\x00Shift",
		"a\x00b\x00c",
		"esc",
		"not a key",
		"Ctrl",
		"",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data string) {
		tokens := strings.Split(data, "\x00")
		sequence, err := parseTokens(tokens)
		if err == nil && len(tokens) > 0 && sequence == "" {
			t.Fatalf("a parsed token list produced no key sequence: %q", tokens)
		}
	})
}

// FuzzNewKeymap checks that the keymap builder never panics on an override
// table. A build that succeeds must answer a binding lookup.
func FuzzNewKeymap(f *testing.F) {
	f.Add("accept", "Ctrl\x00a")
	f.Add("quit", "")
	f.Add("bogus", "q")
	f.Fuzz(func(t *testing.T, action, data string) {
		tokens := strings.Split(data, "\x00")
		keymap, err := NewKeymap(map[string][]string{action: tokens})
		if err != nil {
			return
		}
		_ = keymap.Bindings()
	})
}
