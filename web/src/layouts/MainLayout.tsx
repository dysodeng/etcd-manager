import { useEffect, useState } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import {
  Layout, Menu, Select, Dropdown, Space, Typography, theme,
  Modal, Form, Input, Button, Table, Popconfirm, InputNumber, message, Tag,
} from 'antd'
import {
  DatabaseOutlined,
  SettingOutlined,
  ClusterOutlined,
  UserOutlined,
  AuditOutlined,
  LogoutOutlined,
  KeyOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ApiOutlined,
  CloudServerOutlined,
  TeamOutlined,
} from '@ant-design/icons'
import { useAuthStore, canWrite } from '@/stores/auth'
import { useEnvironmentStore } from '@/stores/environment'
import { authApi } from '@/api/auth'
import { environmentApi } from '@/api/environment'
import type { Environment, EnvironmentCreateRequest } from '@/types'
import { menuItemConfigs, getVisibleMenuKeys } from '@/config/menu'

const { Sider, Header, Content } = Layout
const { Text } = Typography

const iconMap: Record<string, React.ReactNode> = {
  '/cluster': <ClusterOutlined />,
  '/kv': <DatabaseOutlined />,
  '/config': <SettingOutlined />,
  '/gateway': <ApiOutlined />,
  '/grpc': <CloudServerOutlined />,
  '/users': <UserOutlined />,
  '/roles': <TeamOutlined />,
  '/audit': <AuditOutlined />,
}

