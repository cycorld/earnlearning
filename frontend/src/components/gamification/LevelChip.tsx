import * as React from 'react'
import { cn } from '@/lib/utils'

export const LEVELS = ['Seed', 'Bronze', 'Silver', 'Gold', 'Diamond'] as const
export type Level = (typeof LEVELS)[number]

// Khan Academy / Duolingo 풍: 레벨별로 또렷이 구분되는 팔레트.
// Seed 는 Ewha Green 톤, 중간은 금속 톤, Diamond 는 info(푸른 보석) 톤.
const LEVEL_STYLES: Record<Level, { chip: string; icon: string; emoji: string }> = {
  Seed: {
    chip: 'bg-success/15 text-success border-success/30',
    icon: 'bg-success text-white',
    emoji: '🌱',
  },
  Bronze: {
    chip: 'bg-[#B06A2A]/15 text-[#8B5020] border-[#B06A2A]/30',
    icon: 'bg-[#B06A2A] text-white',
    emoji: '🥉',
  },
  Silver: {
    chip: 'bg-muted text-foreground border-border',
    icon: 'bg-[#9AA0A6] text-white',
    emoji: '🥈',
  },
  Gold: {
    chip: 'bg-warning/20 text-[#8A6513] border-warning/40',
    icon: 'bg-warning text-white',
    emoji: '🥇',
  },
  Diamond: {
    chip: 'bg-info/15 text-info border-info/30',
    icon: 'bg-info text-white',
    emoji: '💎',
  },
}

export interface LevelChipProps extends React.HTMLAttributes<HTMLSpanElement> {
  level: Level
  showIcon?: boolean
}

export function LevelChip({
  level,
  showIcon = true,
  className,
  ...props
}: LevelChipProps) {
  const style = LEVEL_STYLES[level]
  return (
    <span
      data-slot="level-chip"
      data-level={level}
      className={cn(
        'inline-flex h-6 items-center gap-1.5 rounded-full border pl-0.5 pr-2.5 text-xs font-semibold',
        !showIcon && 'pl-2.5',
        style.chip,
        className,
      )}
      {...props}
    >
      {showIcon && (
        <span
          data-slot="level-icon"
          aria-hidden
          className={cn(
            'inline-flex h-5 w-5 items-center justify-center rounded-full text-xs',
            style.icon,
          )}
        >
          {style.emoji}
        </span>
      )}
      <span>{level}</span>
    </span>
  )
}
