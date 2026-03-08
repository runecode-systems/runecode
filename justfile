default:
  @just --list

fmt:
  @echo "fmt is a placeholder and will be extended by follow-on specs."

lint:
  @echo "lint is a placeholder and will be extended by follow-on specs."

test:
  @echo "test is a placeholder and will be extended by follow-on specs."

ci:
  @echo "Running CI smoke checks (tool versions)..."
  git --version
  go version
  gopls version
  node --version
  npm --version
  just --version
  jq --version
  rg --version
  fd --version
  curl --version

dev:
  @just --list
