# latex2utf8
a **simple** cli tool to convert latex strings to utf-8 characters

based on [unicode-latex vscode extension](https://github.com/ojsheikh/unicode-latex/)

```zsh
go build
ln -s <path_to>/latex2utf8 /usr/local/bin/lutf
# or
cp latex2utf8 /usr/local/bin/lutf
```


```zsh
lutf mbfGamma
#   output : 𝚪
lutf mitX_1
#   output : 𝑋₁
lutf mitX _1
#   output : 𝑋₁
lutf _n
#   output : ₙ
lutf varepsilon
#   output : ɛ
lutf BbbR
#   output : ℝ
# can also correct, and match to closest match
lutf bbbR
#   output : ℝ
lutf gammma
#        ^
#   output : γ
```

using the stdin

```zsh
echo "mbfGamma mitX neq mitu _0" | lutf
#   output : 𝚪𝑋≠𝑢₀
```
