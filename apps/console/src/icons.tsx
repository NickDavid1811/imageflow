import type { SVGProps } from 'react'

type IconProps = SVGProps<SVGSVGElement>

const defaults = {
  fill: 'none',
  viewBox: '0 0 24 24',
  stroke: 'currentColor',
  strokeWidth: 1.8,
  strokeLinecap: 'round' as const,
  strokeLinejoin: 'round' as const,
  'aria-hidden': true,
}

export function ApertureIcon(props: IconProps) {
  return <svg {...defaults} {...props}><circle cx="12" cy="12" r="9"/><path d="m7.5 4.7 4.2 7.3M16.5 4.7H8.1M21 12h-8.4m3.9 7.3-4.2-7.3M7.5 19.3h8.4M3 12h8.4"/></svg>
}

export function PlusIcon(props: IconProps) {
  return <svg {...defaults} {...props}><path d="M12 5v14M5 12h14"/></svg>
}

export function LinkIcon(props: IconProps) {
  return <svg {...defaults} {...props}><path d="M10 13a5 5 0 0 0 7.1.1l2-2a5 5 0 0 0-7.1-7.1l-1.1 1.1M14 11a5 5 0 0 0-7.1-.1l-2 2A5 5 0 0 0 12 20l1.1-1.1"/></svg>
}

export function ActivityIcon(props: IconProps) {
  return <svg {...defaults} {...props}><path d="M3 12h4l2.5-7 5 14 2.5-7h4"/></svg>
}

export function ArrowIcon(props: IconProps) {
  return <svg {...defaults} {...props}><path d="M5 12h14m-5-5 5 5-5 5"/></svg>
}

export function CheckIcon(props: IconProps) {
  return <svg {...defaults} {...props}><path d="m5 12 4 4L19 6"/></svg>
}

export function ClockIcon(props: IconProps) {
  return <svg {...defaults} {...props}><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></svg>
}

export function CopyIcon(props: IconProps) {
  return <svg {...defaults} {...props}><rect x="8" y="8" width="11" height="11" rx="2"/><path d="M16 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h2"/></svg>
}

export function RefreshIcon(props: IconProps) {
  return <svg {...defaults} {...props}><path d="M20 7v5h-5M4 17v-5h5"/><path d="M6.1 9a7 7 0 0 1 11.6-2L20 12M4 12l2.3 5a7 7 0 0 0 11.6-2"/></svg>
}

export function ImageIcon(props: IconProps) {
  return <svg {...defaults} {...props}><rect x="3" y="4" width="18" height="16" rx="2"/><circle cx="8.5" cy="9" r="1.5"/><path d="m21 15-5-5L5 20"/></svg>
}

export function MoonIcon(props: IconProps) {
  return <svg {...defaults} {...props}><path d="M20.5 14.2A8.5 8.5 0 0 1 9.8 3.5 8.5 8.5 0 1 0 20.5 14.2Z"/></svg>
}

export function SunIcon(props: IconProps) {
  return <svg {...defaults} {...props}><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.42 1.42M17.65 17.65l1.42 1.42M2 12h2M20 12h2M4.93 19.07l1.42-1.42M17.65 6.35l1.42-1.42"/></svg>
}
