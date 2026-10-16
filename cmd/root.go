// Package cmd is the command line interface of lutf.
package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/code-instable/latex2utf8/convert"
	"github.com/code-instable/latex2utf8/symbols"
)

// Version is set at build time with -ldflags "-X github.com/code-instable/latex2utf8/cmd.Version=...".
var Version = "dev"

// IO bundles the streams of a run, so that tests can replace them.
type IO struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
	// InTTY tells whether stdin is a terminal, OutTTY whether stdout is (which only affects how --list is laid out).
	InTTY, OutTTY bool
}

// StdIO returns the process's standard streams.
func StdIO() IO {
	return IO{In: os.Stdin, Out: os.Stdout, Err: os.Stderr, InTTY: isTerminal(os.Stdin), OutTTY: isTerminal(os.Stdout)}
}

// isTerminal reports whether f is an interactive terminal. /dev/null is a
// character device too, but is not one.
func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	if err != nil || st.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	if null, err := os.Stat(os.DevNull); err == nil && os.SameFile(st, null) {
		return false
	}
	return true
}

// Exit codes.
const (
	exitOK       = 0
	exitFailure  = 1 // an error, or an unknown name under --strict, or a cancelled search
	exitUsage    = 2 // bad flags
	exitInterupt = 130
)

// exitError makes Run exit with a code; the message, if any, has been printed already.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return fmt.Sprintf("exit status %d", e.code)
}

type usageError struct{ error }

// Run executes lutf with the given arguments (without the program name) and returns the exit code.
func Run(args []string, streams IO) int {
	root := NewRootCmd(streams)
	root.SetArgs(args)
	err := root.Execute()
	var ee *exitError
	var ue usageError
	switch {
	case err == nil:
		return exitOK
	case errors.As(err, &ee):
		if ee.err != nil {
			_, _ = fmt.Fprintf(streams.Err, "lutf: %v\n", ee.err)
		}
		return ee.code
	case errors.As(err, &ue):
		_, _ = fmt.Fprintf(streams.Err, "lutf: %v\nTry 'lutf --help' for more information.\n", ue.error)
		return exitUsage
	}
	_, _ = fmt.Fprintf(streams.Err, "lutf: %v\n", err)
	return exitFailure
}

// flags are the command line options.
type flags struct {
	search, list, reverse, info          bool
	text, strict, noFuzzy, join          bool
	latex, quiet, copy, newline, jsonOut bool
	completion                           string
}

type runner struct {
	io    IO
	flags flags
	set   *symbols.Set
}

// NewRootCmd builds the lutf command writing to the given streams.
func NewRootCmd(streams IO) *cobra.Command {
	r := &runner{io: streams, set: symbols.Default()}
	cmd := &cobra.Command{
		Use:                   "lutf [flags] [text...]",
		Short:                 "Convert LaTeX names and syntax to Unicode symbols",
		Long:                  longHelp,
		Version:               Version,
		Args:                  cobra.ArbitraryArgs,
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
		CompletionOptions:     cobra.CompletionOptions{DisableDefaultCmd: true},
		ValidArgsFunction:     r.completeNames,
		RunE:                  r.run,
	}
	cmd.SetIn(streams.In)
	cmd.SetOut(streams.Out)
	cmd.SetErr(streams.Err)
	cmd.SetVersionTemplate("lutf {{.Version}}\n")
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageError{err} })

	f := cmd.Flags()
	f.BoolVarP(&r.flags.search, "search", "s", false, "search the symbols interactively with fzf and print the one chosen")
	f.BoolVarP(&r.flags.list, "list", "l", false, "list the symbols; the arguments filter the names")
	f.BoolVarP(&r.flags.reverse, "reverse", "r", false, "turn symbols back into names")
	f.BoolVarP(&r.flags.info, "info", "i", false, "show the code points and names of each character")
	f.BoolVarP(&r.flags.text, "text", "t", false, "convert only \\names, \\commands{...} and braced scripts; leave words alone")
	f.BoolVar(&r.flags.strict, "strict", false, "exact names only; list unknown names and exit with status 1")
	f.BoolVar(&r.flags.noFuzzy, "no-fuzzy", false, "do not correct typos")
	f.BoolVarP(&r.flags.join, "join", "j", false, "remove all whitespace from the output")
	f.BoolVar(&r.flags.latex, "latex", false, "with --reverse: write \\name instead of name")
	f.BoolVarP(&r.flags.quiet, "quiet", "q", false, "do not warn about unknown names")
	f.BoolVarP(&r.flags.copy, "copy", "c", false, "also copy the result (of a conversion, --reverse or --search, not the JSON) to the clipboard")
	f.BoolVarP(&r.flags.newline, "newline", "n", false, "end the converted text with a newline (none is added otherwise; --json, --list and --info print whole lines)")
	f.BoolVar(&r.flags.jsonOut, "json", false, "write the result as JSON, on one line ending with a newline")
	f.StringVar(&r.flags.completion, "completion", "", "print a shell completion script (bash, zsh, fish or powershell)")
	// `lutf --completion <TAB>` offers the shells
	if err := cmd.RegisterFlagCompletionFunc("completion", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return shells, cobra.ShellCompDirectiveNoFileComp
	}); err != nil {
		panic(err) // the flag is defined just above
	}
	return cmd
}

