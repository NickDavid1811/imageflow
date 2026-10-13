import { isAvailable, request } from "../../../shared/api/http-client";
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

export function checkAPI(): Promise<boolean> {
  return isAvailable("/health/ready");
}
