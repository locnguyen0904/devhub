import { ApiError } from "@/shared/api/client";

import { useReadiness } from "./api";

interface RowProps {
  label: string;
  ok: boolean;
}

function Row({ label, ok }: RowProps) {
  return (
    <li className="flex items-center justify-between border-b border-border-subtle py-3 last:border-0">
      <span className="text-text-muted">{label}</span>
      <span
        className={ok ? "font-medium text-success" : "font-medium text-danger"}
      >
        {/* Colour is never the only signal: the words carry the state too. */}
        <span aria-hidden="true">●</span> {ok ? "ok" : "not responding"}
      </span>
    </li>
  );
}

function Skeleton() {
  return (
    <ul aria-busy="true">
      {["api", "postgres", "redis"].map((name) => (
        <li
          key={name}
          className="flex items-center justify-between border-b border-border-subtle py-3 last:border-0"
        >
          <span className="text-text-muted">{name}</span>
          <span className="h-4 w-20 rounded bg-surface-sunken" />
        </li>
      ))}
    </ul>
  );
}

export function StatusPanel() {
  const { data, error, isPending } = useReadiness();

  return (
    <section className="rounded-[--radius-card] border border-border-subtle bg-surface-raised p-6 shadow-[--shadow-card]">
      <h2 className="mb-1 text-xl font-semibold">System status</h2>
      <p className="mb-4 text-sm text-text-subtle">
        Data travels from Postgres through sqlc, the service, huma, to this view.
      </p>

      {isPending ? (
        <Skeleton />
      ) : error ? (
        <div role="alert" className="text-danger">
          <p className="font-medium">
            {error instanceof ApiError ? error.message : "Could not reach the API"}
          </p>
          <p className="mt-1 text-sm text-text-subtle">
            Check that <code className="font-mono">make dev</code> is running.
          </p>
        </div>
      ) : (
        <ul>
          {/* The API answered, so the API itself is up by definition. */}
          <Row label="api" ok />
          {data.checks.map((check) => (
            <Row key={check.name} label={check.name} ok={check.ok} />
          ))}
        </ul>
      )}
    </section>
  );
}