// shells are the values of --completion.
var shells = []string{"bash", "zsh", "fish", "powershell"}

// check rejects combinations of flags that make no sense together.
func (f flags) check() error {
	modes := 0
	for _, on := range []bool{f.search, f.list, f.reverse, f.info} {
		if on {
			modes++
		}
	}
	if modes > 1 {
		return usageError{errors.New("--search, --list, --reverse and --info cannot be combined")}
	}
	if f.text && f.strict {
		return usageError{errors.New("--text and --strict cannot be combined")}
	}
	return nil
}

func (r *runner) run(cmd *cobra.Command, args []string) error {
	f := r.flags
	if err := f.check(); err != nil {
		return err
	}
	switch {
	case f.completion != "":
		return r.printCompletion(cmd, f.completion)
	case f.search:
		return r.runSearch()
	case f.list:
		return r.runList(args)
	}
	input, ok, err := r.readInput(args)
	if err != nil {
		return err
	}
	if !ok { // nothing to convert and nobody typing: show how to use the tool
		return cmd.Help()
	}
	switch {
	case f.reverse:
		return r.runReverse(input)
	case f.info:
		return r.runInfo(input)
	}
	return r.runConvert(input)
}

// readInput returns the text to work on: the arguments, or else stdin. ok is
// false when there are no arguments and stdin is a terminal.
func (r *runner) readInput(args []string) (input string, ok bool, err error) {
	if len(args) > 0 {
		return strings.Join(args, " "), true, nil
	}
	if r.io.InTTY {
		return "", false, nil
	}
	b, err := io.ReadAll(r.io.In)
	if err != nil {
		return "", false, fmt.Errorf("reading stdin: %w", err)
	}
	return trimFinalNewline(string(b)), true, nil
}

// trimFinalNewline drops the newline that ends a single line of input (as
// produced by echo or an editor's selection) so that the result pastes cleanly.
// The line structure of longer input is left alone.
func trimFinalNewline(s string) string {
	if strings.Count(s, "\n") == 1 {
		if t, ok := strings.CutSuffix(s, "\r\n"); ok {
			return t
		}
		if t, ok := strings.CutSuffix(s, "\n"); ok {
			return t
		}
	}
	return s
}

// write prints s. It adds no newline unless asked to (-n), not even on a
// terminal: in an editor the output replaces the selection, and what is
// printed must be exactly the wanted characters (zsh shows a % to say so).
func (r *runner) write(s string) error {
	if r.flags.newline && !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	_, err := io.WriteString(r.io.Out, s)
	return err
}

func (r *runner) runConvert(input string) error {
	f := r.flags
	opt := convert.Options{Fuzzy: !f.noFuzzy, Strict: f.strict, Join: f.join, Set: r.set}
	if f.text {
		opt.Mode = convert.Text
	}
	res := convert.Convert(input, opt)
	unknown := r.report(res.Problems)
	if f.jsonOut {
		if err := r.writeJSON(struct {
			Input   string   `json:"input"`
			Output  string   `json:"output"`
			Unknown []string `json:"unknown"`
		}{input, res.Output, unknown}); err != nil {
			return err
		}
	} else if err := r.write(res.Output); err != nil {
		return err
	}
	r.copyToClipboard(res.Output)
	if f.strict && len(unknown) > 0 {
		return &exitError{code: exitFailure}
	}
	return nil
}

