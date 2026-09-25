// SPDX-License-Identifier: Apache-2.0

package ytt

import (
	"fmt"
	"math"
	"strings"

	"github.com/bladeacer/swag/internal/model"
)

// penTable deduplicates pens and gives each one an id. YouTube numbers pens
// from one, and the head must list them in increasing id order.
type penTable struct {
	ids  map[string]int
	list []pen
}

func newPenTable() *penTable { return &penTable{ids: map[string]int{}} }

// id returns the id of p, adding it to the table on first use.
func (t *penTable) id(p pen) int {
	k := p.key()
	if id, ok := t.ids[k]; ok {
		return id
	}
	id := len(t.list) + 1
	t.ids[k] = id
	t.list = append(t.list, p)
	return id
}

// key is a canonical string of the pen attributes, used for deduplication.
func (p pen) key() string {
	var b strings.Builder
	fmt.Fprintf(&b, "b=%t;i=%t;u=%t;", p.bold, p.italic, p.underline)
	writeColourKey(&b, "fc", p.fore)
	writeColourKey(&b, "bc", p.back)
	writeColourKey(&b, "ec", p.edge)
	fmt.Fprintf(&b, "et=%d;fs=%d;hasfs=%t;sz=%d;hassz=%t;rb=%d;of=%d;hg=%t",
		p.edgeKind, p.fontStyle, p.hasFont, p.fontScale, p.hasScale, p.ruby, p.script, p.packed)
	return b.String()
}

func writeColourKey(b *strings.Builder, name string, c *model.Colour) {
	if c == nil {
		fmt.Fprintf(b, "%s=none;", name)
		return
	}
	fmt.Fprintf(b, "%s=%02X%02X%02X%02X;", name, c.R, c.G, c.B, c.A)
}

// windowKey is the deduplication key of a <ws> element.
type windowKey struct {
	ju int
	pd int
	sd int
}

// defaultWindow is the window YouTube uses when a line carries no <ws>.
var defaultWindow = windowKey{ju: 2}

// windowTable deduplicates window styles.
type windowTable struct {
	ids  map[windowKey]int
	list []windowKey
}

func newWindowTable() *windowTable { return &windowTable{ids: map[windowKey]int{}} }

// id returns the id of k, or -1 when k is the default window, which needs
// no element.
func (t *windowTable) id(k windowKey) int {
	if k == defaultWindow {
		return -1
	}
	if id, ok := t.ids[k]; ok {
		return id
	}
	id := len(t.list) + 1
	t.ids[k] = id
	t.list = append(t.list, k)
	return id
}

// windowFor derives the window style of a line from its spans and its
// anchor. The horizontal justification follows the anchor column; the pitch
// and skew follow vertical text or a right-to-left direction.
func windowFor(spans []model.TextSpan, anchor *model.Anchor) windowKey {
	k := defaultWindow
	if anchor != nil && *anchor >= model.AnchorBottomLeft && *anchor <= model.AnchorTopRight {
		switch (int(*anchor) - 1) % 3 {
		case 0:
			k.ju = 0
		case 2:
			k.ju = 1
		}
	}
	for _, s := range spans {
		if s.Vertical == nil {
			continue
		}
		switch s.Vertical.Mode {
		case model.VerticalColumnsRTL:
			k.pd, k.sd = 2, 0
		case model.VerticalColumnsLTR:
			k.pd, k.sd = 2, 1
		case model.VerticalRotated:
			k.pd, k.sd = 3, 0
		case model.VerticalRotatedReversed:
			k.pd, k.sd = 3, 1
		}
		if k.pd != 0 {
			break
		}
	}
	if k.pd == 0 {
		for _, s := range spans {
			if s.Direction != nil && *s.Direction == model.DirRightToLeft {
				k.pd, k.sd = 1, 0
				break
			}
		}
	}
	return k
}

// writeWindows writes the window style elements in id order.
func writeWindows(out *strings.Builder, windows *windowTable) {
	for i, k := range windows.list {
		fmt.Fprintf(out, "<ws id=\"%d\" ju=\"%d\" wfo=\"0\"", i+1, k.ju)
		if k.pd != 0 {
			fmt.Fprintf(out, " pd=\"%d\" sd=\"%d\"", k.pd, k.sd)
		}
		out.WriteString(" />\n")
	}
}

// positionKey is the deduplication key of a <wp> element. The offsets stay
// as whole percentages because the upload server rejects decimals.
type positionKey struct {
	ap int
	ah int
	av int
}

// positionTable deduplicates window positions.
type positionTable struct {
	ids  map[positionKey]int
	list []positionKey
	dims model.Point
}

func newPositionTable(dims model.Point) *positionTable {
	return &positionTable{ids: map[positionKey]int{}, dims: dims}
}

// id returns the id of the position of layout, or -1 when the cue carries
// no layout.
func (t *positionTable) id(layout *model.Layout) int {
	if layout == nil {
		return -1
	}
	anchor := model.AnchorBottomCentre
	if layout.Anchor != nil {
		anchor = *layout.Anchor
	}
	var ah, av int
	if layout.Position != nil && t.dims.X > 0 && t.dims.Y > 0 {
		ah = int(math.Round(layout.Position.X / t.dims.X * 100))
		av = int(math.Round(layout.Position.Y / t.dims.Y * 100))
	} else {
		ah, av = defaultPercent(anchor)
	}
	k := positionKey{ap: apFromAnchor(anchor), ah: ah, av: av}
	if id, ok := t.ids[k]; ok {
		return id
	}
	id := len(t.list) + 1
	t.ids[k] = id
	t.list = append(t.list, k)
	return id
}

// defaultPercent returns the standard percentage coordinates of an anchor.
func defaultPercent(anchor model.Anchor) (int, int) {
	if anchor < model.AnchorBottomLeft || anchor > model.AnchorTopRight {
		anchor = model.AnchorBottomCentre
	}
	v := int(anchor) - 1
	col, row := v%3, v/3
	ah := []int{0, 50, 100}[col]
	av := []int{90, 50, 10}[row]
	return ah, av
}

// writePositions writes the window position elements in id order.
func writePositions(out *strings.Builder, positions *positionTable) {
	for i, k := range positions.list {
		fmt.Fprintf(out, "<wp id=\"%d\" ap=\"%d\" ah=\"%d\" av=\"%d\" />\n", i+1, k.ap, k.ah, k.av)
	}
}
