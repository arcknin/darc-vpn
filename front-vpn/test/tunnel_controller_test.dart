import 'dart:async';
import 'package:flutter_test/flutter_test.dart';
import 'package:darc_vpn/src/session_controller.dart';
import 'package:darc_vpn/src/storage.dart';
import 'package:darc_vpn/src/tunnel/tunnel_controller.dart';
import 'package:darc_vpn/src/tunnel/tunnel_bridge.dart';
import 'b06_core_test.dart' show FakeFriendApi;

void main() {
  test(
    'disconnected polling during permission keeps connection busy',
    () async {
      final session = FriendSessionController(
        api: FakeFriendApi(),
        tokenStore: MemoryTokenStore(),
        selectionStore: MemorySelectionStore(),
      );
      await session.signIn('f', 'p');
      await session.acceptTerms();
      final bridge = FakeBridge()..pending = Completer<bool>();
      final c = TunnelController(session: session, bridge: bridge);
      final connecting = c.connect();
      bridge.events.add(TunnelState.disconnected);
      await Future<void>.delayed(Duration.zero);
      expect(c.state, TunnelState.connecting);
      expect(c.busy, isTrue);
      bridge.pending!.complete(false);
      await connecting;
      expect(c.state, TunnelState.disconnected);
      c.dispose();
      await bridge.close();
    },
  );
  test('permission denied never starts or reports active', () async {
    final session = FriendSessionController(
      api: FakeFriendApi(),
      tokenStore: MemoryTokenStore(),
      selectionStore: MemorySelectionStore(),
    );
    await session.signIn('f', 'p');
    await session.acceptTerms();
    final bridge = FakeBridge()..permission = false;
    final c = TunnelController(session: session, bridge: bridge);
    await c.connect();
    expect(c.state, TunnelState.disconnected);
    expect(bridge.starts, 0);
    c.dispose();
    await bridge.close();
  });
  test(
    'start acknowledgement without callback is not connected; observed down wins',
    () async {
      final session = FriendSessionController(
        api: FakeFriendApi(),
        tokenStore: MemoryTokenStore(),
        selectionStore: MemorySelectionStore(),
      );
      await session.signIn('f', 'p');
      await session.acceptTerms();
      final bridge = FakeBridge();
      final c = TunnelController(session: session, bridge: bridge);
      await c.connect();
      expect(c.state, TunnelState.connecting);
      bridge.events.add(TunnelState.active);
      await Future<void>.delayed(Duration.zero);
      expect(c.state, TunnelState.active);
      bridge.events.add(TunnelState.disconnected);
      await Future<void>.delayed(Duration.zero);
      expect(c.state, TunnelState.disconnected);
      c.dispose();
      await bridge.close();
    },
  );
  test(
    'logout during permission dialog cannot start a stale profile',
    () async {
      final session = FriendSessionController(
        api: FakeFriendApi(),
        tokenStore: MemoryTokenStore(),
        selectionStore: MemorySelectionStore(),
      );
      await session.signIn('f', 'p');
      await session.acceptTerms();
      final bridge = FakeBridge()..pending = Completer<bool>();
      final c = TunnelController(session: session, bridge: bridge);
      final connecting = c.connect();
      await Future<void>.delayed(Duration.zero);
      await c.logout();
      bridge.pending!.complete(true);
      await connecting;
      expect(bridge.starts, 0);
      expect(session.status, SessionStatus.signedOut);
      c.dispose();
      await bridge.close();
    },
  );
}

class FakeBridge implements TunnelBridge {
  final events = StreamController<TunnelState>.broadcast();
  bool permission = true;
  int starts = 0;
  Completer<bool>? pending;
  @override
  Stream<TunnelState> get states => events.stream;
  @override
  Future<bool> preparePermission() async =>
      pending == null ? permission : pending!.future;
  @override
  Future<void> start(String configuration) async {
    starts++;
  }

  @override
  Future<void> stop() async {
    events.add(TunnelState.disconnected);
  }

  Future<void> close() => events.close();
}
