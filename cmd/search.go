package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// runSearch lets the user pick a symbol with fzf and prints it. Given something to look for, it starts
// fzf with the names closest to it, best first, in that order (see --find): fzf alone only matches
// letters in order, so it cannot find Gamma from "Gamme".
func (r *runner) runSearch(queries []string) error {
	fzf, err := exec.LookPath("fzf")
	if err != nil {
		return &exitError{code: exitFailure, err: errors.New("fzf is not installed; install it to search, or use --find or --list to print the symbols")}
	}
	entries := r.set.Entries()
	fzfArgs := []string{"--reverse", "--header", "LaTeX symbol search", "--delimiter", "\t", "--nth", "1"}
	if len(queries) > 0 {
		entries, _ = r.ranked(queries, r.limitOr(defaultSearchLimit))
		if len(entries) == 0 {
			return &exitError{code: exitFailure, err: fmt.Errorf("nothing close to %s", strings.Join(quoted(queries), " or "))}
		}
		fzfArgs = append(fzfArgs, "--no-sort") // keep the order of the ranking
	}
	width := 0
	for _, e := range entries {
		width = max(width, len(e.Name))
	}
	var list strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&list, "%-*s\t%s\n", width, e.Name, e.Value)
	}

	cmd := exec.Command(fzf, fzfArgs...)
	cmd.Stdin = strings.NewReader(list.String())
	var chosen bytes.Buffer
	cmd.Stdout = &chosen
	cmd.Stderr = r.io.Err // fzf draws its interface on the terminal, not on stdout
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			// 1: nothing matched, 130: interrupted. Neither is an error worth a message.
			return &exitError{code: ee.ExitCode()}
		}
		return &exitError{code: exitFailure, err: fmt.Errorf("running fzf: %w", err)}
	}
	line := strings.TrimRight(chosen.String(), "\r\n")
	name, symbol, ok := strings.Cut(line, "\t")
	if !ok {
		return &exitError{code: exitFailure, err: fmt.Errorf("unexpected fzf output %q", line)}
	}
	if r.flags.jsonOut {
		if err := r.writeJSON(struct {
			Name   string `json:"name"`
			Symbol string `json:"symbol"`
		}{strings.TrimSpace(name), symbol}); err != nil {
			return err
		}
	} else if err := r.write(symbol); err != nil {
		return err
	}
	r.copyToClipboard(symbol)
	return nil
}
