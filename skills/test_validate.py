import importlib.util
import tempfile
import unittest
from pathlib import Path


def load(name, file):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(file))
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


validate = load("skill_validate", "validate.py")
stamp = load("skill_stamp", "stamp.py")

VALID_FRONTMATTER = """---
name: ha-test
version: "0.0.7"
description: "Test skill"
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---
"""


class Rooted:
    """Point the validator at a temporary suite for one test."""

    def __init__(self):
        self.directory = tempfile.TemporaryDirectory()
        self.root = Path(self.directory.name).resolve()

    def __enter__(self):
        self.original = validate.ROOT
        validate.ROOT = self.root
        return self.root

    def __exit__(self, *exc):
        validate.ROOT = self.original
        self.directory.cleanup()


def write(path: Path, text: str) -> Path:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")
    return path


class FrontmatterTests(unittest.TestCase):
    def errors_for(self, text):
        with Rooted() as root:
            skill = write(root / "ha-test" / "SKILL.md", text)
            errors = []
            validate.validate_frontmatter(skill, errors)
            return errors

    def test_valid_frontmatter_passes(self):
        self.assertEqual(self.errors_for(VALID_FRONTMATTER), [])

    def test_version_must_name_a_release(self):
        for version, valid in [('"0.0.7"', True), ("0.0.7", True), ('"v0.0.7"', False), ('"0.3"', False)]:
            errors = self.errors_for(VALID_FRONTMATTER.replace('version: "0.0.7"', f"version: {version}"))
            self.assertEqual(not errors, valid, f"{version}: {errors}")

    def test_a_wrapped_description_is_refused(self):
        """`hass-cli skill list` reads one line per field."""
        errors = self.errors_for(
            VALID_FRONTMATTER.replace('description: "Test skill"', "description: >-\n  Test\n  skill")
        )
        self.assertTrue(any("one line" in error for error in errors), errors)

    def test_description_over_the_limit_is_refused(self):
        errors = self.errors_for(
            VALID_FRONTMATTER.replace("Test skill", "x" * (validate.DESCRIPTION_LIMIT + 1))
        )
        self.assertTrue(any("over the 1024 limit" in error for error in errors), errors)

    def test_metadata_must_gate_on_hass_cli(self):
        errors = self.errors_for(VALID_FRONTMATTER.replace('["hass-cli"]', '["olares-cli"]'))
        self.assertTrue(any("requires.bins" in error for error in errors), errors)

    def test_the_suite_declares_one_version(self):
        with Rooted() as root:
            for name, version in [("ha-a", "0.0.7"), ("ha-b", "0.0.6")]:
                write(root / name / "SKILL.md", VALID_FRONTMATTER.replace('"0.0.7"', f'"{version}"'))
            errors = []
            validate.validate_one_version(sorted(root.iterdir()), errors)
            self.assertEqual(len(errors), 1, errors)
            self.assertIn("0.0.6: ha-b", errors[0])


