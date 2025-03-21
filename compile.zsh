mkdir -p build
# Windows amd64
GOOS=windows GOARCH=amd64 go build -o build/lutf-windows-amd64

# Windows arm64
GOOS=windows GOARCH=arm64 go build -o build/lutf-windows-arm64

# macOS amd64
GOOS=darwin GOARCH=amd64 go build -o build/lutf-darwin-amd64

# macOS arm64
GOOS=darwin GOARCH=arm64 go build -o build/lutf-darwin-arm64

# Linux amd64
GOOS=linux GOARCH=amd64 go build -o build/lutf-linux-amd64

# Linux arm64
GOOS=linux GOARCH=arm64 go build -o build/lutf-linux-arm64
