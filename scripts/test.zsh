# Run the checks, then a few conversions by eye.
#
# Plain POSIX shell, so it runs in zsh, bash and sh.
# Run it from anywhere: the script moves to the repository root itself.

cd "$(dirname "$0")/.." || exit 1

go vet ./... && go test ./... || exit 1

bin=$(mktemp -d)/lutf
go build -o "$bin" || exit 1

"$bin" nabla ; printf "\n"
"$bin" mbfGamma ; printf "\n"
"$bin" gamma mitX mbfGamma ; printf "\n"
echo "mbfGamma mitX neq mitu _0" | "$bin" ; printf "\n"
"$bin" 'x = alpha + frac{1}{2}hat{y}^2' ; printf "\n"
