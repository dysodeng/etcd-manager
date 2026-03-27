import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import MainLayout from '@/layouts/MainLayout'
import LoginPage from '@/pages/login'
import KVPage from '@/pages/kv'
import ConfigPage from '@/pages/config'
import ClusterPage from '@/pages/cluster'
import UsersPage from '@/pages/users'
import RolesPage from '@/pages/roles'
import AuditPage from '@/pages/audit'
import GatewayPage from '@/pages/gateway'
import GrpcPage from '@/pages/grpc'
import { useAuthStore } from '@/stores/auth'
import { getDefaultRoute } from '@/config/menu'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.token)
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}

function DefaultRedirect() {
  const user = useAuthStore((s) => s.user)
  return <Navigate to={getDefaultRoute(user)} replace />
}

export default function App() {
  return (
    <ConfigProvider locale={zhCN}>
      <BrowserRouter>
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
      </BrowserRouter>
    </ConfigProvider>
  )
}
