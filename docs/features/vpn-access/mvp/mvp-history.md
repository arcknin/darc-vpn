# Task History: mvp

<!-- managed_by: orchestration-flow -->

Compact closed-work entries managed by `orchestration-flow`. This file preserves outcomes, evidence, limitations, and residual risk; it does not preserve agent conversation or transient reasoning.

## 2026-09-29 — Preliminary contract draft

- Outcome: documented the owner's direct requirements and proposed acceptance scenarios in `mvp-contract.md`; all accidental form answers remain excluded.
- Evidence: the local workspace was empty and had no Git repository; official WireGuard and Android documentation established that tunnel integration and peer access need separate handling.
- Limitation: the contract is a draft with material decisions, TDD, and owner approval pending. No application code or live server change was made.

## 2026-09-29 — Owner annotation update

- Outcome: updated the draft with Android 7/API 24+ in-app tunnel control, automatic configuration after login, admin-only account creation, and separate WireGuard peer/key per device; kept the VPN endpoint and dashboard host distinct.
- Evidence: direct owner annotations; official Android and WireGuard documentation linked in the contract supports platform minimum and the per-device peer rationale.
- Limitation: acceptance scenarios and TDD are not approved; key handling, device authorization, revocation, exact dashboard/API host, and live server integration remain open. No source code or server change was made.

## 2026-09-29 — Optional QR profile export

- Outcome: the draft now permits QR export of the primary or additional WireGuard profile for import into the official Apple WireGuard app. Separate slots have separate peers/keys, while copying one profile reuses its peer/key for alternate use and triggers an in-app warning about simultaneous-use instability.
- Evidence: direct owner annotation and official WireGuard iPhone/iPad QR import and protocol documentation linked in the contract.
- Limitation: each QR exposes a reusable private-key credential. Key custody/recovery, slot limits/creation, revocation, ATDD approval, and TDD choice remain open. No source code or live server change was made.

## 2026-09-29 — Owner confirmation before independent review

- Outcome: recorded backend encryption and recovery of private profiles, one initial profile per account, extra profiles added only by the administrator, full account-profile revocation, and explicit TDD yes. The draft is frozen for the owner-requested independent contract review.
- Evidence: direct owner confirmation and frozen contract SHA-256 `6d4ac584cf6f54b2881139eeb1562cfac6c610dffc459361157870caf289437e`.
- Limitation: contract and ATDD are not approved; no code, server change, or deployment was performed. Remaining details are documented in the draft.

## 2026-09-29 — Independent contract review

- Outcome: new read-only agent reviewed the complete frozen draft at SHA-256 `6d4ac584cf6f54b2881139eeb1562cfac6c610dffc459361157870caf289437e`; TDD yes was confirmed correctly recorded. The draft is not yet ready for full approval.
- Evidence: review identified two owner behavior decisions (Android profile selection and revocation timing/effect) and missing acceptance for Android VPN consent, negative authorization, encrypted authenticated transport, and secret custody. The review also recommends narrowing Apple QR evidence to iPhone/iPad.
- Limitation: no contract amendment, source code, server access, or deployment followed the review. Owner decisions and complete ATDD approval remain pending.

## 2026-09-29 — Review findings incorporated into proposed ATDD

- Outcome: owner chose connection through the last selected profile. The contract now requires fast server peer removal attempts on block and only reports completed revocation after server confirmation; proposed ATDD covers Android permission, negative authorization, encrypted transport, and encrypted-key recovery. Revision 1 is `ready_for_owner_approval`.
- Evidence: direct owner annotations and updated contract SHA-256 `467b6716a609b84f3893af22611520c3746bdc0c129a51ae405e7d89866efdee`; TDD yes preserved.
- Limitation: proposed ATDD still needs owner approval. No source code, server access, DNS change, or deployment was performed.

## 2026-09-29 — Contract approved with initial terms requirement

- Outcome: owner approved revision 2 and its ATDD, adding electronic acceptance before profile/QR delivery, persistent JSON proof, access IP/time metadata, and no stored browsing history. TDD yes and local planning/implementation authority were preserved.
- Evidence: direct owner approval in this chat; approved contract SHA-256 `477a54dadf1fd064c5b4e2fe8c355b8d51ac157ca33537e7f8393cfb6972b683`; terms text remains at `terms-of-use-draft.md`.
- Limitation: terms need operator/contact/retention before publication and logging must match implementation; no code, tests, server access, or production action has occurred. Next step is code planning.

