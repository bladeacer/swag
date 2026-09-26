// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bladeacer/swag/internal/i18n"
	"github.com/bladeacer/swag/internal/tui"
	"github.com/bladeacer/swag/pkg/sub"
)

// The terminal commands read their answers from terminalIn and paint their
// frames on terminalOut. A test replaces the pair with a scripted reader and
// a buffer.
var (
	terminalIn  io.Reader = os.Stdin
	terminalOut io.Writer = os.Stdout
)

// errInteractiveQuit stops the run when a user presses the quit key. The
// command turns it into a clean stop, because a quit is not a failure.
var errInteractiveQuit = errors.New("interactive: quit")

// prompter gathers the answers of the interactive command.
type prompter interface {
	// ask prints label and reads one line. An empty answer returns the
	// default value.
	ask(label, defaultValue string) (string, error)
	// askChoice lists options and reads a number or a name. An empty answer
	// returns the default option.
	askChoice(label string, options []string, defaultOption string) (string, error)
	// close releases the terminal state of the source.
	close()
}

// newPrompter builds the prompter of the interactive command. It is a
// variable so a test can script the answers.
var newPrompter = defaultPrompter

// defaultPrompter builds the line prompter over the key source. A terminal
// reads one key at a time in raw mode with a timeout, and every other reader
// keeps the line prompt.
func defaultPrompter(in io.Reader, out io.Writer, t *i18n.T, keys *tui.Keymap) prompter {
	source := keySourceFor(in)
	_, raw := source.(*terminalSource)
	reader := &tui.Reader{Keys: keys, Source: source, Timeout: chordTimeout, Linewise: !raw}
	if raw {
		reader.Echo = out
	}
	return &linePrompter{out: out, t: t, keys: keys, source: source, reader: reader}
}

// identify reports the format of the input file. It is a variable so a test
// can force the failure branch.
var identify = sub.Identify

// linePrompter reads one answer per line and prints the questions with
// pterm styling. It holds no terminal state of its own, so it works on a
// pipe as well as on a terminal.
type linePrompter struct {
	out    io.Writer
	t      *i18n.T
	keys   *tui.Keymap
	source tui.ByteSource
	reader *tui.Reader
}

// ask prints a labelled question and returns the answer. An empty answer
// returns the default value.
func (p *linePrompter) ask(label, defaultValue string) (string, error) {
	question := p.t.F(i18n.MsgInteractivePromptBare, label)
	if defaultValue != "" {
		question = p.t.F(i18n.MsgInteractivePrompt, label, defaultValue)
	}
	if _, err := fmt.Fprintln(p.out, questionStyle.Sprint(question)); err != nil {
		return "", err
	}
	answer, err := p.readLine()
	if err != nil {
		return "", err
	}
	if answer == "" {
		return defaultValue, nil
	}
	return answer, nil
}

// askChoice lists the options and returns the chosen one. A number picks an
// option by its place, a name picks it by its text, and an empty answer
// returns the default option.
func (p *linePrompter) askChoice(label string, options []string, defaultOption string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("%s", p.t.S(i18n.MsgInteractiveNone))
	}
	if _, err := fmt.Fprintln(p.out, questionStyle.Sprint(label)); err != nil {
		return "", err
	}
	for i, option := range options {
		if _, err := fmt.Fprintf(p.out, "  %d) %s\n", i+1, option); err != nil {
			return "", err
		}
	}
	question := p.t.F(i18n.MsgInteractivePick, defaultOption)
	if defaultOption == "" {
		question = p.t.S(i18n.MsgInteractivePickBare)
	}
	if _, err := fmt.Fprintln(p.out, question); err != nil {
		return "", err
	}
	answer, err := p.readLine()
	if err != nil {
		return "", err
	}
	return p.matchChoice(answer, options, defaultOption)
}

