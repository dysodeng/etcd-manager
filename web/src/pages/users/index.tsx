import { useEffect, useState } from 'react'
import { Table, Button, Space, Modal, Form, Input, Select, message, Popconfirm, Tag, Pagination } from 'antd'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import type { User, Role } from '@/types'
import { userApi } from '@/api/user'
import { roleApi } from '@/api/role'
import { useAuthStore, isSuper } from '@/stores/auth'
import { formatTime } from '@/utils'

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([])
  const [roles, setRoles] = useState<Role[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [form] = Form.useForm()
  const { user: currentUser } = useAuthStore()
  const isSuperAdmin = isSuper(currentUser)

  // 转移超管
  const [transferOpen, setTransferOpen] = useState(false)
  const [transferForm] = Form.useForm()

  const fetchRoles = async () => {
    try {
      const data = await roleApi.list(1, 100)
      setRoles(data.list)
    } catch { /* ignore */ }
  }

  const fetchData = async (p?: number) => {
    setLoading(true)
    try {
      const data = await userApi.list(p ?? page)
      setUsers(data.list)
      setTotal(data.total)
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData(1)
    fetchRoles()
  }, [])

  const handleCreate = async () => {
    const values = await form.validateFields()
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
    try {
      await userApi.delete(id)
      message.success('删除成功')
      fetchData()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '删除失败')
    }
  }

  const handleTransferSuper = async () => {
    const values = await transferForm.validateFields()
    try {
      await userApi.transferSuper(values.target_user_id as string, values.role_id as string)
      message.success('超管转移成功，请重新登录')
      setTransferOpen(false)
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '转移失败')
    }
  }

  const columns = [
    { title: '用户名', dataIndex: 'username', key: 'username' },
    {
      title: '角色', key: 'role', width: 200,
      render: (_: unknown, record: User) => {
        if (record.is_super) {
          return <Tag color="red">超级管理员</Tag>
        }
        if (!isSuperAdmin) {
          return <Tag color="blue">{record.role_name || '无角色'}</Tag>
        }
        return (
          <Select
            value={record.role_id ?? undefined}
            onChange={(v: string) => handleUpdateRole(record.id, v)}
            options={roles.map(r => ({ label: r.name, value: r.id }))}
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
        record.is_super ? null : (
          <Popconfirm title="确认删除？" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger>删除</Button>
          </Popconfirm>
        ),
    },
  ]

  const nonSuperUsers = users.filter(u => !u.is_super && u.id !== currentUser?.user_id)

  return (
    <>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ReloadOutlined />} onClick={() => fetchData()}>刷新</Button>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => { form.resetFields(); setModalOpen(true) }}>新建用户</Button>
        {isSuperAdmin && (
          <Button onClick={() => { transferForm.resetFields(); setTransferOpen(true) }}>转移超管</Button>
        )}
      </Space>

      <Table rowKey="id" columns={columns} dataSource={users} loading={loading} pagination={false} size="middle" />
      <div style={{ textAlign: 'right', marginTop: 16 }}>
        <Pagination current={page} total={total} pageSize={20} showSizeChanger={false} onChange={(p) => { setPage(p); fetchData(p) }} />
      </div>

      <Modal title="新建用户" open={modalOpen} onOk={handleCreate} onCancel={() => setModalOpen(false)} destroyOnHidden>
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
      <Modal title="转移超级管理员" open={transferOpen} onOk={handleTransferSuper} onCancel={() => setTransferOpen(false)} destroyOnHidden>
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
