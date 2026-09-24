"""Write the release version into the manifests that carry a placeholder.

The git tag is the sole version authority — release.md §1 — so `ts/
package.json` and `python/pyproject.toml` each hold a placeholder that never
moves, and this puts the real number in at release time.

In place, rather than through each ecosystem's own command. `yarn version`
refuses on a tag checkout: it looks for an ancestor among master/main to
resolve a deferred bump against, and a release runs on a detached tag where
there is none. Nothing about setting a field in a manifest needs to know
what a branch is.

Every edit asserts that it matched something. A substitution that matches
nothing succeeds silently, and the failure here would be a package published
at the placeholder version, which cannot be taken back.
"""

from __future__ import annotations

import json
import pathlib
import re
import sys

PLACEHOLDER = "0.0.0"


def stamp_json(path: pathlib.Path, version: str) -> None:
    """Set a package.json's version, preserving the file's own formatting."""
    text = path.read_text(encoding="utf-8")
    current = json.loads(text)["version"]
    if current != PLACEHOLDER:
        raise SystemExit(f"{path}: version is {current!r}, expected the placeholder {PLACEHOLDER!r}")
    out, n = re.subn(rf'"version":\s*"{re.escape(PLACEHOLDER)}"', f'"version": "{version}"', text, count=1)
    if n != 1:
        raise SystemExit(f"{path}: no version line to stamp")
    path.write_text(out, encoding="utf-8")
    print(f"{path}: {version}")


def stamp_toml(path: pathlib.Path, version: str) -> None:
    """Set a pyproject.toml's version."""
    text = path.read_text(encoding="utf-8")
    out, n = re.subn(rf'(?m)^version = "{re.escape(PLACEHOLDER)}"$', f'version = "{version}"', text, count=1)
    if n != 1:
        raise SystemExit(f"{path}: no placeholder version line to stamp")
    path.write_text(out, encoding="utf-8")
    print(f"{path}: {version}")


def check(root: pathlib.Path) -> int:
    """Assert every manifest still holds the placeholder this expects to replace.

    Run by `just lint`, so that a manifest whose version was edited by hand
    fails the gate on the pull request that did it. Without this the same
    mistake surfaces at the tag, as a package published at 0.0.0 — which
    cannot be taken back, and burns the version it should have had.
    """
    wrong = []
    for path, current in (
        (root / "ts" / "package.json", json.loads((root / "ts" / "package.json").read_text())["version"]),
        (root / "python" / "pyproject.toml", _toml_version(root / "python" / "pyproject.toml")),
    ):
        if current != PLACEHOLDER:
            wrong.append(f"    {path.relative_to(root)}: {current!r}, expected {PLACEHOLDER!r}")
    if wrong:
        print("the release stamps these, so they must hold the placeholder:", file=sys.stderr)
        print("\n".join(wrong), file=sys.stderr)
        return 1
    return 0


def _toml_version(path: pathlib.Path) -> str:
    match = re.search(r'(?m)^version = "(.*)"$', path.read_text(encoding="utf-8"))
    if not match:
        raise SystemExit(f"{path}: no version line at all")
    return match.group(1)


def main() -> int:
    """Stamp every manifest, or say which one could not be stamped."""
    if len(sys.argv) != 2:
        print("usage: stamp-version.py <version> | --check", file=sys.stderr)
        return 2

    root = pathlib.Path(__file__).resolve().parent.parent
    if sys.argv[1] == "--check":
        return check(root)

    version = sys.argv[1]
    if not re.fullmatch(r"\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?", version):
        print(f"{version!r} is not a version this stamps", file=sys.stderr)
        return 2

    stamp_json(root / "ts" / "package.json", version)
    stamp_toml(root / "python" / "pyproject.toml", version)
    return 0


if __name__ == "__main__":
    sys.exit(main())
