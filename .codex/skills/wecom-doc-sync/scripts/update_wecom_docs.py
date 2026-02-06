#!/usr/bin/env python3
"""Update WeCom AI bot docs by fetching content_md via docFetch endpoint."""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import time
import tempfile
from pathlib import Path
from typing import Dict, Optional, Tuple

USER_AGENT = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
    "AppleWebKit/537.36 (KHTML, like Gecko) "
    "Chrome/120.0.0.0 Safari/537.36"
)
DOC_FETCH_URL = (
    "https://developer.work.weixin.qq.com/docFetch/fetchCnt"
    "?lang=zh_CN&ajax=1&f=json&random="
)
FRONTMATTER_RE = re.compile(r"\A---\s*\n(.*?)\n---\s*\n", re.S)
HEADING_RE = re.compile(r"^(\s*)(#+)(.*)$")
LIST_RE = re.compile(r"^\s*(?:[-+*]|\d+\.)\s+")
FENCE_RE = re.compile(r"^\s*(```|~~~)")


def parse_frontmatter(text: str) -> Tuple[Optional[str], Optional[Dict[str, str]], str]:
    """Parse YAML frontmatter.

    Args:
        text: Full markdown text.

    Returns:
        A tuple of (frontmatter_raw, meta_dict, body). If no frontmatter exists,
        returns (None, None, full_text).
    """

    match = FRONTMATTER_RE.match(text)
    if not match:
        return None, None, text

    fm_content = match.group(1)
    fm_raw = text[: match.end()]
    body = text[match.end() :]
    meta: Dict[str, str] = {}

    for line in fm_content.splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if ":" not in line:
            continue
        key, value = line.split(":", 1)
        key = key.strip()
        value = value.strip()
        if (
            len(value) >= 2
            and value[0] == value[-1]
            and value[0] in ("\"", "'")
        ):
            value = value[1:-1]
        meta[key] = value

    return fm_raw, meta, body


def fetch_content_md(
    doc_id: str,
    source_url: str,
    cookie: Optional[str],
    timeout: int,
) -> Tuple[Optional[str], Optional[str]]:
    """Fetch content_md from the docFetch endpoint.

    Args:
        doc_id: Document id to request.
        source_url: Original document URL for the Referer header.
        cookie: Optional cookie string passed to curl.
        timeout: Curl timeout in seconds.

    Returns:
        A tuple of (content_md, error_message). If error occurs, content_md is None.
    """

    url = DOC_FETCH_URL + str(int(time.time() * 1000))
    cmd = [
        "curl",
        "-s",
        "--max-time",
        str(timeout),
        url,
        "-H",
        "accept: application/json, text/plain, */*",
        "-H",
        "content-type: application/x-www-form-urlencoded",
        "-H",
        "origin: https://developer.work.weixin.qq.com",
        "-H",
        f"referer: {source_url}",
        "-H",
        f"user-agent: {USER_AGENT}",
        "--data-raw",
        f"doc_id={doc_id}",
    ]
    if cookie:
        cmd.extend(["-b", cookie])

    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        return None, f"curl failed: {result.stderr.strip() or result.returncode}"

    try:
        payload = json.loads(result.stdout)
    except json.JSONDecodeError as exc:
        return None, f"json decode failed: {exc}"

    if "statusCode" in payload and payload.get("statusCode") != 200:
        message = (
            payload.get("result", {}) or {}
        ).get("humanMessage", "unknown error")
        return None, f"server error: {message}"

    data = payload.get("data")
    if not isinstance(data, dict):
        return None, "missing data"

    content_md = data.get("content_md")
    if not content_md:
        return None, "missing content_md"

    return content_md, None


def normalize_heading_line(line: str) -> str:
    """Normalize heading line spacing and extra hashes.

    Args:
        line: Heading line starting with #.

    Returns:
        Normalized heading line.
    """

    match = HEADING_RE.match(line)
    if not match:
        return line

    indent, hashes, rest = match.groups()
    rest = rest.lstrip()
    rest = rest.lstrip("#").lstrip()
    if rest:
        return f"{indent}{hashes} {rest}"
    return f"{indent}{hashes}"


def is_fence_line(line: str) -> bool:
    """Return True if the line starts a fenced code block marker."""

    return bool(FENCE_RE.match(line))


def is_heading_line(line: str) -> bool:
    """Return True if the line is a markdown heading."""

    return line.lstrip().startswith("#")


def is_list_item(line: str) -> bool:
    """Return True if the line looks like a list item."""

    return bool(LIST_RE.match(line))


def is_table_separator(line: str) -> bool:
    """Return True if the line looks like a markdown table separator."""

    s = line.strip()
    if not s or "|" not in s or "-" not in s:
        return False
    for ch in s:
        if ch not in "|:- ":
            return False
    return True


def is_table_header(line: str, next_line: str) -> bool:
    """Return True if line + next_line form a table header+separator."""

    return "|" in line and is_table_separator(next_line)


