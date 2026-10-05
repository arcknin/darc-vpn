import 'package:flutter/foundation.dart';

import 'api/friend_api.dart';
import 'models.dart';
import 'storage.dart';

enum SessionStatus {
  signedOut,
  loading,
  termsRequired,
  acceptingTerms,
  loadingProfiles,
  ready,
  error,
}

class FriendSessionController extends ChangeNotifier {
  FriendSessionController({
    required FriendApi api,
    required TokenStore tokenStore,
    required ProfileSelectionStore selectionStore,
    this.allowPrivateTest = false,
  }) : _api = api,
       _tokenStore = tokenStore,
       _selectionStore = selectionStore;

  final bool allowPrivateTest;
  final FriendApi _api;
  final TokenStore _tokenStore;
  final ProfileSelectionStore _selectionStore;

  SessionStatus _status = SessionStatus.signedOut;
  TermsDocument? _terms;
  List<ProfileSummary> _profiles = const <ProfileSummary>[];
  ProfileSummary? _selectedProfile;
  ApiFailure? _failure;
  bool _termsAccepted = false;

  SessionStatus get status => _status;
  TermsDocument? get terms => _terms;
  List<ProfileSummary> get profiles => _profiles;
  ProfileSummary? get selectedProfile => _selectedProfile;
  ApiFailure? get failure => _failure;
  bool get termsAccepted => _termsAccepted;
  bool get profileAccessAllowed =>
      _termsAccepted ||
      (allowPrivateTest && _terms?.acceptanceRequired == false);
  bool get signedIn => _status != SessionStatus.signedOut;

  Future<void> signIn(String username, String password) async {
    _status = SessionStatus.loading;
    _failure = null;
    _terms = null;
    _profiles = const <ProfileSummary>[];
    _selectedProfile = null;
    _termsAccepted = false;
    notifyListeners();
    try {
      final session = await _api.login(username, password);
      await _tokenStore.write(session.accessToken);
      _terms = await _api.currentTerms();
      if (allowPrivateTest && _terms?.acceptanceRequired == false) {
        await _loadProfiles();
        _status = SessionStatus.ready;
      } else {
        _status = SessionStatus.termsRequired;
      }
      notifyListeners();
    } catch (error) {
      await _tokenStore.clear();
      _status = SessionStatus.error;
      _failure = _toApiFailure(error);
      notifyListeners();
      rethrow;
    }
  }

  Future<void> acceptTerms() async {
    final terms = _terms;
    if (terms == null || _status != SessionStatus.termsRequired) {
      throw StateError('terms are not ready for acceptance');
    }
    _status = SessionStatus.acceptingTerms;
    _failure = null;
    notifyListeners();
    try {
      final receipt = await _api.acceptTerms(
        version: terms.version,
        sha256: terms.sha256,
      );
      if (!receipt.accepted ||
          receipt.version != terms.version ||
          receipt.sha256 != terms.sha256) {
        throw const ApiFailure(
          code: 'invalid_response',
          message: 'resposta de aceite inválida',
        );
      }
      _termsAccepted = true;
      await _loadProfiles();
      _status = SessionStatus.ready;
      notifyListeners();
    } catch (error) {
      _termsAccepted = false;
      _status = SessionStatus.termsRequired;
      _failure = _toApiFailure(error);
      notifyListeners();
      rethrow;
    }
  }

  Future<void> selectProfile(String profileId) async {
    if (_status != SessionStatus.ready || !profileAccessAllowed) {
      throw StateError('profiles are not ready');
    }
    final profile = _findProfile(profileId);
    if (profile == null) {
      throw const ApiFailure(
        code: 'profile_not_found',
        message: 'perfil não encontrado',
      );
    }
    _selectedProfile = profile;
    await _selectionStore.write(profile.id);
    notifyListeners();
  }

  Future<ProfilePrivate> revealProfile(String profileId) async {
    if (_status != SessionStatus.ready || !profileAccessAllowed) {
      throw StateError(
        'terms acceptance is required before private profile delivery',
      );
    }
    if (_findProfile(profileId) == null) {
      throw const ApiFailure(
        code: 'profile_not_found',
        message: 'perfil não encontrado',
      );
    }
    try {
      return await _api.recoverPrivateProfile(profileId);
    } catch (error) {
      throw _toApiFailure(error);
    }
  }

  Future<void> signOut() async {
    try {
      if (await _tokenStore.read() != null) await _api.logout();
    } finally {
      await _tokenStore.clear();
      _status = SessionStatus.signedOut;
      _terms = null;
      _profiles = const <ProfileSummary>[];
      _selectedProfile = null;
      _termsAccepted = false;
      _failure = null;
      notifyListeners();
    }
  }

  Future<void> _loadProfiles() async {
    _status = SessionStatus.loadingProfiles;
    notifyListeners();
    final profiles = await _api.listProfiles();
    final primary = profiles.cast<ProfileSummary?>().firstWhere(
      (profile) => profile!.isPrimary,
      orElse: () => profiles.isEmpty ? null : profiles.first,
    );
    final storedId = await _selectionStore.read();
    final selected = storedId == null
        ? null
        : profiles.cast<ProfileSummary?>().firstWhere(
            (profile) => profile!.id == storedId,
            orElse: () => null,
          );
    _profiles = List<ProfileSummary>.unmodifiable(profiles);
    _selectedProfile = selected ?? primary;
    if (_selectedProfile != null && selected == null) {
      await _selectionStore.write(_selectedProfile!.id);
    }
  }

  ProfileSummary? _findProfile(String id) {
    for (final profile in _profiles) {
      if (profile.id == id) return profile;
    }
    return null;
  }

  ApiFailure _toApiFailure(Object error) {
    if (error is ApiFailure) return error;
    if (error is FormatException) {
      return ApiFailure(code: 'invalid_response', message: error.message);
    }
    return const ApiFailure(
      code: 'unavailable',
      message: 'serviço indisponível',
    );
  }
}
