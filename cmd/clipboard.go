package cmd

import (
	"bytes"
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

// copyToClipboard copies s if --copy was given. A tool that is installed but cannot
// work (wl-copy in an X11 session, say) is skipped in favour of the next one. Failing
// to copy is only a warning: the text has been written to stdout anyway.
func (r *runner) copyToClipboard(s string) {
	if !r.flags.copy {
		return
	}
	var failure string
	for _, tool := range clipboardTools {
		path, err := exec.LookPath(tool[0])
		if err != nil {
			continue
		}
		cmd := exec.Command(path, tool[1:]...)
		cmd.Stdin = strings.NewReader(s)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			failure = fmt.Sprintf("copying with %s failed: %v", tool[0], err)
			if msg := strings.TrimSpace(stderr.String()); msg != "" {
				failure += ": " + msg
			}
			continue
		}
		return
	}
	if failure == "" {
		failure = "no clipboard tool found (tried wl-copy, xclip, xsel, pbcopy, clip.exe)"
	}
	_, _ = fmt.Fprintln(r.io.Err, "lutf: "+failure)
}
