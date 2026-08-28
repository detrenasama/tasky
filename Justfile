VERSION  := `git describe --tags --always --dirty`
RELEASES := "linux-amd64"

default: check

# Сборка для текущей платформы (включая фронтенд, встраиваемый в бинарь)
build: web-build
    CGO_ENABLED=0 go build -ldflags "-s -w -X main.version={{VERSION}}" -o dist/tasky .

# Установка зависимостей фронтенда (web/)
web-install:
    cd web && npm install

# Сборка фронтенда → web/dist (встраивается в бинарь через //go:embed)
web-build: web-install
    cd web && npm run build

# Локальная разработка: фоновый сервер tasky + Vite dev-сервер (HMR, прокси /api)
web-dev:
    go run . serve &
    cd web && npm run dev

# Установка в ~/.local/bin (переопределить: PREFIX=/usr/local)
install: build
    install -Dm755 dist/tasky {{env("PREFIX", env("HOME") + "/.local")}}/bin/tasky

test:
    go test ./...

check: test check-web
    go vet ./...
    @if test -n "`gofmt -l .`"; then gofmt -l .; exit 1; fi

# Проверка типов/линт фронтенда (tsc --noEmit).
check-web:
    cd web && npm run lint

run:
    go run .

clean:
    rm -rf dist

# Перегенерация gRPC-кода из proto/tasky.proto (нужен protoc + protoc-gen-go/go-grpc)
proto:
    PATH="{{env("HOME")}}/go/bin:${PATH}" protoc --go_out=. --go_opt=module=github.com/detrenasama/tasky \
        --go-grpc_out=. --go-grpc_opt=module=github.com/detrenasama/tasky proto/tasky.proto

# Сборка одного таргета: just build-target linux arm64 v1.2.3
build-target os arch ver:
    CGO_ENABLED=0 GOOS={{os}} GOARCH={{arch}} go build -ldflags "-s -w -X main.version={{ver}}" \
        -o dist/release/tasky-{{os}}-{{arch}} .

# Релиз: сборка фронтенда + выбор версии + все таргеты + tar.gz + SHA256SUMS
release: web-build
    RELEASES="{{RELEASES}}" bash scripts/release.sh
