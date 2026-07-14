import { useEffect, useState } from 'react'
import { Table, Input, DatePicker, Button, Pagination, Result, message } from 'antd'
import { ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import type { AuditLog, AuditLogFilter } from '@/types'
import { auditApi } from '@/api/audit'
import { useAuthStore, canRead } from '@/stores/auth'
import { CopyableCode, EmptyState, ErrorState, PageHeader, PageToolbar, SectionCard, StatusBadge } from '@/components/ui'
import { formatTime } from '@/utils'

const { RangePicker } = DatePicker

export default function AuditPage() {
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState<AuditLogFilter>({})
  const { user: currentUser } = useAuthStore()
  const canAccessAudit = canRead(currentUser, 'audit_logs')

  const fetchData = async (p?: number, f?: AuditLogFilter) => {
    setLoading(true)
    setError(null)
    try {
      const data = await auditApi.list({ ...(f ?? filter), page: p ?? page, page_size: 20 })
      setLogs(data.list)
      setTotal(data.total)
    } catch (caught: unknown) {
      const text = caught instanceof Error ? caught.message : '加载失败'
      setError(text)
      if (logs.length > 0) message.error(text)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (!canAccessAudit) return
    fetchData(1)
  }, [canAccessAudit])

  const handleSearch = () => {
    setPage(1)
    fetchData(1)
  }

  const handleReset = () => {
    const empty = {}
    setFilter(empty)
    setPage(1)
    fetchData(1, empty)
  }

  const actionTones: Record<string, 'success' | 'warning' | 'danger' | 'info' | 'neutral'> = {
    create: 'success',
    online: 'success',
    update: 'info',
    login: 'info',
    delete: 'danger',
    offline: 'danger',
    rollback: 'warning',
    restore: 'warning',
    transfer_super: 'warning',
  }

  const columns = [
    { title: '操作人', dataIndex: 'username', key: 'username', width: 100 },
    {
      title: '操作', dataIndex: 'action', key: 'action', width: 100,
      render: (action: string) => <StatusBadge tone={actionTones[action] ?? 'neutral'}>{action}</StatusBadge>,
    },
    { title: '资源类型', dataIndex: 'resource_type', key: 'resource_type', width: 120 },
    {
      title: '资源标识', dataIndex: 'resource_key', key: 'resource_key', ellipsis: true,
      render: (value: string) => value ? <CopyableCode value={value} /> : '-',
    },
    { title: '详情', dataIndex: 'detail', key: 'detail', ellipsis: true },
    {
      title: 'IP', dataIndex: 'ip', key: 'ip', width: 160,
      render: (value: string) => value ? <CopyableCode value={value} /> : '-',
    },
    { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170, render: formatTime },
  ]

  if (!canAccessAudit) {
    return <Result status="403" title="无权访问" subTitle="当前角色没有审计日志权限" />
  }

  if (error && logs.length === 0) return <ErrorState description={error} onRetry={() => fetchData(1)} />

  return (
    <>
      <PageHeader
        eyebrow="Audit Trail"
        title="审计日志"
        description="追踪控制台操作记录、资源变更与访问来源"
      />

      <PageToolbar>
        <Input
          className="audit-filter-input"
          prefix={<SearchOutlined />}
          placeholder="操作类型"
          value={filter.action ?? ''}
          onChange={(e) => setFilter((f) => ({ ...f, action: e.target.value || undefined }))}
        />
        <Input
          className="audit-filter-input"
          placeholder="资源类型"
          value={filter.resource_type ?? ''}
          onChange={(e) => setFilter((f) => ({ ...f, resource_type: e.target.value || undefined }))}
        />
        <RangePicker
          className="audit-filter-range"
          showTime
          value={filter.start_time && filter.end_time
            ? [dayjs(filter.start_time), dayjs(filter.end_time)]
            : null}
          onChange={(dates) => {
            if (dates?.[0] && dates?.[1]) {
              setFilter((f) => ({
                ...f,
                start_time: dates[0]!.toISOString(),
                end_time: dates[1]!.toISOString(),
              }))
            } else {
              setFilter((f) => ({ ...f, start_time: undefined, end_time: undefined }))
            }
          }}
        />
        <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>搜索</Button>
        <Button icon={<ReloadOutlined />} onClick={handleReset}>重置</Button>
      </PageToolbar>

      <SectionCard title="操作记录" description={`共 ${total} 条日志`}>
        <Table
          className="data-table"
          rowKey="id"
          columns={columns}
          dataSource={logs}
          loading={loading}
          pagination={false}
          size="middle"
          locale={{ emptyText: <EmptyState title="暂无审计日志" description="当前筛选条件下没有操作记录" /> }}
        />
      </SectionCard>
      <div className="page-pagination">
        <Pagination current={page} total={total} pageSize={20} showSizeChanger={false} onChange={(p) => { setPage(p); fetchData(p) }} />
      </div>
    </>
  )
}
