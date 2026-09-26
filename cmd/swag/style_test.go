// SPDX-License-Identifier: Apache-2.0

package main

import (
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/pterm/pterm"
)

// TestStyleHelpMarksHeadingsAndNames proves that the theme marks a section
// heading and a name column, and that it leaves a help text plain.
func TestStyleHelpMarksHeadingsAndNames(t *testing.T) {
	pterm.EnableColor()
	t.Cleanup(pterm.DisableColor)

	page := "Usage: swag convert [flags]\n\nFlags:\n  -i, --input=STRING  Input file.\n\nCommands:\n  convert  Convert a file.\n"
	got := styleHelp(page)

	if !strings.Contains(got, helpHeadingStyle.Sprint("Flags:")) {
		t.Errorf("the heading must carry the heading style:\n%s", got)
	}
	if !strings.Contains(got, helpNameStyle.Sprint("-i, --input=STRING")) {
		t.Errorf("the flag name must carry the name style:\n%s", got)
	}
	if !strings.Contains(got, helpNameStyle.Sprint("convert")) {
		t.Errorf("the command name must carry the name style:\n%s", got)
	}
	if !strings.Contains(got, "Input file.") {
		t.Errorf("the help text must stay plain:\n%s", got)
	}
}

// TestStyleHelpLeavesProseAlone covers a page with no list, so the theme
// changes nothing.
func TestStyleHelpLeavesProseAlone(t *testing.T) {
	pterm.DisableColor()
	page := "Usage: swag\n\nRun swag <command> --help for the flags of one command.\n"
	if got := styleHelp(page); got != page {
		t.Errorf("an unmarked page must stay the same:\n%s", got)
	}
}

// TestStyledHelpPrinterRendersThePage covers the printer through a real
// kong parser, so the wiring of the option is proven.
func TestStyledHelpPrinterRendersThePage(t *testing.T) {
	var cli struct {
		Input string `help:"Input subtitle file." short:"i"`
	}
	var out strings.Builder
	parser := kong.Must(&cli, kong.Name("swag"), kong.Help(styledHelpPrinter), kong.Writers(&out, &out))
	ctx, err := parser.Parse(nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := ctx.PrintUsage(false); err != nil {
		t.Fatalf("print usage: %v", err)
	}
	page := out.String()
	if !strings.Contains(page, "Flags:") {
		t.Errorf("the help page lacks its flag section:\n%s", page)
	}
	if !strings.Contains(page, "--input") {
		t.Errorf("the help page lacks a flag name:\n%s", page)
	}
}
