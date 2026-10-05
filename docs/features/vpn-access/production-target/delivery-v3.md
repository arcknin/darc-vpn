# Teste privado sem termos — v3

Ordem explícita do dono: remover barreiras de termos/retenção do teste técnico. Escopo mantém código de autenticação/ownership/WireGuard e proteções técnicas da migração; nenhum corte executado pelo assistente.

Pacote SHA256 `651ca74d8eacb34f667eb6e51f0599cdf18770eb12faf33686ad888132df5428`, destinado a /home/codex/darc-vpn-migration-20261004-v3. Copiar/verificar para /root/darc-vpn-migration-v3 e executar 02b-build-target.sh --private-test.

Modo container-private-test inicia sem terms_file e entrega perfil autenticado sem aceite. Flag production_terms_reviewed permanece falsa; prova marca private_test_only/terms_deferred e o modo ligado ao modelo. Os outros modos ainda exigem termos. Preparação avalia colisões/network/portas e prova de kernel/cold restore; não gera aprovação legal artificial. API continua host-loopback.

Testes locais Go race/vet, 14 stdlib e ShellCheck/sintaxe aprovados. Teste root/realdata da revisão v3 será feito pelo operador; não foi executado pelo assistente.

Entrega verificada como codex: checksum do pacote e manifest completo corretos; 02b syntax/help e recusa sem root corretos; wg-quick@wg0 continua active. Executar cópia root v3 com --private-test; não reutilizar scripts v2 para prova nova.
