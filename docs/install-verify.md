# Install and Verify RuneCode Releases

RuneCode release assets are published on the GitHub Releases page for this repository.
This pre-alpha release flow currently ships flake-built archives containing the Go binaries from `cmd/`:

- `runecode-auditd`
- `runecode-broker`
- `runecode-launcher`
- `runecode-secretsd`
- `runecode-tui`

Each release also publishes:

- `SHA256SUMS`
- a canonical unsigned release manifest (`runecode_<tag>_release-manifest.json`)
- a keyless cosign signature and certificate for each primary asset (`.sig` and `.pem`)
- a release SBOM (`runecode_<tag>_sbom.spdx.json`)
- GitHub build provenance attestations

The canonical flake package emits the final versioned unsigned archives, `SHA256SUMS`, and the release manifest:

```sh
nix build --no-link .#release-artifacts
```

The release workflow generates the SBOM afterward, then signs and attests it separately. The canonical `SHA256SUMS` file covers the unsigned archives and release manifest.

`README.md` contains a shorter verified install path. This document adds release asset layout details, prerelease-aware latest lookup, `gh attestation verify`, pinned-version flows, and Windows full verification.

## Prerequisites

For the full verification flow below, install:

- `gh` (GitHub CLI)
- `cosign`

The commands also use the platform's built-in archive and checksum tooling:

- Linux: `tar`, `sha256sum`
- macOS: `tar`, `shasum`
- Windows: PowerShell `Expand-Archive`, `Get-FileHash`

The `latest` examples below resolve the newest published release including prereleases. `gh release view` without a tag only works after a non-prerelease release exists.

## Linux and macOS: latest release, full verification, install

```bash
set -euo pipefail

REPO="runecode-ai/runecode"
# Newest published release, including prereleases during pre-alpha.
# Ordered by creation date; assumes no out-of-order backport releases.
VERSION="$(gh release list --repo "$REPO" --exclude-drafts --limit 1 --json tagName --jq '.[0].tagName')"

if [ -z "$VERSION" ]; then
  printf 'no published release found for %s\n' "$REPO" >&2
  exit 1
fi

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) printf 'unsupported architecture: %s\n' "$ARCH" >&2; exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  *) printf 'unsupported operating system: %s\n' "$OS" >&2; exit 1 ;;
esac

ARCHIVE="runecode_${VERSION}_${OS}_${ARCH}.tar.gz"
WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

cd "$WORKDIR"

gh release download "$VERSION" --repo "$REPO" \
  --pattern "$ARCHIVE" \
  --pattern "$ARCHIVE.sig" \
  --pattern "$ARCHIVE.pem" \
  --pattern "SHA256SUMS" \
  --pattern "SHA256SUMS.sig" \
  --pattern "SHA256SUMS.pem"

cosign verify-blob \
  --certificate-identity "https://github.com/${REPO}/.github/workflows/release.yml@refs/tags/${VERSION}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  --signature "SHA256SUMS.sig" \
  --certificate "SHA256SUMS.pem" \
  "SHA256SUMS"

cosign verify-blob \
  --certificate-identity "https://github.com/${REPO}/.github/workflows/release.yml@refs/tags/${VERSION}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  --signature "${ARCHIVE}.sig" \
  --certificate "${ARCHIVE}.pem" \
  "$ARCHIVE"

if command -v sha256sum >/dev/null 2>&1; then
  grep -F "  ${ARCHIVE}" SHA256SUMS | sha256sum -c -
else
  grep -F "  ${ARCHIVE}" SHA256SUMS | shasum -a 256 -c -
fi

gh attestation verify "$ARCHIVE" --repo "$REPO"

mkdir unpack
tar -xzf "$ARCHIVE" -C unpack

PACKAGE_DIR="unpack/runecode_${VERSION}_${OS}_${ARCH}"

install -d "$HOME/.local/bin"
install -m 0755 "$PACKAGE_DIR"/bin/runecode-* "$HOME/.local/bin/"

printf 'Installed RuneCode binaries to %s\n' "$HOME/.local/bin"
printf 'Add that directory to PATH if it is not already present.\n'
```

## Linux and macOS: pinned release, full verification, install

If you prefer not to resolve `latest`, set the version explicitly and run the same flow.

```bash
VERSION="v0.1.0"
```

Replace the `VERSION=...` line in the previous script with the pinned tag you want.

## Windows PowerShell: latest release, full verification, install

