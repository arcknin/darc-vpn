import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

abstract interface class TokenStore {
  Future<String?> read();

  Future<void> write(String token);

  Future<void> clear();
}

class SecureTokenStore implements TokenStore {
  SecureTokenStore({FlutterSecureStorage? storage})
    : _storage = storage ?? const FlutterSecureStorage();

  static const key = 'friend_session_token';
  final FlutterSecureStorage _storage;

  @override
  Future<void> clear() => _storage.delete(key: key);

  @override
  Future<String?> read() => _storage.read(key: key);

  @override
  Future<void> write(String token) => _storage.write(key: key, value: token);
}

abstract interface class ProfileSelectionStore {
  Future<String?> read();

  Future<void> write(String profileId);
}

class SharedPreferencesProfileSelectionStore implements ProfileSelectionStore {
  SharedPreferencesProfileSelectionStore(this._preferences);

  static const key = 'selected_profile_id';
  final SharedPreferences _preferences;

  static Future<SharedPreferencesProfileSelectionStore> create() async {
    return SharedPreferencesProfileSelectionStore(
      await SharedPreferences.getInstance(),
    );
  }

  @override
  Future<String?> read() async => _preferences.getString(key);

  @override
  Future<void> write(String profileId) async {
    await _preferences.setString(key, profileId);
  }
}

class MemoryTokenStore implements TokenStore {
  MemoryTokenStore({String? initialValue}) : value = initialValue;

  String? value;
  final List<String?> writes = <String?>[];

  @override
  Future<void> clear() async {
    value = null;
  }

  @override
  Future<String?> read() async => value;

  @override
  Future<void> write(String token) async {
    value = token;
    writes.add(token);
  }
}

class MemorySelectionStore implements ProfileSelectionStore {
  MemorySelectionStore({String? initialValue}) : value = initialValue;

  String? value;
  final List<String> writes = <String>[];

  @override
  Future<String?> read() async => value;

  @override
  Future<void> write(String profileId) async {
    value = profileId;
    writes.add(profileId);
  }
}
