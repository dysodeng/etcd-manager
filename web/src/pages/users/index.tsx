import { useEffect, useState } from 'react'
import { Table, Button, Space, Modal, Form, Input, Select, message, Popconfirm, Tag, Pagination } from 'antd'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import type { User } from '@/types'
import { userApi } from '@/api/user'
import { formatTime } from '@/utils'

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [form] = Form.useForm()

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

  useEffect(() => { fetchData(1) }, [])

  const handleCreate = async () => {
    const values = await form.validateFields()
    try {
      await userApi.create(values as { username: string; password: string; role: 'admin' | 'viewer' })
      message.success('创建成功')
      setModalOpen(false)
      form.resetFields()
      fetchData()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '创建失败')
    }
  }

  const handleUpdateRole = async (id: string, role: 'admin' | 'viewer') => {
    try {
      await userApi.update(id, { role })
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

  const columns = [
    { title: '用户名', dataIndex: 'username', key: 'username' },
    {
      title: '角色', dataIndex: 'role', key: 'role',
      render: (role: string, record: User) =>
        record.username === 'admin' ? (
          <Tag color="red">admin</Tag>
        ) : (
          <Select<'admin' | 'viewer'>
            value={role as 'admin' | 'viewer'}
            onChange={(v: 'admin' | 'viewer') => handleUpdateRole(record.id, v)}
            options={[
              { label: <Tag color="red">admin</Tag>, value: 'admin' },
              { label: <Tag color="blue">viewer</Tag>, value: 'viewer' },
            ]}
            style={{ width: 120 }}
            variant="borderless"
          />
        ),
    },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', render: formatTime },
    {
      title: '操作', key: 'actions', width: 100,
      render: (_: unknown, record: User) =>
        record.username === 'admin' ? null : (
          <Popconfirm title="确认删除？" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger>删除</Button>
          </Popconfirm>
        ),
    },
  ]

  return (
    <>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ReloadOutlined />} onClick={() => fetchData()}>刷新</Button>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => { form.resetFields(); setModalOpen(true) }}>新建用户</Button>
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
          <Form.Item name="role" label="角色" rules={[{ required: true, message: '请选择角色' }]} initialValue="viewer">
            <Select options={[{ label: 'admin', value: 'admin' }, { label: 'viewer', value: 'viewer' }]} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}
