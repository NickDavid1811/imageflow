import type { CreateJobInput, ImageFormat } from './job'

export interface JobFormValues {
  sourceURL: string
  width: string
  height: string
  format: ImageFormat
  quality: number
}

export const initialJobForm: JobFormValues = {
  sourceURL: 'https://images.unsplash.com/photo-1542291026-7eec264c27ff',
  width: '1200',
  height: '',
  format: 'webp',
  quality: 82,
}

export function validateJobForm(values: JobFormValues): string | null {
  if (!values.sourceURL.trim()) return 'Ingresa una URL de imagen válida.'
  try {
    const source = new URL(values.sourceURL)
    if (!['http:', 'https:'].includes(source.protocol)) return 'La imagen debe usar una URL HTTP o HTTPS.'
  } catch {
    return 'Ingresa una URL de imagen válida.'
  }
  if (!values.width && !values.height) return 'Define al menos el ancho o el alto de salida.'
  if (Number(values.width) < 0 || Number(values.height) < 0) return 'Las dimensiones no pueden ser negativas.'
  return null
}

export function toCreateJobInput(values: JobFormValues): CreateJobInput {
  return {
    source_url: values.sourceURL.trim(),
    operations: [
      { type: 'resize', width: values.width ? Number(values.width) : undefined, height: values.height ? Number(values.height) : undefined },
      { type: 'convert', format: values.format, quality: values.quality },
    ],
  }
}

export function describeDimensions(values: Pick<JobFormValues, 'width' | 'height'>): string {
  if (!values.width && !values.height) return 'Sin redimensionar'
  return `${values.width || 'auto'} × ${values.height || 'auto'} px`
}
