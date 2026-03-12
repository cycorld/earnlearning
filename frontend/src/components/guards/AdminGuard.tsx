import { Navigate, Outlet } from 'react-router-dom'
import { useAuth } from '@/hooks/use-auth'

export default function AdminGuard() {
  const { user } = useAuth()

  if (user?.role !== 'admin') {
    return <Navigate to="/feed" replace />
  }

  return <Outlet />
}
