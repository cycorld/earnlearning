import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  StreakBadge,
  LevelChip,
  CoinChip,
  SkillTree,
  LEVELS,
  type SkillNode,
} from '@/components/gamification'

const INITIAL_NODES: SkillNode[] = [
  { id: 'w1', label: '1주차 — 환경', status: 'completed' },
  { id: 'w2', label: '2주차 — 변수', status: 'completed' },
  { id: 'w3', label: '3주차 — 함수', status: 'available' },
  { id: 'w4', label: '4주차 — 조건문', status: 'locked' },
  { id: 'w5', label: '5주차 — 반복문', status: 'locked' },
]

export default function GamificationShowcasePage() {
  const [nodes, setNodes] = useState<SkillNode[]>(INITIAL_NODES)
  const [selected, setSelected] = useState<string | null>(null)

  const handleSelect = (id: string) => {
    setSelected(id)
    // 선택된 노드가 available 이면 완료 처리하고 다음을 available 로
    setNodes((prev) => {
      const idx = prev.findIndex((n) => n.id === id)
      if (idx < 0 || prev[idx].status !== 'available') return prev
      const next = prev.map((n, i) => {
        if (i === idx) return { ...n, status: 'completed' as const }
        if (i === idx + 1 && n.status === 'locked') return { ...n, status: 'available' as const }
        return n
      })
      return next
    })
  }

  const reset = () => {
    setNodes(INITIAL_NODES)
    setSelected(null)
  }

  return (
    <div className="container mx-auto max-w-4xl px-4 py-8 space-y-6">
      <div className="flex items-center gap-3">
        <Link to="/developer">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="h-4 w-4" />
            Developer
          </Button>
        </Link>
        <h1>게임화 컴포넌트 쇼케이스</h1>
      </div>
      <p className="text-sm text-muted-foreground">
        Khan Academy / Duolingo 감성의 재사용 블록. 학생 참여 동기를 구조적으로 올리기 위한
        UI 프리미티브 4종.
      </p>

      <Card>
        <CardHeader>
          <CardTitle>StreakBadge — 연속 접속/제출</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-wrap items-center gap-3">
          <StreakBadge days={0} />
          <StreakBadge days={1} />
          <StreakBadge days={3} />
          <StreakBadge days={7} />
          <StreakBadge days={21} />
          <StreakBadge days={99} label="일 제출" />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>LevelChip — 창업가 등급</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-wrap items-center gap-3">
          {LEVELS.map((level) => (
            <LevelChip key={level} level={level} />
          ))}
          <span className="w-full text-xs text-muted-foreground mt-2">아이콘 숨김:</span>
          {LEVELS.map((level) => (
            <LevelChip key={`plain-${level}`} level={level} showIcon={false} />
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>CoinChip — 보상 / 손익</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-center gap-3">
            <CoinChip amount={10} size="sm" />
            <CoinChip amount={250} size="md" />
            <CoinChip amount={9_999_999} size="lg" />
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <CoinChip amount={500} showSign />
            <CoinChip amount={-1200} showSign />
            <CoinChip amount={0} showSign />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>SkillTree — 주차별 해금 경로</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col items-center space-y-6 py-6">
          <SkillTree nodes={nodes} onSelect={handleSelect} />
          <div className="flex items-center gap-3 text-xs text-muted-foreground">
            {selected && <span>선택: <strong>{selected}</strong></span>}
            <Button variant="outline" size="sm" onClick={reset}>
              초기화
            </Button>
          </div>
          <p className="text-xs text-muted-foreground max-w-md text-center">
            available(주황) 노드를 눌러 진행. completed(초록) 로 바뀌면서 다음 노드가
            자동으로 열린다. locked 노드는 클릭해도 반응하지 않는다.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
