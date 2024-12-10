go build

./latex2utf8 single nabla
./latex2utf8 single mbfGamma


./latex2utf8 multi '\\gamma' '\\mitX' '\\mbfGamma'


echo '\\gamma' '\\mitX' '\\mbfGamma' | ./latex2utf8 replace
echo '\\gamma \\mitX \\mbfGamma' | ./latex2utf8 replace

echo '\\gamma \\mitX \\mbfGamma
\\varphi \\mapsto \\bbbR
' | ./latex2utf8 replace  

./latex2utf8 single BbbR
./latex2utf8 single bbbR

# ln -s /Users/instable/Github/latex2utf8/latex2utf8 ~/.local/bin/latex2utf8
# ln -s /Users/instable/Github/latex2utf8/latex2utf8 /usr/local/bin/latex2utf8

