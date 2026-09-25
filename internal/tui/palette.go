// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

// The probe asks a terminal for its colours with the operating system
// command (OSC) sequences. OSC 4 asks for an indexed colour, OSC 10 for the
// foreground, and OSC 11 for the background. A question mark stands where
// the terminal writes the value, and the string terminator ends each query.
const (
	oscIntroducer = "\x1b]"
	oscTerminator = "\x1b\\"
)

// indexedColours is the number of indexed colours in a query. The first
// sixteen entries are the ANSI palette.
const indexedColours = 16

// replyBuffer bounds the answers that the reading goroutine sends before a
// reader collects them. The probe asks for eighteen colours, so the buffer
// holds them with room to spare.
const replyBuffer = 64

// queries returns the OSC queries of the probe. It asks for the sixteen
// indexed colours, the foreground, and the background.
func queries() string {
	var b strings.Builder
	for i := 0; i < indexedColours; i++ {
		fmt.Fprintf(&b, "%s4;%d;?%s", oscIntroducer, i, oscTerminator)
	}
	b.WriteString(oscIntroducer + "10;?" + oscTerminator)
	b.WriteString(oscIntroducer + "11;?" + oscTerminator)
	return b.String()
}

// Palette is a colour scheme read from a terminal.
//
// The sixteen indexed colours are the ANSI palette. The foreground and the
// background carry no index. Known is false when the terminal gave no
// answer, so a caller knows that the values come from PlainPalette.
type Palette struct {
	Foreground model.Colour
	Background model.Colour
	Indexed    [indexedColours]model.Colour
	Known      bool
}

// PlainPalette returns the fallback scheme: a white foreground and a black
// background. Known stays false, so a caller knows that no terminal
// answered.
func PlainPalette() Palette {
	return Palette{
		Foreground: model.NewColour(255, 255, 255, 255),
		Background: model.NewColour(0, 0, 0, 255),
	}
}

// Probe asks the terminal for its palette. It writes the OSC 4, 10, and 11
// queries to out, then reads the answers from in.
//
// A terminal that ignores the query sends nothing. The probe then waits for
// timeout and returns the plain palette, so the call never hangs. The read
// runs on a goroutine, because a terminal keeps its input open after the
// answers and never reports the end of the stream. A silent terminal leaves
// that goroutine on the input, so a command that calls Probe must first put
// the terminal in raw mode and set a read deadline. Without that, the
// goroutine consumes the next line the user types.
func Probe(in io.Reader, out io.Writer, timeout time.Duration) Palette {
	palette := PlainPalette()
	if _, err := io.WriteString(out, queries()); err != nil {
		return palette
	}
	for _, r := range readReplies(in, timeout) {
		palette.apply(r)
	}
	return palette
}

// reply is one OSC answer. An indexed colour carries an index, and the
// foreground and the background carry no index.
type reply struct {
	code  int
	index int
	value string
}

// readReplies returns the answers that arrive before the reader ends or the
// timeout expires. An answer that arrived before the timeout is kept, so a
// partial answer still helps.
func readReplies(in io.Reader, timeout time.Duration) []reply {
	answers := make(chan reply, replyBuffer)
	go func() {
		defer close(answers)
		scanReplies(in, answers)
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	var replies []reply
	for {
		select {
		case r, ok := <-answers:
			if !ok {
				return replies
			}
			replies = append(replies, r)
		case <-timer.C:
			return replies
		}
	}
}

// scanReplies reads the OSC answers from in and sends each one on answers.
// It stops at the end of the input and as soon as the palette is complete.
func scanReplies(in io.Reader, answers chan<- reply) {
	reader := bufio.NewReader(in)
	var seen []reply
	for {
		body, ok := readOSCBody(reader)
		if !ok {
			return
		}
		r, ok := parseReply(body)
		if !ok {
			continue
		}
		answers <- r
		seen = append(seen, r)
		if complete(seen) {
			return
		}
	}
}

// complete reports whether the answers carry the foreground and the
// background. These two colours pivot a theme, so the probe stops early
// instead of waiting for the rest of the timeout. The indexed colours that
// arrived before them stay in the palette.
func complete(replies []reply) bool {
	var foreground, background bool
	for _, r := range replies {
		switch r.code {
		case 10:
			foreground = true
		case 11:
			background = true
		}
	}
	return foreground && background
}

// readOSCBody returns the body of the next OSC sequence on reader. It
// reports false at the end of the input. The body excludes the introducer
// and the terminator.
func readOSCBody(reader *bufio.Reader) (string, bool) {
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return "", false
		}
		if b != '\x1b' {
			continue
		}
		next, err := reader.ReadByte()
		if err != nil {
			return "", false
		}
		if next != ']' {
			// A stray escape. Keep looking for the introducer.
			continue
		}
		return readOSCValue(reader)
	}
}

