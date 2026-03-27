import { useEffect, useState } from 'react'
import {
  Table, Button, Space, Modal, Form, Input, Checkbox, message,
  Popconfirm, Pagination, Tag, Select,
} from 'antd'
import { PlusOutlined, ReloadOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import type { Role, RolePermission, Environment } from '@/types'
import { roleApi } from '@/api/role'
import { useEnvironmentStore } from '@/stores/environment'
import { formatTime } from '@/utils'

const ALL_MODULES = [
  { key: 'kv', label: 'KV 管理' },
  { key: 'config', label: '配置中心' },
  { key: 'gateway', label: '网关服务' },
  { key: 'grpc', label: 'gRPC 服务' },
  { key: 'users', label: '用户管理' },
  { key: 'environments', label: '环境管理' },
  { key: 'audit_logs', label: '审计日志' },
  { key: 'cluster', label: '集群信息' },
]

export default function RolesPage() {
  const [roles, setRoles] = useState<Role[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editingRole, setEditingRole] = useState<Role | null>(null)
  const [form] = Form.useForm()
  const [permissions, setPermissions] = useState<Record<string, { can_read: boolean; can_write: boolean }>>({})
  const [selectedEnvIds, setSelectedEnvIds] = useState<string[]>([])
  const { environments, fetch: fetchEnvs } = useEnvironmentStore()

  const fetchData = async (p?: number) => {
    setLoading(true)
    try {
      const data = await roleApi.list(p ?? page)
      setRoles(data.list)
      setTotal(data.total)
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData(1)
    fetchEnvs()
  }, [])

  const initPermissions = (perms?: RolePermission[]) => {
    const map: Record<string, { can_read: boolean; can_write: boolean }> = {}
    ALL_MODULES.forEach(m => {
      const existing = perms?.find(p => p.module === m.key)
      map[m.key] = {
        can_read: existing?.can_read ?? false,
        can_write: existing?.can_write ?? false,
      }
    })
    setPermissions(map)
  }

  const openCreate = () => {
    setEditingRole(null)
    form.resetFields()
    initPermissions()
    setSelectedEnvIds([])
    setModalOpen(true)
  }

  const openEdit = async (role: Role) => {
    try {
      const detail = await roleApi.getById(role.id)
      setEditingRole(detail)
      form.setFieldsValue({ name: detail.name, description: detail.description })
      initPermissions(detail.permissions)
      setSelectedEnvIds(detail.environment_ids ?? [])
      setModalOpen(true)
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '获取角色详情失败')
    }
  }

  const handleSave = async () => {
    const values = await form.validateFields()
    const perms: RolePermission[] = ALL_MODULES.map(m => ({
      module: m.key,
      can_read: permissions[m.key]?.can_read ?? false,
      can_write: permissions[m.key]?.can_write ?? false,
    }))
    const payload = {
      name: values.name as string,
      description: (values.description as string) || '',
      permissions: perms,
      environment_ids: selectedEnvIds,
    }
    try {
      if (editingRole) {
        await roleApi.update(editingRole.id, payload)
        message.success('更新成功')
      } else {
        await roleApi.create(payload)
        message.success('创建成功')
      }
      setModalOpen(false)
      fetchData()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '操作失败')
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await roleApi.delete(id)
      message.success('删除成功')
      fetchData()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '删除失败')
    }
  }

  const togglePermission = (module: string, field: 'can_read' | 'can_write') => {
    setPermissions(prev => {
      const current = prev[module] ?? { can_read: false, can_write: false }
      const updated = { ...current }
      if (field === 'can_write') {
        updated.can_write = !updated.can_write
        if (updated.can_write) updated.can_read = true
      } else {
        updated.can_read = !updated.can_read
        if (!updated.can_read) updated.can_write = false
      }
      return { ...prev, [module]: updated }
    })
  }

  const columns = [
    { title: '角色名称', dataIndex: 'name', key: 'name' },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: '关联用户数', dataIndex: 'user_count', key: 'user_count', width: 110,
      render: (count: number) => <Tag>{count}</Tag>,
    },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', render: formatTime, width: 180 },
    {
      title: '操作', key: 'actions', width: 120,
      render: (_: unknown, record: Role) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)} />
          <Popconfirm
            title="确认删除？"
            description={record.user_count > 0 ? '该角色还有关联用户，需先解绑' : undefined}
            onConfirm={() => handleDelete(record.id)}
          >
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const permColumns = [
    { title: '模块', dataIndex: 'label', key: 'label', width: 120 },
    {
      title: '读权限', key: 'can_read', width: 80,
      render: (_: unknown, record: { key: string }) => (
        <Checkbox
          checked={permissions[record.key]?.can_read ?? false}
          onChange={() => togglePermission(record.key, 'can_read')}
        />
      ),
    },
    {
      title: '写权限', key: 'can_write', width: 80,
      render: (_: unknown, record: { key: string }) => (
        <Checkbox
          checked={permissions[record.key]?.can_write ?? false}
          onChange={() => togglePermission(record.key, 'can_write')}
        />
      ),
    },
  ]

  return (
    <>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ReloadOutlined />} onClick={() => fetchData()}>刷新</Button>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新建角色</Button>
      </Space>

      <Table rowKey="id" columns={columns} dataSource={roles} loading={loading} pagination={false} size="middle" />
      <div style={{ textAlign: 'right', marginTop: 16 }}>
        <Pagination current={page} total={total} pageSize={20} showSizeChanger={false} onChange={(p) => { setPage(p); fetchData(p) }} />
      </div>

      <Modal
        title={editingRole ? '编辑角色' : '新建角色'}
        open={modalOpen}
        onOk={handleSave}
        onCancel={() => setModalOpen(false)}
        width={600}
        destroyOnHidden
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="角色名称" rules={[{ required: true, message: '请输入角色名称' }]}>
            <Input placeholder="例如: 运维组" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>

        <div style={{ marginBottom: 16 }}>
          <div style={{ fontWeight: 500, marginBottom: 8 }}>授权环境</div>
          <Select
            mode="multiple"
            style={{ width: '100%' }}
            placeholder="选择授权的环境"
            value={selectedEnvIds}
            onChange={setSelectedEnvIds}
            options={environments.map((e: Environment) => ({ label: e.name, value: e.id }))}
          />
        </div>

        <div>
          <div style={{ fontWeight: 500, marginBottom: 8 }}>模块权限</div>
          <Table
            rowKey="key"
            columns={permColumns}
            dataSource={ALL_MODULES}
            pagination={false}
            size="small"
          />
        </div>
      </Modal>
    </>
  )
}
