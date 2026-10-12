import { CreateJobForm } from '../components/create-job-form'
import { JobDetails } from '../components/job-details'
import { useJobsWorkspace } from '../hooks/use-jobs-workspace'

export function JobsWorkspace() {
  const workspace = useJobsWorkspace()

  return (
    <>
      <section className="mb-6 flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between lg:gap-10">
        <div className="shrink-0">
          <p className="eyebrow">Procesamiento asíncrono</p>
          <h1 className="mt-2 font-display text-2xl font-semibold tracking-[-0.03em] text-ink sm:text-3xl">Prepara imágenes para su próximo destino.</h1>
        </div>
        <p className="max-w-xl text-sm leading-6 text-muted lg:text-right">Configura una transformación, envíala a ImageFlow y sigue su estado desde un único espacio operativo.</p>
      </section>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1.15fr)_minmax(390px,.85fr)]">
        <CreateJobForm error={workspace.error} values={workspace.form} submitting={workspace.submitting} onChange={workspace.updateForm} onSubmit={workspace.submit} />
        <JobDetails job={workspace.job} refreshing={workspace.refreshing} onRefresh={workspace.refresh} />
      </div>
    </>
  )
}
