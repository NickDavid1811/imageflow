import { authorizedFetch, isAvailable, request } from "../../../shared/api/http-client";
import type { CreateJobInput, Job } from "../model/job";

export function createJob(input: CreateJobInput): Promise<Job> {
  return request<Job>("/v1/jobs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
}

export function getJob(id: string): Promise<Job> {
  return request<Job>(`/v1/jobs/${encodeURIComponent(id)}`);
}

export async function downloadJobResult(id: string): Promise<Blob> {
  const response = await authorizedFetch(`/v1/jobs/${encodeURIComponent(id)}/result`);
  if (!response.ok) throw new Error(`No se pudo descargar el resultado (${response.status}).`);
  return response.blob();
}

export function checkAPI(): Promise<boolean> {
  return isAvailable("/health/ready");
}
