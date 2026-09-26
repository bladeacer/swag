// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// The actions that a key binding runs in the interactive mode.
const (
	// ActionAccept answers the current question with its default value.
	ActionAccept = "accept"
	// ActionHelp prints the bindings and asks again.
	ActionHelp = "help"
	// ActionQuit ends the run without a write.
	ActionQuit = "quit"
)

// actionLeader names the binding of the leader key. The key itself runs no
// action, and a later key of a binding writes it as <leader>.
const actionLeader = "leader"

// leaderToken stands in for the leader key inside a binding. A binding
// carries the token where the leader key belongs.
const leaderToken = "<leader>"

// DefaultKeybinds returns the built-in bindings of the interactive mode.
// Each value lists the tokens of one binding in order. A modifier token
// joins the key that follows it, or the key before it at the end of the
// list. An entry of the configuration file replaces the default of the same
// action, and an empty entry removes the default.
func DefaultKeybinds() map[string][]string {
	return map[string][]string{
		actionLeader: {"Ctrl", "x"},
		ActionAccept: {leaderToken, "a"},
		ActionHelp:   {leaderToken, "?"},
		ActionQuit:   {leaderToken, "q"},
	}
}

// Binding is one action and the tokens that run it.
type Binding struct {
	Action   string
	Tokens   []string
	Notation string
}

// ScanState reports how far the typed keys have come against the bindings.
type ScanState uint8

const (
	// ScanNone reports that the typed keys match no binding and no prefix.
	ScanNone ScanState = iota
	// ScanPending reports that the typed keys are a prefix of a binding,
	// so more keys can complete it.
	ScanPending
	// ScanMatched reports that the typed keys complete a binding.
	ScanMatched
)

// Keymap matches a typed line or a key sequence against the bindings of the
// interactive mode.
type Keymap struct {
	match   map[string]string
	binding map[string]Binding
	trie    *trieNode
}

// trieNode is one node of the byte trie of the canonical key sequences.
type trieNode struct {
	children map[byte]*trieNode
	action   string
}

// NewKeymap builds a keymap from the built-in bindings and the entries of
// the configuration file. An unknown action is an error, so a typo never
// leaves a binding inert in silence. Two bindings that share a prefix are an
// error too, because the end of the shorter chain is ambiguous.
func NewKeymap(overrides map[string][]string) (*Keymap, error) {
	bindings := DefaultKeybinds()
	for action, tokens := range overrides {
		if !knownAction(action) {
			return nil, fmt.Errorf("tui: the keybind action %q is not known", action)
		}
		if len(tokens) == 0 {
			delete(bindings, action)
			continue
		}
		bindings[action] = append([]string(nil), tokens...)
	}
	leaderTokens, ok := bindings[actionLeader]
	if !ok {
		return nil, fmt.Errorf("tui: the keybindings carry no leader key")
	}
	if _, err := parseTokens(leaderTokens); err != nil {
		return nil, fmt.Errorf("tui: the leader binding %v: %w", leaderTokens, err)
	}

	keymap := &Keymap{match: map[string]string{}, binding: map[string]Binding{}, trie: &trieNode{}}
	for action, tokens := range bindings {
		if action == actionLeader {
			continue
		}
		sequence, err := parseTokens(expandTokens(tokens, leaderTokens))
		if err != nil {
			return nil, fmt.Errorf("tui: the %s binding %v: %w", action, tokens, err)
		}
		if other, taken := keymap.match[sequence]; taken {
			return nil, fmt.Errorf("tui: the bindings %s and %s use the same keys", other, action)
		}
		if err := keymap.trie.insert(sequence, action); err != nil {
			return nil, fmt.Errorf("tui: the binding %s shares a prefix with another binding", action)
		}
		keymap.match[sequence] = action
		keymap.binding[action] = Binding{Action: action, Tokens: append([]string(nil), tokens...), Notation: strings.Join(tokens, " ")}
	}
	return keymap, nil
}

