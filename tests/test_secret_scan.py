import contextlib
import importlib.util
import io
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

SCRIPT = Path(__file__).resolve().parents[1] / 'scripts' / 'Check-Secrets.py'
spec = importlib.util.spec_from_file_location('source_secret_scan', SCRIPT)
scanner = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = scanner
spec.loader.exec_module(scanner)


class SecretScanTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.git('init', '-q')
        self.git('config', 'user.email', 'fixture@example.invalid')
        self.git('config', 'user.name', 'Synthetic Fixture')
        (self.root / 'clean.txt').write_text('clean fixture')
        self.git('add', '.')
        self.git('commit', '-qm', 'fixture')

    def git(self, *args):
        return subprocess.check_output(['git', '-C', str(self.root), *args], stderr=subprocess.DEVNULL)

    def output(self):
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            status = scanner.main(self.root)
        return status, output.getvalue()

    def test_current_untracked_detection_and_redaction(self):
        token = 'ghp_' + 'A' * 36
        private_name = 'private-path-do-not-print.txt'
        (self.root / private_name).write_text(token)
        status, output = self.output()
        self.assertEqual(status, 1)
        self.assertIn('github_token in working-file', output)
        self.assertNotIn(token, output)
        self.assertNotIn(private_name, output)
        self.assertNotIn(str(self.root), output)

    def test_deleted_history_is_inspected_and_only_oid_exposed(self):
        token = 'AKIA' + 'Z' * 16
        path = self.root / 'private-history-name.txt'
        path.write_text(token)
        self.git('add', '.')
        self.git('commit', '-qm', 'synthetic old state')
        oid = self.git('rev-parse', 'HEAD:private-history-name.txt').decode().strip()
        self.git('rm', '-q', 'private-history-name.txt')
        self.git('commit', '-qm', 'removed')
        status, output = self.output()
        self.assertEqual(status, 1)
        self.assertIn('aws_access_id in ' + oid, output)
        self.assertNotIn(token, output)
        self.assertNotIn(path.name, output)

    def test_all_signature_classes_and_clean_control(self):
        result = scanner.Result()
        result.add(b'ordinary harmless data', 'working-file')
        self.assertEqual(result.findings, [])
        values = [b'ghp_' + b'A' * 36, b'ASIA' + b'Z' * 16, b'-----BEGIN ' + b'RSA PRIVATE KEY-----\n']
        for value in values:
            result.add(value, 'working-file')
        self.assertEqual({name for _, name in result.findings}, set(scanner.PATTERNS))

    def test_exclusions_are_counted_and_ignored_files_are_not_claimed(self):
        (self.root / '.gitignore').write_text('ignored-secret.txt\n')
        (self.root / 'ignored-secret.txt').write_text('ghp_' + 'A' * 36)
        (self.root / 'binary.bin').write_bytes(b'\x00' + b'AKIA' + b'Z' * 16)
        (self.root / 'large.bin').write_bytes(b'X' * 129)
        with patch.object(scanner, 'MAX_BLOB', 128):
            result = scanner.scan_repository(self.root)
        self.assertEqual(result.findings, [])
        self.assertEqual(result.excluded['binary'], 1)
        self.assertEqual(result.excluded['large'], 1)

    def test_working_symlink_is_not_followed(self):
        outside = self.root.parent / (self.root.name + '-outside.txt')
        outside.write_text('ghp_' + 'A' * 36)
        self.addCleanup(lambda: outside.unlink(missing_ok=True))
        try:
            (self.root / 'linked.txt').symlink_to(outside)
        except OSError:
            self.skipTest('fixture cannot create symlink')
        result = scanner.scan_repository(self.root)
        self.assertEqual(result.findings, [])
        self.assertEqual(result.excluded['linked'], 1)

    def test_working_read_failure_is_fail_closed_and_redacted(self):
        with patch.object(scanner, 'read_working', side_effect=PermissionError('sensitive-error-and-path')):
            status, output = self.output()
        self.assertEqual(status, 2)
        self.assertIn('working_input_unreadable', output)
        self.assertNotIn('sensitive-error-and-path', output)
        self.assertNotIn(str(self.root), output)

    def test_total_and_inventory_budgets_fail_closed(self):
        with patch.object(scanner, 'MAX_TOTAL', 1):
            status, output = self.output()
        self.assertEqual(status, 2)
        self.assertIn('input_byte_limit', output)
        with patch.object(scanner, 'MAX_INPUTS', 1):
            status, output = self.output()
        self.assertEqual(status, 2)
        self.assertIn('invalid_or_excess_git_inventory', output)

    def test_git_failure_has_no_raw_stderr_or_traceback(self):
        with tempfile.TemporaryDirectory() as directory:
            result = subprocess.run([sys.executable, str(SCRIPT)], cwd=directory, capture_output=True, text=True)
        self.assertEqual(result.returncode, 2)
        self.assertIn('git_failure', result.stdout)
        self.assertEqual(result.stderr, '')
        self.assertNotIn(directory, result.stdout)

    def test_unborn_repository_still_scans_untracked_input(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            subprocess.run(['git', '-C', str(root), 'init', '-q'], check=True)
            (root / 'untracked.txt').write_text('ghp_' + 'A' * 36)
            result = scanner.scan_repository(root)
        self.assertEqual(result.findings, [('working-file', 'github_token')])

    def test_blocked_git_has_real_whole_operation_timeout(self):
        command = [sys.executable, '-c', 'import time; time.sleep(20)']
        import time
        began = time.monotonic()
        with patch.object(scanner, 'git_command', return_value=command), patch.object(scanner, 'MAX_GIT_SECONDS', .2):
            with self.assertRaisesRegex(scanner.ScanError, '^git_timeout$'):
                scanner.git_bytes(self.root, 'status')
        self.assertLess(time.monotonic() - began, 5)

    def test_git_output_cap_and_hook_override(self):
        command = [sys.executable, '-c', 'import sys; sys.stdout.buffer.write(b"X" * 4096)']
        with patch.object(scanner, 'git_command', return_value=command):
            with self.assertRaisesRegex(scanner.ScanError, '^git_listing_limit$'):
                scanner.git_bytes(self.root, 'status', limit=32)
        args = scanner.git_command(self.root, 'ls-files')
        self.assertIn('core.fsmonitor=false', args)
        self.assertIn('core.hooksPath=' + os.devnull, args)
        hook = self.root / 'fsmonitor.py'
        marker = self.root / 'hook-ran'
        hook.write_text('#!' + Path(sys.executable).as_posix() + '\nimport pathlib,sys; pathlib.Path(' + repr(str(marker)) + ').write_text("unexpected"); sys.stdout.buffer.write(b"token\\0")\n')
        hook.chmod(0o755)
        self.git('config', 'core.fsmonitor', str(hook))
        self.git('status', '--porcelain')
        self.assertTrue(marker.exists(), 'configured fixture hook did not execute in positive control')
        marker.unlink()
        scanner.scan_repository(self.root)
        self.assertFalse(marker.exists())

    def test_clean_repository_passes(self):
        status, output = self.output()
        self.assertEqual(status, 0)
        self.assertIn('0 findings', output)
        self.assertNotIn(str(self.root), output)


if __name__ == '__main__':
    unittest.main()