// matchChoice maps an answer onto one of the options.
func (p *linePrompter) matchChoice(answer string, options []string, defaultOption string) (string, error) {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return defaultOption, nil
	}
	if n, err := strconv.Atoi(answer); err == nil {
		if n >= 1 && n <= len(options) {
			return options[n-1], nil
		}
		return "", fmt.Errorf("%s", p.t.F(i18n.MsgInteractiveChoice, answer))
	}
	for _, option := range options {
		if strings.EqualFold(option, answer) {
			return option, nil
		}
	}
	return "", fmt.Errorf("%s", p.t.F(i18n.MsgInteractiveChoice, answer))
}

// readLine returns one line of the input with the keymap applied. The accept
// binding answers with the empty string, which the caller reads as its
// default value, and the help binding prints the bindings and asks again.
func (p *linePrompter) readLine() (string, error) {
	for {
		line, action, bound, err := p.reader.ReadAnswer()
		if err != nil {
			return "", fmt.Errorf("%s", p.t.F(i18n.MsgInteractiveRead, err))
		}
		if !bound {
			return line, nil
		}
		switch action {
		case tui.ActionAccept:
			return "", nil
		case tui.ActionQuit:
			return "", errInteractiveQuit
		case tui.ActionHelp:
			for _, row := range helpLines(p.t, p.keys) {
				if _, err := fmt.Fprintln(p.out, row); err != nil {
					return "", err
				}
			}
		}
	}
}

// close releases the terminal state of the source.
func (p *linePrompter) close() {
	if p.source != nil {
		p.source.Close()
	}
}

// helpLines names every binding, so a user can read the keys of the mode.
func helpLines(t *i18n.T, keys *tui.Keymap) []string {
	rows := []string{t.S(i18n.MsgInteractiveHelp)}
	for _, binding := range keys.Bindings() {
		rows = append(rows, t.F(i18n.MsgInteractiveKeybind, binding.Notation, binding.Action))
	}
	return rows
}

// InteractiveCmd walks a user through one conversion: it asks for the input
// file, the target format, and the output file, converts, and paints the
// result. A value given on the command line skips its question.
type InteractiveCmd struct {
	Input  string `help:"Input subtitle file. Asked for when it is empty." short:"i"`
	From   string `help:"Input format name. The content decides when it is empty." short:"F" aliases:"input-format"`
	Target string `help:"Target format name. Asked for when it is empty." short:"f" aliases:"to"`
	Output string `help:"Output file. Asked for when it is empty." short:"o"`
	Font   string `help:"Replace the font of every style and span." short:"n"`
	Strict bool   `help:"Fail when the target format drops a feature." short:"s"`
	// StrictCompat turns the integrity block off, so the output stays
	// inside the original specification of the target format.
	StrictCompat bool `help:"Write no integrity block, so the output stays inside the original format." short:"c"`
}

// Run asks for the missing choices, converts the file, and paints two
// frames. The second frame adds the outcome, and the renderer writes only
// the rows that differ between the two.
func (c *InteractiveCmd) Run(ictx *runContext) error {
	err := c.run(ictx)
	if errors.Is(err, errInteractiveQuit) {
		if _, perr := fmt.Fprintln(terminalOut, ictx.T.S(i18n.MsgInteractiveQuit)); perr != nil {
			return perr
		}
		return nil
	}
	return err
}

