// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"io"
	"regexp"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/pterm/pterm"
)

// The theme keeps the terminal in charge. It uses one accent colour for a
// heading and one for a name, and it draws no box and no background, so a
// reader who turns colour off loses no meaning.
var (
	// helpHeadingStyle marks a section heading, for example "Flags:".
	helpHeadingStyle = pterm.NewStyle(pterm.FgLightCyan, pterm.Bold)
	// helpNameStyle marks a command name or a flag name.
	helpNameStyle = pterm.NewStyle(pterm.FgLightGreen)
)

// helpHeading matches a section heading of the help page.
var helpHeading = regexp.MustCompile(`^([A-Z][^:]*):$`)

// helpEntry matches an indented row with a name column and a help column.
// The layout comes from kong, which separates the two columns with two or
// more spaces.
var helpEntry = regexp.MustCompile(`^(  )(\S.*?)(  +)(\S.*)$`)

// styledHelpPrinter renders the kong help page through the pterm theme, so
// the help page and the command output share one look. It renders the
// default page into a buffer and restyles the headings and the names, which
// keeps the kong layout.
func styledHelpPrinter(options kong.HelpOptions, ctx *kong.Context) error {
	var buf bytes.Buffer
	saved := ctx.Stdout
	ctx.Stdout = &buf
	err := kong.DefaultHelpPrinter(options, ctx)
	ctx.Stdout = saved
	if err != nil {
		return err
	}
	_, err = io.WriteString(saved, styleHelp(buf.String()))
	return err
}

// styleHelp applies the theme to a rendered help page. A heading gains the
// heading style, and a row of a list gains the name style on its name
// column. The help text of a row stays plain.
func styleHelp(page string) string {
	lines := strings.Split(page, "\n")
	inList := false
	for i, line := range lines {
		switch {
		case helpHeading.MatchString(line):
			lines[i] = helpHeadingStyle.Sprint(line)
			inList = true
		case line == "":
			inList = false
		case inList:
			if m := helpEntry.FindStringSubmatch(line); m != nil {
				lines[i] = m[1] + helpNameStyle.Sprint(m[2]) + m[3] + m[4]
			}
		}
	}
	return strings.Join(lines, "\n")
}
