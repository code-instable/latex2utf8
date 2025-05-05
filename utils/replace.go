package utils

import (
	"sort"
	"strings"
)

func ReplaceSymbols(input string) string {
	keys := make([]string, 0, len(LatexSymbols))
	for key := range LatexSymbols {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})

	for _, key := range keys {
		input = strings.ReplaceAll(input, key, LatexSymbols[key])
	}

	words := strings.Fields(input)
	for i, word := range words {
		closest := FindClosestMatch(word)
		words[i] = closest
	}
	return strings.Join(words, " ")
}

func ReplaceSymbolsWithSlash(input string) string {
	// Create a slice to hold the keys from the latexSymbols map
	keys := make([]string, 0, len(LatexSymbols))

	// Populate the keys slice with the keys from the latexSymbols map
	for key := range LatexSymbols {
		keys = append(keys, key)
	}

	// Sort the keys by length in descending order to ensure longer keys are replaced first
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})

	// Replace each LaTeX symbol in the input string with its corresponding value from the latexSymbols map
	for _, key := range keys {
		input = strings.ReplaceAll(input, `\\`+key, LatexSymbols[key])
	}

	// Split the input string into words
	words := strings.Fields(input)

	// Iterate over each word in the words slice
	for i, word := range words {
		// Check if the word starts with a backslash
		if strings.HasPrefix(word, `\`) {
			// Find the closest match for the word (excluding the backslash) in the latexSymbols map
			closest := FindClosestMatch(word[1:])
			// Replace the word with the closest match
			words[i] = closest
		}
	}

	// Join the words back into a single string with spaces in between
	return strings.Join(words, " ")
}
