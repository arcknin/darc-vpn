import 'package:flutter_test/flutter_test.dart';
import 'package:flutter/material.dart';

import 'package:darc_vpn/app.dart';
import 'package:darc_vpn/src/api/friend_api.dart';
import 'package:darc_vpn/src/models.dart';
import 'package:darc_vpn/src/session_controller.dart';
import 'package:darc_vpn/src/storage.dart';

void main() {
  testWidgets('friend must accept terms before profiles and QR are available', (
    tester,
  ) async {
    final api = WidgetFakeApi();
    final controller = FriendSessionController(
      api: api,
      tokenStore: MemoryTokenStore(),
      selectionStore: MemorySelectionStore(),
    );
    await tester.pumpWidget(VpnApp(controller: controller));

    expect(find.text('Entrar como amigo'), findsOneWidget);
    await tester.enterText(find.byType(TextField).at(0), 'friend');
    await tester.enterText(find.byType(TextField).at(1), 'password');
    await tester.tap(find.text('Entrar'));
    await tester.pumpAndSettle();

    expect(find.text('Termos de Uso'), findsOneWidget);
    expect(find.text('Termos completos.'), findsOneWidget);
    expect(api.listProfilesCalls, 0);

    await tester.tap(find.byType(Checkbox));
    await tester.pump();
    await tester.tap(find.text('Li e aceito'));
    await tester.pumpAndSettle();

    // Keep this assertion causal: the UI can only leave the gate after the
    // receipt and profile list have completed.
    expect(controller.status, SessionStatus.ready);
    expect(find.text('Seus perfis'), findsOneWidget);
    expect(find.text('Principal'), findsOneWidget);
    expect(api.privateProfileCalls, 0);

    await tester.tap(find.byTooltip('Mostrar QR'));
    await tester.pumpAndSettle();
    expect(find.textContaining('uso simultâneo'), findsOneWidget);
    expect(api.privateProfileCalls, 0);

    await tester.tap(find.text('Continuar'));
    await tester.pumpAndSettle();
    expect(find.text('QR do perfil'), findsOneWidget);
    expect(api.privateProfileCalls, 1);
  });
}

class WidgetFakeApi implements FriendApi {
  int listProfilesCalls = 0;
  int privateProfileCalls = 0;

  @override
  Future<TermsAcceptanceReceipt> acceptTerms({
    required String version,
    required String sha256,
  }) async {
    return TermsAcceptanceReceipt(
      accepted: true,
      accountId: 'account-1',
      version: version,
      sha256: sha256,
      acceptedAt: DateTime.utc(2026),
      eventId: 'event-1',
    );
  }

  @override
  Future<TermsDocument> currentTerms() async => const TermsDocument(
    version: 'initial-1',
    text: 'Termos completos.',
    sha256: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
  );

  @override
  Future<List<ProfileSummary>> listProfiles() async {
    listProfilesCalls++;
    return <ProfileSummary>[
      const ProfileSummary(
        id: 'primary',
        displayName: 'Principal',
        isPrimary: true,
        endpoint: 'vpn.darcanix.com',
        peerPublicKey: 'peer-primary',
        address: '10.0.0.2/32',
        allowedIPs: ['0.0.0.0/0'],
        dns: ['1.1.1.1'],
      ),
    ];
  }

  @override
  Future<LoginSession> login(String username, String password) async =>
      LoginSession(
        accessToken: 'opaque-token',
        expiresAt: DateTime.utc(2026, 12, 31),
        accountId: 'account-1',
        role: 'friend',
      );

  @override
  Future<ProfilePrivate> recoverPrivateProfile(String profileId) async {
    privateProfileCalls++;
    return const ProfilePrivate(
      id: 'primary',
      displayName: 'Principal',
      endpoint: 'vpn.darcanix.com',
      address: '10.0.0.2/32',
      allowedIPs: ['0.0.0.0/0'],
      dns: ['1.1.1.1'],
      peerPublicKey: 'peer-primary',
      serverPublicKey: 'server-public-key',
      endpointPort: 51442,
      privateKey: 'private-primary',
    );
  }

  @override
  Future<void> logout() async {}
}
