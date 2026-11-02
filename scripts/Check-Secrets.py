"""Bounded source/history credential signatures; never emit paths or matched bytes.

A supplemental high-confidence check, not a general-purpose secret detector.
Reachable local Git blobs and current nonignored files are inspected. Fixed
signature classes, explicit exclusions and fail-closed operational limits apply.
"""
import os
import re
import stat
import subprocess
import sys
import threading
import time
from dataclasses import dataclass, field
from pathlib import Path

PATTERNS = {
    "github_token": re.compile(rb"\b(?:gh[pousr]_[A-Za-z0-9]{36,255}|github_pat_[A-Za-z0-9_]{50,255})\b"),
    "aws_access_id": re.compile(rb"\b(?:AKIA|ASIA)[A-Z0-9]{16}\b"),
    "private_key": re.compile(rb"(?m)^-----BEGIN (?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY-----\r?$"),
}
MAX_BLOB = 8 << 20
MAX_TOTAL = 512 << 20
MAX_INPUTS = 100000
MAX_GIT_OUTPUT = 16 << 20
OID = re.compile(rb"[a-f0-9]{40}(?:[a-f0-9]{24})?\Z")


class ScanError(Exception):
    """Only fixed internal reason labels reach the CLI."""


MAX_GIT_SECONDS = 30


def git_command(root, *args):
    # Read-only Git operations need neither repository hooks nor fsmonitor. Blob
    # reads intentionally never request --filters/--textconv or external diffs.
    return ["git", "--no-optional-locks", "-c", "core.fsmonitor=false", "-c",
            "core.hooksPath=" + os.devnull, "-C", str(root), *args]


def git_bytes(root, *args, limit=MAX_GIT_OUTPUT, input_data=None):
    process = None
    threads = []
    output = []
    errors = []
    done = threading.Event()
    deadline = time.monotonic() + MAX_GIT_SECONDS
    try:
        environment = dict(os.environ, GIT_CONFIG_NOSYSTEM="1", GIT_CONFIG_GLOBAL=os.devnull, GIT_NO_REPLACE_OBJECTS="1")
        process = subprocess.Popen(git_command(root, *args), stdin=subprocess.PIPE if input_data is not None else subprocess.DEVNULL,
                                   stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, bufsize=0, env=environment)

        def receive():
            try:
                data = bytearray()
                while len(data) <= limit:
                    chunk = process.stdout.read(min(65536, limit + 1 - len(data)))
                    if not chunk:
                        break
                    data.extend(chunk)
                output.append(bytes(data))
            except Exception:
                errors.append(True)
            finally:
                done.set()

        reader = threading.Thread(target=receive, daemon=True)
        threads.append(reader)
        reader.start()
        if input_data is not None:
            def send():
                try:
                    pending = memoryview(input_data)
                    while pending:
                        count = process.stdin.write(pending)
                        if not count:
                            raise OSError()
                        pending = pending[count:]
                except Exception:
                    errors.append(True)
                finally:
                    process.stdin.close()
            writer = threading.Thread(target=send, daemon=True)
            threads.append(writer)
            writer.start()
        if not done.wait(max(0, deadline - time.monotonic())):
            raise ScanError("git_timeout")
        if errors or not output:
            raise ScanError("git_read_failure")
        if len(output[0]) > limit:
            raise ScanError("git_listing_limit")
        if process.wait(timeout=max(0.001, deadline - time.monotonic())) != 0:
            raise ScanError("git_failure")
        return output[0]
    except subprocess.TimeoutExpired:
        raise ScanError("git_timeout") from None
    except (OSError, subprocess.SubprocessError):
        raise ScanError("git_failure") from None
    finally:
        if process is not None:
            if process.poll() is None:
                process.kill()
            process.wait(timeout=5)
            for thread in threads:
                thread.join(timeout=1)
            # Unbuffered pipes have no BufferedReader lock held by a blocked read.
            process.stdout.close()
            if process.stdin is not None and not process.stdin.closed:
                process.stdin.close()


def linked(info):
    return stat.S_ISLNK(info.st_mode) or bool(getattr(info, "st_file_attributes", 0) & 0x400)


