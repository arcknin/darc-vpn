import 'models.dart';

class WireGuardQrConfig {
  const WireGuardQrConfig({
    required this.profileId,
    required this.peerPublicKey,
    required this.data,
    this.warning = simultaneousUseWarning,
  });

  static const simultaneousUseWarning =
      'O uso simultâneo do mesmo perfil ou chave em dois aparelhos pode causar '
      'instabilidade, lentidão ou perda de conexão. Use este perfil alternadamente.';

  final String profileId;
  final String peerPublicKey;
  final String data;
  final String warning;

  factory WireGuardQrConfig.fromProfile(ProfilePrivate profile) {
    final interfaceLines = <String>[
      '[Interface]',
      'PrivateKey = ${profile.privateKey}',
      'Address = ${profile.address}',
      if (profile.dns.isNotEmpty) 'DNS = ${profile.dns.join(', ')}',
      '',
      '[Peer]',
      'PublicKey = ${profile.serverPublicKey}',
      if (profile.presharedKey != null)
        'PresharedKey = ${profile.presharedKey}',
      'AllowedIPs = ${profile.allowedIPs.join(', ')}',
      'Endpoint = ${profile.endpoint}:${profile.endpointPort}',
    ];
    return WireGuardQrConfig(
      profileId: profile.id,
      peerPublicKey: profile.peerPublicKey,
      data: '${interfaceLines.join('\n')}\n',
    );
  }
}
