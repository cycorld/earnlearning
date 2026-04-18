import { useCallback, useEffect, useState } from 'react'
import { api } from '@/lib/api'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ArrowLeft, GripVertical, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Link } from 'react-router-dom'

interface KanbanTask {
  id: string
  title: string
  description: string
  priority: string
  type: string
  status: string
  filename: string
}

const COLUMNS = [
  { key: 'backlog', label: 'Backlog', color: 'bg-gray-100' },
  { key: 'todo', label: 'To Do', color: 'bg-blue-50' },
  { key: 'in-progress', label: 'In Progress', color: 'bg-yellow-50' },
  { key: 'done', label: 'Done', color: 'bg-green-50' },
]

const PRIORITY_COLORS: Record<string, string> = {
  high: 'bg-red-100 text-red-700',
  medium: 'bg-yellow-100 text-yellow-700',
  low: 'bg-gray-100 text-gray-700',
}

const TYPE_COLORS: Record<string, string> = {
  feat: 'bg-purple-100 text-purple-700',
  fix: 'bg-orange-100 text-orange-700',
  chore: 'bg-gray-100 text-gray-700',
  content: 'bg-blue-100 text-blue-700',
}

export default function AdminTasksPage() {
  const [tasks, setTasks] = useState<KanbanTask[]>([])
  const [loading, setLoading] = useState(true)
  const [expandedId, setExpandedId] = useState<string | null>(null)

  const fetchTasks = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.get<KanbanTask[]>('/admin/tasks')
      setTasks(res ?? [])
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchTasks()
  }, [fetchTasks])

  const getColumnTasks = (status: string) =>
    tasks.filter((t) => t.status === status)

  return (
    <div className="mx-auto max-w-6xl space-y-5 p-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Link to="/admin" className="rounded-full p-1 hover:bg-accent">
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <GripVertical className="h-5 w-5 text-primary" />
          <h1 className="text-xl font-bold">태스크 보드</h1>
          <Badge variant="secondary" className="text-xs">
            {tasks.length}개
          </Badge>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">읽기 전용</span>
          <Button size="sm" variant="outline" onClick={fetchTasks}>
            <RefreshCw className="mr-1 h-3.5 w-3.5" />
            새로고침
          </Button>
        </div>
      </div>

      {loading ? (
        <div className="flex justify-center py-8">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-4">
          {COLUMNS.map((col) => (
            <div key={col.key} className={`rounded-lg ${col.color} p-3`}>
              <div className="mb-3 flex items-center justify-between">
                <h2 className="text-sm font-semibold">{col.label}</h2>
                <Badge variant="outline" className="text-xs">
                  {getColumnTasks(col.key).length}
                </Badge>
              </div>
              <div className="space-y-2">
                {getColumnTasks(col.key).map((task) => (
                  <Card
                    key={task.id + task.filename}
                    className="cursor-pointer transition-shadow hover:shadow-md"
                    onClick={() =>
                      setExpandedId(
                        expandedId === task.id ? null : task.id
                      )
                    }
                  >
                    <CardContent className="p-3">
                      <div className="mb-1 flex items-start justify-between gap-1">
                        <p className="text-sm font-medium leading-tight">
                          {task.title}
                        </p>
                        <span className="shrink-0 text-[10px] text-muted-foreground">
                          #{task.id}
                        </span>
                      </div>
                      <div className="flex items-center gap-1">
                        <Badge
                          variant="secondary"
                          className={`text-[10px] ${PRIORITY_COLORS[task.priority] ?? ''}`}
                        >
                          {task.priority}
                        </Badge>
                        <Badge
                          variant="secondary"
                          className={`text-[10px] ${TYPE_COLORS[task.type] ?? ''}`}
                        >
                          {task.type}
                        </Badge>
                      </div>
                      {expandedId === task.id && task.description && (
                        <div className="mt-2 border-t pt-2">
                          <pre className="whitespace-pre-wrap text-xs text-muted-foreground font-sans leading-relaxed">
                            {task.description.slice(0, 500)}
                            {task.description.length > 500 && '...'}
                          </pre>
                          <p className="mt-2 text-[10px] text-muted-foreground/60">
                            {task.filename}
                          </p>
                        </div>
                      )}
                    </CardContent>
                  </Card>
                ))}
                {getColumnTasks(col.key).length === 0 && (
                  <p className="py-4 text-center text-xs text-muted-foreground">
                    비어있음
                  </p>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      <p className="text-center text-xs text-muted-foreground">
        tasks/ 폴더의 마크다운 파일을 읽어 표시합니다. 수정은 Claude 또는 직접 파일 편집으로 진행하세요.
      </p>
    </div>
  )
}