// run walks the questions and writes the file. The quit key travels out of
// it as a sentinel, so the wrapper can turn it into a clean stop.
func (c *InteractiveCmd) run(ictx *runContext) error {
	t := ictx.T
	keys, err := tui.NewKeymap(ictx.Settings.Keybinds)
	if err != nil {
		return err
	}
	p := newPrompter(terminalIn, terminalOut, t, keys)
	defer p.close()
	renderer := tui.NewRenderer(terminalOut)

	input := c.Input
	if input == "" {
		answer, err := p.ask(t.S(i18n.MsgInteractiveInput), "")
		if err != nil {
			return err
		}
		input = answer
	}
	if err := checkInput(input, t); err != nil {
		return err
	}
	source, err := openInput(input)

	if err != nil {
		return fmt.Errorf("%s", t.F(i18n.MsgInputUnreadable, err))
	}
	defer func() { _ = source.Close() }()
	data, err := io.ReadAll(source)
	if err != nil {
		return fmt.Errorf("%s", t.F(i18n.MsgInputUnreadable, err))
	}
	detected, err := identify(input, bytes.NewReader(data))
	if err != nil {
		return err
	}

	target := c.Target
	if target == "" {
		answer, err := p.askChoice(t.S(i18n.MsgInteractiveTarget), orderFormats(ictx.Settings.Preferred, sub.Registered()), detected)
		if err != nil {
			return err
		}
		target = answer
	}

	output := c.Output
	if output == "" {
		answer, err := p.ask(t.S(i18n.MsgInteractiveOutput), swapExtension(input, target))
		if err != nil {
			return err
		}
		output = answer
	}

	// The preview needs the document, so the parse happens here and an
	// unreadable file fails before the first frame.
	doc, err := sub.Parse(input, bytes.NewReader(data), c.From)
	if err != nil {
		return err
	}

	if _, err := renderer.Draw(gatherLayout(t, input, detected, target, output)); err != nil {
		return err
	}

	sink, closer, err := openOutput(output, t)
	if err != nil {
		return err
	}
	defer closer()

	opts := sub.Options{Target: target, Format: c.From, Font: c.Font, StrictCompat: c.StrictCompat}
	if c.Strict {
		opts.Loss = sub.LossStrict
	}
	losses, err := sub.ConvertWith(input, bytes.NewReader(data), opts, sink)
	if err != nil {
		return err
	}
	result := resultLayout(t, input, detected, target, output, losses)
	result.Rows = append(result.Rows, "")
	result.Rows = append(result.Rows, previewRows(t, doc, interactivePreviewLimit, terminalColour())...)
	if _, err := renderer.Draw(result); err != nil {
		return err
	}
	return nil
}

// interactivePreviewLimit is the number of cue rows the interactive result
// shows before it summarises the rest.
const interactivePreviewLimit = 10

// gatherLayout is the frame that shows the chosen conversion.
func gatherLayout(t *i18n.T, input, detected, target, output string) tui.Layout {
	return tui.Layout{
		Heading: t.S(i18n.MsgInteractiveTitle),
		Rows: []string{
			t.F(i18n.MsgInteractiveInputLine, input, detected),
			t.F(i18n.MsgInteractiveTargetLine, target),
			t.F(i18n.MsgInteractiveOutputLine, output),
		},
	}
}

// resultLayout adds the outcome and the loss rows to the frame.
func resultLayout(t *i18n.T, input, detected, target, output string, losses []string) tui.Layout {
	layout := gatherLayout(t, input, detected, target, output)
	if len(losses) == 0 {
		layout.Status = t.F(i18n.MsgConvertSuccess, output)
		return layout
	}
	layout.Status = t.F(i18n.MsgInteractiveLosses, len(losses))
	layout.Rows = append(layout.Rows, losses...)
	return layout
}

// orderFormats puts the preferred formats first, in their listed order, then
// the rest of the registry. A preferred name that no format claims is
// dropped, and an empty list leaves the registry order alone.
func orderFormats(preferred, all []string) []string {
	if len(preferred) == 0 {
		return all
	}
	known := make(map[string]bool, len(all))
	for _, name := range all {
		known[name] = true
	}
	ordered := make([]string, 0, len(all))
	seen := make(map[string]bool, len(all))
	for _, name := range preferred {
		if known[name] && !seen[name] {
			ordered = append(ordered, name)
			seen[name] = true
		}
	}
	for _, name := range all {
		if !seen[name] {
			ordered = append(ordered, name)
		}
	}
	return ordered
}

// swapExtension replaces the extension of path with the name of the target
// format, so the default output file sits beside the input.
func swapExtension(path, target string) string {
	if ext := filepath.Ext(path); ext != "" {
		path = strings.TrimSuffix(path, ext)
	}
	return path + "." + target
}
