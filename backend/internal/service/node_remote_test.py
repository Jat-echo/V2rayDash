import hashlib
import importlib.util
import json
import os
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch


spec = importlib.util.spec_from_file_location('node_remote', Path(__file__).with_name('node_remote.py'))
remote = importlib.util.module_from_spec(spec)
spec.loader.exec_module(remote)


class PublishTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='v2raydash-publish-test-')
        self.root = Path(self.temp.name)
        self.config = self.root / 'config.json'
        self.original = json.dumps({'inbounds': [{'tag': 'entry', 'port': 443, 'protocol': 'vless', 'streamSettings': {'security': 'reality'}}], 'outbounds': [{'protocol': 'blackhole'}]}).encode()
        self.config.write_bytes(self.original)
        self.binary = self.root / 'xray'
        self.binary.touch()
        self.calls = []
        self.reject = False
        self.patches = [patch.object(remote, 'LAYOUTS', {'exit': (str(self.config), str(self.binary), 'test-core')}), patch.object(remote, 'BACKUP_ROOT', self.root / 'backups'), patch.object(remote.os, 'chown', create=True), patch.object(remote.subprocess, 'run', side_effect=self.run_command)]
        for item in self.patches:
            item.start()

    def tearDown(self):
        for item in reversed(self.patches):
            item.stop()
        self.temp.cleanup()

    def run_command(self, args, **kwargs):
        self.calls.append(args)
        return SimpleNamespace(returncode=1 if self.reject and args[0] == str(self.binary) else 0)

    def request(self):
        candidate = json.loads(self.original)
        candidate['log'] = {'loglevel': 'warning'}
        return {'expected_hash': hashlib.sha256(self.original).hexdigest(), 'config': candidate, 'probes': []}

    def test_invalid_candidate_does_not_restart_or_replace(self):
        self.reject = True
        with self.assertRaisesRegex(ValueError, 'rejected'):
            remote.publish(remote.inspect('exit'), self.request())
        self.assertEqual(self.config.read_bytes(), self.original)
        self.assertFalse(any(call[0] == 'systemctl' for call in self.calls))

    def test_legacy_forwarder_exposes_public_port_without_changing_reality(self):
        config = json.loads(self.original)
        config['inbounds'][0].update(listen='127.0.0.1', port=45987)
        config['inbounds'].append({'protocol': 'dokodemo-door', 'port': 14143, 'settings': {'address': '127.0.0.1', 'port': 45987, 'network': 'tcp'}})
        self.config.write_text(json.dumps(config))
        original = self.config.read_bytes()
        info = remote.inspect('exit')
        self.assertEqual(info['public_port'], 14143)
        self.assertEqual(info['inbound']['port'], 45987)
        self.assertEqual(self.config.read_bytes(), original)
        config['inbounds'].append({'protocol': 'dokodemo-door', 'port': 14144, 'settings': {'address': '127.0.0.1', 'port': 45987}})
        self.config.write_text(json.dumps(config))
        with self.assertRaisesRegex(ValueError, 'unique public port'):
            remote.inspect('exit')

    def test_failed_https_verification_restores_previous_config(self):
        with patch.object(remote, 'probe', side_effect=ValueError('HTTPS failed')):
            with self.assertRaisesRegex(ValueError, 'HTTPS failed'):
                remote.publish(remote.inspect('exit'), self.request())
        self.assertEqual(self.config.read_bytes(), self.original)
        restarts = [call for call in self.calls if call[:2] == ['systemctl', 'restart']]
        self.assertEqual(len(restarts), 2)

    def test_success_returns_hash_and_keeps_private_backup(self):
        with patch.object(remote, 'probe', return_value=[{'exit_ip': '192.0.2.10'}]):
            result = remote.publish(remote.inspect('exit'), self.request())
        self.assertEqual(hashlib.sha256(self.config.read_bytes()).hexdigest(), result['hash'])
        self.assertEqual(Path(result['backup']).read_bytes(), self.original)
        if os.name != 'nt':
            self.assertEqual(Path(result['backup']).stat().st_mode & 0o077, 0)

    def test_core_validation_reads_json_candidate_for_every_layout(self):
        for layout in ('relay', 'exit', 'legacy'):
            with self.subTest(layout=layout):
                self.config.write_bytes(self.original)
                request = self.request()

                def validate(args, **kwargs):
                    if args[0] == str(self.binary):
                        if '-confdir' in args:
                            directory = Path(args[args.index('-confdir') + 1])
                            files = list(directory.glob('*.json'))
                            self.assertEqual(len(files), 1)
                            candidate = files[0]
                        else:
                            candidate = Path(args[args.index('-config') + 1])
                        self.assertEqual(candidate.suffix, '.json', 'Xray requires a recognized config extension')
                        self.assertEqual(json.loads(candidate.read_text()), request['config'])
                        self.assertEqual(self.config.read_bytes(), self.original, 'validation must precede replacement')
                    return self.run_command(args, **kwargs)

                with patch.dict(remote.LAYOUTS, {layout: (str(self.config), str(self.binary), 'test-core')}), patch.object(remote.subprocess, 'run', side_effect=validate), patch.object(remote, 'probe', return_value=[]):
                    remote.publish(remote.inspect(layout), request)

    def test_drift_is_rejected_before_writing(self):
        request = self.request()
        request['expected_hash'] = 'stale'
        with self.assertRaisesRegex(ValueError, 'changed'):
            remote.publish(remote.inspect('exit'), request)
        self.assertEqual(self.config.read_bytes(), self.original)
        self.assertEqual(self.calls, [])

    def test_database_failure_can_restore_only_an_unchanged_publication(self):
        with patch.object(remote, 'probe', return_value=[]):
            published = remote.publish(remote.inspect('exit'), self.request())
        with self.assertRaisesRegex(ValueError, 'changed'):
            remote.rollback(remote.inspect('exit'), {'expected_hash': 'stale', 'backup': published['backup']})
        result = remote.rollback(remote.inspect('exit'), {'expected_hash': published['hash'], 'backup': published['backup']})
        self.assertEqual(self.config.read_bytes(), self.original)
        self.assertEqual(result['hash'], hashlib.sha256(self.original).hexdigest())

    def test_rollback_rejects_paths_outside_the_private_backup_directory(self):
        with self.assertRaisesRegex(ValueError, 'Invalid backup'):
            remote.rollback(remote.inspect('exit'), {'expected_hash': self.request()['expected_hash'], 'backup': str(self.config)})


if __name__ == '__main__':
    unittest.main()
