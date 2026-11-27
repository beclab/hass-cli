#!/usr/bin/env python3

"""Check the hass-cli skill suite against the writing rules in README.md.

Only the rules a machine can check live here; README.md says why each exists.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path
from urllib.parse import unquote

import yaml


ROOT = Path(__file__).resolve().parent
PREFIX = "ha-"
LINK_RE = re.compile(r"(?<!!)\[[^\]]+\]\(([^)]+)\)")
HEADING_RE = re.compile(r"^#{1,6}\s+(.+?)\s*$", re.MULTILINE)
HTML_ANCHOR_RE = re.compile(r'<a\s+(?:name|id)=["\']([^"\']+)["\']', re.IGNORECASE)
SKILL_MAX_LINES = 250
REFERENCE_MAX_LINES = 150
# What an agent reads before its first command is a path: the shared front
# door, the domain SKILL.md, and the references the task triggers. A skill
# states its common paths in a table; this is what one row may total.
READ_PATH_MAX_LINES = 250
FAST_PATHS_HEADINGS = ("Fast paths", "Symptom routing")
PROSE_SECTION_RE = re.compile(r'\b(?:especially|see(?:\s+also)?|section)\s+"([^"\n]{2,80})"')
SOURCE_CITATION_RE = re.compile(r"[\w.-]*(?:/[\w.-]+)*\w\.go\b(?::\d+)?")
TABLE_ROW_RE = re.compile(r"^\s*\|(?P<cells>.*)\|\s*$")
BACKTICKED_RE = re.compile(r"`[^`]+`")
# A skill's version is the hass-cli release it ships in (the tag without its
# `v`). The suite is compiled into the binary, so a number of its own would be
# a second thing to bump and a second thing to get wrong.
RELEASE_VERSION_RE = re.compile(r"\d+\.\d+\.\d+")
SHARED_SKILL = "ha-shared"
# ha-shared is the front door every path is measured through; it is not a
# destination.
NO_FAST_PATH = {SHARED_SKILL}
SAFETY_HEADING = "Safety and escalation"
VERB_INDEX_HEADING = "Verb index"
VERB_INDEX_LAST_COLUMN = "Read when triggered"
FRONT_DOOR_MARKER = "> **Shared front door:**"
# ha-shared routes the suite; the two playbooks are recipes over ha-automation
# and ha-system verbs, whose own skills index them.
NO_VERB_INDEX = {SHARED_SKILL, "ha-workflow-audit", "ha-workflow-automation-builder"}
# Facts that once went missing and cost a user something. Pinned so a rewrite
# cannot drop them without failing here.
REQUIRED_ENTRYPOINT_FACTS = {
    "ha-shared/SKILL.md": [
        (
            "a device a script or automation is driving is stopped there first",
            r"workflow script stop",
        ),
    ],
    "ha-automation/SKILL.md": [
        (
            "save takes the bare id; the entity is script.<id>",
            r"`script\.<id>`",
        ),
    ],
}
DESCRIPTION_LIMIT = 1024


def without_fenced_code(text: str) -> str:
    result: list[str] = []
    fence_char = ""
    fence_length = 0
    for line in text.splitlines(keepends=True):
        if not fence_char:
            opener = re.match(r"^ {0,3}(`{3,}|~{3,})", line)
            if opener:
                fence = opener.group(1)
                fence_char = fence[0]
                fence_length = len(fence)
                continue
            result.append(line)
            continue
        if re.fullmatch(rf" {{0,3}}{re.escape(fence_char)}{{{fence_length},}}\s*", line):
            fence_char = ""
            fence_length = 0
    return "".join(result)


def markdown_anchor(heading: str) -> str:
    heading = re.sub(r"<[^>]+>", "", heading)
    heading = re.sub(r"[^\w\s-]", "", heading, flags=re.UNICODE)
    return re.sub(r"\s", "-", heading.strip().lower())


def anchors(path: Path) -> set[str]:
    text = without_fenced_code(path.read_text(encoding="utf-8"))
    result: set[str] = set()
    counts: dict[str, int] = {}
    for heading in HEADING_RE.findall(text):
        base = markdown_anchor(heading)
        count = counts.get(base, 0)
        result.add(base if count == 0 else f"{base}-{count}")
        counts[base] = count + 1
    result.update(HTML_ANCHOR_RE.findall(text))
    return result


def validate_links(path: Path, errors: list[str]) -> None:
    text = without_fenced_code(path.read_text(encoding="utf-8"))
    for raw_target in LINK_RE.findall(text):
        target = raw_target.strip().strip("<>")
        if target.startswith(("http://", "https://", "mailto:")):
            continue
        file_part, separator, fragment = target.partition("#")
        target_path = path if not file_part else (path.parent / unquote(file_part)).resolve()
        if not target_path.exists():
            errors.append(f"{path.relative_to(ROOT)}: missing link target {target}")
            continue
        if separator and fragment and target_path.suffix.lower() == ".md":
            if unquote(fragment) not in anchors(target_path):
                errors.append(f"{path.relative_to(ROOT)}: missing anchor {target}")


def validate_prose_section_refs(path: Path, errors: list[str]) -> None:
    """A quoted heading is a link nothing can follow and no rename updates."""
    text = without_fenced_code(path.read_text(encoding="utf-8"))
    for name in PROSE_SECTION_RE.findall(text):
        errors.append(
            f"{path.relative_to(ROOT)}: names a section in prose ({name!r}); "
            "link it with an anchor so a rename fails the build"
        )


def validate_no_source_citations(path: Path, errors: list[str]) -> None:
    """Ground claims in the implementation; do not ship where you grounded them."""
    citation = re.search(SOURCE_CITATION_RE, path.read_text(encoding="utf-8"))
    if citation:
        errors.append(
            f"{path.relative_to(ROOT)}: Go source citation {citation.group(0)!r} belongs in "
            "verification, not the shipped skill"
        )


def table_rows(text: str) -> list[list[str]]:
    rows: list[list[str]] = []
    for line in text.splitlines():
        match = TABLE_ROW_RE.match(line)
        if not match:
            continue
        cells = [cell.strip() for cell in match.group("cells").split("|")]
        if all(set(cell) <= set("-: ") for cell in cells):
            continue
        rows.append(cells)
    return rows


def validate_verb_index_rows(skill: Path, errors: list[str]) -> None:
    """A row that only points at `--help` spends a line to say nothing."""
    text = without_fenced_code(skill.read_text(encoding="utf-8"))
    for cells in table_rows(text):
        cell = cells[-1]
        if "](" in cell or "--help" not in cell:
            continue
        spans = BACKTICKED_RE.findall(cell)
        if not spans or not all(span.rstrip("`").endswith("--help") for span in spans):
            continue
        residue = BACKTICKED_RE.sub("", cell)
        if set(residue) <= set(";,. \t/"):
            errors.append(
                f"{skill.relative_to(ROOT)}: verb-index row {cells[0]} points only at "
                f"{cell} — give the caller what --help will not say, or collapse the row"
            )


def line_count(path: Path) -> int:
    return len(path.read_text(encoding="utf-8").splitlines())


def validate_fast_paths(skill_dir: Path, errors: list[str]) -> None:
    """Hold a skill's declared read paths to the first-command budget.

    Each row totals the shared front door, this SKILL.md, and every file the
    row links. Required of every skill outside NO_FAST_PATH: a skill with no
    block is not under budget, it is unmeasured.
    """
    skill = skill_dir / "SKILL.md"
    text = without_fenced_code(skill.read_text(encoding="utf-8"))
    heading = "|".join(re.escape(name) for name in FAST_PATHS_HEADINGS)
    spellings = " or ".join(f"'## {name}'" for name in FAST_PATHS_HEADINGS)
    section = re.split(rf"^#{{1,6}}\s+(?:{heading})\s*$", text, flags=re.MULTILINE)
    if len(section) < 2:
        if skill_dir.name not in NO_FAST_PATH:
            errors.append(
                f"{skill.relative_to(ROOT)}: no {spellings} block — name the tasks an "
                f"agent can act on after one read, so the {READ_PATH_MAX_LINES}-line budget has "
                "something to measure"
            )
        return
    if skill_dir.name in NO_FAST_PATH:
        errors.append(
            f"{skill.relative_to(ROOT)}: declares fast paths but is listed in NO_FAST_PATH; "
            "remove it from that list"
        )
    if len(section) > 2:
        errors.append(
            f"{skill.relative_to(ROOT)}: has more than one of {spellings}; one table routes the "
            "agent, a second one competes with it"
        )
    body = re.split(r"^#{1,6}\s+", section[1], flags=re.MULTILINE)[0]
    if not table_rows(body):
        errors.append(f"{skill.relative_to(ROOT)}: its {spellings} block has no rows")
        return

    front_door = ROOT / SHARED_SKILL / "SKILL.md"
    base = line_count(skill)
    if skill_dir.name != SHARED_SKILL and front_door.exists():
        base += line_count(front_door)

    for cells in table_rows(body):
        total = base
        seen: set[Path] = set()
        for target in LINK_RE.findall(" ".join(cells)):
            link = unquote(target.strip().strip("<>").partition("#")[0])
            if not link or "://" in link:
                continue
            resolved = (skill.parent / link).resolve()
            if not resolved.exists() or resolved in seen or resolved == skill or resolved == front_door:
                continue
            seen.add(resolved)
            total += line_count(resolved)
        if total > READ_PATH_MAX_LINES:
            errors.append(
                f"{skill.relative_to(ROOT)}: fast path {cells[0]} reads {total} lines, over the "
                f"{READ_PATH_MAX_LINES}-line first-command budget — split what it links, or stop calling it fast"
            )


def validate_skill_entrypoint(skill_dir: Path, errors: list[str]) -> None:
    skill = skill_dir / "SKILL.md"
    text = without_fenced_code(skill.read_text(encoding="utf-8"))
    relative_skill = str(skill.relative_to(ROOT))
    for description, pattern in REQUIRED_ENTRYPOINT_FACTS.get(relative_skill, []):
        if not re.search(pattern, skill.read_text(encoding="utf-8"), re.MULTILINE):
            errors.append(f"{relative_skill}: missing entrypoint fact: {description}")

    linked_files = {
        unquote(raw_target.strip().strip("<>").partition("#")[0])
        for raw_target in LINK_RE.findall(text)
    }
    references = skill_dir / "references"
    for reference in sorted(references.glob("*.md")) if references.exists() else []:
        expected = f"references/{reference.name}"
        if expected not in linked_files:
            errors.append(f"{skill.relative_to(ROOT)}: reference is not one hop from SKILL.md: {expected}")


def validate_frontmatter(skill: Path, errors: list[str]) -> None:
    lines = skill.read_text(encoding="utf-8").splitlines()
    if not lines or lines[0] != "---":
        errors.append(f"{skill.relative_to(ROOT)}: frontmatter must start with ---")
        return
    try:
        end = lines.index("---", 1)
    except ValueError:
        errors.append(f"{skill.relative_to(ROOT)}: frontmatter is missing its closing ---")
        return

    class UniqueKeyLoader(yaml.SafeLoader):
        pass

    def construct_mapping(loader, node, deep=False):
        mapping = {}
        for key_node, value_node in node.value:
            key = loader.construct_object(key_node, deep=deep)
            if key in mapping:
                raise yaml.constructor.ConstructorError(
                    "while constructing a mapping",
                    node.start_mark,
                    f"found duplicate key {key!r}",
                    key_node.start_mark,
                )
            mapping[key] = loader.construct_object(value_node, deep=deep)
        return mapping

    UniqueKeyLoader.add_constructor(yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG, construct_mapping)
    try:
        root = yaml.load("\n".join(lines[1:end]), Loader=UniqueKeyLoader)
    except yaml.YAMLError as error:
        errors.append(f"{skill.relative_to(ROOT)}: invalid YAML frontmatter: {error}")
        return
    if not isinstance(root, dict):
        errors.append(f"{skill.relative_to(ROOT)}: frontmatter must be a mapping")
        return

    allowed_keys = {"name", "version", "description", "compatibility", "metadata"}
    unknown_keys = sorted(set(root) - allowed_keys)
    if unknown_keys:
        errors.append(f"{skill.relative_to(ROOT)}: unknown frontmatter keys: {', '.join(unknown_keys)}")

    expected_name = skill.parent.name
    if root.get("name") != expected_name:
        errors.append(f"{skill.relative_to(ROOT)}: frontmatter name must be {expected_name!r}")
    if not isinstance(root.get("version"), str) or not RELEASE_VERSION_RE.fullmatch(root["version"]):
        errors.append(
            f"{skill.relative_to(ROOT)}: frontmatter version must be a quoted hass-cli release, x.y.z"
        )
    for key in ("description", "compatibility"):
        if not isinstance(root.get(key), str) or not root[key].strip():
            errors.append(f"{skill.relative_to(ROOT)}: frontmatter {key!r} is required")

    description = root.get("description")
    if isinstance(description, str):
        if len(description) > DESCRIPTION_LIMIT:
            errors.append(
                f"{skill.relative_to(ROOT)}: frontmatter description is "
                f"{len(description)} characters, over the {DESCRIPTION_LIMIT} limit"
            )
        # `hass-cli skill list` reads one line per field, so a wrapped
        # description lists as its first line only.
        line = next((item for item in lines[1:end] if item.startswith("description:")), "")
        raw = line.partition(":")[2].strip()
        if yaml.safe_load(raw) != description:
            errors.append(f"{skill.relative_to(ROOT)}: frontmatter description must sit on one line")

    if root.get("metadata") != {"requires": {"bins": ["hass-cli"]}}:
        errors.append(f"{skill.relative_to(ROOT)}: frontmatter metadata must be requires.bins: [hass-cli]")


def validate_structure(skill_dir: Path, errors: list[str]) -> None:
    """Enforce the size limits and the ban on deep links into a peer's references.

    A SKILL.md may link ha-shared's references: every runtime skill loads that
    front door as a prerequisite. Any other peer is reached through its SKILL.md.
    """
    skill = skill_dir / "SKILL.md"
    references = sorted(skill_dir.glob("references/*.md"))

    for path, limit in [(skill, SKILL_MAX_LINES)] + [(ref, REFERENCE_MAX_LINES) for ref in references]:
        count = line_count(path)
        if count > limit:
            errors.append(f"{path.relative_to(ROOT)}: {count} lines exceeds the {limit}-line limit")

    for path in [skill] + references:
        for target in LINK_RE.findall(without_fenced_code(path.read_text(encoding="utf-8"))):
            link = unquote(target.split("#", 1)[0]).strip()
            if not link or "://" in link:
                continue
            try:
                rel = (path.parent / link).resolve().relative_to(ROOT)
            except ValueError:
                continue
            if len(rel.parts) < 2 or rel.parts[1] != "references" or rel.parts[0] == skill_dir.name:
                continue
            if path == skill and rel.parts[0] == SHARED_SKILL:
                continue
            errors.append(
                f"{path.relative_to(ROOT)}: deep-links {rel} — reach a peer skill through "
                "its SKILL.md, not by pointing into its references"
            )


def section_body(text: str, heading: str) -> str:
    parts = re.split(rf"^#{{1,6}}\s+{re.escape(heading)}\s*$", without_fenced_code(text), flags=re.MULTILINE)
    if len(parts) < 2:
        return ""
    return re.split(r"^#{1,6}\s+", parts[1], flags=re.MULTILINE)[0]


def validate_shared_sections(skill_dir: Path, errors: list[str]) -> None:
    """The same thing carries the same name in every skill."""
    skill = skill_dir / "SKILL.md"
    text = skill.read_text(encoding="utf-8")
    headings = set(HEADING_RE.findall(without_fenced_code(text)))
    relative = skill.relative_to(ROOT)

    if SAFETY_HEADING not in headings:
        errors.append(
            f"{relative}: no '## {SAFETY_HEADING}' section — every skill closes with what it "
            "must not do and when to hand back, under that name"
        )

    if skill_dir.name not in NO_VERB_INDEX:
        if VERB_INDEX_HEADING not in headings:
            errors.append(
                f"{relative}: no '## {VERB_INDEX_HEADING}' section — the command list goes under "
                "that name so an agent can find it without reading the file"
            )
        elif not any(
            cells[-1] == VERB_INDEX_LAST_COLUMN
            for cells in table_rows(section_body(text, VERB_INDEX_HEADING))
        ):
            errors.append(
                f"{relative}: its '## {VERB_INDEX_HEADING}' table has no "
                f"'{VERB_INDEX_LAST_COLUMN}' column — the first columns differ per tree, but the "
                "last one always answers what to read next"
            )

    if skill_dir.name != SHARED_SKILL and FRONT_DOOR_MARKER not in text:
        errors.append(
            f"{relative}: no '{FRONT_DOOR_MARKER}' block — an agent entering here has not loaded "
            "the shared control rules, and nothing tells it to"
        )


def read_frontmatter_version(skill: Path) -> str | None:
    for line in skill.read_text(encoding="utf-8").splitlines()[1:]:
        if line == "---":
            return None
        if line.startswith("version:"):
            return line.split(":", 1)[1].strip().strip('"')
    return None


def validate_one_version(skill_dirs: list[Path], errors: list[str]) -> None:
    """The suite ships inside one binary, so it carries one version."""
    declared: dict[str, list[str]] = {}
    for skill_dir in skill_dirs:
        version = read_frontmatter_version(skill_dir / "SKILL.md")
        if version is not None:
            declared.setdefault(version, []).append(skill_dir.name)
    if len(declared) > 1:
        spelled = "; ".join(
            f"{version}: {', '.join(sorted(skills))}" for version, skills in sorted(declared.items())
        )
        errors.append(f"the suite declares {len(declared)} versions but ships as one artifact — {spelled}")


def main() -> int:
    errors: list[str] = []
    skill_dirs = sorted(path.parent for path in ROOT.glob(f"{PREFIX}*/SKILL.md"))
    for path in sorted(ROOT.glob(f"{PREFIX}*/**/*.md")):
        validate_links(path, errors)
        validate_prose_section_refs(path, errors)
        validate_no_source_citations(path, errors)
    for skill_dir in skill_dirs:
        validate_frontmatter(skill_dir / "SKILL.md", errors)
        validate_skill_entrypoint(skill_dir, errors)
        validate_shared_sections(skill_dir, errors)
        validate_structure(skill_dir, errors)
        validate_verb_index_rows(skill_dir / "SKILL.md", errors)
        validate_fast_paths(skill_dir, errors)
    validate_one_version(skill_dirs, errors)

    if errors:
        print("Skill validation failed:", file=sys.stderr)
        for error in errors:
            print(f"  - {error}", file=sys.stderr)
        return 1

    print(f"Skill validation OK ({len(skill_dirs)} skills)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
