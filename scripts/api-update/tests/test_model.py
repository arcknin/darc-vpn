import copy
import importlib.util
import pathlib
import unittest

spec = importlib.util.spec_from_file_location('model', pathlib.Path(__file__).parents[1] / 'model.py')
model = importlib.util.module_from_spec(spec)
spec.loader.exec_module(model)


def existing():
    return {'services': {
        'api': {'image': 'sha256:' + 'a' * 64, 'network_mode': 'service:wireguard',
                'environment': {'VPN_MODE': 'container-private-test', 'VPN_DB_PATH': '/data/vpn.db', 'PASSWORD': 'fake$$value'},
                'volumes': [{'type': 'bind', 'source': '/opt/darc-vpn-production/data', 'target': '/data'}]},
        'wireguard': {'image': 'sha256:' + 'b' * 64, 'ports': [{'published': '51442', 'protocol': 'udp'}]},
    }, 'networks': {'vpn': {'enable_ipv6': True}}}


class UpdateModelTests(unittest.TestCase):
    def test_changes_only_api_image_preserving_secrets_modes_volumes_and_wg(self):
        old = existing()
        expected = copy.deepcopy(old)
        expected['services']['api']['image'] = 'sha256:' + 'c' * 64
        self.assertEqual(model.prepare(old, expected['services']['api']['image']), expected)
        self.assertEqual(old, existing())

    def test_requires_pinned_image(self):
        with self.assertRaises(ValueError):
            model.prepare(existing(), 'api:latest')

    def test_rejects_missing_persistent_data(self):
        old = existing()
        old['services']['api']['volumes'] = []
        with self.assertRaises(ValueError):
            model.prepare(old, 'sha256:' + 'c' * 64)

    def test_rejects_different_network_namespace(self):
        old = existing()
        old['services']['api']['network_mode'] = 'host'
        with self.assertRaises(ValueError):
            model.prepare(old, 'sha256:' + 'c' * 64)

    def test_rejects_database_outside_backup_volume(self):
        old = existing()
        old['services']['api']['environment']['VPN_DB_PATH'] = '/other/vpn.db'
        with self.assertRaises(ValueError):
            model.prepare(old, 'sha256:' + 'c' * 64)

    def test_rejects_unexpected_stack_service(self):
        old = existing()
        old['services']['dashboard'] = {}
        with self.assertRaises(ValueError):
            model.prepare(old, 'sha256:' + 'c' * 64)
