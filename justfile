default:
  @just --list

fmt:
  go run ./tools/gofmtcheck --write

lint:
  go run ./tools/gofmtcheck
  go vet ./...
  cd runner && npm run lint
  cd runner && npm run boundary-check

test:
  go test ./...
  cd runner && npm test

ci:
  go run ./tools/gofmtcheck
  go vet ./...
  go test ./...
  go build ./cmd/...
  cd runner && npm ci
  cd runner && npm run lint
  cd runner && npm test
  cd runner && npm run boundary-check

dev:
  @just --list
