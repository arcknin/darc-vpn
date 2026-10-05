"use client";

import Link from "next/link";
import { FormEvent, useEffect, useState } from "react";
import type { AdminAccountMetadata, RevocationOperation } from "../lib/openapi-types";
import { DashboardApiError, dashboardApi } from "../lib/client-api";
import { StatusBadge } from "./StatusBadge";

export function AccountDetail({ accountId }: { accountId: string }) {
  const [account, setAccount] = useState<AdminAccountMetadata | null>(null);
  const [operation, setOperation] = useState<RevocationOperation | null>(null);
  const [displayName, setDisplayName] = useState("");
  const [loading, setLoading] = useState(true);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const result = await dashboardApi.getAccount(accountId);
      setAccount(result);
      try {
        setOperation(await dashboardApi.getRevocation(accountId));
      } catch (caught) {
        if (!(caught instanceof DashboardApiError && caught.status === 404)) throw caught;
      }
    } catch (caught) {
      if (caught instanceof DashboardApiError && caught.status === 401) {
        window.location.assign("/login");
        return;
      }
      setError(caught instanceof DashboardApiError ? caught.message : "não foi possível carregar a conta");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [accountId]);

  async function createProfile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError(null);
    try {
      const profile = await dashboardApi.createProfile(accountId, { displayName });
      setAccount((current) =>
        current
          ? { ...current, profileCount: current.profileCount + 1, profiles: [...current.profiles, profile] }
          : current,
      );
      setDisplayName("");
    } catch (caught) {
      setError(caught instanceof DashboardApiError ? caught.message : "não foi possível criar o perfil");
    } finally {
      setPending(false);
    }
  }

  async function block() {
    setPending(true);
    setError(null);
    try {
      // The server response is the source of truth. No local success state is inferred.
      setOperation(await dashboardApi.blockAccount(accountId));
      setAccount((current) => (current ? { ...current, status: "blocked" } : current));
    } catch (caught) {
      setError(caught instanceof DashboardApiError ? caught.message : "não foi possível bloquear a conta");
    } finally {
      setPending(false);
    }
  }

  async function refreshOperation() {
    setPending(true);
    setError(null);
    try {
      setOperation(await dashboardApi.getRevocation(accountId));
    } catch (caught) {
      setError(caught instanceof DashboardApiError ? caught.message : "não foi possível consultar a revogação");
    } finally {
      setPending(false);
    }
  }

  if (loading) {
    return <main className="shell"><p className="muted">Carregando…</p></main>;
  }
  if (!account) {
    return <main className="shell"><p className="error" role="alert">{error ?? "conta não encontrada"}</p><Link href="/admin">Voltar às contas</Link></main>;
  }

  return (
    <main className="shell">
      <header className="topbar">
        <div>
          <Link className="back" href="/admin">← Contas</Link>
          <p className="eyebrow">Conta administrativa</p>
          <h1>{account.username}</h1>
        </div>
        <span className={`pill pill-${account.status}`}>{account.status}</span>
      </header>
      {error ? <p className="error" role="alert">{error}</p> : null}
      <div className="columns">
        <section className="card" aria-labelledby="profiles-title">
          <div className="row-between">
            <div>
              <p className="eyebrow">Metadados</p>
              <h2 id="profiles-title">Perfis ({account.profileCount})</h2>
            </div>
            <button className="danger" type="button" onClick={() => void block()} disabled={pending || account.status === "blocked"}>
              Bloquear conta
            </button>
          </div>
          <p className="muted">Somente nomes e metadados administrativos são exibidos nesta interface.</p>
          <div className="list">
            {account.profiles.map((profile) => (
              <article className="profile-row" key={profile.id}>
                <span>
                  <strong>{profile.displayName}</strong>
                  <small>{profile.isPrimary ? "perfil principal" : "perfil extra"}</small>
                </span>
                <span className="pill pill-neutral">{profile.isPrimary ? "principal" : "extra"}</span>
              </article>
            ))}
          </div>
          <form className="inline-form" onSubmit={createProfile}>
            <label>
              Nome do novo perfil
              <input required value={displayName} onChange={(event) => setDisplayName(event.target.value)} />
            </label>
            <button type="submit" disabled={pending}>Adicionar perfil</button>
          </form>
        </section>
        <section className="card" aria-labelledby="revocation-title">
          <div className="row-between">
            <div>
              <p className="eyebrow">Revogação</p>
              <h2 id="revocation-title">Estado do bloqueio</h2>
            </div>
            <button className="secondary" type="button" onClick={() => void refreshOperation()} disabled={pending}>
              Consultar
            </button>
          </div>
          {operation ? <StatusBadge operation={operation} /> : <p className="muted">Nenhuma operação de bloqueio registrada.</p>}
          {operation ? (
            <ul className="peer-list">
              {operation.peers.map((peer) => (
                <li key={`${peer.profileId}-${peer.peerId}`}>
                  <span>{peer.profileId}</span>
                  <span className={peer.removed ? "ok" : "error"}>{peer.removed ? "removed" : "not removed"}</span>
                </li>
              ))}
            </ul>
          ) : null}
        </section>
      </div>
    </main>
  );
}