def read_working(path, root):
    current = path
    while True:
        info = current.lstat()
        if linked(info):
            return None, "linked"
        if current == root:
            break
        current = current.parent
    before = path.lstat()
    if not stat.S_ISREG(before.st_mode):
        return None, "other"
    if before.st_size > MAX_BLOB:
        return None, "large"
    flags = os.O_RDONLY | getattr(os, "O_BINARY", 0) | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_NONBLOCK", 0)
    with os.fdopen(os.open(path, flags), "rb") as stream:
        opened = os.fstat(stream.fileno())
        if not stat.S_ISREG(opened.st_mode) or (opened.st_dev, opened.st_ino) != (before.st_dev, before.st_ino):
            raise ScanError("working_input_changed")
        data = stream.read(MAX_BLOB + 1)
        after = os.fstat(stream.fileno())
    if len(data) != before.st_size or (after.st_size, after.st_mtime_ns) != (before.st_size, before.st_mtime_ns):
        raise ScanError("working_input_changed")
    return data, None


@dataclass
class Result:
    scanned: int = 0
    inputs: int = 0
    bytes_read: int = 0
    excluded: dict = field(default_factory=lambda: {"binary": 0, "large": 0, "linked": 0, "other": 0})
    findings: list = field(default_factory=list)

    def add(self, data, label, exclusion=None):
        self.inputs += 1
        if self.inputs > MAX_INPUTS:
            raise ScanError("input_count_limit")
        if exclusion:
            self.excluded[exclusion] += 1
            return
        self.bytes_read += len(data)
        if self.bytes_read > MAX_TOTAL:
            raise ScanError("input_byte_limit")
        if len(data) > MAX_BLOB:
            raise ScanError("input_size_changed")
        if b"\0" in data:
            self.excluded["binary"] += 1
            return
        self.scanned += 1
        for name, pattern in PATTERNS.items():
            if pattern.search(data):
                self.findings.append((label, name))


def scan_repository(root="."):
    root = Path(root).absolute()
    if linked(root.lstat()):
        raise ScanError("linked_repository_root")
    result = Result()
    listing = git_bytes(root, "rev-list", "--objects", "--all", "--no-object-names")
    ids = list(dict.fromkeys(listing.splitlines()))
    if len(ids) > MAX_INPUTS or any(OID.fullmatch(oid) is None for oid in ids):
        raise ScanError("invalid_or_excess_git_inventory")
    # Metadata output is format-bounded by the validated ID count/length. No
    # object paths are requested, and no Git diagnostic text is forwarded.
    metadata = git_bytes(root, "cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)",
                         input_data=b"\n".join(ids) + b"\n") if ids else b""
    if len(metadata) > MAX_GIT_OUTPUT or len(metadata.splitlines()) != len(ids):
        raise ScanError("invalid_git_metadata")
    for oid, row in zip(ids, metadata.splitlines()):
        fields = row.split()
        if len(fields) != 3 or fields[0] != oid or not fields[2].isdigit():
            raise ScanError("invalid_git_metadata")
        _, kind, raw_size = fields
        if kind != b"blob":
            continue
        size = int(raw_size)
        if size > MAX_BLOB:
            result.add(None, None, "large")
            continue
        data = git_bytes(root, "cat-file", "blob", oid.decode("ascii"), limit=MAX_BLOB)
        if len(data) != size:
            raise ScanError("invalid_git_blob_size")
        result.add(data, oid.decode("ascii"))
    paths = git_bytes(root, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
    if paths and not paths.endswith(b"\0"):
        raise ScanError("invalid_working_inventory")
    paths = list(dict.fromkeys(path for path in paths.split(b"\0") if path))
    if len(paths) > MAX_INPUTS:
        raise ScanError("input_count_limit")
    for raw in paths:
        relative = Path(os.fsdecode(raw))
        if relative.is_absolute() or ".." in relative.parts:
            raise ScanError("invalid_working_inventory")
        try:
            data, exclusion = read_working(root / relative, root)
        except OSError:
            raise ScanError("working_input_unreadable") from None
        result.add(data, "working-file", exclusion)
    return result


def main(root="."):
    try:
        result = scan_repository(root)
    except ScanError as error:
        print("FAIL: source/history signature scan incomplete:", str(error))
        return 2
    except Exception:
        print("FAIL: source/history signature scan incomplete: inspection_failed")
        return 2
    print(f"Credential signatures: {len(result.findings)} findings; {result.scanned} text inputs scanned; {result.bytes_read} input bytes read.")
    print("Excluded inputs:", ", ".join(f"{name}={count}" for name, count in result.excluded.items()))
    for label, name in result.findings:
        print(f"REVIEW {name} in {label}")
    print("Coverage excludes arbitrary passwords, unknown token formats, ignored files, artifacts, linked/binary/large inputs and unreachable remote history.")
    return 1 if result.findings else 0


if __name__ == "__main__":
    sys.exit(main())
