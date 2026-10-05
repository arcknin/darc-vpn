# Feature Index: vpn-access

<!-- managed_by: spec-drive; legacy task artifacts preserved -->

Derived task navigation and current task state only. Task artifacts remain authoritative.

## Tasks

- `production-target` — Próximo passo: draft para preservar peers/PSKs legados e preparar stack final; aprovação pendente — [spec](production-target/spec-01-production-target.md), [checkpoint](production-target/production-target-checkpoint.yaml)

- `migration-scripts` — Pacote Bash entregue ao codex na VPS; dono root executará inventário/backup. Corte bloqueado até implantação e restauração legada validadas — [spec](migration-scripts/spec-01-scripts.md), [checkpoint](migration-scripts/migration-scripts-checkpoint.yaml), [validação](migration-scripts/validation.md)

- `wireguard-stack` — Concluída: dois containers, controle direto wgctrl e laboratório VPS aprovados com TDD; produção não migrada. [Evidência](wireguard-stack/validation.md) — [spec](wireguard-stack/spec-01-stack.md), [discovery](wireguard-stack/discovery.md), [checkpoint](wireguard-stack/wireguard-stack-checkpoint.yaml)
- `mvp` — Retomada verificada em 2026-10-04: B06 APK e B05 E2E pendentes; depois B07/B08. Cerca de 18 GiB livres. Integração live/container exige decisão separada — [discovery](mvp/discovery.md), [contract](mvp/mvp-contract.md), [plan](mvp/mvp-plan.md), [checkpoint](mvp/mvp-checkpoint.yaml), [history](mvp/mvp-history.md)
