import io
import runpy
import tarfile
import tempfile
import unittest
import subprocess
import sys
from unittest.mock import patch
import zipfile
from pathlib import Path

SCOPE = runpy.run_path(str(Path(__file__).resolve().parents[1] / 'scripts/Check-ArtifactPrivacy.py'))
scan, PrivacyError = SCOPE['scan'], SCOPE['PrivacyError']
CANARY = b'private-test-canary-' + b'0123456789abcdef'


class ArtifactPrivacyTests(unittest.TestCase):
    def test_clean_binary_and_source_archives(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            (root / 'worker.exe').write_bytes(b'MZ\0ordinary synthetic bytes')
            with zipfile.ZipFile(root / 'source.zip', 'w') as archive:
                archive.writestr('src/readme.txt', 'public licensed source')
            self.assertGreater(scan([root], [CANARY])['bytes_checked'], 0)

    def test_canaries_in_binary_zip_and_tar_are_detected(self):
        for kind in ('binary', 'zip', 'tar'):
            with self.subTest(kind=kind), tempfile.TemporaryDirectory() as folder:
                path = Path(folder) / ('test.' + {'binary': 'exe', 'zip': 'zip', 'tar': 'tar.gz'}[kind])
                if kind == 'zip':
                    with zipfile.ZipFile(path, 'w', compression=zipfile.ZIP_DEFLATED) as archive:
                        archive.writestr('nested/private.txt', CANARY)
                elif kind == 'tar':
                    with tarfile.open(path, 'w:gz') as archive:
                        member = tarfile.TarInfo('private.txt'); member.size = len(CANARY)
                        archive.addfile(member, io.BytesIO(CANARY))
                else:
                    path.write_bytes(b'MZ\0' + CANARY)
                with self.assertRaisesRegex(PrivacyError, 'private_canary_found'):
                    scan([path], [CANARY])

    def test_nested_zip_and_credential_signature(self):
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / 'outer.zip'
            inner = io.BytesIO()
            with zipfile.ZipFile(inner, 'w') as archive:
                archive.writestr('token.txt', b'ghp_' + b'X' * 36)
            with zipfile.ZipFile(path, 'w') as archive:
                archive.writestr('inner.zip', inner.getvalue())
            with self.assertRaisesRegex(PrivacyError, 'credential_signature_found'):
                scan([path])

    def test_missing_and_corrupt_inputs_fail_closed(self):
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / 'bad.zip'; path.write_bytes(b'not a zip')
            with self.assertRaises(zipfile.BadZipFile): scan([path])
            with self.assertRaisesRegex(PrivacyError, 'package_input_missing'): scan([path.with_name('missing')])


    def test_budget_and_archive_depth_fail_closed(self):
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / 'data.bin'; path.write_bytes(b'public source content' * 4)
            with patch.dict(scan.__globals__, {'MAX_TOTAL': 32}):
                with self.assertRaisesRegex(PrivacyError, 'scan_budget_exceeded'): scan([path])
            data = b'public source'
            for _ in range(4):
                buffer = io.BytesIO()
                with zipfile.ZipFile(buffer, 'w') as archive: archive.writestr('nested.zip', data)
                data = buffer.getvalue()
            path = path.with_name('nested.zip'); path.write_bytes(data)
            with self.assertRaisesRegex(PrivacyError, 'archive_depth_exceeded'): scan([path])

    def test_cli_never_prints_canary_or_private_path(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder); payload = root / 'worker.exe'; payload.write_bytes(CANARY)
            forbidden = root / 'canaries.txt'; forbidden.write_bytes(CANARY + b'\n')
            script = Path(__file__).resolve().parents[1] / 'scripts/Check-ArtifactPrivacy.py'
            result = subprocess.run([sys.executable, str(script), '--forbidden-file', str(forbidden), str(payload)], capture_output=True, text=True, timeout=5)
            self.assertEqual(result.returncode, 1)
            self.assertNotIn(CANARY.decode(), result.stdout + result.stderr)
            self.assertNotIn(str(root), result.stdout + result.stderr)


if __name__ == '__main__':
    unittest.main()
