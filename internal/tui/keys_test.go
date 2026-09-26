// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"reflect"
	"strings"
	"testing"
)

func TestDefaultKeybinds(t *testing.T) {
	want := map[string][]string{
		"leader": {"Ctrl", "x"},
		"accept": {"<leader>", "a"},
		"help":   {"<leader>", "?"},
		"quit":   {"<leader>", "q"},
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
	// The leader is Ctrl+x, so every binding starts with its control byte.
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
	if action, state := keymap.Step("anything"); state != ScanNone || action != "" {
		t.Errorf("Step on a missing keymap = %q, %v", action, state)
	}
}

// TestKeymapStepWithoutTrie covers a keymap that carries no trie, which
// matches nothing.
func TestKeymapStepWithoutTrie(t *testing.T) {
	keymap := &Keymap{}
	if action, state := keymap.Step("x"); state != ScanNone || action != "" {
		t.Errorf("Step on a keymap with no trie = %q, %v", action, state)
	}
}

// TestTrieInsert covers both prefix conflicts directly, so each branch is
// covered whatever the map order of a keymap build.
func TestTrieInsert(t *testing.T) {
	// A sequence that carries an earlier sequence as its prefix fails
	// during the walk.
	node := &trieNode{}
	if err := node.insert("a", "first"); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := node.insert("ab", "second"); err == nil {
		t.Fatal("a sequence that extends an earlier one must fail")
	}

	// A sequence that is a prefix of an earlier one fails at its end.
	node = &trieNode{}
	if err := node.insert("ab", "first"); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := node.insert("a", "second"); err == nil {
		t.Fatal("a sequence that ends inside an earlier one must fail")
	}

	// A repeated sequence fails too.
	node = &trieNode{}
	if err := node.insert("a", "first"); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := node.insert("a", "second"); err == nil {
		t.Fatal("a repeated sequence must fail")
	}
}

// TestKeymapStepWithoutBindings covers a keymap whose trie carries no
// binding, so an empty step reaches the end of the matcher.
func TestKeymapStepWithoutBindings(t *testing.T) {
	keymap, err := NewKeymap(map[string][]string{"accept": {}, "help": {}, "quit": {}})
	if err != nil {
		t.Fatalf("NewKeymap: %v", err)
	}
	if action, state := keymap.Step(""); state != ScanNone || action != "" {
		t.Errorf("Step on an empty trie = %q, %v", action, state)
	}
}

// TestKeymapStep covers the incremental matcher: a prefix is pending, a full
// sequence matches, and a divergence matches nothing.
func TestKeymapStep(t *testing.T) {
	keymap, err := NewKeymap(map[string][]string{"quit": {"a", "b"}})
	if err != nil {
		t.Fatalf("NewKeymap: %v", err)
	}
	if action, state := keymap.Step("a"); state != ScanPending || action != "" {
		t.Errorf("Step(a) = %q, %v, want pending", action, state)
	}
	if action, state := keymap.Step("ab"); state != ScanMatched || action != ActionQuit {
		t.Errorf("Step(ab) = %q, %v, want the quit action", action, state)
	}
	if action, state := keymap.Step("ax"); state != ScanNone || action != "" {
		t.Errorf("Step(ax) = %q, %v, want no match", action, state)
	}
}

func TestNewKeymapOverrides(t *testing.T) {
	keymap, err := NewKeymap(map[string][]string{
		"leader": {"Ctrl", "a"},
		"quit":   {"esc"},
		"accept": {"space"},
	})
	if err != nil {
		t.Fatalf("NewKeymap: %v", err)
	}
	// The leader moved to Ctrl+a, so the help binding follows it.
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

// TestNewKeymapDropsABinding covers an empty list, which removes the binding
// instead of leaving it inert.
func TestNewKeymapDropsABinding(t *testing.T) {
	keymap, err := NewKeymap(map[string][]string{"quit": {}})
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

// TestKeymapModifierOrder covers the commutative order of the modifiers. A
// modifier after its key joins the same chord, an upper-case key means
// shift, and a repeated modifier is dropped.
func TestKeymapModifierOrder(t *testing.T) {
	tests := []struct {
		name   string
		tokens []string
		line   string
	}{
		{"a modifier after the key", []string{"Ctrl", "a", "Shift"}, "\x01"},
		{"the same tokens in order", []string{"Ctrl", "Shift", "a"}, "\x01"},
		{"an upper-case key means shift", []string{"Alt", "A"}, "\x1bA"},
		{"a repeated modifier is dropped", []string{"Ctrl", "Ctrl", "a"}, "\x01"},
		{"meta means alt", []string{"Meta", "a"}, "\x1ba"},
		{"a trailing alt joins the key", []string{"Ctrl", "a", "Alt"}, "\x1b\x01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keymap, err := NewKeymap(map[string][]string{"quit": tt.tokens})
			if err != nil {
				t.Fatalf("NewKeymap: %v", err)
			}
			if action, ok := keymap.Match(tt.line); !ok || action != ActionQuit {
				t.Errorf("Match(%q) = %q, %v, want the quit action", tt.line, action, ok)
			}
		})
	}
}

// TestKeymapSeveralKeys covers a binding with more than one key. The modifier
// joins the first key only, so a later key is plain.
func TestKeymapSeveralKeys(t *testing.T) {
	keymap, err := NewKeymap(map[string][]string{"quit": {"Ctrl", "a", "b"}})
	if err != nil {
		t.Fatalf("NewKeymap: %v", err)
	}
	if action, ok := keymap.Match("\x01b"); !ok || action != ActionQuit {
		t.Errorf("Match(Ctrl+a,b) = %q, %v, want the quit action", action, ok)
	}
	if action, ok := keymap.Match("\x01"); ok {
		t.Errorf("a partial sequence matched %q", action)
	}
}

// TestKeymapLeaderExpands covers the leader token, whose list joins the rest
// of the binding before the chords are formed.
func TestKeymapLeaderExpands(t *testing.T) {
	keymap, err := NewKeymap(map[string][]string{"quit": {"<leader>", "Ctrl", "a"}})
	if err != nil {
		t.Fatalf("NewKeymap: %v", err)
	}
	// The leader is Ctrl+x, so the binding is Ctrl+x, then Ctrl+a.
	if action, ok := keymap.Match("\x18\x01"); !ok || action != ActionQuit {
		t.Errorf("Match = %q, %v, want the quit action", action, ok)
	}
}

func TestNewKeymapErrors(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string][]string
		want      string
	}{
		{"unknown action", map[string][]string{"next": {"n"}}, "is not known"},
		{"no leader", map[string][]string{"leader": {}}, "no leader key"},
		{"leader names itself", map[string][]string{"leader": {"<leader>", "x"}}, "cannot name itself"},
		{"modifiers carry no key", map[string][]string{"quit": {"Ctrl"}}, "carry no key"},
		{"unknown key name", map[string][]string{"quit": {"pageup"}}, "is not a single key"},
		{"a word is not a key", map[string][]string{"quit": {"super", "q"}}, "is not a single key"},
		{"control of a long key", map[string][]string{"quit": {"Ctrl", "up"}}, "ctrl needs a single key"},
		{"control of an escape key", map[string][]string{"quit": {"Ctrl", "esc"}}, "does not apply"},
		{"control of a punctuation key", map[string][]string{"quit": {"Ctrl", "?"}}, "does not apply"},
		{"key is not one rune", map[string][]string{"quit": {"notakey"}}, "is not a single key"},
		{"empty key", map[string][]string{"quit": {""}}, "is not a single key"},
		{"two actions on one key", map[string][]string{"help": {"<leader>", "a"}}, "use the same keys"},
		{"a binding is a prefix of another", map[string][]string{"accept": {"a"}, "quit": {"a", "b"}}, "shares a prefix"},
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
	_, err := NewKeymap(map[string][]string{"accept": {"Alt", "esc"}, "help": {"Meta", "esc"}})
	if err == nil {
		t.Fatal("Alt+esc and Meta+esc must clash")
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
		{Action: ActionAccept, Tokens: []string{"<leader>", "a"}, Notation: "<leader> a"},
		{Action: ActionHelp, Tokens: []string{"<leader>", "?"}, Notation: "<leader> ?"},
		{Action: ActionQuit, Tokens: []string{"<leader>", "q"}, Notation: "<leader> q"},
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

// TestIsModifier covers the modifier names, which are case-insensitive.
func TestIsModifier(t *testing.T) {
	for _, token := range []string{"Ctrl", "CTRL", "alt", "Meta", "shift", "SHIFT"} {
		if !isModifier(token) {
			t.Errorf("isModifier(%q) = false, want true", token)
		}
	}
	for _, token := range []string{"q", "esc", "super", ""} {
		if isModifier(token) {
			t.Errorf("isModifier(%q) = true, want false", token)
		}
	}
}

// TestParseTokens covers the chord grouping directly.
func TestParseTokens(t *testing.T) {
	tests := []struct {
		name   string
		tokens []string
		want   string
	}{
		{"plain key", []string{"q"}, "q"},
		{"control letter", []string{"Ctrl", "c"}, "\x03"},
		{"control letter in upper case", []string{"CTRL", "C"}, "\x03"},
		{"control space", []string{"Ctrl", "space"}, "\x00"},
		{"control at", []string{"Ctrl", "@"}, "\x00"},
		{"control bracket", []string{"Ctrl", "["}, "\x1b"},
		{"control underscore", []string{"Ctrl", "_"}, "\x1f"},
		{"shift of a letter", []string{"Shift", "a"}, "A"},
		{"upper case means shift", []string{"A"}, "A"},
		{"shift of a named key changes nothing", []string{"Shift", "esc"}, "\x1b"},
		{"alt of a letter", []string{"Alt", "x"}, "\x1bx"},
		{"named key in upper case", []string{"ESC"}, "\x1b"},
		{"named arrow", []string{"up"}, "\x1b[A"},
		{"every modifier at once", []string{"Ctrl", "Alt", "Shift", "r"}, "\x1b\x12"},
		{"several chords", []string{"a", "b"}, "ab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTokens(tt.tokens)
			if err != nil {
				t.Fatalf("parseTokens(%v): %v", tt.tokens, err)
			}
			if got != tt.want {
				t.Errorf("parseTokens(%v) = %q, want %q", tt.tokens, got, tt.want)
			}
		})
	}
}

func TestParseTokensErrors(t *testing.T) {
	tests := []struct {
		tokens []string
		want   string
	}{
		{[]string{"<leader>"}, "cannot name itself"},
		{[]string{"Ctrl"}, "carry no key"},
		{[]string{"Ctrl", "?"}, "does not apply"},
	}
	for _, tt := range tests {
		if _, err := parseTokens(tt.tokens); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("parseTokens(%v) error = %v, want it to carry %q", tt.tokens, err, tt.want)
		}
	}
}

// TestExpandTokens covers the leader substitution inside a binding.
func TestExpandTokens(t *testing.T) {
	leader := []string{"Ctrl", "x"}
	want := []string{"Ctrl", "x", "a"}
	got := expandTokens([]string{"<leader>", "a"}, leader)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expandTokens = %v, want %v", got, want)
	}
	if got := expandTokens([]string{"esc", "q"}, nil); !reflect.DeepEqual(got, []string{"esc", "q"}) {
		t.Fatalf("expandTokens without a leader token = %v", got)
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