// Match returns the action of a typed line, and reports whether a binding
// matched it. The line is tried as typed and then without the space around
// it, so a binding of the space key still works. A missing keymap matches
// nothing, so a prompt with no bindings reads every line as an answer.
func (m *Keymap) Match(line string) (string, bool) {
	if m == nil {
		return "", false
	}
	if action, ok := m.match[line]; ok {
		return action, true
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == line {
		return "", false
	}
	action, ok := m.match[trimmed]
	return action, ok
}

// Step reports how far the typed keys have come against the bindings. It
// returns the action when the keys complete a binding, and ScanPending while
// the keys are a prefix of a longer binding.
func (m *Keymap) Step(keys string) (string, ScanState) {
	if m == nil || m.trie == nil {
		return "", ScanNone
	}
	node := m.trie
	for i := 0; i < len(keys); i++ {
		child := node.children[keys[i]]
		if child == nil {
			return "", ScanNone
		}
		node = child
	}
	if node.action != "" {
		return node.action, ScanMatched
	}
	if len(node.children) > 0 {
		return "", ScanPending
	}
	return "", ScanNone
}

// Bindings returns the bindings in action order, for the help text.
func (m *Keymap) Bindings() []Binding {
	bindings := make([]Binding, 0, len(m.binding))
	for _, binding := range m.binding {
		bindings = append(bindings, binding)
	}
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].Action < bindings[j].Action })
	return bindings
}

// insert adds a canonical key sequence to the trie. It refuses a sequence
// that is a prefix of another sequence, or that carries another sequence as
// its prefix, because the end of the shorter chain is ambiguous.
func (t *trieNode) insert(sequence, action string) error {
	node := t
	for i := 0; i < len(sequence); i++ {
		if node.action != "" {
			return fmt.Errorf("prefix conflict")
		}
		if node.children == nil {
			node.children = map[byte]*trieNode{}
		}
		child := node.children[sequence[i]]
		if child == nil {
			child = &trieNode{}
			node.children[sequence[i]] = child
		}
		node = child
	}
	if node.action != "" || len(node.children) > 0 {
		return fmt.Errorf("prefix conflict")
	}
	node.action = action
	return nil
}

// knownAction reports whether the interactive mode runs the action.
func knownAction(action string) bool {
	switch action {
	case ActionAccept, ActionHelp, ActionQuit, actionLeader:
		return true
	}
	return false
}

// modifierSet carries the modifiers of one chord. Each modifier appears at
// most once, so a repeated name in the configuration file changes nothing.
type modifierSet struct {
	ctrl  bool
	alt   bool
	shift bool
}

// add folds a modifier token into the set. A token that is not a modifier is
// left alone.
func (m modifierSet) add(token string) modifierSet {
	switch {
	case strings.EqualFold(token, "ctrl"):
		m.ctrl = true
	case strings.EqualFold(token, "alt"), strings.EqualFold(token, "meta"):
		m.alt = true
	case strings.EqualFold(token, "shift"):
		m.shift = true
	}
	return m
}

// isModifier reports whether a token names a modifier.
func isModifier(token string) bool {
	switch {
	case strings.EqualFold(token, "ctrl"):
		return true
	case strings.EqualFold(token, "alt"), strings.EqualFold(token, "meta"):
		return true
	case strings.EqualFold(token, "shift"):
		return true
	}
	return false
}

// expandTokens replaces the leader token with the tokens of the leader. The
// leader token stands alone, so <leader> and <leader> a name the same two
// chords, and a notation reads either way.
func expandTokens(tokens, leader []string) []string {
	out := make([]string, 0, len(tokens)+len(leader))
	for _, token := range tokens {
		if token == leaderToken {
			out = append(out, leader...)
			continue
		}
		out = append(out, token)
	}
	return out
}

