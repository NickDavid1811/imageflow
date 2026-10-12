import { useEffect, useState } from 'react'
import { checkAPI } from '../api/jobs-api'

export function APIStatus() {
  const [online, setOnline] = useState<boolean | null>(null)

  useEffect(() => {
    const refresh = () => { checkAPI().then(setOnline) }
    refresh()
    const timer = window.setInterval(refresh, 10_000)
    return () => window.clearInterval(timer)
  }, [])

  return (
    <span className={`inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs font-semibold ${online === true ? 'border-emerald-200 bg-emerald-50 text-emerald-700' : online === false ? 'border-red-200 bg-red-50 text-red-700' : 'border-line bg-white text-muted'}`}>
      <span className={`size-1.5 rounded-full ${online === true ? 'bg-emerald-500' : online === false ? 'bg-red-500' : 'bg-slate-400'}`} />
      {online === true ? 'Conectada' : online === false ? 'Sin conexión' : 'Verificando'}
    </span>
  )
}
