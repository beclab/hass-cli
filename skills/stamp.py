#!/usr/bin/env python3

"""Write the release being built into every skill's frontmatter.

The suite is compiled into the hass-cli binary, so the release it ships in is
a fact about the build, not something to keep by hand in two dozen files.
What is committed is a placeholder; GoReleaser runs this before compiling.

Standard library only: it runs in a release job that installs nothing else.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
FENCE = "---"
VERSION_KEY = "version:"
RELEASE_VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")
# What is committed, and what a snapshot build embeds. It names no release
# anybody could mistake for one.
PLACEHOLDER_VERSION = "0.0.0"


class StampError(Exception):
    pass


def stamp_file(skill: Path, version: str) -> None:
    """Rewrite the version inside one SKILL.md's frontmatter, never its body."""
    lines = skill.read_text(encoding="utf-8").split("\n")
    if not lines or lines[0].strip() != FENCE:
        raise StampError(f"{skill}: does not open with a {FENCE!r} frontmatter fence")
    for index, line in enumerate(lines[1:], start=1):
        if line.strip() == FENCE:
            raise StampError(f"{skill}: no {VERSION_KEY} line in the frontmatter")
        if line.startswith(VERSION_KEY):
            lines[index] = f'{VERSION_KEY} "{version}"'
            skill.write_text("\n".join(lines), encoding="utf-8")
            return
    raise StampError(f"{skill}: frontmatter fence is never closed")


def stamp(root: Path, version: str) -> list[Path]:
    """Stamp every skill under root and return the files written."""
    version = version.removeprefix("v")
    if not RELEASE_VERSION_RE.match(version):
        raise StampError(f"{version!r} is not a hass-cli release (x.y.z)")
    skills = sorted(root.glob("ha-*/SKILL.md"))
    if not skills:
        raise StampError(f"no skills under {root}")
    for skill in skills:
        stamp_file(skill, version)
    return skills


def main(argv: list[str]) -> int:
    if len(argv) != 2:
        print(f"usage: {Path(argv[0]).name} <x.y.z>", file=sys.stderr)
        return 2
    try:
        stamped = stamp(ROOT, argv[1])
    except StampError as error:
        print(f"stamp failed: {error}", file=sys.stderr)
        return 1
    print(f"stamped {len(stamped)} skills at {argv[1].removeprefix('v')}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
