import { cn } from '@/lib/cn'

const AVATAR_COLORS = [
  'bg-teal-500',
  'bg-amber-500',
  'bg-pink-500',
  'bg-blue-500',
  'bg-violet-500',
  'bg-emerald-500',
  'bg-rose-500',
  'bg-indigo-500',
]

export function initials(name: string): string {
  return name
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((w) => w[0])
    .join('')
    .toUpperCase()
}

export function avatarColor(name: string): string {
  let h = 0
  for (const c of name) h = (h * 31 + c.charCodeAt(0)) % 997
  return AVATAR_COLORS[h % AVATAR_COLORS.length]
}

export function Avatar({ name, className }: { name: string; className?: string }) {
  return (
    <span
      className={cn(
        'flex shrink-0 items-center justify-center rounded-full text-sm font-semibold text-white',
        avatarColor(name),
        className ?? 'h-10 w-10',
      )}
    >
      {initials(name)}
    </span>
  )
}
