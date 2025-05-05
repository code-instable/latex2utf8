package utils

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"syscall"
)

func SearchSymbols() {
	// Check if latexSymbols is initialized
	if LatexSymbols == nil || len(LatexSymbols) == 0 {
		fmt.Fprintln(os.Stderr, "Error: Symbol database is empty")
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Starting symbol search...")

	// Get all identifiers and sort them
	identifiers := make([]string, 0, len(LatexSymbols))
	for id := range LatexSymbols {
		identifiers = append(identifiers, id)
	}
	sort.Strings(identifiers)

	// Find the longest identifier to determine padding
	maxLength := 0
	for _, id := range identifiers {
		if len(id) > maxLength {
			maxLength = len(id)
		}
	}

	// Add some extra padding to separate identifiers from characters
	padding := maxLength + 10

	// Create the formatted list
	var formattedList strings.Builder
	for _, id := range identifiers {
		symbol := LatexSymbols[id]
		formattedList.WriteString(fmt.Sprintf("%-*s%s\n", padding, id, symbol))
	}

	fmt.Fprintln(os.Stderr, "Checking for fzf...")

	// Check if fzf is installed
	if _, err := exec.LookPath("fzf"); err != nil {
		fmt.Fprintf(os.Stderr, "Error: fzf is not installed. Please install fzf to use the search feature.\n")
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Launching fzf...")

	// Call fzf with the formatted list as input
	cmd := exec.Command("fzf", "--ansi", "--reverse", "--header", "LaTeX Symbol Search")
	cmd.Stdin = strings.NewReader(formattedList.String())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Make sure the command runs in the foreground
	cmd.Stdin = os.Stdin

	fmt.Fprintln(os.Stderr, "Running fzf...")

	// Execute fzf and handle exit gracefully
	err := cmd.Run()
	if err != nil {
		// Check if the error is just fzf exiting without a selection (which is normal)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if status.ExitStatus() == 130 {
					// Normal exit condition for fzf (Ctrl+C)
					return
				}
			}
		}
		fmt.Fprintf(os.Stderr, "Error running fzf: %v\n", err)
		os.Exit(1)
	}
}
