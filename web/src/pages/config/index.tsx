import { useEffect, useState } from 'react'
import {
  Table, Button, Input, Space, Modal, Form, message, Popconfirm,
  Drawer, Tag, Upload, Tooltip, Select, Pagination,
} from 'antd'
import {
  PlusOutlined, ReloadOutlined, SearchOutlined,
  HistoryOutlined, ImportOutlined, ExportOutlined, RollbackOutlined, EyeOutlined,
} from '@ant-design/icons'
import type { ConfigItem, ConfigRevision } from '@/types'
import { configApi } from '@/api/config'
import { useAuthStore, canWrite } from '@/stores/auth'
import { useEnvironmentStore } from '@/stores/environment'
import MonacoEditor from '@/components/MonacoEditor'
import { CopyableCode, EmptyState, ErrorState, PageHeader, PageToolbar, SectionCard } from '@/components/ui'
import { formatTime, downloadBlob } from '@/utils'

export default function ConfigPage() {
  const currentEnv = useEnvironmentStore((s) => s.current)
  const user = useAuthStore((s) => s.user)
  const isAdmin = canWrite(user, 'config')

  const [items, setItems] = useState<ConfigItem[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [searchPrefix, setSearchPrefix] = useState('')

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<ConfigItem | null>(null)
  const [form] = Form.useForm()
  const [editorValue, setEditorValue] = useState('')

  const [historyOpen, setHistoryOpen] = useState(false)
  const [historyKey, setHistoryKey] = useState('')
  const [revisions, setRevisions] = useState<ConfigRevision[]>([])
  const [revTotal, setRevTotal] = useState(0)
  const [revPage, setRevPage] = useState(1)
  const [revLoading, setRevLoading] = useState(false)
  const [previewValue, setPreviewValue] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [deletingKey, setDeletingKey] = useState<string | null>(null)
  const [rollingBackId, setRollingBackId] = useState<string | null>(null)

  const envName = currentEnv?.name ?? ''

  const fetchConfigs = async () => {
    if (!envName) return
    setLoading(true)
    setError(null)
    try {
      const data = await configApi.list(envName, searchPrefix || undefined)
      setItems(data ?? [])
    } catch (caught: unknown) {
      const text = caught instanceof Error ? caught.message : '加载失败'
      setError(text)
      if (items.length > 0) message.error(text)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (envName) fetchConfigs()
  }, [envName])

  const fetchRevisions = async (key: string, page: number) => {
    if (!envName) return
    setRevLoading(true)
    try {
      const data = await configApi.revisions(envName, key, page)
      setRevisions(data.list)
      setRevTotal(data.total)
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '加载失败')
    } finally {
      setRevLoading(false)
    }
  }

  const openHistory = (key: string) => {
    setHistoryKey(key)
    setRevPage(1)
    setHistoryOpen(true)
    fetchRevisions(key, 1)
  }

  const handleRollback = async (revisionId: string) => {
    setRollingBackId(revisionId)
    try {
      await configApi.rollback({ env: envName, key: historyKey, revision_id: revisionId })
      message.success('回滚成功')
      fetchConfigs()
      fetchRevisions(historyKey, revPage)
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '回滚失败')
    } finally {
      setRollingBackId(null)
    }
  }

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setEditorValue('')
    setModalOpen(true)
  }

  const openEdit = (item: ConfigItem) => {
    setEditing(item)
    form.setFieldsValue({ key: item.key, comment: '' })
    setEditorValue(item.value)
    setModalOpen(true)
  }

  const handleSave = async () => {
    const values = await form.validateFields()
    setSaving(true)
    try {
      if (editing) {
        await configApi.update({ env: envName, key: values.key as string, value: editorValue, comment: values.comment as string })
        message.success('更新成功')
      } else {
        await configApi.create({ env: envName, key: values.key as string, value: editorValue, comment: values.comment as string })
        message.success('创建成功')
      }
      setModalOpen(false)
      fetchConfigs()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '操作失败')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (key: string) => {
    setDeletingKey(key)
    try {
      await configApi.delete(envName, key)
      message.success('删除成功')
      fetchConfigs()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '删除失败')
    } finally {
      setDeletingKey(null)
    }
  }

  const handleExport = async (format: 'json' | 'yaml') => {
    try {
      const resp = await configApi.export(envName, format)
      downloadBlob(resp.data as Blob, `config-${envName}.${format}`)
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '导出失败')
    }
  }

  const handleImport = async (file: File) => {
    try {
      const result = await configApi.import(envName, file)
      message.success(`导入完成：成功 ${result.success}，失败 ${result.failed.length}`)
      fetchConfigs()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '导入失败')
    }
    return false
  }

  if (!currentEnv) {
    return (
      <>
        <PageHeader
          eyebrow="Configuration Center"
          title="配置管理"
          description="按环境维护配置内容、变更历史与版本回滚"
        />
        <SectionCard className="resource-card">
          <EmptyState title="尚未选择环境" description="请先在顶栏选择环境，再查看和维护配置" />
        </SectionCard>
      </>
    )
  }

  if (error && items.length === 0) return <ErrorState description={error} onRetry={fetchConfigs} />

  const getFullKey = (key: string) => {
    const prefix = currentEnv?.config_prefix ?? ''
    const base = currentEnv?.key_prefix ?? ''
    const normalizedBase = base.endsWith('/') ? base : base + '/'
    return normalizedBase + prefix + key
  }

  const columns = [
    {
      title: 'Key', dataIndex: 'key', key: 'key', ellipsis: true,
      render: (v: string) => <CopyableCode value={v} copyValue={getFullKey(v)} />,
    },
    {
      title: 'Value', dataIndex: 'value', key: 'value', ellipsis: true,
      render: (v: string) => (
        <span className="resource-value-preview">
          <CopyableCode value={v.length > 60 ? v.slice(0, 60) + '...' : v} copyValue={v} />
        </span>
      ),
    },
    {
      title: '操作', key: 'actions', width: 240,
      render: (_: unknown, record: ConfigItem) => (
        <Space>
          <Tooltip title="版本历史">
            <Button size="small" icon={<HistoryOutlined />} onClick={() => openHistory(record.key)} />
          </Tooltip>
          <Button size="small" onClick={() => openEdit(record)} disabled={!isAdmin}>编辑</Button>
          <Popconfirm
            title="确认删除此配置？"
            description={`将从环境「${envName}」中永久删除 ${record.key}`}
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

  const actionTagColor: Record<string, string> = { create: 'green', update: 'blue', delete: 'red' }

  const revisionColumns = [
    {
      title: '操作', dataIndex: 'action', key: 'action', width: 80,
      render: (a: string) => <Tag color={actionTagColor[a]}>{a}</Tag>,
    },
    { title: '备注', dataIndex: 'comment', key: 'comment', ellipsis: true },
    { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170, render: formatTime },
    {
      title: '', key: 'actions', width: 140,
      render: (_: unknown, record: ConfigRevision) => (
        <Space>
          <Tooltip title="查看配置">
            <Button size="small" icon={<EyeOutlined />} onClick={() => setPreviewValue(record.value)} />
          </Tooltip>
          {isAdmin && (
            <Popconfirm
              title="确认回滚到此版本？"
              description={`配置 ${historyKey} 将回滚至 ${formatTime(record.created_at)} 的版本`}
              onConfirm={() => handleRollback(record.id)}
              okButtonProps={{ loading: rollingBackId === record.id }}
            >
              <Button size="small" icon={<RollbackOutlined />}>回滚</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  return (
    <>
      <PageHeader
        eyebrow="Configuration Center"
        title="配置管理"
        description={`浏览和维护「${envName}」环境的配置与版本历史`}
        extra={isAdmin ? <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新建配置</Button> : undefined}
      />

      <PageToolbar
        trailing={(
          <>
            <Select
              className="toolbar-export-select"
              placeholder="导出"
              value={undefined}
              onChange={(value: 'json' | 'yaml') => handleExport(value)}
              options={[
                { label: '导出 JSON', value: 'json' },
                { label: '导出 YAML', value: 'yaml' },
              ]}
              suffixIcon={<ExportOutlined />}
            />
            {isAdmin && (
              <Upload accept=".json,.yaml,.yml" showUploadList={false} beforeUpload={handleImport}>
                <Button icon={<ImportOutlined />}>导入</Button>
              </Upload>
            )}
          </>
        )}
      >
        <Input
          className="toolbar-search toolbar-search--compact"
          prefix={<SearchOutlined />}
          placeholder="Key 前缀过滤"
          value={searchPrefix}
          onChange={(event) => setSearchPrefix(event.target.value)}
          onPressEnter={() => fetchConfigs()}
        />
        <Button icon={<ReloadOutlined />} onClick={() => fetchConfigs()} loading={loading}>刷新</Button>
      </PageToolbar>

      <SectionCard className="resource-card">
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
                title="暂无配置"
                description={`「${envName}」环境还没有配置`}
                action={isAdmin ? <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新建配置</Button> : undefined}
              />
            ),
          }}
        />
      </SectionCard>

      <Modal
        title={editing ? '编辑配置' : '新建配置'}
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
            <Input disabled={!!editing} placeholder="例如: app/db_host" />
          </Form.Item>
          <Form.Item label="Value">
            <MonacoEditor value={editorValue} onChange={setEditorValue} height={360} />
          </Form.Item>
          <Form.Item name="comment" label="变更备注" rules={[{ required: true, message: '请输入变更备注' }]}>
            <Input.TextArea rows={2} placeholder="请输入变更备注" />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={`版本历史 — ${historyKey}`}
        open={historyOpen}
        onClose={() => setHistoryOpen(false)}
        width={700}
      >
        <Table
          rowKey="id"
          columns={revisionColumns}
          dataSource={revisions}
          loading={revLoading}
          pagination={false}
          size="small"
        />
        <div className="drawer-pagination">
          <Pagination
            current={revPage}
            total={revTotal}
            pageSize={20}
            showSizeChanger={false}
            onChange={(p) => { setRevPage(p); fetchRevisions(historyKey, p) }}
          />
        </div>
      </Drawer>

      <Modal
        title="配置内容"
        open={previewValue !== null}
        onCancel={() => setPreviewValue(null)}
        footer={null}
        width={700}
        destroyOnHidden
        className="app-modal"
      >
        {previewValue !== null && (
          <MonacoEditor value={previewValue} readOnly height={400} />
        )}
      </Modal>
    </>
  )
}
