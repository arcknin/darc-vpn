# Darc VPN Android

Cliente Flutter somente para Android 7/API 24 ou superior. Implementa login,
termos, seleção de perfis e QR, além da ponte nativa WireGuard para iniciar,
observar e parar o túnel. Selecionar um perfil não conecta. Compilação e
validação no aparelho têm evidências próprias no
[checkpoint de integração](../docs/features/vpn-access/app-integration/app-integration-checkpoint.yaml).

## Configuração local

Informe a URL da API explicitamente em `API_BASE_URL`. O cliente exige HTTPS;
HTTP só pode ser usado em desenvolvimento local com
`ALLOW_INSECURE_API=true` em build debug. Nenhum hostname de API ou dashboard é presumido pelo
app.

```sh
flutter pub get
flutter run --dart-define=API_BASE_URL=https://api.example.invalid
```

O token de sessão usa `flutter_secure_storage`. A preferência comum guarda
somente o `profile_id` selecionado. Configuração privada e QR são recuperados
sob demanda após o aceite persistido e não são registrados pelo cliente.
Para o teste privado autorizado, `PRIVATE_TEST=true` em debug permite seguir
sem aceite quando a API indica esse modo; não registra consentimento fictício.
O [roteiro Android USB](../docs/features/vpn-access/app-integration/android-usb-test.md)
descreve o acesso à API privada por SSH e `adb reverse`.

## Verificação

```sh
dart format --output=none --set-exit-if-changed .
flutter analyze
flutter test
flutter build apk --debug
```

Os testes B06 usam uma API fake e validam as rotas/modelos definidos em
`../back-vpn/api/openapi.yaml`. Eles não usam contas, servidor ou configuração
WireGuard reais.
