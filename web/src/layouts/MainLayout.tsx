import { useEffect, useState } from 'react'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { Layout, message } from 'antd'
import { useAuthStore, canWrite } from '@/stores/auth'
import { useEnvironmentStore } from '@/stores/environment'
import { useThemeStore } from '@/stores/theme'
import { authApi } from '@/api/auth'
import { environmentApi } from '@/api/environment'
import { syncApi, type EnvSyncStatus } from '@/api/sync'
import type { Environment, EnvironmentCreateRequest } from '@/types'
import AppHeader from './components/AppHeader'
import AppSidebar from './components/AppSidebar'
import EnvironmentManager from './components/EnvironmentManager'
import PasswordModal, { type PasswordValues } from './components/PasswordModal'
import SyncRestorePanel from './components/SyncRestorePanel'

const { Content } = Layout

export default function MainLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { user, fetchProfile, logout } = useAuthStore()
  const { environments, current, fetch: fetchEnvs, setCurrent } = useEnvironmentStore()
  const themeMode = useThemeStore((state) => state.mode)
  const setThemeMode = useThemeStore((state) => state.setMode)

  const [pwdOpen, setPwdOpen] = useState(false)
  const [pwdLoading, setPwdLoading] = useState(false)
  const [envOpen, setEnvOpen] = useState(false)
  const [syncStatuses, setSyncStatuses] = useState<EnvSyncStatus[]>([])
  const [syncModalOpen, setSyncModalOpen] = useState(false)
  const [selectedSyncEnvs, setSelectedSyncEnvs] = useState<string[]>([])
  const [restoring, setRestoring] = useState(false)

  useEffect(() => {
    fetchProfile()
    fetchEnvs()
  }, [fetchProfile, fetchEnvs])

  useEffect(() => {
    if (user?.is_super) {
      syncApi.check()
        .then((statuses) => {
          setSyncStatuses(statuses.filter((status) => status.need_restore))
        })
        .catch(() => { /* ignore */ })
    }
  }, [user?.is_super])

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const handleEnvironmentChange = (id: string) => {
    const environment = environments.find((item) => item.id === id)
    if (environment) setCurrent(environment)
  }

  const handleChangePassword = async (values: PasswordValues) => {
    setPwdLoading(true)
    try {
      await authApi.changePassword(values)
      message.success('密码修改成功')
      setPwdOpen(false)
    } catch (error: unknown) {
      message.error(error instanceof Error ? error.message : '修改失败')
    } finally {
      setPwdLoading(false)
    }
  }

  const handleEnvSave = async (values: EnvironmentCreateRequest, editing: Environment | null) => {
    try {
      if (editing) {
        await environmentApi.update(editing.id, values)
        message.success('更新成功')
      } else {
        await environmentApi.create(values)
        message.success('创建成功')
      }
      fetchEnvs()
      return true
    } catch (error: unknown) {
      message.error(error instanceof Error ? error.message : '操作失败')
      return false
    }
  }

  const handleEnvDelete = async (id: string) => {
    try {
      await environmentApi.delete(id)
      message.success('删除成功')
      fetchEnvs()
    } catch (error: unknown) {
      message.error(error instanceof Error ? error.message : '删除失败')
    }
  }

  const canManageEnv = canWrite(user, 'environments')

  const openSyncModal = () => {
    setSelectedSyncEnvs(syncStatuses.map((status) => status.environment_id))
    setSyncModalOpen(true)
  }

  const handleRestore = async () => {
    if (selectedSyncEnvs.length === 0) {
      message.warning('请选择要恢复的环境')
      return
    }
    setRestoring(true)
    try {
      const results = await syncApi.restore(selectedSyncEnvs)
      const totalSuccess = results.reduce((sum, result) => sum + result.success, 0)
      const totalFailed = results.reduce((sum, result) => sum + (result.failed?.length ?? 0), 0)
      if (totalFailed > 0) {
        message.warning(`恢复完成：成功 ${totalSuccess} 个，失败 ${totalFailed} 个`)
      } else {
        message.success(`恢复完成：共 ${totalSuccess} 个配置`)
      }
      setSyncModalOpen(false)
      setSyncStatuses([])
    } catch (error: unknown) {
      message.error(error instanceof Error ? error.message : '恢复失败')
    } finally {
      setRestoring(false)
    }
  }

  return (
    <Layout className="app-shell">
      <AppSidebar user={user} pathname={location.pathname} onNavigate={navigate} />
      <Layout className="app-workspace">
        <AppHeader
          environments={environments}
          current={current}
          user={user}
          canManageEnvironment={canManageEnv}
          themeMode={themeMode}
          onEnvironmentChange={handleEnvironmentChange}
          onManageEnvironment={() => setEnvOpen(true)}
          onThemeChange={setThemeMode}
          onChangePassword={() => setPwdOpen(true)}
          onLogout={handleLogout}
        />
        <SyncRestorePanel
          statuses={syncStatuses}
          selectedIds={selectedSyncEnvs}
          open={syncModalOpen}
          restoring={restoring}
          onOpen={openSyncModal}
          onClose={() => setSyncModalOpen(false)}
          onDismiss={() => setSyncStatuses([])}
          onSelectionChange={setSelectedSyncEnvs}
          onRestore={handleRestore}
        />
        <Content className="app-content"><Outlet /></Content>
      </Layout>
      <PasswordModal
        open={pwdOpen}
        loading={pwdLoading}
        onCancel={() => setPwdOpen(false)}
        onSubmit={handleChangePassword}
      />
      <EnvironmentManager
        open={envOpen}
        environments={environments}
        canManage={canManageEnv}
        onClose={() => setEnvOpen(false)}
        onDelete={handleEnvDelete}
        onSave={handleEnvSave}
      />
    </Layout>
  )
}
