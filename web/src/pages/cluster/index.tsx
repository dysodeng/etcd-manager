import { useEffect, useState, useCallback, useRef } from 'react'
import {
  Descriptions, Table, Spin, Button, Space, Result, message, Alert,
} from 'antd'
import {
  ReloadOutlined, DisconnectOutlined,
} from '@ant-design/icons'
import type { ClusterStatus, ClusterMetrics, MemberStatus, AlarmInfo } from '@/types'
import { clusterApi } from '@/api/cluster'
import { MetricCard, PageHeader, SectionCard, StatusBadge } from '@/components/ui'
import { FragmentationProgress } from './FragmentationProgress'
import { buildClusterMetricView, formatBytes } from './presentation'

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

  const memberColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id' },
    {
      title: '名称', dataIndex: 'name', key: 'name',
      render: (name: string) => (
        <Space>
          {name}
          {name === status?.leader && <StatusBadge tone="info">Leader</StatusBadge>}
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
      render: (_: unknown, record: { is_learner: boolean }) => record.is_learner
        ? <StatusBadge tone="warning">Learner</StatusBadge>
        : <StatusBadge tone="success">Voter</StatusBadge>,
    },
  ]

  // DB 碎片率：etcd 使用 BoltDB 存储，删除/更新 key 后旧空间不会立即回收，
  // 导致 DB 文件大小 > 实际数据量。碎片率 = (DB总大小 - 实际使用) / DB总大小。
  // 碎片率过高意味着磁盘浪费，备份/快照变大、恢复变慢。
  // 优化方式：执行 etcdctl defrag 压缩数据库（会短暂阻塞该节点，建议逐节点执行）。
  const alarmList = (
    <ul className="alarm-list">
      {alarms.map((alarm) => (
        <li key={`${alarm.member_id}-${alarm.alarm_type}`}>
          成员 {alarm.member_id}：
          {alarm.alarm_type === 'NOSPACE'
            ? '磁盘空间不足'
            : alarm.alarm_type === 'CORRUPT'
              ? '数据损坏'
              : alarm.alarm_type}
        </li>
      ))}
    </ul>
  )

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
      <PageHeader
        eyebrow="Cluster Overview"
        title="集群概览"
        description="实时掌握成员健康、存储使用与 Raft 同步状态"
        extra={(
          <Button type="primary" icon={<ReloadOutlined />} onClick={fetchData} loading={loading}>
            刷新数据
          </Button>
        )}
      />

      {/* 报警 */}
      {alarms.length > 0 && (
        <Alert
          className="page-alert"
          message="集群报警"
          description={alarmList}
          type="error"
          showIcon
        />
      )}

      {metrics && (
        <div className="metric-grid">
          {buildClusterMetricView(metrics).map(({ key, ...metric }) => (
            <MetricCard key={key} {...metric} />
          ))}
        </div>
      )}

      <div className="page-stack">
        <SectionCard title="集群成员" description={`共 ${status?.members.length ?? 0} 个成员`}>
          <Descriptions column={2} size="small" style={{ marginBottom: 16 }}>
            <Descriptions.Item label="集群 ID">{status?.cluster_id}</Descriptions.Item>
            <Descriptions.Item label="Leader">{status?.leader}</Descriptions.Item>
          </Descriptions>
          <Table
            className="data-table"
            rowKey="id"
            dataSource={status?.members ?? []}
            columns={memberColumns}
            pagination={false}
            size="middle"
          />
        </SectionCard>

        {/* 成员详细状态 */}
        {memberStatuses.length > 0 && (
          <SectionCard title="成员详细状态" description="各节点存储与 Raft 同步详情">
            <Table
              className="data-table"
              rowKey="endpoint"
              dataSource={memberStatuses}
              pagination={false}
              size="middle"
              columns={[
                {
                  title: '名称', dataIndex: 'name', key: 'name',
                  render: (name: string, record: MemberStatus) => (
                    <Space>
                      {name || record.endpoint}
                      {record.is_leader && <StatusBadge tone="info">Leader</StatusBadge>}
                      {record.is_learner && <StatusBadge tone="warning">Learner</StatusBadge>}
                    </Space>
                  ),
                },
                { title: 'Endpoint', dataIndex: 'endpoint', key: 'endpoint' },
                { title: '版本', dataIndex: 'version', key: 'version', width: 90 },
                { title: 'DB 大小', key: 'db_size', width: 100, render: (_: unknown, r: MemberStatus) => formatBytes(r.db_size) },
                { title: 'DB 使用', key: 'db_use', width: 100, render: (_: unknown, r: MemberStatus) => formatBytes(r.db_size_in_use) },
                {
                  title: '碎片率', key: 'frag', width: 100,
                  render: (_: unknown, r: MemberStatus) => (
                    <FragmentationProgress dbSize={r.db_size} dbSizeInUse={r.db_size_in_use} />
                  ),
                },
                // Raft Index - Raft 日志最新条目索引，代表集群收到的写操作总序号
                { title: 'Raft Index', dataIndex: 'raft_index', key: 'raft_index', width: 110 },
                // Raft Term - Raft 选举任期号，每次 Leader 选举 +1，Term 增长过快可能说明网络不稳定
                { title: 'Raft Term', dataIndex: 'raft_term', key: 'raft_term', width: 100 },
                // Applied Index - 已应用到状态机的日志索引，正常时应接近 Raft Index，差距大说明节点落后
                { title: 'Applied Index', dataIndex: 'raft_applied_index', key: 'raft_applied_index', width: 120 },
              ]}
            />
          </SectionCard>
        )}

        {healthData.length > 0 && (
          <SectionCard title="健康检查" description="成员端点实时健康状态">
            <Table
              className="data-table"
              rowKey="endpoint"
              dataSource={healthData}
              columns={[
                { title: 'Endpoint', dataIndex: 'endpoint', key: 'endpoint' },
                {
                  title: '状态', dataIndex: 'healthy', key: 'healthy',
                  render: (healthy: boolean) => healthy
                    ? <StatusBadge tone="success">健康</StatusBadge>
                    : <StatusBadge tone="danger">异常</StatusBadge>,
                },
              ]}
              pagination={false}
              size="middle"
            />
          </SectionCard>
        )}
      </div>
    </>
  )
}