export default function MainLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { user, fetchProfile, logout } = useAuthStore()
  const { environments, current, fetch: fetchEnvs, setCurrent } = useEnvironmentStore()
  const { token: { colorBgContainer } } = theme.useToken()

  const [pwdOpen, setPwdOpen] = useState(false)
  const [pwdForm] = Form.useForm()
  const [pwdLoading, setPwdLoading] = useState(false)

  const [envOpen, setEnvOpen] = useState(false)
  const [envModalOpen, setEnvModalOpen] = useState(false)
  const [editingEnv, setEditingEnv] = useState<Environment | null>(null)
  const [envForm] = Form.useForm()

  useEffect(() => {
    fetchProfile()
    fetchEnvs()
  }, [fetchProfile, fetchEnvs])

  // 根据权限过滤菜单
  const visibleKeys = getVisibleMenuKeys(user)
  const visibleMenuItems = menuItemConfigs
    .filter((item) => visibleKeys.includes(item.key))
    .map(({ key, label }) => ({ key, icon: iconMap[key], label }))

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const handleChangePassword = async () => {
    const values = await pwdForm.validateFields()
    setPwdLoading(true)
    try {
      await authApi.changePassword({
        old_password: values.old_password as string,
        new_password: values.new_password as string,
      })
      message.success('密码修改成功')
      setPwdOpen(false)
      pwdForm.resetFields()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '修改失败')
    } finally {
      setPwdLoading(false)
    }
  }

  const openEnvCreate = () => {
    setEditingEnv(null)
    envForm.resetFields()
    setEnvModalOpen(true)
  }

  const openEnvEdit = (env: Environment) => {
    setEditingEnv(env)
    envForm.setFieldsValue({
      name: env.name,
      key_prefix: env.key_prefix,
      config_prefix: env.config_prefix,
      gateway_prefix: env.gateway_prefix,
      grpc_prefix: env.grpc_prefix,
      description: env.description,
      sort_order: env.sort_order,
    })
    setEnvModalOpen(true)
  }

  const handleEnvSave = async () => {
    const values = await envForm.validateFields()
    try {
      if (editingEnv) {
        await environmentApi.update(editingEnv.id, values as EnvironmentCreateRequest)
        message.success('更新成功')
      } else {
        await environmentApi.create(values as EnvironmentCreateRequest)
        message.success('创建成功')
      }
      setEnvModalOpen(false)
      fetchEnvs()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '操作失败')
    }
  }

  const handleEnvDelete = async (id: string) => {
    try {
      await environmentApi.delete(id)
      message.success('删除成功')
      fetchEnvs()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '删除失败')
    }
  }

  const canManageEnv = canWrite(user, 'environments')

  const envColumns = [
    { title: '名称', dataIndex: 'name', key: 'name', width: 100 },
    { title: 'Key 前缀', dataIndex: 'key_prefix', key: 'key_prefix', width: 140 },
    { title: '配置前缀', dataIndex: 'config_prefix', key: 'config_prefix', width: 120 },
    { title: '网关前缀', dataIndex: 'gateway_prefix', key: 'gateway_prefix', width: 130 },
    { title: 'gRPC 前缀', dataIndex: 'grpc_prefix', key: 'grpc_prefix', width: 130 },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    { title: '排序', dataIndex: 'sort_order', key: 'sort_order', width: 80 },
    ...(canManageEnv ? [{
      title: '操作', key: 'actions', width: 120,
      render: (_: unknown, record: Environment) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEnvEdit(record)} />
          <Popconfirm title="确认删除？" onConfirm={() => handleEnvDelete(record.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    }] : []),
  ]

  const userLabel = user?.is_super ? '超级管理员' : (user?.role?.name ?? '无角色')

  return (
    <Layout style={{ height: '100vh' }}>
      <Sider theme="dark" width={200}>
        <div style={{ height: 48, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Text strong style={{ color: '#fff', fontSize: 16 }}>ETCD管理中心</Text>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={visibleMenuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header style={{ background: colorBgContainer, padding: '0 24px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Space>
            <KeyOutlined />
            <Select
              value={current?.id}
              onChange={(id) => {
                const env = environments.find((e) => e.id === id)
                if (env) setCurrent(env)
              }}
              options={environments.map((e) => ({ label: e.name, value: e.id }))}
              style={{ width: 160 }}
              placeholder="选择环境"
            />
            {canManageEnv && (
              <Button size="small" icon={<SettingOutlined />} onClick={() => setEnvOpen(true)}>
                管理环境
              </Button>
            )}
          </Space>
          <Dropdown
            menu={{
              items: [
                { key: 'password', icon: <SettingOutlined />, label: '修改密码' },
                { key: 'logout', icon: <LogoutOutlined />, label: '退出登录' },
              ],
              onClick: ({ key }) => {
                if (key === 'logout') handleLogout()
                if (key === 'password') setPwdOpen(true)
              },
            }}
          >
            <Space style={{ cursor: 'pointer' }}>
              <UserOutlined />
              <Text>{user?.username}</Text>
              <Tag color={user?.is_super ? 'red' : 'blue'}>{userLabel}</Tag>
            </Space>
          </Dropdown>
        </Header>
        <Content style={{ margin: 16, padding: 24, background: colorBgContainer, borderRadius: 8, overflow: 'auto' }}>
          <Outlet />
        </Content>
      </Layout>

      {/* 修改密码 */}
      <Modal
        title="修改密码"
        open={pwdOpen}
        onOk={handleChangePassword}
        onCancel={() => { setPwdOpen(false); pwdForm.resetFields() }}
        confirmLoading={pwdLoading}
        destroyOnHidden
      >
        <Form form={pwdForm} layout="vertical">
          <Form.Item name="old_password" label="当前密码" rules={[{ required: true, message: '请输入当前密码' }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item name="new_password" label="新密码" rules={[{ required: true, message: '请输入新密码' }, { min: 6, message: '至少 6 位' }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="confirm_password"
            label="确认新密码"
            dependencies={['new_password']}
            rules={[
              { required: true, message: '请确认新密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('new_password') === value) return Promise.resolve()
                  return Promise.reject(new Error('两次密码不一致'))
                },
              }),
            ]}
          >
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>

      {/* 环境管理 */}
      <Modal
        title="环境管理"
        open={envOpen}
        onCancel={() => setEnvOpen(false)}
        footer={null}
        width={1000}
      >
        <div style={{ marginBottom: 16 }}>
          {canManageEnv && (
            <Button type="primary" icon={<PlusOutlined />} onClick={openEnvCreate}>新建环境</Button>
          )}
        </div>
        <Table rowKey="id" columns={envColumns} dataSource={environments} pagination={false} size="small" />
      </Modal>

      {/* 新建/编辑环境 */}
      <Modal
        title={editingEnv ? '编辑环境' : '新建环境'}
        open={envModalOpen}
        onOk={handleEnvSave}
        onCancel={() => setEnvModalOpen(false)}
        destroyOnHidden
      >
        <Form form={envForm} layout="vertical">
          <Form.Item name="name" label="环境名称" rules={[{ required: true, message: '请输入环境名称' }]}>
            <Input placeholder="例如: production" />
          </Form.Item>
          <Form.Item name="key_prefix" label="Key 前缀" rules={[{ required: true, message: '请输入 Key 前缀' }]}>
            <Input placeholder="例如: /ai-adp/dev/" disabled={!!editingEnv} />
          </Form.Item>
          <Form.Item name="config_prefix" label="配置前缀" initialValue="config/">
            <Input placeholder="例如: config/" />
          </Form.Item>
          <Form.Item name="gateway_prefix" label="网关服务前缀" initialValue="gw-services/">
            <Input placeholder="例如: gw-services/" />
          </Form.Item>
          <Form.Item name="grpc_prefix" label="gRPC 服务前缀" initialValue="grpc-services/">
            <Input placeholder="例如: grpc-services/" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="sort_order" label="排序" initialValue={0}>
            <InputNumber min={0} />
          </Form.Item>
        </Form>
      </Modal>
    </Layout>
  )
}
