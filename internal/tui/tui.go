// SPDX-License-Identifier: Apache-2.0

// Package tui paints a terminal frame and repaints only the rows that
// change.
//
// The renderer holds the last frame it drew. A frame equal to the last one
// writes no bytes at all, and a frame that differs writes only the rows
// whose text changed, each at its own cursor position. A still screen
// therefore costs nothing, and a small change costs a small write, so a
// terminal does not flicker on every tick.
package tui

import (
	"fmt"
	"io"
	"strings"
)

// clearLine erases the row under the cursor. It also blanks the leftover
// rows when a frame is shorter than the one before it.
const clearLine = "\x1b[2K"

// Layout is one frame of the interface. The heading stays at the top, the
// rows follow in order, and the status sits at the bottom. An empty status
// occupies no row.
type Layout struct {
	Heading string
	Rows    []string
	Status  string
}

// frame returns the lines of the layout in draw order.
func (l Layout) frame() []string {
	lines := make([]string, 0, len(l.Rows)+2)
	lines = append(lines, l.Heading)
	lines = append(lines, l.Rows...)
	if l.Status != "" {
		lines = append(lines, l.Status)
	}
	return lines
}

// Changed reports the zero-based row indexes whose content differs between
// prev and next, in draw order. A row that only one of the two carries
// counts as changed.
func Changed(prev, next Layout) []int {
	before, after := prev.frame(), next.frame()
	rows := len(before)
	if len(after) > rows {
		rows = len(after)
	}
	var changed []int
	for i := 0; i < rows; i++ {
		var a, b string
		if i < len(before) {
			a = before[i]
		}
		if i < len(after) {
			b = after[i]
		}
		if a != b {
			changed = append(changed, i)
		}
	}
	return changed
}

// Renderer paints layouts on a sink and remembers the last one it drew.
type Renderer struct {
	sink   io.Writer
	last   *Layout
	frames int
}

// NewRenderer returns a renderer that paints on sink.
func NewRenderer(sink io.Writer) *Renderer {
	return &Renderer{sink: sink}
}

// Frames reports how many frames the renderer has painted.
func (r *Renderer) Frames() int { return r.frames }

// Reset forgets the last frame, so the next Draw paints every row again. A
// caller uses it after a terminal resize.
func (r *Renderer) Reset() { r.last = nil }

// Draw paints next and reports whether it wrote anything. A frame equal to
// the previous one writes no bytes and reports false. A frame that differs
// paints only the rows that changed, each at its own cursor position, and
// clears a row that the new frame leaves out.
func (r *Renderer) Draw(next Layout) (bool, error) {
	if r.last != nil && len(Changed(*r.last, next)) == 0 {
		return false, nil
	}

	var before []string
	if r.last != nil {
		before = r.last.frame()
	}
	after := next.frame()

	rows := len(before)
	if len(after) > rows {
		rows = len(after)
	}
	var out strings.Builder
	for i := 0; i < rows; i++ {
		var old, new string
		if i < len(before) {
			old = before[i]
		}
		if i < len(after) {
			new = after[i]
		}
		// A row survives untouched when both frames carry it with the same
		// text. On the first draw there is no previous row, so every row is
		// painted.
		if r.last != nil && i < len(before) && i < len(after) && old == new {
			continue
		}
		fmt.Fprintf(&out, "\x1b[%d;1H%s%s", i+1, clearLine, new)
	}
	if _, err := io.WriteString(r.sink, out.String()); err != nil {
		return false, err
	}
	r.frames++
	r.last = &next
	return true, nil
}
