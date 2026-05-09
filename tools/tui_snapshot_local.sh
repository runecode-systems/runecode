#!/bin/sh

set -eu

usage() {
	printf '%s\n' 'usage: sh ./tools/tui_snapshot_local.sh --binary <path> --scenario <name> [--viewport <name>] [--review] [--review-mode <summary|list|open>]' >&2
}

binary_path=''
scenario=''
viewport=''
review='0'
review_mode='summary'

while [ "$#" -gt 0 ]; do
	case "$1" in
		--binary)
			[ "$#" -ge 2 ] || { usage; exit 2; }
			binary_path=$2
			shift 2
			;;
		--scenario)
			[ "$#" -ge 2 ] || { usage; exit 2; }
			scenario=$2
			shift 2
			;;
		--viewport)
			[ "$#" -ge 2 ] || { usage; exit 2; }
			viewport=$2
			shift 2
			;;
		--review)
			review='1'
			shift
			;;
		--review-mode)
			[ "$#" -ge 2 ] || { usage; exit 2; }
			review_mode=$2
			shift 2
			;;
		*)
			usage
			exit 2
			;;
	esac
done

[ -n "$binary_path" ] || { usage; exit 2; }
[ -n "$scenario" ] || { usage; exit 2; }

case "$review_mode" in
	summary|list|open) ;;
	*)
		usage
		exit 2
		;;
esac

requested_output_dir=${TUI_SNAPSHOT_DIR:-}
if [ -n "$requested_output_dir" ]; then
	output_dir=$(go run ./tools/tuisnapshotci local-output-dir --output-dir "$requested_output_dir")
else
	output_dir=$(go run ./tools/tuisnapshotci local-output-dir)
fi

cleanup_before() {
	go run ./tools/tuisnapshotci cleanup-local-dir --output-dir "$output_dir"
}

cleanup_after() {
	if [ "${TUI_SNAPSHOT_KEEP:-}" = '1' ]; then
		return 0
	fi
	cleanup_before
}

finish() {
	status=$1
	trap - EXIT HUP INT TERM
	if ! cleanup_after; then
		if [ "$status" -eq 0 ]; then
			exit 1
		fi
	fi
	exit "$status"
}

cleanup_before
trap 'finish $?' EXIT HUP INT TERM

mkdir -p "$output_dir"

if [ -n "$viewport" ]; then
	"$binary_path" --snapshot-scenario "$scenario" --snapshot-output-dir "$output_dir" --snapshot-viewport "$viewport"
else
	"$binary_path" --snapshot-scenario "$scenario" --snapshot-output-dir "$output_dir"
fi

if command -v magick >/dev/null 2>&1; then
	for svg in "$output_dir"/*.svg; do
		[ -e "$svg" ] || continue
		magick "$svg" "${svg%.svg}.png"
	done
elif command -v convert >/dev/null 2>&1; then
	for svg in "$output_dir"/*.svg; do
		[ -e "$svg" ] || continue
		convert "$svg" "${svg%.svg}.png"
	done
fi

if [ "$review" = '1' ]; then
	python3 ./tools/tui_snapshot_review.py --mode "$review_mode" "$output_dir"
fi
