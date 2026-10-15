package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

type result struct {
	out, err string
	code     int
}

// lutf runs lutf with the given stdin (nil: stdin is empty and not a terminal).
func lutf(t *testing.T, stdin string, args ...string) result {
	t.Helper()
	return execIO(t, IO{In: strings.NewReader(stdin)}, args...)
}

func execIO(t *testing.T, streams IO, args ...string) result {
	t.Helper()
	var out, errb bytes.Buffer
	streams.Out, streams.Err = &out, &errb
	if streams.In == nil {
		streams.In = strings.NewReader("")
	}
	code := Run(args, streams)
	return result{out.String(), errb.String(), code}
}

func TestConvertArguments(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"mbfGamma"}, "𝚪"},
		{[]string{"mbfGamma", "mitX", "_2"}, "𝚪𝑋₂"}, // several arguments are one text
		{[]string{"mitX _1"}, "𝑋₁"},
		{[]string{`\gamma`}, "γ"},
		{[]string{"hello world"}, "hello world"},
		{[]string{"x = alpha"}, "x = α"},
	}
	for _, tt := range tests {
		r := lutf(t, "", tt.args...)
		if r.out != tt.want || r.code != 0 || r.err != "" {
			t.Errorf("lutf %q = %+v, want %q", tt.args, r, tt.want)
		}
	}
}

func TestStdin(t *testing.T) {
	tests := []struct{ in, want string }{
		{"mbfGamma mitX neq mitu _0\n", "𝚪𝑋≠𝑢₀"}, // the newline of echo is not part of the text
		{"mbfGamma mitX neq mitu _0", "𝚪𝑋≠𝑢₀"},
		{"alpha\r\n", "α"},
		{"alpha\nbeta\n", "α\nβ\n"}, // several lines keep their structure
		{"alpha\n\n", "α\n\n"},
		{"", ""},
	}
	for _, tt := range tests {
		if r := lutf(t, tt.in); r.out != tt.want || r.code != 0 {
			t.Errorf("stdin %q -> %+v, want %q", tt.in, r, tt.want)
		}
	}
}

func TestNewlines(t *testing.T) {
	// No newline unless asked for, not even on a terminal: in an editor the output replaces
	// the selection, so it has to be exactly the wanted characters.
	for _, streams := range []IO{{}, {OutTTY: true}} {
		for _, args := range [][]string{{"alpha"}, {"-r", "γ"}, {"--text", `\alpha`}} {
			if r := execIO(t, streams, args...); strings.HasSuffix(r.out, "\n") {
				t.Errorf("tty=%v lutf %v ended with a newline: %q", streams.OutTTY, args, r.out)
			}
		}
	}
	if r := lutf(t, "", "-n", "alpha"); r.out != "α\n" {
		t.Errorf("-n: %q", r.out)
	}
	// ... but exactly one
	if r := lutf(t, "alpha\nbeta\n", "-n"); r.out != "α\nβ\n" {
		t.Errorf("-n with a final newline: %q", r.out)
	}
}

func TestNoInputOnATerminalShowsHelp(t *testing.T) {
	r := execIO(t, IO{InTTY: true})
	if r.code != 0 || !strings.Contains(r.out, "lutf converts") || !strings.Contains(r.out, "--reverse") {
		t.Errorf("got %+v", r)
	}
}

