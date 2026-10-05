import 'package:flutter_test/flutter_test.dart';

import 'package:darc_vpn/src/api/friend_api.dart';
import 'package:darc_vpn/src/models.dart';
import 'package:darc_vpn/src/qr_config.dart';
import 'package:darc_vpn/src/session_controller.dart';
import 'package:darc_vpn/src/storage.dart';

void main() {
  group('B06 friend session flow', () {
    test(
      'does not request profiles or secrets before terms acceptance',
      () async {
        final api = FakeFriendApi();
        final controller = FriendSessionController(
          api: api,
          tokenStore: MemoryTokenStore(),
          selectionStore: MemorySelectionStore(),
        );

        await controller.signIn('friend', 'correct-password');

        expect(controller.status, SessionStatus.termsRequired);
        expect(api.listProfilesCalls, 0);
        expect(api.privateProfileCalls, 0);
        expect(
          () => controller.revealProfile('primary'),
          throwsA(isA<StateError>()),
        );
      },
    );

    test(
      'keeps the gate closed when acceptance persistence fails and allows retry',
      () async {
        final api = FakeFriendApi()..failAcceptance = true;
        final controller = FriendSessionController(
          api: api,
          tokenStore: MemoryTokenStore(),
          selectionStore: MemorySelectionStore(),
        );
        await controller.signIn('friend', 'correct-password');

        await expectLater(controller.acceptTerms(), throwsA(isA<ApiFailure>()));
        expect(controller.status, SessionStatus.termsRequired);
        expect(controller.termsAccepted, isFalse);
        expect(api.listProfilesCalls, 0);

        api.failAcceptance = false;
        await controller.acceptTerms();
        expect(controller.status, SessionStatus.ready);
        expect(controller.termsAccepted, isTrue);
        expect(api.listProfilesCalls, 1);
      },
    );

    test(
      'selecting a profile persists only its id and never connects',
      () async {
        final selectionStore = MemorySelectionStore(
          initialValue: 'removed-profile',
        );
        final api = FakeFriendApi()
          ..profiles = <ProfileSummary>[
            _profile(id: 'primary', displayName: 'Principal', isPrimary: true),
            _profile(id: 'extra', displayName: 'Notebook'),
          ];
        final controller = FriendSessionController(
          api: api,
          tokenStore: MemoryTokenStore(),
          selectionStore: selectionStore,
        );

        await controller.signIn('friend', 'correct-password');
        await controller.acceptTerms();
        expect(controller.selectedProfile?.id, 'primary');
        expect(selectionStore.value, 'primary');

        await controller.selectProfile('extra');
        expect(controller.selectedProfile?.id, 'extra');
        expect(selectionStore.value, 'extra');
        expect(api.connectionCalls, 0);
        expect(selectionStore.writes, ['primary', 'extra']);
      },
    );

    test(
      'reuses the selected profile private material for QR without new identity',
      () async {
        final private = ProfilePrivate(
          id: 'extra',
          displayName: 'Notebook',
          endpoint: 'vpn.darcanix.com',
          address: '10.0.0.3/32',
          allowedIPs: const ['0.0.0.0/0', '::/0'],
          dns: const ['1.1.1.1'],
          peerPublicKey: 'peer-public-extra',
          serverPublicKey: 'server-public-key',
          endpointPort: 51442,
          privateKey: 'private-extra',
          presharedKey: 'preshared-extra',
        );
        final api = FakeFriendApi()
          ..profiles = <ProfileSummary>[
            _profile(id: 'primary', displayName: 'Principal', isPrimary: true),
            _profile(id: 'extra', displayName: 'Notebook'),
          ]
          ..privateById['extra'] = private;
        final controller = FriendSessionController(
          api: api,
          tokenStore: MemoryTokenStore(),
          selectionStore: MemorySelectionStore(),
        );
        await controller.signIn('friend', 'correct-password');
        await controller.acceptTerms();
        final recovered = await controller.revealProfile('extra');
        final qr = WireGuardQrConfig.fromProfile(recovered);

        expect(qr.data, contains('PrivateKey = private-extra'));
        expect(qr.data, contains('PresharedKey = preshared-extra'));
        expect(qr.data, contains('PublicKey = server-public-key'));
        expect(qr.profileId, private.id);
        expect(qr.peerPublicKey, private.peerPublicKey);
        expect(api.privateProfileCalls, 1);
        expect(qr.warning, contains('uso simultâneo'));
      },
    );

    test(
      'keeps the bearer token out of the common profile selection store',
      () async {
        final tokenStore = MemoryTokenStore();
        final selectionStore = MemorySelectionStore();
        final controller = FriendSessionController(
          api: FakeFriendApi(),
          tokenStore: tokenStore,
          selectionStore: selectionStore,
        );

        await controller.signIn('friend', 'correct-password');
        await controller.acceptTerms();

        expect(tokenStore.value, 'token-secret');
        expect(selectionStore.value, 'primary');
        expect(selectionStore.writes, everyElement(isNot('token-secret')));
        expect(
          selectionStore.writes,
          everyElement(isNot(contains('PrivateKey'))),
        );
        expect(
          selectionStore.writes,
          everyElement(isNot(contains('PresharedKey'))),
        );
      },
    );

    test(
      'clears any session material and exposes a stable blocked-account error',
      () async {
        final tokenStore = MemoryTokenStore(initialValue: 'stale-token');
        final api = FakeFriendApi()
          ..loginFailure = const ApiFailure(
            code: 'account_blocked',
            message: 'conta bloqueada',
            statusCode: 423,
          );
        final controller = FriendSessionController(
          api: api,
          tokenStore: tokenStore,
          selectionStore: MemorySelectionStore(),
        );

        await expectLater(
          controller.signIn('friend', 'correct-password'),
          throwsA(isA<ApiFailure>()),
        );

        expect(controller.status, SessionStatus.error);
        expect(controller.failure?.code, 'account_blocked');
        expect(tokenStore.value, isNull);
      },
    );
  });
}

