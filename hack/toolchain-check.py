"""Assert that the toolchain is declared once.

Every package in `devbox.json` names a version: `latest` is refused. A
manifest that says "latest" means two checkouts of the same commit build with
different tools (the repository contract's rule 3).
"""

from __future__ import annotations

import json
import pathlib
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent


def fail(message: str) -> None:
    print(f"TOOLCHAIN: {message}", file=sys.stderr)


def main() -> int:
    """Check every claim this repository makes about its toolchain."""
    problems = 0
    devbox = json.loads((ROOT / "devbox.json").read_text(encoding="utf-8"))
    packages = devbox["packages"]

    # Every tool names a version (the component contract's C6). "latest"
    # means two checkouts of the same commit build with different tools,
    # and the lock only hides that until the next `devbox update` moves
    # every one of them at once, in a diff nobody asked for.
    for name, entry in packages.items():
        version = entry.get("version") if isinstance(entry, dict) else entry
        if not isinstance(version, str) or not version or version == "latest":
            fail(f"devbox.json: {name} must name a version, not {version!r}")
            problems += 1

    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