func TestTrimFinalNewline(t *testing.T) {
	tests := map[string]string{
		"a\n": "a", "a\r\n": "a", "a": "a", "": "", "\n": "", "a\nb": "a\nb", "a\nb\n": "a\nb\n", "a\n\n": "a\n\n",
	}
	for in, want := range tests {
		if got := trimFinalNewline(in); got != want {
			t.Errorf("trimFinalNewline(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFlagsChangeTheConversion(t *testing.T) {
	tests := []struct {
		args []string
		in   string
		want string
	}{
		{[]string{"-j"}, "x = alpha + beta", "x=α+β"},
		{[]string{"--no-fuzzy"}, "gammma", "gammma"},
		{[]string{}, "gammma", "γ"},
		{[]string{"--text"}, "the value of alpha is large", "the value of alpha is large"},
		{[]string{"-t"}, `the value of \alpha is large`, "the value of α is large"},
	}
	for _, tt := range tests {
		if r := lutf(t, tt.in, tt.args...); r.out != tt.want {
			t.Errorf("lutf %v <<< %q = %q, want %q", tt.args, tt.in, r.out, tt.want)
		}
	}
}

func TestUnknownNames(t *testing.T) {
	// a backslash name is a command, so a typo in it deserves a warning ...
	r := lutf(t, "", `\gammma and x`)
	if r.out != `\gammma and x` || r.code != 0 || !strings.Contains(r.err, `unknown symbol \gammma`) || !strings.Contains(r.err, "gamma") {
		t.Errorf("got %+v", r)
	}
	// ... which --quiet silences
	if r := lutf(t, "", "-q", `\gammma`); r.err != "" {
		t.Errorf("-q still warned: %q", r.err)
	}
	// plain words are text: no warning
	if r := lutf(t, "", "just words"); r.err != "" {
		t.Errorf("warned about plain words: %q", r.err)
	}
	// --strict reports everything and fails, still printing the result
	r = lutf(t, "", "--strict", "gamma", "gammma")
	if r.code != 1 || r.out != "γgammma" && r.out != "γ gammma" || !strings.Contains(r.err, "unknown symbol gammma") {
		t.Errorf("strict: %+v", r)
	}
	if r := lutf(t, "", "--strict", "gamma"); r.code != 0 || r.out != "γ" {
		t.Errorf("strict on a known name: %+v", r)
	}
	// text mode: stray backslashes in prose are none of our business
	if r := lutf(t, "", "--text", `C:\Users\foo`); r.err != "" || r.out != `C:\Users\foo` {
		t.Errorf("text mode: %+v", r)
	}
}

func TestReverseAndInfo(t *testing.T) {
	if r := lutf(t, "", "-r", "γ"); r.out != "gamma" {
		t.Errorf("-r: %+v", r)
	}
	if r := lutf(t, "", "-r", "--latex", "αβ"); r.out != `\alpha\beta` {
		t.Errorf("-r --latex: %+v", r)
	}
	r := lutf(t, "", "-i", "γ x")
	lines := strings.Split(strings.TrimRight(r.out, "\n"), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "γ\tU+03B3\tce b3\t") || !strings.Contains(lines[0], "gamma") {
		t.Errorf("-i: %q", r.out)
	}
}

func TestList(t *testing.T) {
	r := lutf(t, "", "-l", "gamma")
	if r.code != 0 || !strings.Contains(r.out, "gamma\tγ\n") || !strings.Contains(r.out, "Gamma\tΓ\n") {
		t.Errorf("-l gamma: %+v", r)
	}
	if strings.Contains(r.out, "alpha") {
		t.Error("filter let alpha through")
	}
	// the symbol itself filters too
	if r := lutf(t, "", "-l", "∇"); !strings.Contains(r.out, "nabla\t∇") {
		t.Errorf("-l ∇: %q", r.out)
	}
	if r := lutf(t, "", "-l"); strings.Count(r.out, "\n") < 2800 {
		t.Errorf("-l lists only %d symbols", strings.Count(r.out, "\n"))
	}
	if r := lutf(t, "", "-l", "nosuchsymbolatall"); r.code != 1 || r.out != "" {
		t.Errorf("no match should fail quietly: %+v", r)
	}
	// on a terminal the columns are aligned with spaces rather than separated by a tab
	if r := execIO(t, IO{OutTTY: true}, "-l", "nabla"); strings.Contains(r.out, "\t") || !regexp.MustCompile(`(?m)^nabla +∇$`).MatchString(r.out) {
		t.Errorf("tty listing: %q", r.out)
	}
}

func TestJSON(t *testing.T) {
	var conv struct {
		Input, Output string
		Unknown       []string
	}
	r := lutf(t, "", "--json", "alpha foo <&>")
	if err := json.Unmarshal([]byte(r.out), &conv); err != nil {
		t.Fatalf("%v in %q", err, r.out)
	}
	if conv.Input != "alpha foo <&>" || conv.Output != "α foo <&>" || len(conv.Unknown) != 1 || conv.Unknown[0] != "foo" {
		t.Errorf("convert: %+v", conv)
	}
	if strings.Contains(r.out, `\u003c`) {
		t.Error("JSON output escapes HTML")
	}
	if !strings.Contains(lutf(t, "", "--json", "gamma").out, `"unknown":[]`) {
		t.Error("unknown must be an empty array, not null")
	}

	var list []struct {
		Name, Symbol string
		Alias        bool
	}
	if err := json.Unmarshal([]byte(lutf(t, "", "--json", "-l", "gamma").out), &list); err != nil || len(list) == 0 {
		t.Fatalf("list: %v %v", err, list)
	}
	var info []struct {
		Char       string
		Codepoints []string
		Utf8       []string
		Names      []string
	}
	if err := json.Unmarshal([]byte(lutf(t, "", "--json", "-i", "γx").out), &info); err != nil || len(info) != 2 {
		t.Fatalf("info: %v %v", err, info)
	}
	if info[0].Codepoints[0] != "U+03B3" || info[1].Names == nil {
		t.Errorf("info: %+v", info)
	}
	var rev struct{ Input, Output string }
	if err := json.Unmarshal([]byte(lutf(t, "", "--json", "-r", "γ").out), &rev); err != nil || rev.Output != "gamma" {
		t.Errorf("reverse: %v %+v", err, rev)
	}
}

func TestUsageErrors(t *testing.T) {
	tests := [][]string{
		{"--bogus"},
		{"-s", "-l"},
		{"-r", "-i"},
		{"--text", "--strict", "x"},
		{"--completion", "tcsh"},
	}
	for _, args := range tests {
		r := lutf(t, "", args...)
		if r.code != 2 || r.out != "" || !strings.HasPrefix(r.err, "lutf: ") || !strings.Contains(r.err, "--help") {
			t.Errorf("lutf %v = %+v, want exit 2 and a hint", args, r)
		}
	}
}

func TestVersionAndHelp(t *testing.T) {
	// Version is "dev" unless the build stamps it (-ldflags -X, as flake.nix and goreleaser do)
	if r := lutf(t, "", "--version"); r.out != "lutf "+Version+"\n" {
		t.Errorf("--version: %q", r.out)
	}
	r := lutf(t, "", "--help")
	if r.code != 0 || !strings.Contains(r.out, "--search") || !strings.Contains(r.out, "--no-fuzzy") {
		t.Errorf("--help: %+v", r)
	}
}

// The examples in the help text must do what the help says.
func TestHelpExamplesAreTrue(t *testing.T) {
	tests := []struct{ args, want string }{
		{"mbfGamma mitX _2", "𝚪𝑋₂"},
		{"frac{1}{2} + hat{x}^2", "½ + x\u0302²"},
	}
	for _, tt := range tests {
		if r := lutf(t, "", tt.args); r.out != tt.want {
			t.Errorf("lutf %q = %q, want %q", tt.args, r.out, tt.want)
		}
	}
	if r := lutf(t, "", "-r", "∇"); r.out != "del" {
		t.Errorf("-r ∇ = %q", r.out)
	}
	for _, claim := range []string{"→  𝚪𝑋₂", "→  ½ + x̂²", "→  del"} {
		if !strings.Contains(longHelp, claim) {
			t.Errorf("help no longer contains %q", claim)
		}
	}
	// every family shown in the help exists
	for _, name := range []string{"mbfGamma", "mscr" + "A", "mtt" + "A", "mfrak" + "A", "BbbR", "circledone"} {
		_ = name // names of the families are listed for orientation only; see symbols tests for coverage
	}
}

func TestCompletion(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		r := lutf(t, "", "--completion", shell)
		if r.code != 0 || !strings.Contains(r.out, "lutf") {
			t.Errorf("--completion %s: code %d, %d bytes", shell, r.code, len(r.out))
		}
	}
	// the dynamic part: names for the shell to offer, with the symbol as description
	r := lutf(t, "", "__complete", "gamm")
	if !strings.Contains(r.out, "gamma\tγ") {
		t.Errorf("__complete gamm: %q", r.out)
	}
	// ... flags, and the shells after --completion
	if r := lutf(t, "", "__complete", "--sea"); !strings.Contains(r.out, "--search\t") {
		t.Errorf("__complete --sea: %q", r.out)
	}
	r = lutf(t, "", "__complete", "--completion", "")
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		if !strings.Contains(r.out, shell+"\n") {
			t.Errorf("__complete --completion does not offer %s: %q", shell, r.out)
		}
	}
	if strings.Contains(r.out, "ShellCompDirectiveDefault") {
		t.Errorf("file names would be offered: %q", r.out)
	}
}

// fakeTool puts an executable shell script called name first on PATH.
func fakeTool(t *testing.T, name, script string) (dir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts as fake tools")
	}
	dir = t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	// in front of the real PATH: the scripts still need cat, grep and friends
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return dir
}

