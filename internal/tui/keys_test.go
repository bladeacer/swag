// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"reflect"
	"strings"
	"testing"
)

func TestDefaultKeybinds(t *testing.T) {
	want := map[string]string{
		"leader": "ctrl+x",
		"accept": "<leader>a",
		"help":   "<leader>?",
		"quit":   "<leader>q",
	}
	if got := DefaultKeybinds(); !reflect.DeepEqual(got, want) {
		t.Errorf("DefaultKeybinds = %v, want %v", got, want)
	}
}

func TestNewKeymapDefaults(t *testing.T) {
	keymap, err := NewKeymap(nil)
	if err != nil {
		t.Fatalf("NewKeymap: %v", err)
	}
	// The leader is ctrl+x, so every binding starts with its control byte.
	tests := []struct {
		line   string
		action string
	}{
		{"\x18q", ActionQuit},
		{"\x18a", ActionAccept},
		{"\x18?", ActionHelp},
	}
	for _, tt := range tests {
		action, ok := keymap.Match(tt.line)
		if !ok || action != tt.action {
			t.Errorf("Match(%q) = %q, %v, want %q", tt.line, action, ok, tt.action)
		}
	}
	// A plain answer runs no action, and neither does the leader alone.
	for _, line := range []string{"", "vtt", "\x18", "no such key"} {
		if action, ok := keymap.Match(line); ok {
			t.Errorf("Match(%q) = %q, want no match", line, action)
		}
	}
	// A typed answer with space around it still matches, because the
	// prompter reads the raw line and the answer cuts the space off.
	if action, ok := keymap.Match(" \x18q "); !ok || action != ActionQuit {
		t.Errorf("Match with space = %q, %v, want the quit action", action, ok)
	}
}

// TestKeymapMatchWithoutKeys covers a keymap that no caller built, so a
// prompt with no bindings treats every line as an answer.
func TestKeymapMatchWithoutKeys(t *testing.T) {
	var keymap *Keymap
	if action, ok := keymap.Match("anything"); ok {
		t.Errorf("a missing keymap matched %q", action)
	}
}

func TestNewKeymapOverrides(t *testing.T) {
	keymap, err := NewKeymap(map[string]string{
		"leader": "ctrl+a",
		"quit":   "esc",
		"accept": "space",
	})
	if err != nil {
		t.Fatalf("NewKeymap: %v", err)
	}
	// The leader moved to ctrl+a, so the help binding follows it.
	if action, ok := keymap.Match("\x01?"); !ok || action != ActionHelp {
		t.Errorf("the help binding must follow the new leader: %q, %v", action, ok)
	}
	// The quit binding carries no leader now.
	if action, ok := keymap.Match("\x1b"); !ok || action != ActionQuit {
		t.Errorf("Match(esc) = %q, %v, want the quit action", action, ok)
	}
	// The named space key is a binding of its own.
	if action, ok := keymap.Match(" "); !ok || action != ActionAccept {
		t.Errorf("Match(space) = %q, %v, want the accept action", action, ok)
	}
	// The old quit binding is gone.
	if action, ok := keymap.Match("\x18q"); ok {
		t.Errorf("the old quit binding still matched %q", action)
	}
}

// TestNewKeymapDropsABinding covers a whitespace-only action, which removes
// the binding instead of leaving it inert.
func TestNewKeymapDropsABinding(t *testing.T) {
	keymap, err := NewKeymap(map[string]string{"quit": "   "})
	if err != nil {
		t.Fatalf("NewKeymap: %v", err)
	}
	if _, ok := keymap.Match("\x18q"); ok {
		t.Error("a removed binding must not match")
	}
	if len(keymap.Bindings()) != 2 {
		t.Errorf("bindings = %v, want the accept and help bindings", keymap.Bindings())
	}
}

func TestNewKeymapErrors(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string]string
		want      string
	}{
		{"unknown action", map[string]string{"next": "n"}, "is not known"},
		{"no leader", map[string]string{"leader": ""}, "no leader key"},
		{"leader names itself", map[string]string{"leader": "<leader>a"}, "cannot name itself"},
		{"unknown modifier", map[string]string{"quit": "super+q"}, "not known"},
		{"unknown key name", map[string]string{"quit": "ctrl+pageup"}, "is not a single key"},
		{"control of a long key", map[string]string{"quit": "ctrl+up"}, "ctrl needs a single key"},
		{"control of an escape key", map[string]string{"quit": "ctrl+esc"}, "does not apply"},
		{"key is not one rune", map[string]string{"quit": "notakey"}, "is not a single key"},
		{"missing key", map[string]string{"quit": "ctrl+"}, "is not a single key"},
		{"empty key", map[string]string{"quit": "+"}, "not known"},
		{"two actions on one key", map[string]string{"help": "<leader>a"}, "use the same keys"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewKeymap(tt.overrides)
			if err == nil {
				t.Fatalf("NewKeymap(%v) must fail", tt.overrides)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want it to carry %q", err, tt.want)
			}
		})
	}
}

// TestNewKeymapAltAndMetaAreTheSame covers two notations that name the same
// key, because meta is another name for alt.
func TestNewKeymapAltAndMetaAreTheSame(t *testing.T) {
	_, err := NewKeymap(map[string]string{"accept": "alt+esc", "help": "meta+esc"})
	if err == nil {
		t.Fatal("alt+esc and meta+esc must clash")
	}
	if !strings.Contains(err.Error(), "use the same keys") {
		t.Errorf("error = %q, want it to name the shared keys", err)
	}
}