class StructureTests(unittest.TestCase):
    def test_a_section_named_in_prose_is_refused(self):
        with Rooted() as root:
            path = write(root / "ha-test" / "SKILL.md", 'Read it (especially "Quick control") first.\n')
            errors = []
            validate.validate_prose_section_refs(path, errors)
            self.assertEqual(len(errors), 1, errors)

    def test_go_source_citations_are_refused(self):
        with Rooted() as root:
            path = write(root / "ha-test" / "references" / "r.md", "see cmd/workflow.go:160\n")
            errors = []
            validate.validate_no_source_citations(path, errors)
            self.assertEqual(len(errors), 1, errors)

    def test_a_verb_row_that_only_points_at_help_is_refused(self):
        with Rooted() as root:
            skill = write(
                root / "ha-test" / "SKILL.md",
                "| Verb | Does | Read when triggered |\n|---|---|---|\n"
                "| `get` | read | `workflow --help` |\n"
                "| `save` | write | id is bare; `workflow --help` |\n",
            )
            errors = []
            validate.validate_verb_index_rows(skill, errors)
            self.assertEqual(len(errors), 1, errors)
            self.assertIn("`get`", errors[0])

    def test_a_fast_path_over_the_budget_is_refused(self):
        with Rooted() as root:
            write(root / validate.SHARED_SKILL / "SKILL.md", "front door\n" * 100)
            write(root / "ha-test" / "references" / "big.md", "x\n" * 149)
            skill_dir = root / "ha-test"
            write(
                skill_dir / "SKILL.md",
                "## Fast paths\n\n| Task | Read | First command |\n|---|---|---|\n"
                "| build | [big](references/big.md) | `hass-cli workflow script save` |\n",
            )
            errors = []
            validate.validate_fast_paths(skill_dir, errors)
            self.assertEqual(len(errors), 1, errors)
            self.assertIn("first-command budget", errors[0])

    def test_a_skill_without_fast_paths_is_refused_but_the_front_door_is_exempt(self):
        with Rooted() as root:
            for name, expected in [("ha-test", 1), (validate.SHARED_SKILL, 0)]:
                skill_dir = root / name
                write(skill_dir / "SKILL.md", "## Verb index\n")
                errors = []
                validate.validate_fast_paths(skill_dir, errors)
                self.assertEqual(len(errors), expected, errors)

    def test_the_shared_sections_are_required(self):
        complete = (
            "> **Shared front door:** read ha-shared first.\n\n"
            "## Verb index\n\n| Verb | Does | Read when triggered |\n|---|---|---|\n"
            "| `list` | list | this file |\n\n"
            "## Safety and escalation\n\n- Stop.\n"
        )
        for missing in ["## Safety and escalation", "## Verb index", "> **Shared front door:**"]:
            with self.subTest(missing=missing), Rooted() as root:
                skill_dir = root / "ha-test"
                write(skill_dir / "SKILL.md", complete.replace(missing, "## Other"))
                errors = []
                validate.validate_shared_sections(skill_dir, errors)
                self.assertTrue(errors, f"{missing} went unnoticed")
        with Rooted() as root:
            skill_dir = root / "ha-test"
            write(skill_dir / "SKILL.md", complete)
            errors = []
            validate.validate_shared_sections(skill_dir, errors)
            self.assertEqual(errors, [])

    def test_only_the_front_door_references_may_be_deep_linked(self):
        with Rooted() as root:
            write(root / "ha-shared" / "references" / "ha-shared-connection.md", "x\n")
            write(root / "ha-automation" / "references" / "ha-automation-scripts.md", "x\n")
            skill_dir = root / "ha-lighting"
            write(
                skill_dir / "SKILL.md",
                "[connection](../ha-shared/references/ha-shared-connection.md)\n"
                "[scripts](../ha-automation/references/ha-automation-scripts.md)\n",
            )
            errors = []
            validate.validate_structure(skill_dir, errors)
            self.assertEqual(len(errors), 1, errors)
            self.assertIn("deep-links ha-automation/references", errors[0])

    def test_an_unlinked_reference_is_refused(self):
        with Rooted() as root:
            skill_dir = root / "ha-test"
            write(skill_dir / "references" / "ha-test-x.md", "x\n")
            write(skill_dir / "SKILL.md", "no links\n")
            errors = []
            validate.validate_skill_entrypoint(skill_dir, errors)
            self.assertEqual(len(errors), 1, errors)


class StampTests(unittest.TestCase):
    def write_suite(self, root, names):
        for name in names:
            write(
                root / name / "SKILL.md",
                VALID_FRONTMATTER.replace("name: ha-test", f"name: {name}")
                + "\n## Example\n\nversion: 9.9.9\n",
            )

    def test_every_skill_is_stamped_and_the_body_left_alone(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.write_suite(root, ["ha-a", "ha-b"])
            stamped = stamp.stamp(root, "v0.1.0")
            self.assertEqual(len(stamped), 2)
            for skill in stamped:
                self.assertEqual(validate.read_frontmatter_version(skill), "0.1.0")
                self.assertIn("version: 9.9.9", skill.read_text(encoding="utf-8"))

    def test_a_release_that_is_not_one_is_refused(self):
        for version in ["", "latest", "0.1", "0.1.0-SNAPSHOT-abc"]:
            with tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                self.write_suite(root, ["ha-a"])
                with self.assertRaises(stamp.StampError, msg=version):
                    stamp.stamp(root, version)

    def test_an_empty_tree_is_an_error(self):
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(stamp.StampError):
                stamp.stamp(Path(directory), "0.1.0")

    def test_the_committed_suite_declares_the_placeholder(self):
        for skill in sorted(Path(__file__).parent.glob("ha-*/SKILL.md")):
            self.assertEqual(
                validate.read_frontmatter_version(skill),
                stamp.PLACEHOLDER_VERSION,
                f"{skill.parent.name} carries a hand-written version",
            )


if __name__ == "__main__":
    unittest.main()
