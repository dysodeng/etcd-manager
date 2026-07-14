import {
  DesktopOutlined,
  KeyOutlined,
  LogoutOutlined,
  MoonOutlined,
  SettingOutlined,
  SunOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { Button, Dropdown, Layout, Select, Space, Tag, Typography } from 'antd'
import type { ThemeMode } from '@/stores/theme'
import type { Environment, UserProfile } from '@/types'

const { Header } = Layout
const { Text } = Typography

interface AppHeaderProps {
  environments: Environment[]
  current: Environment | null
  user: UserProfile | null
  canManageEnvironment: boolean
  themeMode: ThemeMode
  onEnvironmentChange: (id: string) => void
  onManageEnvironment: () => void
  onThemeChange: (mode: ThemeMode) => void
  onChangePassword: () => void
  onLogout: () => void
}

export default function AppHeader({
  environments,
  current,
  user,
  canManageEnvironment,
  themeMode,
  onEnvironmentChange,
  onManageEnvironment,
  onThemeChange,
  onChangePassword,
  onLogout,
}: AppHeaderProps) {
  const userLabel = user?.is_super ? '超级管理员' : (user?.role?.name ?? '无角色')
  const themeIcon = themeMode === 'dark'
    ? <MoonOutlined />
    : themeMode === 'system' ? <DesktopOutlined /> : <SunOutlined />

  return (
    <Header className="app-header">
      <Space>
        <KeyOutlined />
        <Select
          value={current?.id}
          onChange={onEnvironmentChange}
          options={environments.map((environment) => ({ label: environment.name, value: environment.id }))}
          style={{ width: 180 }}
          placeholder="选择环境"
        />
        {canManageEnvironment && (
          <Button icon={<SettingOutlined />} onClick={onManageEnvironment}>管理环境</Button>
        )}
      </Space>
      <div className="app-header__actions">
        <Dropdown
          menu={{
            items: [
              { key: 'light', icon: <SunOutlined />, label: '浅色' },
              { key: 'dark', icon: <MoonOutlined />, label: '深色' },
              { key: 'system', icon: <DesktopOutlined />, label: '跟随系统' },
            ],
            selectedKeys: [themeMode],
            onClick: ({ key }) => onThemeChange(key as ThemeMode),
          }}
        >
          <Button
            type="text"
            aria-label="切换主题"
            icon={themeIcon}
          />
        </Dropdown>
        <Dropdown
          menu={{
            items: [
              { key: 'password', icon: <SettingOutlined />, label: '修改密码' },
              { key: 'logout', icon: <LogoutOutlined />, label: '退出登录' },
            ],
            onClick: ({ key }) => {
              if (key === 'logout') onLogout()
              if (key === 'password') onChangePassword()
            },
          }}
        >
          <Space className="app-header__account">
            <UserOutlined />
            <Text>{user?.username}</Text>
            <Tag color={user?.is_super ? 'red' : 'blue'}>{userLabel}</Tag>
          </Space>
        </Dropdown>
      </div>
    </Header>
  )
}
