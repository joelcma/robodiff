#!/usr/bin/env python3
"""Find useful FAIL message examples in a large Robot Framework output.xml.

This script is intentionally dependency-free and uses streaming XML parsing.

Typical usage:
  python3 scripts/find_robot_fail_examples.py /path/to/output.xml
  python3 scripts/find_robot_fail_examples.py /path/to/output.xml -k "Should Be Equal" --operator "!=" --jsonish
  python3 scripts/find_robot_fail_examples.py /path/to/output.xml -k "Should Contain" --limit 50 --full

Notes:
- For BuiltIn keywords like "Should Be Equal", the <arg> values are often variables/expressions.
  The interesting payload is usually in FAIL <msg> and/or the FAIL <status> element text.
"""

from __future__ import annotations

import argparse
import re
import sys
import xml.etree.ElementTree as ET


def _text(e: ET.Element | None) -> str:
    if e is None or e.text is None:
        return ""
    return e.text.strip()


def _iter_fail_texts(kw: ET.Element) -> list[str]:
    texts: list[str] = []

    for msg in kw.findall("msg"):
        level = (msg.get("level") or "").upper()
        if level == "FAIL":
            t = _text(msg)
            if t:
                texts.append(t)

    status = kw.find("status")
    if status is not None:
        status_attr = (status.get("status") or "").upper()
        if status_attr == "FAIL":
            t = _text(status)
            if t and t not in texts:
                texts.append(t)

    return texts


def _kw_to_xml(kw: ET.Element) -> str:
    # xml.etree uses no pretty-printing, but it's good enough for copy/paste.
    # Preserve all child nodes (msg/arg/status/nested kw).
    return ET.tostring(kw, encoding="unicode")


def _looks_jsonish(s: str) -> bool:
    # Heuristic: JSON-ish if it contains obvious container delimiters.
    # (We keep this broad; the UI prettifier has more intelligence.)
    return "{" in s or "[" in s


def _looks_like_comparison(s: str, operator: str) -> bool:
    if operator == "any":
        return "!=" in s or "==" in s
    return operator in s


def _iter_tests(xml_path: str):
    # Stream end-events; process each <test> subtree and clear it.
    # This keeps memory bounded even for huge output.xml.
    context = ET.iterparse(xml_path, events=("end",))
    for _, elem in context:
        if elem.tag == "test":
            yield elem
            elem.clear()


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("xml", help="Path to Robot Framework output.xml")
    ap.add_argument(
        "-k",
        "--keyword",
        default="Should Be Equal",
        help="Keyword name substring to match (case-insensitive). Use 'any' to match all keywords.",
    )
    ap.add_argument(
        "--operator",
        choices=["!=", "==", "any"],
        default="!=",
        help="Only show messages containing this operator.",
    )
    ap.add_argument(
        "--contains",
        default="",
        help="Only show messages containing this substring.",
    )
    ap.add_argument(
        "--jsonish",
        action="store_true",
        help="Only show messages that look like they contain JSON/arrays.",
    )
    ap.add_argument(
        "--limit",
        type=int,
        default=20,
        help="Maximum number of matches to print.",
    )
    ap.add_argument(
        "--full",
        action="store_true",
        help="Print full FAIL message text (default prints a shortened preview).",
    )
    ap.add_argument(
        "--no-kw",
        dest="kw",
        action="store_false",
        help="Do not print the full matching <kw>...</kw> XML block.",
    )
    ap.set_defaults(kw=True)
    ap.add_argument(
        "--kw-max-chars",
        type=int,
        default=200_000,
        help="Max characters to print for the <kw> XML block (use 0 for no limit).",
    )
    ap.add_argument(
        "--show-args",
        action="store_true",
        help="Also print the first few <arg> values for the keyword.",
    )

    args = ap.parse_args(argv)

    kw_filter = args.keyword.strip().lower()
    contains_filter = args.contains

    printed = 0
    total_fail_kws = 0

    try:
        for test in _iter_tests(args.xml):
            test_name = test.get("name") or "(unnamed test)"

            # Iterate all keywords under this test.
            for kw in test.iter("kw"):
                kw_name = (kw.get("name") or "").strip()
                if not kw_name:
                    continue

                if kw_filter != "any" and kw_filter not in kw_name.lower():
                    continue

                fail_texts = _iter_fail_texts(kw)
                if not fail_texts:
                    continue

                total_fail_kws += 1

                for ft in fail_texts:
                    if contains_filter and contains_filter not in ft:
                        continue
                    if not _looks_like_comparison(ft, args.operator):
                        continue
                    if args.jsonish and not _looks_jsonish(ft):
                        continue

                    # Found a match.
                    printed += 1

                    header = f"[{printed}] test={test_name} | kw={kw_name}"
                    print("=" * len(header))
                    print(header)
                    print("=" * len(header))

                    if args.kw:
                        kw_xml = _kw_to_xml(kw)
                        if (
                            args.kw_max_chars
                            and args.kw_max_chars > 0
                            and len(kw_xml) > args.kw_max_chars
                        ):
                            kw_xml = kw_xml[: args.kw_max_chars] + "\n...(kw truncated)"
                        print("kw xml:")
                        print(kw_xml)
                        print()

                    if args.show_args:
                        kw_args = [(_text(a) or "") for a in kw.findall("arg")]
                        if kw_args:
                            print("args:")
                            for a in kw_args[:6]:
                                print(f"  - {a}")
                            if len(kw_args) > 6:
                                print(f"  ... ({len(kw_args) - 6} more)")

                    if args.full:
                        print("FAIL text:")
                        print(ft)
                    else:
                        preview = ft
                        if len(preview) > 800:
                            preview = preview[:800] + " …(truncated)"
                        print("FAIL text (preview):")
                        print(preview)

                    print()

                    if printed >= args.limit:
                        print(
                            f"Stopped after {printed} matches (limit). Searched FAIL keywords: {total_fail_kws}."
                        )
                        return 0

        print(
            f"Done. Found {printed} matches. Searched FAIL keywords: {total_fail_kws}."
        )
        if printed == 0:
            print(
                "Tip: try --operator any, drop --jsonish, or use --keyword any to broaden the search."
            )
        return 0

    except FileNotFoundError:
        print(f"File not found: {args.xml}", file=sys.stderr)
        return 2
    except ET.ParseError as e:
        print(
            f"XML parse error (file may be incomplete while writing): {e}",
            file=sys.stderr,
        )
        return 3


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