// readOSCValue reads the bytes of an OSC body up to the BEL or the escape
// that ends the sequence.
func readOSCValue(reader *bufio.Reader) (string, bool) {
	var body []byte
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return "", false
		}
		if b == '\x07' || b == '\x1b' {
			return string(body), true
		}
		body = append(body, b)
	}
}

// parseReply reads an OSC body and returns the answer it carries. The shape
// is "<code>;<value>" for the foreground and the background, and
// "<code>;<index>;<value>" for an indexed colour. A body of another shape,
// or a body of another query, yields no answer.
func parseReply(body string) (reply, bool) {
	fields := strings.Split(body, ";")
	if len(fields) < 2 {
		return reply{}, false
	}
	code, err := strconv.Atoi(strings.TrimSpace(fields[0]))
	if err != nil {
		return reply{}, false
	}
	switch code {
	case 10, 11:
		return reply{code: code, value: fields[1]}, true
	case 4:
		if len(fields) < 3 {
			return reply{}, false
		}
		index, err := strconv.Atoi(strings.TrimSpace(fields[1]))
		if err != nil {
			return reply{}, false
		}
		return reply{code: code, index: index, value: fields[2]}, true
	}
	return reply{}, false
}

// apply adds one answer to the palette. A value that does not parse is
// ignored, so a stray answer does not spoil a good one.
func (p *Palette) apply(r reply) {
	colour, ok := parseColour(r.value)
	if !ok {
		return
	}
	switch r.code {
	case 10:
		p.Foreground = colour
	case 11:
		p.Background = colour
	case 4:
		if r.index >= 0 && r.index < len(p.Indexed) {
			p.Indexed[r.index] = colour
		}
	}
	p.Known = true
}

// parseColour reads a colour value from an OSC answer. The xterm form is
// "rgb:RRRR/GGGG/BBBB", and some terminals write "#RRGGBB" or "#RGB".
func parseColour(value string) (model.Colour, bool) {
	value = strings.TrimSpace(value)
	switch {
	case strings.HasPrefix(value, "rgb:"):
		return parseRGB(strings.TrimPrefix(value, "rgb:"))
	case strings.HasPrefix(value, "#"):
		return parseHex(strings.TrimPrefix(value, "#"))
	}
	return model.Colour{}, false
}

// parseRGB reads the three slash-separated components of an "rgb:" value.
func parseRGB(value string) (model.Colour, bool) {
	parts := strings.Split(value, "/")
	if len(parts) < 3 {
		return model.Colour{}, false
	}
	r, ok := parseChannel(parts[0])
	if !ok {
		return model.Colour{}, false
	}
	g, ok := parseChannel(parts[1])
	if !ok {
		return model.Colour{}, false
	}
	b, ok := parseChannel(parts[2])
	if !ok {
		return model.Colour{}, false
	}
	return model.NewColour(r, g, b, 255), true
}

// parseChannel scales one hexadecimal channel onto a byte. The xterm form
// uses four digits, other terminals use two, and a short form is accepted
// too.
func parseChannel(value string) (uint8, bool) {
	value = strings.TrimSpace(value)
	if len(value) == 0 || len(value) > 4 {
		return 0, false
	}
	n, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return 0, false
	}
	scale := uint64(1)<<(4*uint(len(value))) - 1
	return uint8(n * 255 / scale), true
}

// parseHex reads the "#RRGGBB" and "#RGB" forms.
func parseHex(value string) (model.Colour, bool) {
	value = strings.TrimSpace(value)
	if len(value) == 3 {
		value = string([]byte{value[0], value[0], value[1], value[1], value[2], value[2]})
	}
	if len(value) != 6 {
		return model.Colour{}, false
	}
	return parseRGB(value[0:2] + "/" + value[2:4] + "/" + value[4:6])
}
