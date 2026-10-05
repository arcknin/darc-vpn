"use client";

import { useState } from "react";

export function LogoutButton() {
  const [pending, setPending] = useState(false);

  async function logout() {
    setPending(true);
    try {
      await fetch("/api/session/logout", { method: "POST" });
    } finally {
      window.location.assign("/login");
    }
  }

  return (
    <button className="secondary" type="button" onClick={logout} disabled={pending}>
      {pending ? "Saindo…" : "Sair"}
    </button>
  );
}
