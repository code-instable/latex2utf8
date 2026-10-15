package cmd

import (
	"fmt"
	"os/exec"
	"strings"
)

// clipboardTools are the programs that put their stdin on the clipboard, in order of preference.
var clipboardTools = [][]string{
	{"wl-copy"},
	{"xclip", "-selection", "clipboard"},
	{"xsel", "--clipboard", "--input"},
	{"pbcopy"},
	{"clip.exe"}, // Windows, and WSL
}

// copyToClipboard copies s if --copy was given. Failing to do so is only a warning:
// the text has been written to stdout anyway.
func (r *runner) copyToClipboard(s string) {
	if !r.flags.copy {
		return
	}
	for _, tool := range clipboardTools {
		path, err := exec.LookPath(tool[0])
		if err != nil {
			continue
		}
		cmd := exec.Command(path, tool[1:]...)
		cmd.Stdin = strings.NewReader(s)
		if err := cmd.Run(); err != nil {
			_, _ = fmt.Fprintf(r.io.Err, "lutf: copying with %s failed: %v\n", tool[0], err)
			return
		}
		return
	}
	_, _ = fmt.Fprintln(r.io.Err, "lutf: no clipboard tool found (tried wl-copy, xclip, xsel, pbcopy, clip.exe)")
}
