"""Bounded signature/canary checks over package inputs and nested source archives.

This supplements review; it cannot recognize all private content or credentials.
It prints counts/reasons, never matched bytes, canaries, member names or paths.
"""
import argparse
import io
import runpy
import sys
import tarfile
import zipfile
from pathlib import Path

PATTERNS = runpy.run_path(str(Path(__file__).with_name('Check-Secrets.py')))['PATTERNS']
MAX_ENTRY = 128 << 20
MAX_TOTAL = 512 << 20
MAX_MEMBERS = 50000
MAX_DEPTH = 3


class PrivacyError(Exception):
    pass


def scan(paths, forbidden=()):
    total = members = 0
    values = tuple(forbidden)
    if any(len(v) < 16 or len(v) > 1024 for v in values):
        raise PrivacyError('invalid_canary_length')

    def check(data):
        nonlocal total, members
        total += len(data)
        members += 1
        if len(data) > MAX_ENTRY or total > MAX_TOTAL or members > MAX_MEMBERS:
            raise PrivacyError('scan_budget_exceeded')
        if any(v in data for v in values):
            raise PrivacyError('private_canary_found')
        if any(pattern.search(data) for pattern in PATTERNS.values()):
            raise PrivacyError('credential_signature_found')

    def payload(name, data, depth):
        check(data)
        if name.lower().endswith('.zip'):
            if depth >= MAX_DEPTH:
                raise PrivacyError('archive_depth_exceeded')
            with zipfile.ZipFile(io.BytesIO(data)) as archive:
                for member in archive.infolist():
                    check(member.filename.encode('utf-8', 'replace'))
                    if member.is_dir():
                        continue
                    if member.file_size > MAX_ENTRY:
                        raise PrivacyError('archive_member_too_large')
                    with archive.open(member) as stream:
                        payload(member.filename, stream.read(MAX_ENTRY + 1), depth + 1)
        elif name.lower().endswith(('.tar.gz', '.tgz', '.tar')):
            if depth >= MAX_DEPTH:
                raise PrivacyError('archive_depth_exceeded')
            with tarfile.open(fileobj=io.BytesIO(data), mode='r:*') as archive:
                for member in archive:
                    check((member.name + '\n' + member.linkname).encode('utf-8', 'replace'))
                    if not member.isfile():
                        continue  # Inspect link metadata, never follow or extract it.
                    if member.size > MAX_ENTRY:
                        raise PrivacyError('archive_member_too_large')
                    with archive.extractfile(member) as stream:
                        payload(member.name, stream.read(MAX_ENTRY + 1), depth + 1)

    def visit(path):
        if path.is_symlink() or getattr(path, 'is_junction', lambda: False)():
            raise PrivacyError('linked_package_input_rejected')
        if path.is_dir():
            for child in path.iterdir():
                visit(child)
        elif path.is_file():
            check(path.name.encode('utf-8', 'replace'))
            if path.stat().st_size > MAX_ENTRY:
                raise PrivacyError('package_input_too_large')
            with path.open('rb') as stream:
                payload(path.name, stream.read(MAX_ENTRY + 1), 0)
        else:
            raise PrivacyError('package_input_missing')

    for path in paths:
        visit(Path(path))
    return {'inputs_and_members': members, 'bytes_checked': total}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--forbidden-file', type=Path)
    parser.add_argument('paths', nargs='+')
    args = parser.parse_args()
    try:
        forbidden = ()
        if args.forbidden_file:
            if args.forbidden_file.stat().st_size > 8192:
                raise PrivacyError('canary_file_too_large')
            forbidden = tuple(v for v in args.forbidden_file.read_bytes().splitlines() if v)
        result = scan(args.paths, forbidden)
        print('PASS: package signature/canary scan', result)
        print('Coverage: literal supplied canaries and selected credential signatures; not a complete private-content audit.')
        return 0
    except PrivacyError as error:
        print('FAIL: package privacy check:', str(error))
    except Exception:
        print('FAIL: package privacy check could not inspect every input')
    return 1


if __name__ == '__main__':
    sys.exit(main())
