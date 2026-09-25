// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/bladeacer/swag/internal/model"
)

// reset ends an ANSI colour run.
const reset = "\x1b[0m"

// sampleWidth is the width of a colour swatch in terminal cells.
const sampleWidth = 6

// Hex renders a colour as #RRGGBB.
func Hex(c model.Colour) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// Swatch renders a colour as a block of sampleWidth cells. It falls back to
// the hex value in brackets when the terminal carries no colour.
func Swatch(c model.Colour, colour bool) string {
	block := strings.Repeat(" ", sampleWidth)
	if !colour {
		return "[" + Hex(c) + "]"
	}
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm%s%s", c.R, c.G, c.B, block, reset)
}

// Tint renders s in a colour. It returns s unchanged when the terminal
// carries no colour.
func Tint(c model.Colour, s string, colour bool) string {
	if !colour {
		return s
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s%s", c.R, c.G, c.B, s, reset)
}

// Clock renders a duration as H:MM:SS.mmm.
func Clock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	ms := int64(d / time.Millisecond)
	return fmt.Sprintf("%d:%02d:%02d.%03d", ms/3600000, (ms/60000)%60, (ms/1000)%60, ms%1000)
}

// StyleRows returns one preview row per style. A row carries a swatch of the
// primary colour, the name, the font, the size, the flags, and a second
// swatch when the style carries a box or an outline.
func StyleRows(styles []model.Style, colour bool) []string {
	rows := make([]string, 0, len(styles))
	for _, style := range styles {
		rows = append(rows, styleRow(style, colour))
	}
	return rows
}

// styleRow renders one style preview row.
func styleRow(style model.Style, colour bool) string {
	var b strings.Builder
	b.WriteString(Swatch(style.Primary, colour))
	if style.Name != "" {
		fmt.Fprintf(&b, " %s", style.Name)
	}
	if style.Font != "" {
		fmt.Fprintf(&b, " %s", style.Font)
	}
	if style.Size != 0 {
		fmt.Fprintf(&b, " %.0f", style.Size)
	}
	for _, flag := range styleFlags(style) {
		fmt.Fprintf(&b, " %s", flag)
	}
	switch {
	case style.Box:
		fmt.Fprintf(&b, " box %s", Swatch(style.Outline, colour))
	case style.OutlineWidth > 0:
		fmt.Fprintf(&b, " outline %s", Swatch(style.Outline, colour))
	}
	return b.String()
}

// styleFlags returns the flags of a style in a stable order.
func styleFlags(style model.Style) []string {
	var flags []string
	if style.Bold {
		flags = append(flags, "bold")
	}
	if style.Italic {
		flags = append(flags, "italic")
	}
	if style.Underline {
		flags = append(flags, "underline")
	}
	return flags
}

// TimelineRows returns the cue rows of a document. A cue row carries its
// time range and its text, and a karaoke cue gains a bar row underneath. The
// second return value counts the cues that the limit left out. A limit of
// zero or below shows every cue.
func TimelineRows(doc *model.Document, width, limit int, colour bool) ([]string, int) {
	var rows []string
	dropped := 0
	for i, cue := range doc.Cues {
		if limit > 0 && i >= limit {
			dropped++
			continue
		}
		rows = append(rows, fmt.Sprintf("%s  %s", timeRange(cue), cue.Text()))
		if cue.Karaoke() {
			rows = append(rows, "  "+barLine(cue, width, colour))
		}
	}
	return rows, dropped
}

// timeRange renders the time span of a cue.
func timeRange(cue model.Cue) string {
	return Clock(cue.Start) + "-" + Clock(cue.End)
}

// KaraokeBar renders the karaoke windows of a cue as a bar of width cells. A
// covered cell is a hash and the rest is a dot.
func KaraokeBar(cue model.Cue, width int) string {
	cells := barCells(cue, width)
	out := make([]byte, len(cells))
	for i, cell := range cells {
		out[i] = cell.ch
	}
	return string(out)
}

// barLine renders the karaoke bar of a cue. A covered cell takes the sung
// colour and the rest take the unsung colour.
func barLine(cue model.Cue, width int, colour bool) string {
	cells := barCells(cue, width)
	sung, unsung := barColours(cue)
	var b strings.Builder
	current, painted := false, false
	for _, cell := range cells {
		if colour && (!painted || cell.sung != current) {
			tint := unsung
			if cell.sung {
				tint = sung
			}
			fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm", tint.R, tint.G, tint.B)
			current, painted = cell.sung, true
		}
		b.WriteByte(cell.ch)
	}
	if colour {
		b.WriteString(reset)
	}
	return b.String()
}

// barCell is one cell of a karaoke bar.
type barCell struct {
	ch   byte
	sung bool
}

// barCells maps the karaoke windows of a cue onto a bar of width cells.
func barCells(cue model.Cue, width int) []barCell {
	cells := make([]barCell, width)
	for i := range cells {
		cells[i].ch = '.'
	}
	total := cue.Duration()
	if total <= 0 || width <= 0 {
		return cells
	}
	for _, span := range cue.Spans {
		if span.Start == 0 && span.End == 0 {
			continue
		}
		from := cellIndex(span.Start, total, width)
		to := cellIndex(span.End, total, width)
		if to <= from {
			to = from + 1
		}
		for i := from; i < to && i < width; i++ {
			cells[i].ch = '#'
			cells[i].sung = true
		}
	}
	return cells
}

// cellIndex maps an offset onto a cell position.
func cellIndex(d, total time.Duration, width int) int {
	if d <= 0 {
		return 0
	}
	index := int(float64(d) / float64(total) * float64(width))
	if index > width {
		return width
	}
	return index
}

// barColours returns the sung and the unsung colour of a cue. A span
// override wins over the default white and grey.
func barColours(cue model.Cue) (model.Colour, model.Colour) {
	sung := model.NewColour(255, 255, 255, 255)
	unsung := model.NewColour(120, 120, 120, 255)
	for _, span := range cue.Spans {
		if span.Fore != nil {
			sung = *span.Fore
		}
		if span.Secondary != nil {
			unsung = *span.Secondary
		}
	}
	return sung, unsung
}
