import { useEffect, useState } from 'react'
import { Table, Button, Input, Space, Modal, Form, message, Popconfirm } from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { KVItem } from '@/types'
import { kvApi } from '@/api/kv'
import { useAuthStore } from '@/stores/auth'
import MonacoEditor from '@/components/MonacoEditor'

export default function KVPage() {
  const [items, setItems] = useState<KVItem[]>([])
  const [loading, setLoading] = useState(false)
  const [prefix, setPrefix] = useState('/')
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<KVItem | null>(null)
  const [form] = Form.useForm()
  const [editorValue, setEditorValue] = useState('')
  const isAdmin = useAuthStore((s) => s.user?.role === 'admin')

  const fetchData = async (p?: string) => {
    setLoading(true)
    try {
      const data = await kvApi.list(p ?? prefix)
      setItems(data ?? [])
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchData('/') }, [])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setEditorValue('')
    setModalOpen(true)
  }

  const openEdit = (item: KVItem) => {
    setEditing(item)
    form.setFieldsValue({ key: item.key })
    setEditorValue(item.value)
    setModalOpen(true)
  }

  const handleSave = async () => {
    const values = await form.validateFields()
    try {
      if (editing) {
        await kvApi.update(values.key as string, editorValue)
        message.success('更新成功')
      } else {
        await kvApi.create(values.key as string, editorValue)
        message.success('创建成功')
      }
      setModalOpen(false)
      fetchData()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '操作失败')
    }
  }

  const handleDelete = async (key: string) => {
    try {
      await kvApi.delete(key)
      message.success('删除成功')
      fetchData()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '删除失败')
    }
  }

  const columns = [
    { title: 'Key', dataIndex: 'key', key: 'key', ellipsis: true },
    {
      title: 'Value', dataIndex: 'value', key: 'value', ellipsis: true,
      render: (v: string) => <span style={{ fontFamily: 'monospace' }}>{v.length > 80 ? v.slice(0, 80) + '...' : v}</span>,
    },
    { title: 'Version', dataIndex: 'version', key: 'version', width: 100 },
    {
      title: '操作', key: 'actions', width: 160,
      render: (_: unknown, record: KVItem) => (
        <Space>
          <Button size="small" onClick={() => openEdit(record)} disabled={!isAdmin}>编辑</Button>
          <Popconfirm title="确认删除？" onConfirm={() => handleDelete(record.key)} disabled={!isAdmin}>
            <Button size="small" danger disabled={!isAdmin}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <>
      <Space style={{ marginBottom: 16 }}>
        <Input
          prefix={<SearchOutlined />}
          placeholder="Key 前缀"
          value={prefix}
          onChange={(e) => setPrefix(e.target.value)}
          onPressEnter={() => fetchData()}
          style={{ width: 300 }}
        />
        <Button icon={<ReloadOutlined />} onClick={() => fetchData()}>刷新</Button>
        {isAdmin && <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新建</Button>}
      </Space>
      <Table
        rowKey="key"
        columns={columns}
        dataSource={items}
        loading={loading}
        pagination={false}
        size="middle"
      />
      <Modal
        title={editing ? '编辑 KV' : '新建 KV'}
        open={modalOpen}
        onOk={handleSave}
        onCancel={() => setModalOpen(false)}
        width={700}
        destroyOnHidden
      >
        <Form form={form} layout="vertical">
          <Form.Item name="key" label="Key" rules={[{ required: true, message: '请输入 Key' }]}>
            <Input disabled={!!editing} placeholder="例如: /app/config/key" />
          </Form.Item>
          <Form.Item label="Value">
            <MonacoEditor value={editorValue} onChange={setEditorValue} height={400} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}
