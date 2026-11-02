"""Bounded credential-signature check. Prints counts/IDs only, never matched bytes.

This is a supplemental high-confidence check, not a general-purpose secret scanner.
Scan reachable Git history plus current tracked/untracked nonignored text files.
"""
import re
import subprocess
import sys
from pathlib import Path

PATTERNS = {
    "github_token": re.compile(rb"\b(?:gh[pousr]_[A-Za-z0-9]{36,255}|github_pat_[A-Za-z0-9_]{50,255})\b"),
    "aws_access_id": re.compile(rb"\b(?:AKIA|ASIA)[A-Z0-9]{16}\b"),
    "private_key": re.compile(rb"(?m)^-----BEGIN (?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY-----\r?$"),
}
MAX_BLOB = 8 << 20


def git(*args):
    return subprocess.check_output(["git", *args])


def main():
    objects = git("rev-list", "--objects", "--all").splitlines()
    ids = list(dict.fromkeys(line.split(b" ", 1)[0] for line in objects))
    metadata = subprocess.run(
        ["git", "cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)"],
        input=b"\n".join(ids) + b"\n", stdout=subprocess.PIPE, check=True,
    ).stdout.splitlines()
    findings = []
    skipped = scanned = 0

    def check(data, label):
        nonlocal scanned, skipped
        if len(data) > MAX_BLOB or b"\0" in data:
            skipped += 1
            return
        scanned += 1
        for name, pattern in PATTERNS.items():
            if pattern.search(data):
                findings.append((label, name))

    for row in metadata:
        oid, kind, size = row.split()
        if kind != b"blob":
            continue
        if int(size) > MAX_BLOB:
            skipped += 1
            continue
        check(git("cat-file", "blob", oid.decode()), oid.decode())
    paths = git("ls-files", "-z", "--cached", "--others", "--exclude-standard").split(b"\0")
    for raw in dict.fromkeys(paths):
        if not raw:
            continue
        path = Path(raw.decode())
        if path.is_file() and not path.is_symlink():
            if path.stat().st_size > MAX_BLOB:
                skipped += 1
            else:
                # Do not print paths; a future private checkout may contain them.
                check(path.read_bytes(), "working-file")
    print(f"Credential signatures: {len(findings)} findings; {scanned} text inputs scanned; {skipped} binary/large inputs excluded.")
    for label, name in findings:
        print(f"REVIEW {name} in {label}")
    print("Coverage excludes arbitrary passwords, unknown token formats, ignored files, artifacts and unreachable remote history.")
    return 1 if findings else 0


if __name__ == "__main__":
    sys.exit(main())
