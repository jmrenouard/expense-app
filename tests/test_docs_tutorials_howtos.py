#!/usr/bin/env python3
"""Executable documentation tests for Diataxis EN/FR docs, Mermaid diagrams, links, and Go code parity."""

from __future__ import annotations

import pathlib
import re
import shutil
import subprocess
import tempfile
import unittest

REPO_ROOT = pathlib.Path(__file__).resolve().parents[1]

EN_QUADRANTS = ("01-tutorials", "02-how-to-guides", "03-reference", "04-explanation")
FR_QUADRANTS = ("01-tutoriels", "02-guides-pratiques", "03-reference", "04-explications")

VALID_MERMAID_HEADERS = re.compile(
    r"^\s*(?:flowchart(?:-elk)?|graph\s+(?:TD|TB|LR|RL|BT)|sequenceDiagram|"
    r"stateDiagram(?:-v2)?|classDiagram|erDiagram|gantt|pie|gitGraph|"
    r"mindmap|timeline|xychart-beta)\b"
)


def iter_markdown_files() -> list[pathlib.Path]:
    """Return sorted Markdown files in the repository excluding hidden directories."""
    out: list[pathlib.Path] = []
    for p in sorted(REPO_ROOT.rglob("*.md")):
        if any(part.startswith(".") for part in p.relative_to(REPO_ROOT).parts):
            continue
        out.append(p)
    return out


def parse_fenced_blocks(text: str) -> list[dict[str, object]]:
    """Parse CommonMark fenced code blocks and assert zero unclosed fences."""
    blocks: list[dict[str, object]] = []
    in_fence = False
    fence_char = ""
    fence_len = 0
    open_line = 0
    indent = ""
    lang = ""
    buf: list[str] = []

    for idx, line in enumerate(text.splitlines(), start=1):
        if not in_fence:
            m = re.match(r"^([ ]{0,3})(`{3,}|~{3,})(.*)$", line)
            if not m:
                continue
            ind, marker, info = m.groups()
            if marker[0] == "`" and "`" in info:
                continue
            in_fence = True
            fence_char = marker[0]
            fence_len = len(marker)
            open_line = idx
            indent = ind
            lang = (info.strip().split()[0].lower().lstrip(".")) if info.strip() else ""
            buf = []
        else:
            if re.match(rf"^[ \t]*{re.escape(fence_char)}{{{fence_len},}}[ \t]*$", line):
                blocks.append(
                    {
                        "open_line": open_line,
                        "close_line": idx,
                        "indent": indent,
                        "lang": lang,
                        "body": "\n".join(buf),
                    }
                )
                in_fence = False
            else:
                buf.append(line)

    if in_fence:
        raise AssertionError(f"Unclosed code fence at line {open_line}")
    return blocks


