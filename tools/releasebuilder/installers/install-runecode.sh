#!/usr/bin/env bash
set -euo pipefail

REPO="runecode-systems/runecode"
COSIGN_VERSION="v2.4.1"
GH_VERSION="v2.74.2"
WORKFLOW_IDENTITY_PREFIX="https://github.com/${REPO}/.github/workflows/release.yml@refs/tags/"
OIDC_ISSUER="https://token.actions.githubusercontent.com"
DEFAULT_TO_LATEST=false

COSIGN_DIGEST_LINUX_AMD64="8b24b946dd5809c6bd93de08033bcf6bc0ed7d336b7785787c080f574b89249b"
COSIGN_DIGEST_LINUX_ARM64="3b2e2e3854d0356c45fe6607047526ccd04742d20bd44afb5be91fa2a6e7cb4a"
COSIGN_DIGEST_DARWIN_AMD64="666032ca283da92b6f7953965688fd51200fdc891a86c19e05c98b898ea0af4e"
COSIGN_DIGEST_DARWIN_ARM64="13343856b69f70388c4fe0b986a31dde5958e444b41be22d785d3dc5e1a9cc62"

GH_DIGEST_LINUX_AMD64="c421091ae5800390e6aef1f50bfda59cc1d4f2ef2200bcd4e1a662c05c28c444"
GH_DIGEST_LINUX_ARM64="f0b07f0aeaf00f137df1bd33a76e717b1945f4b83bd6a3296b365414d3eb413f"
GH_DIGEST_DARWIN_AMD64="c716ecdd9d5e381e521d772caa4dea29d647d465b94b1c67df9154e71f33451e"
GH_DIGEST_DARWIN_ARM64="42dceea5beae957a8a9545e38043a6d75fa3b5505fdcea75a278f873f0845b26"

usage() {
  cat <<'EOF'
Usage: install-runecode.sh [--tag <tag>] [--version <tag>]

Downloads, verifies, and installs RuneCode from GitHub release assets.
Pass --latest to resolve the newest published release automatically.
EOF
}

die() {
  printf 'error: %s\n' "$1" >&2
  exit 1
}

resolve_tag() {
  if [ -n "${REQUESTED_TAG}" ]; then
    printf '%s\n' "${REQUESTED_TAG}"
    return
  fi

  if [ "${DEFAULT_TO_LATEST}" != true ]; then
    die "--version <tag> is required; pass --latest to opt into automatic latest selection"
  fi

  tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases?per_page=1" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  [ -n "${tag}" ] || die "unable to resolve latest release tag"
  printf '%s\n' "${tag}"
}

download_and_verify_checksum() {
  local url="$1"
  local target="$2"
  local expected_sha="$3"
  local actual_sha

  curl -fsSL -o "${target}" "${url}"
  actual_sha="$(compute_sha256 "${target}")"
  [ "${actual_sha}" = "${expected_sha}" ] || die "checksum mismatch for downloaded helper ${target}"
}

download_asset() {
  local tag="$1"
  local asset="$2"
  curl -fsSL -o "${asset}" "https://github.com/${REPO}/releases/download/${tag}/${asset}"
}

ensure_cosign() {
  if command -v cosign >/dev/null 2>&1; then
    COSIGN_BIN="$(command -v cosign)"
    cosign_version="$(${COSIGN_BIN} version 2>/dev/null | awk '/GitVersion:/ {print $2; exit}')"
    if [ "${cosign_version}" = "${COSIGN_VERSION}" ]; then
      COSIGN_SOURCE="system (${cosign_version})"
      return
    fi
  fi

  local cosign_os cosign_arch expected_sha
  case "${OS}" in
    linux) cosign_os="linux" ;;
    darwin) cosign_os="darwin" ;;
    *) die "unsupported OS for cosign: ${OS}" ;;
  esac
  case "${ARCH}" in
    amd64) cosign_arch="amd64" ;;
    arm64) cosign_arch="arm64" ;;
    *) die "unsupported architecture for cosign: ${ARCH}" ;;
  esac

  COSIGN_BIN="${TMP_DIR}/cosign"
  download_asset_url="https://github.com/sigstore/cosign/releases/download/${COSIGN_VERSION}/cosign-${cosign_os}-${cosign_arch}"
  case "${OS}-${ARCH}" in
    linux-amd64) expected_sha="${COSIGN_DIGEST_LINUX_AMD64}" ;;
    linux-arm64) expected_sha="${COSIGN_DIGEST_LINUX_ARM64}" ;;
    darwin-amd64) expected_sha="${COSIGN_DIGEST_DARWIN_AMD64}" ;;
    darwin-arm64) expected_sha="${COSIGN_DIGEST_DARWIN_ARM64}" ;;
    *) die "unsupported OS/architecture for cosign: ${OS}/${ARCH}" ;;
  esac
  download_and_verify_checksum "${download_asset_url}" "${COSIGN_BIN}" "${expected_sha}"
  chmod +x "${COSIGN_BIN}"
  COSIGN_SOURCE="temporary (${COSIGN_VERSION})"
}