// report warns about unresolved names and returns them as typed. Names
// written with a backslash were meant as commands, so they are always worth
// a warning (except in text mode, where backslashes are common); bare words
// are reported only under --strict, since they are probably just text.
func (r *runner) report(problems []convert.Problem) []string {
	unknown := []string{}
	for _, p := range problems {
		name := p.Name
		if p.Backslash {
			name = `\` + name
		}
		unknown = append(unknown, name)
		warn := r.flags.strict || (p.Backslash && !r.flags.text)
		if !warn || r.flags.quiet || r.flags.jsonOut {
			continue
		}
		msg := "unknown symbol " + name
		if s := r.set.Suggest(p.Name, 3); len(s) > 0 {
			msg += " (did you mean " + strings.Join(s, ", ") + "?)"
		}
		_, _ = fmt.Fprintf(r.io.Err, "lutf: %s\n", msg)
	}
	return unknown
}

func (r *runner) runReverse(input string) error {
	out := convert.Reverse(input, convert.ReverseOptions{Backslash: r.flags.latex, Set: r.set})
	if r.flags.jsonOut {
		if err := r.writeJSON(struct {
			Input  string `json:"input"`
			Output string `json:"output"`
		}{input, out}); err != nil {
			return err
		}
	} else if err := r.write(out); err != nil {
		return err
	}
	r.copyToClipboard(out)
	return nil
}

const longHelp = `lutf converts the names of LaTeX and Unicode symbols into the symbols
themselves. Type names, with or without a backslash, as arguments or on stdin.

Names
  gamma  \gamma  Gamma  mbfGamma  BbbR  neq  nabla    the backslash is optional
  \L  \o  \S                                         one letter names need it
  gammma  bbbR  alphabeta                            typos, wrong case and run-together
                                                     names are repaired
  alpha{}x                                           an empty {} ends a name: αx, not "α x"

Scripts, commands and accents
  x_1  x^2  x_{12}  x^{n+1}  _n  ^T                  subscripts and superscripts
  frac{1}{2}  frac{12}{35}  sqrt[3]{x}               fractions and roots
  hat{x}  bar{x}  tilde{n}  vec{v}  not{=}           accents; â, ñ, ≠ are composed
  mathbb{R}  mathcal{A}  mathfrak{g}  mathbf{x}      math alphabets
  text{...}                                          copied as it is
  \"{o}  \'e  \c{c}                                  LaTeX text accents

Spaces between two converted names are dropped, so "alpha beta" is "αβ"
while "x = alpha" keeps its spaces. Text that is not a name is left alone.

Examples
  lutf mbfGamma mitX _2        →  𝚪𝑋₂
  echo "mbfGamma mitX _2" | lutf
  lutf 'frac{1}{2} + hat{x}^2' →  ½ + x̂²
  lutf -r ∇                    →  del
  lutf -l arrow                list the symbols whose name contains "arrow"
  lutf -s                      search interactively (needs fzf)

Families of symbols
  L : Ł Ø ¶                           l : ł ø ˘
  superscript (^{1}) : ᴬ ᴮ ᴰ          subscript (_{1}) : ₍ ₎ ₊
  (mit)Bbb{A/one} : 𝔸 𝔹 ℂ             mscr : 𝒜 ℬ 𝒞
  mtt : 𝙰 𝙱 𝙲                         mit : 𝐴 𝐵 𝐶
  mitAlpha : 𝛢 𝛥 𝛤                    mitsans : 𝘈 𝘉 𝘊
  mfrak : 𝔄 𝔅 ℭ                       msans : 𝖠 𝖡 𝖢
  mbf : 𝐀 𝐁 𝐂                         mbf{Mu} : 𝚳 𝚴 𝚷
  mbfit : 𝑨 𝑩 𝑪                       mbfitsans : 𝘼 𝘽 𝘾
  mbfsans : 𝗔 𝗕 𝗖                     mbffrak : 𝕬 𝕭 𝕮
  mbfscr : 𝓐 𝓑 𝓒                      circled{one} : ⓪ ① ②
  circ{one} : ➀ ➁ ➂                   fcirc{one} : ➊ ➋ ➌
  sq : 🄰 🄱 🄲                          fsq : 🅰 🅱 🅲
  circ{A} : Ⓐ Ⓑ Ⓒ                     circ{a} : ⓐ ⓑ ⓒ
  dcirc{one} : ⓵ ⓶ ⓷                  fcirc{A} : 🅐 🅑 🅒
  box[x][y] : ┏ ┳ ━                   dbox[x][y] : ═ ═ ═
  [dir]harpoon[supp] : ↼ ↾ ⇀          [dir]harpoons : ⥢ ⥦ ⥤
  arrows : ⬾ ⇈ ⥆                      measangle{l/r/u/d}to{n/s/e/w}{n/s/e/w} : ⦯ ⦮ ⦫
  black[shape][dir] : ☻ ■ ⧫           bigblack : ▲ ▼
  {a/c/d/v}dots : ⋰ · ⋯               shade{light/dense/bold} : ░ ▒ ▓
  {r/u/l/d}tri : ▷ ◁ △                cyr<Name> : Б б Ж ж (Cyrillic)

Use --list to see every name, or --search to browse them with fzf.`