class TestDocsTutorialsHowTos(unittest.TestCase):
    """Verify Diataxis structure, links, Mermaid diagrams, FR/EN hygiene, and code/doc parity."""

    def test_diataxis_structure_and_relative_links(self) -> None:
        """Verify EN and FR Diataxis trees exist and every relative Markdown link resolves."""
        for base, quads in ((REPO_ROOT / "docs", EN_QUADRANTS), (REPO_ROOT / "docs" / "fr", FR_QUADRANTS)):
            self.assertTrue((base / "index.md").is_file(), f"Missing {base / 'index.md'}")
            for q in quads:
                q_dir = base / q
                self.assertTrue((q_dir / "index.md").is_file(), f"Missing {q_dir / 'index.md'}")
                pages = [p for p in q_dir.glob("*.md") if p.name != "index.md"]
                self.assertGreaterEqual(len(pages), 2, f"Expected >=2 content pages in {q_dir}")

        link_re = re.compile(r"!?\[[^\]]*\]\(([^)\n]+)\)")
        broken: list[str] = []
        for md in iter_markdown_files():
            txt = md.read_text(encoding="utf-8")
            blocks = parse_fenced_blocks(txt)
            fenced = {ln for b in blocks for ln in range(int(b["open_line"]), int(b["close_line"]) + 1)}
            for lineno, line in enumerate(txt.splitlines(), start=1):
                if lineno in fenced:
                    continue
                cleaned = re.sub(r"`[^`]*`", " ", line)
                for m in link_re.finditer(cleaned):
                    tgt = m.group(1).strip().split()[0].split("#")[0].split("?")[0]
                    if not tgt or re.match(r"^(?:https?|mailto|ftp):", tgt, re.I):
                        continue
                    if not (md.parent / tgt).resolve().exists():
                        broken.append(f"{md.relative_to(REPO_ROOT)}:{lineno} -> {tgt}")
        self.assertEqual(broken, [], f"Broken relative links: {broken}")

    def test_mermaid_diagrams_fenced_valid_and_no_escaped_underscores(self) -> None:
        """Verify zero bare/indented/ASCII diagrams and validate Mermaid blocks with mmdc if available."""
        mermaid_blocks: list[tuple[str, str]] = []
        bare_mermaid = re.compile(r"^\s*(?:erDiagram|sequenceDiagram|stateDiagram(?:-v2)?|flowchart(?:\s+(?:LR|TD|TB))?)\b")

        for md in iter_markdown_files():
            txt = md.read_text(encoding="utf-8")
            rel = md.relative_to(REPO_ROOT).as_posix()
            blocks = parse_fenced_blocks(txt)
            fenced = {ln for b in blocks for ln in range(int(b["open_line"]), int(b["close_line"]) + 1)}

            for lineno, line in enumerate(txt.splitlines(), start=1):
                if lineno not in fenced and bare_mermaid.match(line):
                    self.fail(f"Bare unfenced Mermaid diagram in {rel}:{lineno}")

            for b in blocks:
                lang = str(b["lang"])
                body = str(b["body"])
                if lang == "mermaid":
                    self.assertEqual(b["indent"], "", f"Indented Mermaid block in {rel}:{b['open_line']}")
                    first = next((l for l in body.splitlines() if l.strip() and not l.strip().startswith("%%")), "")
                    self.assertRegex(first, VALID_MERMAID_HEADERS, f"Invalid Mermaid header in {rel}:{b['open_line']}")
                    self.assertNotIn(r"\_", body, f"Escaped underscore inside Mermaid block in {rel}:{b['open_line']}")
                    mermaid_blocks.append((f"{rel}:{b['open_line']}", body))
                elif lang not in ("diff", "patch", "sql", "mysql", "psql"):
                    self.assertFalse(
                        re.search(r"[┌┐┘╔╗╚╝]|\+[-=]{2,}\+", body),
                        f"ASCII box diagram in {rel}:{b['open_line']}",
                    )

        self.assertGreaterEqual(len(mermaid_blocks), 10)
        mmdc_bin = shutil.which("mmdc") or (
            "/home/jmren/node_modules/.bin/mmdc"
            if pathlib.Path("/home/jmren/node_modules/.bin/mmdc").is_file()
            else None
        )
        if mmdc_bin:
            with tempfile.TemporaryDirectory() as tmp:
                for label, body in mermaid_blocks:
                    src = pathlib.Path(tmp) / "diagram.mmd"
                    out = pathlib.Path(tmp) / "diagram.svg"
                    src.write_text(body, encoding="utf-8")
                    proc = subprocess.run([mmdc_bin, "-i", str(src), "-o", str(out)], capture_output=True, text=True)
                    self.assertEqual(proc.returncode, 0, f"mmdc error for {label}: {proc.stderr}")
                    self.assertTrue(out.is_file() and out.stat().st_size > 0)

    def test_code_and_documentation_parity(self) -> None:
        """Verify documented HTTP routes, env vars, and frontend stack match Go/JS source."""
        main_go = (REPO_ROOT / "main.go").read_text(encoding="utf-8")
        route_matches = re.findall(r'(r|api|admin)\.(GET|POST|PUT|DELETE)\("([^"]+)"', main_go)
        expected_routes: set[tuple[str, str]] = set()
        for grp, method, sub in route_matches:
            if grp == "r":
                full = sub
            elif grp == "api":
                full = "/api" + sub
            else:
                full = "/api/admin" + sub
            expected_routes.add((method, full))

        self.assertEqual(len(expected_routes), 21, f"Unexpected Go route count: {sorted(expected_routes)}")

        for doc_rel in (
            "SPECIFICATIONS.md",
            "SPECIFICATIONS_FR.md",
            "docs/03-reference/01-rest-api-and-permissions.md",
            "docs/fr/03-reference/01-api-rest-et-permissions.md",
        ):
            doc_text = (REPO_ROOT / doc_rel).read_text(encoding="utf-8")
            for method, route in sorted(expected_routes):
                self.assertIn(route, doc_text, f"Route {method} {route} missing from {doc_rel}")
                self.assertRegex(
                    doc_text,
                    rf"{method}.*{re.escape(route)}",
                    f"Method+path pair {method} {route} missing from {doc_rel}",
                )

        readme = (REPO_ROOT / "README.md").read_text(encoding="utf-8")
        self.assertNotIn("Svelte", readme)
        for env_var in ("PORT", "DATADIR", "JWT_SECRET", "ADMIN_EMAIL", "ADMIN_PASSWORD"):
            self.assertIn(env_var, readme)

        tests_dockerfile = (REPO_ROOT / "tests" / "Dockerfile").read_text(encoding="utf-8")
        self.assertNotIn("../go.mod", tests_dockerfile)

    def test_fr_en_separation_and_runbook_governance(self) -> None:
        """Verify strict FR/EN separation and Diataxis STE/STF structural compliance."""
        specs_en = (REPO_ROOT / "SPECIFICATIONS.md").read_text(encoding="utf-8")
        self.assertNotIn("ex: reports:approve", specs_en)

        for md in sorted((REPO_ROOT / "docs" / "02-how-to-guides").glob("*.md")):
            if md.name == "index.md":
                continue
            txt = md.read_text(encoding="utf-8")
            self.assertRegex(txt, r"(?im)^##\s+Verification\b")
            self.assertRegex(txt, r"(?im)^##\s+Rollback\b")
            self.assertNotRegex(txt, r"(?im)^##+\s+.*(?:Why|Architecture|Theory)\b")
            stripped = re.sub(r"```[\s\S]*?```|`[^`]*`", "", txt)
            self.assertNotIn(";", stripped)

        for md in sorted((REPO_ROOT / "docs" / "fr" / "02-guides-pratiques").glob("*.md")):
            if md.name == "index.md":
                continue
            txt = md.read_text(encoding="utf-8")
            self.assertRegex(txt, r"(?im)^##\s+V[ée]rification\b")
            self.assertRegex(txt, r"(?im)^##\s+Retour\s+arri[èe]re\b")
            self.assertNotRegex(txt, r"(?im)^##+\s+.*(?:Pourquoi|Architecture|Th[ée]orie)\b")
            stripped = re.sub(r"```[\s\S]*?```|`[^`]*`", "", txt)
            self.assertNotIn(";", stripped)


if __name__ == "__main__":
    unittest.main()