ensure_gh() {
  if command -v gh >/dev/null 2>&1; then
    GH_BIN="$(command -v gh)"
    gh_version="$(${GH_BIN} --version 2>/dev/null | awk 'NR==1 {print $3; exit}')"
    if [ "${gh_version}" = "${GH_VERSION#v}" ]; then
      GH_SOURCE="system (v${gh_version})"
      return
    fi
  fi

  local gh_os gh_arch gh_asset gh_root expected_sha
  case "${OS}" in
    linux) gh_os="linux" ;;
    darwin) gh_os="macOS" ;;
    *) die "unsupported OS for gh: ${OS}" ;;
  esac
  case "${ARCH}" in
    amd64) gh_arch="x86_64" ;;
    arm64) gh_arch="arm64" ;;
    *) die "unsupported architecture for gh: ${ARCH}" ;;
  esac

  case "${OS}-${ARCH}" in
    linux-amd64) expected_sha="${GH_DIGEST_LINUX_AMD64}" ;;
    linux-arm64) expected_sha="${GH_DIGEST_LINUX_ARM64}" ;;
    darwin-amd64) expected_sha="${GH_DIGEST_DARWIN_AMD64}" ;;
    darwin-arm64) expected_sha="${GH_DIGEST_DARWIN_ARM64}" ;;
    *) die "unsupported OS/architecture for gh: ${OS}/${ARCH}" ;;
  esac

  if [ "${OS}" = "darwin" ]; then
    gh_asset="gh_${GH_VERSION#v}_${gh_os}_${gh_arch}.zip"
    download_and_verify_checksum "https://github.com/cli/cli/releases/download/${GH_VERSION}/${gh_asset}" "${TMP_DIR}/${gh_asset}" "${expected_sha}"
    unzip -q "${TMP_DIR}/${gh_asset}" -d "${TMP_DIR}"
  else
    gh_asset="gh_${GH_VERSION#v}_${gh_os}_${gh_arch}.tar.gz"
    download_and_verify_checksum "https://github.com/cli/cli/releases/download/${GH_VERSION}/${gh_asset}" "${TMP_DIR}/${gh_asset}" "${expected_sha}"
    tar -xzf "${TMP_DIR}/${gh_asset}" -C "${TMP_DIR}"
  fi
  gh_root="${TMP_DIR}/gh_${GH_VERSION#v}_${gh_os}_${gh_arch}"
  GH_BIN="${gh_root}/bin/gh"
  [ -x "${GH_BIN}" ] || die "downloaded gh binary not found"
  GH_SOURCE="temporary (${GH_VERSION})"
}

verify_blob() {
  local file="$1"
  "${COSIGN_BIN}" verify-blob \
    --certificate-identity "${WORKFLOW_IDENTITY_PREFIX}${TAG}" \
    --certificate-oidc-issuer "${OIDC_ISSUER}" \
    --signature "${file}.sig" \
    --certificate "${file}.pem" \
    "${file}" >/dev/null
}

compute_sha256() {
  local file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${file}" | awk '{print $1}'
  else
    shasum -a 256 "${file}" | awk '{print $1}'
  fi
}

