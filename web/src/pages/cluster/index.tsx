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

  // DB 碎片率：etcd 使用 BoltDB 存储，删除/更新 key 后旧空间不会立即回收，
  // 导致 DB 文件大小 > 实际数据量。碎片率 = (DB总大小 - 实际使用) / DB总大小。
  // 碎片率过高意味着磁盘浪费，备份/快照变大、恢复变慢。
  // 优化方式：执行 etcdctl defrag 压缩数据库（会短暂阻塞该节点，建议逐节点执行）。
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
          <Col span={4}>
            <Card style={{ height: '100%' }}>
              <Statistic
                title="集群 ID"
                value={metrics.cluster_id}
                valueStyle={{ fontSize: 14, wordBreak: 'break-all' }}
              />
            </Card>
          </Col>
          <Col span={4}><Card style={{ height: '100%' }}><Statistic title="成员数量" value={metrics.member_count} /></Card></Col>
          <Col span={4}><Card style={{ height: '100%' }}><Statistic title="DB 大小" value={formatBytes(metrics.db_size)} /></Card></Col>
          <Col span={4}><Card style={{ height: '100%' }}><Statistic title="DB 实际使用" value={formatBytes(metrics.db_size_in_use)} /></Card></Col>
          <Col span={4}>
            <Card style={{ height: '100%' }}>
              <Statistic
                title="DB 碎片率"
                value={fragPercent}
                suffix="%"
                valueStyle={{ color: fragPercent > 50 ? '#cf1322' : fragPercent > 30 ? '#faad14' : '#3f8600' }}
              />
            </Card>
          </Col>
          <Col span={4}><Card style={{ height: '100%' }}><Statistic title="etcd 版本" value={metrics.version} /></Card></Col>
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
                // 成员角色：
                // Voter - 正式投票成员，参与 Raft 共识（选举 Leader、确认写入），集群需要多数 Voter 存活才能工作
                // Learner - 只读追随者，同步数据但不参与投票，用于安全扩容（先追数据再提升为 Voter）
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
              // Raft Index - Raft 日志最新条目索引，代表集群收到的写操作总序号
              { title: 'Raft Index', dataIndex: 'raft_index', key: 'raft_index', width: 110 },
              // Raft Term - Raft 选举任期号，每次 Leader 选举 +1，Term 增长过快可能说明网络不稳定
              { title: 'Raft Term', dataIndex: 'raft_term', key: 'raft_term', width: 100 },
              // Applied Index - 已应用到状态机的日志索引，正常时应接近 Raft Index，差距大说明节点落后
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
