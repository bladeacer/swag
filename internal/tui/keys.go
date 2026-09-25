// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"fmt"
	"sort"
	"strings"
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

// leaderToken stands in for the leader key inside a binding.
const leaderToken = "<leader>"

// DefaultKeybinds returns the built-in bindings of the interactive mode. An
// entry of the configuration file replaces the default of the same action,
// and an empty entry removes the default.
func DefaultKeybinds() map[string]string {
	return map[string]string{
		actionLeader: "ctrl+x",
		ActionAccept: leaderToken + "a",
		ActionHelp:   leaderToken + "?",
		ActionQuit:   leaderToken + "q",
	}
}

// Binding is one action and the notation that runs it.
type Binding struct {
	Action   string
	Notation string
}

// Keymap matches a typed line against the bindings of the interactive mode.
type Keymap struct {
	match   map[string]string
	binding map[string]string
}

// NewKeymap builds a keymap from the built-in bindings and the entries of
// the configuration file. An unknown action is an error, so a typo never
// leaves a binding inert in silence.
func NewKeymap(overrides map[string]string) (*Keymap, error) {
	bindings := DefaultKeybinds()
	for action, notation := range overrides {
		if !knownAction(action) {
			return nil, fmt.Errorf("tui: the keybind action %q is not known", action)
		}
		if strings.TrimSpace(notation) == "" {
			delete(bindings, action)
			continue
		}
		bindings[action] = notation
	}
	leaderNotation, ok := bindings[actionLeader]
	if !ok {
		return nil, fmt.Errorf("tui: the keybindings carry no leader key")
	}
	leader, err := leaderTokens(leaderNotation)
	if err != nil {
		return nil, fmt.Errorf("tui: the leader binding %q: %w", leaderNotation, err)
	}

	keymap := &Keymap{match: map[string]string{}, binding: map[string]string{}}
	for action, notation := range bindings {
		if action == actionLeader {
			continue
		}
		sequence, err := parseTokens(expandTokens(notation, leader))
		if err != nil {
			return nil, fmt.Errorf("tui: the %s binding %q: %w", action, notation, err)
		}
		if other, taken := keymap.match[sequence]; taken {
			return nil, fmt.Errorf("tui: the bindings %s and %s use the same keys", other, action)
		}
		keymap.match[sequence] = action
		keymap.binding[action] = notation
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

// Bindings returns the bindings in action order, for the help text.
func (m *Keymap) Bindings() []Binding {
	bindings := make([]Binding, 0, len(m.binding))
	for action, notation := range m.binding {
		bindings = append(bindings, Binding{Action: action, Notation: notation})
	}
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].Action < bindings[j].Action })
	return bindings
}

// knownAction reports whether the interactive mode runs the action.
func knownAction(action string) bool {
	switch action {
	case ActionAccept, ActionHelp, ActionQuit, actionLeader:
		return true
	}
	return false
}

// leaderTokens returns the key tokens of the leader notation. The leader
// cannot name itself.
func leaderTokens(notation string) ([]string, error) {
	tokens := strings.Fields(notation)
	for _, token := range tokens {
		if strings.HasPrefix(token, leaderToken) {
			return nil, fmt.Errorf("the leader cannot name itself")
		}
	}
	return tokens, nil
}

// expandTokens replaces the leader token with the keys of the leader. The
// leader token stands alone or starts a larger key, so <leader>a and
// <leader> a name the same two keys, and a notation reads either way.
func expandTokens(notation string, leader []string) []string {
	var out []string
	for _, token := range strings.Fields(notation) {
		rest, found := strings.CutPrefix(token, leaderToken)
		if !found {
			out = append(out, token)
			continue
		}
		out = append(out, leader...)
		if rest != "" {
			out = append(out, rest)
		}
	}
	return out
}

// parseTokens turns a list of key tokens into the bytes of the sequence.
func parseTokens(tokens []string) (string, error) {
	var out strings.Builder
	for _, token := range tokens {
		bytes, err := parseKey(token)
		if err != nil {
			return "", err
		}
		out.WriteString(bytes)
	}
	return out.String(), nil
}

// namedKeys maps the name of a key onto the bytes that a terminal sends.
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

// parseKey turns one key of a binding into the bytes that a terminal sends.
// A modifier list joins with +, so ctrl+shift+r is one key. A control key
// carries the control byte of its letter, and the shift of a letter is
// invisible there, because a terminal sends the same byte either way.
func parseKey(token string) (string, error) {
	parts := strings.Split(token, "+")
	base := parts[len(parts)-1]
	var ctrl, alt, shift bool
	for _, mod := range parts[:len(parts)-1] {
		switch strings.ToLower(mod) {
		case "ctrl":
			ctrl = true
		case "alt", "meta":
			alt = true
		case "shift":
			shift = true
		default:
			return "", fmt.Errorf("the modifier %q is not known", mod)
		}
	}

	named, isNamed := namedKeys[strings.ToLower(base)]
	if !isNamed {
		if len([]rune(base)) != 1 {
			return "", fmt.Errorf("the key %q is not a single key", base)
		}
		named = base
	}
	if ctrl {
		runes := []rune(named)
		if len(runes) != 1 {
			return "", fmt.Errorf("ctrl needs a single key, and %q is not one", base)
		}
		control, err := controlByte(runes[0])
		if err != nil {
			return "", err
		}
		return prefixAlt(string(control), alt), nil
	}
	if !isNamed && shift {
		named = strings.ToUpper(named)
	}
	return prefixAlt(named, alt), nil
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
