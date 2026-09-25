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
	"strings"

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

// convertCommand names the default command, which carries the conversion
// flags.
const convertCommand = "convert"

// CLI is the kong grammar of the command. Every flag carries a short form,
// so `-i in.srt` and `--input in.srt` both work.
type CLI struct {
	Version kong.VersionFlag `help:"Print the version." short:"V"`
	Locale  string           `help:"Message locale, en-GB or en-US." short:"l" default:"en-GB" env:"SWAG_LOCALE"`
	Verbose bool             `help:"Print the conversion report." short:"v"`

	// The convert command is the default one, and "withargs" lets its flags
	// stand at the top level, so `swag -i in.srt -o out.sbv` works without
	// the command word.
	Convert     ConvertCmd     `cmd:"" help:"Convert a subtitle file to another format." default:"withargs"`
	Interactive InteractiveCmd `cmd:"" help:"Ask for the input, the target, and the output, then show the result."`
}

// ConvertCmd carries the conversion flags.
type ConvertCmd struct {
	Input  string `help:"Input subtitle file." short:"i" required:""`
	From   string `help:"Input format name, for example srt. The content decides when it is empty." short:"F" aliases:"input-format"`
	Output string `help:"Output file. The extension picks the target format. Omit to write to stdout." short:"o"`
	Format string `help:"Target format name, for example srt or sbv. Overrides the output extension." short:"f" aliases:"to"`
	Font   string `help:"Replace the font of every style and span." short:"n"`
	Strict bool   `help:"Fail when the target format drops a feature." short:"s"`
}

// Run executes the conversion command.
func (c *ConvertCmd) Run(ictx *runContext) error {
	verbose := ictx.CLI.Verbose

	if err := checkInput(c.Input, ictx.T); err != nil {
		return err
	}
	target, err := resolveTarget(c, ictx.T)
	if err != nil {
		return err
	}
	pterm.Info.Println(ictx.T.F(i18n.MsgConvertStart, c.Input, target))

	source, err := openInput(c.Input)
	if err != nil {
		return fmt.Errorf("%s", ictx.T.F(i18n.MsgInputUnreadable, err))
	}
	defer source.Close()

	sink, closer, err := openOutput(c.Output, ictx.T)
	if err != nil {
		return err
	}
	defer closer()

	opts := sub.Options{Target: target, Format: c.From, Font: c.Font}
	if c.Strict {
		opts.Loss = sub.LossStrict
	}
	losses, err := sub.ConvertWith(c.Input, source, opts, sink)
	if err != nil {
		return err
	}

	pterm.Success.Println(ictx.T.F(i18n.MsgConvertSuccess, outputLabel(c.Output, ictx.T)))
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
		return fmt.Errorf("%s", t.F(i18n.MsgInputIsDirectory, path))
	}
	return nil
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
	return "", fmt.Errorf("%s", t.S(i18n.MsgTargetMissing))
}

// openInput opens the input file for reading. It is a variable so a test
// can force the open failure branch.
var openInput = func(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

// openOutput returns the writer for the converted file. A missing -o
// writes to stdout, and the closer does nothing in that case.
func openOutput(output string, t *i18n.T) (io.Writer, func(), error) {
	if output == "" {
		return os.Stdout, func() {}, nil
	}
	f, err := os.Create(output)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", t.F(i18n.MsgOutputCreate, output), err)
	}
	return f, func() { _ = f.Close() }, nil
}

// outputLabel names the destination in the success message.
func outputLabel(output string, t *i18n.T) string {
	if output == "" {
		return t.S(i18n.MsgOutputStdout)
	}
	return output
}

// banner prints the startup header. It carries no background colour, so the
// terminal theme decides how the title looks. The title stands on its own
// line, and the two hints below it carry the same prefix as every other
// message of the tool.
func banner(t *i18n.T) {
	pterm.Println()
	pterm.Println(pterm.LightCyan(pterm.Bold.Sprint(t.S(i18n.MsgBannerTitle))))
	pterm.Info.Println(t.S(i18n.MsgBannerTagline))
}

// versionLine returns the version line of the build. It names the licence,
// so each English locale spells that word its own way.
func versionLine(t *i18n.T) string {
	return fmt.Sprintf("%s (commit %s, built %s), %s", version, commit, date, t.S(i18n.MsgVersionLicence))
}

