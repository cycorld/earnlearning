import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { Toaster } from '@/components/ui/sonner'
import App from './App'
import { initAnalytics } from '@/lib/analytics'
import './index.css'

// GA4 — production + VITE_GA_ID 있을 때만 실제 동작 (개발 중 noop).
initAnalytics()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
    <Toaster />
  </StrictMode>,
)
