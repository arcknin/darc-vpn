"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import type { AdminAccountMetadata } from "../lib/openapi-types";
import { DashboardApiError, dashboardApi } from "../lib/client-api";
import { AccountForm } from "./AccountForm";
import { LogoutButton } from "./LogoutButton";

export function AccountList() {
  const [accounts, setAccounts] = useState<AdminAccountMetadata[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function refresh() {
    setLoading(true);
    setError(null);
    try {
      const result = await dashboardApi.listAccounts();
      setAccounts(result.accounts);
    } catch (caught) {
      if (caught instanceof DashboardApiError && caught.status === 401) {
        window.location.assign("/login");
        return;
      }
      setError(caught instanceof DashboardApiError ? caught.message : "não foi possível carregar as contas");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void refresh();
  }, []);

  return (
    <main className="shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">VPN Access</p>
          <h1>Contas administrativas</h1>
        </div>
        <LogoutButton />
      </header>
      <div className="columns">
        <AccountForm onCreated={(account) => setAccounts((current) => [account, ...current])} />
        <section className="card" aria-labelledby="accounts-title">
          <div className="row-between">
            <div>
              <p className="eyebrow">Acesso</p>
              <h2 id="accounts-title">Contas</h2>
            </div>
            <button className="secondary" type="button" onClick={() => void refresh()} disabled={loading}>
              Atualizar
            </button>
          </div>
          {loading ? <p className="muted">Carregando…</p> : null}
          {error ? <p className="error" role="alert">{error}</p> : null}
          {!loading && !error && accounts.length === 0 ? (
            <p className="muted">Nenhuma conta criada.</p>
          ) : null}
          <div className="list">
            {accounts.map((account) => (
              <Link className="account-row" href={`/admin/accounts/${encodeURIComponent(account.id)}`} key={account.id}>
                <span>
                  <strong>{account.username}</strong>
                  <small>{account.profileCount} perfil(is) · principal: {account.profiles.find((profile) => profile.isPrimary)?.displayName ?? "—"}</small>
                </span>
                <span className={`pill pill-${account.status}`}>{account.status}</span>
              </Link>
            ))}
          </div>
        </section>
      </div>
    </main>
  );
}
