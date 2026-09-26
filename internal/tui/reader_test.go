// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// scriptStep is one reply of a scripted key source.
type scriptStep struct {
	b   byte
	err error
}

// scriptSource yields a fixed list of replies, and io.EOF after them.
type scriptSource struct {
	steps []scriptStep
}

func (s *scriptSource) Next(time.Duration) (byte, error) {
	if len(s.steps) == 0 {
		return 0, io.EOF
	}
	step := s.steps[0]
	s.steps = s.steps[1:]
	return step.b, step.err
}

func (s *scriptSource) Close() {}

// bytesSource turns a string into a scripted source.
func bytesSource(script string) *scriptSource {
	source := &scriptSource{}
	for i := 0; i < len(script); i++ {
		source.steps = append(source.steps, scriptStep{b: script[i]})
	}
	return source
}

func defaultKeymap(t *testing.T) *Keymap {
	t.Helper()
	keymap, err := NewKeymap(nil)
	if err != nil {
		t.Fatalf("NewKeymap: %v", err)
	}
	return keymap
}

// TestReaderLinewise covers the line prompt: a whole line matches once, a
// plain line is an answer, and the trailing carriage return is dropped.
func TestReaderLinewise(t *testing.T) {
	keymap := defaultKeymap(t)
	reader := &Reader{Keys: keymap, Source: bytesSource("\x18q\r\nvtt\n"), Linewise: true}

	line, action, bound, err := reader.ReadAnswer()
	if err != nil || !bound || action != ActionQuit || line != "" {
		t.Fatalf("first answer = %q, %q, %v, %v", line, action, bound, err)
	}
	line, action, bound, err = reader.ReadAnswer()
	if err != nil || bound || action != "" || line != "vtt" {
		t.Fatalf("second answer = %q, %q, %v, %v", line, action, bound, err)
	}
	// The script is exhausted, so the next read reports the end.
	if _, _, _, err := reader.ReadAnswer(); !errors.Is(err, io.EOF) {
		t.Fatalf("the end of the input = %v, want io.EOF", err)
	}
}

// TestReaderLinewiseLastLine covers a last line without a newline, which
// still counts.
func TestReaderLinewiseLastLine(t *testing.T) {
	reader := &Reader{Keys: defaultKeymap(t), Source: bytesSource("vtt"), Linewise: true}
	line, _, bound, err := reader.ReadAnswer()
	if err != nil || bound || line != "vtt" {
		t.Fatalf("answer = %q, %v, %v", line, bound, err)
	}
}

// TestReaderLinewiseTimeout covers a timeout in the line prompt, which reads
// on rather than ending the answer.
func TestReaderLinewiseTimeout(t *testing.T) {
	source := &scriptSource{steps: []scriptStep{{err: ErrTimeout}, {b: 'v'}, {b: 't'}, {b: 't'}, {b: '\n'}}}
	reader := &Reader{Keys: defaultKeymap(t), Source: source, Linewise: true}
	line, _, _, err := reader.ReadAnswer()
	if err != nil || line != "vtt" {
		t.Fatalf("answer = %q, %v", line, err)
	}
}

// TestReaderLinewiseErrors covers a read failure and an empty input.
func TestReaderLinewiseErrors(t *testing.T) {
	reader := &Reader{Keys: defaultKeymap(t), Source: &scriptSource{}, Linewise: true}
	if _, _, _, err := reader.ReadAnswer(); !errors.Is(err, io.EOF) {
		t.Fatalf("an empty input = %v, want io.EOF", err)
	}
	boom := errors.New("boom")
	reader = &Reader{Keys: defaultKeymap(t), Source: &scriptSource{steps: []scriptStep{{err: boom}}}, Linewise: true}
	if _, _, _, err := reader.ReadAnswer(); !errors.Is(err, boom) {
		t.Fatalf("a read failure = %v, want boom", err)
	}
}

// TestReaderSequenceMatches covers the incremental reader: a chord fires as
// soon as its last key arrives.
func TestReaderSequenceMatches(t *testing.T) {
	reader := &Reader{Keys: defaultKeymap(t), Source: bytesSource("\x18q"), Timeout: time.Millisecond}
	line, action, bound, err := reader.ReadAnswer()
	if err != nil || !bound || action != ActionQuit || line != "" {
		t.Fatalf("answer = %q, %q, %v, %v", line, action, bound, err)
	}
}

