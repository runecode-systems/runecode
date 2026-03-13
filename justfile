golangci_lint := "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8"

default:
  @just --list

fmt:
  go run ./tools/gofmtcheck --write

lint:
  go run ./tools/gofmtcheck
  go run {{golangci_lint}} run
  go vet ./...
  go run ./tools/checksourcequality
  cd runner && npm run lint
  cd runner && npm run boundary-check

test:
  go test ./...
  cd runner && npm test

ci:
  go run ./tools/gofmtcheck
  go run {{golangci_lint}} run
  go vet ./...
  go run ./tools/checksourcequality
  go test ./...
  go build ./cmd/...
  cd runner && npm ci
  cd runner && npm run lint
  cd runner && npm test
  cd runner && npm run boundary-check

dev:
  @just --list
