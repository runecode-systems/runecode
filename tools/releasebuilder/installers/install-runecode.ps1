param(
  [string]$Tag,
  [switch]$Latest
)

$ErrorActionPreference = "Stop"

$Repo = "runecode-systems/runecode"
$CosignVersion = "v2.4.1"
$GhVersion = "v2.74.2"
$OidcIssuer = "https://token.actions.githubusercontent.com"

$CosignDigestWindowsAmd64 = "8d57f8a42a981d27290c4227271fa9f0f62ca6630eb4a21d316bd6b01405b87c"

$GhDigestWindowsAmd64 = "3ac27af5ee8dd13b1b0002e4e4764163683889cea7f21231c9951e300c95eb29"
$GhDigestWindowsArm64 = "73ddec7ec071254b4de64e95312eb40c4654c428b7081720c44ae9422aa6f1ef"

function Get-LatestTag {
  $releases = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases?per_page=1"
  if (-not $releases -or -not $releases[0].tag_name) {
    throw "Unable to resolve latest release tag"
  }
  return [string]$releases[0].tag_name
}

function Download-Asset {
  param(
    [string]$Version,
    [string]$AssetName,
    [string]$TargetPath
  )

  $url = "https://github.com/$Repo/releases/download/$Version/$AssetName"
  Invoke-WebRequest -Uri $url -OutFile $TargetPath
}

function Download-AndVerifyChecksum {
  param(
    [string]$Url,
    [string]$TargetPath,
    [string]$ExpectedHash
  )

  Invoke-WebRequest -Uri $Url -OutFile $TargetPath
  $actualHash = Get-Sha256Hex -Path $TargetPath
  if ($actualHash -ne $ExpectedHash.ToLowerInvariant()) {
    throw "Checksum mismatch for downloaded helper $TargetPath"
  }
}

function Ensure-Cosign {
  param([string]$TempDir)

  $cmd = Get-Command cosign -ErrorAction SilentlyContinue
  if ($cmd) {
    $versionLine = (& $cmd.Source version 2>$null | Select-String -Pattern 'GitVersion:\s+(\S+)' | Select-Object -First 1)
    if ($versionLine -and $versionLine.Matches[0].Groups[1].Value -eq $CosignVersion) {
      return @{ Path = $cmd.Source; Source = "system ($CosignVersion)" }
    }
  }

  $assetArch = if ($Arch -eq "amd64") { "amd64" } elseif ($Arch -eq "arm64") { "arm64" } else { throw "Unsupported arch for cosign: $Arch" }
  if ($assetArch -ne "amd64") {
    throw "Temporary cosign bootstrap is only supported on Windows amd64; install cosign $CosignVersion manually for $assetArch"
  }
  $cosignPath = Join-Path $TempDir "cosign.exe"
  $url = "https://github.com/sigstore/cosign/releases/download/$CosignVersion/cosign-windows-$assetArch.exe"
  Download-AndVerifyChecksum -Url $url -TargetPath $cosignPath -ExpectedHash $CosignDigestWindowsAmd64
  return @{ Path = $cosignPath; Source = "temporary ($CosignVersion)" }
}

function Ensure-Gh {
  param([string]$TempDir)

  $cmd = Get-Command gh -ErrorAction SilentlyContinue
  if ($cmd) {
    $versionLine = (& $cmd.Source --version 2>$null | Select-Object -First 1)
    if ($versionLine) {
      $parts = ($versionLine -split '\s+')
      if ($parts.Count -ge 3 -and $parts[2] -eq $GhVersion.TrimStart('v')) {
        return @{ Path = $cmd.Source; Source = "system ($GhVersion)" }
      }
    }
  }

  $assetArch = if ($Arch -eq "amd64") { "amd64" } elseif ($Arch -eq "arm64") { "arm64" } else { throw "Unsupported arch for gh: $Arch" }
  $zipName = "gh_$($GhVersion.TrimStart('v'))_windows_$assetArch.zip"
  $zipPath = Join-Path $TempDir $zipName
  $expectedHash = if ($assetArch -eq "amd64") { $GhDigestWindowsAmd64 } else { $GhDigestWindowsArm64 }
  Download-AndVerifyChecksum -Url "https://github.com/cli/cli/releases/download/$GhVersion/$zipName" -TargetPath $zipPath -ExpectedHash $expectedHash

  $extractDir = Join-Path $TempDir "gh"
  Expand-Archive -Path $zipPath -DestinationPath $extractDir -Force
  $ghPath = Get-ChildItem -Path $extractDir -Recurse -Filter "gh.exe" | Select-Object -First 1
  if (-not $ghPath) {
    throw "Downloaded gh binary not found"
  }

  return @{ Path = $ghPath.FullName; Source = "temporary ($GhVersion)" }
}

function Get-Sha256Hex {
  param([string]$Path)
  return (Get-FileHash -Path $Path -Algorithm SHA256).Hash.ToLowerInvariant()
}

$ArchMap = @{
  "AMD64" = "amd64"
  "ARM64" = "arm64"
}

$Arch = $ArchMap[$env:PROCESSOR_ARCHITECTURE]
if (-not $Arch) {
  throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"
}

if (-not $Tag) {
  if (-not $Latest) {
    throw "-Tag is required; pass -Latest to opt into automatic latest selection"
  }
  $Tag = Get-LatestTag
}

