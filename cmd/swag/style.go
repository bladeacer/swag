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
// heading and the program title, one for a name, and one for a question, and
// it draws no box and no background, so a reader who turns colour off loses
// no meaning. Every command reads the same theme, so the banner, the
// prompts, and the help page look like one program.
var (
	// headingStyle marks a section heading, the program title, and the
	// usage line.
	headingStyle = pterm.NewStyle(pterm.FgLightCyan, pterm.Bold)
	// nameStyle marks a command name or a flag name.
	nameStyle = pterm.NewStyle(pterm.FgLightGreen)
	// questionStyle marks a question of the interactive mode.
	questionStyle = pterm.NewStyle(pterm.FgLightCyan)
)

// helpHeading matches a section heading of the help page.
var helpHeading = regexp.MustCompile(`^([A-Z][^:]*):$`)

// helpUsage matches the usage line of the help page.
var helpUsage = regexp.MustCompile(`^(Usage:)(.*)$`)

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
	// The default printer writes to a buffer here, so its error cannot
	// happen. The write to the real sink is the one that can fail.
	_ = kong.DefaultHelpPrinter(options, ctx)
	ctx.Stdout = saved
	_, err := io.WriteString(saved, styleHelp(buf.String()))
	return err
}

// styleHelp applies the theme to a rendered help page. A heading and the
// usage line gain the heading style, and a row of a list gains the name
// style on its name column. The help text of a row stays plain.
func styleHelp(page string) string {
	lines := strings.Split(page, "\n")
	inList := false
	for i, line := range lines {
		switch {
		case helpHeading.MatchString(line):
			lines[i] = headingStyle.Sprint(line)
			inList = true
		case helpUsage.MatchString(line):
			m := helpUsage.FindStringSubmatch(line)
			lines[i] = headingStyle.Sprint(m[1]) + m[2]
		case line == "":
			inList = false
		case inList:
			if m := helpEntry.FindStringSubmatch(line); m != nil {
				lines[i] = m[1] + nameStyle.Sprint(m[2]) + m[3] + m[4]
			}
		}
	}
	return strings.Join(lines, "\n")
}
