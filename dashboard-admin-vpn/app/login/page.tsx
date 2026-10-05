import { LoginForm } from "../../src/components/LoginForm";

export default function LoginPage() {
  return (
    <main className="auth-shell">
      <section className="card auth-card">
        <p className="eyebrow">VPN Access</p>
        <h1>Administração</h1>
        <p className="muted">Entre com a credencial administrativa para gerenciar contas e perfis.</p>
        <LoginForm />
      </section>
    </main>
  );
}
