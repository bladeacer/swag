// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"errors"
	"io"
	"os"
	"time"

	"golang.org/x/term"

	"github.com/bladeacer/swag/internal/tui"
)

// chordTimeout bounds the wait for the next key of a sequence. A pause of
// this length ends the sequence, so the typed keys then read as text.
const chordTimeout = 150 * time.Millisecond

// streamSource reads bytes from a plain reader. The reader can be a pipe or
// a script in a test, so the source carries no timeout and no terminal
// state, and the prompt reads a whole line.
type streamSource struct {
	reader *bufio.Reader
}

// Next returns the next byte. The reader blocks, because the source has no
// timeout.
func (s *streamSource) Next(time.Duration) (byte, error) {
	return s.reader.ReadByte()
}

// Close releases no state.
func (s *streamSource) Close() {}

// terminalSource reads bytes from a terminal in raw mode, so a key arrives
// without the Enter key. A read deadline gives the sequence its timeout.
type terminalSource struct {
	in       *os.File
	state    *term.State
	deadline bool
}

// Next returns the next byte, or ErrTimeout when the deadline expires.
func (s *terminalSource) Next(timeout time.Duration) (byte, error) {
	if s.deadline && timeout > 0 {
		_ = s.in.SetReadDeadline(time.Now().Add(timeout))
	}
	var buf [1]byte
	if _, err := s.in.Read(buf[:]); err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return 0, tui.ErrTimeout
		}
		return 0, err
	}
	return buf[0], nil
}

// Close restores the terminal and clears the read deadline.
func (s *terminalSource) Close() {
	if s.state != nil {
		_ = restoreTerminal(int(s.in.Fd()), s.state)
		s.state = nil
	}
	_ = s.in.SetReadDeadline(time.Time{})
}

// The terminal calls are variables, so a test can cover both the raw setup
// and the plain path without a real terminal.
var (
	isTerminal      = term.IsTerminal
	makeRawTerminal = term.MakeRaw
	restoreTerminal = term.Restore
)

// beginTerminal puts a terminal into raw mode and reports whether a read
// deadline is available. A terminal without the deadline still reads, and
// the timeout then does not apply.
var beginTerminal = func(in *os.File) (tui.ByteSource, error) {
	state, err := makeRawTerminal(int(in.Fd()))
	if err != nil {
		return nil, err
	}
	source := &terminalSource{in: in, state: state}
	if err := in.SetReadDeadline(time.Now()); err != nil {
		source.deadline = false
		return source, nil
	}
	source.deadline = true
	_ = in.SetReadDeadline(time.Time{})
	return source, nil
}

// keySourceFor picks the key source of the interactive prompt. A terminal
// reads one key at a time in raw mode, and every other reader keeps the line
// prompt.
var keySourceFor = func(in io.Reader) tui.ByteSource {
	if file, ok := in.(*os.File); ok && isTerminal(int(file.Fd())) {
		if source, err := beginTerminal(file); err == nil {
			return source
		}
	}
	return &streamSource{reader: bufio.NewReader(in)}
}
