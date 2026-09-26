// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/term"

	"github.com/bladeacer/swag/internal/i18n"
	"github.com/bladeacer/swag/internal/tui"
)

func TestStreamSourceNext(t *testing.T) {
	source := &streamSource{reader: bufio.NewReader(strings.NewReader("ab"))}
	if b, err := source.Next(time.Second); err != nil || b != 'a' {
		t.Fatalf("first byte = %q, %v", b, err)
	}
	if b, err := source.Next(time.Second); err != nil || b != 'b' {
		t.Fatalf("second byte = %q, %v", b, err)
	}
	if _, err := source.Next(time.Second); !errors.Is(err, io.EOF) {
		t.Fatalf("the end of the stream = %v, want io.EOF", err)
	}
	source.Close()
}

// TestTerminalSourceTimeout covers a terminal read that reaches its deadline
// before a key arrives, and a read that carries one.
func TestTerminalSourceTimeout(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() { _ = read.Close(); _ = write.Close() })

	source := &terminalSource{in: read, deadline: true}
	if _, err := source.Next(20 * time.Millisecond); !errors.Is(err, tui.ErrTimeout) {
		t.Fatalf("a read with no key = %v, want a timeout", err)
	}
	if _, err := write.Write([]byte("z")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if b, err := source.Next(time.Second); err != nil || b != 'z' {
		t.Fatalf("the next byte = %q, %v", b, err)
	}
	source.Close()
}

// TestTerminalSourceWithoutDeadline covers a source that cannot carry a read
// deadline, which reads without one.
func TestTerminalSourceWithoutDeadline(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "keys")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() { _ = file.Close() }()
	if _, err := file.WriteString("q"); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("seek: %v", err)
	}
	source := &terminalSource{in: file}
	if b, err := source.Next(0); err != nil || b != 'q' {
		t.Fatalf("the next byte = %q, %v", b, err)
	}
}

// TestTerminalSourceReadError covers a read that fails for a reason other
// than the deadline.
func TestTerminalSourceReadError(t *testing.T) {
	file, err := os.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = file.Close() }()
	source := &terminalSource{in: file, deadline: true}
	if _, err := source.Next(time.Second); err == nil || errors.Is(err, tui.ErrTimeout) {
		t.Fatalf("a failing read = %v, want the read error", err)
	}
}

// TestTerminalSourceClose covers the restore of the terminal state.
func TestTerminalSourceClose(t *testing.T) {
	originalRestore := restoreTerminal
	calls := 0
	restoreTerminal = func(int, *term.State) error { calls++; return nil }
	t.Cleanup(func() { restoreTerminal = originalRestore })

	source := &terminalSource{state: &term.State{}}
	source.Close()
	source.Close() // The second close carries no state.
	if calls != 1 {
		t.Fatalf("restore calls = %d, want one", calls)
	}
}

// TestBeginTerminal covers the raw setup that reports a read deadline.
func TestBeginTerminal(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() { _ = read.Close(); _ = write.Close() })

	originalRaw, originalRestore := makeRawTerminal, restoreTerminal
	makeRawTerminal = func(int) (*term.State, error) { return &term.State{}, nil }
	restoreTerminal = func(int, *term.State) error { return nil }
	t.Cleanup(func() { makeRawTerminal, restoreTerminal = originalRaw, originalRestore })

	source, err := beginTerminal(read)
	if err != nil {
		t.Fatalf("beginTerminal: %v", err)
	}
	if terminal, ok := source.(*terminalSource); !ok || !terminal.deadline {
		t.Fatalf("a pipe must carry a read deadline: %#v", source)
	}
	source.Close()
}

// TestBeginTerminalWithoutDeadline covers a file that accepts raw mode but
// cannot carry a read deadline, such as a regular file.
func TestBeginTerminalWithoutDeadline(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "keys")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() { _ = file.Close() }()

	originalRaw := makeRawTerminal
	makeRawTerminal = func(int) (*term.State, error) { return &term.State{}, nil }
	t.Cleanup(func() { makeRawTerminal = originalRaw })

	source, err := beginTerminal(file)
	if err != nil {
		t.Fatalf("beginTerminal: %v", err)
	}
	terminal, ok := source.(*terminalSource)
	if !ok || terminal.deadline {
		t.Fatalf("a regular file must carry no read deadline: %#v", source)
	}
	source.Close()
}

// TestBeginTerminalRawError covers a terminal that refuses raw mode.
func TestBeginTerminalRawError(t *testing.T) {
	originalRaw := makeRawTerminal
	makeRawTerminal = func(int) (*term.State, error) { return nil, errors.New("not a terminal") }
	t.Cleanup(func() { makeRawTerminal = originalRaw })

	if _, err := beginTerminal(os.Stdin); err == nil {
		t.Fatal("a failed raw mode must stop the source")
	}
}

// TestKeySourceFor covers the choice between the terminal source and the
// stream source.
func TestKeySourceFor(t *testing.T) {
	originalTerminal, originalBegin := isTerminal, beginTerminal
	t.Cleanup(func() { isTerminal, beginTerminal = originalTerminal, originalBegin })

	// A plain reader keeps the line prompt.
	if _, ok := keySourceFor(strings.NewReader("x")).(*streamSource); !ok {
		t.Fatal("a plain reader must use the stream source")
	}

	read, write, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() { _ = read.Close(); _ = write.Close() })

	// A file that is not a terminal keeps the line prompt.
	isTerminal = func(int) bool { return false }
	if _, ok := keySourceFor(read).(*streamSource); !ok {
		t.Fatal("a file that is not a terminal must use the stream source")
	}

	// A terminal reads one key at a time.
	isTerminal = func(int) bool { return true }
	beginTerminal = func(*os.File) (tui.ByteSource, error) { return &terminalSource{}, nil }
	if _, ok := keySourceFor(read).(*terminalSource); !ok {
		t.Fatal("a terminal must use the terminal source")
	}

	// A raw setup that fails falls back to the line prompt.
	beginTerminal = func(*os.File) (tui.ByteSource, error) { return nil, errors.New("boom") }
	if _, ok := keySourceFor(read).(*streamSource); !ok {
		t.Fatal("a failed raw setup must use the stream source")
	}
}

// TestDefaultPrompterTerminal covers the echo and the incremental reader of a
// terminal source.
func TestDefaultPrompterTerminal(t *testing.T) {
	original := keySourceFor
	keySourceFor = func(io.Reader) tui.ByteSource { return &terminalSource{} }
	t.Cleanup(func() { keySourceFor = original })

	var out strings.Builder
	p := defaultPrompter(strings.NewReader(""), &out, i18n.New("en-GB"), nil).(*linePrompter)
	if p.reader.Linewise {
		t.Fatal("a terminal source must read one key at a time")
	}
	if p.reader.Echo != &out {
		t.Fatal("a terminal source must echo the answer")
	}
	p.close()
}
