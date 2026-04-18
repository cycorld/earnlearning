import * as React from 'react'
import { cn } from '@/lib/utils'

export type SkillStatus = 'locked' | 'available' | 'completed'

export interface SkillNode {
  id: string
  label: string
  status: SkillStatus
  hint?: string
}

const nodeStyles: Record<SkillStatus, string> = {
  completed:
    'bg-primary text-primary-foreground border-primary shadow-[0_3px_0_0_var(--primary-shadow)]',
  available:
    'bg-highlight text-highlight-foreground border-highlight shadow-[0_3px_0_0_var(--highlight-shadow)] animate-pulse',
  locked:
    'bg-muted text-muted-foreground border-border',
}

const nodeEmoji: Record<SkillStatus, string> = {
  completed: '✓',
  available: '→',
  locked: '🔒',
}

export interface SkillTreeProps
  extends Omit<React.HTMLAttributes<HTMLDivElement>, 'onSelect'> {
  nodes: SkillNode[]
  onSelect?: (id: string) => void
}

export function SkillTree({ nodes, onSelect, className, ...props }: SkillTreeProps) {
  return (
    <div
      data-slot="skill-tree"
      className={cn('flex flex-col items-center gap-0', className)}
      {...props}
    >
      {nodes.map((node, idx) => {
        const clickable = onSelect && node.status !== 'locked'
        const handleClick = () => {
          if (clickable) onSelect(node.id)
        }

        return (
          <React.Fragment key={node.id}>
            <button
              type="button"
              data-slot="skill-node"
              data-status={node.status}
              disabled={!clickable}
              onClick={handleClick}
              className={cn(
                'relative flex h-14 w-14 items-center justify-center rounded-full border-2 text-lg font-bold transition-transform',
                nodeStyles[node.status],
                clickable && 'cursor-pointer hover:scale-105 active:translate-y-px active:shadow-none',
                !clickable && 'cursor-default',
              )}
              aria-label={`${node.label} — ${node.status}`}
            >
              <span aria-hidden>{nodeEmoji[node.status]}</span>
              <span className="absolute -bottom-5 text-[10px] font-semibold text-foreground whitespace-nowrap">
                {node.label}
              </span>
            </button>
            {idx < nodes.length - 1 && (
              <span
                data-slot="skill-connector"
                data-active={String(node.status === 'completed')}
                aria-hidden
                className={cn(
                  'my-5 h-8 w-1 rounded-full',
                  node.status === 'completed' ? 'bg-primary' : 'bg-border',
                )}
              />
            )}
          </React.Fragment>
        )
      })}
    </div>
  )
}
