package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	// Import the functions from the main package
	"github.com/code-instable/latex2utf8/utils"
)

// besoin d'une majuscule pour que la variable/fonction soit exportée du package
var RootCmd = &cobra.Command{
	Use:   "lutf [text]",
	Short: "A CLI tool to replace words with UTF-8 symbols",
	Long: `lutf is a command-line tool that replaces specific words in the input text with their corresponding UTF-8 symbols.
    
ⓘ Usage:
  lutf [text]    Replace words in the provided text argument
  lutf           Replace words in the text read from stdin

ⓘ Examples:
  • lutf "mbfGamma" "mitX" "_2"
  • echo "mbfGamma mitX _2" | lutf
  → output : 𝚪𝑋₂
  
ⓘ families of symbols:
    • L : Ł Ø ¶
    • l : ł ø ˘
    • superscript (^{1}) : ᴬ ᴮ ᴰ
    • subscript (_{1}) : ₍ ₎ ₊
    • (mit)Bbb{A/one} : 𝔸 𝔹 ℂ
    • mscr : 𝒜 ℬ 𝒞
    • mtt : 𝙰 𝙱 𝙲
    • mit : 𝐴 𝐵 𝐶
    • mitAlpha : 𝛢 𝛥 𝛤
    • mitsans : 𝘈 𝘉 𝘊
    • mfrak : 𝔄 𝔅 ℭ
    • msans : 𝖠 𝖡 𝖢
    • mbf : 𝐀 𝐁 𝐂
    • mbf{Mu} : 𝚳 𝚴 𝚷
    • mbfit : 𝑨 𝑩 𝑪
    • mbfitsans : 𝘼 𝘽 𝘾
    • mbfsans : 𝗔 𝗕 𝗖
    • mbffrak : 𝕬 𝕭 𝕮
    • mbfscr : 𝓐 𝓑 𝓒
    • circled{one} : ⓪ ① ②
    • circ{one} : ➀ ➁ ➂
    • fcirc{one} : ➊ ➋ ➌
    • sq : 🄰 🄱 🄲
    • fsq : 🅰 🅱 🅲
    • circ{A} : Ⓐ Ⓑ Ⓒ
    • circ{a} : ⓐ ⓑ ⓒ
    • dcirc{one} : ⓵ ⓶ ⓷
    • fcirc{A} : 🅐 🅑 🅒
    • box[x][y] : ┏ ┳ ━
    • dbox[x][y] : ═ ═ ═
    • [dir]harpoon[supp] : ↼ ↾ ⇀
    • [dir]harpoons : ⥢ ⥦ ⥤
    • arrows : ⬾ ⇈ ⥆
    • measangle{l/r/u/d}to{n/s/e/w}{n/s/e/w} : ⦯ ⦮ ⦫
    • black[shape][dir] : ☻ ■ ⧫
    • bigblack : ▲ ▼
    • {a/c/d/v}dots : ⋰ · ⋯
    • shade{light/dense/bold} : ░ ▒ ▓
    • {r/u/l/d}tri : ▷ ◁ △
`,
	Run: func(cmd *cobra.Command, args []string) {
		var input string
		var e error

		if searchFlag {
			// log debug to stderr
			// fmt.Fprintln(os.Stderr, "Search flag is set. Searching for symbols...\n")
			// Call the search function if the search flag is set
			fmt.Print("Searching for symbols...\n")
			utils.SearchSymbols()
			return
		} else {
			// log debug to stderr
			// fmt.Fprintln(os.Stderr, "Input provided. Replacing symbols...")
			// read from stdin if no argument is provided
			if len(args) > 0 {
				// log debug to stderr
				// fmt.Fprintln(os.Stderr, "Input provided as argument. Replacing symbols...\n")
				input = strings.Join(args, " ")
			} else {
				// log debug to stderr
				// fmt.Fprintln(os.Stderr, "No input provided as argument. Reading from stdin...\n")
				var inputBytes []byte
				inputBytes, e = io.ReadAll(os.Stdin)
				if e != nil {
					fmt.Fprintln(os.Stderr, "Error reading stdin:", e)
					os.Exit(1)
				}
				input = string(inputBytes)
			}

			output := utils.ReplaceSymbols(string(input))
			output = strings.ReplaceAll(output, " ", "")
			fmt.Print(output)
			return
		}
	},
}
