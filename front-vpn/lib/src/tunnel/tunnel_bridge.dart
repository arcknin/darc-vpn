import 'dart:async';
import 'package:flutter/services.dart';

enum TunnelState { disconnected, connecting, active, disconnecting, failed }

abstract interface class TunnelBridge {
  Stream<TunnelState> get states;
  Future<bool> preparePermission();
  Future<void> start(String configuration);
  Future<void> stop();
}

class AndroidTunnelBridge implements TunnelBridge {
  static const _methods = MethodChannel('darc.vpn/tunnel');
  static const _events = EventChannel('darc.vpn/tunnel-state');
  late final Stream<TunnelState> _states = _events.receiveBroadcastStream().map(
    (event) {
      switch (event) {
        case 'active':
          return TunnelState.active;
        case 'disconnected':
          return TunnelState.disconnected;
        case 'connecting':
          return TunnelState.connecting;
        case 'disconnecting':
          return TunnelState.disconnecting;
        default:
          return TunnelState.failed;
      }
    },
  );
  @override
  Stream<TunnelState> get states => _states;
  @override
  Future<bool> preparePermission() async =>
      await _methods.invokeMethod<bool>('preparePermission') ?? false;
  @override
  Future<void> start(String configuration) =>
      _methods.invokeMethod<void>('start', {'configuration': configuration});
  @override
  Future<void> stop() => _methods.invokeMethod<void>('stop');
}
