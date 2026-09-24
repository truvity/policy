"""Assert that the toolchain is declared once and resolves to what it declares.

Node is the case that needs this, and it needs it because the failure is
silent. `devbox.json` names a yarn; `ts/package.json` names one in
`packageManager`; and a third yarn — whichever corepack decides to fetch —
is the one that actually runs unless the node plugin is disabled. All three
can disagree while every command succeeds.

That is not hypothetical. Before the plugin was disabled here, devbox
installed yarn 4.14.1 into the nix profile, put node's corepack shims ahead
of it on PATH, and ran 4.18.0 at the repository root — with the declared
4.14.1 sitting unused a directory away.
"""

from __future__ import annotations

import json
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent


def fail(message: str) -> None:
    print(f"TOOLCHAIN: {message}", file=sys.stderr)


def main() -> int:
    """Check every claim this repository makes about its Node toolchain."""
    problems = 0
    devbox = json.loads((ROOT / "devbox.json").read_text(encoding="utf-8"))
    packages = devbox["packages"]

    # The node plugin's corepack shims take precedence over everything
    # devbox installed, so with it enabled the yarn pin below is decoration.
    node = packages.get("nodejs")
    if not isinstance(node, dict) or not node.get("disable_plugin"):
        fail(
            "devbox.json: nodejs must set disable_plugin — its corepack shims "
            "shadow the yarn this file pins, and download their own instead",
        )
        problems += 1

    declared = packages.get("yarn-berry")
    if not isinstance(declared, str):
        fail("devbox.json: yarn-berry must name a version")
        problems += 1
        declared = None

    manifest = json.loads((ROOT / "ts" / "package.json").read_text(encoding="utf-8"))
    stated = manifest.get("packageManager", "")
    match = re.fullmatch(r"yarn@(\d+\.\d+\.\d+)", stated)
    if not match:
        fail(f"ts/package.json: packageManager is {stated!r}, expected yarn@<version>")
        problems += 1
    elif declared and match.group(1) != declared:
        # Whoever is not using devbox reads packageManager, so both have to
        # say the same thing or two people build with two compilers.
        fail(
            f"devbox.json pins yarn-berry {declared} and ts/package.json "
            f"names {stated} — they must agree",
        )
        problems += 1

    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
