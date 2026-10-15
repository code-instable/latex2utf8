package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/code-instable/latex2utf8/convert"
	"github.com/code-instable/latex2utf8/symbols"
)

func (r *runner) writeJSON(v any) error {
	enc := json.NewEncoder(r.io.Out)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// runList prints every symbol whose name contains one of the arguments (or all
// of them if there are none), or whose value is one of them.
func (r *runner) runList(filters []string) error {
	var entries []symbols.Entry
	for _, e := range r.set.Entries() {
		if matchesAny(e, filters) {
			entries = append(entries, e)
		}
	}
	if r.flags.jsonOut {
		out := make([]struct {
			Name   string `json:"name"`
			Symbol string `json:"symbol"`
			Alias  bool   `json:"alias"`
		}, len(entries))
		for i, e := range entries {
			out[i].Name, out[i].Symbol, out[i].Alias = e.Name, e.Value, e.Alias
		}
		return r.writeJSON(out)
	}
	width := 0
	if r.io.OutTTY {
		for _, e := range entries {
			width = max(width, len(e.Name))
		}
	}
	var b strings.Builder
	for _, e := range entries {
		if r.io.OutTTY {
			fmt.Fprintf(&b, "%-*s  %s\n", width, e.Name, e.Value)
		} else {
			fmt.Fprintf(&b, "%s\t%s\n", e.Name, e.Value)
		}
	}
	if _, err := io.WriteString(r.io.Out, b.String()); err != nil {
		return err
	}
	if len(entries) == 0 && len(filters) > 0 {
		return &exitError{code: exitFailure}
	}
	return nil
}

func matchesAny(e symbols.Entry, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	name := strings.ToLower(e.Name)
	for _, f := range filters {
		if e.Value == f || strings.Contains(name, strings.ToLower(f)) {
			return true
		}
	}
	return false
}

// runInfo describes every character of the input.
func (r *runner) runInfo(input string) error {
	infos := convert.Describe(input, r.set)
	if r.flags.jsonOut {
		out := make([]struct {
			Char       string   `json:"char"`
			CodePoints []string `json:"codepoints"`
			UTF8       []string `json:"utf8"`
			Names      []string `json:"names"`
		}, len(infos))
		for i, in := range infos {
			out[i].Char, out[i].CodePoints, out[i].UTF8, out[i].Names = in.Text, codePoints(in), hexBytes(in), nonNil(in.Names)
		}
		return r.writeJSON(out)
	}
	var b strings.Builder
	for _, in := range infos {
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\n", in.Text, strings.Join(codePoints(in), " "), strings.Join(hexBytes(in), " "), strings.Join(in.Names, " "))
	}
	_, err := io.WriteString(r.io.Out, b.String())
	return err
}

func codePoints(in convert.Info) []string {
	out := make([]string, len(in.Runes))
	for i, c := range in.Runes {
		out[i] = fmt.Sprintf("U+%04X", c)
	}
	return out
}

func hexBytes(in convert.Info) []string {
	out := make([]string, len(in.Bytes))
	for i, c := range in.Bytes {
		out[i] = fmt.Sprintf("%02x", c)
	}
	return out
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// completeNames offers symbol names for shell completion, each with its symbol as description.
func (r *runner) completeNames(_ *cobra.Command, _ []string, prefix string) ([]string, cobra.ShellCompDirective) {
	var out []string
	for _, e := range r.set.Entries() {
		if strings.HasPrefix(e.Name, prefix) {
			out = append(out, e.Name+"\t"+e.Value)
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func (r *runner) printCompletion(cmd *cobra.Command, shell string) error {
	root := cmd.Root()
	switch shell {
	case "bash":
		return root.GenBashCompletionV2(r.io.Out, true)
	case "zsh":
		return root.GenZshCompletion(r.io.Out)
	case "fish":
		return root.GenFishCompletion(r.io.Out, true)
	case "powershell":
		return root.GenPowerShellCompletionWithDesc(r.io.Out)
	}
	return usageError{fmt.Errorf("unknown shell %q: want bash, zsh, fish or powershell", shell)}
}
