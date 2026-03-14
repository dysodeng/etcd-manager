import { useEffect, useState } from 'react'
import { Table, Space, Input, DatePicker, Button, Tag, Pagination, message } from 'antd'
import { ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { AuditLog, AuditLogFilter } from '@/types'
import { auditApi } from '@/api/audit'
import { formatTime } from '@/utils'

const { RangePicker } = DatePicker

export default function AuditPage() {
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [filter, setFilter] = useState<AuditLogFilter>({})

  const fetchData = async (p?: number, f?: AuditLogFilter) => {
    setLoading(true)
    try {
      const data = await auditApi.list({ ...(f ?? filter), page: p ?? page, page_size: 20 })
      setLogs(data.list)
      setTotal(data.total)
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchData(1) }, [])

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

  const actionColors: Record<string, string> = {
    create: 'green', update: 'blue', delete: 'red', login: 'purple', rollback: 'orange',
  }

  const columns = [
    { title: '操作人', dataIndex: 'username', key: 'username', width: 100 },
    {
      title: '操作', dataIndex: 'action', key: 'action', width: 100,
      render: (a: string) => <Tag color={actionColors[a] ?? 'default'}>{a}</Tag>,
    },
    { title: '资源类型', dataIndex: 'resource_type', key: 'resource_type', width: 120 },
    { title: '资源标识', dataIndex: 'resource_key', key: 'resource_key', ellipsis: true },
    { title: '详情', dataIndex: 'detail', key: 'detail', ellipsis: true },
    { title: 'IP', dataIndex: 'ip', key: 'ip', width: 140 },
    { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170, render: formatTime },
  ]

  return (
    <>
      <Space style={{ marginBottom: 16 }} wrap>
        <Input
          prefix={<SearchOutlined />}
          placeholder="操作类型"
          value={filter.action ?? ''}
          onChange={(e) => setFilter((f) => ({ ...f, action: e.target.value || undefined }))}
          style={{ width: 150 }}
        />
        <Input
          placeholder="资源类型"
          value={filter.resource_type ?? ''}
          onChange={(e) => setFilter((f) => ({ ...f, resource_type: e.target.value || undefined }))}
          style={{ width: 150 }}
        />
        <RangePicker
          showTime
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
      </Space>

      <Table rowKey="id" columns={columns} dataSource={logs} loading={loading} pagination={false} size="middle" />
      <div style={{ textAlign: 'right', marginTop: 16 }}>
        <Pagination current={page} total={total} pageSize={20} showSizeChanger={false} onChange={(p) => { setPage(p); fetchData(p) }} />
      </div>
    </>
  )
}
