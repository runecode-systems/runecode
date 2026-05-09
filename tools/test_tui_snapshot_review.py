import importlib.util
import io
import json
import tempfile
import unittest
from unittest import mock
from pathlib import Path


MODULE_PATH = Path(__file__).with_name("tui_snapshot_review.py")
SPEC = importlib.util.spec_from_file_location("tui_snapshot_review", MODULE_PATH)
assert SPEC is not None and SPEC.loader is not None
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class SelectArtifactPathsTest(unittest.TestCase):
    def require_symlink(self, path: Path, target: Path, *, target_is_directory: bool = False) -> None:
        try:
            path.symlink_to(target, target_is_directory=target_is_directory)
        except (NotImplementedError, OSError):
            self.skipTest("symlinks unsupported on this platform")

    def test_confines_paths_and_falls_back_to_confined_svg(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            temp_path = Path(temp_dir)
            output_dir = temp_path / "snapshots"
            output_dir.mkdir()

            outside_png = temp_path / "outside.png"
            outside_png.write_bytes(b"png")
            escaped_png = output_dir / "escaped.png"
            self.require_symlink(escaped_png, outside_png)

            fallback_svg = output_dir / "fallback.svg"
            fallback_svg.write_text("<svg />", encoding="utf-8")

            inside_png = output_dir / "inside.png"
            inside_png.write_bytes(b"png")

            manifest_path = output_dir / "manifest.json"
            manifest_path.write_text(
                json.dumps(
                    {
                        "scenarios": [
                            {
                                "artifacts": {
                                    "png": {"path": "escaped.png"},
                                    "svg": {"path": "fallback.svg"},
                                }
                            },
                            {
                                "artifacts": {
                                    "png": {"path": "file:///etc/passwd"},
                                    "svg": {"path": "../outside.svg"},
                                }
                            },
                            {
                                "artifacts": {
                                    "png": {"path": str(inside_png)},
                                    "svg": {"path": "../outside.svg"},
                                }
                            },
                            {
                                "artifacts": {
                                    "png": {"path": "inside.png"},
                                    "svg": {"path": "../outside.svg"},
                                }
                            },
                        ]
                    }
                ),
                encoding="utf-8",
            )

            paths = MODULE.select_artifact_paths(str(output_dir), str(manifest_path))

            self.assertEqual(paths, [str(fallback_svg.resolve()), str(inside_png.resolve())])
            self.assertEqual(MODULE.resolve_artifact_path(str(output_dir), "inside.png"), str(inside_png.resolve()))
            self.assertIsNone(MODULE.resolve_artifact_path(str(output_dir), str(inside_png)))

    def test_resolve_output_dir_rejects_symlink_outside_temp(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            temp_path = Path(temp_dir)
            outside_dir = Path(__file__).resolve().parent
            linked_output_dir = temp_path / "linked-output"
            self.require_symlink(linked_output_dir, outside_dir, target_is_directory=True)

            self.assertIsNone(MODULE.resolve_output_dir(str(linked_output_dir)))

    def test_resolve_output_dir_accepts_real_tmp_when_tempdir_differs(self) -> None:
        tmp_root = Path("/tmp")
        if not tmp_root.is_dir():
            self.skipTest("/tmp unavailable on this platform")

        requested = str(tmp_root / "runecode-tui-snapshots")
        with mock.patch.object(MODULE.tempfile, "gettempdir", return_value=str(Path(tempfile.gettempdir()) / "alternate-temp-root")):
            self.assertEqual(MODULE.resolve_output_dir(requested), str(Path(requested).resolve()))

    def test_windows_absolute_paths_are_not_treated_as_uris(self) -> None:
        self.assertFalse(MODULE.is_uri_like_path(r"C:\temp\snapshot.png"))
        self.assertTrue(MODULE.is_uri_like_path("file:///tmp/snapshot.png"))

    def test_select_artifact_paths_returns_empty_for_invalid_manifest(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            output_dir = Path(temp_dir)
            manifest_path = output_dir / "manifest.json"
            manifest_path.write_text("{not-json", encoding="utf-8")

            self.assertEqual(MODULE.select_artifact_paths(str(output_dir), str(manifest_path)), [])

    def test_select_artifact_paths_skips_malformed_manifest_entries(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            output_dir = Path(temp_dir)
            svg_path = output_dir / "fallback.svg"
            svg_path.write_text("<svg />", encoding="utf-8")
            manifest = {
                "scenarios": [
                    "not-an-object",
                    {"artifacts": "bad"},
                    {"artifacts": {"png": "bad", "svg": {"path": "fallback.svg"}}},
                ]
            }

            paths = MODULE.select_artifact_paths_from_manifest(str(output_dir), manifest)

            self.assertEqual(paths, [str(svg_path.resolve())])


class ReviewModesTest(unittest.TestCase):
    def create_snapshot_dir(self) -> tuple[Path, Path, Path]:
        temp_dir = Path(tempfile.mkdtemp())
        png_path = temp_dir / "dashboard.desktop.png"
        png_path.write_bytes(b"png")
        svg_path = temp_dir / "action-center.desktop.svg"
        svg_path.write_text("<svg />", encoding="utf-8")
        manifest_path = temp_dir / "manifest.json"
        manifest_path.write_text(
            json.dumps(
                {
                    "version": 2,
                    "viewport": "desktop",
                    "theme": "dark",
                    "scenarios": [
                        {
                            "name": "dashboard",
                            "viewport": "desktop",
                            "route": "dashboard",
                            "artifacts": {
                                "png": {"path": png_path.name},
                                "svg": {"path": "dashboard.desktop.svg"},
                            },
                        },
                        {
                            "name": "action-center",
                            "viewport": "desktop",
                            "route": "approvals",
                            "artifacts": {
                                "png": {"path": "missing.png"},
                                "svg": {"path": svg_path.name},
                            },
                        },
                    ],
                }
            ),
            encoding="utf-8",
        )
        self.addCleanup(lambda: __import__("shutil").rmtree(temp_dir, ignore_errors=True))
        return temp_dir, png_path, svg_path

    def run_main(self, *argv: str) -> tuple[int, str, str]:
        stdout = io.StringIO()
        stderr = io.StringIO()
        with mock.patch.object(MODULE.sys, "argv", [str(MODULE_PATH), *argv]), mock.patch("sys.stdout", stdout), mock.patch("sys.stderr", stderr):
            return MODULE.main(), stdout.getvalue(), stderr.getvalue()

    def test_default_mode_prints_summary_without_opening_gui(self) -> None:
        output_dir, png_path, svg_path = self.create_snapshot_dir()
        with mock.patch.object(MODULE, "detect_opener", return_value="xdg-open"), mock.patch.object(MODULE.subprocess, "run") as run_mock:
            exit_code, stdout, stderr = self.run_main(str(output_dir))

        self.assertEqual(exit_code, 0)
        self.assertEqual(stderr, "")
        self.assertIn(f"Output dir: {output_dir.resolve()}", stdout)
        self.assertIn("Review artifacts: 2", stdout)
        self.assertIn(f"- dashboard | viewport=desktop | route=dashboard | artifact=png | {png_path.resolve()}", stdout)
        self.assertIn(f"- action-center | viewport=desktop | route=approvals | artifact=svg | {svg_path.resolve()}", stdout)
        run_mock.assert_not_called()

    def test_list_mode_prints_selected_artifact_paths(self) -> None:
        output_dir, png_path, svg_path = self.create_snapshot_dir()
        with mock.patch.object(MODULE.subprocess, "run") as run_mock:
            exit_code, stdout, stderr = self.run_main("--mode", "list", str(output_dir))

        self.assertEqual(exit_code, 0)
        self.assertEqual(stderr, "")
        self.assertEqual(stdout.splitlines(), [str(png_path.resolve()), str(svg_path.resolve())])
        run_mock.assert_not_called()

    def test_open_mode_uses_desktop_opener(self) -> None:
        output_dir, png_path, svg_path = self.create_snapshot_dir()
        with mock.patch.object(MODULE, "detect_opener", return_value="xdg-open"), mock.patch.object(MODULE.subprocess, "run") as run_mock:
            exit_code, stdout, stderr = self.run_main("--mode", "open", str(output_dir))

        self.assertEqual(exit_code, 0)
        self.assertEqual(stdout, "")
        self.assertEqual(stderr, "")
        self.assertEqual(
            run_mock.call_args_list,
            [
                mock.call(["xdg-open", str(png_path.resolve())], check=True),
                mock.call(["xdg-open", str(svg_path.resolve())], check=True),
            ],
        )


if __name__ == "__main__":
    unittest.main()
