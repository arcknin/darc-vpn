import 'package:flutter_test/flutter_test.dart';
import 'package:darc_vpn/src/models.dart';
import 'package:darc_vpn/src/qr_config.dart';
import 'package:darc_vpn/src/session_controller.dart';
import 'b06_core_test.dart' show FakeFriendApi;
import 'package:darc_vpn/src/storage.dart';

void main() {
  test('private config uses server public key and API endpoint port', () {
    final p = ProfilePrivate.fromJson({
      'id': 'p',
      'displayName': 'Primary',
      'endpoint': 'vpn.darcanix.com',
      'address': '10.66.66.8/32, fd42:42:42::8/128',
      'allowedIPs': ['0.0.0.0/0', '::/0'],
      'dns': ['1.1.1.1'],
      'peerPublicKey': 'client-key',
      'serverPublicKey': 'server-key',
      'endpointPort': 51442,
      'privateKey': 'private-client',
    });
    final config = WireGuardQrConfig.fromProfile(p).data;
    expect(config, contains('PublicKey = server-key'));
    expect(config, contains('Endpoint = vpn.darcanix.com:51442'));
    expect(config, isNot(contains('PublicKey = client-key')));
  });
  test('missing server identity or invalid port fails closed', () {
    final data = {
      'id': 'p',
      'displayName': 'P',
      'endpoint': 'vpn.darcanix.com',
      'address': '10.66.66.8/32',
      'allowedIPs': ['0.0.0.0/0'],
      'dns': ['1.1.1.1'],
      'peerPublicKey': 'client',
      'privateKey': 'private',
    };
    expect(() => ProfilePrivate.fromJson(data), throwsFormatException);
  });
  test('debug private session does not fake terms consent', () async {
    final api = PrivateApi();
    final controller = FriendSessionController(
      api: api,
      tokenStore: MemoryTokenStore(),
      selectionStore: MemorySelectionStore(),
      allowPrivateTest: true,
    );
    await controller.signIn('friend', 'fixture');
    expect(controller.status, SessionStatus.ready);
    expect(controller.termsAccepted, isFalse);
    expect(api.listProfilesCalls, 1);
    await controller.revealProfile('primary');
  });
  test(
    'normal client keeps terms gate even if API signals private mode',
    () async {
      final controller = FriendSessionController(
        api: PrivateApi(),
        tokenStore: MemoryTokenStore(),
        selectionStore: MemorySelectionStore(),
      );
      await controller.signIn('friend', 'fixture');
      expect(controller.status, SessionStatus.termsRequired);
      expect(controller.termsAccepted, isFalse);
    },
  );
  test('malformed acceptanceRequired cannot disable the gate', () {
    expect(
      () => TermsDocument.fromJson({
        'version': 'private-technical-test',
        'text': 'Test',
        'sha256':
            'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
        'acceptanceRequired': 'false',
      }),
      throwsFormatException,
    );
  });
}

class PrivateApi extends FakeFriendApi {
  @override
  Future<TermsDocument> currentTerms() async => const TermsDocument(
    version: 'private-technical-test',
    text: 'Private test',
    sha256: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
    acceptanceRequired: false,
  );
}