$Archive = "runecode_${Tag}_windows_${Arch}.zip"
$WorkflowIdentity = "https://github.com/$Repo/.github/workflows/release.yml@refs/tags/$Tag"
$InstallerName = "install-runecode.ps1"
$LocalScript = (Resolve-Path $PSCommandPath).Path

$TempDir = Join-Path $env:TEMP ("runecode-install-" + [guid]::NewGuid())
$null = New-Item -ItemType Directory -Path $TempDir -Force
$Pushed = $false

try {
  $cosign = Ensure-Cosign -TempDir $TempDir
  $gh = Ensure-Gh -TempDir $TempDir

  Write-Host "RuneCode installer verification"
  Write-Host "  repo: $Repo"
  Write-Host "  tag: $Tag"
  Write-Host "  archive: $Archive"
  Write-Host "  expected workflow identity: $WorkflowIdentity"
  Write-Host "  expected OIDC issuer: $OidcIssuer"
  Write-Host "  cosign: $($cosign.Path) ($($cosign.Source))"
  Write-Host "  gh: $($gh.Path) ($($gh.Source))"

  Push-Location $TempDir
  $Pushed = $true

  foreach ($asset in @($Archive, "$Archive.sig", "$Archive.pem", "$InstallerName.sig", "$InstallerName.pem", "SHA256SUMS", "SHA256SUMS.sig", "SHA256SUMS.pem")) {
    Download-Asset -Version $Tag -AssetName $asset -TargetPath (Join-Path $TempDir $asset)
  }

  & $cosign.Path verify-blob `
    --certificate-identity $WorkflowIdentity `
    --certificate-oidc-issuer $OidcIssuer `
    --signature "SHA256SUMS.sig" `
    --certificate "SHA256SUMS.pem" `
    "SHA256SUMS" | Out-Null

  & $cosign.Path verify-blob `
    --certificate-identity $WorkflowIdentity `
    --certificate-oidc-issuer $OidcIssuer `
    --signature "$InstallerName.sig" `
    --certificate "$InstallerName.pem" `
    $LocalScript | Out-Null

  & $gh.Path attestation verify $LocalScript --repo $Repo | Out-Null

  & $cosign.Path verify-blob `
    --certificate-identity $WorkflowIdentity `
    --certificate-oidc-issuer $OidcIssuer `
    --signature "$Archive.sig" `
    --certificate "$Archive.pem" `
    $Archive | Out-Null

  $line = Select-String -Path "SHA256SUMS" -Pattern ("\s" + [regex]::Escape($Archive) + '$') | Select-Object -First 1
  if (-not $line) {
    throw "SHA256SUMS missing entry for $Archive"
  }

  $parts = ($line.Line -split '\s+', 2)
  if ($parts.Count -ne 2 -or $parts[1] -ne $Archive) {
    throw "Malformed SHA256SUMS entry for $Archive"
  }

  $expected = $parts[0].ToLowerInvariant()
  $actual = Get-Sha256Hex -Path $Archive
  if ($expected -ne $actual) {
    throw "Checksum mismatch for $Archive"
  }

  $installerEntry = Select-String -Path "SHA256SUMS" -Pattern ("\s" + [regex]::Escape($InstallerName) + '$') | Select-Object -First 1
  if (-not $installerEntry) {
    throw "SHA256SUMS missing entry for $InstallerName"
  }

  $installerParts = ($installerEntry.Line -split '\s+', 2)
  if ($installerParts.Count -ne 2 -or $installerParts[1] -ne $InstallerName) {
    throw "Malformed SHA256SUMS entry for $InstallerName"
  }

  $expectedInstallerHash = $installerParts[0].ToLowerInvariant()
  $localHash = Get-Sha256Hex -Path $LocalScript
  if ($expectedInstallerHash -ne $localHash) {
    throw "Checksum mismatch for $InstallerName"
  }

  & $gh.Path attestation verify $Archive --repo $Repo | Out-Null

  Write-Host "  SHA256SUMS entry for $InstallerName: $($installerEntry.Line)"
  Write-Host "  local running installer checksum: $localHash  $LocalScript"

  Write-Host ""
  Write-Host "Verification summary:"
  Write-Host "  [PASS] Running installer signature verified"
  Write-Host "  [PASS] Running installer attestation verified"
  Write-Host "  [PASS] Running installer checksum matched SHA256SUMS"
  Write-Host "  [PASS] Signed SHA256SUMS verified"
  Write-Host "  [PASS] Signed archive verified"
  Write-Host "  [PASS] Archive checksum matched SHA256SUMS"
  Write-Host "  [PASS] GitHub attestation verified"

  $answer = Read-Host "Install RuneCode binaries to $env:LOCALAPPDATA\Programs\RuneCode\bin ? Type yes to continue"
  if ($answer -ne "yes") {
    throw "Installation aborted"
  }

  $extractDir = Join-Path $TempDir "unpack"
  Expand-Archive -Path $Archive -DestinationPath $extractDir -Force
  $packageDir = Join-Path $extractDir ("runecode_" + $Tag + "_windows_" + $Arch)
  if (-not (Test-Path $packageDir)) {
    throw "Expected package directory not found: $packageDir"
  }

  $installDir = Join-Path $env:LOCALAPPDATA "Programs\RuneCode\bin"
  $null = New-Item -ItemType Directory -Path $installDir -Force
  Copy-Item (Join-Path $packageDir "bin\*.exe") $installDir -Force
  Write-Host "Installed RuneCode binaries to $installDir"
} finally {
  if ($Pushed) {
    Pop-Location
  }
  Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
}
