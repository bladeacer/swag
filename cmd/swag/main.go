// Command swag converts subtitle files.
//
// The CLI wires the public library (pkg/sub) to the terminal: kong parses
// the flags, pterm styles the output, and the message catalogue in
// internal/i18n carries every user-facing string.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/pterm/pterm"

	"github.com/bladeacer/swag/internal/i18n"
	"github.com/bladeacer/swag/pkg/sub"
)

// Build information, set by -X ldflags at release time. A plain
// `go install` keeps the defaults; release builds stamp them.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// CLI is the kong grammar of the command.
type CLI struct {
	Version kong.VersionFlag `help:"Print the version."`
	Locale  string           `help:"Message locale, for example en-GB." default:"en-GB" env:"SWAG_LOCALE"`
	Verbose bool             `help:"Print the conversion report." short:"V"`

	Convert ConvertCmd `cmd:"" help:"Convert a subtitle file to another format." default:"1"`
}

// ConvertCmd carries the conversion flags.
type ConvertCmd struct {
	Input   string `help:"Input subtitle file." short:"i" required:""`
	Output  string `help:"Output file. The extension picks the target format. Omit to write to stdout." short:"o"`
	Format  string `help:"Target format name, for example srt or sbv. Overrides the output extension." short:"f"`
	Verbose bool   `help:"Print the conversion report." short:"v" name:"report"`
}

// Run executes the conversion command.
func (c *ConvertCmd) Run(ictx *runContext) error {
	verbose := c.Verbose || ictx.CLI.Verbose

	if err := checkInput(c.Input, ictx.T); err != nil {
		return err
	}
	ictx.T.F(i18n.MsgConvertStart, c.Input, targetName(c))

	source, err := os.Open(c.Input)
	if err != nil {
		return fmt.Errorf("%s", ictx.T.F(i18n.MsgInputUnreadable, err))
	}
	defer source.Close()

	doc, err := sub.Parse(c.Input, source, "")
	if err != nil {
		return err
	}

	target, err := resolveTarget(c, ictx.T)
	if err != nil {
		return err
	}
	sink, closer, err := openOutput(c.Output)
	if err != nil {
		return err
	}
	defer closer()

	losses, err := sub.Render(doc, target, sink)
	if err != nil {
		return err
	}

	pterm.Success.Printf(ictx.T.S(i18n.MsgConvertSuccess), outputLabel(c.Output))
	if verbose && len(losses) > 0 {
		pterm.Warning.Println(ictx.T.F(i18n.MsgConvertLosses, len(losses)))
		for _, loss := range losses {
			pterm.Warning.Printf("  %s\n", loss)
		}
	}
	return nil
}

// runContext carries the parsed CLI and the message catalogue into
// command Run methods.
type runContext struct {
	CLI *CLI
	T   *i18n.T
}

// checkInput fails early when the input file is missing or a directory.
func checkInput(path string, t *i18n.T) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s", t.S(i18n.MsgInputMissing))
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory; give the path of a subtitle file", path)
	}
	return nil
}

// targetName reports the target format for the start message.
func targetName(c *ConvertCmd) string {
	if c.Format != "" {
		return c.Format
	}
	if c.Output != "" {
		if name := sub.DetectFormat(c.Output); name != "" {
			return name
		}
	}
	return "?"
}

// resolveTarget determines the target format from -f or the output
// extension.
func resolveTarget(c *ConvertCmd, t *i18n.T) (string, error) {
	if c.Format != "" {
		return c.Format, nil
	}
	if c.Output != "" {
		if name := sub.DetectFormat(c.Output); name != "" {
			return name, nil
		}
		return "", fmt.Errorf("%s", t.F(i18n.MsgFormatUnknown, c.Output))
	}
	return "", fmt.Errorf("%s", t.S(i18n.MsgFormatUnsupported))
}

// openOutput returns the writer for the converted file. A missing -o
// writes to stdout, and the closer does nothing in that case.
func openOutput(output string) (io.Writer, func(), error) {
	if output == "" {
		return os.Stdout, func() {}, nil
	}
	f, err := os.Create(output)
	if err != nil {
		return nil, nil, fmt.Errorf("create %s: %w", output, err)
	}
	return f, func() { _ = f.Close() }, nil
}

// outputLabel names the destination in the success message.
func outputLabel(output string) string {
	if output == "" {
		return "standard output"
	}
	return output
}

// banner prints the startup header.
func banner(t *i18n.T) {
	pterm.DefaultHeader.WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(pterm.BgBlack, pterm.FgLightWhite)).
		Println(t.S(i18n.MsgBannerTitle))
}

func main() {
	cli := CLI{}
	parser := kong.Must(&cli,
		kong.Name("swag"),
		kong.Description("Subtitles With A Gopher: read, write, and convert subtitles."),
		kong.Vars{
			"version": fmt.Sprintf("%s (commit %s, built %s)", version, commit, date),
		},
	)
	kongCtx, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)

	ictx := &runContext{CLI: &cli, T: i18n.New(cli.Locale)}
	if err := kongCtx.Run(ictx); err != nil {
		pterm.Error.Printf(ictx.T.S(i18n.MsgConvertFailed), err)
		os.Exit(1)
	}
}
