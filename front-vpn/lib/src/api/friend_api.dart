import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

import '../models.dart';

typedef TokenReader = Future<String?> Function();

abstract interface class FriendApi {
  Future<LoginSession> login(String username, String password);

  Future<void> logout();

  Future<TermsDocument> currentTerms();

  Future<TermsAcceptanceReceipt> acceptTerms({
    required String version,
    required String sha256,
  });

  Future<List<ProfileSummary>> listProfiles();

  Future<ProfilePrivate> recoverPrivateProfile(String profileId);
}

class ApiContract {
  const ApiContract._();

  static const friendLogin = '/api/v1/auth/friend/login';
  static const logout = '/api/v1/auth/logout';
  static const currentTerms = '/api/v1/terms/current';
  static const acceptance = '/api/v1/terms/acceptance';
  static const profiles = '/api/v1/profiles';

  static String privateProfile(String profileId) =>
      '$profiles/${Uri.encodeComponent(profileId)}/private';
}

class HttpFriendApi implements FriendApi {
  HttpFriendApi({
    required String baseUrl,
    required TokenReader tokenReader,
    http.Client? client,
    bool allowInsecureForDevelopment = false,
  }) : _baseUri = _parseBaseUri(baseUrl, allowInsecureForDevelopment),
       _tokenReader = tokenReader,
       _client = client ?? http.Client();

  final Uri _baseUri;
  final TokenReader _tokenReader;
  final http.Client _client;

  @override
  Future<TermsAcceptanceReceipt> acceptTerms({
    required String version,
    required String sha256,
  }) async {
    final response = await _send(
      'POST',
      ApiContract.acceptance,
      body: <String, dynamic>{
        'accepted': true,
        'version': version,
        'sha256': sha256,
      },
    );
    return TermsAcceptanceReceipt.fromJson(decodeJsonObject(response.body));
  }

  @override
  Future<TermsDocument> currentTerms() async {
    final response = await _send('GET', ApiContract.currentTerms);
    return TermsDocument.fromJson(decodeJsonObject(response.body));
  }

  @override
  Future<List<ProfileSummary>> listProfiles() async {
    final response = await _send('GET', ApiContract.profiles);
    final object = decodeJsonObject(response.body);
    final values = object['profiles'];
    if (values is! List) {
      throw const FormatException('profiles response is invalid');
    }
    return List<ProfileSummary>.unmodifiable(
      values.map(
        (value) =>
            ProfileSummary.fromJson(Map<String, dynamic>.from(value as Map)),
      ),
    );
  }

  @override
  Future<LoginSession> login(String username, String password) async {
    final response = await _send(
      'POST',
      ApiContract.friendLogin,
      body: <String, dynamic>{'username': username, 'password': password},
      includeToken: false,
    );
    final session = LoginSession.fromJson(decodeJsonObject(response.body));
    if (session.role != 'friend') {
      throw const ApiFailure(
        code: 'invalid_response',
        message: 'resposta de autenticação inválida',
      );
    }
    return session;
  }

  @override
  Future<ProfilePrivate> recoverPrivateProfile(String profileId) async {
    final response = await _send('GET', ApiContract.privateProfile(profileId));
    return ProfilePrivate.fromJson(decodeJsonObject(response.body));
  }

  @override
  Future<void> logout() async {
    await _send('POST', ApiContract.logout);
  }

  Future<http.Response> _send(
    String method,
    String path, {
    Map<String, dynamic>? body,
    bool includeToken = true,
  }) async {
    final headers = <String, String>{'Accept': 'application/json'};
    if (body != null) headers['Content-Type'] = 'application/json';
    if (includeToken) {
      final token = await _tokenReader();
      if (token == null || token.isEmpty) {
        throw const ApiFailure(
          code: 'not_authenticated',
          message: 'autenticação necessária',
          statusCode: 401,
        );
      }
      headers['Authorization'] = 'Bearer $token';
    }

    final request = http.Request(method, _baseUri.resolve(path));
    request.headers.addAll(headers);
    if (body != null) request.body = jsonEncode(body);

    try {
      final response = await _client
          .send(request)
          .then(http.Response.fromStream);
      if (response.statusCode < 200 || response.statusCode >= 300) {
        throw _failureFromResponse(response);
      }
      return response;
    } on ApiFailure {
      rethrow;
    } on TimeoutException {
      throw const ApiFailure(
        code: 'unavailable',
        message: 'serviço indisponível',
      );
    } on http.ClientException {
      throw const ApiFailure(
        code: 'unavailable',
        message: 'serviço indisponível',
      );
    } on FormatException {
      rethrow;
    } catch (_) {
      throw const ApiFailure(
        code: 'unavailable',
        message: 'serviço indisponível',
      );
    }
  }

  ApiFailure _failureFromResponse(http.Response response) {
    try {
      final object = decodeJsonObject(response.body);
      final code = object['code'];
      final message = object['message'];
      if (code is String && message is String) {
        return ApiFailure(
          code: code,
          message: message,
          statusCode: response.statusCode,
        );
      }
    } on FormatException {
      // Fall through to the stable generic error below.
    } on Object {
      // A malformed error body must not escape response details to the UI.
    }
    return ApiFailure(
      code: _codeForStatus(response.statusCode),
      message: _messageForStatus(response.statusCode),
      statusCode: response.statusCode,
    );
  }
}

Uri _parseBaseUri(String raw, bool allowInsecureForDevelopment) {
  final value = raw.trim();
  if (value.isEmpty) {
    throw ArgumentError.value(raw, 'baseUrl', 'API base URL is required');
  }
  final uri = Uri.tryParse(value);
  if (uri == null ||
      !uri.hasScheme ||
      uri.host.isEmpty ||
      uri.path.contains(' ')) {
    throw ArgumentError.value(raw, 'baseUrl', 'API base URL is invalid');
  }
  if (uri.scheme != 'https' &&
      !(allowInsecureForDevelopment && uri.scheme == 'http')) {
    throw ArgumentError.value(
      raw,
      'baseUrl',
      'HTTPS is required outside local development',
    );
  }
  return uri.replace(path: uri.path.endsWith('/') ? uri.path : '${uri.path}/');
}

String _codeForStatus(int statusCode) {
  if (statusCode == 401) return 'invalid_credentials';
  if (statusCode == 403) return 'forbidden';
  if (statusCode == 423) return 'account_blocked';
  if (statusCode == 428) return 'terms_acceptance_required';
  return 'unavailable';
}

String _messageForStatus(int statusCode) {
  if (statusCode == 401) return 'credenciais inválidas';
  if (statusCode == 403) return 'operação não autorizada';
  if (statusCode == 423) return 'conta bloqueada';
  if (statusCode == 428) return 'aceite dos termos necessário';
  return 'serviço indisponível';
}
