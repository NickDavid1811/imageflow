const configuredAPIURL = import.meta.env.VITE_API_URL || '/api'

export const env = {
  apiURL: configuredAPIURL.replace(/\/$/, ''),
  apiKey: import.meta.env.VITE_IMAGEFLOW_API_KEY || '',
  pollingInterval: 3_000,
} as const
