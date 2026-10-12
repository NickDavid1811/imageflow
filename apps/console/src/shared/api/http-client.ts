import { env } from '../config/env'

interface APIErrorBody {
  error?: { code?: string; message?: string }
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${env.apiURL}${path}`, init)
  const body = (await response.json()) as T | APIErrorBody

  if (!response.ok) {
    const error = body as APIErrorBody
    throw new Error(error.error?.message ?? `La API respondió con ${response.status}`)
  }
  return body as T
}

export async function isAvailable(path: string): Promise<boolean> {
  try {
    const response = await fetch(`${env.apiURL}${path}`)
    return response.ok
  } catch {
    return false
  }
}
