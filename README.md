# latex2utf8

a **simple** cli tool to convert latex names to utf-8 characters

based on [unicode-latex vscode extension](https://github.com/ojsheikh/unicode-latex/)

```zsh
go build
ln -s <path_to>/latex2utf8 /usr/local/bin/lutf
# or
cp latex2utf8 /usr/local/bin/lutf
```

## usage

```zsh
lutf mbfGamma
#   output : 𝚪
lutf mitX_1
#   output : 𝑋₁
lutf mitX _1
#   output : 𝑋₁
lutf _n
#   output : ₙ
lutf varepsilon
#   output : ɛ
lutf BbbR
#   output : ℝ
# can also correct, and match to closest match
lutf bbbR
#   output : ℝ
lutf gammma
#        ^
#   output : γ
```

using the stdin

```zsh
echo "mbfGamma mitX neq mitu _0" | lutf
#   output : 𝚪𝑋≠𝑢₀
```

### the backslash is optional

`gamma` is as good as `\gamma`, and it is much easier to type. Use whichever you like:

```zsh
lutf gamma Gamma '\gamma'      # γ Γ γ
lutf '\alpha+\beta'            # α+β
```

The backslash is only *required* for one letter names, which are far too common in text to be read as
names without it: `\L` is `Ł`, `\o` is `ø`, `\S` is `§`.

A name written with a backslash is exact: no typo correction, and a warning if it does not exist.

### scripts, fractions, accents and fonts

These need no backslash either; they are recognised by the `{...}` that follows.

| input                          | output     |                                              |
| ------------------------------ | ---------- | -------------------------------------------- |
| `x_1` `x^2` `x_{12}` `x^{n+1}` | x₁ x² x₁₂ xⁿ⁺¹ | sub- and superscripts (`_n`, `^T` alone too) |
| `_12`                          | ₁₂         | an unbraced number is taken whole            |
| `frac{1}{2}`                   | ½          | vulgar fractions when Unicode has them       |
| `frac{12}{35}`                 | ¹²⁄₃₅      | otherwise super/subscript around ⁄           |
| `sqrt{2}` `sqrt[3]{x}`         | √2 ∛x      | roots                                        |
| `hat{a}` `tilde{n}` `acute{e}` | â ñ é      | accents, composed when Unicode has the letter |
| `hat{x}` `vec{v}` `bar{x}`     | x̂ v⃗ x̄      | ... and combined otherwise                   |
| `not{=}` `not{in}`             | ≠ ∉        |                                              |
| `overline{xy}`                 | x̅y̅         | `overline`, `underline`, `widehat`, `cancel` cover every letter |
| `mathbb{R}` `mathcal{A}` `mathfrak{g}` `mathbf{x}` | ℝ 𝒜 𝔤 𝐱 | math alphabets (`mbf{Mu}` is `mbfMu`) |
| `text{ and }`                  | ` and `    | copied as it is                              |
| `\"{o}` `\'e` `\c{c}` `\v{s}`  | ö é ç š    | the LaTeX text accents                       |

The word on its own keeps working as a symbol: `sqrt` is `√`, `tilde` is the combining `◌̃`, and `bar` or
`frac` without a `{` after them are just words.

### text that is not a name stays as it is

```zsh
lutf "x = alpha + beta"        # x = α + β
lutf hello world               # hello world
```

Whitespace between two converted names is dropped (`alpha beta` is `αβ`, so `mbfGamma mitX _2` works);
whitespace next to anything else, and line breaks, are kept. `-j` removes all of it.

To put a letter or digit right after a name, end the name with an empty group, as in LaTeX:
`alpha{}x` is `αx`, where `alpha x` would keep the space (`∇f` is written `nabla{}f`). The `{}` only
disappears when it directly follows a converted name; anywhere else it stays.

Words are only read as names when they are whole words. Case slips (`bbbR`, `GAMMA`), names run together
(`alphabeta`) and one typo in a longer word (`gammma`) are repaired; use `--no-fuzzy` for the typos and
`--strict` for none of it. Words typed entirely in lower case must match a name exactly, so `and` stays `and`.

### for prose: `--text`

Many names are also words (`to`, `in`, `sum`, `top`, `star`...). When converting running text, use
`--text`: then only the explicit syntax is converted, and ordinary words are left alone.

```zsh
lutf --text 'the sum of \alpha_{1} and x^2 is in R'
#   output : the sum of α₁ and x² is in R
```

## more tools

| flag                  |                                                                              |
| --------------------- | ---------------------------------------------------------------------------- |
| `-r`, `--reverse`     | symbols to names: `lutf -r ∇` is `del`. `--latex` writes `\del`. Feeding the result back to `lutf` gives the text again, so `∇f` becomes `del{}f` and `xγ` becomes `x\gamma`. This is checked on every call. The one thing that cannot come back is a space between two symbols (`α β`), which `lutf` drops |
| `-i`, `--info`        | code points, UTF-8 bytes and every name of each character                    |
| `-l`, `--list [text]` | list the names; arguments keep those containing them (or the symbol itself): `lutf -l arrow` |
| `-s`, `--search`      | browse and pick a symbol with [fzf](https://github.com/junegunn/fzf); prints it |
| `-c`, `--copy`        | also copy the result to the clipboard (`wl-copy`, `xclip`, `xsel`, `pbcopy` or `clip.exe`) |
| `--strict`            | exact names only, report the unknown ones, exit status 1 if there are any    |
| `-q`, `--quiet`       | no warnings for unknown names                                                |
| `--json`              | machine readable output (for editors and scripts)                            |
| `-n`, `--newline`     | end the output with a newline. None is added otherwise, even on a terminal    |
| `--completion SHELL`  | print a completion script for `bash`, `zsh`, `fish` or `powershell`, which also completes the names |
| `--version`           |                                                                              |

The output never ends with a newline unless you ask for one with `-n`, so an editor gets exactly the
characters (in zsh the missing newline shows as a `%`). A single trailing newline of one line of input
is dropped as well, so `echo alpha | lutf` gives just `α`.

### shell completions

`lutf --completion SHELL` prints a completion script. It completes the flags and the symbol names (zsh and
fish show the symbol next to each name), and the shell names after `--completion`.

```zsh
# zsh: put the script on your fpath, then (re)load completions
mkdir -p ~/.zfunc && lutf --completion zsh > ~/.zfunc/_lutf
# in ~/.zshrc, before compinit:
#   fpath=(~/.zfunc $fpath)
#   autoload -Uz compinit && compinit
# (or, after compinit:  source <(lutf --completion zsh) )

# bash: needs the bash-completion package (version 2)
mkdir -p ~/.local/share/bash-completion/completions
lutf --completion bash > ~/.local/share/bash-completion/completions/lutf
# (or, in ~/.bashrc:  source <(lutf --completion bash) )

# fish
lutf --completion fish > ~/.config/fish/completions/lutf.fish
```

### in an editor

Anything that pipes a selection through a command works. In [helix](https://helix-editor.com/), select
some text and press `|`, then type `lutf`; in vim or neovim, select lines and run `:'<,'>!lutf`.

## adding or fixing symbols

The symbols live in tab separated files in [`symbols/data/`](symbols/data), embedded in the binary:

| file            |                                                                             |
| --------------- | --------------------------------------------------------------------------- |
| `symbols.tsv`   | `name`⇥`character`. Names are letters (and digits after the first letter); `^x`/`_x` give a character's superscript/subscript |
| `aliases.tsv`   | `alias`⇥`existing name`, for other spellings of a symbol (`ldots`, `cyrBe`) |
| `commands.tsv`  | the commands taking a `{...}`: accents, math alphabets, `frac`, `sqrt`, `text` |
| `fractions.tsv`, `compose.tsv` | generated from Unicode by `scripts/gen_unicode_data.py`      |

`go test ./...` checks the files: names are valid and unique, every alias points at a symbol, every math
alphabet is complete, and every name converts to its value both bare and with a backslash.
`scripts/audit_upstream.py` compares the table with the unicode-latex project it was based on.

```zsh
scripts/test.zsh       # vet, test, and a few conversions by eye
scripts/compile.zsh    # binaries for every platform in ./build
```

## changes in 2.0

- Output no longer depends on luck: it used to differ from run to run for the same input.
- Text that is not a name is kept. Names used to be replaced inside any word, so `hello world`
  came out as a single `𝒹`.
- One letter names (`L`, `o`, `S`, ... and the Cyrillic `I`, `i`) need a backslash. The Cyrillic names
  also have `cyr` aliases (`cyrBe`, `cyri`).
- Spaces are kept next to text that is not converted (use `-j` for the old behaviour).
- `--search` works. `lutf _n` really gives `ₙ`, as this readme always said.
- New: the optional backslash, `x_{12}`, fractions, roots, accents, math alphabet commands,
  `--text`, `--strict`, `--reverse`, `--info`, `--list`, `--json`, `--copy`, completions,
  subscript letters, and the missing symbols of unicode-latex (music, joins, fractions, ...).
- Much faster on long input: 1000 words took 1.3 s and now take 8 ms (10 000 words: 14.5 s, now 11 ms).
  A single short conversion is about a millisecond slower than before, as the symbol files are now
  read from the binary at startup.
