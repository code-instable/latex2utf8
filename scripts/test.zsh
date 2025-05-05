if [[ $0 == "test.zsh" ]]; then
  cd ..
fi

pwd
go build -o lutf

./lutf nabla ; printf "\n"
./lutf mbfGamma ; printf "\n"


./lutf 'gamma' 'mitX' 'mbfGamma' ; printf "\n"
echo "mbfGamma mitX neq mitu _0" | ./lutf ; printf "\n"