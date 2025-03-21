package main

import (
	// "bufio"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
	"strings"
)

var rootCmd = &cobra.Command{
	Use:   "lutf [text]",
	Short: "A CLI tool to replace words with UTF-8 symbols",
	Long: `lutf is a command-line tool that replaces specific words in the input text with their corresponding UTF-8 symbols.
    
ⓘ Usage:
  lutf [text]    Replace words in the provided text argument
  lutf           Replace words in the text read from stdin

ⓘ Examples:
  • lutf "mbfGamma" "mitX" "_2"
  • echo "mbfGamma mitX _2" | lutf
  → output : 𝚪𝑋₂`,
	Run: func(cmd *cobra.Command, args []string) {
		var input string
		var e error

		if len(args) > 0 {
			input = strings.Join(args, " ")
		} else {
			var inputBytes []byte
			inputBytes, e = io.ReadAll(os.Stdin)
			if e != nil {
				fmt.Fprintln(os.Stderr, "Error reading stdin:", e)
				os.Exit(1)
			}
			input = string(inputBytes)
		}

		if input == "" {
			cmd.Help()
			return
		}

		output := replaceSymbols(string(input))
		output = strings.ReplaceAll(output, " ", "")
		fmt.Print(output)
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
