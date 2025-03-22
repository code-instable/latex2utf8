mkdir -p build

# Windows amd64
echo "➀ compiling for windows amd64..."
GOOS=windows GOARCH=amd64 go build -o build/lutf-windows-amd64

# Windows arm64
echo "➁ compiling for windows arm64..."
GOOS=windows GOARCH=arm64 go build -o build/lutf-windows-arm64

# macOS amd64
echo "➂ compiling for macos amd64..."
GOOS=darwin GOARCH=amd64 go build -o build/lutf-darwin-amd64

# macOS arm64
echo "➃ compiling for macos arm64..."
GOOS=darwin GOARCH=arm64 go build -o build/lutf-darwin-arm64

# Linux amd64
echo "➄ compiling for linux amd64..."
GOOS=linux GOARCH=amd64 go build -o build/lutf-linux-amd64

# Linux arm64
echo "➅ compiling for linux arm64..."
GOOS=linux GOARCH=arm64 go build -o build/lutf-linux-arm64
