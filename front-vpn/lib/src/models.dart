import 'dart:convert';

const vpnEndpoint = 'vpn.darcanix.com';

class ApiFailure implements Exception {
  const ApiFailure({
    required this.code,
    required this.message,
    this.statusCode,
  });

  final String code;
  final String message;
  final int? statusCode;

  @override
  String toString() => 'ApiFailure($code): $message';
}

class LoginSession {
  const LoginSession({
    required this.accessToken,
    required this.expiresAt,
    required this.accountId,
    required this.role,
  });

  final String accessToken;
  final DateTime expiresAt;
  final String accountId;
  final String role;

  factory LoginSession.fromJson(Map<String, dynamic> json) {
    final role = _requiredString(json, 'role');
    if (role != 'friend' && role != 'admin') {
      throw const FormatException('invalid role field');
    }
    return LoginSession(
      accessToken: _requiredString(json, 'accessToken'),
      expiresAt: _requiredDateTime(json, 'expiresAt'),
      accountId: _requiredString(json, 'accountId'),
      role: role,
    );
  }
}

class TermsDocument {
  const TermsDocument({
    required this.version,
    required this.text,
    required this.sha256,
    this.acceptanceRequired = true,
  });

  final String version;
  final String text;
  final String sha256;
  final bool acceptanceRequired;

  factory TermsDocument.fromJson(Map<String, dynamic> json) {
    final sha256 = _requiredString(json, 'sha256');
    _validateSha256(sha256);
    if (json.containsKey('acceptanceRequired') &&
        json['acceptanceRequired'] is! bool) {
      throw const FormatException('invalid contract field: acceptanceRequired');
    }
    return TermsDocument(
      version: _requiredString(json, 'version'),
      text: _requiredString(json, 'text'),
      sha256: sha256,
      acceptanceRequired: json['acceptanceRequired'] is bool
          ? json['acceptanceRequired'] as bool
          : _requiredString(json, 'version') != 'private-technical-test',
    );
  }
}

class TermsAcceptanceReceipt {
  const TermsAcceptanceReceipt({
    required this.accepted,
    required this.accountId,
    required this.version,
    required this.sha256,
    required this.acceptedAt,
    required this.eventId,
  });

  final bool accepted;
  final String accountId;
  final String version;
  final String sha256;
  final DateTime acceptedAt;
  final String eventId;

  factory TermsAcceptanceReceipt.fromJson(Map<String, dynamic> json) {
    final sha256 = _requiredString(json, 'sha256');
    _validateSha256(sha256);
    return TermsAcceptanceReceipt(
      accepted: json['accepted'] == true,
      accountId: _requiredString(json, 'accountId'),
      version: _requiredString(json, 'version'),
      sha256: sha256,
      acceptedAt: _requiredDateTime(json, 'acceptedAt'),
      eventId: _requiredString(json, 'eventId'),
    );
  }
}

class ProfileSummary {
  const ProfileSummary({
    required this.id,
    required this.displayName,
    required this.isPrimary,
    required this.endpoint,
    required this.peerPublicKey,
    required this.address,
    required this.allowedIPs,
    required this.dns,
  });

  final String id;
  final String displayName;
  final bool isPrimary;
  final String endpoint;
  final String peerPublicKey;
  final String address;
  final List<String> allowedIPs;
  final List<String> dns;

  factory ProfileSummary.fromJson(Map<String, dynamic> json) {
    final endpoint = _requiredString(json, 'endpoint');
    if (endpoint != vpnEndpoint) {
      throw const FormatException(
        'profile endpoint is outside the approved contract',
      );
    }
    return ProfileSummary(
      id: _requiredString(json, 'id'),
      displayName: _requiredString(json, 'displayName'),
      isPrimary: json['isPrimary'] == true,
      endpoint: endpoint,
      peerPublicKey: _requiredString(json, 'peerPublicKey'),
      address: _requiredString(json, 'address'),
      allowedIPs: _stringList(json, 'allowedIPs'),
      dns: _stringList(json, 'dns'),
    );
  }
}

class ProfilePrivate {
  const ProfilePrivate({
    required this.id,
    required this.displayName,
    required this.endpoint,
    required this.address,
    required this.allowedIPs,
    required this.dns,
    required this.peerPublicKey,
    required this.serverPublicKey,
    required this.endpointPort,
    required this.privateKey,
    this.presharedKey,
  });

  final String id;
  final String displayName;
  final String endpoint;
  final String address;
  final List<String> allowedIPs;
  final List<String> dns;
  final String peerPublicKey;
  final String serverPublicKey;
  final int endpointPort;
  final String privateKey;
  final String? presharedKey;

  factory ProfilePrivate.fromJson(Map<String, dynamic> json) {
    final endpoint = _requiredString(json, 'endpoint');
    if (endpoint != vpnEndpoint) {
      throw const FormatException(
        'profile endpoint is outside the approved contract',
      );
    }
    return ProfilePrivate(
      id: _requiredString(json, 'id'),
      displayName: _requiredString(json, 'displayName'),
      endpoint: endpoint,
      address: _requiredString(json, 'address'),
      allowedIPs: _stringList(json, 'allowedIPs'),
      dns: _stringList(json, 'dns'),
      peerPublicKey: _requiredString(json, 'peerPublicKey'),
      serverPublicKey: _requiredString(json, 'serverPublicKey'),
      endpointPort: _endpointPort(json),
      privateKey: _requiredString(json, 'privateKey'),
      presharedKey: _optionalString(json, 'presharedKey'),
    );
  }
}

String _requiredString(Map<String, dynamic> json, String key) {
  final value = json[key];
  if (value is! String || value.isEmpty) {
    throw FormatException('missing contract field: $key');
  }
  return value;
}

String? _optionalString(Map<String, dynamic> json, String key) {
  final value = json[key];
  if (value == null) return null;
  if (value is! String || value.isEmpty) {
    throw FormatException('invalid contract field: $key');
  }
  return value;
}

void _validateSha256(String value) {
  if (!RegExp(r'^[a-f0-9]{64}$').hasMatch(value)) {
    throw const FormatException('invalid SHA-256 field');
  }
}

DateTime _requiredDateTime(Map<String, dynamic> json, String key) {
  final value = _requiredString(json, key);
  return DateTime.parse(value).toUtc();
}

List<String> _stringList(Map<String, dynamic> json, String key) {
  final value = json[key];
  if (value is! List || value.any((item) => item is! String || item.isEmpty)) {
    throw FormatException('invalid contract field: $key');
  }
  return List<String>.unmodifiable(value.cast<String>());
}

Map<String, dynamic> decodeJsonObject(String body) {
  final value = jsonDecode(body);
  if (value is! Map) {
    throw const FormatException('response is not a JSON object');
  }
  return Map<String, dynamic>.from(value);
}

int _endpointPort(Map<String, dynamic> json) {
  final value = json["endpointPort"];
  if (value is! int || value < 1 || value > 65535) {
    throw const FormatException("invalid contract field: endpointPort");
  }
  return value;
}
