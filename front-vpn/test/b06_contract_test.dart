import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

import 'package:darc_vpn/src/api/friend_api.dart';
import 'package:darc_vpn/src/models.dart';
import 'package:darc_vpn/src/storage.dart';

void main() {
  test(
    'client routes and payload names remain aligned with the OpenAPI contract',
    () {
      final openApi = File('../back-vpn/api/openapi.yaml').readAsStringSync();

      expect(openApi, contains(ApiContract.friendLogin));
      expect(openApi, contains(ApiContract.currentTerms));
      expect(openApi, contains(ApiContract.acceptance));
      expect(openApi, contains(ApiContract.profiles));
      expect(openApi, contains('/private'));
      expect(openApi, contains('accessToken'));
      expect(openApi, contains('privateKey'));
      expect(openApi, contains('presharedKey'));
      expect(openApi, contains('vpn.darcanix.com'));
    },
  );

  test(
    'HTTPS is mandatory unless the caller explicitly opts into local HTTP',
    () {
      expect(
        () => HttpFriendApi(
          baseUrl: 'http://127.0.0.1:8080',
          tokenReader: () async => null,
        ),
        throwsArgumentError,
      );

      expect(
        () => HttpFriendApi(
          baseUrl: 'http://127.0.0.1:8080',
          tokenReader: () async => null,
          allowInsecureForDevelopment: true,
        ),
        returnsNormally,
      );
    },
  );

  test(
    'HTTP adapter sends opaque bearer token and decodes only contract fields',
    () async {
      String? authorization;
      final client = MockClient((request) async {
        authorization = request.headers['authorization'];
        expect(request.url.path, ApiContract.friendLogin);
        final body = jsonDecode(request.body) as Map<String, dynamic>;
        expect(body, <String, dynamic>{
          'username': 'friend',
          'password': 'password',
        });
        return http.Response(
          jsonEncode(<String, dynamic>{
            'accessToken': 'opaque-token',
            'expiresAt': '2026-12-31T00:00:00Z',
            'accountId': 'account-1',
            'role': 'friend',
          }),
          200,
          headers: <String, String>{'content-type': 'application/json'},
        );
      });
      final api = HttpFriendApi(
        baseUrl: 'https://api.test',
        tokenReader: () async => null,
        client: client,
      );

      final session = await api.login('friend', 'password');

      expect(session.accessToken, 'opaque-token');
      expect(authorization, isNull);
    },
  );

  test(
    'profile endpoint and secret fields are validated against the contract',
    () {
      expect(
        () => ProfileSummary.fromJson(<String, dynamic>{
          'id': 'profile',
          'displayName': 'Principal',
          'isPrimary': true,
          'endpoint': 'example.invalid',
          'peerPublicKey': 'peer',
          'address': '10.0.0.2/32',
          'allowedIPs': <String>['0.0.0.0/0'],
          'dns': <String>['1.1.1.1'],
        }),
        throwsFormatException,
      );

      const storeKey = SecureTokenStore.key;
      expect(storeKey, isNot(SharedPreferencesProfileSelectionStore.key));
      expect(storeKey, isNot(contains('PrivateKey')));
    },
  );
}
