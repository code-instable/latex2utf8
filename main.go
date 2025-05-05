package main

import (
	"fmt"
	"os"

	"github.com/code-instable/latex2utf8/cmd" // Import the cmd package
)

func main() {
	// execute the rootCmd defined in cmd/root.go
	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
