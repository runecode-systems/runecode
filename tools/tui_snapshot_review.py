#!/usr/bin/env python3

import argparse
import json
import ntpath
import os
import platform
import shutil
import subprocess
import sys
import tempfile
from typing import Dict, List, Optional
from urllib.parse import urlparse


REVIEW_MODE_LIST = "list"
REVIEW_MODE_SUMMARY = "summary"
REVIEW_MODE_OPEN = "open"
REVIEW_MODES = (REVIEW_MODE_LIST, REVIEW_MODE_SUMMARY, REVIEW_MODE_OPEN)


def is_windows_absolute_path(path: str) -> bool:
    return ntpath.isabs(path)


def is_uri_like_path(path: str) -> bool:
    if is_windows_absolute_path(path):
        return False

    parsed = urlparse(path)
    return bool(parsed.scheme or parsed.netloc)


def resolve_output_dir(output_dir: object) -> Optional[str]:
    if not isinstance(output_dir, str) or not output_dir or is_uri_like_path(output_dir):
        return None

    output_dir_real = os.path.realpath(output_dir)
    for trusted_root in trusted_temp_roots():
        try:
            if os.path.commonpath([trusted_root, output_dir_real]) == trusted_root:
                return output_dir_real
        except ValueError:
            continue

    return None


def trusted_temp_roots() -> List[str]:
    roots = [os.path.realpath(tempfile.gettempdir())]
    if os.path.isdir("/tmp"):
        roots.append(os.path.realpath("/tmp"))

    unique_roots: List[str] = []
    for root in roots:
        if root not in unique_roots:
            unique_roots.append(root)
    return unique_roots


def resolve_artifact_path(output_dir: str, artifact_path: object) -> Optional[str]:
    if not isinstance(artifact_path, str) or not artifact_path:
        return None

    if is_uri_like_path(artifact_path):
        return None

    if os.path.isabs(artifact_path) or is_windows_absolute_path(artifact_path):
        return None

    output_dir_real = resolve_output_dir(output_dir)
    if output_dir_real is None:
        return None

    normalized_artifact_path = os.path.normpath(artifact_path)
    candidate_path = os.path.join(output_dir_real, normalized_artifact_path)

    candidate_real = os.path.realpath(candidate_path)
    try:
        if os.path.commonpath([output_dir_real, candidate_real]) != output_dir_real:
            return None
    except ValueError:
        return None

    return candidate_real


def select_artifact_paths_from_manifest(output_dir: str, manifest: object) -> List[str]:
    output_dir_real = resolve_output_dir(output_dir)
    if output_dir_real is None:
        return []

    paths: List[str] = []
    if not isinstance(manifest, dict):
        return paths
    for entry in manifest.get("scenarios", []):
        if not isinstance(entry, dict):
            continue
        artifacts = entry.get("artifacts", {})
        if not isinstance(artifacts, dict):
            continue
        png_artifact = artifacts.get("png", {})
        if not isinstance(png_artifact, dict):
            png_artifact = {}
        svg_artifact = artifacts.get("svg", {})
        if not isinstance(svg_artifact, dict):
            svg_artifact = {}
        png_path = resolve_artifact_path(output_dir_real, png_artifact.get("path"))
        svg_path = resolve_artifact_path(output_dir_real, svg_artifact.get("path"))
        chosen_path = png_path if png_path and os.path.exists(png_path) else svg_path
        if chosen_path:
            paths.append(chosen_path)
    return paths


def select_artifact_paths(output_dir: str, manifest_path: str) -> List[str]:
    try:
        with open(manifest_path, encoding="utf-8") as manifest_file:
            manifest = json.load(manifest_file)
    except OSError:
        return []
    except json.JSONDecodeError:
        return []
    return select_artifact_paths_from_manifest(output_dir, manifest)


def select_review_entries_from_manifest(output_dir: str, manifest: object) -> List[Dict[str, str]]:
    output_dir_real = resolve_output_dir(output_dir)
    if output_dir_real is None or not isinstance(manifest, dict):
        return []

    entries: List[Dict[str, str]] = []
    for entry in manifest.get("scenarios", []):
        if not isinstance(entry, dict):
            continue
        artifacts = entry.get("artifacts", {})
        if not isinstance(artifacts, dict):
            continue

        png_artifact = artifacts.get("png", {})
        if not isinstance(png_artifact, dict):
            png_artifact = {}
        svg_artifact = artifacts.get("svg", {})
        if not isinstance(svg_artifact, dict):
            svg_artifact = {}

        png_path = resolve_artifact_path(output_dir_real, png_artifact.get("path"))
        svg_path = resolve_artifact_path(output_dir_real, svg_artifact.get("path"))
        if png_path and os.path.exists(png_path):
            entries.append(
                {
                    "name": str(entry.get("name", "unnamed")),
                    "viewport": str(entry.get("viewport", manifest.get("viewport", "")) or ""),
                    "route": str(entry.get("route", "")),
                    "artifact_kind": "png",
                    "artifact_path": png_path,
                }
            )
            continue
        if svg_path and os.path.exists(svg_path):
            entries.append(
                {
                    "name": str(entry.get("name", "unnamed")),
                    "viewport": str(entry.get("viewport", manifest.get("viewport", "")) or ""),
                    "route": str(entry.get("route", "")),
                    "artifact_kind": "svg",
                    "artifact_path": svg_path,
                }
            )

    return entries


