import { useEffect, useState } from 'react'
import {
  Table, Button, Space, Tag, Modal, Popconfirm,
  Collapse, Badge, Tooltip, Spin, message,
} from 'antd'
import {
  ReloadOutlined, EyeOutlined,
  StopOutlined, PlayCircleOutlined,
} from '@ant-design/icons'
import type { GrpcServiceGroup, GrpcInstance } from '@/types'
import { grpcApi } from '@/api/grpc'
import { useAuthStore, canWrite } from '@/stores/auth'
import { useEnvironmentStore } from '@/stores/environment'
import MonacoEditor from '@/components/MonacoEditor'
import { CopyableCode, EmptyState, MetricCard, PageHeader, SectionCard, StatusBadge } from '@/components/ui'
import { formatUnixTime } from '@/utils'
import { buildServiceSummary } from '@/pages/services/presentation'

export default function GrpcPage() {
  const currentEnv = useEnvironmentStore((s) => s.current)
  const user = useAuthStore((s) => s.user)
  const isAdmin = canWrite(user, 'grpc')

  const [groups, setGroups] = useState<GrpcServiceGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [previewJson, setPreviewJson] = useState<string | null>(null)

  const fetchData = async () => {
    if (!currentEnv?.name) return
    setLoading(true)
    try {
      const data = await grpcApi.list(currentEnv.name)
      setGroups(data ?? [])
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (currentEnv?.name) fetchData()
  }, [currentEnv])

  const handleUpdateStatus = async (instance: GrpcInstance, status: 'up' | 'down') => {
    if (!currentEnv?.name) return
    try {
      await grpcApi.updateStatus(currentEnv.name, instance.key, status)
      message.success(status === 'up' ? '实例已上线' : '实例已下线')
      fetchData()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '操作失败')
    }
  }

  const summary = buildServiceSummary(groups)

  const instanceColumns = [
    {
      title: '实例ID', dataIndex: 'instance_id', key: 'instance_id',
      render: (id: string) => <CopyableCode value={id} />,
    },
    {
      title: '地址', dataIndex: 'address', key: 'address',
      render: (addr: string) => <CopyableCode value={addr} />,
    },
    { title: '版本', dataIndex: 'version', key: 'version', width: 100 },
    { title: '权重', dataIndex: 'weight', key: 'weight', width: 80 },
    {
      title: '标签', dataIndex: 'tags', key: 'tags',
      render: (tags: string[]) => (
        <Space size={[0, 4]} wrap>
          {(tags || []).map((tag) => (
            <Tag key={tag} color="blue">{tag}</Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 100,
      render: (s: string) => s === 'up'
        ? <StatusBadge tone="success">正常</StatusBadge>
        : <StatusBadge tone="danger">已下线</StatusBadge>,
    },
    {
      title: '注册时间', dataIndex: 'register_time', key: 'register_time', width: 170,
      render: formatUnixTime,
    },
    {
      title: '操作', key: 'actions', width: 160,
      render: (_: unknown, record: GrpcInstance) => (
        <Space>
          <Tooltip title="查看详情">
            <Button
              size="small"
              icon={<EyeOutlined />}
              onClick={() => setPreviewJson(JSON.stringify(record, null, 2))}
            />
          </Tooltip>
          {isAdmin && record.status === 'up' && (
            <Popconfirm title="确认下线该实例？" onConfirm={() => handleUpdateStatus(record, 'down')}>
              <Tooltip title="下线">
                <Button size="small" danger icon={<StopOutlined />} />
              </Tooltip>
            </Popconfirm>
          )}
          {isAdmin && record.status !== 'up' && (
            <Popconfirm title="确认上线该实例？" onConfirm={() => handleUpdateStatus(record, 'up')}>
              <Tooltip title="上线">
                <Button size="small" type="primary" icon={<PlayCircleOutlined />} />
              </Tooltip>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  const collapseItems = groups.map((group) => ({
    key: group.service_name,
    label: (
      <Space>
        <span style={{ fontWeight: 500 }}>{group.service_name}</span>
        <Badge className="service-count" count={group.instance_count} />
        <StatusBadge tone="success">{group.healthy_count} 正常</StatusBadge>
        {group.unhealthy_count > 0 && <StatusBadge tone="danger">{group.unhealthy_count} 下线</StatusBadge>}
      </Space>
    ),
    children: (
      <Table
        className="data-table"
        rowKey="instance_id"
        columns={instanceColumns}
        dataSource={group.instances}
        pagination={false}
        size="small"
      />
    ),
  }))

  return (
    <>
      <PageHeader
        eyebrow="gRPC Services"
        title="gRPC 服务"
        description="查看当前环境的 gRPC 服务注册与实例健康状态"
        extra={(
          <Button type="primary" icon={<ReloadOutlined />} onClick={() => fetchData()} loading={loading}>
            刷新数据
          </Button>
        )}
      />

      <div className="metric-grid metric-grid--three">
        <MetricCard label="服务数" value={summary.services} hint="已注册服务" />
        <MetricCard label="实例总数" value={summary.instances} hint={`${summary.healthy} 个健康实例`} />
        <MetricCard label="健康率" value={summary.healthDisplay} hint="实例整体健康度" tone={summary.tone} />
      </div>

      <SectionCard title="服务实例" description={`共 ${summary.services} 个服务，${summary.instances} 个实例`}>
        {loading ? (
          <Spin style={{ display: 'block', margin: '48px auto' }} />
        ) : groups.length > 0 ? (
          <Collapse
            className="service-groups"
            items={collapseItems}
            defaultActiveKey={groups.map((g) => g.service_name)}
          />
        ) : (
          <EmptyState title="当前环境暂无注册 gRPC 服务" description="gRPC 服务注册后将在这里展示" />
        )}
      </SectionCard>

      <Modal
        title="实例详情"
        open={previewJson !== null}
        onCancel={() => setPreviewJson(null)}
        footer={null}
        width={700}
      >
        {previewJson !== null && (
          <MonacoEditor value={previewJson} language="json" readOnly height={400} />
        )}
      </Modal>
    </>
  )
}
