import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart';

import 'app.dart';
import 'src/api/friend_api.dart';
import 'src/session_controller.dart';
import 'src/storage.dart';
import 'src/tunnel/tunnel_bridge.dart';
import 'src/tunnel/tunnel_controller.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  const baseUrl = String.fromEnvironment('API_BASE_URL');
  const allowInsecure = bool.fromEnvironment('ALLOW_INSECURE_API');
  const privateTest = bool.fromEnvironment('PRIVATE_TEST');

  if (baseUrl.isEmpty) {
    runApp(const MissingConfigurationApp());
    return;
  }

  try {
    final tokenStore = SecureTokenStore();
    final selectionStore =
        await SharedPreferencesProfileSelectionStore.create();
    final api = HttpFriendApi(
      baseUrl: baseUrl,
      tokenReader: tokenStore.read,
      allowInsecureForDevelopment: allowInsecure && !kReleaseMode,
    );
    final session = FriendSessionController(
      api: api,
      tokenStore: tokenStore,
      selectionStore: selectionStore,
      allowPrivateTest: privateTest && !kReleaseMode,
    );
    final tunnel = TunnelController(
      session: session,
      bridge: AndroidTunnelBridge(),
    );
    runApp(VpnApp(controller: session, tunnel: tunnel));
  } on Object {
    runApp(const MissingConfigurationApp(invalid: true));
  }
}

class MissingConfigurationApp extends StatelessWidget {
  const MissingConfigurationApp({super.key, this.invalid = false});

  final bool invalid;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Darc VPN',
      home: Scaffold(
        appBar: AppBar(title: const Text('Darc VPN')),
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Text(
              invalid
                  ? 'A configuração da API é inválida. Use uma URL HTTPS válida.'
                  : 'A API ainda não foi configurada. Defina API_BASE_URL para iniciar.',
              textAlign: TextAlign.center,
            ),
          ),
        ),
      ),
    );
  }
}
