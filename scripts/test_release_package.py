#!/usr/bin/env python3
"""Extract a platform package and verify its launchers and CLI."""

import argparse
import json
import plistlib
import subprocess
import sys
import tempfile
import zipfile
from pathlib import Path


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("rid", choices=("win-x64", "osx-arm64", "osx-x64", "linux-x64"))
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    version = (root / "VERSION").read_text(encoding="utf-8").strip()
    archive = root / "dist" / f"xbde-save-editor-v{version}-{args.rid}.zip"
    with tempfile.TemporaryDirectory(prefix="xbde-package-test-") as directory:
        target = Path(directory)
        if args.rid.startswith("osx-"):
            subprocess.run(["ditto", "-x", "-k", str(archive), str(target)], check=True)
            app = target / "XBDE Save Editor.app"
            with (app / "Contents/Info.plist").open("rb") as source:
                info = plistlib.load(source)
            assert info["CFBundleShortVersionString"] == version
            gui = app / "Contents/MacOS" / info["CFBundleExecutable"]
            assert gui.is_file()
            if (root / "assets/app-icon.png").is_file():
                assert info["CFBundleIconFile"] == "AppIcon.icns"
                assert (app / "Contents/Resources/AppIcon.icns").stat().st_size > 0
            subprocess.run(["codesign", "--verify", "--deep", "--strict", str(app)], check=True)
            cli = target / "Cli/XbdeEditor.Cli"
        else:
            with zipfile.ZipFile(archive) as source:
                source.extractall(target)
                for member in source.infolist():
                    path = target / member.filename
                    if path.is_file() and args.rid != "win-x64":
                        path.chmod(member.external_attr >> 16)
            suffix = ".exe" if args.rid == "win-x64" else ""
            gui = target / ("XBDE Save Editor.exe" if suffix else "XbdeEditor.Gui")
            assert gui.is_file()
            cli = target / f"XbdeEditor.Cli{suffix}"
        if list(target.rglob("*.dll")) or list(target.rglob("*.runtimeconfig.json")):
            raise ValueError("The package contains obsolete runtime files")
        assert (target / "licenses/MyGo.txt").is_file()
        result = subprocess.run([str(cli), "--version"], capture_output=True, text=True, check=True)
        assert result.stdout.strip() == version
        assert not list(target.rglob("*.pdb"))
        subprocess.run([sys.executable, str(root / "scripts/test_cli.py"), "--cli", str(cli)], check=True)
        with tempfile.TemporaryFile() as log:
            process = subprocess.Popen([str(gui)], stdout=log, stderr=log)
            try:
                try:
                    code = process.wait(timeout=3)
                except subprocess.TimeoutExpired:
                    pass
                else:
                    log.seek(0)
                    raise ValueError(f"GUI exited early ({code}): {log.read().decode(errors='replace')[-2000:]}")
            finally:
                if process.poll() is None:
                    process.terminate()
                    try:
                        process.wait(timeout=10)
                    except subprocess.TimeoutExpired:
                        process.kill()
                        process.wait(timeout=10)
    print("Release package tests passed.")


if __name__ == "__main__":
    main()