```powershell
$ErrorActionPreference = "Stop"

$Repo = "runecode-ai/runecode"
# Newest published release, including prereleases during pre-alpha.
# Ordered by creation date; assumes no out-of-order backport releases.
$Version = gh release list --repo $Repo --exclude-drafts --limit 1 --json tagName --jq '.[0].tagName'
if (-not $Version) {
  throw "No published release found for $Repo"
}

$ArchMap = @{
  "AMD64" = "amd64"
  "ARM64" = "arm64"
}

$Arch = $ArchMap[$env:PROCESSOR_ARCHITECTURE]
if (-not $Arch) {
  throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"
}

$Archive = "runecode_${Version}_windows_${Arch}.zip"
$WorkDir = Join-Path $env:TEMP ("runecode-" + [guid]::NewGuid())
$null = New-Item -ItemType Directory -Force -Path $WorkDir
$PushedLocation = $false

try {
  Push-Location $WorkDir
  $PushedLocation = $true

  gh release download $Version --repo $Repo `
    --pattern $Archive `
    --pattern "$Archive.sig" `
    --pattern "$Archive.pem" `
    --pattern "SHA256SUMS" `
    --pattern "SHA256SUMS.sig" `
    --pattern "SHA256SUMS.pem"

  cosign verify-blob `
    --certificate-identity "https://github.com/$Repo/.github/workflows/release.yml@refs/tags/$Version" `
    --certificate-oidc-issuer "https://token.actions.githubusercontent.com" `
    --signature "SHA256SUMS.sig" `
    --certificate "SHA256SUMS.pem" `
    "SHA256SUMS"

  cosign verify-blob `
    --certificate-identity "https://github.com/$Repo/.github/workflows/release.yml@refs/tags/$Version" `
    --certificate-oidc-issuer "https://token.actions.githubusercontent.com" `
    --signature "$Archive.sig" `
    --certificate "$Archive.pem" `
    $Archive

  $Match = Select-String -Path "SHA256SUMS" -Pattern ("\s" + [regex]::Escape($Archive) + '$')
  if (-not $Match) {
    throw "SHA256SUMS is missing an entry for $Archive"
  }

  $Fields = ($Match.Line -split '\s+', 2)
  if ($Fields.Count -ne 2 -or $Fields[1] -ne $Archive) {
    throw "SHA256SUMS entry is malformed for $Archive"
  }

  $ExpectedHash = $Fields[0].ToLower()
  $ActualHash = (Get-FileHash -Path $Archive -Algorithm SHA256).Hash.ToLower()
  if ($ActualHash -ne $ExpectedHash) {
    throw "Checksum mismatch for $Archive"
  }

  gh attestation verify $Archive --repo $Repo

  $ExtractDir = Join-Path $WorkDir "unpack"
  Expand-Archive -Path $Archive -DestinationPath $ExtractDir -Force

  $PackageDir = Join-Path $ExtractDir ("runecode_" + $Version + "_windows_" + $Arch)
  $InstallDir = Join-Path $env:LOCALAPPDATA "Programs\RuneCode\bin"
  $null = New-Item -ItemType Directory -Force -Path $InstallDir
  Copy-Item "$PackageDir\bin\*.exe" $InstallDir -Force

  Write-Host "Installed RuneCode binaries to $InstallDir"
  Write-Host "Add that directory to PATH if it is not already present."
} finally {
  if ($PushedLocation) {
    Pop-Location
  }
  Remove-Item -Recurse -Force $WorkDir -ErrorAction SilentlyContinue
}
```

## Windows PowerShell: pinned release, full verification, install

If you prefer not to resolve `latest`, set the version explicitly and run the same flow.

```powershell
$Version = "v0.1.0"
```

Replace the `$Version=...` line in the previous script with the pinned tag you want.

## What the verification steps prove

- `cosign verify-blob` on `SHA256SUMS` proves the checksum manifest was signed by the official `release.yml` workflow for that exact tag.
- `cosign verify-blob` on the archive proves the archive itself was signed by the same workflow identity.
- checksum verification proves the file you downloaded matches the canonical checksum manifest emitted by the flake-built unsigned release set.
- the SBOM is verified separately through its signature/certificate and GitHub attestation rather than the canonical checksum manifest.
- `gh attestation verify` proves GitHub recorded build provenance for the downloaded asset.

If any verification step fails, stop and do not install the binaries.