## 2026-09-29 — Frozen local code plan

- Outcome: complete code plan revision 1 binds to approved contract revision 2 and TDD yes; eight functional batches cover backend, admin dashboard, Android app, and local acceptance evidence.
- Evidence: `mvp-plan.md` SHA-256 `481bee806bd631096eb9f6b44311ec850903df7980c44d8b06a631b318505bcc`; planner validated structure and ownership. B01 is the next execution batch.
- Limitation: no source code or tests yet. Real WireGuard, logging infrastructure, DNS, and production proof remain outside local authority.

## 2026-09-29 — B01 backend foundation and HTTP contract

- Outcome: Go module, versioned OpenAPI, domain/port boundaries, route metadata, local fakes, and contract checks were created in `back-vpn/` under the B01 scope.
- Evidence: executor recorded expected red for absent OpenAPI; root reran `GOCACHE=/tmp/darc-vpn-go-cache go test ./...`, `go vet ./...`, and `gofmt -l cmd internal test` successfully. Contract/plan hashes remained unchanged.
- Limitation: identity, profile persistence, terms, revocation, clients, and live WireGuard behavior are not yet implemented. Next B02 owns backend identity and encrypted profiles.

## 2026-09-29 — Plan scope correction before B02

- Outcome: plan revision 2 adds Go module manifests to B02 and Flutter lockfile to B06 allowed scopes; behavior, contract revision 2, TDD yes, and eight-batch order are unchanged. B01 remains complete.
- Evidence: `mvp-plan.md` SHA-256 `2501468dec43e83ffe6883d2479e10b9b0ff0b62b33676cba297f196feb22044`; contract SHA-256 unchanged.
- Limitation: B02 has not started; external dependency resolution may be constrained by the sandbox. Next action is B02.

## 2026-09-29 — B02 identity and encrypted profiles

- Outcome: SQLite migrations, account/session services, principal and extra profiles, Argon2id, digest-only session tokens, independent WireGuard keys, and AES-256-GCM private profile storage were implemented in `back-vpn/` within B02 ownership.
- Evidence: executor recorded expected red before implementation; root reran `GOCACHE=/tmp/darc-vpn-go-cache go test -race ./...` and `go vet ./...` successfully, with gofmt clean. Temporary SQLite migration/reopen and negative authorization/crypto tests passed. Contract/plan hashes unchanged.
- Limitation: `cmd/api` still runs the B01 reserved handler because B02 did not own executable composition; B03/B04 plan scopes need to own that wiring. No real server or production data used.

## 2026-09-29 — Plan correction for executable HTTP composition

- Outcome: plan revision 3 assigns `cmd/api` composition to B03 and B04 and requires black-box HTTP tests against the executable; B01/B02 completion, contract revision 2, and TDD yes remain unchanged.
- Evidence: `mvp-plan.md` SHA-256 `0561be3fbda9c670f0c24a8bbf0995f76701d9fc9df71d47c873a8a1308a1ee7`; YAML parsed and contract hash unchanged.
- Limitation: B03 has not started. All validation remains local with temporary data and no live WireGuard controller.

## 2026-09-29 — B03 terms and access metadata

- Outcome: backend gate requires initial terms acceptance before profiles/QR, records an atomic JSON receipt, logs approved access metadata, and composes real B02/B03 services in `cmd/api`.
- Evidence: executor recorded expected red; root reran `go test -race ./...`, where package tests passed but loopback bind was sandbox-denied, then reran `go test -race ./cmd/api` with local loopback permission and it passed. `go vet ./...` and gofmt passed. The black-box test covers login, terms, 428 pre-acceptance, profiles, private recovery, and restart.
- Limitation: terms fixture is fictional; the editorial draft is not published. VPN event logging is only a local port/fake, not live server evidence.

## 2026-09-29 — B04 account block and simulated peer revocation

- Outcome: blocking persists account/session denial before controller call, snapshots every peer, and reports pending/failed/completed only from active plus persistent removal confirmation; executable HTTP routes expose these states.
- Evidence: executor recorded expected red; root reran `go test -race ./...` with permitted local loopback, `go vet ./...`, and gofmt successfully. Tests cover partial confirmation, failure, idempotency, concurrency, and black-box API behavior.
- Limitation: controller is a fake. No actual WireGuard peer was removed; the live effect remains unverified and unauthorized.

