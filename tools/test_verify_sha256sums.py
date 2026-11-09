"""Adversarial tests for the repository-wide tracked-file checksum gate."""

import hashlib
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent))
from verify_sha256sums import VerificationError, verify


class ChecksumManifestTests(unittest.TestCase):
    def setUp(self):
        self.work = tempfile.TemporaryDirectory()
        self.addCleanup(self.work.cleanup)
        self.root = Path(self.work.name)
        subprocess.run(["git", "init", "-q", str(self.root)], check=True)
        (self.root / "a.txt").write_bytes(b"a\n")
        (self.root / "b.txt").write_bytes(b"b\n")
        self.write_manifest(self.entries())
        subprocess.run(["git", "-C", str(self.root), "add", "--", "a.txt", "b.txt", "SHA256SUMS"], check=True)

    def entries(self):
        return [f"{hashlib.sha256((self.root / name).read_bytes()).hexdigest()}  {name}" for name in ("a.txt", "b.txt")]

    def write_manifest(self, lines):
        (self.root / "SHA256SUMS").write_text("\n".join(lines) + "\n", encoding="utf-8")

    def assert_rejected(self):
        with self.assertRaises(VerificationError):
            verify(self.root)

    def test_valid_and_untracked_files_are_not_in_manifest(self):
        (self.root / "local-secret.txt").write_text("untracked")
        self.assertEqual(verify(self.root), 2)

    def test_hash_drift_is_rejected(self):
        (self.root / "b.txt").write_bytes(b"tampered\n")
        self.assert_rejected()

    def test_missing_path_is_rejected(self):
        self.write_manifest(self.entries()[:1])
        self.assert_rejected()

    def test_duplicate_path_is_rejected(self):
        lines = self.entries()
        self.write_manifest([lines[0], lines[0], lines[1]])
        self.assert_rejected()

    def test_unsorted_path_is_rejected(self):
        self.write_manifest(list(reversed(self.entries())))
        self.assert_rejected()

    def test_untracked_manifest_entry_is_rejected(self):
        self.write_manifest(self.entries() + ["0" * 64 + "  not-tracked.txt"])
        self.assert_rejected()

    def test_malformed_hash_or_line_ending_is_rejected(self):
        self.write_manifest(["A" * 64 + "  a.txt", self.entries()[1]])
        self.assert_rejected()
        (self.root / "SHA256SUMS").write_bytes(("\r\n".join(self.entries()) + "\r\n").encode())
        self.assert_rejected()

    def test_tracked_symlink_is_rejected(self):
        (self.root / "a.txt").unlink()
        (self.root / "a.txt").symlink_to("b.txt")
        self.assert_rejected()


if __name__ == "__main__":
    unittest.main()