ProfileSummary _profile({
  required String id,
  required String displayName,
  bool isPrimary = false,
}) {
  return ProfileSummary(
    id: id,
    displayName: displayName,
    isPrimary: isPrimary,
    endpoint: 'vpn.darcanix.com',
    peerPublicKey: 'peer-$id',
    address: '10.0.0.2/32',
    allowedIPs: const ['0.0.0.0/0'],
    dns: const ['1.1.1.1'],
  );
}

class FakeFriendApi implements FriendApi {
  bool failAcceptance = false;
  ApiFailure? loginFailure;
  int listProfilesCalls = 0;
  int privateProfileCalls = 0;
  int connectionCalls = 0;
  List<ProfileSummary> profiles = <ProfileSummary>[
    _profile(id: 'primary', displayName: 'Principal', isPrimary: true),
  ];
  final Map<String, ProfilePrivate> privateById = <String, ProfilePrivate>{};

  @override
  Future<TermsAcceptanceReceipt> acceptTerms({
    required String version,
    required String sha256,
  }) async {
    if (failAcceptance) {
      throw const ApiFailure(
        code: 'acceptance_persistence_failed',
        message: 'retry',
      );
    }
    return TermsAcceptanceReceipt(
      accepted: true,
      accountId: 'account-1',
      version: version,
      sha256: sha256,
      acceptedAt: DateTime.utc(2026, 1, 1),
      eventId: 'event-1',
    );
  }

  @override
  Future<TermsDocument> currentTerms() async => const TermsDocument(
    version: 'initial-1',
    text: 'Termos completos para teste.',
    sha256: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
  );

  @override
  Future<List<ProfileSummary>> listProfiles() async {
    listProfilesCalls++;
    return profiles;
  }

  @override
  Future<ProfilePrivate> recoverPrivateProfile(String profileId) async {
    privateProfileCalls++;
    return privateById[profileId] ??
        ProfilePrivate(
          id: profileId,
          displayName: 'Principal',
          endpoint: 'vpn.darcanix.com',
          address: '10.0.0.2/32',
          allowedIPs: const ['0.0.0.0/0'],
          dns: const ['1.1.1.1'],
          peerPublicKey: 'peer-$profileId',
          serverPublicKey: 'server-public-key',
          endpointPort: 51442,
          privateKey: 'private-$profileId',
        );
  }

  @override
  Future<LoginSession> login(String username, String password) async {
    if (loginFailure != null) throw loginFailure!;
    return LoginSession(
      accessToken: 'token-secret',
      expiresAt: DateTime.utc(2026, 12, 31),
      accountId: 'account-1',
      role: 'friend',
    );
  }

  @override
  Future<void> logout() async {}
}