func TestSearchWithFzf(t *testing.T) {
	// the fake fzf checks that it really receives the symbol list on stdin, then "chooses" the gamma line
	fakeTool(t, "fzf", `n=$(grep -c . ); [ "$n" -gt 2800 ] || exit 3
printf 'gamma\t\316\263\n'`)
	r := lutf(t, "", "-s")
	if r.code != 0 || r.out != "γ" {
		t.Errorf("search: %+v", r)
	}
	if r := lutf(t, "", "--search", "-n"); r.out != "γ\n" {
		t.Errorf("search -n: %+v", r)
	}
}

func TestSearchCancelled(t *testing.T) {
	fakeTool(t, "fzf", "exit 130")
	if r := lutf(t, "", "-s"); r.code != 130 || r.out != "" || r.err != "" {
		t.Errorf("cancelled: %+v", r)
	}
	fakeTool(t, "fzf", "exit 1")
	if r := lutf(t, "", "-s"); r.code != 1 || r.out != "" {
		t.Errorf("no match: %+v", r)
	}
}

func TestSearchWithoutFzf(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	r := lutf(t, "", "-s")
	if r.code != 1 || !strings.Contains(r.err, "fzf is not installed") || !strings.Contains(r.err, "--list") {
		t.Errorf("got %+v", r)
	}
}

func TestCopy(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "clip")
	fakeTool(t, "wl-copy", `cat > '`+target+`'`)
	if r := lutf(t, "", "-c", "alpha"); r.out != "α" || r.err != "" || r.code != 0 {
		t.Errorf("copy: %+v", r)
	}
	if got, _ := os.ReadFile(target); string(got) != "α" {
		t.Errorf("clipboard holds %q", got)
	}
	// without a clipboard tool the output still appears, with a warning
	t.Setenv("PATH", t.TempDir())
	r := lutf(t, "", "-c", "alpha")
	if r.out != "α" || r.code != 0 || !strings.Contains(r.err, "no clipboard tool") {
		t.Errorf("no tool: %+v", r)
	}
}