def is_table_row(line: str) -> bool:
    """Return True if the line looks like a table row."""

    return "|" in line


def format_markdown(content: str) -> str:
    """Format markdown using the SKILL.md rules.

    Args:
        content: Raw markdown content.

    Returns:
        Formatted markdown content.
    """

    lines = content.splitlines()
    out: list[str] = []
    in_fence = False
    in_table = False
    pending_blank_after_heading = False
    pending_blank_after_fence = False
    prev_block: Optional[str] = None

    for idx, raw_line in enumerate(lines):
        line = raw_line.rstrip("\r")

        if is_fence_line(line):
            if not in_fence and out and out[-1].strip():
                out.append("")
            out.append(line.rstrip())
            in_fence = not in_fence
            if not in_fence:
                pending_blank_after_fence = True
            prev_block = "fence"
            continue

        if in_fence:
            out.append(line)
            continue

        if pending_blank_after_fence:
            if line.strip():
                out.append("")
            pending_blank_after_fence = False

        if in_table and not is_table_row(line):
            if line.strip() and out and out[-1].strip():
                out.append("")
            in_table = False
            prev_block = None

        if is_heading_line(line):
            normalized = normalize_heading_line(line).rstrip()
            out.append(normalized)
            pending_blank_after_heading = True
            prev_block = "heading"
            continue

        if pending_blank_after_heading:
            if line.strip():
                out.append("")
            pending_blank_after_heading = False

        next_line = lines[idx + 1] if idx + 1 < len(lines) else ""
        if not in_table and is_table_header(line, next_line):
            if out and out[-1].strip():
                out.append("")
            in_table = True
            prev_block = "table"

        is_blank = not line.strip()
        is_list = is_list_item(line)
        if not is_blank and not in_table:
            if is_list and prev_block == "paragraph":
                if out and out[-1].strip():
                    out.append("")
            if (not is_list) and prev_block == "list":
                if out and out[-1].strip():
                    out.append("")

        out.append(line.rstrip())

        if not is_blank:
            if in_table or is_table_row(line):
                prev_block = "table"
            elif is_list:
                prev_block = "list"
            else:
                prev_block = "paragraph"

    formatted = "\n".join(out).rstrip() + "\n"
    return formatted


def diff_important_changes(
    old_body: str,
    new_body: str,
    label: str,
) -> Tuple[Optional[str], Optional[str]]:
    """Diff old vs new markdown, ignoring whitespace/blank lines.

    Args:
        old_body: Existing markdown body.
        new_body: Newly fetched markdown body.
        label: Label for diff output.

    Returns:
        A tuple of (diff_text, error_message). diff_text is None if no changes.
    """

    with tempfile.NamedTemporaryFile("w+", delete=False) as old_file:
        old_file.write(old_body)
        old_path = old_file.name
    with tempfile.NamedTemporaryFile("w+", delete=False) as new_file:
        new_file.write(new_body)
        new_path = new_file.name

    cmd = [
        "diff",
        "-u",
        "-w",
        "-B",
        "--label",
        f"{label} (old)",
        "--label",
        f"{label} (new)",
        old_path,
        new_path,
    ]
    result = subprocess.run(cmd, capture_output=True, text=True)
    os.unlink(old_path)
    os.unlink(new_path)

    if result.returncode == 0:
        return None, None
    if result.returncode == 1:
        return result.stdout, None
    return None, result.stderr.strip() or "diff failed"


def update_markdown(
    path: Path,
    cookie: Optional[str],
    dry_run: bool,
    timeout: int,
    show_changes: bool,
) -> Tuple[bool, str, Optional[str]]:
    """Update a single markdown file.

    Args:
        path: Path to markdown file.
        cookie: Optional cookie string passed to curl.
        dry_run: If True, do not write file.
        timeout: Curl timeout in seconds.
        show_changes: If True, print important changes diff.

    Returns:
        A tuple of (success, message, change_output).
    """

    text = path.read_text(encoding="utf-8")
    fm_raw, meta, _body = parse_frontmatter(text)

    if not fm_raw or meta is None:
        return False, "missing frontmatter", None

    doc_id = meta.get("doc_id")
    source_url = meta.get("source_url")
    doc_name = meta.get("doc_name") or path.stem

    if not doc_id or not doc_id.isdigit():
        return False, f"invalid doc_id for {doc_name}", None
    if not source_url:
        return False, f"missing source_url for {doc_name}", None

    # Fetch latest markdown content from WeCom docs.
    content_md, error = fetch_content_md(doc_id, source_url, cookie, timeout)
    if error:
        return False, f"{doc_name}: {error}", None

    change_output: Optional[str] = None
    if show_changes:
        diff_text, diff_error = diff_important_changes(
            _body, content_md, path.as_posix()
        )
        if diff_error:
            change_output = f"[CHANGE] {path.name}: diff failed: {diff_error}"
        elif diff_text:
            change_output = f"[CHANGE] {path.name}\n{diff_text.rstrip()}"
        else:
            change_output = f"[CHANGE] {path.name}: no important changes"

    formatted = format_markdown(content_md)
    new_text = fm_raw.rstrip("\n") + "\n\n" + formatted.rstrip() + "\n"
    if dry_run:
        return True, f"{doc_name}: dry-run (len={len(content_md)})", change_output

    path.write_text(new_text, encoding="utf-8")
    return True, f"{doc_name}: updated (len={len(content_md)})", change_output


