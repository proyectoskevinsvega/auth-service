import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'react-hot-toast'
import { AppLayout } from '@/layouts/AppLayout'
import { PrivateRoute } from '@/components/PrivateRoute'
import { LoginPage } from '@/pages/LoginPage'
import { DashboardPage } from '@/pages/DashboardPage'
import { UsersPage } from '@/pages/UsersPage'
import { RBACPage } from '@/pages/RBACPage'
import { CompliancePage } from '@/pages/CompliancePage'
import { CertificatesPage } from '@/pages/CertificatesPage'
import { AuditLogsPage } from '@/pages/AuditLogsPage'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 30_000,
    },
  },
})

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route element={<PrivateRoute />}>
            <Route element={<AppLayout />}>
              <Route index element={<DashboardPage />} />
              <Route path="users" element={<UsersPage />} />
              <Route path="rbac" element={<RBACPage />} />
              <Route path="compliance" element={<CompliancePage />} />
              <Route path="certificates" element={<CertificatesPage />} />
              <Route path="audit-logs" element={<AuditLogsPage />} />
            </Route>
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
      <Toaster
        position="bottom-right"
        toastOptions={{
          style: {
            background: '#1e2433',
            color: '#e2e8f0',
            border: '1px solid rgba(100,116,139,0.3)',
            fontSize: '13px',
          },
          success: { iconTheme: { primary: '#34d399', secondary: '#1e2433' } },
          error: { iconTheme: { primary: '#f87171', secondary: '#1e2433' } },
        }}
      />
    </QueryClientProvider>
  )
}
