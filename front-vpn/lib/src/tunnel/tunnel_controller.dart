import 'dart:async';
import 'package:flutter/foundation.dart';
import '../session_controller.dart';
import '../qr_config.dart';
import 'tunnel_bridge.dart';
export 'tunnel_bridge.dart' show TunnelState;

class TunnelController extends ChangeNotifier {
  TunnelController({
    required FriendSessionController session,
    required TunnelBridge bridge,
  }) : _session = session,
       _bridge = bridge {
    _subscription = bridge.states.listen(
      _observed,
      onError: (Object _) {
        _state = TunnelState.failed;
        _message = 'Não foi possível consultar o estado da VPN.';
        _notify();
      },
    );
  }
  final FriendSessionController _session;
  final TunnelBridge _bridge;
  late final StreamSubscription<TunnelState> _subscription;
  TunnelState _state = TunnelState.disconnected;
  String? _message;
  String? _activeProfileId;
  int _operation = 0;
  bool _disposed = false;
  bool _preparing = false;
  TunnelState get state => _state;
  String? get message => _message;
  String? get activeProfileId => _activeProfileId;
  bool get busy =>
      _state == TunnelState.connecting || _state == TunnelState.disconnecting;
  void _notify() {
    if (!_disposed) notifyListeners();
  }

  void _observed(TunnelState state) {
    // A down interface is expected while consent/profile retrieval is pending.
    if (_preparing &&
        _state == TunnelState.connecting &&
        state == TunnelState.disconnected) {
      return;
    }
    if (_state == TunnelState.disconnecting && state == TunnelState.active) {
      return;
    }
    _state = state;
    if (state == TunnelState.disconnected || state == TunnelState.failed) {
      _activeProfileId = null;
    }
    _notify();
  }

  Future<void> connect() async {
    if (busy || _state == TunnelState.active) return;
    final profile = _session.selectedProfile;
    if (profile == null ||
        _session.status != SessionStatus.ready ||
        !_session.profileAccessAllowed) {
      _message = 'Selecione um perfil disponível.';
      _notify();
      return;
    }
    final operation = ++_operation;
    _preparing = true;
    _state = TunnelState.connecting;
    _message = null;
    _notify();
    try {
      final granted = await _bridge.preparePermission();
      if (operation != _operation || _disposed) return;
      if (!granted) {
        _state = TunnelState.disconnected;
        _message = 'Permissão de VPN não concedida.';
        _notify();
        return;
      }
      final private = await _session.revealProfile(profile.id);
      if (operation != _operation ||
          _disposed ||
          _session.status != SessionStatus.ready) {
        return;
      }
      _activeProfileId = profile.id;
      _preparing = false;
      await _bridge.start(WireGuardQrConfig.fromProfile(private).data);
      // Only native state callbacks/queries can transition this controller to active.
      if (operation != _operation) await _bridge.stop();
    } catch (_) {
      if (operation != _operation || _disposed) return;
      _state = TunnelState.failed;
      _activeProfileId = null;
      _message = 'Não foi possível iniciar a VPN. Tente novamente.';
      _notify();
    } finally {
      if (operation == _operation) _preparing = false;
    }
  }

  Future<void> disconnect() async {
    ++_operation;
    _preparing = false;
    _state = TunnelState.disconnecting;
    _message = null;
    _notify();
    try {
      await _bridge.stop();
    } catch (_) {
      _state = TunnelState.failed;
      _message = 'Não foi possível confirmar a desconexão.';
      _notify();
    }
  }

  Future<void> logout() async {
    await disconnect();
    if (_state == TunnelState.failed) return;
    await _session.signOut();
  }

  @override
  void dispose() {
    _disposed = true;
    ++_operation;
    _subscription.cancel();
    super.dispose();
  }
}
