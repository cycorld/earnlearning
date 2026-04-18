import * as React from 'react'
import { cn } from '@/lib/utils'

export type StreakVariant = 'cold' | 'warm' | 'hot'

function classifyStreak(days: number): StreakVariant {
  if (days <= 0) return 'cold'
  if (days < 7) return 'warm'
  return 'hot'
}

const variantStyles: Record<StreakVariant, string> = {
  cold: 'bg-muted text-muted-foreground border-border',
  warm: 'bg-highlight/15 text-highlight border-highlight/25',
  hot: 'bg-highlight text-highlight-foreground border-highlight',
}

export interface StreakBadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  days: number
  label?: string
}

export function StreakBadge({
  days,
  label = '일 연속',
  className,
  ...props
}: StreakBadgeProps) {
  const clamped = Math.max(0, Math.floor(days))
  const variant = classifyStreak(clamped)

  return (
    <span
      data-slot="streak-badge"
      data-variant={variant}
      className={cn(
        'inline-flex h-6 items-center gap-1 rounded-full border px-2.5 text-xs font-semibold tabular-nums transition-colors',
        variantStyles[variant],
        className,
      )}
      {...props}
    >
      <span aria-hidden className="text-sm leading-none">🔥</span>
      <span>
        {clamped}
        {label ? <span className="ml-0.5 font-medium">{label}</span> : null}
      </span>
    </span>
  )
}
