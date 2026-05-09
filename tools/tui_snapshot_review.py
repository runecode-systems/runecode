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
REPO_SNAPSHOT_DIR_NAME = ".tui-snapshots"


def normalize_string_list(value: object) -> List[str]:
    if not isinstance(value, list):
        return []
    normalized: List[str] = []
    for item in value:
        if isinstance(item, str) and item:
            normalized.append(item)
    return normalized


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
    for trusted_root in trusted_output_roots():
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


def repo_snapshot_root() -> str:
    script_dir = os.path.dirname(os.path.realpath(__file__))
    repo_root = os.path.realpath(os.path.join(script_dir, os.pardir))
    return os.path.join(repo_root, REPO_SNAPSHOT_DIR_NAME)


def trusted_output_roots() -> List[str]:
    roots = trusted_temp_roots()
    repo_root = os.path.realpath(repo_snapshot_root())
    if repo_root not in roots:
        roots.append(repo_root)
    return roots


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
        if manifest.get("bundle"):
            print(f"Bundle: {manifest['bundle']}")
        if manifest.get("viewport"):
            print(f"Viewport: {manifest['viewport']}")
        if manifest.get("theme"):
            print(f"Theme: {manifest['theme']}")
        coverage = manifest.get("coverage", {})
        if isinstance(coverage, dict):
            if coverage.get("routes"):
                print(f"Coverage routes: {', '.join(coverage['routes'])}")
            if coverage.get("viewports"):
                print(f"Coverage viewports: {', '.join(coverage['viewports'])}")
            if coverage.get("scenarios"):
                print(f"Coverage scenarios: {', '.join(coverage['scenarios'])}")
    print(f"Scenarios: {scenario_count}")
    print(f"Review artifacts: {len(entries)}")
    coverage_summary = bundle_coverage_summary(manifest)
    if coverage_summary:
        print(f"Coverage summary: {coverage_summary}")
    manifest_scenarios = set()
    selected_scenarios = set()
    if isinstance(manifest, dict):
        coverage = manifest.get("coverage", {})
        if isinstance(coverage, dict):
            manifest_scenarios = {str(name) for name in coverage.get("scenarios", []) if isinstance(name, str)}
    for entry in entries:
        selected_scenarios.add(entry["name"])
        details = [entry["name"]]
        if entry.get("viewport"):
            details.append(f"viewport={entry['viewport']}")
        if entry.get("route"):
            details.append(f"route={entry['route']}")
        details.append(f"artifact={entry['artifact_kind']}")
        details.append(entry["artifact_path"])
        print("- " + " | ".join(details))
    missing = sorted(manifest_scenarios - selected_scenarios)
    if missing:
        print("Missing review artifacts: " + ", ".join(missing))


def suggested_preserved_bundle_command(output_dir: str) -> str:
    return f"TUI_SNAPSHOT_KEEP=1 TUI_SNAPSHOT_DIR={output_dir} just tui-snapshot-audit-full"


def print_missing_manifest_guidance(output_dir: str, manifest_path: str) -> None:
    print(f"No preserved snapshot manifest was found at {manifest_path}.", file=sys.stderr)
    print("tui_snapshot_review.py only reviews existing artifacts; it does not generate snapshots.", file=sys.stderr)
    print("If write-producing snapshot commands are allowed, generate preserved artifacts with:", file=sys.stderr)
    print(f"  {suggested_preserved_bundle_command(output_dir)}", file=sys.stderr)
    print("If you are in plan/read-only mode, provide an existing TUI_SNAPSHOT_DIR containing manifest.json.", file=sys.stderr)


def print_missing_artifact_guidance(output_dir: str, manifest_path: str) -> None:
    print(f"No snapshot review artifacts found in {manifest_path}", file=sys.stderr)
    print("The manifest does not currently reference any readable PNG or SVG review artifacts.", file=sys.stderr)
    print("This blocks snapshot review; do not treat missing artifacts as a TUI audit finding.", file=sys.stderr)
    print("If write-producing snapshot commands are allowed, regenerate preserved artifacts with the matching audit recipe, for example:", file=sys.stderr)
    print(f"  {suggested_preserved_bundle_command(output_dir)}", file=sys.stderr)
    print(f"Otherwise, point the review tool at a different preserved temp directory than {output_dir}.", file=sys.stderr)


def manifest_coverage(manifest: object) -> Dict[str, object]:
    if not isinstance(manifest, dict):
        return {}
    coverage = manifest.get("coverage", {})
    if not isinstance(coverage, dict):
        return {}
    return coverage


def manifest_bundle_name(manifest: object) -> str:
    coverage = manifest_coverage(manifest)
    bundle = coverage.get("bundle")
    if isinstance(bundle, str) and bundle:
        return bundle
    if isinstance(manifest, dict):
        top_level_bundle = manifest.get("bundle")
        if isinstance(top_level_bundle, str):
            return top_level_bundle
    return ""


