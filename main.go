package main

import (
	// "bufio"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

var rootCmd = &cobra.Command{
	Use:   "utf8replacer",
	Short: "A CLI tool to replace words with UTF-8 symbols",
}

//	var replaceCmd = &cobra.Command{
//	    Use:   "replace",
//	    Short: "Replace words from stdin with UTF-8 symbols",
//	    Run: func(cmd *cobra.Command, args []string) {
//	        reader := bufio.NewReader(os.Stdin)
//	        for {
//	            input, err := reader.ReadString('\n')
//	            if err != nil {
//	                break
//	            }
//	            output := replaceSymbolsWithSlash(input)
//	            fmt.Print(output)
//	        }
//	    },
//	}
var replaceCmd = &cobra.Command{
	Use:   "replace",
	Short: "Replace words from stdin with UTF-8 symbols",
	Run: func(cmd *cobra.Command, args []string) {
		input, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading stdin:", err)
			os.Exit(1)
		}
		output := replaceSymbolsWithSlash(string(input))
		fmt.Print(output)
	},
}

var singleCmd = &cobra.Command{
	Use:   "single [word]",
	Short: "Replace a single word with UTF-8 symbol",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		input := args[0]
		output := replaceSymbols(input)
		fmt.Println(output)
	},
}

var multiCmd = &cobra.Command{
	Use:   "multi [text]",
	Short: "Replace multiple words with UTF-8 symbols, requires \\ at the beginning of tokens",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		input := strings.Join(args, " ")
		output := replaceSymbolsWithSlash(input)
		fmt.Println(output)
	},
}

func replaceSymbolsWithSlash(input string) string {
	// Create a slice to hold the keys from the latexSymbols map
	keys := make([]string, 0, len(latexSymbols))

	// Populate the keys slice with the keys from the latexSymbols map
	for key := range latexSymbols {
		keys = append(keys, key)
	}

	// Sort the keys by length in descending order to ensure longer keys are replaced first
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})

	// Replace each LaTeX symbol in the input string with its corresponding value from the latexSymbols map
	for _, key := range keys {
		input = strings.ReplaceAll(input, `\\`+key, latexSymbols[key])
	}

	// Split the input string into words
	words := strings.Fields(input)

	// Iterate over each word in the words slice
	for i, word := range words {
		// Check if the word starts with a backslash
		if strings.HasPrefix(word, `\`) {
			// Find the closest match for the word (excluding the backslash) in the latexSymbols map
			closest := findClosestMatch(word[1:])
			// Replace the word with the closest match
			words[i] = closest
		}
	}

	// Join the words back into a single string with spaces in between
	return strings.Join(words, " ")
}

// func replaceSymbolsWithSlash(input string) string {
//     tokens := strings.Split(input, `\\`)
//     for i, token := range tokens {
//         if token == "" {
//             continue
//         }
//         for key, value := range latexSymbols {
//             if strings.HasPrefix(token, key) {
//                 tokens[i] = value + token[len(key):]
//                 break
//             }
//         }
//     }

//     for i, token := range tokens {
//         if token == "" {
//             continue
//         }
//         if strings.HasPrefix(token, `\`) {
//             closest := findClosestMatch(token[1:])
//             tokens[i] = closest
//         }
//     }

//     return strings.Join(tokens, `\\`)
// }

func replaceSymbols(input string) string {
	keys := make([]string, 0, len(latexSymbols))
	for key := range latexSymbols {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})

	for _, key := range keys {
		input = strings.ReplaceAll(input, key, latexSymbols[key])
	}

	words := strings.Fields(input)
	for i, word := range words {
		closest := findClosestMatch(word)
		words[i] = closest
	}
	return strings.Join(words, " ")
}

func findClosestMatch(word string) string {
	minDistance := len(word)
	closest := word
	for key := range latexSymbols {
		distance := levenshteinDistance(word, key)
		if distance < minDistance {
			minDistance = distance
			closest = latexSymbols[key]
		}
	}
	return closest
}

func levenshteinDistance(a, b string) int {
	la, lb := len(a), len(b)
	d := make([][]int, la+1)
	for i := range d {
		d[i] = make([]int, lb+1)
	}
	for i := 0; i <= la; i++ {
		d[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		d[0][j] = j
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
		}
	}
	return d[la][lb]
}

func min(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

func main() {
	rootCmd.AddCommand(replaceCmd)
	rootCmd.AddCommand(singleCmd)
	rootCmd.AddCommand(multiCmd)
	// Add other commands as needed
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
