import { useEffect } from 'react'
import { env } from '../../../shared/config/env'
import { getJob } from '../api/jobs-api'
import { isActiveJob, type Job } from '../model/job'

export function useJobPolling(job: Job | null, onUpdate: (job: Job) => void) {
  useEffect(() => {
    if (!job || !isActiveJob(job)) return
    const timer = window.setInterval(async () => {
      try {
        onUpdate(await getJob(job.id))
      } catch {
        // Temporary connectivity failures should not discard visible job data.
      }
    }, env.pollingInterval)
    return () => window.clearInterval(timer)
  }, [job, onUpdate])
}
