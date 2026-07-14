import { lazy, Suspense, useEffect, type ReactNode } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ConfigProvider, Spin } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { useAuthStore } from '@/stores/auth'
import { getDefaultRoute } from '@/config/menu'
import { useIsDark } from '@/stores/theme'
import { createAppTheme } from '@/theme'

const MainLayout = lazy(() => import('@/layouts/MainLayout'))
const LoginPage = lazy(() => import('@/pages/login'))
const KVPage = lazy(() => import('@/pages/kv'))
const ConfigPage = lazy(() => import('@/pages/config'))
const ClusterPage = lazy(() => import('@/pages/cluster'))
const UsersPage = lazy(() => import('@/pages/users'))
const RolesPage = lazy(() => import('@/pages/roles'))
const AuditPage = lazy(() => import('@/pages/audit'))
const GatewayPage = lazy(() => import('@/pages/gateway'))
const GrpcPage = lazy(() => import('@/pages/grpc'))

function ProtectedRoute({ children }: { children: ReactNode }) {
  const token = useAuthStore((s) => s.token)
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}

function DefaultRedirect() {
  const user = useAuthStore((s) => s.user)
  return <Navigate to={getDefaultRoute(user)} replace />
}

export default function App() {
  const isDark = useIsDark()

  useEffect(() => {
    document.documentElement.dataset.theme = isDark ? 'dark' : 'light'
  }, [isDark])

  return (
    <ConfigProvider
      locale={zhCN}
      theme={createAppTheme(isDark)}
    >
      <BrowserRouter>
        <Suspense fallback={<Spin fullscreen tip="加载中..." />}>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route
              path="/"
              element={
                <ProtectedRoute>
                  <MainLayout />
                </ProtectedRoute>
              }
            >
              <Route index element={<DefaultRedirect />} />
              <Route path="kv" element={<KVPage />} />
              <Route path="config" element={<ConfigPage />} />
              <Route path="cluster" element={<ClusterPage />} />
              <Route path="users" element={<UsersPage />} />
              <Route path="roles" element={<RolesPage />} />
              <Route path="audit" element={<AuditPage />} />
              <Route path="gateway" element={<GatewayPage />} />
              <Route path="grpc" element={<GrpcPage />} />
            </Route>
          </Routes>
        </Suspense>
      </BrowserRouter>
    </ConfigProvider>
  )
}
