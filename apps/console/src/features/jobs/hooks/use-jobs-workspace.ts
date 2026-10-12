import { useCallback, useState } from 'react'
import { createJob, getJob } from '../api/jobs-api'
import { initialJobForm, toCreateJobInput, validateJobForm, type JobFormValues } from '../model/job-form'
import type { Job } from '../model/job'
import { useJobPolling } from './use-job-polling'

export function useJobsWorkspace() {
  const [form, setForm] = useState<JobFormValues>(initialJobForm)
  const [job, setJob] = useState<Job | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState('')

  const updateJob = useCallback((updated: Job) => setJob(updated), [])
  useJobPolling(job, updateJob)

  const updateForm = useCallback(<K extends keyof JobFormValues,>(field: K, value: JobFormValues[K]) => {
    setForm((current) => ({ ...current, [field]: value }))
  }, [])

  const submit = useCallback(async () => {
    setError('')
    const validationError = validateJobForm(form)
    if (validationError) {
      setError(validationError)
      return
    }
    setSubmitting(true)
    try {
      setJob(await createJob(toCreateJobInput(form)))
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'No se pudo crear el trabajo.')
    } finally {
      setSubmitting(false)
    }
  }, [form])

  const refresh = useCallback(async () => {
    if (!job) return
    setRefreshing(true)
    try {
      setJob(await getJob(job.id))
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'No se pudo consultar el trabajo.')
    } finally {
      setRefreshing(false)
    }
  }, [job])

  return { error, form, job, refreshing, submitting, refresh, submit, updateForm }
}