func TestKeymapBindings(t *testing.T) {
	keymap, err := NewKeymap(nil)
	if err != nil {
		t.Fatalf("NewKeymap: %v", err)
	}
	want := []Binding{
		{Action: ActionAccept, Notation: "<leader>a"},
		{Action: ActionHelp, Notation: "<leader>?"},
		{Action: ActionQuit, Notation: "<leader>q"},
	}
	if got := keymap.Bindings(); !reflect.DeepEqual(got, want) {
		t.Errorf("Bindings = %v, want %v", got, want)
	}
}

func TestKnownAction(t *testing.T) {
	tests := []struct {
		action string
		want   bool
	}{
		{ActionAccept, true},
		{ActionHelp, true},
		{ActionQuit, true},
		{actionLeader, true},
		{"next", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := knownAction(tt.action); got != tt.want {
			t.Errorf("knownAction(%q) = %v, want %v", tt.action, got, tt.want)
		}
	}
}

func TestParseKey(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{"plain letter", "q", "q"},
		{"named escape", "esc", "\x1b"},
		{"named key in upper case", "ESC", "\x1b"},
		{"named arrow", "up", "\x1b[A"},
		{"control letter", "ctrl+c", "\x03"},
		{"control letter in upper case", "CTRL+C", "\x03"},
		{"control space", "ctrl+space", "\x00"},
		{"control at", "ctrl+@", "\x00"},
		{"control bracket", "ctrl+[", "\x1b"},
		{"control underscore", "ctrl+_", "\x1f"},
		{"shift of a letter", "shift+a", "A"},
		{"shift of a named key changes nothing", "shift+esc", "\x1b"},
		{"alt of a letter", "alt+x", "\x1bx"},
		{"meta is alt", "meta+x", "\x1bx"},
		{"alt of a named key", "alt+up", "\x1b\x1b[A"},
		{"control and alt", "ctrl+alt+c", "\x1b\x03"},
		{"every modifier at once", "ctrl+alt+shift+r", "\x1b\x12"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseKey(tt.token)
			if err != nil {
				t.Fatalf("parseKey(%q): %v", tt.token, err)
			}
			if got != tt.want {
				t.Errorf("parseKey(%q) = %q, want %q", tt.token, got, tt.want)
			}
		})
	}
}

func TestParseKeyErrors(t *testing.T) {
	tests := []struct {
		token string
		want  string
	}{
		{"super+q", "not known"},
		{"alt", "is not a single key"},
		{"ab", "is not a single key"},
		{"", "is not a single key"},
		{"ctrl+up", "ctrl needs a single key"},
		{"ctrl+esc", "does not apply"},
		{"ctrl+1", "does not apply"},
	}
	for _, tt := range tests {
		_, err := parseKey(tt.token)
		if err == nil {
			t.Errorf("parseKey(%q) must fail", tt.token)
			continue
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Errorf("parseKey(%q) error = %q, want it to carry %q", tt.token, err, tt.want)
		}
	}
}

func TestControlByte(t *testing.T) {
	tests := []struct {
		r    rune
		want byte
	}{
		{'a', 0x01},
		{'z', 0x1a},
		{'A', 0x01},
		{' ', 0x00},
		{'@', 0x00},
		{'[', 0x1b},
		{'_', 0x1f},
	}
	for _, tt := range tests {
		got, err := controlByte(tt.r)
		if err != nil {
			t.Fatalf("controlByte(%q): %v", tt.r, err)
		}
		if got != tt.want {
			t.Errorf("controlByte(%q) = %#x, want %#x", tt.r, got, tt.want)
		}
	}
	if _, err := controlByte('?'); err == nil {
		t.Error("controlByte(?) must fail")
	}
}

// TestExpandTokens covers the leader substitution inside a binding. The
// leader token stands alone or runs into the key that follows it.
func TestExpandTokens(t *testing.T) {
	leader := []string{"ctrl+x"}
	want := []string{"ctrl+x", "a"}
	for _, notation := range []string{"<leader>a", "<leader> a"} {
		got := expandTokens(notation, leader)
		if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
			t.Fatalf("expandTokens(%q) = %v, want %v", notation, got, want)
		}
	}
	if got := expandTokens("<leader>", leader); len(got) != 1 || got[0] != "ctrl+x" {
		t.Fatalf("expandTokens of a bare leader = %v", got)
	}
	if got := expandTokens("esc q", nil); len(got) != 2 || got[1] != "q" {
		t.Fatalf("expandTokens with no leader token = %v", got)
	}
}

// TestLeaderTokens covers the token list of the leader notation.
func TestLeaderTokens(t *testing.T) {
	got, err := leaderTokens("ctrl+x")
	if err != nil {
		t.Fatalf("leaderTokens: %v", err)
	}
	if len(got) != 1 || got[0] != "ctrl+x" {
		t.Fatalf("leaderTokens = %v, want one token", got)
	}
	if _, err := leaderTokens(" <leader> "); err == nil {
		t.Error("a leader that names itself must fail")
	}
}
