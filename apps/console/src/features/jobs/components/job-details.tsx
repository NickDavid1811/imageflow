import { useState } from "react";
import {
  ActivityIcon,
  ArrowIcon,
  CheckIcon,
  ClockIcon,
  CopyIcon,
  ImageIcon,
  RefreshIcon,
} from "../../../shared/icons";
import { statusLabels, type Job } from "../model/job";
import { env } from "../../../shared/config/env";

interface Props {
  job: Job | null;
  refreshing: boolean;
  onRefresh: () => void;
}

export function JobDetails({ job, refreshing, onRefresh }: Props) {
  const [copied, setCopied] = useState(false);

  async function copyID() {
    if (!job) return;
    await navigator.clipboard.writeText(job.id);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1500);
  }

  return (
    <aside className="panel min-h-[590px] overflow-hidden">
      <div className="panel-heading">
        <div>
          <p className="section-kicker">Actividad</p>
          <h2 className="section-title">Trabajo actual</h2>
        </div>
        {job && (
          <button
            onClick={onRefresh}
            disabled={refreshing}
            className="icon-button"
            aria-label="Actualizar trabajo"
          >
            <RefreshIcon
              className={`size-4 ${refreshing ? "animate-spin" : ""}`}
            />
          </button>
        )}
      </div>

      {!job ? (
        <div className="grid min-h-[480px] place-items-center p-8 text-center">
          <div className="max-w-xs">
            <div className="mx-auto grid size-16 place-items-center rounded-2xl border border-dashed border-line bg-sand text-muted">
              <ActivityIcon className="size-7" />
            </div>
            <h3 className="mt-5 font-display text-xl font-semibold">
              Aún no hay actividad
            </h3>
            <p className="mt-2 text-sm leading-6 text-muted">
              Crea un trabajo para ver aquí su identificador, progreso y
              operaciones.
            </p>
          </div>
        </div>
      ) : (
        <div className="p-5 sm:p-7">
          <div className="flex items-start justify-between gap-4">
            <div>
              <span className={`status status-${job.status}`}>
                <span />
                {statusLabels[job.status]}
              </span>
              <h3 className="mt-4 font-display text-2xl font-semibold tracking-tight">
                Procesamiento de imagen
              </h3>
            </div>
            <div className="grid size-12 place-items-center rounded-2xl bg-sand text-pine">
              <ImageIcon className="size-6" />
            </div>
          </div>

          <div className="mt-7">
            <div className="mb-2 flex justify-between text-xs font-semibold">
              <span className="text-muted">Progreso</span>
              <span>{job.progress}%</span>
            </div>
            <div className="h-2 overflow-hidden rounded-full bg-sand">
              <div
                className="h-full rounded-full bg-coral transition-all duration-500"
                style={{
                  width: `${Math.max(job.progress, job.status === "queued" ? 6 : 0)}%`,
                }}
              />
            </div>
          </div>

          <dl className="mt-7 divide-y divide-line rounded-2xl border border-line bg-surface">
            <div className="detail-row">
              <dt>ID del trabajo</dt>
              <dd>
                <code>{job.id}</code>
                <button
                  onClick={copyID}
                  className="copy-button"
                  aria-label="Copiar ID"
                >
                  {copied ? <CheckIcon /> : <CopyIcon />}
                </button>
              </dd>
            </div>
            <div className="detail-row">
              <dt>Creado</dt>
              <dd>
                <ClockIcon className="size-4 text-muted" />
                {new Intl.DateTimeFormat("es-PE", {
                  dateStyle: "medium",
                  timeStyle: "short",
                }).format(new Date(job.created_at))}
              </dd>
            </div>
            <div className="detail-row">
              <dt>Intentos</dt>
              <dd>{job.attempts}</dd>
            </div>
          </dl>

          {job.output && (
            <a
              className="mt-5 flex w-full items-center justify-center rounded-xl bg-pine px-4 py-3 text-sm font-semibold text-white transition hover:opacity-90"
              href={`${env.apiURL}${job.output.url}`}
              target="_blank"
              rel="noreferrer"
            >
              Ver imagen procesada · {job.output.width} × {job.output.height}
            </a>
          )}

          <div className="mt-6">
            <p className="section-kicker">Pipeline solicitado</p>
            <div className="mt-3 flex flex-wrap items-center gap-2">
              <span className="operation-pill">Original</span>
              {job.operations.map((operation, index) => (
                <span className="contents" key={`${operation.type}-${index}`}>
                  <ArrowIcon className="size-3 text-muted" />
                  <span className="operation-pill uppercase">
                    {describeOperation(operation)}
                  </span>
                </span>
              ))}
            </div>
          </div>

          <div className="code-panel mt-6 overflow-hidden rounded-2xl">
            <div className="code-panel-heading flex items-center justify-between px-4 py-3">
              <span className="text-xs font-semibold">Payload</span>
              <span className="code-panel-type font-mono text-[10px]">
                application/json
              </span>
            </div>
            <pre className="overflow-x-auto p-4 font-mono text-[11px] leading-5">
              {JSON.stringify(job, null, 2)}
            </pre>
          </div>
        </div>
      )}
    </aside>
  );
}

function describeOperation(operation: Job["operations"][number]): string {
  if (operation.type === "resize")
    return `${operation.width || "auto"} × ${operation.height || "auto"} px · ${operation.fit || "contain"}`;
  if (operation.type === "crop")
    return `crop ${operation.width} × ${operation.height} @ ${operation.x},${operation.y}`;
  return `${operation.format} · ${operation.quality}%`;
}
