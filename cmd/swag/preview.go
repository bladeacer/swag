// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"

	"github.com/pterm/pterm"

	"github.com/bladeacer/swag/internal/i18n"
	"github.com/bladeacer/swag/internal/model"
	"github.com/bladeacer/swag/internal/tui"
	"github.com/bladeacer/swag/pkg/sub"
)

// timelineWidth is the width of a karaoke bar in terminal cells.
const timelineWidth = 40

// PreviewCmd paints the colours, the styles, and the karaoke timeline of a
// document in the terminal. The colours are a best effort: a terminal that
// carries no colour shows the hex value of each colour.
type PreviewCmd struct {
	Input string `help:"Input subtitle file." short:"i" required:""`
	From  string `help:"Input format name. The content decides when it is empty." short:"F" aliases:"input-format"`
	Limit int    `help:"The most cue rows to show. Zero shows every cue." short:"m" default:"40"`
}

// Run reads the input and paints one frame with the preview.
func (c *PreviewCmd) Run(ictx *runContext) error {
	t := ictx.T
	if err := checkInput(c.Input, t); err != nil {
		return err
	}
	source, err := openInput(c.Input)
	if err != nil {
		return fmt.Errorf("%s", t.F(i18n.MsgInputUnreadable, err))
	}
	defer func() { _ = source.Close() }()
	doc, err := sub.Parse(c.Input, source, c.From)
	if err != nil {
		return err
	}

	renderer := tui.NewRenderer(terminalOut)
	_, err = renderer.Draw(tui.Layout{
		Heading: t.S(i18n.MsgPreviewTitle),
		Rows:    previewRows(t, doc, c.Limit, terminalColour()),
		Status:  t.F(i18n.MsgPreviewStatus, len(doc.Styles), len(doc.Cues)),
	})
	return err
}

// terminalColour reports whether the terminal carries colour, so the
// preview stays readable on a plain pipe.
func terminalColour() bool {
	return pterm.PrintColor && !pterm.RawOutput
}

// previewRows builds the preview block: the style rows, then the timeline
// rows, with a summary when the limit leaves cues out.
func previewRows(t *i18n.T, doc *model.Document, limit int, colour bool) []string {
	rows := []string{t.S(i18n.MsgPreviewStyles)}
	if len(doc.Styles) == 0 {
		rows = append(rows, t.S(i18n.MsgPreviewNoStyles))
	} else {
		rows = append(rows, tui.StyleRows(doc.Styles, colour)...)
	}

	rows = append(rows, t.S(i18n.MsgPreviewTimeline))
	if len(doc.Cues) == 0 {
		return append(rows, t.S(i18n.MsgPreviewNoCues))
	}
	timeline, dropped := tui.TimelineRows(doc, timelineWidth, limit, colour)
	rows = append(rows, timeline...)
	if dropped > 0 {
		rows = append(rows, t.F(i18n.MsgPreviewMore, dropped))
	}
	return rows
}