def manifest_viewports(manifest: object) -> List[str]:
    coverage = manifest_coverage(manifest)
    viewports = normalize_string_list(coverage.get("viewports"))
    if viewports:
        return viewports
    if isinstance(manifest, dict):
        viewport = manifest.get("viewport")
        if isinstance(viewport, str) and viewport:
            return [viewport]
    return []


def manifest_routes(manifest: object) -> List[str]:
    coverage = manifest_coverage(manifest)
    routes = normalize_string_list(coverage.get("routes"))
    if routes:
        return routes
    if not isinstance(manifest, dict):
        return []
    scenario_routes: List[str] = []
    for entry in manifest.get("scenarios", []):
        if not isinstance(entry, dict):
            continue
        route = entry.get("route")
        if isinstance(route, str) and route and route not in scenario_routes:
            scenario_routes.append(route)
    return scenario_routes


def validate_manifest_requirements(manifest: object, require_bundle: str, require_routes: List[str], require_viewport: str) -> List[str]:
    errors: List[str] = []
    bundle = manifest_bundle_name(manifest)
    if require_bundle and bundle != require_bundle:
        found_bundle = bundle or "<none>"
        errors.append(f"Required bundle {require_bundle!r} was not found in manifest coverage (found {found_bundle!r}).")

    available_viewports = manifest_viewports(manifest)
    if require_viewport and require_viewport not in available_viewports:
        found_viewports = ", ".join(available_viewports) if available_viewports else "<none>"
        errors.append(f"Required viewport {require_viewport!r} was not found in manifest coverage (found {found_viewports}).")

    available_routes = manifest_routes(manifest)
    missing_routes = [route for route in require_routes if route not in available_routes]
    if missing_routes:
        found_routes = ", ".join(available_routes) if available_routes else "<none>"
        errors.append(f"Required routes missing from manifest coverage: {', '.join(missing_routes)} (found {found_routes}).")

    return errors


def print_requirement_failure(output_dir: str, errors: List[str]) -> None:
    for err in errors:
        print(err, file=sys.stderr)
    print("This blocks preserved-artifact review; no audit coverage can be claimed for the requested scope.", file=sys.stderr)
    print("Point the review tool at a preserved temp directory containing the requested scope, or regenerate it in write-capable mode with the matching audit recipe.", file=sys.stderr)
    print(f"Example write-capable command: {suggested_preserved_bundle_command(output_dir)}", file=sys.stderr)


def bundle_coverage_summary(manifest: object) -> str:
    if not isinstance(manifest, dict):
        return ""
    coverage = manifest.get("coverage", {})
    if not isinstance(coverage, dict):
        return ""
    routes = coverage.get("routes", [])
    scenarios = coverage.get("scenarios", [])
    viewports = coverage.get("viewports", [])
    if not isinstance(routes, list) or not isinstance(scenarios, list) or not isinstance(viewports, list):
        return ""
    return f"bundle={coverage.get('bundle', '')} routes={len(routes)} viewports={len(viewports)} scenarios={len(scenarios)}"


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
        default=repo_snapshot_root(),
        help="snapshot output directory under a trusted snapshot root",
    )
    parser.add_argument(
        "--mode",
        choices=REVIEW_MODES,
        default=REVIEW_MODE_SUMMARY,
        help="review mode: summary (default), list, or open",
    )
    parser.add_argument(
        "--require-bundle",
        default="",
        help="require a specific manifest bundle before review succeeds",
    )
    parser.add_argument(
        "--require-route",
        action="append",
        default=[],
        help="require a specific covered route before review succeeds; repeat for multiple routes",
    )
    parser.add_argument(
        "--require-viewport",
        default="",
        help="require a specific covered viewport before review succeeds",
    )
    return parser.parse_args(argv)


def main() -> int:
    args = parse_args(sys.argv[1:])
    requested_output_dir = args.output_dir
    output_dir = resolve_output_dir(requested_output_dir)
    if output_dir is None:
        print("Requested output directory must resolve inside a trusted snapshot root", file=sys.stderr)
        return 1

    manifest_path = os.path.join(output_dir, "manifest.json")
    try:
        with open(manifest_path, encoding="utf-8") as manifest_file:
            manifest = json.load(manifest_file)
    except FileNotFoundError as err:
        print(f"Failed to read snapshot manifest {manifest_path}: {err}", file=sys.stderr)
        print_missing_manifest_guidance(output_dir, manifest_path)
        return 1
    except OSError as err:
        print(f"Failed to read snapshot manifest {manifest_path}: {err}", file=sys.stderr)
        return 1
    except json.JSONDecodeError as err:
        print(f"Failed to parse snapshot manifest {manifest_path}: {err}", file=sys.stderr)
        return 1

    requirement_errors = validate_manifest_requirements(manifest, args.require_bundle, args.require_route, args.require_viewport)
    if requirement_errors:
        print_requirement_failure(output_dir, requirement_errors)
        return 1

    entries = select_review_entries_from_manifest(output_dir, manifest)
    if not entries:
        print_missing_artifact_guidance(output_dir, manifest_path)
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