// parseTokens turns a token list into the bytes of the key sequence. A
// modifier joins the key that follows it. A modifier at the end of the list
// joins the key before it, so ctrl, x, shift and ctrl, shift, x name the
// same chord. The order of modifiers is free, and a repeated modifier is
// dropped. A token with no key is a mistake.
func parseTokens(tokens []string) (string, error) {
	type chord struct {
		key  string
		mods modifierSet
	}
	var chords []chord
	var pending modifierSet
	for _, token := range tokens {
		switch {
		case token == leaderToken:
			return "", fmt.Errorf("the leader cannot name itself")
		case isModifier(token):
			pending = pending.add(token)
		default:
			chords = append(chords, chord{key: token, mods: pending})
			pending = modifierSet{}
		}
	}
	if pending.ctrl || pending.alt || pending.shift {
		if len(chords) == 0 {
			return "", fmt.Errorf("the modifiers carry no key")
		}
		last := &chords[len(chords)-1]
		last.mods.ctrl = last.mods.ctrl || pending.ctrl
		last.mods.alt = last.mods.alt || pending.alt
		last.mods.shift = last.mods.shift || pending.shift
	}

	var out strings.Builder
	for _, c := range chords {
		bytes, err := encodeChord(c.key, c.mods)
		if err != nil {
			return "", err
		}
		out.WriteString(bytes)
	}
	return out.String(), nil
}

// namedKeys maps the name of a key onto the bytes that a terminal sends. A
// name is case-insensitive.
var namedKeys = map[string]string{
	"esc":       "\x1b",
	"tab":       "\t",
	"space":     " ",
	"backspace": "\x7f",
	"up":        "\x1b[A",
	"down":      "\x1b[B",
	"right":     "\x1b[C",
	"left":      "\x1b[D",
}

// resolveKey maps a key token onto the bytes that name it, and reports
// whether the token is a named key. A token with no name must be a single
// rune. A named key is case-insensitive, and every other key keeps its case.
func resolveKey(token string) (string, bool, error) {
	if named, ok := namedKeys[strings.ToLower(token)]; ok {
		return named, true, nil
	}
	if len([]rune(token)) != 1 {
		return "", false, fmt.Errorf("the key %q is not a single key", token)
	}
	return token, false, nil
}

// encodeChord turns one key and its modifiers into the bytes that a
// terminal sends. A single upper-case letter means shift with the lower-case
// letter, so Ctrl, X and Ctrl, Shift, x name the same chord. The shift of a
// named key changes nothing. A modifier name is case-insensitive, and a
// regular key keeps its case.
func encodeChord(token string, mods modifierSet) (string, error) {
	base, named, err := resolveKey(token)
	if err != nil {
		return "", err
	}
	runes := []rune(base)
	if mods.ctrl {
		if len(runes) != 1 {
			return "", fmt.Errorf("ctrl needs a single key, and %q is not one", token)
		}
		control, err := controlByte(runes[0])
		if err != nil {
			return "", err
		}
		return prefixAlt(string(control), mods.alt), nil
	}
	if named {
		return prefixAlt(base, mods.alt), nil
	}
	r := runes[0]
	if unicode.IsLetter(r) && unicode.IsUpper(r) {
		mods.shift = true
		r = unicode.ToLower(r)
	}
	if mods.shift {
		r = unicode.ToUpper(r)
	}
	return prefixAlt(string(r), mods.alt), nil
}

// prefixAlt adds the escape byte that a terminal sends before an alt key.
func prefixAlt(bytes string, alt bool) string {
	if alt {
		return "\x1b" + bytes
	}
	return bytes
}

// controlByte returns the byte that a terminal sends for a control key
// combination.
func controlByte(r rune) (byte, error) {
	switch {
	case r >= 'a' && r <= 'z':
		return byte(r-'a') + 1, nil
	case r >= 'A' && r <= 'Z':
		return byte(r-'A') + 1, nil
	case r == ' ' || r == '@':
		return 0x00, nil
	case r >= '[' && r <= '_':
		return byte(r-'[') + 0x1b, nil
	}
	return 0, fmt.Errorf("ctrl does not apply to %q", r)
}
