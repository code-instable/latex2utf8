# Cross-compile lutf for every platform into ./build
# (for published releases use goreleaser, see .goreleaser.yaml)
#
# Plain POSIX shell, so it runs in zsh, bash and sh.
# Run it from anywhere: the script moves to the repository root itself.

cd "$(dirname "$0")/.." || exit 1

version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)
ldflags="-s -w -X github.com/code-instable/latex2utf8/cmd.Version=${version}"

mkdir -p build

while read -r goos goarch; do
  out="build/lutf-${goos}-${goarch}"
  [ "$goos" = windows ] && out="${out}.exe"
  echo "compiling ${goos}/${goarch} (${version})..."
  CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build -trimpath -ldflags "$ldflags" -o "$out" || exit 1
done <<EOF
windows amd64
windows arm64
darwin amd64
darwin arm64
linux amd64
linux arm64
EOF
