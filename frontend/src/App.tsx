import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from '@/hooks/use-auth'
import { ErrorBoundary } from '@/components/ErrorBoundary'

// Layouts
import MainLayout from '@/components/layout/MainLayout'
import AuthLayout from '@/components/layout/AuthLayout'

// Guards
import AuthGuard from '@/components/guards/AuthGuard'
import AdminGuard from '@/components/guards/AdminGuard'

// Auth pages
import LoginPage from '@/routes/auth/LoginPage'
import RegisterPage from '@/routes/auth/RegisterPage'
import PendingPage from '@/routes/auth/PendingPage'

// Main pages
import FeedPage from '@/routes/feed/FeedPage'
import WalletPage from '@/routes/wallet/WalletPage'
import TransactionsPage from '@/routes/wallet/TransactionsPage'
import MarketPage from '@/routes/market/MarketPage'
import MarketNewPage from '@/routes/market/MarketNewPage'
import MarketDetailPage from '@/routes/market/MarketDetailPage'
import CompanyListPage from '@/routes/company/CompanyListPage'
import CompanyNewPage from '@/routes/company/CompanyNewPage'
import CompanyDetailPage from '@/routes/company/CompanyDetailPage'
import BusinessCardPage from '@/routes/company/BusinessCardPage'
import InvestPage from '@/routes/invest/InvestPage'
import InvestDetailPage from '@/routes/invest/InvestDetailPage'
import ExchangePage from '@/routes/exchange/ExchangePage'
import ExchangeDetailPage from '@/routes/exchange/ExchangeDetailPage'
import BankPage from '@/routes/bank/BankPage'
import LoanApplyPage from '@/routes/bank/LoanApplyPage'
import ProfilePage from '@/routes/profile/ProfilePage'
import UserProfilePage from '@/routes/profile/UserProfilePage'
import NotificationsPage from '@/routes/notifications/NotificationsPage'

// Grants
import GrantListPage from '@/routes/grant/GrantListPage'
import GrantDetailPage from '@/routes/grant/GrantDetailPage'
import GrantNewPage from '@/routes/grant/GrantNewPage'

// Changelog
import ChangelogPage from '@/routes/changelog/ChangelogPage'

// Admin pages
import AdminPage from '@/routes/admin/AdminPage'
import AdminUsersPage from '@/routes/admin/AdminUsersPage'
import AdminClassroomPage from '@/routes/admin/AdminClassroomPage'
import AdminLoansPage from '@/routes/admin/AdminLoansPage'
import AdminKpiPage from '@/routes/admin/AdminKpiPage'

export default function App() {
  return (
    <ErrorBoundary>
    <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <AuthProvider>
        <Routes>
          {/* Public */}
          <Route element={<AuthLayout />}>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />
          </Route>
          <Route path="/pending" element={<PendingPage />} />

          {/* Protected */}
          <Route element={<AuthGuard />}>
            <Route element={<MainLayout />}>
              <Route path="/feed" element={<FeedPage />} />
              <Route path="/wallet" element={<WalletPage />} />
              <Route path="/wallet/transactions" element={<TransactionsPage />} />
              <Route path="/market" element={<MarketPage />} />
              <Route path="/market/new" element={<MarketNewPage />} />
              <Route path="/market/:id" element={<MarketDetailPage />} />
              <Route path="/company" element={<CompanyListPage />} />
              <Route path="/company/new" element={<CompanyNewPage />} />
              <Route path="/company/:id" element={<CompanyDetailPage />} />
              <Route path="/company/:id/card" element={<BusinessCardPage />} />
              <Route path="/invest" element={<InvestPage />} />
              <Route path="/invest/:id" element={<InvestDetailPage />} />
              <Route path="/exchange" element={<ExchangePage />} />
              <Route path="/exchange/:id" element={<ExchangeDetailPage />} />
              <Route path="/grant" element={<GrantListPage />} />
              <Route path="/grant/new" element={<GrantNewPage />} />
              <Route path="/grant/:id" element={<GrantDetailPage />} />
              <Route path="/bank" element={<BankPage />} />
              <Route path="/bank/apply" element={<LoanApplyPage />} />
              <Route path="/profile" element={<ProfilePage />} />
              <Route path="/profile/:id" element={<UserProfilePage />} />
              <Route path="/notifications" element={<NotificationsPage />} />
              <Route path="/changelog" element={<ChangelogPage />} />

              <Route element={<AdminGuard />}>
                <Route path="/admin" element={<AdminPage />} />
                <Route path="/admin/users" element={<AdminUsersPage />} />
                <Route path="/admin/classroom" element={<AdminClassroomPage />} />
                <Route path="/admin/loans" element={<AdminLoansPage />} />
                <Route path="/admin/kpi" element={<AdminKpiPage />} />
              </Route>
            </Route>
          </Route>

          <Route path="*" element={<Navigate to="/feed" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
    </ErrorBoundary>
  )
}
