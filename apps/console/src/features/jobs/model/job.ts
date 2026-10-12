export type ImageFormat = 'webp' | 'avif' | 'jpeg' | 'png'

export type ImageOperation =
  | { type: 'resize'; width?: number; height?: number }
  | { type: 'convert'; format: ImageFormat; quality: number }

export type JobStatus = 'queued' | 'processing' | 'completed' | 'failed' | 'cancelled'

export interface Job {
  id: string
  source_url: string
  operations: ImageOperation[]
  status: JobStatus
  progress: number
  attempts: number
  error_code?: string
  error_message?: string
  created_at: string
  started_at?: string
  completed_at?: string
  updated_at: string
}

export interface CreateJobInput {
  source_url: string
  operations: ImageOperation[]
}

export const statusLabels: Record<JobStatus, string> = {
  queued: 'En cola',
  processing: 'Procesando',
  completed: 'Completado',
  failed: 'Falló',
  cancelled: 'Cancelado',
}

export function isActiveJob(job: Job): boolean {
  return job.status === 'queued' || job.status === 'processing'
}
