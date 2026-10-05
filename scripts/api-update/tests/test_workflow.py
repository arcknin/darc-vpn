"""Run shell lifecycle against fake Docker and temporary data, never real containers."""
import json
import os
import pathlib
import subprocess
import sys
import tempfile
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[3]
NEW = 'sha256:' + 'c' * 64


class LifecycleTests(unittest.TestCase):
    def exercise(self, fail_health=False, verified=True):
        with tempfile.TemporaryDirectory() as temp:
            root = pathlib.Path(temp)
            scripts = root / 'scripts/api-update'
            scripts.mkdir(parents=True)
            infra = root / 'infra/wireguard-production'
            infra.mkdir(parents=True)
            (infra / 'compose.update-api.yaml').write_text((ROOT / 'infra/wireguard-production/compose.update-api.yaml').read_text())
            state = root / 'state'
            state.mkdir()
            (state / 'stage').write_text('verified' if verified else 'awaiting-client-verification')
            data = root / 'data'
            data.mkdir()
            (data / 'vpn.db').write_text('fixture database')
            (data / 'master.hex').write_text('fixture key')
            original = {'services': {
                'api': {'image': 'sha256:' + 'a' * 64, 'network_mode': 'service:wireguard',
                        'environment': {'VPN_DB_PATH': '/data/vpn.db', 'SECRET': 'fixture-secret'},
                        'volumes': [{'type': 'bind', 'source': str(data), 'target': '/data'}]},
                'wireguard': {'image': 'sha256:' + 'b' * 64}}}
            base = state / 'runtime-model.json'
            base.write_text(json.dumps(original))
            base.chmod(0o600)
            # Substitute privilege/path boundaries only in isolated harness copies.
            helper = (ROOT / 'scripts/api-update/model.py').read_text().replace('info.st_uid != 0', 'info.st_uid != os.getuid()')
            (scripts / 'model.py').write_text(helper)
            shell = (ROOT / 'scripts/api-update/update.sh').read_text()
            guard = '[[ $EUID -eq 0 ]] || { echo \'Execute como root.\' >&2; exit 1; }'
            self.assertIn(guard, shell)
            shell = shell.replace(guard, 'true').replace('STATE=/var/lib/darc-vpn-migration', f'STATE={state}')
            (scripts / 'update.sh').write_text(shell)
            tools = root / 'bin'
            tools.mkdir()
            (tools / 'flock').write_text('#!/bin/sh\nexit 0\n')
            docker = tools / 'docker'
            docker.write_text(f'#!{sys.executable}\n' + '''import json, os, sys
a = sys.argv[1:]
with open(os.environ['CALL_LOG'], 'a') as f: f.write(json.dumps(a) + '\\n')
if a[0] == 'inspect':
    print('sha256:' + 'a'*64 if a[-1] == 'api-id' else 'unchanged-start-time')
elif 'ps' in a:
    print('api-id' if a[-1] == 'api' else 'wg-id')
elif 'exec' in a and os.environ.get('FAIL_HEALTH') == '1':
    sys.exit(1)
''')
            for tool in tools.iterdir():
                tool.chmod(0o700)
            log = root / 'calls.jsonl'
            env = dict(os.environ, PATH=str(tools) + os.pathsep + os.environ['PATH'],
                       CALL_LOG=str(log), FAIL_HEALTH='1' if fail_health else '0')
            result = subprocess.run(['bash', str(scripts / 'update.sh'), '--image', NEW], env=env,
                                    capture_output=True, text=True, timeout=15)
            calls = [json.loads(line) for line in log.read_text().splitlines()] if log.exists() else []
            after = json.loads(base.read_text())
            backups = list(state.glob('api-update.*/data-before.tar.gz'))
            self.assertNotIn('fixture-secret', result.stdout + result.stderr)
            for call in calls:
                if 'stop' in call or 'up' in call:
                    self.assertEqual(call[-1], 'api')
                if 'up' in call:
                    self.assertIn('--no-deps', call)
                    self.assertIn('--no-build', call)
                    self.assertEqual(call[call.index('--pull') + 1], 'never')
            return result, original, after, calls, bool(backups)

    def test_success_updates_only_api_and_preserves_backup(self):
        result, before, after, _, backup = self.exercise()
        self.assertEqual(result.returncode, 0, result.stderr)
        before['services']['api']['image'] = NEW
        self.assertEqual(before, after)
        self.assertTrue(backup)

    def test_failed_api_returns_old_image_without_restoring_database(self):
        result, before, after, calls, backup = self.exercise(fail_health=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(before, after)
        self.assertEqual(sum('up' in call for call in calls), 2)
        self.assertTrue(backup)
        self.assertIn('dados não foram restaurados', result.stderr)

    def test_unverified_migration_does_not_touch_containers(self):
        result, before, after, calls, backup = self.exercise(verified=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(before, after)
        self.assertEqual(calls, [])
        self.assertFalse(backup)
