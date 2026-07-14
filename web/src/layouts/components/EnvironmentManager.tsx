import { useState } from 'react'
import { CopyOutlined, DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons'
import { Button, Form, Input, InputNumber, message, Modal, Popconfirm, Space, Table } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import type { Environment, EnvironmentCreateRequest } from '@/types'
import { copyText } from '@/utils'

interface EnvironmentManagerProps {
  open: boolean
  environments: Environment[]
  canManage: boolean
  onClose: () => void
  onDelete: (id: string) => void | Promise<void>
  onSave: (
    values: EnvironmentCreateRequest,
    editing: Environment | null,
  ) => boolean | void | Promise<boolean | void>
}

function fullPrefix(record: Environment, suffix: string) {
  const base = record.key_prefix.endsWith('/') ? record.key_prefix : `${record.key_prefix}/`
  return base + suffix
}

function CopyButton({ text }: { text: string }) {
  return (
    <CopyOutlined
      className="copy-action"
      onClick={() => { copyText(text).then(() => message.success('已复制')) }}
    />
  )
}

export default function EnvironmentManager({
  open,
  environments,
  canManage,
  onClose,
  onDelete,
  onSave,
}: EnvironmentManagerProps) {
  const [form] = Form.useForm<EnvironmentCreateRequest>()
  const [editorOpen, setEditorOpen] = useState(false)
  const [editing, setEditing] = useState<Environment | null>(null)
  const [saving, setSaving] = useState(false)
  const [deletingId, setDeletingId] = useState<string | null>(null)

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setEditorOpen(true)
  }

  const openEdit = (environment: Environment) => {
    setEditing(environment)
    form.setFieldsValue({
      name: environment.name,
      key_prefix: environment.key_prefix,
      config_prefix: environment.config_prefix,
      gateway_prefix: environment.gateway_prefix,
      grpc_prefix: environment.grpc_prefix,
      description: environment.description,
      sort_order: environment.sort_order,
    })
    setEditorOpen(true)
  }

  const handleSave = async () => {
    const values = await form.validateFields()
    setSaving(true)
    try {
      const saved = await onSave(values, editing)
      if (saved !== false) setEditorOpen(false)
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (environment: Environment) => {
    setDeletingId(environment.id)
    try {
      await onDelete(environment.id)
    } finally {
      setDeletingId(null)
    }
  }

  const columns: ColumnsType<Environment> = [
    { title: '名称', dataIndex: 'name', key: 'name', width: 100 },
    {
      title: 'Key 前缀', dataIndex: 'key_prefix', key: 'key_prefix', width: 160,
      render: (value: string) => <>{value}<CopyButton text={value} /></>,
    },
    {
      title: '配置前缀', dataIndex: 'config_prefix', key: 'config_prefix', width: 140,
      render: (value: string, record) => <>{value}{value && <CopyButton text={fullPrefix(record, value)} />}</>,
    },
    {
      title: '网关前缀', dataIndex: 'gateway_prefix', key: 'gateway_prefix', width: 150,
      render: (value: string, record) => <>{value}{value && <CopyButton text={fullPrefix(record, value)} />}</>,
    },
    {
      title: 'gRPC 前缀', dataIndex: 'grpc_prefix', key: 'grpc_prefix', width: 150,
      render: (value: string, record) => <>{value}{value && <CopyButton text={fullPrefix(record, value)} />}</>,
    },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    { title: '排序', dataIndex: 'sort_order', key: 'sort_order', width: 60 },
    ...(canManage ? [{
      title: '操作',
      key: 'actions',
      width: 100,
      render: (_: unknown, record: Environment) => (
        <Space>
          <Button size="small" aria-label={`编辑 ${record.name}`} icon={<EditOutlined />} onClick={() => openEdit(record)} />
          <Popconfirm
            title={`确认删除环境「${record.name}」？`}
            description={`将删除环境「${record.name}」及其管理配置，此操作无法恢复`}
            onConfirm={() => handleDelete(record)}
            okButtonProps={{ danger: true, loading: deletingId === record.id }}
          >
            <Button size="small" danger aria-label={`删除 ${record.name}`} icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    }] : []),
  ]

  return (
    <>
      <Modal
        title="环境管理"
        open={open}
        onCancel={onClose}
        footer={null}
        width={1100}
        destroyOnHidden
        className="app-modal app-modal--wide"
      >
        <div className="app-modal-section">
          {canManage && (
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新建环境</Button>
          )}
        </div>
        <div className="app-modal-section">
          <Table rowKey="id" columns={columns} dataSource={environments} pagination={false} size="small" />
        </div>
      </Modal>

      <Modal
        title={editing ? '编辑环境' : '新建环境'}
        open={editorOpen}
        onOk={handleSave}
        onCancel={() => setEditorOpen(false)}
        afterClose={() => form.resetFields()}
        destroyOnHidden
        className="app-modal"
        okText="保存"
        cancelText="取消"
        confirmLoading={saving}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="环境名称" rules={[{ required: true, message: '请输入环境名称' }]}>
            <Input placeholder="例如: production" />
          </Form.Item>
          <Form.Item name="key_prefix" label="Key 前缀" rules={[{ required: true, message: '请输入 Key 前缀' }]}>
            <Input placeholder="例如: /ai-adp/dev/" disabled={Boolean(editing)} />
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
    </>
  )
}