def detect_opener() -> Optional[str]:
    system = platform.system()
    if system == "Darwin":
        return shutil.which("open")
    if system == "Linux":
        return shutil.which("xdg-open")
    return None


def print_review_list(entries: List[Dict[str, str]]) -> None:
    for entry in entries:
        print(entry["artifact_path"])


def print_review_summary(output_dir: str, manifest_path: str, manifest: object, entries: List[Dict[str, str]]) -> None:
    scenario_count = len(manifest.get("scenarios", [])) if isinstance(manifest, dict) and isinstance(manifest.get("scenarios", []), list) else 0
    print(f"Output dir: {output_dir}")
    print(f"Manifest: {manifest_path}")
    if isinstance(manifest, dict):
        if manifest.get("viewport"):
            print(f"Viewport: {manifest['viewport']}")
        if manifest.get("theme"):
            print(f"Theme: {manifest['theme']}")
    print(f"Scenarios: {scenario_count}")
    print(f"Review artifacts: {len(entries)}")
    for entry in entries:
        details = [entry["name"]]
        if entry.get("viewport"):
            details.append(f"viewport={entry['viewport']}")
        if entry.get("route"):
            details.append(f"route={entry['route']}")
        details.append(f"artifact={entry['artifact_kind']}")
        details.append(entry["artifact_path"])
        print("- " + " | ".join(details))


def open_review_artifacts(entries: List[Dict[str, str]]) -> int:
    opener = detect_opener()
    if opener is None:
        print("No supported desktop opener found for snapshot review; selected artifact paths:", file=sys.stderr)
        print_review_list(entries)
        return 1

    failed_paths: List[str] = []
    for entry in entries:
        artifact_path = entry["artifact_path"]
        _, ext = os.path.splitext(artifact_path)
        if ext.lower() not in {".png", ".svg"} or not os.path.isfile(artifact_path):
            failed_paths.append(artifact_path)
            continue
        try:
            subprocess.run([opener, artifact_path], check=True)
        except OSError:
            failed_paths.append(artifact_path)
        except subprocess.CalledProcessError:
            failed_paths.append(artifact_path)

    for path in failed_paths:
        print(path)
    if failed_paths:
        return 1
    return 0


def parse_args(argv: List[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Review TUI snapshot artifacts without changing release behavior")
    parser.add_argument(
        "output_dir",
        nargs="?",
        default=os.path.join(tempfile.gettempdir(), "runecode-tui-snapshots"),
        help="snapshot output directory under the trusted temp root",
    )
    parser.add_argument(
        "--mode",
        choices=REVIEW_MODES,
        default=REVIEW_MODE_SUMMARY,
        help="review mode: summary (default), list, or open",
    )
    return parser.parse_args(argv)


def main() -> int:
    args = parse_args(sys.argv[1:])
    requested_output_dir = args.output_dir
    output_dir = resolve_output_dir(requested_output_dir)
    if output_dir is None:
        print("Requested output directory must resolve inside the system temp directory", file=sys.stderr)
        return 1

    manifest_path = os.path.join(output_dir, "manifest.json")
    try:
        with open(manifest_path, encoding="utf-8") as manifest_file:
            manifest = json.load(manifest_file)
    except OSError as err:
        print(f"Failed to read snapshot manifest {manifest_path}: {err}", file=sys.stderr)
        return 1
    except json.JSONDecodeError as err:
        print(f"Failed to parse snapshot manifest {manifest_path}: {err}", file=sys.stderr)
        return 1

    entries = select_review_entries_from_manifest(output_dir, manifest)
    if not entries:
        print(f"No snapshot review artifacts found in {manifest_path}", file=sys.stderr)
        return 1

    if args.mode == REVIEW_MODE_LIST:
        print_review_list(entries)
        return 0
    if args.mode == REVIEW_MODE_SUMMARY:
        print_review_summary(output_dir, manifest_path, manifest, entries)
        return 0
    return open_review_artifacts(entries)


if __name__ == "__main__":
    raise SystemExit(main())