// localeList names the shipped locales in a stable order.
func localeList() string {
	locales := i18n.Supported()
	names := make([]string, 0, len(locales))
	for _, locale := range locales {
		names = append(names, string(locale))
	}
	return strings.Join(names, ", ")
}

// checkLocale fails when the locale is not shipped, so a typo never falls
// back to the default locale in silence.
func checkLocale(tag string, t *i18n.T) error {
	if _, ok := i18n.Resolve(tag); ok {
		return nil
	}
	return fmt.Errorf("%s", t.F(i18n.MsgLocaleUnknown, tag, localeList()))
}

// hasWord reports whether args carries one of the words before the
// separator that ends the flag list.
func hasWord(args []string, words ...string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		for _, word := range words {
			if arg == word {
				return true
			}
		}
	}
	return false
}

// isHelp reports whether the run asks for the help page.
func isHelp(args []string) bool { return hasWord(args, "-h", "--help", "help") }

// isVersion reports whether the run asks for the version line.
func isVersion(args []string) bool { return hasWord(args, "-V", "--version") }

// helpArgs removes the help words from args, so the trace can see the
// command that the run names. The separator ends the scan, because the
// words after it are values.
func helpArgs(args []string) []string {
	rest := make([]string, 0, len(args))
	for i, arg := range args {
		if arg == "--" {
			return append(rest, args[i:]...)
		}
		switch arg {
		case "-h", "--help", "help":
			continue
		}
		rest = append(rest, arg)
	}
	return rest
}

// printHelp prints the help page. It traces the remaining arguments, so a
// run that names a command shows the page of that command, and an empty one
// shows the page of the default command, which lists every flag. A bad
// argument reports an error instead of gaining a page of its own.
func printHelp(parser *kong.Kong, args []string) error {
	if len(args) == 0 {
		args = []string{convertCommand}
	}
	// Trace reports a bad argument in the context, and its own error return
	// stays nil for every argument list.
	ctx, _ := kong.Trace(parser, args)
	if ctx.Error != nil {
		return ctx.Error
	}
	return ctx.PrintUsage(false)
}

// bareRun reports whether the run carries no work. `air` runs the built
// binary with no arguments, and `make run` passes a bare separator, so both
// open the banner and the first step instead of failing on the missing
// input flag.
func bareRun(args []string) bool {
	return len(args) == 0 || (len(args) == 1 && args[0] == "--")
}

// run parses the arguments, runs the selected command, and returns the
// process exit code.
func run(args []string) int {
	// The parser needs its description before the flags are parsed, so the
	// description comes from the environment locale.
	envCopy := i18n.New(os.Getenv("SWAG_LOCALE"))
	if bareRun(args) {
		banner(envCopy)
		pterm.Info.Println(envCopy.S(i18n.MsgUsageBare))
		return 0
	}

	cli := CLI{}
	parser := kong.Must(&cli,
		kong.Name("swag"),
		kong.Description(envCopy.S(i18n.MsgCliDescription)),
		kong.Vars{"version": versionLine(envCopy)},
	)
	if isHelp(args) {
		if err := printHelp(parser, helpArgs(args)); err != nil {
			pterm.Error.Printf(envCopy.S(i18n.MsgUsageFailed), err)
			return 1
		}
		return 0
	}
	if isVersion(args) {
		pterm.Println(versionLine(envCopy))
		return 0
	}

	kongCtx, err := parser.Parse(args)
	ictx := &runContext{CLI: &cli, T: i18n.New(cli.Locale)}
	if err != nil {
		pterm.Error.Printf(ictx.T.S(i18n.MsgUsageFailed), err)
		return 1
	}
	if err := checkLocale(cli.Locale, ictx.T); err != nil {
		pterm.Error.Println(err)
		return 1
	}
	if err := kongCtx.Run(ictx); err != nil {
		pterm.Error.Printf(ictx.T.S(i18n.MsgConvertFailed), err)
		return 1
	}
	return 0
}

// exit ends the process. It is a variable so a test can observe the exit
// code that main would return without ending the test process.
var exit = os.Exit

func main() {
	exit(run(os.Args[1:]))
}
