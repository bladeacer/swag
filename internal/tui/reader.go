// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"errors"
	"io"
	"strings"
	"time"
)

// ErrTimeout reports that a key source saw no key before the timeout. A
// reader treats the timeout as the end of a key sequence.
var ErrTimeout = errors.New("tui: key timeout")

// ByteSource yields one byte at a time. A source that cannot time a read
// blocks instead, and the caller then reads a whole line.
type ByteSource interface {
	// Next returns the next byte. It returns ErrTimeout when the timeout
	// expires before a byte arrives, and io.EOF at the end of the input.
	Next(timeout time.Duration) (byte, error)
	// Close releases the state of the source, for example a terminal in
	// raw mode.
	Close()
}

// Reader reads one answer through the keymap. A linewise reader reads a
// whole line and matches it once. An incremental reader reads one key at a
// time, so a sequence ends as soon as it completes a binding, and a timeout
// ends a sequence that no longer waits for a key.
type Reader struct {
	// Keys matches the typed keys. A nil keymap matches nothing.
	Keys *Keymap
	// Source yields the bytes of the input.
	Source ByteSource
	// Timeout bounds the wait for the next key of a sequence.
	Timeout time.Duration
	// Linewise reads a whole line before it matches, so a pipe or a script
	// keeps the line prompt.
	Linewise bool
	// Echo receives the printable bytes of the answer. A nil Echo writes
	// nothing.
	Echo io.Writer
}

// ReadAnswer returns one answer line, or the action of a matched chord with
// bound set. The line is the text of the answer, and a read failure comes
// back as an error. An empty input returns io.EOF.
func (r *Reader) ReadAnswer() (line, action string, bound bool, err error) {
	if r.Linewise {
		return r.readLine()
	}
	return r.readSequence()
}

// readLine reads one line and matches it whole, so a pipe or a script keeps
// the line prompt.
func (r *Reader) readLine() (string, string, bool, error) {
	var buf []byte
	for {
		b, err := r.Source.Next(0)
		if err != nil {
			if errors.Is(err, io.EOF) {
				if len(buf) == 0 {
					return "", "", false, io.EOF
				}
				return r.matchLine(buf)
			}
			if errors.Is(err, ErrTimeout) {
				continue
			}
			return "", "", false, err
		}
		if b == '\n' {
			return r.matchLine(buf)
		}
		buf = append(buf, b)
	}
}

// matchLine matches a whole line against the bindings.
func (r *Reader) matchLine(buf []byte) (string, string, bool, error) {
	line := strings.TrimSuffix(string(buf), "\r")
	if action, ok := r.Keys.Match(line); ok {
		return "", action, true, nil
	}
	return strings.TrimSpace(line), "", false, nil
}

// readSequence reads one key at a time. The leading keys are a sequence
// while they remain a prefix of a binding. A key that leaves the bindings,
// or a timeout, turns the read into a text answer until the end of the line.
func (r *Reader) readSequence() (string, string, bool, error) {
	var sequence, text []byte
	inText := false
	for {
		b, err := r.Source.Next(r.Timeout)
		if err != nil {
			switch {
			case errors.Is(err, ErrTimeout):
				if !inText && len(sequence) > 0 {
					text = append(text, sequence...)
					r.echo(sequence)
					sequence = sequence[:0]
					inText = true
				}
				continue
			case errors.Is(err, io.EOF):
				if !inText && len(sequence) == 0 && len(text) == 0 {
					return "", "", false, io.EOF
				}
				return r.finish(sequence, text, inText), "", false, nil
			default:
				return "", "", false, err
			}
		}
		if b == '\n' || b == '\r' {
			return r.finish(sequence, text, inText), "", false, nil
		}
		if inText {
			if b == '\x7f' || b == '\b' {
				text = dropLastByte(text)
				r.erase()
				continue
			}
			text = append(text, b)
			r.echo([]byte{b})
			continue
		}
		sequence = append(sequence, b)
		action, state := r.Keys.Step(string(sequence))
		switch state {
		case ScanMatched:
			return "", action, true, nil
		case ScanPending:
			continue
		default:
			text = append(text, sequence...)
			r.echo(sequence)
			sequence = sequence[:0]
			inText = true
		}
	}
}

// finish closes a sequence at the end of the line. A sequence that has not
// completed a binding by now joins the text, because a completed binding
// fires as soon as its last key arrives.
func (r *Reader) finish(sequence, text []byte, inText bool) string {
	if !inText {
		text = append(text, sequence...)
	}
	return strings.TrimSpace(string(text))
}

// echo writes the printable bytes of the answer to the echo sink.
func (r *Reader) echo(bytes []byte) {
	if r.Echo == nil {
		return
	}
	_, _ = r.Echo.Write(bytes)
}

// erase removes the last character from the terminal row.
func (r *Reader) erase() {
	if r.Echo == nil {
		return
	}
	_, _ = io.WriteString(r.Echo, "\b \b")
}

// dropLastByte removes the last byte of a text buffer. The answer is a byte
// buffer, so a backspace removes the last byte and not a whole rune.
func dropLastByte(text []byte) []byte {
	if len(text) == 0 {
		return text
	}
	return text[:len(text)-1]
}
