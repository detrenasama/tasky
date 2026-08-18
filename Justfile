VERSION  := `git describe --tags --always --dirty`
RELEASES := "linux-amd64"

default: check

# Сборка для текущей платформы
build:
    CGO_ENABLED=0 go build -ldflags "-s -w -X main.version={{VERSION}}" -o dist/tasky .

# Установка в ~/.local/bin (переопределить: PREFIX=/usr/local)
install: build
    install -Dm755 dist/tasky {{env("PREFIX", env("HOME") + "/.local")}}/bin/tasky

test:
    go test ./...

check: test
    go vet ./...
    @if test -n "`gofmt -l .`"; then gofmt -l .; exit 1; fi

run:
    go run .

clean:
    rm -rf dist

# Перегенерация gRPC-кода из proto/tasky.proto (нужен protoc + protoc-gen-go/go-grpc)
proto:
    PATH="{{env("HOME")}}/go/bin:${PATH}" protoc --go_out=. --go_opt=module=github.com/detrenasama/tasky \
        --go-grpc_out=. --go-grpc_opt=module=github.com/detrenasama/tasky proto/tasky.proto

# Сборка одного таргета: just build-target linux arm64
build-target os arch:
    CGO_ENABLED=0 GOOS={{os}} GOARCH={{arch}} go build -ldflags "-s -w -X main.version={{VERSION}}" \
        -o dist/release/tasky-{{os}}-{{arch}} .

# Релиз: все таргеты + tar.gz + SHA256SUMS + подсказка публикации
release:
    @rm -rf dist/release && mkdir -p dist/release dist/release/stage
    @for t in {{RELEASES}}; do \
        os=${t%-*}; arch=${t#*-}; \
        just build-target $os $arch; \
        cp dist/release/tasky-$os-$arch dist/release/stage/tasky; \
        tar czf dist/release/tasky-$os-$arch.tar.gz -C dist/release/stage tasky; \
    done
    @rm -rf dist/release/stage
    @cd dist/release && sha256sum *.tar.gz > SHA256SUMS
    @echo ""
    @echo "Готово: dist/release/*.tar.gz + SHA256SUMS"
    @echo "Публикация:"
    @echo "  gh release create v{{VERSION}} dist/release/*.tar.gz dist/release/SHA256SUMS \\"
    @echo "      --title 'Tasky v{{VERSION}}' --notes '...'"
