import { Navigate, Outlet } from 'react-router-dom'
import { useAuth } from '@/hooks/use-auth'
import { Spinner } from '@/components/ui/spinner'

export function ApprovedGuard() {
  const { user, isLoading } = useAuth()

  if (isLoading) {
    return (
      <div className="flex min-h-dvh items-center justify-center">
        <Spinner />
      </div>
    )
  }

  if (!user) {
    return <Navigate to="/login" replace />
  }

  if (user.status !== 'approved') {
    return <Navigate to="/pending" replace />
  }

  return <Outlet />
}