// TestReaderSequenceText covers a plain line, which leaves the bindings at
// the first key and then reads as text.
func TestReaderSequenceText(t *testing.T) {
	reader := &Reader{Keys: defaultKeymap(t), Source: bytesSource("vtt\n"), Timeout: time.Millisecond}
	line, _, bound, err := reader.ReadAnswer()
	if err != nil || bound || line != "vtt" {
		t.Fatalf("answer = %q, %v, %v", line, bound, err)
	}
}

// TestReaderSequenceTimeout covers a partial chord whose timeout turns the
// typed keys into text.
func TestReaderSequenceTimeout(t *testing.T) {
	source := &scriptSource{steps: []scriptStep{{b: '\x18'}, {err: ErrTimeout}, {b: 'x'}, {b: '\n'}}}
	reader := &Reader{Keys: defaultKeymap(t), Source: source, Timeout: time.Millisecond}
	line, _, bound, err := reader.ReadAnswer()
	if err != nil || bound {
		t.Fatalf("answer = %q, %v, %v", line, bound, err)
	}
	if !strings.Contains(line, "x") {
		t.Fatalf("the timed out keys must join the text: %q", line)
	}
}

// TestReaderSequenceEnd covers the end of the input with and without text.
func TestReaderSequenceEnd(t *testing.T) {
	reader := &Reader{Keys: defaultKeymap(t), Source: &scriptSource{}, Timeout: time.Millisecond}
	if _, _, _, err := reader.ReadAnswer(); !errors.Is(err, io.EOF) {
		t.Fatalf("an empty input = %v, want io.EOF", err)
	}
	reader = &Reader{Keys: defaultKeymap(t), Source: bytesSource("ab"), Timeout: time.Millisecond}
	line, _, _, err := reader.ReadAnswer()
	if err != nil || line != "ab" {
		t.Fatalf("a last line without a newline = %q, %v", line, err)
	}
}

// TestReaderSequenceErrors covers a read failure.
func TestReaderSequenceErrors(t *testing.T) {
	boom := errors.New("boom")
	reader := &Reader{Keys: defaultKeymap(t), Source: &scriptSource{steps: []scriptStep{{err: boom}}}, Timeout: time.Millisecond}
	if _, _, _, err := reader.ReadAnswer(); !errors.Is(err, boom) {
		t.Fatalf("a read failure = %v, want boom", err)
	}
}

// TestReaderEcho covers the terminal echo of a typed answer, including a
// backspace that erases the last character.
func TestReaderEcho(t *testing.T) {
	var out strings.Builder
	source := bytesSource("ab\x7f\n")
	reader := &Reader{Keys: defaultKeymap(t), Source: source, Timeout: time.Millisecond, Echo: &out}
	line, _, _, err := reader.ReadAnswer()
	if err != nil {
		t.Fatalf("answer: %v", err)
	}
	if line != "a" {
		t.Fatalf("a backspace must remove the last character: %q", line)
	}
	if !strings.Contains(out.String(), "ab") || !strings.Contains(out.String(), "\b \b") {
		t.Fatalf("the echo is missing a character or the erase: %q", out.String())
	}
}

// TestReaderEchoWithoutSink covers the echo of a reader with no echo sink,
// which writes nothing, and a backspace with no sink.
func TestReaderEchoWithoutSink(t *testing.T) {
	reader := &Reader{Keys: defaultKeymap(t), Source: bytesSource("ab\n"), Timeout: time.Millisecond}
	if line, _, _, err := reader.ReadAnswer(); err != nil || line != "ab" {
		t.Fatalf("answer = %q, %v", line, err)
	}
	reader = &Reader{Keys: defaultKeymap(t), Source: bytesSource("a\x7f\n"), Timeout: time.Millisecond}
	if line, _, _, err := reader.ReadAnswer(); err != nil || line != "" {
		t.Fatalf("an erased answer = %q, %v", line, err)
	}
}

// TestReaderFinishPendingAtLineEnd covers a partial sequence at the end of a
// line, which joins the text with no text before it.
func TestReaderFinishPendingAtLineEnd(t *testing.T) {
	keymap, err := NewKeymap(map[string][]string{"quit": {"a", "b"}})
	if err != nil {
		t.Fatalf("NewKeymap: %v", err)
	}
	reader := &Reader{Keys: keymap, Source: bytesSource("a\n"), Timeout: time.Millisecond}
	line, _, bound, err := reader.ReadAnswer()
	if err != nil || bound || line != "a" {
		t.Fatalf("answer = %q, %v, %v", line, bound, err)
	}
}

// TestDropLastByteEmpty covers a backspace on an empty buffer.
func TestDropLastByteEmpty(t *testing.T) {
	if got := dropLastByte(nil); len(got) != 0 {
		t.Fatalf("dropLastByte(nil) = %v", got)
	}
}
