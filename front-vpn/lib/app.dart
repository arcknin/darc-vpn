import 'package:flutter/material.dart';
import 'package:qr_flutter/qr_flutter.dart';

import 'src/models.dart';
import 'src/qr_config.dart';
import 'src/session_controller.dart';
import 'src/tunnel/tunnel_controller.dart';

class VpnApp extends StatelessWidget {
  const VpnApp({super.key, required this.controller, this.tunnel});

  final FriendSessionController controller;
  final TunnelController? tunnel;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Darc VPN',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xff0b5cad)),
        useMaterial3: true,
      ),
      home: SessionRouter(controller: controller, tunnel: tunnel),
    );
  }
}

class SessionRouter extends StatelessWidget {
  const SessionRouter({super.key, required this.controller, this.tunnel});

  final FriendSessionController controller;
  final TunnelController? tunnel;

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: controller,
      builder: (context, _) {
        switch (controller.status) {
          case SessionStatus.signedOut:
          case SessionStatus.loading:
          case SessionStatus.error:
            return LoginPage(controller: controller);
          case SessionStatus.termsRequired:
          case SessionStatus.acceptingTerms:
          case SessionStatus.loadingProfiles:
            return TermsPage(controller: controller);
          case SessionStatus.ready:
            return ProfilesPage(controller: controller, tunnel: tunnel);
        }
      },
    );
  }
}

class LoginPage extends StatefulWidget {
  const LoginPage({super.key, required this.controller});

  final FriendSessionController controller;

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final _username = TextEditingController();
  final _password = TextEditingController();
  bool _submitting = false;

  @override
  void dispose() {
    _username.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_username.text.trim().isEmpty || _password.text.isEmpty) return;
    setState(() => _submitting = true);
    try {
      await widget.controller.signIn(_username.text.trim(), _password.text);
    } on Object {
      // The controller exposes the stable error code for rendering below.
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final failure = widget.controller.failure;
    return Scaffold(
      appBar: AppBar(title: const Text('Darc VPN')),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(24),
          children: [
            const Icon(Icons.shield_outlined, size: 64),
            const SizedBox(height: 16),
            Text(
              'Entrar como amigo',
              style: Theme.of(context).textTheme.headlineSmall,
            ),
            const SizedBox(height: 8),
            const Text('Use as credenciais fornecidas pelo administrador.'),
            const SizedBox(height: 24),
            TextField(
              controller: _username,
              autofillHints: const [AutofillHints.username],
              decoration: const InputDecoration(
                labelText: 'Usuário',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: _password,
              obscureText: true,
              autofillHints: const [AutofillHints.password],
              decoration: const InputDecoration(
                labelText: 'Senha',
                border: OutlineInputBorder(),
              ),
              onSubmitted: (_) => _submit(),
            ),
            if (failure != null) ...[
              const SizedBox(height: 16),
              _ErrorMessage(message: _messageForFailure(failure.code)),
            ],
            const SizedBox(height: 24),
            FilledButton(
              onPressed: _submitting ? null : _submit,
              child: _submitting
                  ? const SizedBox(
                      height: 20,
                      width: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('Entrar'),
            ),
          ],
        ),
      ),
    );
  }
}

class TermsPage extends StatefulWidget {
  const TermsPage({super.key, required this.controller});

  final FriendSessionController controller;

  @override
  State<TermsPage> createState() => _TermsPageState();
}

class _TermsPageState extends State<TermsPage> {
  bool _accepted = false;
  bool _submitting = false;

  Future<void> _accept() async {
    setState(() => _submitting = true);
    try {
      await widget.controller.acceptTerms();
    } on Object {
      // The gate remains visible and the user can retry.
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final terms = widget.controller.terms;
    final failure = widget.controller.failure;
    if (terms == null) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    final busy =
        _submitting || widget.controller.status != SessionStatus.termsRequired;
    return Scaffold(
      appBar: AppBar(title: const Text('Termos de Uso')),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(20),
          children: [
            Text(
              'Versão ${terms.version}',
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 8),
            SelectableText('SHA-256: ${terms.sha256}'),
            const SizedBox(height: 16),
            DecoratedBox(
              decoration: BoxDecoration(
                border: Border.all(
                  color: Theme.of(context).colorScheme.outline,
                ),
                borderRadius: BorderRadius.circular(12),
              ),
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: SelectableText(terms.text),
              ),
            ),
            const SizedBox(height: 12),
            CheckboxListTile(
              contentPadding: EdgeInsets.zero,
              value: _accepted,
              onChanged: busy
                  ? null
                  : (value) => setState(() => _accepted = value ?? false),
              title: const Text('Li e aceito os Termos de Uso'),
              controlAffinity: ListTileControlAffinity.leading,
            ),
            if (failure != null) ...[
              const SizedBox(height: 8),
              _ErrorMessage(message: _messageForFailure(failure.code)),
            ],
            const SizedBox(height: 16),
            FilledButton(
              onPressed: !_accepted || busy ? null : _accept,
              child: busy && _submitting
                  ? const SizedBox(
                      height: 20,
                      width: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('Li e aceito'),
            ),
          ],
        ),
      ),
    );
  }
}

class ProfilesPage extends StatelessWidget {
  const ProfilesPage({super.key, required this.controller, this.tunnel});

  final FriendSessionController controller;
  final TunnelController? tunnel;

