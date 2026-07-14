import { useEffect, useState } from 'react'
import { Alert, Table, Button, Modal, Form, Input, Select, message, Popconfirm, Pagination, Result } from 'antd'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import type { User, Role } from '@/types'
import { userApi } from '@/api/user'
import { roleApi } from '@/api/role'
import { useAuthStore, canRead, canWrite, isSuper } from '@/stores/auth'
import { EmptyState, ErrorState, PageHeader, PageToolbar, SectionCard, StatusBadge } from '@/components/ui'
import { formatTime } from '@/utils'

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([])
  const [roles, setRoles] = useState<Role[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [modalOpen, setModalOpen] = useState(false)
  const [form] = Form.useForm()
  const { user: currentUser } = useAuthStore()
  const isSuperAdmin = isSuper(currentUser)
  const canAccessUsers = canRead(currentUser, 'users')
  const isAdmin = canWrite(currentUser, 'users')

  // 转移超管
  const [transferOpen, setTransferOpen] = useState(false)
  const [transferForm] = Form.useForm()
  const [creating, setCreating] = useState(false)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [transferring, setTransferring] = useState(false)
  const transferTargetId = Form.useWatch('target_user_id', transferForm) as string | undefined

  const fetchRoles = async () => {
    try {
      const data = await roleApi.list(1, 100)
      setRoles(data.list)
    } catch { /* ignore */ }
  }

  const fetchData = async (p?: number) => {
    setLoading(true)
    setError(null)
    try {
      const data = await userApi.list(p ?? page)
      setUsers(data.list)
      setTotal(data.total)
    } catch (caught: unknown) {
      const text = caught instanceof Error ? caught.message : '加载失败'
      setError(text)
      if (users.length > 0) message.error(text)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (!canAccessUsers) return
    fetchData(1)
    if (isSuperAdmin) fetchRoles()
  }, [canAccessUsers, isSuperAdmin])

  const handleCreate = async () => {
    const values = await form.validateFields()
    setCreating(true)
    try {
      await userApi.create({
        username: values.username as string,
        password: values.password as string,
        role_id: values.role_id as string,
      })
      message.success('创建成功')
      setModalOpen(false)
      form.resetFields()
      fetchData()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '创建失败')
    } finally {
      setCreating(false)
    }
  }

  const handleUpdateRole = async (id: string, roleId: string) => {
    try {
      await userApi.update(id, { role_id: roleId })
      message.success('更新成功')
      fetchData()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '更新失败')
    }
  }

  const handleDelete = async (id: string) => {
    setDeletingId(id)
    try {
      await userApi.delete(id)
      message.success('删除成功')
      fetchData()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '删除失败')
    } finally {
      setDeletingId(null)
    }
  }

  const handleTransferSuper = async () => {
    const values = await transferForm.validateFields()
    setTransferring(true)
    try {
      await userApi.transferSuper(values.target_user_id as string, values.role_id as string)
      message.success('超管转移成功，请重新登录')
      setTransferOpen(false)
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '转移失败')
    } finally {
      setTransferring(false)
    }
  }

  const columns = [
    { title: '用户名', dataIndex: 'username', key: 'username' },
    {
      title: '角色', key: 'role', width: 200,
      render: (_: unknown, record: User) => {
        if (record.is_super) {
          return <StatusBadge tone="danger">超级管理员</StatusBadge>
        }
        if (!isSuperAdmin) {
          return record.role_name
            ? <StatusBadge tone="info">{record.role_name}</StatusBadge>
            : <StatusBadge tone="neutral">无角色</StatusBadge>
        }
        return (
          <Select
            value={record.role_id ?? undefined}
            onChange={(v: string) => handleUpdateRole(record.id, v)}
            options={roles.map(r => ({ label: <StatusBadge tone="info">{r.name}</StatusBadge>, value: r.id }))}
            style={{ width: 160 }}
            variant="borderless"
            placeholder="选择角色"
          />
        )
      },
    },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', render: formatTime },
    {
      title: '操作', key: 'actions', width: 100,
      render: (_: unknown, record: User) =>
        record.is_super || !isAdmin ? null : (
          <Popconfirm
            title={`确认删除用户「${record.username}」？`}
            description="删除后该用户将无法登录控制台"
            onConfirm={() => handleDelete(record.id)}
            okButtonProps={{ danger: true, loading: deletingId === record.id }}
          >
            <Button size="small" danger>删除</Button>
          </Popconfirm>
        ),
    },
  ]

  const nonSuperUsers = users.filter(u => !u.is_super && u.id !== currentUser?.user_id)

  if (!canAccessUsers) {
    return <Result status="403" title="无权访问" subTitle="当前角色没有用户管理权限" />
  }

  if (error && users.length === 0) return <ErrorState description={error} onRetry={() => fetchData(1)} />

  const transferTarget = nonSuperUsers.find((candidate) => candidate.id === transferTargetId)

  return (
    <>
      <PageHeader
        eyebrow="User Management"
        title="用户管理"
        description="管理控制台用户、角色归属与超级管理员身份"
        extra={isSuperAdmin ? (
          <Button type="primary" icon={<PlusOutlined />} onClick={() => { form.resetFields(); setModalOpen(true) }}>
            新建用户
          </Button>
        ) : undefined}
      />

      <PageToolbar
        trailing={isSuperAdmin ? (
          <Button onClick={() => { transferForm.resetFields(); setTransferOpen(true) }}>转移超管</Button>
        ) : undefined}
      >
        <Button icon={<ReloadOutlined />} onClick={() => fetchData()} loading={loading}>刷新</Button>
      </PageToolbar>

      <SectionCard title="用户列表" description={`共 ${total} 个用户`}>
        <Table
          className="data-table"
          rowKey="id"
          columns={columns}
          dataSource={users}
          loading={loading}
          pagination={false}
          size="middle"
          locale={{
            emptyText: (
              <EmptyState
                title="暂无用户"
                description="尚未创建可管理的控制台用户"
                action={isSuperAdmin ? <Button type="primary" icon={<PlusOutlined />} onClick={() => { form.resetFields(); setModalOpen(true) }}>新建用户</Button> : undefined}
              />
            ),
          }}
        />
      </SectionCard>
      <div className="page-pagination">
        <Pagination current={page} total={total} pageSize={20} showSizeChanger={false} onChange={(p) => { setPage(p); fetchData(p) }} />
      </div>

      <Modal
        title="新建用户"
        open={modalOpen}
        onOk={handleCreate}
        onCancel={() => setModalOpen(false)}
        destroyOnHidden
        className="app-modal"
        okText="创建"
        cancelText="取消"
        confirmLoading={creating}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }, { min: 6, message: '至少 6 位' }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item name="role_id" label="角色" rules={[{ required: true, message: '请选择角色' }]}>
            <Select
              placeholder="选择角色"
              options={roles.map(r => ({ label: r.name, value: r.id }))}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 转移超管 */}
      <Modal
        title={transferTarget ? `转移超级管理员给「${transferTarget.username}」` : '转移超级管理员'}
        open={transferOpen}
        onOk={handleTransferSuper}
        onCancel={() => setTransferOpen(false)}
        destroyOnHidden
        className="app-modal app-modal--danger"
        okText="确认转移"
        cancelText="取消"
        confirmLoading={transferring}
        okButtonProps={{ danger: true }}
      >
        {transferTarget && (
          <Alert
            className="app-modal-alert"
            type="warning"
            showIcon
            message={`超级管理员身份将转移给「${transferTarget.username}」`}
            description="操作成功后当前账号需要重新登录，并将按所选角色降级。"
          />
        )}
        <Form form={transferForm} layout="vertical">
          <Form.Item name="target_user_id" label="目标用户" rules={[{ required: true, message: '请选择目标用户' }]}>
            <Select
              placeholder="选择接收超管权限的用户"
              options={nonSuperUsers.map(u => ({ label: u.username, value: u.id }))}
            />
          </Form.Item>
          <Form.Item name="role_id" label="转移后你的角色" rules={[{ required: true, message: '请选择你降级后的角色' }]}>
            <Select
              placeholder="选择降级后的角色"
              options={roles.map(r => ({ label: r.name, value: r.id }))}
            />
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}
