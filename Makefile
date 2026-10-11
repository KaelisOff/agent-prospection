# Construire, tester et empaqueter l'agent. `make help` pour la liste.
ID      := $(shell sed -n 's/^id *= *"\([^"]*\)".*/\1/p' manifest.toml)
VERSION := $(shell sed -n 's/^version *= *"\([^"]*\)".*/\1/p' manifest.toml)
PACKAGE := dist/$(ID)-$(VERSION).kagent

.PHONY: help test check package verify clean

help: ## Liste des commandes
	@grep -E '^[a-z]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  make %-8s %s\n", $$1, $$2}'

test: ## Lance les tests (box simulée)
	go vet ./...
	go test -race ./...

check: ## Vérifie le manifeste avec les règles de la box
	go tool kagent -check manifest.toml

package: check ## Construit le paquet .kagent dans dist/
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILDFLAGS) -trimpath -ldflags '-s -w' -o dist/agent .
	go tool kagent -manifest manifest.toml -executable dist/agent -o $(PACKAGE)

verify: package ## Reconstruit tout et vérifie que le paquet est identique (reproductible)
	@mkdir -p dist/check && cp $(PACKAGE) dist/check/first.kagent
	@$(MAKE) --no-print-directory package BUILDFLAGS=-a >/dev/null
	cmp dist/check/first.kagent $(PACKAGE) && echo "reproductible : ok"

clean: ## Supprime dist/
	rm -rf dist
