#!/usr/bin/env bash
set -euo pipefail

export CGO_ENABLED=0
export GOFLAGS="-trimpath -mod=vendor"
# 1980-01-01T00:00:00Z keeps tar and zip timestamps aligned.
export SOURCE_DATE_EPOCH=315532800
export TZ=UTC
export LC_ALL=C

mkdir -p release/payload release/dist

release_helper="$(pwd)/releasebuilder"
trap 'rm -f "${release_helper}"' EXIT

go build -o "${release_helper}" ./tools/releasebuilder

mapfile -t binaries < "@binariesFile@"
mapfile -t targets < "@targetsFile@"

for target in "${targets[@]}"; do
  [ -n "${target}" ] || continue
  read -r goos goarch archive_ext <<<"${target}"

  archive_base="@packageName@_@tag@_${goos}_${goarch}"
  package_dir="release/payload/${archive_base}"
  bin_dir="${package_dir}/bin"
  mkdir -p "${bin_dir}"

  for binary in "${binaries[@]}"; do
    output="${bin_dir}/${binary}"
    if [ "${goos}" = "windows" ]; then
      output="${output}.exe"
    fi

    GOOS="${goos}" GOARCH="${goarch}" go build -ldflags="-s -w" -o "${output}" "./cmd/${binary}"
  done

  cp LICENSE NOTICE README.md "${package_dir}/"

  case "${archive_ext}" in
    zip)
      "${release_helper}" zip --source "${package_dir}" --target "release/dist/${archive_base}.zip"
      ;;
    tar.gz)
      package_parent="$(dirname "${package_dir}")"
      package_name="$(basename "${package_dir}")"
      @gnutar@/bin/tar --format=gnu --sort=name --mtime='UTC 1980-01-01' --owner=0 --group=0 --numeric-owner -C "${package_parent}" -cf - "${package_name}" \
        | @gzip@/bin/gzip -n > "release/dist/${archive_base}.tar.gz"
      ;;
    *)
      printf 'unsupported archive format: %s\n' "${archive_ext}" >&2
      exit 1
      ;;
  esac
done

archive_checksums="release/archive-sha256sums"
(
  shopt -s nullglob
  cd release/dist
  archive_files=( *.tar.gz *.zip )
  if [ "${#archive_files[@]}" -eq 0 ]; then
    printf 'expected at least one archive in release/dist for checksum generation\n' >&2
    exit 1
  fi
  @coreutils@/bin/sha256sum "${archive_files[@]}" > "../archive-sha256sums"
)

"${release_helper}" manifest \
  --package-name "@packageName@" \
  --version "@version@" \
  --tag "@tag@" \
  --binaries-file "@binariesFile@" \
  --targets-file "@targetsFile@" \
  --checksums-file "${archive_checksums}" \
  --output "release/dist/@packageName@_@tag@_release-manifest.json"

(
  shopt -s nullglob
  cd release/dist
  release_files=( *.tar.gz *.zip *.json )
  if [ "${#release_files[@]}" -eq 0 ]; then
    printf 'expected release assets in release/dist for checksum generation\n' >&2
    exit 1
  fi
  @coreutils@/bin/sha256sum "${release_files[@]}" > SHA256SUMS
)
