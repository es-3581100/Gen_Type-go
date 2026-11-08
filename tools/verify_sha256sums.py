#!/usr/bin/env python3
"""Verify tracked source checksums; this is integrity evidence, not authority."""

import argparse
import hashlib
from pathlib import Path
import re
import subprocess
import sys


_ENTRY = re.compile(r"([0-9a-f]{64})  (.+)")


class VerificationError(Exception):
    pass


def verify(root: Path) -> int:
    root = root.resolve()
    git = subprocess.run(
        ["git", "-C", str(root), "ls-files", "--cached", "-z"],
        check=True, capture_output=True,
    )
    try:
        tracked = [p.decode("utf-8") for p in git.stdout.split(b"\x00") if p]
    except UnicodeDecodeError as exc:
        raise VerificationError("tracked filename is not UTF-8") from exc
    if "SHA256SUMS" not in tracked:
        raise VerificationError("SHA256SUMS must be tracked")
    expected = sorted((p for p in tracked if p != "SHA256SUMS"), key=lambda p: p.encode("utf-8"))
    manifest = root / "SHA256SUMS"
    raw = manifest.read_bytes()
    if not raw or not raw.endswith(b"\n") or b"\r" in raw:
        raise VerificationError("manifest must be nonempty LF-terminated text")
    try:
        lines = raw[:-1].decode("utf-8").split("\n")
    except UnicodeDecodeError as exc:
        raise VerificationError("manifest must be UTF-8") from exc
    records = []
    for number, line in enumerate(lines, start=1):
        match = _ENTRY.fullmatch(line)
        if not match:
            raise VerificationError(f"invalid manifest entry on line {number}")
        records.append((match.group(2), match.group(1)))
    paths = [path for path, _ in records]
    if paths != expected:
        raise VerificationError("manifest paths must equal all tracked paths except SHA256SUMS, in unique bytewise-sorted order")
    for path, expected_hash in records:
        source = root / path
        if source.is_symlink() or not source.is_file():
            raise VerificationError(f"tracked entry is not a regular file: {path}")
        with source.open("rb") as stream:
            actual_hash = hashlib.file_digest(stream, "sha256").hexdigest()
        if actual_hash != expected_hash:
            raise VerificationError(f"SHA-256 mismatch: {path}")
    return len(records)


def main(argv=None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    args = parser.parse_args(argv)
    try:
        count = verify(args.root)
    except (VerificationError, OSError, subprocess.CalledProcessError) as exc:
        print(f"CHECKSUM_VERIFY_FAIL: {exc}", file=sys.stderr)
        return 1
    print(f"CHECKSUM_VERIFY_OK: {count} tracked files (manifest excluded); not authority evidence")
    return 0


if __name__ == "__main__":
    sys.exit(main())
