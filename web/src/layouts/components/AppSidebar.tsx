import type { ReactNode } from 'react'
import {
  ApiOutlined,
  AuditOutlined,
  CloudServerOutlined,
  ClusterOutlined,
  DatabaseOutlined,
  SettingOutlined,
  TeamOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { Layout, Menu } from 'antd'
import { getVisibleMenuGroups } from '@/config/menu'
import type { UserProfile } from '@/types'

const { Sider } = Layout

const iconMap: Record<string, ReactNode> = {
  '/cluster': <ClusterOutlined />,
  '/kv': <DatabaseOutlined />,
  '/config': <SettingOutlined />,
  '/gateway': <ApiOutlined />,
  '/grpc': <CloudServerOutlined />,
  '/users': <UserOutlined />,
  '/roles': <TeamOutlined />,
  '/audit': <AuditOutlined />,
}

interface AppSidebarProps {
  user: UserProfile | null
  pathname: string
  onNavigate: (path: string) => void
  clusterSummary?: ReactNode
}

export default function AppSidebar({ user, pathname, onNavigate, clusterSummary }: AppSidebarProps) {
  const groups = getVisibleMenuGroups(user)

  return (
    <Sider className="app-sidebar" width={220} theme="dark">
      <div className="app-sidebar__brand">
        <span className="app-sidebar__brand-mark"><DatabaseOutlined /></span>
        <span>etcd manager</span>
      </div>
      <nav className="app-sidebar__groups" aria-label="主导航">
        {groups.map((group) => (
          <section key={group.key} className="app-sidebar__group">
            <div className="app-sidebar__label">{group.label}</div>
            <Menu
              mode="inline"
              theme="dark"
              selectedKeys={[pathname]}
              items={group.items.map(({ key, label }) => ({ key, label, icon: iconMap[key] }))}
              onClick={({ key }) => onNavigate(key)}
            />
          </section>
        ))}
      </nav>
      {clusterSummary && <div className="app-sidebar__summary">{clusterSummary}</div>}
    </Sider>
  )
}
