VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  = -s -w -X main.version=$(VERSION)

.PHONY: all build app cli docs clean test ci

all: build

## Compiler tous les binaires principaux
build: app cli

## myr-app — serveur HTTP, API REST uniquement (Linux amd64 — tourne sur le serveur distant)
app:
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/myr-app-linux ./cmd/api

## myr — outil CLI admin (Linux amd64 — tourne sur le serveur distant)
cli:
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/myr-linux ./cmd/cli

## Générer les man pages dans docs/man/man1/
docs:
	go run ./cmd/mangen

## Tests
test:
	go test ./...

## Vérifications CI (isolation hexagonale ENF18 + licences AGPL ENF25)
ci:
	./scripts/ci/check-domain-imports.sh
	./scripts/ci/check-licenses.sh

## Déployer sur le serveur distant (compile Linux + SCP + systemd)
deploy:
	powershell -ExecutionPolicy Bypass -File scripts\deploy.ps1

## Nettoyage
clean:
	rm -rf bin
