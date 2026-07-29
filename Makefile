VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS    = -s -w -X main.version=$(VERSION)
SWAG_VERSION = v1.16.4

.PHONY: all build api cli docs docs-api clean test ci

all: build

## Compiler tous les binaires principaux
build: api cli

## myr-api — serveur HTTP, API REST uniquement (Linux amd64 — tourne sur le serveur distant)
api:
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/myr-api-linux ./cmd/api

## myr — outil CLI admin (Linux amd64 — tourne sur le serveur distant)
cli:
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/myr-cli ./cmd/cli

## Générer les man pages dans docs/man/man1/
docs:
	go run ./cmd/mangen

## Générer la spec OpenAPI (api/swagger.json, api/swagger.yaml) depuis les
## commentaires swag des handlers REST — outil de dev uniquement (go run,
## aucune dépendance ajoutée à go.mod/vendor, voir specs/3-Conception/API_REST.md §13)
docs-api:
	go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) init \
		-g cmd/api/main.go \
		-o api \
		--outputTypes json,yaml \
		--parseDependency --parseInternal

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
