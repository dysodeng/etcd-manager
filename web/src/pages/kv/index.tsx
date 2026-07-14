import { useEffect, useState } from 'react'
import { Table, Button, Input, Space, Modal, Form, message, Popconfirm, Segmented } from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined, UnorderedListOutlined, ApartmentOutlined } from '@ant-design/icons'
import type { KVItem } from '@/types'
import { kvApi } from '@/api/kv'
import { useAuthStore, canWrite } from '@/stores/auth'
import MonacoEditor from '@/components/MonacoEditor'
import { EmptyState, ErrorState, PageHeader, PageToolbar, SectionCard } from '@/components/ui'
import KVTreeView from './KVTreeView'
import { buildKVTree } from './buildKVTree'
import { useSubmissionLock } from '@/hooks/useSubmissionLock'

export default function KVPage() {
  const [items, setItems] = useState<KVItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [prefix, setPrefix] = useState('/')
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<KVItem | null>(null)
  const [form] = Form.useForm()
  const [editorValue, setEditorValue] = useState('')
  const user = useAuthStore((s) => s.user)
  const isAdmin = canWrite(user, 'kv')
  const [viewMode, setViewMode] = useState<'list' | 'tree'>('list')
  const [saving, runSaveLocked] = useSubmissionLock()
  const [deletingKey, setDeletingKey] = useState<string | null>(null)
  const hasData = items.length > 0

  const fetchData = async (prefixOverride?: string) => {
    setLoading(true)
    setError(null)
    try {
      const data = await kvApi.list(prefixOverride ?? prefix)
      setItems(data ?? [])
    } catch (caught: unknown) {
      const text = caught instanceof Error ? caught.message : '加载失败'
      setError(text)
      if (hasData) message.error(text)
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

  const handleSave = () => runSaveLocked(async () => {
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
  })

  const handleDelete = async (key: string) => {
    setDeletingKey(key)
    try {
      await kvApi.delete(key)
      message.success('删除成功')
      fetchData()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '删除失败')
    } finally {
      setDeletingKey(null)
    }
  }

  const columns = [
    { title: 'Key', dataIndex: 'key', key: 'key', ellipsis: true },
    {
      title: 'Value', dataIndex: 'value', key: 'value', ellipsis: true,
      render: (v: string) => <span className="resource-value-preview">{v.length > 80 ? v.slice(0, 80) + '...' : v}</span>,
    },
    { title: 'Version', dataIndex: 'version', key: 'version', width: 100 },
    {
      title: '操作', key: 'actions', width: 160,
      render: (_: unknown, record: KVItem) => (
        <Space>
          <Button size="small" onClick={() => openEdit(record)} disabled={!isAdmin}>编辑</Button>
          <Popconfirm
            title="确认删除此键值？"
            description={`将永久删除 ${record.key}`}
            onConfirm={() => handleDelete(record.key)}
            disabled={!isAdmin}
            okButtonProps={{ loading: deletingKey === record.key }}
          >
            <Button size="small" danger disabled={!isAdmin}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  if (error && !hasData) return <ErrorState description={error} onRetry={fetchData} />

  return (
    <>
      <PageHeader
        eyebrow="Key Value Store"
        title="KV 管理"
        description="浏览、检索和维护当前集群中的键值数据"
        extra={isAdmin ? <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新建键值</Button> : undefined}
      />
      <PageToolbar
        trailing={(
          <Segmented
            value={viewMode}
            onChange={(value) => setViewMode(value as 'list' | 'tree')}
            options={[
              { value: 'list', label: '列表', icon: <UnorderedListOutlined /> },
              { value: 'tree', label: '树形', icon: <ApartmentOutlined /> },
            ]}
          />
        )}
      >
        <Input
          className="toolbar-search"
          prefix={<SearchOutlined />}
          placeholder="Key 前缀"
          value={prefix}
          onChange={(event) => setPrefix(event.target.value)}
          onPressEnter={() => fetchData()}
        />
        <Button icon={<ReloadOutlined />} onClick={() => fetchData()} loading={loading}>刷新</Button>
      </PageToolbar>

      <SectionCard className="resource-card">
        {viewMode === 'list' ? (
          <Table
            className="data-table"
            rowKey="key"
            columns={columns}
            dataSource={items}
            loading={loading}
            pagination={false}
            size="middle"
            locale={{
              emptyText: (
                <EmptyState
                  title="暂无 KV 数据"
                  description="当前前缀下没有键值"
                  action={isAdmin ? <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新建键值</Button> : undefined}
                />
              ),
            }}
          />
        ) : (
          <KVTreeView
            treeData={buildKVTree(items)}
            isAdmin={isAdmin}
            deletingKey={deletingKey}
            onCreate={openCreate}
            onEdit={openEdit}
            onDelete={handleDelete}
          />
        )}
      </SectionCard>

      <Modal
        title={editing ? '编辑 KV' : '新建 KV'}
        open={modalOpen}
        onOk={handleSave}
        onCancel={() => setModalOpen(false)}
        width={700}
        destroyOnHidden
        className="app-modal"
        okText="保存"
        cancelText="取消"
        confirmLoading={saving}
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