## 2026-09-29 — B05 dashboard local implementation

- Outcome: Next.js admin dashboard implements server-side login proxy, secure session cookie, account/profile operations, and faithful pending/failed/completed revocation display without private profile material.
- Evidence: after a reproducibility correction, project-local `npm ci --offline` installed dependencies; root reran `npm test` (15 tests), `npm run typecheck`, and `npm run lint` successfully. Executor reported a successful local Next build.
- Limitation: Playwright/browser E2E was skipped because unavailable locally; it remains outstanding. Project dependencies consume about 389 MiB while system disk is nearly full. No live backend or deploy was used.

## 2026-09-29 — Read-only VPS and container migration discovery

- Outcome: owner-authorized SSH as `codex` confirmed host `wg0` active and host IPv6 HTTPS working; owner-provided root reports identified three peers, recent handshake activity, IPv4/IPv6 hooks, broad duplicate NAT/firewall rules, and the current UDP port. Docker's default bridge lacks IPv6. A separate dual-stack container test and rollback path were scoped without changing the server.
- Evidence: owner-provided root-shell reports and bounded `codex` SSH reads summarized in `mvp-checkpoint.yaml`; independent local source reading found no live peer provisioning, only fake revocation control, a 253-value IPv4 address generator without uniqueness, and client `::/0` without an IPv6 address.
- Limitation: no network or container was created; no WG configuration or secrets were read by the agent; no client tunnel was tested. Moving active `wg0` and integrating the backend require a separately approved contract/operations plan. The local B06 build remains independently resumable.

## 2026-09-29 — Disposable WireGuard container feasibility lab

- Outcome: a temporary dual-stack Docker bridge and two Ubuntu-derived containers on alternate UDP 51443 demonstrated WireGuard interface creation with bounded NET_ADMIN/NET_RAW, new handshakes through VPS IPv4 and IPv6 endpoints, 2/2 bidirectional ICMP packets in each tunnel family, and working IPv4 internet egress. An initial IPv6 egress attempt failed without retained stderr; a later instrumented cycle succeeded with verified HTTPS IPv6 HTTP 200 through the tunnel and NAT66 counter advancement. A plain Alpine bridge also reached HTTPS over IPv6.
- Evidence: executor's bounded lab outputs and instrumented follow-up, plus independent post-lab `codex` SSH verification of zero lab containers/networks/images, unbound UDP 51443, and original host `wg-quick@wg0` active with unchanged interface addresses. No real peers or secrets were used.
- Limitation: the peer was another container on the same VPS, so independent external-client access was not proven. The first IPv6 failure's cause remains unknown; selected later IPv6 egress worked. No persistent deployment or backend integration was created, and the active VPN was not changed.

## 2026-09-29 — Temporary Mac WireGuard client profile

- Outcome: a second disposable WireGuard server is running on VPS UDP 51443 with fresh lab identities, dual-stack Docker bridge, and only NET_ADMIN/NET_RAW at runtime. A split-tunnel profile was placed on the owner's Mac at `/private/tmp/darc-vpn-lab-mac.conf` with mode 0600. The host production `wg0` remains active.
- Evidence: worker confirmed WireGuard interface and peer configuration, IPv4/IPv6 UDP publication, and a VPS-side nohup cleanup process surviving a fresh SSH session. Root independently checked the 340-byte local file, lab container and published port, and production service state. Cleanup is due at 2026-09-29T16:11:11Z.
- Limitation: the Mac profile has not been activated, so no external handshake or traffic has been demonstrated. The temporary credential must be deleted locally after the test. The cleanup process may not survive a VPS reboot, although the container has restart=no.

### Mac activation and external handshake

- Outcome: owner activated the WireGuard profile on the Mac; the app showed an active tunnel, a recent handshake, and bytes in both directions.
- Evidence: owner screenshot showed a handshake nine seconds earlier and 92 B received / 180 B sent. Root independently observed one recent handshake and 212 B received / 92 B sent on the lab server at 2026-09-29T15:26Z.
- Limitation: this confirms the external handshake and initial packets, but tunnel IPv4/IPv6 ping results have not been provided yet.
