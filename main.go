package main

import (
	"os"

	"github.com/code-instable/latex2utf8/cmd"
)

func main() {
	os.Exit(cmd.Run(os.Args[1:], cmd.StdIO()))
}
