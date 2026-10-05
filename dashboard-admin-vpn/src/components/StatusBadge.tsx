import type { RevocationOperation } from "../lib/openapi-types";
import { statusDescription } from "../lib/status";

export function StatusBadge({ operation }: { operation: RevocationOperation }) {
  return (
    <div
      className={`status status-${operation.status}`}
      data-revocation-status={operation.status}
      aria-label={`revogação ${operation.status}`}
    >
      <strong>{statusDescription(operation.status)}</strong>
      <span>Atualizado em {new Date(operation.updatedAt).toLocaleString("pt-BR")}</span>
      {operation.error ? <span>{operation.error}</span> : null}
    </div>
  );
}
