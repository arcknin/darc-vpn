"use client";

import { FormEvent, useState } from "react";
import type { AdminAccountMetadata } from "../lib/openapi-types";
import { DashboardApiError, dashboardApi } from "../lib/client-api";

export function AccountForm({ onCreated }: { onCreated: (account: AdminAccountMetadata) => void }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [primaryDisplayName, setPrimaryDisplayName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setPending(true);
    try {
      const account = await dashboardApi.createAccount({
        username,
        password,
        primaryDisplayName,
      });
      onCreated(account);
      setUsername("");
      setPassword("");
      setPrimaryDisplayName("");
    } catch (caught) {
      setError(caught instanceof DashboardApiError ? caught.message : "não foi possível criar a conta");
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="card stack" onSubmit={submit} aria-label="Criar conta">
      <div>
        <p className="eyebrow">Nova conta</p>
        <h2>Criar amigo</h2>
      </div>
      <label>
        Usuário
        <input required value={username} onChange={(event) => setUsername(event.target.value)} />
      </label>
      <label>
        Senha inicial
        <input
          required
          type="password"
          autoComplete="new-password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />
      </label>
      <label>
        Nome do perfil principal
        <input
          required
          value={primaryDisplayName}
          onChange={(event) => setPrimaryDisplayName(event.target.value)}
        />
      </label>
      {error ? <p className="error" role="alert">{error}</p> : null}
      <button type="submit" disabled={pending}>
        {pending ? "Criando…" : "Criar conta e perfil principal"}
      </button>
    </form>
  );
}
