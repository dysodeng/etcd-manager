import { useEffect, useState, useCallback, useRef } from 'react'
import {
  Card, Descriptions, Table, Tag, Spin, Button, Space, Statistic, Row, Col,
  Result, message, Progress, Alert,
} from 'antd'
import {
  ReloadOutlined, CheckCircleOutlined, CloseCircleOutlined, DisconnectOutlined,
  WarningOutlined, CrownOutlined, BookOutlined,
} from '@ant-design/icons'
import type { ClusterStatus, ClusterMetrics, MemberStatus, AlarmInfo } from '@/types'
import { clusterApi } from '@/api/cluster'

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`
}

export default function ClusterPage() {
  const [status, setStatus] = useState<ClusterStatus | null>(null)
  const [metrics, setMetrics] = useState<ClusterMetrics | null>(null)
  const [memberStatuses, setMemberStatuses] = useState<MemberStatus[]>([])
  const [alarms, setAlarms] = useState<AlarmInfo[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const hasData = useRef(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [s, m, ms, al] = await Promise.all([
        clusterApi.status(),
        clusterApi.metrics(),
        clusterApi.memberStatuses(),
        clusterApi.alarms(),
      ])
      setStatus(s)
      setMetrics(m)
      setMemberStatuses(ms ?? [])
      setAlarms(al ?? [])
      hasData.current = true
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '无法连接 etcd 集群'
      setError(msg)
      if (hasData.current) message.error(msg)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchData() }, [fetchData])

  const healthData = metrics?.health
    ? Object.entries(metrics.health).map(([endpoint, healthy]) => ({ endpoint, healthy }))
    : []

  // DB 碎片率
  const fragPercent = metrics && metrics.db_size > 0
    ? Math.round((1 - metrics.db_size_in_use / metrics.db_size) * 100)
    : 0

  if (loading && !status) return <Spin style={{ display: 'block', margin: '48px auto' }} />

  if (error && !status) {
    return (
      <Result
        icon={<DisconnectOutlined />}
        title="无法连接 etcd 集群"
        subTitle={error}
        extra={<Button type="primary" icon={<ReloadOutlined />} onClick={fetchData} loading={loading}>重试</Button>}
      />
    )
  }

  return (
    <>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ReloadOutlined />} onClick={fetchData} loading={loading}>刷新</Button>
      </Space>

      {/* 报警 */}
      {alarms.length > 0 && (
        <Alert
          message="集群报警"
          description={
            <ul style={{ margin: 0, paddingLeft: 20 }}>
              {alarms.map((a, i) => (
                <li key={i}>
                  成员 {a.member_id}：
                  <Tag color="error" style={{ marginLeft: 4 }}>
                    {a.alarm_type === 'NOSPACE' ? '磁盘空间不足' : a.alarm_type === 'CORRUPT' ? '数据损坏' : a.alarm_type}
                  </Tag>
                </li>
              ))}
            </ul>
          }
          type="error"
          showIcon
          icon={<WarningOutlined />}
          style={{ marginBottom: 24 }}
        />
      )}

      {metrics && (
        <Row gutter={16} style={{ marginBottom: 24 }}>
          <Col span={4}><Card><Statistic title="集群 ID" value={metrics.cluster_id} /></Card></Col>
          <Col span={4}><Card><Statistic title="成员数量" value={metrics.member_count} /></Card></Col>
          <Col span={4}><Card><Statistic title="DB 大小" value={formatBytes(metrics.db_size)} /></Card></Col>
          <Col span={4}><Card><Statistic title="DB 实际使用" value={formatBytes(metrics.db_size_in_use)} /></Card></Col>
          <Col span={4}>
            <Card>
              <div style={{ marginBottom: 4, color: 'rgba(0,0,0,0.45)', fontSize: 14 }}>DB 碎片率</div>
              <Progress
                percent={fragPercent}
                status={fragPercent > 50 ? 'exception' : fragPercent > 30 ? 'active' : 'success'}
                size="small"
              />
            </Card>
          </Col>
          <Col span={4}><Card><Statistic title="etcd 版本" value={metrics.version} /></Card></Col>
        </Row>
      )}

      {status && (
        <Card title="集群成员" style={{ marginBottom: 24 }}>
          <Descriptions column={2} size="small" style={{ marginBottom: 16 }}>
            <Descriptions.Item label="集群 ID">{status.cluster_id}</Descriptions.Item>
            <Descriptions.Item label="Leader">{status.leader}</Descriptions.Item>
          </Descriptions>
          <Table
            rowKey="id"
            dataSource={status.members}
            pagination={false}
            size="small"
            columns={[
              { title: 'ID', dataIndex: 'id', key: 'id' },
              {
                title: '名称', dataIndex: 'name', key: 'name',
                render: (name: string) => (
                  <Space>
                    {name}
                    {name === status.leader && <Tag icon={<CrownOutlined />} color="blue">Leader</Tag>}
                  </Space>
                ),
              },
              { title: 'Peer URLs', dataIndex: 'peer_urls', key: 'peer_urls', render: (urls: string[]) => urls.join(', ') },
              { title: 'Client URLs', dataIndex: 'client_urls', key: 'client_urls', render: (urls: string[]) => urls.join(', ') },
              {
                title: '角色', key: 'role', width: 100,
                render: (_: unknown, record: { is_learner: boolean }) =>
                  record.is_learner ? <Tag icon={<BookOutlined />} color="orange">Learner</Tag> : <Tag color="green">Voter</Tag>,
              },
            ]}
          />
        </Card>
      )}

      {/* 成员详细状态 */}
      {memberStatuses.length > 0 && (
        <Card title="成员详细状态" style={{ marginBottom: 24 }}>
          <Table
            rowKey="endpoint"
            dataSource={memberStatuses}
            pagination={false}
            size="small"
            columns={[
              {
                title: '名称', dataIndex: 'name', key: 'name',
                render: (name: string, record: MemberStatus) => (
                  <Space>
                    {name || record.endpoint}
                    {record.is_leader && <Tag icon={<CrownOutlined />} color="blue">Leader</Tag>}
                    {record.is_learner && <Tag icon={<BookOutlined />} color="orange">Learner</Tag>}
                  </Space>
                ),
              },
              { title: 'Endpoint', dataIndex: 'endpoint', key: 'endpoint' },
              { title: '版本', dataIndex: 'version', key: 'version', width: 90 },
              { title: 'DB 大小', key: 'db_size', width: 100, render: (_: unknown, r: MemberStatus) => formatBytes(r.db_size) },
              { title: 'DB 使用', key: 'db_use', width: 100, render: (_: unknown, r: MemberStatus) => formatBytes(r.db_size_in_use) },
              {
                title: '碎片率', key: 'frag', width: 100,
                render: (_: unknown, r: MemberStatus) => {
                  const pct = r.db_size > 0 ? Math.round((1 - r.db_size_in_use / r.db_size) * 100) : 0
                  return <Progress percent={pct} size="small" status={pct > 50 ? 'exception' : 'success'} />
                },
              },
              { title: 'Raft Index', dataIndex: 'raft_index', key: 'raft_index', width: 110 },
              { title: 'Raft Term', dataIndex: 'raft_term', key: 'raft_term', width: 100 },
              { title: 'Applied Index', dataIndex: 'raft_applied_index', key: 'raft_applied_index', width: 120 },
            ]}
          />
        </Card>
      )}

      {healthData.length > 0 && (
        <Card title="健康检查">
          <Table
            rowKey="endpoint"
            dataSource={healthData}
            columns={[
              { title: 'Endpoint', dataIndex: 'endpoint', key: 'endpoint' },
              {
                title: '状态', dataIndex: 'healthy', key: 'healthy',
                render: (h: boolean) => h
                  ? <Tag icon={<CheckCircleOutlined />} color="success">健康</Tag>
                  : <Tag icon={<CloseCircleOutlined />} color="error">异常</Tag>,
              },
            ]}
            pagination={false}
            size="small"
          />
        </Card>
      )}
    </>
  )
}