  Future<void> _openQr(BuildContext context, ProfileSummary profile) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Aviso antes de mostrar o QR'),
        content: const Text(WireGuardQrConfig.simultaneousUseWarning),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancelar'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Continuar'),
          ),
        ],
      ),
    );
    if (confirmed != true || !context.mounted) return;
    try {
      final privateProfile = await controller.revealProfile(profile.id);
      if (!context.mounted) return;
      await Navigator.of(context).push<void>(
        MaterialPageRoute<void>(
          builder: (_) => WireGuardQrPage(
            config: WireGuardQrConfig.fromProfile(privateProfile),
          ),
        ),
      );
    } on ApiFailure catch (failure) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(_messageForFailure(failure.code))));
    }
  }

  @override
  Widget build(BuildContext context) {
    final selectedId = controller.selectedProfile?.id;
    return Scaffold(
      appBar: AppBar(
        title: const Text('Seus perfis'),
        actions: [
          IconButton(
            tooltip: 'Sair',
            onPressed: tunnel == null ? controller.signOut : tunnel!.logout,
            icon: const Icon(Icons.logout),
          ),
        ],
      ),
      body: controller.profiles.isEmpty
          ? const Center(child: Text('Nenhum perfil disponível.'))
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                const Text(
                  'Escolha o perfil que será usado na próxima conexão.',
                ),
                const SizedBox(height: 12),
                if (tunnel != null) TunnelPanel(controller: tunnel!),
                RadioGroup<String>(
                  groupValue: selectedId,
                  onChanged: (value) {
                    if (value != null) controller.selectProfile(value);
                  },
                  child: Column(
                    children: controller.profiles
                        .map(
                          (profile) => Card(
                            child: ListTile(
                              leading: Radio<String>(value: profile.id),
                              title: Text(profile.displayName),
                              subtitle: Text(
                                profile.isPrimary
                                    ? 'Perfil principal'
                                    : 'Perfil adicional',
                              ),
                              trailing: IconButton(
                                tooltip: 'Mostrar QR',
                                onPressed: () => _openQr(context, profile),
                                icon: const Icon(Icons.qr_code_2),
                              ),
                              onTap: () => controller.selectProfile(profile.id),
                            ),
                          ),
                        )
                        .toList(),
                  ),
                ),
              ],
            ),
    );
  }
}

class WireGuardQrPage extends StatefulWidget {
  const WireGuardQrPage({super.key, required this.config});

  final WireGuardQrConfig config;

  @override
  State<WireGuardQrPage> createState() => _WireGuardQrPageState();
}

class _WireGuardQrPageState extends State<WireGuardQrPage> {
  WireGuardQrConfig? _config;

  @override
  void initState() {
    super.initState();
    _config = widget.config;
  }

  @override
  void dispose() {
    _config = null;
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final config = _config;
    if (config == null) return const SizedBox.shrink();
    return Scaffold(
      appBar: AppBar(title: const Text('QR do perfil')),
      body: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          Text(
            'Perfil ${config.profileId}',
            style: Theme.of(context).textTheme.titleLarge,
          ),
          const SizedBox(height: 12),
          const Text(WireGuardQrConfig.simultaneousUseWarning),
          const SizedBox(height: 20),
          Center(
            child: ColoredBox(
              color: Colors.white,
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: QrImageView(data: config.data, size: 280),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _ErrorMessage extends StatelessWidget {
  const _ErrorMessage({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Text(
      message,
      style: TextStyle(color: Theme.of(context).colorScheme.error),
    );
  }
}

String _messageForFailure(String code) {
  switch (code) {
    case 'invalid_credentials':
      return 'Credenciais inválidas.';
    case 'account_blocked':
      return 'Esta conta está bloqueada.';
    case 'terms_acceptance_required':
      return 'Aceite os termos para continuar.';
    case 'acceptance_persistence_failed':
      return 'Não foi possível registrar o aceite. Tente novamente.';
    case 'profile_recovery_failed':
      return 'Não foi possível recuperar este perfil.';
    case 'forbidden':
      return 'Operação não autorizada.';
    default:
      return 'O serviço está indisponível. Tente novamente.';
  }
}

class TunnelPanel extends StatelessWidget {
  const TunnelPanel({super.key, required this.controller});
  final TunnelController controller;
  @override
  Widget build(BuildContext context) => AnimatedBuilder(
    animation: controller,
    builder: (context, _) {
      final active = controller.state == TunnelState.active;
      final label = switch (controller.state) {
        TunnelState.disconnected => 'VPN desligada',
        TunnelState.connecting => 'Iniciando VPN…',
        TunnelState.active => 'VPN ativa no aparelho',
        TunnelState.disconnecting => 'Desligando VPN…',
        TunnelState.failed => 'Falha na VPN',
      };
      return Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(label, style: Theme.of(context).textTheme.titleMedium),
              if (active)
                const Text(
                  'O túnel local está ativo. Acesso ao servidor e internet precisam de validação.',
                ),
              if (controller.message != null) Text(controller.message!),
              const SizedBox(height: 8),
              FilledButton.icon(
                onPressed: controller.busy
                    ? null
                    : active
                    ? controller.disconnect
                    : controller.connect,
                icon: Icon(active ? Icons.stop : Icons.power_settings_new),
                label: Text(active ? 'Desligar VPN' : 'Ligar VPN'),
              ),
            ],
          ),
        ),
      );
    },
  );
}
