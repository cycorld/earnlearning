import * as React from 'react'
import { cn } from '@/lib/utils'

export type CoinTone = 'gain' | 'loss' | 'neutral'
export type CoinSize = 'sm' | 'md' | 'lg'

const sizeStyles: Record<CoinSize, string> = {
  sm: 'h-5 px-2 text-[11px] gap-0.5',
  md: 'h-6 px-2.5 text-xs gap-1',
  lg: 'h-8 px-3 text-sm gap-1.5',
}

const toneStyles: Record<CoinTone, string> = {
  gain: 'bg-highlight/15 text-highlight border-highlight/25',
  loss: 'bg-coral/15 text-coral border-coral/25',
  neutral: 'bg-muted text-muted-foreground border-border',
}

function resolveTone(amount: number, showSign: boolean): CoinTone {
  if (!showSign) return 'gain'
  if (amount > 0) return 'gain'
  if (amount < 0) return 'loss'
  return 'neutral'
}

export interface CoinChipProps extends React.HTMLAttributes<HTMLSpanElement> {
  amount: number
  size?: CoinSize
  showSign?: boolean
}

export function CoinChip({
  amount,
  size = 'md',
  showSign = false,
  className,
  ...props
}: CoinChipProps) {
  const tone = resolveTone(amount, showSign)
  const formatted = new Intl.NumberFormat('ko-KR').format(Math.abs(amount))
  const prefix = showSign ? (amount > 0 ? '+' : amount < 0 ? '-' : '') : ''

  return (
    <span
      data-slot="coin-chip"
      data-tone={tone}
      data-size={size}
      className={cn(
        'inline-flex items-center rounded-full border font-semibold tabular-nums transition-colors',
        sizeStyles[size],
        toneStyles[tone],
        className,
      )}
      {...props}
    >
      <span aria-hidden className="leading-none">💰</span>
      <span>
        {prefix}
        {formatted}
      </span>
    </span>
  )
}
