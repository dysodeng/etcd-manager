import { useEffect, useState } from 'react'
import {
  Card, Table, Button, Space, Tag, Modal, Popconfirm,
  Statistic, Row, Col, Collapse, Badge, Tooltip, Empty, Spin, message,
} from 'antd'
import {
  ReloadOutlined, EyeOutlined,
  StopOutlined, CheckCircleOutlined, CloseCircleOutlined, PlayCircleOutlined,
} from '@ant-design/icons'
import type { GrpcServiceGroup, GrpcInstance } from '@/types'
import { grpcApi } from '@/api/grpc'
import { useAuthStore } from '@/stores/auth'
import { useEnvironmentStore } from '@/stores/environment'
import MonacoEditor from '@/components/MonacoEditor'
import { formatUnixTime } from '@/utils'

export default function GrpcPage() {
  const currentEnv = useEnvironmentStore((s) => s.current)
  const isAdmin = useAuthStore((s) => s.user?.role === 'admin')

  const [groups, setGroups] = useState<GrpcServiceGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [previewJson, setPreviewJson] = useState<string | null>(null)

  const getPrefix = () => {
    if (!currentEnv?.key_prefix) return ''
    const base = currentEnv.key_prefix.endsWith('/')
      ? currentEnv.key_prefix
      : currentEnv.key_prefix + '/'
    return base + (currentEnv.grpc_prefix || 'grpc-services/')
  }

  const fetchData = async () => {
    const prefix = getPrefix()
    if (!prefix) return
    setLoading(true)
    try {
      const data = await grpcApi.list(prefix)
      setGroups(data ?? [])
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (currentEnv?.key_prefix) fetchData()
  }, [currentEnv])

  const handleUpdateStatus = async (instance: GrpcInstance, status: 'up' | 'down') => {
    const key = getPrefix() + instance.service_name + '/' + instance.instance_id
    try {
      await grpcApi.updateStatus(key, status)
      message.success(status === 'up' ? '实例已上线' : '实例已下线')
      fetchData()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '操作失败')
    }
  }

  const totalInstances = groups.reduce((sum, g) => sum + g.instance_count, 0)
  const totalHealthy = groups.reduce((sum, g) => sum + g.healthy_count, 0)

  const instanceColumns = [
    {
      title: '实例ID', dataIndex: 'instance_id', key: 'instance_id',
      render: (id: string) => (
        <span style={{ fontFamily: 'monospace' }}>{id}</span>
      ),
    },
    {
      title: '地址', dataIndex: 'address', key: 'address',
      render: (addr: string) => (
        <span style={{ fontFamily: 'monospace' }}>{addr}</span>
      ),
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
      render: (s: string) => {
        if (s === 'up') return <Tag icon={<CheckCircleOutlined />} color="success">正常</Tag>
        return <Tag icon={<CloseCircleOutlined />} color="error">已下线</Tag>
      },
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
        <Badge count={group.instance_count} style={{ backgroundColor: '#1677ff' }} />
        <Tag color="success">{group.healthy_count} 正常</Tag>
        {group.unhealthy_count > 0 && <Tag color="error">{group.unhealthy_count} 下线</Tag>}
      </Space>
    ),
    children: (
      <Table
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
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ReloadOutlined />} onClick={() => fetchData()}>刷新</Button>
      </Space>

      {groups.length > 0 && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={8}>
            <Card><Statistic title="服务数" value={groups.length} /></Card>
          </Col>
          <Col span={8}>
            <Card><Statistic title="实例总数" value={totalInstances} /></Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title="健康率"
                value={totalInstances > 0 ? ((totalHealthy / totalInstances) * 100).toFixed(1) : 0}
                suffix="%"
                valueStyle={{ color: totalHealthy === totalInstances ? '#3f8600' : '#cf1322' }}
              />
            </Card>
          </Col>
        </Row>
      )}

      {loading ? (
        <Spin style={{ display: 'block', margin: '48px auto' }} />
      ) : groups.length > 0 ? (
        <Collapse items={collapseItems} defaultActiveKey={groups.map((g) => g.service_name)} />
      ) : (
        <Empty description="当前环境暂无注册 gRPC 服务" />
      )}

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
