const configuredAPIURL = import.meta.env.VITE_API_URL || '/api'

export const env = {
  apiURL: configuredAPIURL.replace(/\/$/, ''),
  pollingInterval: 3_000,
} as const
