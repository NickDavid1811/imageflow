import type { FormEvent } from 'react'
import { ArrowIcon, LinkIcon, PlusIcon } from '../../../shared/icons'
import { describeDimensions, type JobFormValues } from '../model/job-form'

interface Props {
  error: string
  values: JobFormValues
  submitting: boolean
  onChange: <K extends keyof JobFormValues>(field: K, value: JobFormValues[K]) => void
  onSubmit: () => void
}

export function CreateJobForm({ error, values, submitting, onChange, onSubmit }: Props) {
  function handleSubmit(event: FormEvent) {
    event.preventDefault()
    onSubmit()
  }

  return (
    <form onSubmit={handleSubmit} className="panel overflow-hidden">
      <div className="panel-heading">
        <div><p className="section-kicker">Nuevo trabajo</p><h2 className="section-title">Configura la salida</h2></div>
        <div className="step-chip"><PlusIcon className="size-4" /> Job</div>
      </div>

      <div className="space-y-7 p-5 sm:p-7">
        <fieldset>
          <label htmlFor="source" className="label">Imagen de origen</label>
          <div className="input-shell mt-2"><LinkIcon className="size-4 shrink-0 text-muted" /><input id="source" type="url" required value={values.sourceURL} onChange={(event) => onChange('sourceURL', event.target.value)} className="field" placeholder="https://ejemplo.com/producto.jpg" /></div>
          <p className="hint">ImageFlow descargará la imagen desde esta URL cuando el worker procese el trabajo.</p>
        </fieldset>

        <div className="h-px bg-line" />

        <fieldset>
          <div className="mb-3 flex items-end justify-between"><div><legend className="label">Dimensiones</legend><p className="hint mt-1">El campo vacío mantendrá la proporción original.</p></div><span className="rounded-lg bg-sand px-2.5 py-1 text-xs font-semibold text-pine">{describeDimensions(values)}</span></div>
          <div className="grid grid-cols-2 gap-3">
            <label className="subfield"><span>Ancho</span><div><input inputMode="numeric" min="1" type="number" value={values.width} onChange={(event) => onChange('width', event.target.value)} placeholder="Auto" /><em>px</em></div></label>
            <label className="subfield"><span>Alto</span><div><input inputMode="numeric" min="1" type="number" value={values.height} onChange={(event) => onChange('height', event.target.value)} placeholder="Auto" /><em>px</em></div></label>
          </div>
        </fieldset>

        <fieldset>
          <legend className="label">Formato y compresión</legend>
          <div className="mt-3 grid gap-5 sm:grid-cols-[180px_1fr]">
            <label className="subfield"><span>Formato</span><select value={values.format} onChange={(event) => onChange('format', event.target.value as JobFormValues['format'])}><option value="webp">WebP</option><option value="avif">AVIF</option><option value="jpeg">JPEG</option><option value="png">PNG</option></select></label>
            <label className="range-field"><span><b>Calidad</b><strong>{values.quality}%</strong></span><input type="range" min="1" max="100" value={values.quality} onChange={(event) => onChange('quality', Number(event.target.value))} /><small><i>Menor peso</i><i>Mayor detalle</i></small></label>
          </div>
        </fieldset>

        {error && <div role="alert" className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-300">{error}</div>}

        <button disabled={submitting} className="primary-button group" type="submit">
          <span>{submitting ? 'Creando trabajo…' : 'Procesar imagen'}</span>
          <ArrowIcon className="size-4 transition-transform group-hover:translate-x-0.5" />
        </button>
      </div>
    </form>
  )
}
