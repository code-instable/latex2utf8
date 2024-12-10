# latex2utf8
a cli tool to convert latex strings to utf-8 characters

based on [unicode-latex vscode extension](https://github.com/ojsheikh/unicode-latex/)

```zsh
go build
ln -s <path_to>/latex2utf8 /usr/local/bin/latex2utf8
ln -s <path_to>/latex2utf8 /usr/local/bin/lutf
```


```zsh
latex2utf8 single mbfGamma
#   output : 𝚪
latex2utf8 single mitX_1
#   output : 𝑋₁
latex2utf8 single _n
#   output : ₙ
latex2utf8 single varepsilon
#   output : ɛ
latex2utf8 single BbbR
#   output : ℝ
# can also correct, and match to closest match
latex2utf8 single bbbR
#   output : ℝ
```

(TODO) WIP : (still has some edge cases)
```zsh
latex2utf8 multi \\mbfGamma \\varepsilon \\mitW
#   output : 𝚪 ɛ 𝑊
```


(TODO) WIP : with stdin (partially working)
```zsh
echo '\\mbfGamma \\varepsilon \\mitW' | latex2utf8 replace
#   output : 𝚪 ɛ 𝑊
```