def collect_targets(target_dir: Path, target_file: Optional[Path]) -> list[Path]:
    """Collect markdown files to process.

    Args:
        target_dir: Directory containing markdown files.
        target_file: Optional single file override.

    Returns:
        List of markdown file paths.
    """

    if target_file:
        return [target_file]
    return sorted(target_dir.glob("*.md"))


def git_diff_numstat(
    target: Path,
) -> Tuple[list[Tuple[str, str, str]], int, int, Optional[str]]:
    """Collect git diff numstat output for the target.

    Args:
        target: File or directory to diff against HEAD.

    Returns:
        A tuple of (rows, total_add, total_del, error_message). rows is a list of
        (add_raw, del_raw, path). error_message is None on success.
    """

    cmd = ["git", "diff", "--numstat", "HEAD", "--", str(target)]
    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        return [], 0, 0, result.stderr.strip() or "git diff failed"

    rows: list[Tuple[str, str, str]] = []
    total_add = 0
    total_del = 0

    for line in result.stdout.splitlines():
        parts = line.split("\t")
        if len(parts) < 3:
            continue
        add_raw, del_raw, path = parts[0], parts[1], parts[2]
        rows.append((add_raw, del_raw, path))
        if add_raw.isdigit():
            total_add += int(add_raw)
        if del_raw.isdigit():
            total_del += int(del_raw)

    return rows, total_add, total_del, None


def build_arg_parser() -> argparse.ArgumentParser:
    """Build CLI argument parser.

    Returns:
        Configured ArgumentParser instance.
    """

    parser = argparse.ArgumentParser(
        description="Update WeCom AI bot docs from docFetch content_md"
    )
    parser.add_argument(
        "--dir",
        default="docs/appendix/wecom-official/wecom_ai_bot",
        help="Target directory with markdown files",
    )
    parser.add_argument(
        "--file",
        help="Update a single markdown file",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Fetch and report only, do not write files",
    )
    parser.add_argument(
        "--show-changes",
        dest="show_changes",
        action="store_true",
        help="Print important changes (diff -u -w -B) before formatting",
    )
    parser.add_argument(
        "--no-show-changes",
        dest="show_changes",
        action="store_false",
        help="Disable important changes output",
    )
    parser.add_argument(
        "--show-diff",
        action="store_true",
        help="After sync, print git diff --numstat stats against HEAD",
    )
    parser.add_argument(
        "--timeout",
        type=int,
        default=30,
        help="Curl timeout in seconds",
    )
    parser.add_argument(
        "--cookie",
        help="Optional cookie string for curl (-b).",
    )
    parser.set_defaults(show_changes=True)
    return parser


def main() -> int:
    """CLI entrypoint.

    Returns:
        Process exit code.
    """

    parser = build_arg_parser()
    args = parser.parse_args()

    target_dir = Path(args.dir)
    target_file = Path(args.file) if args.file else None
    cookie = args.cookie or os.getenv("WEWORK_COOKIE")

    if target_file and not target_file.exists():
        print(f"[ERROR] file not found: {target_file}", file=sys.stderr)
        return 2
    if not target_file and not target_dir.exists():
        print(f"[ERROR] dir not found: {target_dir}", file=sys.stderr)
        return 2

    # Collect targets and process sequentially for predictable logs.
    targets = collect_targets(target_dir, target_file)
    if not targets:
        print("[WARN] no markdown files found")
        return 0

    ok_count = 0
    for path in targets:
        success, message, change_output = update_markdown(
            path,
            cookie,
            args.dry_run,
            args.timeout,
            args.show_changes,
        )
        if success:
            ok_count += 1
            print(f"[OK] {path.name}: {message}")
        else:
            print(f"[SKIP] {path.name}: {message}")
        if change_output:
            print(change_output)

    print(f"[DONE] {ok_count}/{len(targets)} updated")

    if args.show_diff:
        # Show git diff summary for the updated target scope.
        diff_target = target_file or target_dir
        rows, total_add, total_del, error = git_diff_numstat(diff_target)
        if error:
            print(f"[WARN] diff failed: {error}")
            return 0
        print(f"[DIFF] target={diff_target}")
        if not rows:
            print("[DIFF] no changes")
            return 0
        for add_raw, del_raw, path in rows:
            print(f"[DIFF] +{add_raw} -{del_raw} {path}")
        print(f"[DIFF] total +{total_add} -{total_del} files={len(rows)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
