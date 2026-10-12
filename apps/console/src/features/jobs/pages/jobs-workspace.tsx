import { CreateJobForm } from '../components/create-job-form'
import { JobDetails } from '../components/job-details'
import { useJobsWorkspace } from '../hooks/use-jobs-workspace'

export function JobsWorkspace() {
  const workspace = useJobsWorkspace()

  return (
    <>
      <section className="mb-8 max-w-3xl">
        <p className="eyebrow">Procesamiento asíncrono</p>
        <h1 className="mt-3 font-display text-4xl font-semibold tracking-[-0.035em] text-ink sm:text-5xl">Prepara cada imagen<br className="hidden sm:block" /> para su próximo destino.</h1>
        <p className="mt-4 max-w-2xl text-base leading-7 text-muted">Configura una transformación, envíala a ImageFlow y sigue su estado desde un único espacio operativo.</p>
      </section>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1.15fr)_minmax(390px,.85fr)]">
        <CreateJobForm error={workspace.error} values={workspace.form} submitting={workspace.submitting} onChange={workspace.updateForm} onSubmit={workspace.submit} />
        <JobDetails job={workspace.job} refreshing={workspace.refreshing} onRefresh={workspace.refresh} />
      </div>
    </>
  )
}