REQUESTED_TAG=""
LOCAL_SCRIPT_PATH="${BASH_SOURCE[0]}"
case "${LOCAL_SCRIPT_PATH}" in
  /*) ;;
  *) LOCAL_SCRIPT_PATH="$(pwd -P)/${LOCAL_SCRIPT_PATH}" ;;
esac

while [ "$#" -gt 0 ]; do
  case "$1" in
    --tag|--version)
      [ "$#" -ge 2 ] || die "$1 requires a value"
      REQUESTED_TAG="$2"
      shift 2
      ;;
    --latest)
      DEFAULT_TO_LATEST=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown argument: $1"
      ;;
  esac
done

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "${OS}" in
  linux|darwin) ;;
  *) die "unsupported operating system: ${OS}" ;;
esac

ARCH_RAW="$(uname -m)"
case "${ARCH_RAW}" in
  x86_64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) die "unsupported architecture: ${ARCH_RAW}" ;;
esac

TAG="$(resolve_tag)"
ARCHIVE="runecode_${TAG}_${OS}_${ARCH}.tar.gz"
INSTALLER_NAME="install-runecode.sh"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

ensure_cosign
ensure_gh

printf 'RuneCode installer verification\n'
printf '  repo: %s\n' "${REPO}"
printf '  tag: %s\n' "${TAG}"
printf '  archive: %s\n' "${ARCHIVE}"
printf '  expected workflow identity: %s%s\n' "${WORKFLOW_IDENTITY_PREFIX}" "${TAG}"
printf '  expected OIDC issuer: %s\n' "${OIDC_ISSUER}"
printf '  cosign: %s (%s)\n' "${COSIGN_BIN}" "${COSIGN_SOURCE}"
printf '  gh: %s (%s)\n' "${GH_BIN}" "${GH_SOURCE}"

cd "${TMP_DIR}"
for asset in "${ARCHIVE}" "${ARCHIVE}.sig" "${ARCHIVE}.pem" "${INSTALLER_NAME}.sig" "${INSTALLER_NAME}.pem" "SHA256SUMS" "SHA256SUMS.sig" "SHA256SUMS.pem"; do
  download_asset "${TAG}" "${asset}"
done

verify_blob "SHA256SUMS"
"${COSIGN_BIN}" verify-blob \
  --certificate-identity "${WORKFLOW_IDENTITY_PREFIX}${TAG}" \
  --certificate-oidc-issuer "${OIDC_ISSUER}" \
  --signature "${INSTALLER_NAME}.sig" \
  --certificate "${INSTALLER_NAME}.pem" \
  "${LOCAL_SCRIPT_PATH}" >/dev/null
"${GH_BIN}" attestation verify "${LOCAL_SCRIPT_PATH}" --repo "${REPO}" >/dev/null
verify_blob "${ARCHIVE}"

checksum_line="$(grep -F "  ${ARCHIVE}" SHA256SUMS || true)"
[ -n "${checksum_line}" ] || die "SHA256SUMS missing entry for ${ARCHIVE}"
expected_archive_sha="$(printf '%s\n' "${checksum_line}" | awk '{print $1}')"
actual_archive_sha="$(compute_sha256 "${ARCHIVE}")"
[ "${expected_archive_sha}" = "${actual_archive_sha}" ] || die "checksum mismatch for ${ARCHIVE}"

script_sum_entry="$(grep -F "  ${INSTALLER_NAME}" SHA256SUMS || true)"
[ -n "${script_sum_entry}" ] || die "SHA256SUMS missing entry for ${INSTALLER_NAME}"
expected_script_sha="$(printf '%s\n' "${script_sum_entry}" | awk '{print $1}')"
local_script_sha="$(compute_sha256 "${LOCAL_SCRIPT_PATH}")"
[ "${expected_script_sha}" = "${local_script_sha}" ] || die "checksum mismatch for ${INSTALLER_NAME}"

"${GH_BIN}" attestation verify "${ARCHIVE}" --repo "${REPO}" >/dev/null

printf '  SHA256SUMS entry for %s: %s\n' "${INSTALLER_NAME}" "${script_sum_entry}"
printf '  local running installer checksum: %s  %s\n' "${local_script_sha}" "${LOCAL_SCRIPT_PATH}"

printf '\nVerification summary:\n'
printf '  [PASS] Running installer signature verified\n'
printf '  [PASS] Running installer attestation verified\n'
printf '  [PASS] Running installer checksum matched SHA256SUMS\n'
printf '  [PASS] Signed SHA256SUMS verified\n'
printf '  [PASS] Signed archive verified\n'
printf '  [PASS] Archive checksum matched SHA256SUMS\n'
printf '  [PASS] GitHub attestation verified\n'

printf '\nInstall RuneCode binaries to %s/.local/bin ? Type yes to continue: ' "${HOME}"
read -r answer
if [ "${answer}" != "yes" ]; then
  printf 'Installation aborted.\n'
  exit 1
fi

mkdir -p unpack
tar -xzf "${ARCHIVE}" -C unpack
pkg_dir="unpack/runecode_${TAG}_${OS}_${ARCH}"
[ -d "${pkg_dir}" ] || die "expected package directory not found: ${pkg_dir}"

install -d "${HOME}/.local/bin"
install -m 0755 "${pkg_dir}"/bin/runecode* "${HOME}/.local/bin/"
printf 'Installed RuneCode binaries to %s/.local/bin\n' "${HOME}"
printf 'Add that directory to PATH if it is not already present.\n'
